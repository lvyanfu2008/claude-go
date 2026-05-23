package plugins

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCatalog_ResolvePlugin(t *testing.T) {
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"tag_name": "v5.1.0",
			"assets": [{"name": "plugin.tar.gz", "browser_download_url": "` + srv.URL + `/assets/plugin.tar.gz"}]
		}`))
	}))
	defer srv.Close()

	cat := NewGitHubCatalog(&http.Client{})
	cat.BaseURL = srv.URL
	src := Source{Type: SourceGitHubRelease, Repo: "obra/superpowers"}

	plugin, err := cat.Resolve(context.Background(), src, "latest")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if plugin.Meta.Version != "5.1.0" {
		t.Errorf("expected 5.1.0, got %q", plugin.Meta.Version)
	}
	plugin.Payload.Close()
}

func TestResolveSource_GitHub(t *testing.T) {
	src, err := ResolveSource("github:obra/superpowers")
	if err != nil {
		t.Fatalf("ResolveSource: %v", err)
	}
	if src.Type != SourceGitHubRelease {
		t.Errorf("expected github-release, got %q", src.Type)
	}
	if src.Repo != "obra/superpowers" {
		t.Errorf("expected 'obra/superpowers', got %q", src.Repo)
	}
}

func TestResolveSource_Local(t *testing.T) {
	src, err := ResolveSource("local:/tmp/my-plugin")
	if err != nil {
		t.Fatalf("ResolveSource: %v", err)
	}
	if src.Type != SourceLocalPath {
		t.Errorf("expected local-path, got %q", src.Type)
	}
	if src.Path != "/tmp/my-plugin" {
		t.Errorf("expected '/tmp/my-plugin', got %q", src.Path)
	}
}

func TestResolveSource_Invalid(t *testing.T) {
	_, err := ResolveSource("invalid-ref")
	if err == nil {
		t.Fatal("expected error for invalid source ref")
	}
}
