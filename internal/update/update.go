package update

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const (
	repo         = "lnbotdev/cli"
	checkTimeout = 3 * time.Second
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

	// Check cache first
	if data, err := os.ReadFile(cacheFile()); err == nil {
		var cached cachedCheck
		if json.Unmarshal(data, &cached) == nil {
			if time.Since(time.Unix(cached.CheckedAt, 0)) < checkInterval {
				if cached.Latest != "" && cached.Latest != current {
					return cached.Latest, true
				}
				return "", false
			}
		}
	}

	// Fetch latest release from GitHub
	latest, err := fetchLatest()
	if err != nil {
		return "", false
	}

	// Cache the result
	saveCache(latest)

	if latest != current && latest != "" {
		return latest, true
	}
	return "", false
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
