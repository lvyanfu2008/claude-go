// Package plugins provides the core plugin system types and interfaces
// for Claude Code plugin management. It handles marketplace discovery,
// installation, and plugin metadata parsing.
package plugins

import (
	"context"
	"encoding/json"
	"io"
)

// SourceType enumerates supported plugin sources.
type SourceType string

const (
	SourceGitHubRelease SourceType = "github-release"
	SourceLocalPath     SourceType = "local-path"
)

// Source describes where a plugin is fetched from.
type Source struct {
	Type    SourceType `json:"type"`
	Repo    string     `json:"repo,omitempty"`    // "obra/superpowers"
	Path    string     `json:"path,omitempty"`    // local dir (type=local-path)
	BaseURL string     `json:"baseUrl,omitempty"` // custom marketplace base URL
}

// PluginManifest mirrors TS PluginManifest. Go canonical name is plugin.json;
// parser also falls back to package.json for TS/npm-origin plugins.
type PluginManifest struct {
	Name        string             `json:"name"`
	Description string             `json:"description,omitempty"`
	Version     string             `json:"version"`
	Author      string             `json:"author,omitempty"`
	License     string             `json:"license,omitempty"`
	Main        string             `json:"main,omitempty"`     // JS entrypoint (Go ignores)
	Skills      []string           `json:"skills,omitempty"`   // relative paths to skill dirs
	Hooks       json.RawMessage    `json:"hooks,omitempty"`    // raw hooks.json content
	MCPServers  json.RawMessage    `json:"mcpServers,omitempty"`
	DependsOn   []string           `json:"dependsOn,omitempty"`
	Engines     *EnginesConstraint `json:"engines,omitempty"`
}

// EnginesConstraint specifies compatibility requirements.
type EnginesConstraint struct {
	Claude string `json:"claude,omitempty"` // semver range
}

// PluginMeta is marketplace-returned summary without full manifest.
type PluginMeta struct {
	ID          string `json:"id"`   // "superpowers@claude-plugins-official"
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     string `json:"version"`
	Source      Source `json:"source"`
	Downloads   int    `json:"downloads,omitempty"`
}

// InstalledPlugin represents a plugin in the local cache.
type InstalledPlugin struct {
	PluginMeta
	InstallPath string         `json:"installPath"`
	Enabled     bool           `json:"enabled"`
	Manifest    PluginManifest `json:"manifest"`
}

// Plugin is a resolved plugin ready for installation.
// Meta.Source describes the marketplace origin (where the plugin was discovered).
// The top-level Source field describes how to fetch the plugin payload (download source).
type Plugin struct {
	Meta     PluginMeta
	Manifest PluginManifest
	Source   Source
	Payload  io.ReadCloser // downloaded tarball/zip
}

// Marketplace searches and resolves plugins from a source.
type Marketplace interface {
	Search(ctx context.Context, source Source, query string) ([]PluginMeta, error)
	Resolve(ctx context.Context, source Source, name string) (*Plugin, error)
	ListAvailable(ctx context.Context, source Source) ([]PluginMeta, error)
}

// Store manages local plugin installation and cache.
type Store interface {
	Install(ctx context.Context, plugin Plugin, cacheDir string) (*InstalledPlugin, error)
	Uninstall(ctx context.Context, cacheDir, pluginID string) error
	ListInstalled(ctx context.Context, cacheDir string) ([]InstalledPlugin, error)
	GC(ctx context.Context, cacheDir string, keepVersions int) (removed int, err error)
}
