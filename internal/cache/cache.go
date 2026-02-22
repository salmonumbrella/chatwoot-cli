// Package cache provides a generic file-based cache for API responses.
//
// Cache files are JSON, scoped per resource type, server URL, and account ID.
// Default TTL is 5 minutes. Disable with CHATWOOT_NO_CACHE=1.
package cache

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const DefaultTTL = 5 * time.Minute

type entry struct {
	CachedAt time.Time       `json:"cached_at"`
	Items    json.RawMessage `json:"items"`
}

// Store reads and writes a single cache key (resource+server+account).
type Store struct {
	path string
	ttl  time.Duration
}

// NewStore creates a Store with the default 5-minute TTL.
// dir is the cache directory (typically from DefaultDir).
// key is the resource type (e.g. "inboxes").
// baseURL is the Chatwoot server URL.
// accountID is the Chatwoot account ID.
func NewStore(dir, key, baseURL string, accountID int) *Store {
	return NewStoreWithTTL(dir, key, baseURL, accountID, DefaultTTL)
}

// NewStoreWithTTL creates a Store with a custom TTL.
func NewStoreWithTTL(dir, key, baseURL string, accountID int, ttl time.Duration) *Store {
	key = sanitizeKey(key)
	hash := sha1.Sum([]byte(baseURL))
	suffix := hex.EncodeToString(hash[:6]) // matches completions cache key scheme
	filename := fmt.Sprintf("%s_%s_%d.json", key, suffix, accountID)
	return &Store{
		path: filepath.Join(dir, filename),
		ttl:  ttl,
	}
}

// Get loads cached items into dst. Returns false on miss (no file, expired, disabled).
func (s *Store) Get(dst any) bool {
	if disabled() {
		return false
	}
	data, err := os.ReadFile(s.path)
	if err != nil {
		return false
	}
	var e entry
	if err := json.Unmarshal(data, &e); err != nil {
		return false
	}
	if time.Since(e.CachedAt) > s.ttl {
		return false
	}
	return json.Unmarshal(e.Items, dst) == nil
}

// Put writes items to the cache. Silently no-ops on error or when disabled.
func (s *Store) Put(items any) {
	if disabled() {
		return
	}
	raw, err := json.Marshal(items)
	if err != nil {
		return
	}
	data, err := json.Marshal(entry{
		CachedAt: time.Now(),
		Items:    raw,
	})
	if err != nil {
		return
	}
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return
	}

	// Atomic-ish write: write temp then rename.
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		_ = os.Remove(tmp)
		return
	}
	_ = os.Rename(tmp, s.path)
}

// Clear removes this cache file.
func (s *Store) Clear() {
	_ = os.Remove(s.path)
}

// ClearAll removes all cache files from the directory.
// For safety, it only removes files matching this project's cache filename scheme.
func ClearAll(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !isCacheFilename(name) {
			continue
		}
		_ = os.Remove(filepath.Join(dir, name))
	}
}

// DefaultDir returns the platform-appropriate cache directory.
// Returns "$XDG_CACHE_HOME/chatwoot-cli" or equivalent.
func DefaultDir() (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "chatwoot-cli"), nil
}

func disabled() bool {
	return os.Getenv("CHATWOOT_NO_CACHE") != ""
}

func sanitizeKey(key string) string {
	key = strings.TrimSpace(key)
	if key == "" {
		return "cache"
	}
	key = strings.ReplaceAll(key, "/", "-")
	key = strings.ReplaceAll(key, "\\", "-")
	return key
}

func isCacheFilename(name string) bool {
	// Expected: "<key>_<12hex>_<account>.json"
	if filepath.Ext(name) != ".json" {
		return false
	}
	base := strings.TrimSuffix(name, ".json")
	parts := strings.Split(base, "_")
	if len(parts) != 3 {
		return false
	}
	if parts[0] == "" {
		return false
	}
	if len(parts[1]) != 12 || !isHex(parts[1]) {
		return false
	}
	if _, err := strconv.Atoi(parts[2]); err != nil {
		return false
	}
	return true
}

func isHex(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= '0' && c <= '9':
		case c >= 'a' && c <= 'f':
		case c >= 'A' && c <= 'F':
		default:
			return false
		}
	}
	return true
}
