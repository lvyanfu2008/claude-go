package localtools

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"goc/claudemd"
)

// Mirrors src/tools/FileReadTool/prompt.ts FILE_UNCHANGED_STUB.
const fileUnchangedStub = "File unchanged since last read. The content from the earlier Read tool_result in this conversation is still current — refer to that instead of re-reading."

// Mirrors src/utils/file.ts MAX_OUTPUT_SIZE (default max read file bytes).
const defaultMaxReadFileBytes = 262144 // 0.25 * 1024 * 1024

const defaultMaxReadOutputTokens = 25000

var imageExtensions = map[string]struct{}{
	"png": {}, "jpg": {}, "jpeg": {}, "gif": {}, "webp": {},
}

// ReadTextOutput mirrors FileReadTool text branch (tool JSON / toolUseResult).
type ReadTextOutput struct {
	Type string `json:"type"`
	File struct {
		FilePath   string `json:"filePath"`
		Content    string `json:"content"`
		NumLines   int    `json:"numLines"`
		StartLine  int    `json:"startLine"`
		TotalLines int    `json:"totalLines"`
	} `json:"file"`
}

type readNotebookOutput struct {
	Type string `json:"type"`
	File struct {
		FilePath string          `json:"filePath"`
		Cells    json.RawMessage `json:"cells"`
	} `json:"file"`
}

type readImageOutput struct {
	Type string `json:"type"`
	File struct {
		Base64       string `json:"base64"`
		Type         string `json:"type"`
		OriginalSize int    `json:"originalSize"`
	} `json:"file"`
}

type readFileUnchangedOutput struct {
	Type string `json:"type"`
	File struct {
		FilePath string `json:"filePath"`
	} `json:"file"`
}

// FileReadingLimits optional overrides (mirrors types.FileReadingLimits).
type FileReadingLimits struct {
	MaxTokens    *int
	MaxSizeBytes *int
}

func defaultReadLimits() (maxBytes, maxTokens int) {
	maxBytes = defaultMaxReadFileBytes
	maxTokens = defaultMaxReadOutputTokens
	if v := strings.TrimSpace(os.Getenv("CLAUDE_CODE_FILE_READ_MAX_OUTPUT_TOKENS")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			maxTokens = n
		}
	}
	return maxBytes, maxTokens
}

func roughTokenEstimate(s string) int {
	// TS uses API count; bound with bytes/4 (common heuristic for English/code).
	return (len(s) + 3) / 4
}

func validateReadOutputTokens(content string, maxTokens int) error {
	if roughTokenEstimate(content) <= maxTokens/4 {
		return nil
	}
	if roughTokenEstimate(content) > maxTokens {
		return fmt.Errorf("file content exceeds maximum token budget (~%d tokens estimated; limit %d). Read a smaller range with offset and limit", roughTokenEstimate(content), maxTokens)
	}
	return nil
}

// readFileInRangeFast mirrors src/utils/readFileInRange.ts readFileInRangeFast (offset is 0-based line index).
func readFileInRangeFast(absPath string, offset0 int, maxLines *int, maxFileBytes int) (content string, lineCount, totalLines int, totalBytes int64, mtimeMs int64, err error) {
	st, err := os.Stat(absPath)
	if err != nil {
		return "", 0, 0, 0, 0, err
	}
	if st.IsDir() {
		return "", 0, 0, 0, 0, fmt.Errorf("EISDIR: illegal operation on a directory, read '%s'", absPath)
	}
	if st.Size() > int64(maxFileBytes) {
		return "", 0, 0, 0, 0, fmt.Errorf("File content (%d bytes) exceeds maximum allowed size (%d bytes). Use offset and limit parameters to read specific portions of the file, or search for specific content instead of reading the whole file", st.Size(), maxFileBytes)
	}
	raw, err := os.ReadFile(absPath)
	if err != nil {
		return "", 0, 0, 0, 0, err
	}
	text := string(raw)
	totalBytes = int64(len(raw))
	mtimeMs = st.ModTime().UnixMilli()

	// Strip UTF-8 BOM (EF BB BF)
	if strings.HasPrefix(text, "\ufeff") {
		_, sz := utf8.DecodeRuneInString(text)
		text = text[sz:]
	}

	endLine := int(^uint(0) >> 1)
	if maxLines != nil {
		endLine = offset0 + *maxLines
	}

	var selected []string
	lineIndex := 0
	startPos := 0
	for {
		newlinePos := strings.IndexByte(text[startPos:], '\n')
		if newlinePos == -1 {
			// final fragment
			if lineIndex >= offset0 && lineIndex < endLine {
				line := text[startPos:]
				if strings.HasSuffix(line, "\r") {
					line = line[:len(line)-1]
				}
				selected = append(selected, line)
			}
			lineIndex++
			break
		}
		absNew := startPos + newlinePos
		if lineIndex >= offset0 && lineIndex < endLine {
			line := text[startPos:absNew]
			if strings.HasSuffix(line, "\r") {
				line = line[:len(line)-1]
			}
			selected = append(selected, line)
		}
		lineIndex++
		startPos = absNew + 1
	}

	totalLines = lineIndex
	content = strings.Join(selected, "\n")
	lineCount = len(selected)
	return content, lineCount, totalLines, totalBytes, mtimeMs, nil
}

// ReadFromJSON runs the Read tool with TS-shaped JSON output (mirrors FileReadTool.call + data for text/image/notebook/unchanged).
// Gaps vs TS: see [FileReadFeatureStatus] in filetool_parity.go; PDF paths use [ErrReadPDFPagesNotImplementedInGo] / [ErrReadPDFFullNotImplementedInGo].
func ReadFromJSON(raw []byte, roots []string, state *ReadFileState, limits *FileReadingLimits) (string, bool, error) {
	var in struct {
		FilePath string `json:"file_path"`
		Offset   int    `json:"offset"`
		Limit    *int   `json:"limit"`
		Pages    string `json:"pages"`
	}
	if err := json.Unmarshal(raw, &in); err != nil {
		return "", true, err
	}
	if err := validateReadPath(in.FilePath, roots); err != nil {
		return "", true, err
	}
	if err := ValidateReadPagesParameter(in.Pages); err != nil {
		return "", true, err
	}
	abs, err := ResolveUnderRoots(in.FilePath, roots)
	if err != nil {
		return "", true, err
	}
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(abs)), ".")

	maxBytes, maxTok := defaultReadLimits()
	if limits != nil {
		if limits.MaxSizeBytes != nil && *limits.MaxSizeBytes > 0 {
			maxBytes = *limits.MaxSizeBytes
		}
		if limits.MaxTokens != nil && *limits.MaxTokens > 0 {
			maxTok = *limits.MaxTokens
		}
	}

	// --- Dedup (mirrors FileReadTool.call file_unchanged) ---
	if state != nil {
		if prev := state.Get(abs); prev != nil && !prev.IsPartialView && prev.Offset != nil {
			off := in.Offset
			if off <= 0 {
				off = 1
			}
			if *prev.Offset == off && ptrEq(prev.Limit, in.Limit) {
				st, err := os.Stat(abs)
				if err == nil && st.ModTime().UnixMilli() == prev.Timestamp {
					out := readFileUnchangedOutput{Type: "file_unchanged"}
					out.File.FilePath = in.FilePath
					b, _ := json.Marshal(out)
					return string(b), false, nil
				}
			}
		}
	}

	// --- Notebook ---
	if ext == "ipynb" {
		rawF, err := os.ReadFile(abs)
		if err != nil {
			return "", true, err
		}
		if len(rawF) > maxBytes {
			return "", true, fmt.Errorf("Notebook content exceeds maximum allowed size (%d bytes)", maxBytes)
		}
		var root map[string]json.RawMessage
		if err := json.Unmarshal(rawF, &root); err != nil {
			return "", true, err
		}
		cells := root["cells"]
		if cells == nil {
			cells = json.RawMessage("[]")
		}
		var out readNotebookOutput
		out.Type = "notebook"
		out.File.FilePath = in.FilePath
		out.File.Cells = cells
		if err := validateReadOutputTokens(string(rawF), maxTok); err != nil {
			return "", true, err
		}
		st, _ := os.Stat(abs)
		if state != nil {
			state.Set(abs, &ReadFileEntry{
				Content:   string(rawF),
				Timestamp: st.ModTime().UnixMilli(),
				Offset:    ptrInt(1),
				Limit:     nil,
			})
		}
		b, _ := json.Marshal(out)
		return string(b), false, nil
	}

	// --- Image ---
	if _, ok := imageExtensions[ext]; ok {
		buf, err := os.ReadFile(abs)
		if err != nil {
			return "", true, err
		}
		mt := "image/" + ext
		if ext == "jpg" {
			mt = "image/jpeg"
		}
		var out readImageOutput
		out.Type = "image"
		out.File.Base64 = base64.StdEncoding.EncodeToString(buf)
		out.File.Type = mt
		out.File.OriginalSize = len(buf)
		// TS uses image token budget; skip rough ASCII token estimate on base64.
		st, _ := os.Stat(abs)
		if state != nil {
			state.Set(abs, &ReadFileEntry{
				Content:   out.File.Base64,
				Timestamp: st.ModTime().UnixMilli(),
				Offset:    ptrInt(1),
				Limit:     nil,
			})
		}
		b, _ := json.Marshal(out)
		return string(b), false, nil
	}

	// --- PDF: reserved — see read_pdf_stub.go (poppler extract + document blocks + TS parity).
	if ext == "pdf" {
		if strings.TrimSpace(in.Pages) != "" {
			return "", true, fmt.Errorf("%w", ErrReadPDFPagesNotImplementedInGo)
		}
		return "", true, fmt.Errorf("%w", ErrReadPDFFullNotImplementedInGo)
	}

	// --- Binary guard (subset of TS hasBinaryExtension) ---
	if isLikelyBinaryExt(ext) {
		return "", true, fmt.Errorf("This tool cannot read binary files. The file appears to be a binary .%s file. Please use appropriate tools for binary file analysis.", ext)
	}

	// --- Text ---
	off := in.Offset
	if off <= 0 {
		off = 1
	}
	offset0 := off - 1
	var content string
	var lineCount, totalLines int
	var mtimeMs int64
	var rerr error
	content, lineCount, totalLines, _, mtimeMs, rerr = readFileInRangeFast(abs, offset0, in.Limit, maxBytes)
	if rerr != nil {
		if os.IsNotExist(rerr) {
			rerr = enrichNotExistError(abs, rerr)
		}
		return "", true, rerr
	}
	if err := validateReadOutputTokens(content, maxTok); err != nil {
		return "", true, err
	}

	var out ReadTextOutput
	out.Type = "text"
	out.File.FilePath = in.FilePath
	out.File.Content = content
	out.File.NumLines = lineCount
	out.File.StartLine = off
	out.File.TotalLines = totalLines

	if state != nil {
		state.Set(abs, &ReadFileEntry{
			Content:   content,
			Timestamp: mtimeMs,
			Offset:    ptrInt(off),
			Limit:     in.Limit,
		})
	}
	b, err := json.Marshal(out)
	if err != nil {
		return "", true, err
	}
	return string(b), false, nil
}

func ptrInt(v int) *int { return &v }

func ptrEq(a, b *int) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func isLikelyBinaryExt(ext string) bool {
	switch ext {
	case "exe", "dll", "so", "dylib", "bin", "o", "a", "zip", "tar", "gz", "bz2", "xz", "7z", "rar",
		"class", "jar", "wasm", "sqlite", "db", "ico", "woff", "woff2", "ttf", "eot":
		return true
	default:
		return false
	}
}

func validateReadPath(filePath string, roots []string) error {
	trimmed := strings.TrimSpace(filePath)
	// Catch obvious device paths before workspace-root resolution.
	if strings.HasPrefix(filepath.Clean(trimmed), "/dev/") {
		return fmt.Errorf("cannot read from device path: %s", filepath.Clean(trimmed))
	}
	abs, err := ResolveUnderRoots(filePath, roots)
	if err != nil {
		return nil // Keep existing root/path errors from ResolveUnderRoots in the main path.
	}
	if isDevicePath(abs) {
		return fmt.Errorf("cannot read from device path: %s", abs)
	}
	return nil
}

func isDevicePath(abs string) bool {
	clean := filepath.Clean(strings.TrimSpace(abs))
	if clean == "" {
		return false
	}
	// Mirror TS safety intent: block direct reads from OS device files.
	if strings.HasPrefix(clean, "/dev/") {
		return true
	}
	base := strings.ToUpper(filepath.Base(clean))
	switch base {
	case "CON", "PRN", "AUX", "NUL", "COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9", "LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9":
		return true
	default:
		return false
	}
}

func enrichNotExistError(abs string, baseErr error) error {
	dir := filepath.Dir(abs)
	target := filepath.Base(abs)
	if target == "" || target == "." || target == string(os.PathSeparator) {
		return baseErr
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return baseErr
	}
	type cand struct {
		name  string
		score int
	}
	var cands []cand
	targetLower := strings.ToLower(target)
	for _, e := range entries {
		name := e.Name()
		score := similarityScore(targetLower, strings.ToLower(name))
		if score > 0 {
			cands = append(cands, cand{name: name, score: score})
		}
	}
	if len(cands) == 0 {
		return baseErr
	}
	sort.Slice(cands, func(i, j int) bool {
		if cands[i].score == cands[j].score {
			return cands[i].name < cands[j].name
		}
		return cands[i].score > cands[j].score
	})
	suggest := cands[0].name
	return fmt.Errorf("%w. Did you mean %q?", baseErr, filepath.Join(dir, suggest))
}

func similarityScore(a, b string) int {
	if a == "" || b == "" {
		return 0
	}
	if a == b {
		return 1000
	}
	score := 0
	if strings.HasPrefix(b, a) || strings.HasPrefix(a, b) {
		score += 120
	}
	if strings.Contains(b, a) || strings.Contains(a, b) {
		score += 80
	}
	common := 0
	max := len(a)
	if len(b) < max {
		max = len(b)
	}
	for i := 0; i < max; i++ {
		if a[i] != b[i] {
			break
		}
		common++
	}
	score += common * 8
	if score < 40 {
		return 0
	}
	return score
}

// Mirrors src/tools/FileReadTool/FileReadTool.ts CYBER_RISK_MITIGATION_REMINDER.
const cyberRiskMitigationReminder = "\n\n<system-reminder>\nWhenever you read a file, you should consider whether it would be considered malware. You CAN and SHOULD provide analysis of malware, what it is doing. But you MUST refuse to improve or augment the code. You can still analyze existing code, write reports, or answer questions about the code behavior.\n</system-reminder>\n"

// ReadToolResultMapOpts carries presentation-only inputs for MapReadToolResultToAssistantText
// (mirrors TS FileReadTool mapToolResultToToolResultBlockParam side channels).
type ReadToolResultMapOpts struct {
	MainLoopModel     string
	MemoryFileMtimeMs *int64
	// Now overrides the clock for memory-age tests; zero means time.Now().
	Now time.Time
}

// ReadToolResultMapOptsForToolInput builds opts from Read tool input JSON (file_path) and cwd used for auto-memory root.
func ReadToolResultMapOptsForToolInput(input []byte, roots []string, memCwd, mainLoopModel string) *ReadToolResultMapOpts {
	opts := &ReadToolResultMapOpts{MainLoopModel: strings.TrimSpace(mainLoopModel)}
	var in struct {
		FilePath string `json:"file_path"`
	}
	_ = json.Unmarshal(input, &in)
	fp := strings.TrimSpace(in.FilePath)
	if fp == "" {
		return opts
	}
	abs, err := ResolveUnderRoots(fp, roots)
	if err != nil {
		return opts
	}
	mc := strings.TrimSpace(memCwd)
	if claudemd.IsAutoMemoryEnabled() && claudemd.IsAutoMemPath(abs, mc) {
		if st, err := os.Stat(abs); err == nil {
			ms := st.ModTime().UnixMilli()
			opts.MemoryFileMtimeMs = &ms
		}
	}
	return opts
}

// MapReadToolResultToAssistantText mirrors FileReadTool.mapToolResultToToolResultBlockParam (text + file_unchanged).
// Pass opts nil for defaults (mitigation on except opus-4-6 canonical; compact line prefix per TS killswitch env).
func MapReadToolResultToAssistantText(dataJSON string, opts *ReadToolResultMapOpts) (string, error) {
	var probe struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal([]byte(dataJSON), &probe); err != nil {
		return dataJSON, nil
	}
	switch probe.Type {
	case "text":
		var p ReadTextOutput
		if err := json.Unmarshal([]byte(dataJSON), &p); err != nil {
			return "", err
		}
		return formatReadTextForModel(p, opts), nil
	case "file_unchanged":
		return fileUnchangedStub, nil
	default:
		return dataJSON, nil
	}
}

func optsNow(o *ReadToolResultMapOpts) time.Time {
	if o != nil && !o.Now.IsZero() {
		return o.Now
	}
	return time.Now()
}

// canonicalMainLoopModel mirrors src/utils/model/model.ts firstPartyNameToCanonical (subset sufficient for tool gating).
func canonicalMainLoopModel(full string) string {
	n := strings.ToLower(strings.TrimSpace(full))
	if n == "" {
		return ""
	}
	if strings.Contains(n, "claude-opus-4-6") {
		return "claude-opus-4-6"
	}
	if strings.Contains(n, "claude-opus-4-5") {
		return "claude-opus-4-5"
	}
	if strings.Contains(n, "claude-opus-4-1") {
		return "claude-opus-4-1"
	}
	if strings.Contains(n, "claude-opus-4") {
		return "claude-opus-4"
	}
	if strings.Contains(n, "claude-sonnet-4-6") {
		return "claude-sonnet-4-6"
	}
	if strings.Contains(n, "claude-sonnet-4-5") {
		return "claude-sonnet-4-5"
	}
	if strings.Contains(n, "claude-sonnet-4") {
		return "claude-sonnet-4"
	}
	if strings.Contains(n, "claude-haiku-4-5") {
		return "claude-haiku-4-5"
	}
	if strings.Contains(n, "claude-3-7-sonnet") {
		return "claude-3-7-sonnet"
	}
	if strings.Contains(n, "claude-3-5-sonnet") {
		return "claude-3-5-sonnet"
	}
	if strings.Contains(n, "claude-3-5-haiku") {
		return "claude-3-5-haiku"
	}
	if strings.Contains(n, "claude-3-opus") {
		return "claude-3-opus"
	}
	if strings.Contains(n, "claude-3-sonnet") {
		return "claude-3-sonnet"
	}
	if strings.Contains(n, "claude-3-haiku") {
		return "claude-3-haiku"
	}
	return n
}

func shouldIncludeFileReadMitigation(opts *ReadToolResultMapOpts) bool {
	model := ""
	if opts != nil {
		model = opts.MainLoopModel
	}
	return canonicalMainLoopModel(model) != "claude-opus-4-6"
}

// isCompactLinePrefixEnabled mirrors src/utils/file.ts isCompactLinePrefixEnabled (feature flag name in TS).
// Env TENGU_COMPACT_LINE_PREFIX_KILLSWITCH: truthy disables compact tab prefixes (uses padded → instead).
func isCompactLinePrefixEnabled() bool {
	return !envTruthy("TENGU_COMPACT_LINE_PREFIX_KILLSWITCH")
}

func memoryAgeDays(mtimeMs int64, now time.Time) int {
	d := int(now.Sub(time.UnixMilli(mtimeMs)).Hours() / 24)
	if d < 0 {
		return 0
	}
	return d
}

func memoryFreshnessText(mtimeMs int64, now time.Time) string {
	d := memoryAgeDays(mtimeMs, now)
	if d <= 1 {
		return ""
	}
	return fmt.Sprintf("This memory is %d days old. Memories are point-in-time observations, not live state — claims about code behavior or file:line citations may be outdated. Verify against current code before asserting as fact.", d)
}

func memoryFreshnessNote(mtimeMs int64, now time.Time) string {
	text := memoryFreshnessText(mtimeMs, now)
	if text == "" {
		return ""
	}
	return "<system-reminder>" + text + "</system-reminder>\n"
}

func formatReadTextForModel(p ReadTextOutput, opts *ReadToolResultMapOpts) string {
	if p.File.Content == "" {
		if p.File.TotalLines == 0 {
			return "<system-reminder>Warning: the file exists but the contents are empty.</system-reminder>"
		}
		return fmt.Sprintf("<system-reminder>Warning: the file exists but is shorter than the provided offset (%d). The file has %d lines.</system-reminder>", p.File.StartLine, p.File.TotalLines)
	}
	var b strings.Builder
	if opts != nil && opts.MemoryFileMtimeMs != nil {
		if note := memoryFreshnessNote(*opts.MemoryFileMtimeMs, optsNow(opts)); note != "" {
			b.WriteString(note)
		}
	}
	b.WriteString(formatFileLinesNumbered(p.File.Content, p.File.StartLine))
	if shouldIncludeFileReadMitigation(opts) {
		b.WriteString(cyberRiskMitigationReminder)
	}
	return b.String()
}

// formatFileLinesNumbered mirrors src/utils/file.ts addLineNumbers (compact vs padded per killswitch).
func formatFileLinesNumbered(content string, startLine int) string {
	if content == "" {
		return ""
	}
	text := strings.ReplaceAll(content, "\r\n", "\n")
	lines := strings.Split(text, "\n")
	if isCompactLinePrefixEnabled() {
		var b strings.Builder
		for i, line := range lines {
			if i > 0 {
				b.WriteByte('\n')
			}
			fmt.Fprintf(&b, "%d\t%s", startLine+i, line)
		}
		return b.String()
	}
	var b strings.Builder
	for i, line := range lines {
		if i > 0 {
			b.WriteByte('\n')
		}
		n := startLine + i
		ns := strconv.Itoa(n)
		if len(ns) >= 6 {
			fmt.Fprintf(&b, "%s→%s", ns, line)
		} else {
			fmt.Fprintf(&b, "%6s→%s", ns, line)
		}
	}
	return b.String()
}
