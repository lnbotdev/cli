package update

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseVersion(t *testing.T) {
	tests := []struct {
		name string
		v    string
		want [3]int
	}{
		{"with v prefix", "v1.2.3", [3]int{1, 2, 3}},
		{"without v prefix", "1.2.3", [3]int{1, 2, 3}},
		{"major only", "2", [3]int{2, 0, 0}},
		{"major.minor", "1.5", [3]int{1, 5, 0}},
		{"zeros", "0.0.0", [3]int{0, 0, 0}},
		{"large numbers", "10.20.30", [3]int{10, 20, 30}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseVersion(tt.v); got != tt.want {
				t.Errorf("parseVersion(%q) = %v, want %v", tt.v, got, tt.want)
			}
		})
	}
}

func TestIsNewer(t *testing.T) {
	tests := []struct {
		name    string
		latest  string
		current string
		want    bool
	}{
		{"major bump", "2.0.0", "1.0.0", true},
		{"minor bump", "1.1.0", "1.0.0", true},
		{"patch bump", "1.0.1", "1.0.0", true},
		{"equal", "1.0.0", "1.0.0", false},
		{"older major", "1.0.0", "2.0.0", false},
		{"older minor", "1.0.0", "1.1.0", false},
		{"older patch", "1.0.0", "1.0.1", false},
		{"with v prefix", "v1.1.0", "v1.0.0", true},
		{"mixed prefix", "1.1.0", "v1.0.0", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isNewer(tt.latest, tt.current); got != tt.want {
				t.Errorf("isNewer(%q, %q) = %v, want %v", tt.latest, tt.current, got, tt.want)
			}
		})
	}
}

func TestCheckForUpdate_DisabledByEnv(t *testing.T) {
	t.Setenv("LNBOT_NO_UPDATE_CHECK", "1")
	latest, available := CheckForUpdate("0.1.0")
	if available {
		t.Errorf("expected update check disabled, got available=true, latest=%q", latest)
	}
}

func TestCheckForUpdate_CachedResult(t *testing.T) {
	t.Setenv("LNBOT_NO_UPDATE_CHECK", "")

	// Set HOME to temp dir so cacheFile() uses our temp location
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	// Create cache directory and file with a fresh timestamp and newer version
	cacheDir := filepath.Join(dir, ".config", "lnbot")
	os.MkdirAll(cacheDir, 0o700)

	cached := cachedCheck{
		Latest:    "9.9.9",
		CheckedAt: time.Now().Unix(),
	}
	data, _ := json.Marshal(cached)
	os.WriteFile(filepath.Join(cacheDir, ".update-check"), data, 0o600)

	latest, available := CheckForUpdate("0.1.0")
	if !available {
		t.Fatal("expected update available from cache")
	}
	if latest != "9.9.9" {
		t.Errorf("latest = %q, want %q", latest, "9.9.9")
	}
}

func TestCheckForUpdate_CachedNoUpdate(t *testing.T) {
	t.Setenv("LNBOT_NO_UPDATE_CHECK", "")

	dir := t.TempDir()
	t.Setenv("HOME", dir)

	cacheDir := filepath.Join(dir, ".config", "lnbot")
	os.MkdirAll(cacheDir, 0o700)

	cached := cachedCheck{
		Latest:    "0.1.0",
		CheckedAt: time.Now().Unix(),
	}
	data, _ := json.Marshal(cached)
	os.WriteFile(filepath.Join(cacheDir, ".update-check"), data, 0o600)

	_, available := CheckForUpdate("0.1.0")
	if available {
		t.Error("expected no update available when cached version equals current")
	}
}
