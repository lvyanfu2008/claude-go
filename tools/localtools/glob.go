package localtools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

const globMaxFiles = 100

// GlobFromJSON lists files matching pattern under path (default: first root). Prefer ripgrep when available.
func GlobFromJSON(ctx context.Context, raw []byte, roots []string) (string, bool, error) {
	var in struct {
		Pattern string `json:"pattern"`
		Path    string `json:"path"`
	}
	if err := json.Unmarshal(raw, &in); err != nil {
		return "", true, err
	}
	pat := strings.TrimSpace(in.Pattern)
	if pat == "" {
		return "", true, fmt.Errorf("empty pattern")
	}
	base, err := ResolveDirUnderRoots(strings.TrimSpace(in.Path), roots)
	if err != nil {
		return "", true, err
	}

	var matches []string
	if path, err := exec.LookPath("rg"); err == nil && path != "" {
		out, err := rgGlob(ctx, base, pat)
		if err == nil {
			matches = out
		}
	}
	if matches == nil {
		var err error
		matches, err = filepath.Glob(filepath.Join(base, pat))
		if err != nil {
			return "", true, err
		}
	}
	sort.Strings(matches)
	truncated := false
	if len(matches) > globMaxFiles {
		truncated = true
		matches = matches[:globMaxFiles]
	}
	if len(matches) == 0 {
		return "No files found", false, nil
	}
	var b strings.Builder
	for _, m := range matches {
		b.WriteString(m)
		b.WriteByte('\n')
	}
	s := strings.TrimSuffix(b.String(), "\n")
	if truncated {
		s += "\n(Results are truncated. Consider using a more specific path or pattern.)"
	}
	return s, false, nil
}

func rgGlob(ctx context.Context, base, pattern string) ([]string, error) {
	cctx := ctx
	if cctx == nil {
		cctx = context.Background()
	}
	cmd := exec.CommandContext(cctx, "rg", "--files", "--glob", pattern, "--sort", "path", base)
	out, err := cmd.Output()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) && ee.ExitCode() == 1 {
			return []string{}, nil
		}
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return []string{}, nil
	}
	return lines, nil
}
