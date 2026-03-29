package updatecheck

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	DisableEnv            = "EITANGO_DISABLE_UPDATE_CHECK"
	DefaultTTL            = 24 * time.Hour
	DefaultTimeout        = 1500 * time.Millisecond
	defaultLatestReleases = "https://api.github.com/repos/harumiWeb/eitango/releases/latest"
)

type Service interface {
	Check(ctx context.Context, currentVersion string) (Result, error)
	CheckNow(ctx context.Context, currentVersion string) (Result, error)
}

type ReleaseInfo struct {
	TagName    string `json:"tag_name"`
	HTMLURL    string `json:"html_url"`
	Prerelease bool   `json:"prerelease"`
}

type Result struct {
	CurrentVersion  string
	Latest          ReleaseInfo
	UpdateAvailable bool
	ShouldNotify    bool
	Disabled        bool
	Checked         bool
	Compared        bool
}

type Checker struct {
	StatePath        string
	LatestReleaseURL string
	HTTPClient       *http.Client
	Now              func() time.Time
	TTL              time.Duration
	Timeout          time.Duration
}

type state struct {
	LastCheckedAt    time.Time `json:"last_checked_at,omitempty"`
	LastSuccessfulAt time.Time `json:"last_successful_at,omitempty"`
	LatestTag        string    `json:"latest_tag,omitempty"`
	LatestURL        string    `json:"latest_url,omitempty"`
	LatestPrerelease bool      `json:"latest_prerelease,omitempty"`
}

type semver struct {
	Major      int64
	Minor      int64
	Patch      int64
	Prerelease []identifier
}

type identifier struct {
	Raw     string
	Numeric bool
	Number  int64
}

func New(statePath string) *Checker {
	return &Checker{
		StatePath:        statePath,
		LatestReleaseURL: defaultLatestReleases,
		Now:              time.Now,
		TTL:              DefaultTTL,
		Timeout:          DefaultTimeout,
	}
}

func DefaultStatePath(dataDir string) string {
	return filepath.Join(dataDir, "update-check.json")
}

func (c *Checker) Check(ctx context.Context, currentVersion string) (Result, error) {
	return c.check(ctx, currentVersion, false)
}

func (c *Checker) CheckNow(ctx context.Context, currentVersion string) (Result, error) {
	return c.check(ctx, currentVersion, true)
}

func (c *Checker) check(ctx context.Context, currentVersion string, force bool) (Result, error) {
	result := Result{CurrentVersion: strings.TrimSpace(currentVersion)}
	if updateChecksDisabled() {
		result.Disabled = true
		return result, nil
	}

	now := c.now().UTC()
	cachedState, stateErr := c.loadState()
	if !force && shouldUseCachedState(cachedState, now, c.ttl()) {
		return cachedResult(currentVersion, cachedState), nil
	}

	cachedState.LastCheckedAt = now
	release, err := c.fetchLatest(ctx)
	if err != nil {
		if saveErr := c.saveState(cachedState); saveErr != nil {
			if stateErr != nil {
				return cachedResult(currentVersion, cachedState), errors.Join(stateErr, err, saveErr)
			}
			return cachedResult(currentVersion, cachedState), errors.Join(err, saveErr)
		}
		if stateErr != nil {
			return cachedResult(currentVersion, cachedState), errors.Join(stateErr, err)
		}
		return cachedResult(currentVersion, cachedState), err
	}

	hadSuccessfulCheck := !cachedState.LastSuccessfulAt.IsZero()
	cachedState.LastSuccessfulAt = now
	cachedState.LatestTag = strings.TrimSpace(release.TagName)
	cachedState.LatestURL = strings.TrimSpace(release.HTMLURL)
	cachedState.LatestPrerelease = release.Prerelease

	result = checkedResult(currentVersion, release, hadSuccessfulCheck)
	if err := c.saveState(cachedState); err != nil {
		return result, err
	}
	return result, nil
}

func checkedResult(currentVersion string, release ReleaseInfo, hadSuccessfulCheck bool) Result {
	result := Result{
		CurrentVersion: strings.TrimSpace(currentVersion),
		Latest: ReleaseInfo{
			TagName:    strings.TrimSpace(release.TagName),
			HTMLURL:    strings.TrimSpace(release.HTMLURL),
			Prerelease: release.Prerelease,
		},
		Checked: true,
	}
	applyComparison(&result, hadSuccessfulCheck)
	return result
}

func cachedResult(currentVersion string, saved state) Result {
	result := Result{
		CurrentVersion: strings.TrimSpace(currentVersion),
		Latest: ReleaseInfo{
			TagName:    strings.TrimSpace(saved.LatestTag),
			HTMLURL:    strings.TrimSpace(saved.LatestURL),
			Prerelease: saved.LatestPrerelease,
		},
	}
	applyComparison(&result, !saved.LastSuccessfulAt.IsZero())
	return result
}

func applyComparison(result *Result, shouldNotify bool) {
	if result == nil || strings.TrimSpace(result.Latest.TagName) == "" || result.Latest.Prerelease {
		return
	}
	newer, err := IsNewer(result.Latest.TagName, result.CurrentVersion)
	if err != nil {
		return
	}
	result.Compared = true
	result.UpdateAvailable = newer
	result.ShouldNotify = newer && shouldNotify
}

func (c *Checker) fetchLatest(ctx context.Context) (ReleaseInfo, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, c.timeout())
	defer cancel()

	req, err := http.NewRequestWithContext(timeoutCtx, http.MethodGet, c.latestReleaseURL(), nil)
	if err != nil {
		return ReleaseInfo{}, fmt.Errorf("create latest release request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "eitango")

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return ReleaseInfo{}, fmt.Errorf("fetch latest release: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return ReleaseInfo{}, fmt.Errorf("fetch latest release: unexpected status: %s", resp.Status)
	}

	var release ReleaseInfo
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return ReleaseInfo{}, fmt.Errorf("decode latest release: %w", err)
	}
	release.TagName = strings.TrimSpace(release.TagName)
	release.HTMLURL = strings.TrimSpace(release.HTMLURL)
	return release, nil
}

func (c *Checker) loadState() (state, error) {
	if strings.TrimSpace(c.StatePath) == "" {
		return state{}, nil
	}

	data, err := os.ReadFile(c.StatePath)
	if errors.Is(err, os.ErrNotExist) {
		return state{}, nil
	}
	if err != nil {
		return state{}, fmt.Errorf("read update check state %s: %w", c.StatePath, err)
	}

	var saved state
	if err := json.Unmarshal(data, &saved); err != nil {
		return state{}, fmt.Errorf("parse update check state %s: %w", c.StatePath, err)
	}
	return saved, nil
}

func (c *Checker) saveState(saved state) error {
	if strings.TrimSpace(c.StatePath) == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(c.StatePath), 0o755); err != nil {
		return fmt.Errorf("create update check dir: %w", err)
	}

	tmp, err := os.CreateTemp(filepath.Dir(c.StatePath), "eitango-update-check-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp update check state: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() {
		_ = os.Remove(tmpPath)
	}()

	encoder := json.NewEncoder(tmp)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(saved); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("encode update check state: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync update check state: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close update check state: %w", err)
	}
	if err := replaceFile(tmpPath, c.StatePath); err != nil {
		return fmt.Errorf("replace update check state: %w", err)
	}
	return nil
}

func CompareVersions(left, right string) (int, error) {
	lv, err := parseVersion(left)
	if err != nil {
		return 0, err
	}
	rv, err := parseVersion(right)
	if err != nil {
		return 0, err
	}

	switch {
	case lv.Major != rv.Major:
		return compareInt64(lv.Major, rv.Major), nil
	case lv.Minor != rv.Minor:
		return compareInt64(lv.Minor, rv.Minor), nil
	case lv.Patch != rv.Patch:
		return compareInt64(lv.Patch, rv.Patch), nil
	}
	return comparePrerelease(lv.Prerelease, rv.Prerelease), nil
}

func IsNewer(candidate, current string) (bool, error) {
	comparison, err := CompareVersions(candidate, current)
	if err != nil {
		return false, err
	}
	return comparison > 0, nil
}

func parseVersion(raw string) (semver, error) {
	version := strings.TrimSpace(raw)
	if version == "" {
		return semver{}, fmt.Errorf("empty version")
	}
	version = strings.TrimPrefix(strings.TrimPrefix(version, "v"), "V")
	if version == "" {
		return semver{}, fmt.Errorf("empty version")
	}

	withoutBuild := strings.SplitN(version, "+", 2)[0]
	core := withoutBuild
	pre := ""
	if parts := strings.SplitN(withoutBuild, "-", 2); len(parts) == 2 {
		core = parts[0]
		pre = parts[1]
	}

	coreParts := strings.Split(core, ".")
	if len(coreParts) != 3 {
		return semver{}, fmt.Errorf("invalid semantic version %q", raw)
	}

	major, err := parseNumericIdentifier(coreParts[0])
	if err != nil {
		return semver{}, fmt.Errorf("invalid major version in %q: %w", raw, err)
	}
	minor, err := parseNumericIdentifier(coreParts[1])
	if err != nil {
		return semver{}, fmt.Errorf("invalid minor version in %q: %w", raw, err)
	}
	patch, err := parseNumericIdentifier(coreParts[2])
	if err != nil {
		return semver{}, fmt.Errorf("invalid patch version in %q: %w", raw, err)
	}

	parsed := semver{
		Major: major,
		Minor: minor,
		Patch: patch,
	}
	if pre == "" {
		return parsed, nil
	}

	preParts := strings.Split(pre, ".")
	parsed.Prerelease = make([]identifier, 0, len(preParts))
	for _, part := range preParts {
		if strings.TrimSpace(part) == "" {
			return semver{}, fmt.Errorf("invalid prerelease in %q", raw)
		}
		id := identifier{Raw: part}
		if number, err := parseNumericIdentifier(part); err == nil {
			id.Numeric = true
			id.Number = number
		}
		parsed.Prerelease = append(parsed.Prerelease, id)
	}
	return parsed, nil
}

func parseNumericIdentifier(raw string) (int64, error) {
	if raw == "" {
		return 0, fmt.Errorf("empty identifier")
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, err
	}
	return value, nil
}

func comparePrerelease(left, right []identifier) int {
	switch {
	case len(left) == 0 && len(right) == 0:
		return 0
	case len(left) == 0:
		return 1
	case len(right) == 0:
		return -1
	}

	limit := len(left)
	if len(right) < limit {
		limit = len(right)
	}
	for i := 0; i < limit; i++ {
		comparison := compareIdentifier(left[i], right[i])
		if comparison != 0 {
			return comparison
		}
	}
	return compareInt(len(left), len(right))
}

func compareIdentifier(left, right identifier) int {
	switch {
	case left.Numeric && right.Numeric:
		return compareInt64(left.Number, right.Number)
	case left.Numeric:
		return -1
	case right.Numeric:
		return 1
	default:
		return compareInt(strings.Compare(left.Raw, right.Raw), 0)
	}
}

func compareInt64(left, right int64) int {
	switch {
	case left < right:
		return -1
	case left > right:
		return 1
	default:
		return 0
	}
}

func compareInt(left, right int) int {
	switch {
	case left < right:
		return -1
	case left > right:
		return 1
	default:
		return 0
	}
}

func shouldUseCachedState(saved state, now time.Time, ttl time.Duration) bool {
	if saved.LastCheckedAt.IsZero() || ttl <= 0 {
		return false
	}
	return now.Sub(saved.LastCheckedAt) < ttl
}

func updateChecksDisabled() bool {
	return strings.TrimSpace(os.Getenv(DisableEnv)) == "1"
}

func (c *Checker) latestReleaseURL() string {
	if url := strings.TrimSpace(c.LatestReleaseURL); url != "" {
		return url
	}
	return defaultLatestReleases
}

func (c *Checker) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return http.DefaultClient
}

func (c *Checker) now() time.Time {
	if c.Now != nil {
		return c.Now()
	}
	return time.Now()
}

func (c *Checker) ttl() time.Duration {
	if c.TTL > 0 {
		return c.TTL
	}
	return DefaultTTL
}

func (c *Checker) timeout() time.Duration {
	if c.Timeout > 0 {
		return c.Timeout
	}
	return DefaultTimeout
}

func replaceFile(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	if err := os.Remove(dst); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return os.Rename(src, dst)
}
