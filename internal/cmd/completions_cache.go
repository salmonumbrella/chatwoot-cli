package cmd

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/chatwoot/chatwoot-cli/internal/api"
)

const completionsCacheTTL = 5 * time.Minute

type completionsCache struct {
	CachedAt time.Time        `json:"cached_at"`
	Items    []CompletionItem `json:"items"`
}

func completionsCacheDisabled() bool {
	return completionsNoCache || os.Getenv("CHATWOOT_COMPLETIONS_NO_CACHE") != "" || os.Getenv("CHATWOOT_NO_CACHE") != ""
}

func completionsCacheDir() (string, error) {
	if dir := os.Getenv("CHATWOOT_COMPLETIONS_CACHE_DIR"); dir != "" {
		return dir, nil
	}
	dir := resolveCacheDir()
	if dir == "" {
		return "", fmt.Errorf("could not determine cache directory")
	}
	return dir, nil
}

func completionsCachePath(client *api.Client, key string) (string, error) {
	dir, err := completionsCacheDir()
	if err != nil {
		return "", err
	}
	hash := sha1.Sum([]byte(client.BaseURL))
	suffix := hex.EncodeToString(hash[:6])
	filename := key + "_" + suffix + "_" + strconv.Itoa(client.AccountID) + ".json"
	return filepath.Join(dir, filename), nil
}

func loadCompletionsCache(client *api.Client, key string) ([]CompletionItem, bool) {
	if completionsCacheDisabled() {
		return nil, false
	}
	path, err := completionsCachePath(client, key)
	if err != nil {
		return nil, false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	var cached completionsCache
	if err := json.Unmarshal(data, &cached); err != nil {
		return nil, false
	}
	if time.Since(cached.CachedAt) > completionsCacheTTL {
		return nil, false
	}
	return cached.Items, true
}

func saveCompletionsCache(client *api.Client, key string, items []CompletionItem) {
	if completionsCacheDisabled() {
		return
	}
	path, err := completionsCachePath(client, key)
	if err != nil {
		return
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return
	}
	data, err := json.Marshal(completionsCache{
		CachedAt: time.Now(),
		Items:    items,
	})
	if err != nil {
		return
	}
	_ = os.WriteFile(path, data, 0o644)
}
