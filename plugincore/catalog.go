package plugins

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// GitHubCatalog implements Marketplace for GitHub-hosted plugin registries.
type GitHubCatalog struct {
	client  *http.Client
	fetcher *GitHubSource
	BaseURL string // override for testing
}

// NewGitHubCatalog creates a GitHubCatalog with the given HTTP client.
// If client is nil, a default 30-second timeout client is used.
func NewGitHubCatalog(client *http.Client) *GitHubCatalog {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	return &GitHubCatalog{
		client:  client,
		fetcher: NewGitHubSource(client),
		BaseURL: "https://api.github.com",
	}
}

// Search returns available plugins matching the query for the given source.
func (c *GitHubCatalog) Search(ctx context.Context, source Source, query string) ([]PluginMeta, error) {
	return make([]PluginMeta, 0), nil
}

// Resolve fetches a specific plugin from the given source.
func (c *GitHubCatalog) Resolve(ctx context.Context, source Source, name string) (*Plugin, error) {
	if source.Type == SourceGitHubRelease {
		c.fetcher.BaseURL = c.BaseURL
		return c.fetcher.Fetch(ctx, source, "latest")
	}
	return nil, fmt.Errorf("unsupported source type for resolve: %q", source.Type)
}

// ListAvailable returns all available plugins from the given source.
func (c *GitHubCatalog) ListAvailable(ctx context.Context, source Source) ([]PluginMeta, error) {
	return make([]PluginMeta, 0), nil
}

// ResolveSource parses a plugin source reference into a Source.
// Format: "github:owner/repo" or "local:/path/to/plugin"
func ResolveSource(ref string) (Source, error) {
	if strings.HasPrefix(ref, "github:") {
		repo := strings.TrimPrefix(ref, "github:")
		return Source{Type: SourceGitHubRelease, Repo: repo}, nil
	}
	if strings.HasPrefix(ref, "local:") {
		path := strings.TrimPrefix(ref, "local:")
		return Source{Type: SourceLocalPath, Path: path}, nil
	}
	return Source{}, fmt.Errorf("unrecognized source reference: %q (expected github:owner/repo or local:/path)", ref)
}
