package plugins

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// DiskStore implements Store using the local filesystem with TS-compatible layout:
// {cacheDir}/{marketplace}/{pluginName}/{version}/...
// {cacheDir}/{marketplace}/{pluginName}/current -> {version} (symlink)
type DiskStore struct{}

// NewDiskStore creates a new DiskStore.
func NewDiskStore() *DiskStore {
	return &DiskStore{}
}

// Install extracts a plugin tarball into the cache and creates a "current" symlink.
func (s *DiskStore) Install(ctx context.Context, sourcePlugin Plugin, cacheDir string) (*InstalledPlugin, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	marketplace := marketplaceName(sourcePlugin.Source)
	pluginDir := filepath.Join(cacheDir, marketplace, sourcePlugin.Manifest.Name)
	versionDir := filepath.Join(pluginDir, sourcePlugin.Manifest.Version)

	if err := os.MkdirAll(versionDir, 0755); err != nil {
		return nil, fmt.Errorf("create version dir: %w", err)
	}

	if err := extractTarball(sourcePlugin.Payload, versionDir); err != nil {
		os.RemoveAll(versionDir)
		return nil, fmt.Errorf("extract: %w", err)
	}
	sourcePlugin.Payload.Close()

	currentLink := filepath.Join(pluginDir, "current")
	os.Remove(currentLink)
	if err := os.Symlink(versionDir, currentLink); err != nil {
		// Windows fallback: symlink requires Developer Mode or admin privileges.
		// Write a current_version file instead so plugin resolution still works.
		versionFile := filepath.Join(pluginDir, "current_version")
		if err2 := os.WriteFile(versionFile, []byte(sourcePlugin.Manifest.Version), 0644); err2 != nil {
			return nil, fmt.Errorf("create symlink: %w (current_version fallback: %v)", err, err2)
		}
	}

	return &InstalledPlugin{
		PluginMeta:  sourcePlugin.Meta,
		InstallPath: versionDir,
		Enabled:     true,
		Manifest:    sourcePlugin.Manifest,
	}, nil
}

// Uninstall removes the entire plugin directory (all versions).
func (s *DiskStore) Uninstall(ctx context.Context, cacheDir, pluginID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	name, marketplace, _ := strings.Cut(pluginID, "@")
	pluginDir := filepath.Join(cacheDir, marketplace, name)

	if _, err := os.Stat(pluginDir); os.IsNotExist(err) {
		return fmt.Errorf("plugin %q not installed", pluginID)
	}
	return os.RemoveAll(pluginDir)
}

// ResolveCurrentPath resolves the real plugin path from a plugin directory.
// Tries os.Readlink on "current" symlink first; falls back to reading
// a "current_version" file (used on Windows where symlinks need admin/Developer Mode).
func ResolveCurrentPath(pluginDir string) (string, error) {
	currentLink := filepath.Join(pluginDir, "current")
	realPath, err := os.Readlink(currentLink)
	if err == nil {
		if !filepath.IsAbs(realPath) {
			realPath = filepath.Join(pluginDir, realPath)
		}
		return realPath, nil
	}
	versionFile := filepath.Join(pluginDir, "current_version")
	data, err := os.ReadFile(versionFile)
	if err != nil {
		return "", fmt.Errorf("no current symlink or current_version file in %s", pluginDir)
	}
	version := strings.TrimSpace(string(data))
	if version == "" {
		return "", fmt.Errorf("empty current_version file in %s", pluginDir)
	}
	return filepath.Join(pluginDir, version), nil
}

// ListInstalled returns all plugins with a valid "current" symlink or current_version file.
func (s *DiskStore) ListInstalled(ctx context.Context, cacheDir string) ([]InstalledPlugin, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	var result []InstalledPlugin

	marketplaces, err := os.ReadDir(cacheDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	for _, mkt := range marketplaces {
		if !mkt.IsDir() {
			continue
		}
		mktPath := filepath.Join(cacheDir, mkt.Name())
		plugins, _ := os.ReadDir(mktPath)
		for _, p := range plugins {
			if !p.IsDir() {
				continue
			}
			pluginPath := filepath.Join(mktPath, p.Name())
				realPath, err := ResolveCurrentPath(pluginPath)
				if err != nil {
					continue
				}

			manifest, err := LoadManifest(realPath)
			if err != nil {
				continue
			}

			result = append(result, InstalledPlugin{
				PluginMeta: PluginMeta{
					ID:      p.Name() + "@" + mkt.Name(),
					Name:    manifest.Name,
					Version: manifest.Version,
				},
				InstallPath: realPath,
				Enabled:     true,
				Manifest:    *manifest,
			})
		}
	}
	return result, nil
}

// GC removes older versions of each plugin, keeping only the N most recent.
func (s *DiskStore) GC(ctx context.Context, cacheDir string, keepVersions int) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if keepVersions < 1 {
		keepVersions = 1
	}

	removed := 0
	marketplaces, err := os.ReadDir(cacheDir)
	if err != nil {
		return 0, nil
	}

	for _, mkt := range marketplaces {
		if !mkt.IsDir() {
			continue
		}
		mktPath := filepath.Join(cacheDir, mkt.Name())
		plugins, _ := os.ReadDir(mktPath)
		for _, p := range plugins {
			if !p.IsDir() {
				continue
			}
			pluginPath := filepath.Join(mktPath, p.Name())
			versions, _ := os.ReadDir(pluginPath)

			type versionEntry struct {
				name string
				time int64
			}
			var vers []versionEntry
			for _, v := range versions {
				if !v.IsDir() || v.Name() == "current" {
					continue
				}
				info, _ := v.Info()
				vers = append(vers, versionEntry{v.Name(), info.ModTime().UnixNano()})
			}

			if len(vers) <= keepVersions {
				continue
			}

			sort.Slice(vers, func(i, j int) bool { return vers[i].time > vers[j].time })
			for _, v := range vers[keepVersions:] {
				os.RemoveAll(filepath.Join(pluginPath, v.name))
				removed++
			}
		}
	}
	return removed, nil
}

// marketplaceName extracts the marketplace segment from a source repo string.
// For "owner/repo" it returns "owner"; for empty repo it returns "local".
func marketplaceName(source Source) string {
	if source.Repo != "" {
		parts := strings.Split(source.Repo, "/")
		if len(parts) > 0 {
			return parts[0]
		}
	}
	return "local"
}

// extractTarball decompresses a .tar.gz stream into dest directory.
// Includes path traversal protection.
func extractTarball(r io.Reader, dest string) error {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("gzip reader: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("tar next: %w", err)
		}

		clean := filepath.Clean(hdr.Name)
		if strings.HasPrefix(clean, "..") {
			continue
		}

		target := filepath.Join(dest, clean)
		// Ensure target is within dest (catches absolute paths and traversal)
		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(dest)+string(filepath.Separator)) && filepath.Clean(target) != filepath.Clean(dest) {
			continue
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			os.MkdirAll(target, 0755)
		case tar.TypeReg:
			os.MkdirAll(filepath.Dir(target), 0755)
			f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(hdr.Mode&0777))
			if err != nil {
				return fmt.Errorf("create file %s: %w", target, err)
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return err
			}
			f.Close()
		}
	}
	return nil
}
