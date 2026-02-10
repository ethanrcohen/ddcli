package version

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckForUpdate_CacheHit(t *testing.T) {
	// Write a fresh cache entry so the network call is skipped
	home := t.TempDir()
	t.Setenv("HOME", home)

	entry := CacheEntry{
		LatestVersion: "0.4.0",
		CheckedAt:     time.Now(),
	}
	data, _ := json.Marshal(entry)
	require.NoError(t, os.WriteFile(filepath.Join(home, cacheFileName), data, 0600))

	// Server should never be called
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not hit the server when cache is fresh")
	}))
	defer server.Close()

	result := checkForUpdate("0.3.0", server.URL, server.Client())
	require.NotNil(t, result)
	assert.Equal(t, "0.3.0", result.CurrentVersion)
	assert.Equal(t, "0.4.0", result.LatestVersion)
	assert.True(t, result.IsOutdated)
}

func TestCheckForUpdate_CacheMiss(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(githubRelease{TagName: "v0.5.0"})
	}))
	defer server.Close()

	result := checkForUpdate("0.3.0", server.URL, server.Client())
	require.NotNil(t, result)
	assert.Equal(t, "0.3.0", result.CurrentVersion)
	assert.Equal(t, "0.5.0", result.LatestVersion)
	assert.True(t, result.IsOutdated)

	// Verify cache was written
	cacheData, err := os.ReadFile(filepath.Join(home, cacheFileName))
	require.NoError(t, err)
	var cached CacheEntry
	require.NoError(t, json.Unmarshal(cacheData, &cached))
	assert.Equal(t, "0.5.0", cached.LatestVersion)
}

func TestCheckForUpdate_ExpiredCache(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	// Write an expired cache entry
	entry := CacheEntry{
		LatestVersion: "0.3.0",
		CheckedAt:     time.Now().Add(-5 * time.Hour),
	}
	data, _ := json.Marshal(entry)
	require.NoError(t, os.WriteFile(filepath.Join(home, cacheFileName), data, 0600))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(githubRelease{TagName: "v0.4.0"})
	}))
	defer server.Close()

	result := checkForUpdate("0.3.0", server.URL, server.Client())
	require.NotNil(t, result)
	assert.Equal(t, "0.4.0", result.LatestVersion)
	assert.True(t, result.IsOutdated)
}

func TestCheckForUpdate_APIError(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	result := checkForUpdate("0.3.0", server.URL, server.Client())
	assert.Nil(t, result)
}

func TestCheckForUpdate_SameVersion(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(githubRelease{TagName: "v0.3.0"})
	}))
	defer server.Close()

	result := checkForUpdate("v0.3.0", server.URL, server.Client())
	require.NotNil(t, result)
	assert.False(t, result.IsOutdated)
}

func TestCheckForUpdate_CurrentNewerThanLatest(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	// GitHub returns an older version (e.g. cache or release lag)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(githubRelease{TagName: "v0.3.0"})
	}))
	defer server.Close()

	result := checkForUpdate("0.5.0", server.URL, server.Client())
	require.NotNil(t, result)
	assert.False(t, result.IsOutdated, "should not suggest downgrade from 0.5.0 to 0.3.0")
}

func TestCheckForUpdate_CurrentNewerThanLatest_CacheHit(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	// Stale cached version that is older than current
	entry := CacheEntry{
		LatestVersion: "0.3.0",
		CheckedAt:     time.Now(),
	}
	data, _ := json.Marshal(entry)
	require.NoError(t, os.WriteFile(filepath.Join(home, cacheFileName), data, 0600))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not hit the server when cache is fresh")
	}))
	defer server.Close()

	result := checkForUpdate("0.5.0", server.URL, server.Client())
	require.NotNil(t, result)
	assert.False(t, result.IsOutdated, "should not suggest downgrade from cached 0.3.0 when running 0.5.0")
}

func TestFormatNotice(t *testing.T) {
	assert.Equal(t, "", FormatNotice(nil))
	assert.Equal(t, "", FormatNotice(&CheckResult{IsOutdated: false}))
	assert.Equal(t,
		`Update available: v0.3.0 → v0.4.0 — run "ddcli update" to install`,
		FormatNotice(&CheckResult{
			CurrentVersion: "0.3.0",
			LatestVersion:  "0.4.0",
			IsOutdated:     true,
		}),
	)
}

func TestStripV(t *testing.T) {
	assert.Equal(t, "0.3.0", stripV("v0.3.0"))
	assert.Equal(t, "0.3.0", stripV("0.3.0"))
}
