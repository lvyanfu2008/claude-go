package slashresolve

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"goc/types"
)

// ClaudeAPISkillFiles when non-nil is merged like TS claudeApiContent.SKILL_FILES (path -> markdown).
// Populate via codegen from claude-code (see claudeApiContent.ts). When nil, resolveClaudeAPI uses bundleddata/claude-api.md stub.
var ClaudeAPISkillFiles map[string]string

// ClaudeAPISkillPrompt when set overrides embedded SKILL.md body (prefix before "## Reading Guide").
var ClaudeAPISkillPrompt string

// ClaudeAPIModelVars matches claudeApiContent.ts SKILL_MODEL_VARS.
var ClaudeAPIModelVars = map[string]string{
	"OPUS_ID":        "claude-opus-4-6",
	"OPUS_NAME":      "Claude Opus 4.6",
	"SONNET_ID":      "claude-sonnet-4-6",
	"SONNET_NAME":    "Claude Sonnet 4.6",
	"HAIKU_ID":       "claude-haiku-4-5",
	"HAIKU_NAME":     "Claude Haiku 4.5",
	"PREV_SONNET_ID": "claude-sonnet-4-5",
}

var (
	claudeAPIHTMLCommentRe = regexp.MustCompile(`<!--[\s\S]*?-->\n?`)
	claudeAPIVarRe         = regexp.MustCompile(`\{\{(\w+)\}\}`)
)

// languageIndicators order matches src/skills/bundled/claudeApi.ts LANGUAGE_INDICATORS iteration.
var claudeAPILanguageIndicators = []struct {
	lang        string
	indicators  []string
}{
	{"python", []string{".py", "requirements.txt", "pyproject.toml", "setup.py", "Pipfile"}},
	{"typescript", []string{".ts", ".tsx", "tsconfig.json", "package.json"}},
	{"java", []string{".java", "pom.xml", "build.gradle"}},
	{"go", []string{".go", "go.mod"}},
	{"ruby", []string{".rb", "Gemfile"}},
	{"csharp", []string{".cs", ".csproj"}},
	{"php", []string{".php", "composer.json"}},
}

// DetectClaudeAPILanguage mirrors detectLanguage() in claudeApi.ts (root of cwd only).
func DetectClaudeAPILanguage(cwd string) string {
	if cwd == "" {
		cwd = "."
	}
	entries, err := os.ReadDir(cwd)
	if err != nil {
		return ""
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	for _, row := range claudeAPILanguageIndicators {
		for _, ind := range row.indicators {
			if strings.HasPrefix(ind, ".") {
				for _, n := range names {
					if strings.HasSuffix(n, ind) {
						return row.lang
					}
				}
			} else {
				for _, n := range names {
					if n == ind {
						return row.lang
					}
				}
			}
		}
	}
	return ""
}

func processClaudeAPIMarkdown(md string, vars map[string]string) string {
	out := md
	for {
		next := claudeAPIHTMLCommentRe.ReplaceAllString(out, "")
		if next == out {
			break
		}
		out = next
	}
	out = claudeAPIVarRe.ReplaceAllStringFunc(out, func(m string) string {
		sub := claudeAPIVarRe.FindStringSubmatch(m)
		if len(sub) < 2 {
			return m
		}
		key := sub[1]
		if v, ok := vars[key]; ok {
			return v
		}
		return m
	})
	return out
}

func claudeAPIFilesForLanguage(lang string, files map[string]string) []string {
	if files == nil {
		return nil
	}
	var keys []string
	prefix := lang + "/"
	for k := range files {
		if strings.HasPrefix(k, prefix) || strings.HasPrefix(k, "shared/") {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	return keys
}

func buildClaudeAPIInlineReference(paths []string, files map[string]string, vars map[string]string) string {
	var sections []string
	for _, p := range paths {
		md := files[p]
		if md == "" {
			continue
		}
		body := strings.TrimSpace(processClaudeAPIMarkdown(md, vars))
		sections = append(sections, `<doc path="`+p+`">`+"\n"+body+"\n</doc>")
	}
	return strings.Join(sections, "\n\n")
}

// claudeAPIInlineReadingGuideForLang matches INLINE_READING_GUIDE in claudeApi.ts.
func claudeAPIInlineReadingGuideForLang(lang string) string {
	return fmt.Sprintf(`## Reference Documentation

The relevant documentation for your detected language is included below in `+"`<doc>`"+` tags. Each tag has a `+"`path`"+` attribute showing its original file path. Use this to find the right section:

### Quick Task Reference

**Single text classification/summarization/extraction/Q&A:**
→ Refer to `+"`%s/claude-api/README.md`"+`

**Chat UI or real-time response display:**
→ Refer to `+"`%s/claude-api/README.md`"+` + `+"`%s/claude-api/streaming.md`"+`

**Long-running conversations (may exceed context window):**
→ Refer to `+"`%s/claude-api/README.md`"+` — see Compaction section

**Prompt caching / optimize caching / "why is my cache hit rate low":**
→ Refer to `+"`shared/prompt-caching.md`"+` + `+"`%s/claude-api/README.md`"+` (Prompt Caching section)

**Function calling / tool use / agents:**
→ Refer to `+"`%s/claude-api/README.md`"+` + `+"`shared/tool-use-concepts.md`"+` + `+"`%s/claude-api/tool-use.md`"+`

**Batch processing (non-latency-sensitive):**
→ Refer to `+"`%s/claude-api/README.md`"+` + `+"`%s/claude-api/batches.md`"+`

**File uploads across multiple requests:**
→ Refer to `+"`%s/claude-api/README.md`"+` + `+"`%s/claude-api/files-api.md`"+`

**Agent with built-in tools (file/web/terminal) (Python & TypeScript only):**
→ Refer to `+"`%s/agent-sdk/README.md`"+` + `+"`%s/agent-sdk/patterns.md`"+`

**Error handling:**
→ Refer to `+"`shared/error-codes.md`"+`

**Latest docs via WebFetch:**
→ Refer to `+"`shared/live-sources.md`"+` for URLs`,
		lang, lang, lang, lang, lang, lang, lang, lang, lang, lang, lang, lang, lang)
}

// BuildClaudeAPIPrompt mirrors buildPrompt() in claudeApi.ts.
func BuildClaudeAPIPrompt(lang string, args string, skillPrompt string, files map[string]string, modelVars map[string]string) string {
	if modelVars == nil {
		modelVars = ClaudeAPIModelVars
	}
	cleanPrompt := processClaudeAPIMarkdown(skillPrompt, modelVars)
	idx := strings.Index(cleanPrompt, "## Reading Guide")
	basePrompt := cleanPrompt
	if idx >= 0 {
		basePrompt = strings.TrimRight(cleanPrompt[:idx], " \t\n")
	}
	var parts []string
	parts = append(parts, basePrompt)

	langTag := strings.TrimSpace(lang)
	if langTag != "" {
		guide := claudeAPIInlineReadingGuideForLang(langTag)
		paths := claudeAPIFilesForLanguage(langTag, files)
		parts = append(parts, guide)
		parts = append(parts, "---\n\n## Included Documentation\n\n"+buildClaudeAPIInlineReference(paths, files, modelVars))
	} else {
		guide := claudeAPIInlineReadingGuideForLang("unknown")
		parts = append(parts, guide)
		parts = append(parts, "No project language was auto-detected. Ask the user which language they are using, then refer to the matching docs below.")
		var allKeys []string
		for k := range files {
			allKeys = append(allKeys, k)
		}
		sort.Strings(allKeys)
		parts = append(parts, "---\n\n## Included Documentation\n\n"+buildClaudeAPIInlineReference(allKeys, files, modelVars))
	}
	webFetchIdx := strings.Index(cleanPrompt, "## When to Use WebFetch")
	if webFetchIdx >= 0 {
		parts = append(parts, strings.TrimRight(cleanPrompt[webFetchIdx:], " \t\n"))
	}
	if strings.TrimSpace(args) != "" {
		parts = append(parts, "## User Request\n\n"+args)
	}
	return strings.Join(parts, "\n\n")
}

func resolveClaudeAPI(args string, cwd string) (types.SlashResolveResult, error) {
	if cwd == "" {
		cwd = "."
	}
	cwd, _ = filepath.Abs(cwd)

	if ClaudeAPISkillFiles != nil && len(ClaudeAPISkillFiles) > 0 && strings.TrimSpace(ClaudeAPISkillPrompt) != "" {
		lang := DetectClaudeAPILanguage(cwd)
		text := BuildClaudeAPIPrompt(lang, args, ClaudeAPISkillPrompt, ClaudeAPISkillFiles, ClaudeAPIModelVars)
		return types.SlashResolveResult{UserText: text, Source: types.SlashResolveBundledEmbed}, nil
	}

	body, err := readBundledText("claude-api.md")
	if err != nil {
		return types.SlashResolveResult{}, err
	}
	if lang := DetectClaudeAPILanguage(cwd); lang != "" {
		body = "## Language detection (project root)\nAuto-detected **" + lang + "** using the same filename markers as TypeScript `claudeApi.ts`.\n\n" + body
	}
	return types.SlashResolveResult{
		UserText: appendUserSection(body, args),
		Source:   types.SlashResolveBundledEmbed,
	}, nil
}
