package plugins

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// SourceFetcher downloads/fetches a plugin from a source.
type SourceFetcher interface {
	Fetch(ctx context.Context, source Source, version string) (*Plugin, error)
}

// GitHubSource downloads plugins from GitHub Releases.
type GitHubSource struct {
	client  *http.Client
	BaseURL string // exported so tests can override
}

// NewGitHubSource creates a GitHubSource with the given HTTP client.
// If client is nil, a default 30-second timeout client is used.
func NewGitHubSource(client *http.Client) *GitHubSource {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	return &GitHubSource{
		client:  client,
		BaseURL: "https://api.github.com",
	}
}

// Fetch downloads a plugin from a GitHub release. If version is "latest" or empty,
// it fetches the latest release; otherwise it fetches the specific version tag.
func (g *GitHubSource) Fetch(ctx context.Context, source Source, version string) (*Plugin, error) {
	owner, repo, ok := strings.Cut(source.Repo, "/")
	if !ok {
		return nil, fmt.Errorf("invalid repo format: %q (expected owner/repo)", source.Repo)
	}

	url := fmt.Sprintf("%s/repos/%s/%s/releases/latest", g.BaseURL, owner, repo)
	if version != "" && version != "latest" {
		url = fmt.Sprintf("%s/repos/%s/%s/releases/tags/v%s", g.BaseURL, owner, repo, version)
	}

	release, err := g.fetchRelease(ctx, url)
	if err != nil {
		return nil, err
	}

	assetURL := g.findAssetURL(release)
	if assetURL == "" {
		return nil, fmt.Errorf("no downloadable asset found in release %s", release.TagName)
	}

	body, err := g.downloadAsset(ctx, assetURL)
	if err != nil {
		return nil, fmt.Errorf("download asset: %w", err)
	}

	ver := strings.TrimPrefix(release.TagName, "v")
	return &Plugin{
		Meta: PluginMeta{
			ID:      source.Repo,
			Name:    source.Repo,
			Version: ver,
			Source:  source,
		},
		Payload: body,
	}, nil
}

type githubRelease struct {
	TagName string        `json:"tag_name"`
	Assets  []githubAsset `json:"assets"`
}

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

func (g *GitHubSource) fetchRelease(ctx context.Context, url string) (*githubRelease, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "claude-go-plugins")

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("plugin not found at %s", url)
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}

	var rel githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, fmt.Errorf("decode release: %w", err)
	}
	return &rel, nil
}

func (g *GitHubSource) findAssetURL(release *githubRelease) string {
	for _, a := range release.Assets {
		name := strings.ToLower(a.Name)
		if strings.HasSuffix(name, ".tar.gz") || strings.HasSuffix(name, ".tgz") {
			return a.BrowserDownloadURL
		}
	}
	if len(release.Assets) > 0 {
		return release.Assets[0].BrowserDownloadURL
	}
	return ""
}

func (g *GitHubSource) downloadAsset(ctx context.Context, url string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "claude-go-plugins")

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download asset: %w", err)
	}
	if resp.StatusCode != 200 {
		resp.Body.Close()
		return nil, fmt.Errorf("download returned %d", resp.StatusCode)
	}
	return resp.Body, nil
}

// LocalSource resolves plugins from local filesystem directories.
type LocalSource struct{}

// NewLocalSource creates a new LocalSource.
func NewLocalSource() *LocalSource {
	return &LocalSource{}
}

// Fetch loads a plugin from a local directory. It reads the manifest and
// wraps the directory contents as a tarball in the payload.
func (l *LocalSource) Fetch(ctx context.Context, source Source, version string) (*Plugin, error) {
	path := source.Path
	if path == "" {
		return nil, fmt.Errorf("local source requires a path")
	}

	manifest, err := LoadManifest(path)
	if err != nil {
		return nil, fmt.Errorf("load manifest from %s: %w", path, err)
	}

	return &Plugin{
		Meta: PluginMeta{
			ID:      manifest.Name,
			Name:    manifest.Name,
			Version: manifest.Version,
			Source:  source,
		},
		Manifest: *manifest,
		Payload:  dirAsTarball(path),
	}, nil
}

// dirAsTarball creates a tar.gz of the directory and returns a pipe reader.
// Errors during archival are propagated to the reader via pw.CloseWithError.
func dirAsTarball(dir string) io.ReadCloser {
	pr, pw := io.Pipe()
	go func() {
		gw := gzip.NewWriter(pw)
		tw := tar.NewWriter(gw)

		walkErr := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(dir, path)
			if err != nil {
				return err
			}
			if rel == "." {
				return nil
			}
			hdr, err := tar.FileInfoHeader(info, "")
			if err != nil {
				return err
			}
			hdr.Name = rel
			if err := tw.WriteHeader(hdr); err != nil {
				return err
			}
			if !info.IsDir() {
				f, err := os.Open(path)
				if err != nil {
					return err
				}
				if _, copyErr := io.Copy(tw, f); copyErr != nil {
					f.Close()
					return copyErr
				}
				if closeErr := f.Close(); closeErr != nil {
					return closeErr
				}
			}
			return nil
		})

		// Always close tar/gzip writers
		closeErr := tw.Close()
		gzCloseErr := gw.Close()

		if walkErr != nil {
			pw.CloseWithError(walkErr)
			return
		}
		if closeErr != nil {
			pw.CloseWithError(closeErr)
			return
		}
		if gzCloseErr != nil {
			pw.CloseWithError(gzCloseErr)
			return
		}
		pw.Close()
	}()
	return pr
}
