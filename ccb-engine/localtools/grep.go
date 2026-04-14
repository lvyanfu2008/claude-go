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
	"sort"
	"strings"
)

var grepSkipDirNames = map[string]bool{
	".git": true, "node_modules": true, "vendor": true,
}

type grepToolInput struct {
	Pattern           string `json:"pattern"`
	Path              string `json:"path"`
	Glob              string `json:"glob"`
	OutputMode        string `json:"output_mode"`
	TypeFilter        string `json:"type"`
	CaseFold          bool   `json:"-i"`
	HeadLimit         *int   `json:"head_limit"`
	Offset            int    `json:"offset"`
	Multiline         bool   `json:"multiline"`
	ContextBefore     *int   `json:"-B"`
	ContextAfter      *int   `json:"-A"`
	ContextC          *int   `json:"-C"`
	Context           *int   `json:"context"`
	ShowLineNumbers   *bool  `json:"-n"`
}

type grepStructuredOutput struct {
	Mode          string   `json:"mode,omitempty"`
	NumFiles      int      `json:"numFiles"`
	Filenames     []string `json:"filenames"`
	Content       string   `json:"content,omitempty"`
	NumLines      *int     `json:"numLines,omitempty"`
	NumMatches    *int     `json:"numMatches,omitempty"`
	AppliedLimit  *int     `json:"appliedLimit,omitempty"`
	AppliedOffset *int     `json:"appliedOffset,omitempty"`
}

// GrepFromJSON runs ripgrep when available; otherwise a small Go walker (subset of TS Grep).
// On success it returns JSON matching the TS Grep tool output shape (for toolUseResult),
// with paths relative to the primary workspace root when under that cwd (mirrors toRelativePath).
func GrepFromJSON(ctx context.Context, raw []byte, roots []string) (string, bool, error) {
	var in grepToolInput
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
	off := in.Offset
	if off < 0 {
		off = 0
	}
	cwd := primaryRootAbs(roots)

	var out grepStructuredOutput
	if path, err := exec.LookPath("rg"); err == nil && path != "" {
		out, err = rgGrepStructured(ctx, base, pat, in, mode, cwd)
		if err == nil {
			b, mErr := json.Marshal(out)
			if mErr != nil {
				return "", true, mErr
			}
			return string(b), false, nil
		}
	}
	out, isErr, err := goGrepStructured(ctx, base, pat, in, mode, cwd)
	if err != nil || isErr {
		return "", isErr, err
	}
	b, mErr := json.Marshal(out)
	if mErr != nil {
		return "", true, mErr
	}
	return string(b), false, nil
}

func primaryRootAbs(roots []string) string {
	if len(roots) == 0 || strings.TrimSpace(roots[0]) == "" {
		a, _ := filepath.Abs(".")
		return a
	}
	a, err := filepath.Abs(strings.TrimSpace(roots[0]))
	if err != nil {
		a, _ = filepath.Abs(".")
	}
	return a
}

func rgGrepStructured(ctx context.Context, base, pattern string, in grepToolInput, mode, cwd string) (grepStructuredOutput, error) {
	cctx := ctx
	if cctx == nil {
		cctx = context.Background()
	}
	args := []string{"--no-heading", "--hidden"}
	for _, dir := range []string{".git", ".svn", ".hg", ".bzr", ".jj", ".sl"} {
		args = append(args, "--glob", "!"+dir)
	}
	args = append(args, "--max-columns", "500")
	if in.Multiline {
		args = append(args, "-U", "--multiline-dotall")
	}
	if in.CaseFold {
		args = append(args, "-i")
	}
	showNums := true
	if in.ShowLineNumbers != nil {
		showNums = *in.ShowLineNumbers
	}
	switch mode {
	case "content":
		if showNums {
			args = append(args, "-n")
		}
		if in.Context != nil {
			args = append(args, "-C", fmt.Sprintf("%d", *in.Context))
		} else if in.ContextC != nil {
			args = append(args, "-C", fmt.Sprintf("%d", *in.ContextC))
		} else {
			if in.ContextBefore != nil {
				args = append(args, "-B", fmt.Sprintf("%d", *in.ContextBefore))
			}
			if in.ContextAfter != nil {
				args = append(args, "-A", fmt.Sprintf("%d", *in.ContextAfter))
			}
		}
	case "count":
		args = append(args, "-c")
	case "files_with_matches":
		args = append(args, "-l")
	default:
		args = append(args, "-l")
	}
	if strings.TrimSpace(in.TypeFilter) != "" {
		args = append(args, "--type", strings.TrimSpace(in.TypeFilter))
	}
	if strings.TrimSpace(in.Glob) != "" {
		for _, g := range splitGlobPatterns(in.Glob) {
			args = append(args, "--glob", g)
		}
	}
	if strings.HasPrefix(strings.TrimSpace(pattern), "-") {
		args = append(args, "-e", pattern)
	} else {
		args = append(args, pattern)
	}
	args = append(args, base)
	cmd := exec.CommandContext(cctx, "rg", args...)
	outBytes, err := cmd.Output()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) && ee.ExitCode() == 1 {
			return emptyGrepStructured(mode, offPtr(in)), nil
		}
		if errors.As(err, &ee) && len(ee.Stderr) > 0 {
			return grepStructuredOutput{}, fmt.Errorf("%w: %s", err, strings.TrimSpace(string(ee.Stderr)))
		}
		return grepStructuredOutput{}, err
	}
	text := strings.TrimSpace(string(outBytes))
	var lines []string
	if text != "" {
		lines = strings.Split(text, "\n")
	}
	return finalizeGrepLines(lines, mode, in, cwd)
}

func splitGlobPatterns(glob string) []string {
	glob = strings.TrimSpace(glob)
	if glob == "" {
		return nil
	}
	var out []string
	for _, raw := range strings.Fields(glob) {
		if raw == "" {
			continue
		}
		if strings.Contains(raw, "{") && strings.Contains(raw, "}") {
			out = append(out, raw)
			continue
		}
		for _, p := range strings.Split(raw, ",") {
			p = strings.TrimSpace(p)
			if p != "" {
				out = append(out, p)
			}
		}
	}
	return out
}

func offPtr(in grepToolInput) int {
	o := in.Offset
	if o < 0 {
		return 0
	}
	return o
}

func emptyGrepStructured(mode string, offset int) grepStructuredOutput {
	z := 0
	out := grepStructuredOutput{Mode: mode, Filenames: []string{}}
	if offset > 0 {
		o := offset
		out.AppliedOffset = &o
	}
	switch mode {
	case "content":
		out.NumFiles = 0
		out.NumLines = &z
	case "count":
		out.NumFiles = 0
		out.NumMatches = &z
	default:
		out.NumFiles = 0
	}
	return out
}

func finalizeGrepLines(lines []string, mode string, in grepToolInput, cwd string) (grepStructuredOutput, error) {
	off := offPtr(in)
	var appliedOff *int
	if off > 0 {
		appliedOff = &off
	}
	switch mode {
	case "content":
		limited, appliedLimit := GrepApplyHeadLimit(lines, in.HeadLimit, off)
		finalLines := make([]string, 0, len(limited))
		for _, line := range limited {
			finalLines = append(finalLines, relativizeRgLinePath(line, cwd))
		}
		content := strings.Join(finalLines, "\n")
		nl := len(finalLines)
		out := grepStructuredOutput{
			Mode:          "content",
			NumFiles:      0,
			Filenames:     []string{},
			Content:       content,
			NumLines:      &nl,
			AppliedLimit:  appliedLimit,
			AppliedOffset: appliedOff,
		}
		return out, nil
	case "count":
		limited, appliedLimit := GrepApplyHeadLimit(lines, in.HeadLimit, off)
		totalMatches := 0
		fileCount := 0
		finalLines := make([]string, 0, len(limited))
		for _, line := range limited {
			rel := relativizeCountLine(line, cwd)
			finalLines = append(finalLines, rel)
			if i := strings.LastIndex(rel, ":"); i > 0 {
				n, err := parseTrailingInt(rel[i+1:])
				if err == nil {
					totalMatches += n
					fileCount++
				}
			}
		}
		nm := totalMatches
		return grepStructuredOutput{
			Mode:          "count",
			NumFiles:      fileCount,
			Filenames:     []string{},
			Content:       strings.Join(finalLines, "\n"),
			NumMatches:    &nm,
			AppliedLimit:  appliedLimit,
			AppliedOffset: appliedOff,
		}, nil
	default: // files_with_matches
		// Sort by mtime desc, name tiebreaker (TS non-test). Deterministic: use name for parity in tests.
		type pair struct {
			path  string
			mtime int64
		}
		ps := make([]pair, 0, len(lines))
		for _, line := range lines {
			p := strings.TrimSpace(line)
			if p == "" {
				continue
			}
			var mt int64
			if st, err := os.Stat(p); err == nil {
				mt = st.ModTime().UnixNano()
			}
			ps = append(ps, pair{path: p, mtime: mt})
		}
		sort.SliceStable(ps, func(i, j int) bool {
			if ps[i].mtime != ps[j].mtime {
				return ps[i].mtime > ps[j].mtime
			}
			return ps[i].path < ps[j].path
		})
		absPaths := make([]string, len(ps))
		for i := range ps {
			absPaths[i] = ps[i].path
		}
		limited, appliedLimit := GrepApplyHeadLimit(absPaths, in.HeadLimit, off)
		rel := make([]string, len(limited))
		for i := range limited {
			rel[i] = ToRelativePathFromCWD(cwd, limited[i])
		}
		return grepStructuredOutput{
			Mode:          "files_with_matches",
			NumFiles:      len(rel),
			Filenames:     rel,
			AppliedLimit:  appliedLimit,
			AppliedOffset: appliedOff,
		}, nil
	}
}

func parseTrailingInt(s string) (int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty")
	}
	var n int
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("non-digit")
		}
		n = n*10 + int(c-'0')
	}
	return n, nil
}

func relativizeRgLinePath(line, cwd string) string {
	idx := strings.Index(line, ":")
	if idx <= 0 {
		return line
	}
	filePath := line[:idx]
	rest := line[idx:]
	return ToRelativePathFromCWD(cwd, filePath) + rest
}

func relativizeCountLine(line, cwd string) string {
	idx := strings.LastIndex(line, ":")
	if idx <= 0 {
		return line
	}
	filePath := line[:idx]
	suffix := line[idx:]
	return ToRelativePathFromCWD(cwd, filePath) + suffix
}

func goGrepStructured(ctx context.Context, base, pattern string, in grepToolInput, mode, cwd string) (grepStructuredOutput, bool, error) {
	var prefix string
	if in.CaseFold {
		prefix += "(?i)"
	}
	if in.Multiline {
		prefix += "(?s)"
	}
	re, err := regexp.Compile(prefix + pattern)
	if err != nil {
		return grepStructuredOutput{}, true, fmt.Errorf("invalid pattern: %w", err)
	}
	globRe := (*regexp.Regexp)(nil)
	if strings.TrimSpace(in.Glob) != "" {
		globRe, err = globToRegex(in.Glob)
		if err != nil {
			return grepStructuredOutput{}, true, err
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
		return grepStructuredOutput{}, true, errWalk
	}

	off := offPtr(in)
	var appliedOff *int
	if off > 0 {
		appliedOff = &off
	}

	switch mode {
	case "files_with_matches":
		absPaths := make([]string, 0, len(hits))
		for _, h := range hits {
			absPaths = append(absPaths, h.path)
		}
		sort.SliceStable(absPaths, func(i, j int) bool {
			si, _ := os.Stat(absPaths[i])
			sj, _ := os.Stat(absPaths[j])
			var mi, mj int64
			if si != nil {
				mi = si.ModTime().UnixNano()
			}
			if sj != nil {
				mj = sj.ModTime().UnixNano()
			}
			if mi != mj {
				return mi > mj
			}
			return absPaths[i] < absPaths[j]
		})
		limited, appliedLimit := GrepApplyHeadLimit(absPaths, in.HeadLimit, off)
		rel := make([]string, len(limited))
		for i := range limited {
			rel[i] = ToRelativePathFromCWD(cwd, limited[i])
		}
		return grepStructuredOutput{
			Mode:          "files_with_matches",
			NumFiles:      len(rel),
			Filenames:     rel,
			AppliedLimit:  appliedLimit,
			AppliedOffset: appliedOff,
		}, false, nil
	case "count":
		var lines []string
		for _, h := range hits {
			lines = append(lines, h.path+":"+itoaLocal(h.count))
		}
		sort.Strings(lines)
		limited, appliedLimit := GrepApplyHeadLimit(lines, in.HeadLimit, off)
		totalMatches := 0
		fileCount := 0
		finalLines := make([]string, 0, len(limited))
		for _, line := range limited {
			rel := relativizeCountLine(line, cwd)
			finalLines = append(finalLines, rel)
			if i := strings.LastIndex(rel, ":"); i > 0 {
				n, err := parseTrailingInt(rel[i+1:])
				if err == nil {
					totalMatches += n
					fileCount++
				}
			}
		}
		nm := totalMatches
		return grepStructuredOutput{
			Mode:          "count",
			NumFiles:      fileCount,
			Filenames:     []string{},
			Content:       strings.Join(finalLines, "\n"),
			NumMatches:    &nm,
			AppliedLimit:  appliedLimit,
			AppliedOffset: appliedOff,
		}, false, nil
	default: // content
		var rawLines []string
		for _, h := range hits {
			for _, ln := range h.lines {
				rawLines = append(rawLines, h.path+":"+ln)
			}
		}
		sort.Strings(rawLines)
		limited, appliedLimit := GrepApplyHeadLimit(rawLines, in.HeadLimit, off)
		finalLines := make([]string, 0, len(limited))
		for _, line := range limited {
			finalLines = append(finalLines, relativizeRgLinePath(line, cwd))
		}
		content := strings.Join(finalLines, "\n")
		nl := len(finalLines)
		return grepStructuredOutput{
			Mode:          "content",
			NumFiles:      0,
			Filenames:     []string{},
			Content:       content,
			NumLines:      &nl,
			AppliedLimit:  appliedLimit,
			AppliedOffset: appliedOff,
		}, false, nil
	}
}

func itoaLocal(n int) string {
	return fmt.Sprintf("%d", n)
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
