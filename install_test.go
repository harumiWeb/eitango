package eitango_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
)

//go:embed install.sh
var installScript string

func TestInstallScriptInstallsLatestRelease(t *testing.T) {
	skipOnWindows(t)

	const version = "v9.8.7"
	archiveName := releaseArchiveName(version, runtimeGOOS(t), runtimeGOARCH(t))
	archiveBytes := makeArchive(t)
	checksums := fmt.Sprintf("%s  %s\n", sha256Hex(archiveBytes), archiveName)

	var latestHits int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/test/eitango/releases/latest":
			atomic.AddInt32(&latestHits, 1)
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, fmt.Sprintf(`{"tag_name":"%s"}`, version))
		case fmt.Sprintf("/test/eitango/releases/download/%s/%s", version, archiveName):
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(archiveBytes)
		case fmt.Sprintf("/test/eitango/releases/download/%s/checksums.txt", version):
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, checksums)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	home := t.TempDir()
	runInstallScript(t, home, []string{}, installTestEnv(server.URL, nil)...)

	assertFileContains(t, filepath.Join(home, ".eitango", "version"), version)
	assertExecutableExists(t, filepath.Join(home, ".eitango", "bin", "eitango"))
	assertFileContains(t, filepath.Join(home, ".eitango", "share", "LICENSE"), "fixture license")
	assertFileContains(t, filepath.Join(home, ".eitango", "share", "README.md"), "fixture readme")
	assertFileContains(t, filepath.Join(home, ".eitango", "share", "README.en.md"), "fixture english readme")
	assertFileContains(t, filepath.Join(home, ".eitango", "share", "THIRD_PARTY_NOTICES.md"), "fixture notices")
	assertFileContains(t, filepath.Join(home, ".eitango", "share", "third_party", "licenses", "fixture.txt"), "fixture third-party license")
	if got := atomic.LoadInt32(&latestHits); got != 1 {
		t.Fatalf("latest release endpoint hits = %d, want 1", got)
	}
}

func TestInstallScriptPinnedVersionSkipsLatestLookup(t *testing.T) {
	skipOnWindows(t)

	const version = "v1.2.3"
	archiveName := releaseArchiveName(version, runtimeGOOS(t), runtimeGOARCH(t))
	archiveBytes := makeArchive(t)
	checksums := fmt.Sprintf("%s  %s\n", sha256Hex(archiveBytes), archiveName)

	var latestHits int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/test/eitango/releases/latest":
			atomic.AddInt32(&latestHits, 1)
			http.NotFound(w, r)
		case fmt.Sprintf("/test/eitango/releases/download/%s/%s", version, archiveName):
			_, _ = w.Write(archiveBytes)
		case fmt.Sprintf("/test/eitango/releases/download/%s/checksums.txt", version):
			_, _ = io.WriteString(w, checksums)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	home := t.TempDir()
	runInstallScript(t, home, []string{"--version", "1.2.3"}, installTestEnv(server.URL, nil)...)

	assertFileContains(t, filepath.Join(home, ".eitango", "version"), version)
	if got := atomic.LoadInt32(&latestHits); got != 0 {
		t.Fatalf("latest release endpoint hits = %d, want 0", got)
	}
}

func TestInstallScriptChecksumMismatchKeepsExistingInstall(t *testing.T) {
	skipOnWindows(t)

	const (
		oldVersion = "v0.9.0"
		newVersion = "v1.0.0"
	)
	archiveName := releaseArchiveName(newVersion, runtimeGOOS(t), runtimeGOARCH(t))
	archiveBytes := makeArchive(t)
	checksums := fmt.Sprintf("%s  %s\n", strings.Repeat("0", 64), archiveName)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/test/eitango/releases/latest":
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, fmt.Sprintf(`{"tag_name":"%s"}`, newVersion))
		case fmt.Sprintf("/test/eitango/releases/download/%s/%s", newVersion, archiveName):
			_, _ = w.Write(archiveBytes)
		case fmt.Sprintf("/test/eitango/releases/download/%s/checksums.txt", newVersion):
			_, _ = io.WriteString(w, checksums)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	home := t.TempDir()
	mustWriteFile(t, filepath.Join(home, ".eitango", "bin", "eitango"), "#!/bin/sh\necho old\n", 0o755)
	mustWriteFile(t, filepath.Join(home, ".eitango", "version"), oldVersion+"\n", 0o644)
	mustWriteFile(t, filepath.Join(home, ".eitango", "share", "README.md"), "old readme\n", 0o644)

	err := runInstallScriptErr(home, []string{}, installTestEnv(server.URL, nil)...)
	if err == nil {
		t.Fatal("install.sh succeeded, want checksum mismatch failure")
	}
	if !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("install.sh error = %v, want checksum mismatch", err)
	}

	assertFileContains(t, filepath.Join(home, ".eitango", "version"), oldVersion)
	assertFileContains(t, filepath.Join(home, ".eitango", "share", "README.md"), "old readme")
}

func TestInstallScriptFailsWithoutChecksumTool(t *testing.T) {
	skipOnWindows(t)

	const version = "v2.0.0"
	archiveName := releaseArchiveName(version, runtimeGOOS(t), runtimeGOARCH(t))
	archiveBytes := makeArchive(t)
	checksums := fmt.Sprintf("%s  %s\n", sha256Hex(archiveBytes), archiveName)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/test/eitango/releases/latest":
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, fmt.Sprintf(`{"tag_name":"%s"}`, version))
		case fmt.Sprintf("/test/eitango/releases/download/%s/%s", version, archiveName):
			_, _ = w.Write(archiveBytes)
		case fmt.Sprintf("/test/eitango/releases/download/%s/checksums.txt", version):
			_, _ = io.WriteString(w, checksums)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	wrappers := map[string]string{
		"sha256sum": "#!/bin/sh\nexit 127\n",
		"shasum":    "#!/bin/sh\nexit 127\n",
		"openssl":   "#!/bin/sh\nexit 127\n",
	}

	home := t.TempDir()
	err := runInstallScriptErr(home, []string{}, installTestEnv(server.URL, wrappers)...)
	if err == nil {
		t.Fatal("install.sh succeeded, want missing checksum tool failure")
	}
	if !strings.Contains(err.Error(), "no usable SHA256 tool found") {
		t.Fatalf("install.sh error = %v, want missing checksum tool", err)
	}
}

func TestInstallScriptFailsWhenChecksumEntryIsMissing(t *testing.T) {
	skipOnWindows(t)

	const version = "v2.1.0"
	archiveName := releaseArchiveName(version, runtimeGOOS(t), runtimeGOARCH(t))
	archiveBytes := makeArchive(t)
	checksums := fmt.Sprintf("%s  %s\n", sha256Hex(archiveBytes), "eitango_other.tar.gz")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/test/eitango/releases/latest":
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, fmt.Sprintf(`{"tag_name":"%s"}`, version))
		case fmt.Sprintf("/test/eitango/releases/download/%s/%s", version, archiveName):
			_, _ = w.Write(archiveBytes)
		case fmt.Sprintf("/test/eitango/releases/download/%s/checksums.txt", version):
			_, _ = io.WriteString(w, checksums)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	home := t.TempDir()
	err := runInstallScriptErr(home, []string{}, installTestEnv(server.URL, nil)...)
	if err == nil {
		t.Fatal("install.sh succeeded, want missing checksum entry failure")
	}
	if !strings.Contains(err.Error(), "checksum entry") {
		t.Fatalf("install.sh error = %v, want missing checksum entry", err)
	}
}

func TestInstallScriptFailsOnUnsupportedArchitecture(t *testing.T) {
	skipOnWindows(t)

	unamePath, err := exec.LookPath("uname")
	if err != nil {
		t.Fatalf("lookpath uname: %v", err)
	}

	wrappers := map[string]string{
		"uname": fmt.Sprintf("#!/bin/sh\nif [ \"$1\" = \"-s\" ]; then exec %s -s; fi\nif [ \"$1\" = \"-m\" ]; then echo mips64; exit 0; fi\nexec %s \"$@\"\n", shellQuoteForScript(unamePath), shellQuoteForScript(unamePath)),
	}

	home := t.TempDir()
	err = runInstallScriptErr(home, []string{}, installTestEnv("http://127.0.0.1.invalid", wrappers)...)
	if err == nil {
		t.Fatal("install.sh succeeded, want unsupported architecture failure")
	}
	if !strings.Contains(err.Error(), "unsupported architecture") {
		t.Fatalf("install.sh error = %v, want unsupported architecture", err)
	}
}

func TestInstallScriptUninstallKeepsAndPurgesData(t *testing.T) {
	skipOnWindows(t)

	osName := runtimeGOOS(t)
	home := t.TempDir()

	mustWriteFile(t, filepath.Join(home, ".eitango", "bin", "eitango"), "#!/bin/sh\n", 0o755)
	mustWriteFile(t, filepath.Join(home, ".eitango", "version"), "v1.0.0\n", 0o644)
	defaultDataDir := filepath.Join(home, ".local", "share", "eitango-cli")
	if osName == "darwin" {
		defaultDataDir = filepath.Join(home, "Library", "Application Support", "eitango-cli")
	}
	mustWriteFile(t, filepath.Join(defaultDataDir, "user.db"), "fixture-db", 0o644)

	runInstallScript(t, home, []string{"--uninstall"}, installTestEnv("http://127.0.0.1.invalid", nil)...)
	if _, err := os.Stat(filepath.Join(home, ".eitango")); !os.IsNotExist(err) {
		t.Fatalf("installer root still exists after uninstall: %v", err)
	}
	assertFileContains(t, filepath.Join(defaultDataDir, "user.db"), "fixture-db")

	mustWriteFile(t, filepath.Join(home, ".eitango", "bin", "eitango"), "#!/bin/sh\n", 0o755)
	runInstallScript(t, home, []string{"--uninstall", "--purge-data"}, installTestEnv("http://127.0.0.1.invalid", nil)...)
	if _, err := os.Stat(defaultDataDir); !os.IsNotExist(err) {
		t.Fatalf("default data dir still exists after purge uninstall: %v", err)
	}
}

func installTestEnv(serverURL string, wrappers map[string]string) []string {
	env := []string{
		"EITANGO_INSTALL_REPO=test/eitango",
		"EITANGO_INSTALL_API_BASE=" + serverURL,
		"EITANGO_INSTALL_DOWNLOAD_BASE=" + serverURL,
	}
	for name, script := range wrappers {
		env = append(env, "EITANGO_TEST_WRAPPER_"+strings.ToUpper(name)+"="+script)
	}
	return env
}

func runInstallScript(t *testing.T, home string, args []string, env ...string) {
	t.Helper()
	if err := runInstallScriptErr(home, args, env...); err != nil {
		t.Fatalf("install.sh failed: %v", err)
	}
}

func runInstallScriptErr(home string, args []string, env ...string) error {
	scriptPath, err := writeInstallScript(home)
	if err != nil {
		return err
	}

	cmd := exec.Command("sh", append([]string{scriptPath}, args...)...)
	cmd.Dir = filepath.Clean(".")
	cmd.Env = append(os.Environ(), env...)
	cmd.Env = append(cmd.Env, "HOME="+home)

	wrappers := wrapperScripts(env)
	if len(wrappers) > 0 {
		wrapperDir, err := makeWrapperDir(home, wrappers)
		if err != nil {
			return err
		}
		cmd.Env = append(cmd.Env, "PATH="+wrapperDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	}

	output, err := cmd.CombinedOutput()
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
}

func writeInstallScript(home string) (string, error) {
	path := filepath.Join(home, "install.sh")
	if err := os.WriteFile(path, []byte(installScript), 0o755); err != nil {
		return "", err
	}
	return path, nil
}

func makeWrapperDir(base string, wrappers map[string]string) (string, error) {
	dir := filepath.Join(base, "wrappers")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	for name, script := range wrappers {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
			return "", err
		}
	}
	return dir, nil
}

func shellQuoteForScript(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}

func makeArchive(t *testing.T) []byte {
	t.Helper()
	tempDir := t.TempDir()
	mustWriteFile(t, filepath.Join(tempDir, "eitango"), "#!/bin/sh\necho fixture\n", 0o755)
	mustWriteFile(t, filepath.Join(tempDir, "LICENSE"), "fixture license\n", 0o644)
	mustWriteFile(t, filepath.Join(tempDir, "README.md"), "fixture readme\n", 0o644)
	mustWriteFile(t, filepath.Join(tempDir, "README.en.md"), "fixture english readme\n", 0o644)
	mustWriteFile(t, filepath.Join(tempDir, "THIRD_PARTY_NOTICES.md"), "fixture notices\n", 0o644)
	mustWriteFile(t, filepath.Join(tempDir, "third_party", "licenses", "fixture.txt"), "fixture third-party license\n", 0o644)

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)

	err := filepath.Walk(tempDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if path == tempDir {
			return nil
		}
		rel, err := filepath.Rel(tempDir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = rel
		if info.IsDir() {
			header.Name += "/"
		}
		if err := tw.WriteHeader(header); err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		_, err = tw.Write(data)
		return err
	})
	if err != nil {
		t.Fatalf("make archive: %v", err)
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("close tar: %v", err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("close gzip: %v", err)
	}
	return buf.Bytes()
}

func releaseArchiveName(version, goos, arch string) string {
	version = strings.TrimPrefix(strings.TrimPrefix(version, "v"), "V")
	return fmt.Sprintf("eitango_%s_%s_%s.tar.gz", version, goos, arch)
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func mustWriteFile(t *testing.T, path string, content string, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func assertFileContains(t *testing.T, path string, want string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if !strings.Contains(string(data), want) {
		t.Fatalf("%s = %q, want substring %q", path, string(data), want)
	}
}

func assertExecutableExists(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	if info.Mode()&0o111 == 0 {
		t.Fatalf("%s mode = %v, want executable", path, info.Mode())
	}
}

func wrapperScripts(env []string) map[string]string {
	scripts := map[string]string{}
	for _, entry := range env {
		if !strings.HasPrefix(entry, "EITANGO_TEST_WRAPPER_") {
			continue
		}
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) != 2 {
			continue
		}
		name := strings.TrimPrefix(parts[0], "EITANGO_TEST_WRAPPER_")
		scripts[strings.ToLower(name)] = parts[1]
	}
	return scripts
}

func runtimeGOOS(t *testing.T) string {
	t.Helper()
	switch runtime.GOOS {
	case "darwin":
		return "darwin"
	case "linux":
		return "linux"
	default:
		t.Fatalf("unsupported GOOS in test: %s", runtime.GOOS)
		return ""
	}
}

func runtimeGOARCH(t *testing.T) string {
	t.Helper()
	switch runtime.GOARCH {
	case "amd64":
		return "x86_64"
	case "arm64":
		return "arm64"
	default:
		t.Fatalf("unsupported GOARCH in test: %s", runtime.GOARCH)
		return ""
	}
}

func skipOnWindows(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("install.sh integration tests are unix-only")
	}
}
