package claudemd

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"goc/hookexec"
)

// Session memo + InstructionsLoaded gating mirror src/utils/claudemd.ts getMemoryFiles memo,
// clearMemoryFileCaches, and resetGetMemoryFilesCache.
var (
	memFileMu    sync.Mutex
	memFileCache map[string][]MemoryFileInfo
	memInstrMu   sync.Mutex
	// next after consume() unless ResetMemoryFilesCache
	nextInstrReason       = "session_start"
	shouldFireInstrOnMiss = true
)

func bool01(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

// envFingerprint captures env and opts fields that change which files load or how they are formatted
// (mirrors why TS getMemoryFiles only keys on forceBoolean — Go keys more precisely).
func memoryFileRelevantEnvFingerprint() string {
	keys := []string{
		"CLAUDE_CODE_SETTING_SOURCES",
		"CLAUDE_CODE_DISABLE_USER_MEMORY",
		"CLAUDE_CODE_DISABLE_PROJECT_MEMORY",
		"CLAUDE_CODE_DISABLE_LOCAL_MEMORY",
		"CLAUDE_CODE_ADDITIONAL_DIRECTORIES_CLAUDE_MD",
		"CLAUDE_CODE_CLAUDE_MD_EXTERNAL_INCLUDES_APPROVED",
		"CLAUDE_CODE_TENGU_MOTH_COPSE",
		"CLAUDE_CODE_TENGU_PAPER_HALYARD",
		"CLAUDE_CODE_DISABLE_AUTO_MEMORY",
		"CLAUDE_CODE_SIMPLE",
		"CLAUDE_CODE_REMOTE",
		"CLAUDE_CODE_REMOTE_MEMORY_DIR",
		"CLAUDE_CODE_AUTO_MEMORY_ENABLED",
		"CLAUDE_CODE_AUTO_MEMORY_DIRECTORY",
		"CLAUDE_COWORK_MEMORY_PATH_OVERRIDE",
		"USER_TYPE",
		"CLAUDE_CODE_MANAGED_SETTINGS_PATH",
		"CLAUDE_CONFIG_DIR",
		"FEATURE_TEAMMEM",
		"CLAUDE_CODE_TEAM_MEMORY_ENABLED",
		"CLAUDE_CODE_FLAG_SETTINGS_PATH",
		"CLAUDE_CODE_USE_COWORK_PLUGINS",
		"HOME",
	}
	var b strings.Builder
	for _, k := range keys {
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(os.Getenv(k))
		b.WriteByte(0x1e)
	}
	return b.String()
}

func memoryFileCacheKey(opts LoadOptions) string {
	original := strings.TrimSpace(opts.OriginalCwd)
	if original == "" {
		original, _ = os.Getwd()
	}
	absOrig, err := filepath.Abs(original)
	if err != nil {
		absOrig = original
	} else {
		absOrig = filepath.Clean(absOrig)
	}

	extras := make([]string, 0, len(opts.AdditionalWorkingDirs))
	for _, d := range opts.AdditionalWorkingDirs {
		s := strings.TrimSpace(d)
		if s == "" {
			continue
		}
		if a, e := filepath.Abs(s); e == nil {
			extras = append(extras, filepath.Clean(a))
		} else {
			extras = append(extras, filepath.Clean(s))
		}
	}
	sort.Strings(extras)
	inc1 := opts.ForceIncludeExternal || opts.HasClaudeMdExternalIncludesApproved
	if truthy(os.Getenv("CLAUDE_CODE_CLAUDE_MD_EXTERNAL_INCLUDES_APPROVED")) {
		inc1 = true
	}
	addDir := truthy(os.Getenv("CLAUDE_CODE_ADDITIONAL_DIRECTORIES_CLAUDE_MD")) || len(extras) > 0
	var excl string
	if opts.ClaudeMdExcludesOverride != nil {
		join := make([]string, len(*opts.ClaudeMdExcludesOverride))
		copy(join, *opts.ClaudeMdExcludesOverride)
		sort.Strings(join)
		excl = strings.Join(join, "\n")
	} else {
		excl = "<nil>"
	}
	return strings.Join([]string{
		absOrig,
		strings.Join(extras, "\x1e"),
		bool01(inc1),
		bool01(opts.ForceIncludeExternal),
		bool01(opts.HasClaudeMdExternalIncludesApproved),
		bool01(addDir),
		excl,
		memoryFileRelevantEnvFingerprint(),
	}, "\x1f")
}

func cloneMemoryFiles(in []MemoryFileInfo) []MemoryFileInfo {
	if len(in) == 0 {
		return nil
	}
	out := make([]MemoryFileInfo, len(in))
	for i := range in {
		out[i] = in[i]
		if in[i].Globs != nil {
			out[i].Globs = append([]string(nil), in[i].Globs...)
		}
	}
	return out
}

// ClearMemoryFileCaches invalidates the getMemoryFiles-equivalent cache without re-arming
// InstructionsLoaded for the next miss (TS clearMemoryFileCaches).
func ClearMemoryFileCaches() {
	memFileMu.Lock()
	memFileCache = nil
	memFileMu.Unlock()
}

// ResetMemoryFilesCache clears the memo and sets the next InstructionsLoaded load_reason for
// the next cache miss (TS resetGetMemoryFilesCache). Empty reason defaults to session_start.
func ResetMemoryFilesCache(reason string) {
	r := strings.TrimSpace(reason)
	if r == "" {
		r = "session_start"
	}
	memInstrMu.Lock()
	nextInstrReason = r
	shouldFireInstrOnMiss = true
	memInstrMu.Unlock()
	ClearMemoryFileCaches()
}

func consumeNextInstructionsLoadReason() (string, bool) {
	memInstrMu.Lock()
	defer memInstrMu.Unlock()
	if !shouldFireInstrOnMiss {
		return "", false
	}
	shouldFireInstrOnMiss = false
	r := nextInstrReason
	if r == "" {
		r = "session_start"
	}
	nextInstrReason = "session_start"
	return r, true
}

func runInstructionsLoadedHooksForLoadResult(opts *LoadOptions, absOrig string, result []MemoryFileInfo) {
	if opts == nil || opts.ForceIncludeExternal {
		return
	}
	reason, ok := consumeNextInstructionsLoadReason()
	if !ok {
		return
	}
	tab, err := hookexec.MergedHooksForCwd(absOrig)
	if err != nil || !hookexec.HasInstructionsLoaded(tab) {
		return
	}
	base := hookexec.BaseHookInput{
		SessionID:      "local",
		TranscriptPath: "",
		Cwd:            absOrig,
	}
	for _, file := range result {
		if !isInstructionsMemoryType(file.Type) {
			continue
		}
		loadReason := reason
		if file.Parent != "" {
			loadReason = "include"
		}
		hookexec.FireInstructionsLoaded(context.Background(), tab, absOrig, base, hookexec.InstructionsLoadedFields{
			FilePath:       file.Path,
			MemoryType:     string(file.Type),
			LoadReason:     loadReason,
			Globs:          file.Globs,
			ParentFilePath: file.Parent,
		}, hookexec.DefaultHookTimeoutMs)
	}
}
