package claudemd

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"goc/hookexec"
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

// LoadMemoryFiles mirrors getMemoryFiles (no memoization, no hooks/analytics).
func LoadMemoryFiles(opts LoadOptions) []MemoryFileInfo {
	// 使用增强版的记忆层次结构管理
	return LoadMemoryFilesEnhanced(opts)
}

// LoadMemoryFilesEnhanced 使用完整的记忆层次结构管理加载记忆文件
func LoadMemoryFilesEnhanced(opts LoadOptions) []MemoryFileInfo {
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

	// InstructionsLoaded hooks (TS getMemoryFiles): observability-only, fire-and-forget; skip forceIncludeExternal path.
	if !opts.ForceIncludeExternal {
		tab, err := hookexec.MergedHooksForCwd(absOrig)
		if err == nil && hookexec.HasInstructionsLoaded(tab) {
			base := hookexec.BaseHookInput{
				SessionID:      "local",
				TranscriptPath: "",
				Cwd:            absOrig,
			}
			for _, file := range result {
				if !isInstructionsMemoryType(file.Type) {
					continue
				}
				hookexec.FireInstructionsLoaded(context.Background(), tab, absOrig, base, hookexec.InstructionsLoadedFields{
					FilePath:       file.Path,
					MemoryType:     string(file.Type),
					LoadReason:     "session_start",
					Globs:          file.Globs,
					ParentFilePath: file.Parent,
				}, hookexec.DefaultHookTimeoutMs)
			}
		}
	}

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
