package commands

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"goc/claudemd"
	"goc/commands/featuregates"
)

// Embedded auto-only bodies: TS src/memdir/memdir.ts buildMemoryLines('auto memory', '/__MEMORY_DIR__/', …).
// Regenerate (from claude-code):
//
//	bun -e "const {buildMemoryLines}=require('./src/memdir/memdir.ts');require('fs').writeFileSync('../claude-go/commands/data/memory_prompt_auto_individual.txt', buildMemoryLines('auto memory','/__MEMORY_DIR__/',undefined,false).join('\n'))"
//	bun -e "const {buildMemoryLines}=require('./src/memdir/memdir.ts');require('fs').writeFileSync('../claude-go/commands/data/memory_prompt_auto_skip_index.txt', buildMemoryLines('auto memory','/__MEMORY_DIR__/',undefined,true).join('\n'))"
//
// KAIROS + TEAM combined: scripts/dump-memory-prompts-for-go.ts

//go:embed data/memory_prompt_auto_individual.txt
var memoryPromptAutoIndividualTmpl string

//go:embed data/memory_prompt_auto_skip_index.txt
var memoryPromptAutoSkipIndexTmpl string

//go:embed data/memory_prompt_kairos_daily_index.txt
var memoryPromptKairosDailyIndexTmpl string

//go:embed data/memory_prompt_kairos_daily_skip_index.txt
var memoryPromptKairosDailySkipIndexTmpl string

//go:embed data/memory_prompt_team_combined_index.txt
var memoryPromptTeamCombinedIndexTmpl string

//go:embed data/memory_prompt_team_combined_skip_index.txt
var memoryPromptTeamCombinedSkipIndexTmpl string

const (
	memoryPromptDirPlaceholder         = "/__MEMORY_DIR__/"
	memoryPromptKairosPathPlaceholder  = "/__AUTO_MEM_PATH__/"
	memoryPromptTeamAutoDirPlaceholder = "/__AUTO_DIR__/"
	memoryPromptTeamTeamDirPlaceholder = "/__TEAM_DIR__/"
)

func memoryDirDisplayPath(memDir string) string {
	p := strings.TrimSpace(memDir)
	if p == "" {
		return ""
	}
	p = filepath.Clean(strings.TrimSuffix(p, string(filepath.Separator)))
	p = filepath.ToSlash(p)
	return p + "/"
}

// claudeProjectSessionDir mirrors getProjectDir(getOriginalCwd()) for transcript paths in buildSearchingPastContextSection.
func claudeProjectSessionDir(originalCwd string) string {
	abs, err := filepath.Abs(strings.TrimSpace(originalCwd))
	if err != nil || abs == "" {
		abs = "."
	}
	key := claudemd.SanitizePath(abs)
	if cr := claudemd.ResolveCanonicalGitRoot(abs); cr != "" {
		key = claudemd.SanitizePath(cr)
	}
	base := claudemd.MemoryBaseDir()
	if base == "" {
		return ""
	}
	return filepath.Join(base, "projects", key)
}

func memorySearchingPastContextBlock(memoryDir, projectDir string, embeddedOrRepl bool) string {
	memSearch := fmt.Sprintf(`%s with pattern="<search term>" path="%s" glob="*.md"`, grepToolName, memoryDir)
	transcriptSearch := fmt.Sprintf(`%s with pattern="<search term>" path="%s/" glob="*.jsonl"`, grepToolName, strings.TrimSuffix(filepath.ToSlash(projectDir), "/"))
	if embeddedOrRepl {
		memSearch = fmt.Sprintf(`grep -rn "<search term>" %s --include="*.md"`, memoryDir)
		transcriptSearch = fmt.Sprintf(`grep -rn "<search term>" %s/ --include="*.jsonl"`, strings.TrimSuffix(filepath.ToSlash(projectDir), "/"))
	}
	return fmt.Sprintf("## Searching past context\n\nWhen looking for past context:\n1. Search topic files in your memory directory:\n```\n%s\n```\n2. Session transcript logs (last resort — large files, slow):\n```\n%s\n```\nUse narrow search terms (error messages, file paths, function names) rather than broad keywords.\n", memSearch, transcriptSearch)
}

func appendMemorySearchPastContext(s string, memDir, cwd string, o GouDemoSystemOpts) string {
	if !o.MemorySearchPastContext {
		return s
	}
	pdir := claudeProjectSessionDir(cwd)
	if pdir == "" {
		return s
	}
	md := strings.TrimSuffix(filepath.ToSlash(memDir), "/")
	return s + "\n\n" + memorySearchingPastContextBlock(md, pdir, o.EmbeddedSearchTools || o.ReplModeEnabled)
}

// BuildAutoMemoryPrompt mirrors loadMemoryPrompt() (src/memdir/memdir.ts): KAIROS daily log, TEAMMEM combined,
// auto-only buildMemoryLines; ensureMemoryDirExists on team + auto-only branches; cowork extra only on team + auto.
func BuildAutoMemoryPrompt(o GouDemoSystemOpts) string {
	if !claudemd.IsAutoMemoryEnabled() {
		return ""
	}
	cwd := strings.TrimSpace(o.Cwd)
	if cwd == "" {
		cwd = "."
	}
	memDir := claudemd.GetAutoMemPath(cwd)
	skipIndex := o.MemorySkipIndex

	if featuregates.Feature("KAIROS") && o.KairosActive {
		tmpl := memoryPromptKairosDailyIndexTmpl
		if skipIndex {
			tmpl = memoryPromptKairosDailySkipIndexTmpl
		}
		s := strings.ReplaceAll(tmpl, memoryPromptKairosPathPlaceholder, memoryDirDisplayPath(memDir))
		s = appendMemorySearchPastContext(s, memDir, cwd, o)
		return strings.TrimSpace(s)
	}

	if featuregates.Feature("TEAMMEM") && claudemd.IsTeamMemoryPromptActive() {
		teamDir := claudemd.GetTeamMemPath(cwd)
		_ = claudemd.EnsureMemoryDirExists(teamDir)
		tmpl := memoryPromptTeamCombinedIndexTmpl
		if skipIndex {
			tmpl = memoryPromptTeamCombinedSkipIndexTmpl
		}
		s := strings.ReplaceAll(tmpl, memoryPromptTeamAutoDirPlaceholder, memoryDirDisplayPath(memDir))
		s = strings.ReplaceAll(s, memoryPromptTeamTeamDirPlaceholder, memoryDirDisplayPath(teamDir))
		if x := strings.TrimSpace(os.Getenv("CLAUDE_COWORK_MEMORY_EXTRA_GUIDELINES")); x != "" {
			s += "\n\n" + x + "\n"
		}
		s = appendMemorySearchPastContext(s, memDir, cwd, o)
		return strings.TrimSpace(s)
	}

	_ = claudemd.EnsureMemoryDirExists(memDir)
	tmpl := memoryPromptAutoIndividualTmpl
	if skipIndex {
		tmpl = memoryPromptAutoSkipIndexTmpl
	}
	s := strings.ReplaceAll(tmpl, memoryPromptDirPlaceholder, memoryDirDisplayPath(memDir))
	if x := strings.TrimSpace(os.Getenv("CLAUDE_COWORK_MEMORY_EXTRA_GUIDELINES")); x != "" {
		s += "\n\n" + x + "\n"
	}
	s = appendMemorySearchPastContext(s, memDir, cwd, o)
	return strings.TrimSpace(s)
}
