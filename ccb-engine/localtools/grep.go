package localtools

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

var grepSkipDirNames = map[string]bool{
	".git": true, "node_modules": true, "vendor": true,
}

// GrepFromJSON runs ripgrep when available; otherwise a small Go walker (subset of TS Grep).
func GrepFromJSON(ctx context.Context, raw []byte, roots []string) (string, bool, error) {
	var in struct {
		Pattern     string `json:"pattern"`
		Path        string `json:"path"`
		Glob        string `json:"glob"`
		OutputMode  string `json:"output_mode"`
		CaseFold    bool   `json:"-i"`
		HeadLimit   int    `json:"head_limit"`
		Offset      int    `json:"offset"`
		Multiline   bool   `json:"multiline"`
	}
	if err := json.Unmarshal(raw, &in); err != nil {
		return "", true, err
	}
	pat := strings.TrimSpace(in.Pattern)
	if pat == "" {
		return "", true, fmt.Errorf("empty pattern")
	}
	searchPath := strings.TrimSpace(in.Path)
	base, err := ResolveDirUnderRoots(searchPath, roots)
	if err != nil {
		return "", true, err
	}
	mode := strings.TrimSpace(in.OutputMode)
	if mode == "" {
		mode = "files_with_matches"
	}
	head := in.HeadLimit
	if head <= 0 {
		head = 250
	}
	off := in.Offset
	if off < 0 {
		off = 0
	}

	if path, err := exec.LookPath("rg"); err == nil && path != "" {
		s, err := rgGrep(ctx, base, pat, in.Glob, mode, in.CaseFold, in.Multiline, head, off)
		if err == nil {
			return s, false, nil
		}
	}
	return goGrep(ctx, base, pat, in.Glob, mode, in.CaseFold, in.Multiline, head, off)
}

func rgGrep(ctx context.Context, base, pattern, globPat, mode string, caseFold, multiline bool, headLimit, offset int) (string, error) {
	cctx := ctx
	if cctx == nil {
		cctx = context.Background()
	}
	args := []string{"--no-heading"}
	if caseFold {
		args = append(args, "-i")
	}
	if multiline {
		args = append(args, "-U", "--multiline-dotall")
	}
	if strings.TrimSpace(globPat) != "" {
		args = append(args, "--glob", globPat)
	}
	switch mode {
	case "content":
		args = append(args, "-n")
		args = append(args, "--max-count", fmt.Sprintf("%d", max(1, headLimit)))
	case "count":
		args = append(args, "--count-matches")
	case "files_with_matches":
		args = append(args, "-l")
	default:
		args = append(args, "-l")
	}
	args = append(args, pattern, base)
	cmd := exec.CommandContext(cctx, "rg", args...)
	out, err := cmd.Output()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) && ee.ExitCode() == 1 {
			return "No matches found", nil
		}
		if errors.As(err, &ee) && len(ee.Stderr) > 0 {
			return "", fmt.Errorf("%w: %s", err, strings.TrimSpace(string(ee.Stderr)))
		}
		return "", err
	}
	text := strings.TrimSpace(string(out))
	if text == "" {
		return "No matches found", nil
	}
	lines := strings.Split(text, "\n")
	if offset > 0 && offset < len(lines) {
		lines = lines[offset:]
	}
	if headLimit > 0 && len(lines) > headLimit {
		lines = lines[:headLimit]
		lines = append(lines, fmt.Sprintf("… (truncated to %d lines)", headLimit))
	}
	return strings.Join(lines, "\n"), nil
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func goGrep(ctx context.Context, base, pattern, globPat, mode string, caseFold, multiline bool, headLimit, offset int) (string, bool, error) {
	var prefix string
	if caseFold {
		prefix += "(?i)"
	}
	if multiline {
		prefix += "(?s)"
	}
	re, err := regexp.Compile(prefix + pattern)
	if err != nil {
		return "", true, fmt.Errorf("invalid pattern: %w", err)
	}
	globRe := (*regexp.Regexp)(nil)
	if strings.TrimSpace(globPat) != "" {
		globRe, err = globToRegex(globPat)
		if err != nil {
			return "", true, err
		}
	}

	type fileHit struct {
		path  string
		lines []string
		count int
	}
	var hits []fileHit
	_ = ctx

	errWalk := filepath.WalkDir(base, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if grepSkipDirNames[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if globRe != nil && !globRe.MatchString(filepath.Base(path)) {
			return nil
		}
		fi, err := d.Info()
		if err != nil || fi.Size() > 2<<20 {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		s := string(data)
		switch mode {
		case "files_with_matches":
			if re.MatchString(s) {
				hits = append(hits, fileHit{path: path})
			}
		case "count":
			n := len(re.FindAllString(s, -1))
			if n > 0 {
				hits = append(hits, fileHit{path: path, count: n})
			}
		default: // content
			sc := bufio.NewScanner(strings.NewReader(s))
			sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
			ln := 0
			var lines []string
			for sc.Scan() {
				ln++
				line := sc.Text()
				if re.MatchString(line) {
					lines = append(lines, fmt.Sprintf("%d:%s", ln, line))
				}
			}
			if len(lines) > 0 {
				hits = append(hits, fileHit{path: path, lines: lines})
			}
		}
		return nil
	})
	if errWalk != nil {
		return "", true, errWalk
	}

	var out []string
	switch mode {
	case "files_with_matches":
		for _, h := range hits {
			out = append(out, h.path)
		}
	case "count":
		for _, h := range hits {
			out = append(out, fmt.Sprintf("%s:%d", h.path, h.count))
		}
	default:
		for _, h := range hits {
			for _, ln := range h.lines {
				out = append(out, h.path+":"+ln)
			}
		}
	}
	if len(out) == 0 {
		return "No matches found", false, nil
	}
	if offset > 0 && offset < len(out) {
		out = out[offset:]
	}
	if headLimit > 0 && len(out) > headLimit {
		out = append(out[:headLimit], fmt.Sprintf("… (truncated to %d lines)", headLimit))
	}
	return strings.Join(out, "\n"), false, nil
}

// globToRegex is a tiny adapter for "*.go" style globs in the pure-Go fallback.
func globToRegex(g string) (*regexp.Regexp, error) {
	g = strings.TrimSpace(g)
	if g == "" {
		return regexp.Compile(".*")
	}
	var b strings.Builder
	b.WriteByte('^')
	for i := 0; i < len(g); i++ {
		switch g[i] {
		case '*':
			b.WriteString(".*")
		case '?':
			b.WriteByte('.')
		case '.', '+', '(', ')', '[', ']', '{', '}', '^', '$', '|', '\\':
			b.WriteByte('\\')
			b.WriteByte(g[i])
		default:
			b.WriteByte(g[i])
		}
	}
	b.WriteByte('$')
	return regexp.Compile(b.String())
}
