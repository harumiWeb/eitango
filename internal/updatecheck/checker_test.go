package updatecheck

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"
)

func TestCompareVersions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		left    string
		right   string
		want    int
		wantErr bool
	}{
		{name: "equal stable", left: "v1.2.3", right: "1.2.3", want: 0},
		{name: "newer patch", left: "v1.2.4", right: "v1.2.3", want: 1},
		{name: "older minor", left: "v1.1.9", right: "v1.2.0", want: -1},
		{name: "stable beats prerelease", left: "v1.2.3", right: "v1.2.3-rc.1", want: 1},
		{name: "prerelease ordering", left: "v1.2.3-rc.2", right: "v1.2.3-rc.1", want: 1},
		{name: "invalid version", left: "dev", right: "v1.2.3", wantErr: true},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := CompareVersions(test.left, test.right)
			if test.wantErr {
				if err == nil {
					t.Fatal("CompareVersions() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("CompareVersions() error = %v", err)
			}
			if got != test.want {
				t.Fatalf("CompareVersions() = %d, want %d", got, test.want)
			}
		})
	}
}

func TestCheckerFirstSuccessfulCheckSuppressesNotice(t *testing.T) {
	var hits int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		if got := r.Header.Get("Accept"); got != "application/vnd.github+json" {
			t.Fatalf("Accept = %q, want GitHub media type", got)
		}
		if err := json.NewEncoder(w).Encode(ReleaseInfo{
			TagName: "v1.2.0",
			HTMLURL: "https://example.com/eitango/v1.2.0",
		}); err != nil {
			t.Fatalf("Encode() error = %v", err)
		}
	}))
	defer server.Close()

	now := time.Date(2026, 3, 29, 10, 0, 0, 0, time.UTC)
	checker := New(filepath.Join(t.TempDir(), "update-check.json"))
	checker.LatestReleaseURL = server.URL
	checker.Now = func() time.Time { return now }

	result, err := checker.Check(context.Background(), "v1.1.0")
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if hits != 1 {
		t.Fatalf("hits = %d, want 1", hits)
	}
	if !result.Checked {
		t.Fatal("Checked = false, want true")
	}
	if !result.Compared {
		t.Fatal("Compared = false, want true")
	}
	if !result.UpdateAvailable {
		t.Fatal("UpdateAvailable = false, want true")
	}
	if result.ShouldNotify {
		t.Fatal("ShouldNotify = true, want false on first successful check")
	}

	saved, err := checker.loadState()
	if err != nil {
		t.Fatalf("loadState() error = %v", err)
	}
	if saved.LatestTag != "v1.2.0" {
		t.Fatalf("LatestTag = %q, want v1.2.0", saved.LatestTag)
	}
	if saved.LastSuccessfulAt.IsZero() {
		t.Fatal("LastSuccessfulAt is zero, want saved timestamp")
	}
}

func TestCheckerUsesCachedLatestWithinTTL(t *testing.T) {
	var hits int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if err := json.NewEncoder(w).Encode(ReleaseInfo{
			TagName: "v1.2.0",
			HTMLURL: "https://example.com/eitango/v1.2.0",
		}); err != nil {
			t.Fatalf("Encode() error = %v", err)
		}
	}))
	defer server.Close()

	now := time.Date(2026, 3, 29, 10, 0, 0, 0, time.UTC)
	checker := New(filepath.Join(t.TempDir(), "update-check.json"))
	checker.LatestReleaseURL = server.URL
	checker.Now = func() time.Time { return now }

	first, err := checker.Check(context.Background(), "v1.1.0")
	if err != nil {
		t.Fatalf("first Check() error = %v", err)
	}
	if first.ShouldNotify {
		t.Fatal("first ShouldNotify = true, want false")
	}

	now = now.Add(time.Hour)
	second, err := checker.Check(context.Background(), "v1.1.0")
	if err != nil {
		t.Fatalf("second Check() error = %v", err)
	}
	if hits != 1 {
		t.Fatalf("hits = %d, want cached response without another HTTP call", hits)
	}
	if second.Checked {
		t.Fatal("Checked = true, want false for cached result")
	}
	if !second.UpdateAvailable {
		t.Fatal("UpdateAvailable = false, want true")
	}
	if !second.ShouldNotify {
		t.Fatal("ShouldNotify = false, want true after first successful check")
	}
}

func TestCheckerCheckNowBypassesTTL(t *testing.T) {
	var hits int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if err := json.NewEncoder(w).Encode(ReleaseInfo{
			TagName: "v1.2.0",
			HTMLURL: "https://example.com/eitango/v1.2.0",
		}); err != nil {
			t.Fatalf("Encode() error = %v", err)
		}
	}))
	defer server.Close()

	now := time.Date(2026, 3, 29, 10, 0, 0, 0, time.UTC)
	checker := New(filepath.Join(t.TempDir(), "update-check.json"))
	checker.LatestReleaseURL = server.URL
	checker.Now = func() time.Time { return now }

	if _, err := checker.Check(context.Background(), "v1.1.0"); err != nil {
		t.Fatalf("Check() error = %v", err)
	}

	now = now.Add(time.Hour)
	result, err := checker.CheckNow(context.Background(), "v1.1.0")
	if err != nil {
		t.Fatalf("CheckNow() error = %v", err)
	}
	if hits != 2 {
		t.Fatalf("hits = %d, want 2 because CheckNow bypasses TTL", hits)
	}
	if !result.Checked {
		t.Fatal("Checked = false, want true")
	}
	if !result.ShouldNotify {
		t.Fatal("ShouldNotify = false, want true after first successful check")
	}
}

func TestCheckerDisabledByEnv(t *testing.T) {
	t.Setenv(DisableEnv, "1")

	var hits int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
	}))
	defer server.Close()

	checker := New(filepath.Join(t.TempDir(), "update-check.json"))
	checker.LatestReleaseURL = server.URL

	result, err := checker.Check(context.Background(), "v1.1.0")
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if !result.Disabled {
		t.Fatal("Disabled = false, want true")
	}
	if hits != 0 {
		t.Fatalf("hits = %d, want 0 while disabled", hits)
	}
}
