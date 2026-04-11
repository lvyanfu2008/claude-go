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

// LoadMemoryFiles mirrors getMemoryFiles (no memoization, no hooks/analytics).
func LoadMemoryFiles(opts LoadOptions) []MemoryFileInfo {
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
	excludeChecker := NewExcludeChecker(excl)

	processed := map[string]struct{}{}
	var result []MemoryFileInfo

	// Managed
	managedPath := MemoryPath(MemoryManaged, absOrig)
	result = append(result, ProcessMemoryFile(managedPath, MemoryManaged, processed, includeExternal, absOrig, 0, "", excludeChecker)...)
	result = append(result, ProcessMdRules(managedClaudeRulesDir(), MemoryManaged, processed, includeExternal, absOrig, false, nil, excludeChecker)...)

	// User (mirrors isSettingSourceEnabled('userSettings'); env can disable)
	if userMemoryEnabled() {
		if udir, err := userClaudeRulesDir(); err == nil {
			userPath := MemoryPath(MemoryUser, absOrig)
			result = append(result, ProcessMemoryFile(userPath, MemoryUser, processed, true, absOrig, 0, "", excludeChecker)...)
			result = append(result, ProcessMdRules(udir, MemoryUser, processed, true, absOrig, false, nil, excludeChecker)...)
		}
	}

	dirs := directoryChainUp(absOrig)
	gitRoot := FindGitRoot(absOrig)
	canonicalRoot := ResolveCanonicalGitRoot(absOrig)
	nested := gitRoot != "" && canonicalRoot != "" &&
		NormalizePathForComparison(gitRoot) != NormalizePathForComparison(canonicalRoot) &&
		PathInWorkingPath(gitRoot, canonicalRoot)

	// root → cwd
	for i := len(dirs) - 1; i >= 0; i-- {
		dir := dirs[i]
		skipProject := nested &&
			PathInWorkingPath(dir, canonicalRoot) &&
			!PathInWorkingPath(dir, gitRoot)

		if projectMemoryEnabled() && !skipProject {
			p := filepath.Join(dir, "CLAUDE.md")
			result = append(result, ProcessMemoryFile(p, MemoryProject, processed, includeExternal, absOrig, 0, "", excludeChecker)...)
			dot := filepath.Join(dir, ".claude", "CLAUDE.md")
			result = append(result, ProcessMemoryFile(dot, MemoryProject, processed, includeExternal, absOrig, 0, "", excludeChecker)...)
			rules := filepath.Join(dir, ".claude", "rules")
			result = append(result, ProcessMdRules(rules, MemoryProject, processed, includeExternal, absOrig, false, nil, excludeChecker)...)
		}
		if localMemoryEnabled() {
			local := filepath.Join(dir, "CLAUDE.local.md")
			result = append(result, ProcessMemoryFile(local, MemoryLocal, processed, includeExternal, absOrig, 0, "", excludeChecker)...)
		}
	}

	// TS gates env-only extra dirs on CLAUDE_CODE_ADDITIONAL_DIRECTORIES_CLAUDE_MD; when callers pass
	// [LoadOptions.AdditionalWorkingDirs] explicitly (e.g. gou-demo [querycontext.ExtraClaudeMdRootsForFetch]),
	// always scan them so userContext includes those CLAUDE.md files without an extra env toggle.
	if truthy(os.Getenv("CLAUDE_CODE_ADDITIONAL_DIRECTORIES_CLAUDE_MD")) || len(opts.AdditionalWorkingDirs) > 0 {
		for _, d := range opts.AdditionalWorkingDirs {
			ad, err := filepath.Abs(strings.TrimSpace(d))
			if err != nil || ad == "" {
				continue
			}
			result = append(result, ProcessMemoryFile(filepath.Join(ad, "CLAUDE.md"), MemoryProject, processed, includeExternal, absOrig, 0, "", excludeChecker)...)
			result = append(result, ProcessMemoryFile(filepath.Join(ad, ".claude", "CLAUDE.md"), MemoryProject, processed, includeExternal, absOrig, 0, "", excludeChecker)...)
			result = append(result, ProcessMdRules(filepath.Join(ad, ".claude", "rules"), MemoryProject, processed, includeExternal, absOrig, false, nil, excludeChecker)...)
		}
	}

	if IsAutoMemoryEnabled() {
		autoPath := filepath.Join(strings.TrimSuffix(GetAutoMemPath(absOrig), string(filepath.Separator)), "MEMORY.md")
		if b, err := os.ReadFile(autoPath); err == nil {
			// TS: safelyReadMemoryFileAsync without includeBasePath — no @include expansion for AutoMem index.
			if info, _ := ParseMemoryFileContent(string(b), autoPath, MemoryAutoMem, ""); info != nil && strings.TrimSpace(info.Content) != "" {
				norm := NormalizePathForComparison(info.Path)
				if _, ok := processed[norm]; !ok {
					processed[norm] = struct{}{}
					result = append(result, *info)
				}
			}
		}
	}

	if featureTeamMem() && IsAutoMemoryEnabled() && teamMemoryEnabled() {
		teamPath := filepath.Join(strings.TrimSuffix(GetAutoMemPath(absOrig), string(filepath.Separator)), "team", "MEMORY.md")
		if b, err := os.ReadFile(teamPath); err == nil {
			if info, _ := ParseMemoryFileContent(string(b), teamPath, MemoryTeamMem, ""); info != nil && strings.TrimSpace(info.Content) != "" {
				norm := NormalizePathForComparison(info.Path)
				if _, ok := processed[norm]; !ok {
					processed[norm] = struct{}{}
					result = append(result, *info)
				}
			}
		}
	}

	return result
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
