package plugins

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGitHubSource_DownloadLatest(t *testing.T) {
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/releases/latest") {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{
				"tag_name": "v5.1.0",
				"assets": [{"name": "plugin.tar.gz", "browser_download_url": "` + srv.URL + `/assets/plugin.tar.gz"}]
			}`))
			return
		}
		if strings.Contains(r.URL.Path, "/assets/") {
			w.Write([]byte("fake-tarball-content"))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	gh := NewGitHubSource(&http.Client{})
	gh.BaseURL = srv.URL
	src := Source{Type: SourceGitHubRelease, Repo: "obra/superpowers"}

	plugin, err := gh.Fetch(context.Background(), src, "latest")
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if plugin.Meta.Version != "5.1.0" {
		t.Errorf("expected version 5.1.0, got %q", plugin.Meta.Version)
	}
	if plugin.Payload == nil {
		t.Fatal("expected non-nil payload")
	}
	plugin.Payload.Close()
}

func TestGitHubSource_FetchSpecificVersion(t *testing.T) {
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"tag_name": "v5.0.0",
			"assets": [{"name": "plugin.tar.gz", "browser_download_url": "` + srv.URL + `/assets/plugin.tar.gz"}]
		}`))
	}))
	defer srv.Close()

	gh := NewGitHubSource(&http.Client{})
	gh.BaseURL = srv.URL
	src := Source{Type: SourceGitHubRelease, Repo: "obra/superpowers"}

	plugin, err := gh.Fetch(context.Background(), src, "5.0.0")
	if err != nil {
		t.Fatalf("Fetch specific version: %v", err)
	}
	if plugin.Meta.Version != "5.0.0" {
		t.Errorf("expected version 5.0.0, got %q", plugin.Meta.Version)
	}
}

func TestGitHubSource_RepoNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		w.Write([]byte(`{"message":"Not Found"}`))
	}))
	defer srv.Close()

	gh := NewGitHubSource(&http.Client{})
	gh.BaseURL = srv.URL
	src := Source{Type: SourceGitHubRelease, Repo: "nonexistent/plugin"}

	_, err := gh.Fetch(context.Background(), src, "latest")
	if err == nil {
		t.Fatal("expected error for nonexistent repo")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention 'not found': %v", err)
	}
}

func TestLocalSource_Fetch(t *testing.T) {
	tmp := t.TempDir()
	manifestData := `{"name": "local-plugin", "version": "1.0.0"}`
	os.WriteFile(filepath.Join(tmp, "plugin.json"), []byte(manifestData), 0644)

	ls := NewLocalSource()
	src := Source{Type: SourceLocalPath, Path: tmp}
	plugin, err := ls.Fetch(context.Background(), src, "")
	if err != nil {
		t.Fatalf("Fetch local: %v", err)
	}
	if plugin.Meta.Name != "local-plugin" {
		t.Errorf("expected 'local-plugin', got %q", plugin.Meta.Name)
	}
	plugin.Payload.Close()
}
