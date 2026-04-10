package commands

import (
	"regexp"
	"strings"
)

// SplitPathInFrontmatter mirrors src/utils/frontmatterParser.ts splitPathInFrontmatter.
func SplitPathInFrontmatter(input interface{}) []string {
	switch v := input.(type) {
	case nil:
		return nil
	case string:
		return splitPathInFrontmatterString(v)
	case []string:
		var out []string
		for _, s := range v {
			out = append(out, splitPathInFrontmatterString(s)...)
		}
		return out
	case []interface{}:
		var out []string
		for _, x := range v {
			out = append(out, SplitPathInFrontmatter(x)...)
		}
		return out
	default:
		return nil
	}
}

func splitPathInFrontmatterString(input string) []string {
	var parts []string
	var current strings.Builder
	braceDepth := 0
	for _, r := range input {
		switch r {
		case '{':
			braceDepth++
			current.WriteRune(r)
		case '}':
			braceDepth--
			current.WriteRune(r)
		case ',':
			if braceDepth == 0 {
				s := strings.TrimSpace(current.String())
				if s != "" {
					parts = append(parts, s)
				}
				current.Reset()
			} else {
				current.WriteRune(r)
			}
		default:
			current.WriteRune(r)
		}
	}
	s := strings.TrimSpace(current.String())
	if s != "" {
		parts = append(parts, s)
	}
	var out []string
	for _, p := range parts {
		if p == "" {
			continue
		}
		out = append(out, expandBraces(p)...)
	}
	return out
}

var braceGroup = regexp.MustCompile(`^([^{]*)\{([^}]+)\}(.*)$`)

func expandBraces(pattern string) []string {
	m := braceGroup.FindStringSubmatch(pattern)
	if m == nil {
		return []string{pattern}
	}
	prefix, alternatives, suffix := m[1], m[2], m[3]
	alts := strings.Split(alternatives, ",")
	var expanded []string
	for _, alt := range alts {
		alt = strings.TrimSpace(alt)
		combined := prefix + alt + suffix
		expanded = append(expanded, expandBraces(combined)...)
	}
	return expanded
}

// ParseSkillPaths mirrors parseSkillPaths in src/skills/loadSkillsDir.ts (paths frontmatter → optional []string).
func ParseSkillPaths(pathsRaw interface{}) []string {
	patterns := SplitPathInFrontmatter(pathsRaw)
	if len(patterns) == 0 {
		return nil
	}
	out := make([]string, 0, len(patterns))
	for _, p := range patterns {
		p = strings.TrimSpace(p)
		if strings.HasSuffix(p, "/**") {
			p = strings.TrimSuffix(p, "/**")
		}
		if p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return nil
	}
	allGlob := true
	for _, p := range out {
		if p != "**" {
			allGlob = false
			break
		}
	}
	if allGlob {
		return nil
	}
	return out
}
