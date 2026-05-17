package handlers

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SkillsResult is the JSON payload for /skills.
type SkillsResult struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// HandleSkillsCommand handles /skills — lists loaded skills.
func HandleSkillsCommand(args string) ([]byte, error) {
	cwd, _ := os.Getwd()
	skillDirs := []string{
		filepath.Join(cwd, ".claude", "skills"),
		filepath.Join(cwd, "skills"),
	}

	var lines []string
	lines = append(lines, "Loaded skills:")

	found := false
	for _, dir := range skillDirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() {
				skillFile := filepath.Join(dir, e.Name(), "SKILL.md")
				if _, err := os.Stat(skillFile); err == nil {
					lines = append(lines, fmt.Sprintf("  %s (%s)", e.Name(), dir))
					found = true
				}
			}
		}
	}

	if !found {
		lines = append(lines, "  No skills found.")
		lines = append(lines, "\nSkills are defined in .claude/skills/ or skills/ directories.")
		lines = append(lines, "Each skill has a SKILL.md file with a skill frontmatter.")
	}
	return json.Marshal(SkillsResult{Type: "text", Value: strings.Join(lines, "\n")})
}
