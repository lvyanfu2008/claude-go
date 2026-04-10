package commands

import (
	"path/filepath"
	"strings"
	"sync"

	gitignore "github.com/sabhiram/go-gitignore"

	"goc/types"
)

// Session state mirrors src/skills/loadSkillsDir.ts conditionalSkills (pending + activated names).
// Activated skills are merged into the shared dynamic map via [dynamicSkillsSessionSet] (TS dynamicSkills.set).

var (
	condMu sync.RWMutex

	// conditionalPending: skills with paths frontmatter not yet matched by a touched file.
	conditionalPending map[string]types.Command

	// activatedConditionalNames: names activated this session (TS activatedConditionalSkillNames).
	activatedConditionalNames map[string]struct{}
)

// clearConditionalRuntimeMaps clears conditional pending and activation tracking (TS conditionalSkills +
// activatedConditionalSkillNames). Used by [ClearDynamicSkills] and tests via [ClearLoadAllCommandsCache].
func clearConditionalRuntimeMaps() {
	condMu.Lock()
	defer condMu.Unlock()
	conditionalPending = nil
	activatedConditionalNames = nil
}

// ConditionalPendingCount returns len(conditionalPending) (for tests).
func ConditionalPendingCount() int {
	condMu.RLock()
	defer condMu.RUnlock()
	if conditionalPending == nil {
		return 0
	}
	return len(conditionalPending)
}

// ActivatedConditionalSkillName reports whether name was activated via paths (for listing).
func ActivatedConditionalSkillName(name string) bool {
	condMu.RLock()
	defer condMu.RUnlock()
	if activatedConditionalNames == nil {
		return false
	}
	_, ok := activatedConditionalNames[name]
	return ok
}

func syncConditionalSkillsFromLoaded(cmds []types.Command) {
	condMu.Lock()
	defer condMu.Unlock()
	if conditionalPending == nil {
		conditionalPending = make(map[string]types.Command)
	}
	next := make(map[string]types.Command)
	for _, c := range cmds {
		if c.Type != "prompt" || len(c.Paths) == 0 || PathsAreOnlyDoubleStar(c.Paths) {
			continue
		}
		if activatedConditionalNames != nil {
			if _, ok := activatedConditionalNames[c.Name]; ok {
				continue
			}
		}
		next[c.Name] = c
	}
	conditionalPending = next
}

// ActivateConditionalSkillsForPaths mirrors TS activateConditionalSkillsForPaths: for each pending
// conditional skill, if any file path matches paths (gitignore-style), move skill to dynamic list
// and record activation. Returns newly activated names.
func ActivateConditionalSkillsForPaths(filePaths []string, cwd string) []string {
	if len(filePaths) == 0 {
		return nil
	}
	cwdAbs, err := filepath.Abs(cwd)
	if err != nil {
		return nil
	}
	cwdAbs = filepath.Clean(cwdAbs)

	condMu.Lock()
	if len(conditionalPending) == 0 {
		condMu.Unlock()
		return nil
	}

	toActivate := make([]string, 0)

	for name, skill := range conditionalPending {
		if skill.Type != "prompt" || len(skill.Paths) == 0 {
			continue
		}
		gi := compileSkillPathsIgnore(skill.Paths)
		if gi == nil {
			continue
		}
		for _, fp := range filePaths {
			rel := relativePathForConditionalMatch(fp, cwdAbs)
			if rel == "" {
				continue
			}
			if gi.MatchesPath(rel) {
				toActivate = append(toActivate, name)
				break
			}
		}
	}

	type activatedPair struct {
		name  string
		skill types.Command
	}
	var batch []activatedPair
	var activated []string
	for _, name := range toActivate {
		skill, ok := conditionalPending[name]
		if !ok {
			continue
		}
		delete(conditionalPending, name)
		if activatedConditionalNames == nil {
			activatedConditionalNames = make(map[string]struct{})
		}
		activatedConditionalNames[name] = struct{}{}
		batch = append(batch, activatedPair{name: name, skill: skill})
		activated = append(activated, name)
	}
	condMu.Unlock()

	for _, p := range batch {
		dynamicSkillsSessionSet(p.name, p.skill)
	}
	if len(batch) > 0 {
		ClearCommandMemoizationCaches()
	}
	return activated
}

func compileSkillPathsIgnore(patterns []string) *gitignore.GitIgnore {
	lines := make([]string, 0, len(patterns))
	for _, p := range patterns {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if strings.HasSuffix(p, "/**") {
			p = strings.TrimSuffix(p, "/**")
		}
		if p != "" {
			lines = append(lines, p)
		}
	}
	if len(lines) == 0 {
		return nil
	}
	return gitignore.CompileIgnoreLines(lines...)
}

func relativePathForConditionalMatch(filePath string, cwdAbs string) string {
	fp := strings.TrimSpace(filePath)
	if fp == "" {
		return ""
	}
	var abs string
	if filepath.IsAbs(fp) {
		abs = filepath.Clean(fp)
	} else {
		var err error
		abs, err = filepath.Abs(filepath.Join(cwdAbs, fp))
		if err != nil {
			return ""
		}
	}
	rel, err := filepath.Rel(cwdAbs, abs)
	if err != nil {
		return ""
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return ""
	}
	if filepath.IsAbs(rel) {
		return ""
	}
	return filepath.ToSlash(rel)
}
