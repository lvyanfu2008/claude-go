package commands

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// findMarkdownFilesNative mirrors src/utils/markdownConfigLoader.ts findMarkdownFilesNative:
// collect *.md under root, follow symbolic links to directories, skip cycles via canonical path keys.
// Does not use .gitignore (same as TS ripgrep --no-ignore and the native finder comment there).
func findMarkdownFilesNative(root string) ([]string, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, nil
	}
	st, err := os.Stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if !st.IsDir() {
		return nil, nil
	}
	visited := make(map[string]struct{})
	var out []string
	var walk func(string) error
	walk = func(dir string) error {
		key, err := canonicalDirKey(dir)
		if err != nil {
			return nil
		}
		if _, ok := visited[key]; ok {
			return nil
		}
		visited[key] = struct{}{}

		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil
		}
		for _, ent := range entries {
			path := filepath.Join(dir, ent.Name())
			switch {
			case ent.IsDir():
				_ = walk(path)
			case ent.Type()&fs.ModeSymlink != 0:
				info, err := os.Stat(path)
				if err != nil {
					continue
				}
				if info.IsDir() {
					_ = walk(path)
				} else if strings.EqualFold(filepath.Ext(ent.Name()), ".md") {
					out = append(out, path)
				}
			default:
				if strings.EqualFold(filepath.Ext(ent.Name()), ".md") {
					out = append(out, path)
				}
			}
		}
		return nil
	}
	if err := walk(root); err != nil {
		return nil, err
	}
	return out, nil
}

func canonicalDirKey(dir string) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return filepath.Clean(abs), nil
	}
	return filepath.Clean(resolved), nil
}
