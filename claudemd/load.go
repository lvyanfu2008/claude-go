package claudemd

import (
	"os"
	"path/filepath"
	"strings"
)

// LoadOptions drives getMemoryFiles-equivalent loading.
type LoadOptions struct {
	OriginalCwd string
	// AdditionalWorkingDirs used when CLAUDE_CODE_ADDITIONAL_DIRECTORIES_CLAUDE_MD is set.
	AdditionalWorkingDirs               []string
	ForceIncludeExternal                bool
	HasClaudeMdExternalIncludesApproved bool
	// ClaudeMdExcludesOverride when non-nil replaces MergedClaudeMdExcludes (tests).
	ClaudeMdExcludesOverride *[]string
}

// LoadMemoryFiles mirrors getMemoryFiles with session-level memo and InstructionsLoaded gating; see
// [ClearMemoryFileCaches], [ResetMemoryFilesCache], and claudemd.ts clearMemoryFileCaches / resetGetMemoryFilesCache.
func LoadMemoryFiles(opts LoadOptions) []MemoryFileInfo {
	key := memoryFileCacheKey(opts)
	memFileMu.Lock()
	if memFileCache != nil {
		if v, ok := memFileCache[key]; ok {
			out := cloneMemoryFiles(v)
			memFileMu.Unlock()
			return out
		}
	}
	memFileMu.Unlock()
	out := loadMemoryFilesUncached(opts)
	memFileMu.Lock()
	if memFileCache == nil {
		memFileCache = make(map[string][]MemoryFileInfo)
	}
	memFileCache[key] = cloneMemoryFiles(out)
	memFileMu.Unlock()
	return cloneMemoryFiles(out)
}

// LoadMemoryFilesEnhanced is the same as [LoadMemoryFiles] (TS parity: one memoized entry point).
func LoadMemoryFilesEnhanced(opts LoadOptions) []MemoryFileInfo {
	return LoadMemoryFiles(opts)
}

func loadMemoryFilesUncached(opts LoadOptions) []MemoryFileInfo {
	original := strings.TrimSpace(opts.OriginalCwd)
	if original == "" {
		original, _ = os.Getwd()
	}
	absOrig, err := filepath.Abs(original)
	if err != nil {
		absOrig = original
	}

	includeExternal := opts.ForceIncludeExternal || opts.HasClaudeMdExternalIncludesApproved
	if truthy(os.Getenv("CLAUDE_CODE_CLAUDE_MD_EXTERNAL_INCLUDES_APPROVED")) {
		includeExternal = true
	}

	var excl []string
	if opts.ClaudeMdExcludesOverride != nil {
		excl = *opts.ClaudeMdExcludesOverride
	} else {
		excl = MergedClaudeMdExcludes(absOrig)
	}

	// 创建记忆层次结构管理器
	mh := NewMemoryHierarchy(absOrig, excl)

	// 加载所有记忆文件
	result := mh.LoadAllMemoryFiles(absOrig, includeExternal)

	// 处理额外的目录（--add-dir）
	if truthy(os.Getenv("CLAUDE_CODE_ADDITIONAL_DIRECTORIES_CLAUDE_MD")) || len(opts.AdditionalWorkingDirs) > 0 {
		processed := make(map[string]struct{})
		for _, d := range opts.AdditionalWorkingDirs {
			ad, err := filepath.Abs(strings.TrimSpace(d))
			if err != nil || ad == "" {
				continue
			}

			// 处理额外目录中的 CLAUDE.md 文件
			claudeMdPath := filepath.Join(ad, "CLAUDE.md")
			if files := mh.processMemoryFileWithIncludes(claudeMdPath, MemoryProject, processed, includeExternal, absOrig); len(files) > 0 {
				result = append(result, files...)
			}

			// 处理额外目录中的 .claude/CLAUDE.md 文件
			dotClaudePath := filepath.Join(ad, ".claude", "CLAUDE.md")
			if files := mh.processMemoryFileWithIncludes(dotClaudePath, MemoryProject, processed, includeExternal, absOrig); len(files) > 0 {
				result = append(result, files...)
			}

			// 处理额外目录中的规则文件
			rulesDir := filepath.Join(ad, ".claude", "rules")
			if rulesFiles := ProcessMdRules(rulesDir, MemoryProject, processed, includeExternal, absOrig, false, nil, mh.excludeChecker); len(rulesFiles) > 0 {
				result = append(result, rulesFiles...)
			}
		}
	}

	// TS getMemoryFiles: one-shot eager InstructionsLoaded on cache miss with reason from
	// consumeNextEagerLoadReason; forceIncludeExternal path skips.
	runInstructionsLoadedHooksForLoadResult(&opts, absOrig, result)

	return result
}

func isInstructionsMemoryType(t MemoryType) bool {
	switch t {
	case MemoryUser, MemoryProject, MemoryLocal, MemoryManaged:
		return true
	default:
		return false
	}
}

// IsTeamMemoryPromptActive mirrors loadMemoryPrompt's isTeamMemoryEnabled gate inside the TEAMMEM branch
// (FEATURE_TEAMMEM + auto memory + teamMemoryEnabled).
func IsTeamMemoryPromptActive() bool {
	return featureTeamMem() && IsAutoMemoryEnabled() && teamMemoryEnabled()
}

func teamMemoryEnabled() bool {
	if v := strings.TrimSpace(os.Getenv("CLAUDE_CODE_TEAM_MEMORY_ENABLED")); v != "" {
		return truthy(v)
	}
	// Without GrowthBook, default ON when FEATURE_TEAMMEM and auto-memory are on (team dir lives under auto mem).
	return true
}

// BuildClaudeMdString runs LoadMemoryFiles → FilterInjectedMemoryFiles → FormatGetClaudeMds.
func BuildClaudeMdString(opts LoadOptions) string {
	files := LoadMemoryFiles(opts)
	files = FilterInjectedMemoryFiles(files)
	return FormatGetClaudeMds(files)
}
