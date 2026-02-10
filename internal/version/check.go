package version

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
	cacheFileName = ".ddcli_update_check.json"
	cacheTTL      = 4 * time.Hour
	githubTimeout = 3 * time.Second
	releaseURL    = "https://api.github.com/repos/ethanrcohen/ddcli/releases/latest"
)

type CheckResult struct {
	CurrentVersion string
	LatestVersion  string
	IsOutdated     bool
}

type CacheEntry struct {
	LatestVersion string    `json:"latest_version"`
	CheckedAt     time.Time `json:"checked_at"`
}

type githubRelease struct {
	TagName string `json:"tag_name"`
}

func cachePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, cacheFileName), nil
}

// CheckForUpdate checks GitHub for a newer release. It caches results for 4 hours
// and uses a 3-second HTTP timeout. Any error returns nil silently.
func CheckForUpdate(currentVersion string) *CheckResult {
	return checkForUpdate(currentVersion, releaseURL, nil)
}

// checkForUpdate is the internal implementation that accepts a custom URL and HTTP client for testing.
func checkForUpdate(currentVersion, url string, client *http.Client) *CheckResult {
	if client == nil {
		client = &http.Client{Timeout: githubTimeout}
	}

	// Try cache first
	if entry, err := readCache(); err == nil {
		if time.Since(entry.CheckedAt) < cacheTTL {
			latest := stripV(entry.LatestVersion)
			current := stripV(currentVersion)
			return &CheckResult{
				CurrentVersion: current,
				LatestVersion:  latest,
				IsOutdated:     isNewer(latest, current),
			}
		}
	}

	// Fetch from GitHub
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil
	}

	latest := stripV(release.TagName)

	// Write cache (best-effort)
	_ = writeCache(CacheEntry{
		LatestVersion: latest,
		CheckedAt:     time.Now(),
	})

	current := stripV(currentVersion)
	return &CheckResult{
		CurrentVersion: current,
		LatestVersion:  latest,
		IsOutdated:     isNewer(latest, current),
	}
}

func FormatNotice(r *CheckResult) string {
	if r == nil || !r.IsOutdated {
		return ""
	}
	return fmt.Sprintf("Update available: v%s → v%s — run \"ddcli update\" to install", r.CurrentVersion, r.LatestVersion)
}

func readCache() (*CacheEntry, error) {
	path, err := cachePath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var entry CacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, err
	}
	return &entry, nil
}

func writeCache(entry CacheEntry) error {
	path, err := cachePath()
	if err != nil {
		return err
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func stripV(s string) string {
	return strings.TrimPrefix(s, "v")
}

// isNewer returns true if latest is a higher semver than current.
func isNewer(latest, current string) bool {
	lp := parseSemver(latest)
	cp := parseSemver(current)
	for i := 0; i < 3; i++ {
		if lp[i] != cp[i] {
			return lp[i] > cp[i]
		}
	}
	return false
}

func parseSemver(v string) [3]int {
	var parts [3]int
	for i, s := range strings.SplitN(stripV(v), ".", 3) {
		parts[i], _ = strconv.Atoi(s)
	}
	return parts
}
