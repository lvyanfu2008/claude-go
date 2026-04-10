package claudemd

import (
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// frontmatterRegex mirrors src/utils/frontmatterParser.ts FRONTMATTER_REGEX.
var frontmatterRegex = regexp.MustCompile(`(?s)^---\s*\n([\s\S]*?)---\s*\n?`)

// ParseFrontmatter mirrors parseFrontmatter (YAML body + content after closing ---).
func ParseFrontmatter(markdown string) (frontmatter map[string]interface{}, content string) {
	m := frontmatterRegex.FindStringSubmatchIndex(markdown)
	if m == nil {
		return map[string]interface{}{}, markdown
	}
	fmText := markdown[m[2]:m[3]]
	content = markdown[m[1]:] // after closing --- of frontmatter
	raw := map[string]interface{}{}
	if err := yaml.Unmarshal([]byte(fmText), &raw); err != nil {
		// TS retries quoteProblematicValues; we keep empty frontmatter + body after ---.
		return map[string]interface{}{}, content
	}
	return raw, content
}

// ParseFrontmatterPaths mirrors claudemd.ts parseFrontmatterPaths.
func ParseFrontmatterPaths(rawContent string) (content string, globs []string) {
	fm, body := ParseFrontmatter(rawContent)
	if len(fm) == 0 {
		return body, nil
	}
	pathsVal, ok := fm["paths"]
	if !ok || pathsVal == nil {
		return body, nil
	}
	patterns := splitPathInFrontmatter(pathsVal)
	var filtered []string
	for _, pattern := range patterns {
		p := strings.TrimSpace(pattern)
		if strings.HasSuffix(p, "/**") {
			p = strings.TrimSuffix(p, "/**")
		}
		p = strings.TrimSpace(p)
		if p != "" {
			filtered = append(filtered, p)
		}
	}
	if len(filtered) == 0 {
		return body, nil
	}
	allStar := true
	for _, p := range filtered {
		if p != "**" {
			allStar = false
			break
		}
	}
	if allStar {
		return body, nil
	}
	return body, filtered
}

func splitPathInFrontmatter(input interface{}) []string {
	switch v := input.(type) {
	case string:
		return splitCommaRespectBraces(v)
	case []interface{}:
		var out []string
		for _, it := range v {
			if s, ok := it.(string); ok {
				out = append(out, splitCommaRespectBraces(s)...)
			}
		}
		return out
	default:
		return nil
	}
}

func splitCommaRespectBraces(input string) []string {
	var parts []string
	var current strings.Builder
	braceDepth := 0
	for _, ch := range input {
		switch ch {
		case '{':
			braceDepth++
			current.WriteRune(ch)
		case '}':
			braceDepth--
			current.WriteRune(ch)
		case ',':
			if braceDepth == 0 {
				s := strings.TrimSpace(current.String())
				if s != "" {
					parts = append(parts, s)
				}
				current.Reset()
			} else {
				current.WriteRune(ch)
			}
		default:
			current.WriteRune(ch)
		}
	}
	s := strings.TrimSpace(current.String())
	if s != "" {
		parts = append(parts, s)
	}
	var expanded []string
	for _, p := range parts {
		if p == "" {
			continue
		}
		expanded = append(expanded, expandBraces(p)...)
	}
	return expanded
}

func expandBraces(pattern string) []string {
	re := regexp.MustCompile(`^([^{]*)\{([^}]+)\}(.*)$`)
	m := re.FindStringSubmatch(pattern)
	if m == nil {
		return []string{pattern}
	}
	prefix, alts, suffix := m[1], m[2], m[3]
	var altsParts []string
	for _, alt := range strings.Split(alts, ",") {
		t := strings.TrimSpace(alt)
		if t != "" {
			altsParts = append(altsParts, t)
		}
	}
	var out []string
	for _, part := range altsParts {
		combined := prefix + part + suffix
		out = append(out, expandBraces(combined)...)
	}
	return out
}
