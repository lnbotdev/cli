package update

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	repo          = "lnbotdev/cli"
	checkTimeout  = 3 * time.Second
	checkInterval = 24 * time.Hour
)

type cachedCheck struct {
	Latest    string `json:"latest"`
	CheckedAt int64  `json:"checked_at"`
}

func cacheFile() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "lnbot", ".update-check")
}

func CheckForUpdate(current string) (latest string, available bool) {
	if os.Getenv("LNBOT_NO_UPDATE_CHECK") != "" {
		return "", false
	}

	if data, err := os.ReadFile(cacheFile()); err == nil {
		var cached cachedCheck
		if json.Unmarshal(data, &cached) == nil {
			if time.Since(time.Unix(cached.CheckedAt, 0)) < checkInterval {
				if cached.Latest != "" && isNewer(cached.Latest, current) {
					return cached.Latest, true
				}
				return "", false
			}
		}
	}

	latest, err := fetchLatest()
	if err != nil {
		return "", false
	}

	saveCache(latest)

	if isNewer(latest, current) {
		return latest, true
	}
	return "", false
}

// isNewer returns true if latest is a higher semver than current.
func isNewer(latest, current string) bool {
	l := parseVersion(latest)
	c := parseVersion(current)
	for i := 0; i < 3; i++ {
		if l[i] > c[i] {
			return true
		}
		if l[i] < c[i] {
			return false
		}
	}
	return false
}

func parseVersion(v string) [3]int {
	v = strings.TrimPrefix(v, "v")
	parts := strings.SplitN(v, ".", 3)
	var out [3]int
	for i := 0; i < len(parts) && i < 3; i++ {
		out[i], _ = strconv.Atoi(parts[i])
	}
	return out
}

func fetchLatest() (string, error) {
	client := &http.Client{Timeout: checkTimeout}
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)

	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("status %d", resp.StatusCode)
	}

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}

	tag := release.TagName
	if len(tag) > 0 && tag[0] == 'v' {
		tag = tag[1:]
	}
	return tag, nil
}

func saveCache(latest string) {
	data, _ := json.Marshal(cachedCheck{
		Latest:    latest,
		CheckedAt: time.Now().Unix(),
	})
	p := cacheFile()
	os.MkdirAll(filepath.Dir(p), 0o700)
	os.WriteFile(p, data, 0o600)
}
