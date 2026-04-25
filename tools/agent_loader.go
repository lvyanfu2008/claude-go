package tools

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"goc/claudemd"
	"goc/commands"
	"goc/tools/hookstypes"
)

type AgentLoadFailure struct {
	Path  string `json:"path"`
	Error string `json:"error"`
}

type AgentDefinitionsReport struct {
	ActiveAgents []AgentDefinition  `json:"activeAgents"`
	AllAgents    []AgentDefinition  `json:"allAgents"`
	FailedFiles  []AgentLoadFailure `json:"failedFiles,omitempty"`
}

func LoadAgentDefinitionsReport(cwd string) AgentDefinitionsReport {
	builtins := LoadAgentDefinitionsBuiltins()
	all := make([]AgentDefinition, 0, len(builtins)+16)
	all = append(all, builtins...)
	var failed []AgentLoadFailure

	loadOrder := []struct {
		source string
		dir    string
	}{
		{source: "userSettings", dir: filepath.Join(commands.ClaudeConfigHome(), "agents")},
		{source: "projectSettings", dir: filepath.Join(strings.TrimSpace(cwd), ".claude", "agents")},
		{source: "policySettings", dir: filepath.Join(commands.ManagedFilePath(), ".claude", "agents")},
	}

	for _, item := range loadOrder {
		entries, ferrs := loadAgentMarkdownDir(item.dir, item.source)
		all = append(all, entries...)
		failed = append(failed, ferrs...)
	}

	active := dedupeAgentsByTypeOrder(all)
	return AgentDefinitionsReport{
		ActiveAgents: active,
		AllAgents:    all,
		FailedFiles:  failed,
	}
}

func LoadAgentDefinitionsBuiltins() []AgentDefinition {
	cfg := builtinConfigFromEnv()
	builtins := getBuiltinAgents(cfg)
	out := make([]AgentDefinition, 0, len(builtins))
	for _, b := range builtins {
		out = append(out, AgentDefinition{
			AgentType:                          b.AgentType,
			WhenToUse:                          b.WhenToUse,
			Tools:                              append([]string(nil), b.Tools...),
			DisallowedTools:                    append([]string(nil), b.DisallowedTools...),
			Source:                             b.Source,
			Model:                              b.Model,
			PermissionMode:                     b.PermissionMode,
			Background:                         b.Background,
			SystemPrompt:                       b.SystemPrompt,
			OmitClaudeMd:                       b.OmitClaudeMd,
			Hooks:                             b.Hooks,
		})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].AgentType < out[j].AgentType })
	return out
}

func dedupeAgentsByTypeOrder(all []AgentDefinition) []AgentDefinition {
	m := map[string]AgentDefinition{}
	for _, a := range all {
		m[a.AgentType] = a
	}
	out := make([]AgentDefinition, 0, len(m))
	for _, a := range m {
		out = append(out, a)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].AgentType < out[j].AgentType })
	return out
}

func loadAgentMarkdownDir(dir, source string) ([]AgentDefinition, []AgentLoadFailure) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return nil, nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, []AgentLoadFailure{{Path: dir, Error: err.Error()}}
	}
	var out []AgentDefinition
	var failed []AgentLoadFailure
	for _, ent := range entries {
		if ent.IsDir() {
			continue
		}
		if !strings.HasSuffix(strings.ToLower(ent.Name()), ".md") {
			continue
		}
		path := filepath.Join(dir, ent.Name())
		raw, err := os.ReadFile(path)
		if err != nil {
			failed = append(failed, AgentLoadFailure{Path: path, Error: err.Error()})
			continue
		}
		agent, ok, parseErr := parseAgentMarkdown(path, string(raw), source)
		if parseErr != "" {
			failed = append(failed, AgentLoadFailure{Path: path, Error: parseErr})
		}
		if ok {
			out = append(out, agent)
		}
	}
	return out, failed
}

func parseAgentMarkdown(path, markdown, source string) (AgentDefinition, bool, string) {
	fm, _ := claudemd.ParseFrontmatter(markdown)
	name, _ := fm["name"].(string)
	desc, _ := fm["description"].(string)
	name = strings.TrimSpace(name)
	desc = strings.TrimSpace(desc)
	if name == "" && desc == "" {
		return AgentDefinition{}, false, ""
	}
	if name == "" {
		return AgentDefinition{}, false, `missing required "name" in frontmatter`
	}
	if desc == "" {
		return AgentDefinition{}, false, `missing required "description" in frontmatter`
	}
	model, _ := fm["model"].(string)
	model = strings.TrimSpace(model)
	if strings.EqualFold(model, "inherit") {
		model = ""
	}
	background := false
	switch v := fm["background"].(type) {
	case bool:
		background = v
	case string:
		background = strings.EqualFold(strings.TrimSpace(v), "true")
	}
	omitClaudeMd := false
	switch v := fm["omitClaudeMd"].(type) {
	case bool:
		omitClaudeMd = v
	case string:
		omitClaudeMd = strings.EqualFold(strings.TrimSpace(v), "true")
	}
	iso, _ := fm["isolation"].(string)
	iso = strings.TrimSpace(iso)
	if iso != "" && iso != "worktree" && iso != "remote" {
		return AgentDefinition{}, false, `invalid "isolation" value`
	}
	tools := parseToolList(fm["tools"])
	disallowed := parseToolList(fm["disallowedTools"])
	skills := parseToolList(fm["skills"])
	requiredMcp := parseToolList(fm["requiredMcpServers"])
	permMode, _ := fm["permissionMode"].(string)
	permMode = strings.TrimSpace(permMode)
	maxTurns := parsePositiveInt(fm["maxTurns"])
	systemPrompt, _ := fm["systemPrompt"].(string)
	systemPrompt = strings.TrimSpace(systemPrompt)
	criticalReminder, _ := fm["criticalSystemReminder_EXPERIMENTAL"].(string)
	criticalReminder = strings.TrimSpace(criticalReminder)
	hooks := parseHooksFromFrontmatter(fm)

	return AgentDefinition{
		AgentType:                          name,
		WhenToUse:                          desc,
		Tools:                              tools,
		DisallowedTools:                    disallowed,
		Skills:                             skills,
		Source:                             source,
		Model:                              model,
		PermissionMode:                     permMode,
		MaxTurns:                           maxTurns,
		Background:                         background,
		OmitClaudeMd:                       omitClaudeMd,
		Isolation:                          iso,
		RequiredMcpServers:                 requiredMcp,
		SystemPrompt:                       systemPrompt,
		Hooks:                             hooks,
	}, true, ""
}

func parseHooksFromFrontmatter(fm map[string]interface{}) json.RawMessage {
	v, ok := fm["hooks"]
	if !ok || v == nil {
		return nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	// Validate that top-level keys are known hook events (mirrors TS HooksSchema).safeParse).
	var raw map[string]interface{}
	if err := json.Unmarshal(b, &raw); err != nil {
		return nil
	}
	for key := range raw {
		if !hookstypes.KnownHookEvent(key) {
			return nil
		}
	}
	return json.RawMessage(b)
}

func parseToolList(v any) []string {
	switch t := v.(type) {
	case nil:
		return nil
	case string:
		s := strings.TrimSpace(t)
		if s == "" {
			return nil
		}
		if strings.Contains(s, ",") {
			var out []string
			for _, p := range strings.Split(s, ",") {
				p = strings.TrimSpace(p)
				if p != "" {
					out = append(out, p)
				}
			}
			return out
		}
		return strings.Fields(s)
	case []string:
		var out []string
		for _, p := range t {
			p = strings.TrimSpace(p)
			if p != "" {
				out = append(out, p)
			}
		}
		return out
	case []any:
		var out []string
		for _, x := range t {
			if s, ok := x.(string); ok {
				s = strings.TrimSpace(s)
				if s != "" {
					out = append(out, s)
				}
			}
		}
		return out
	default:
		return nil
	}
}

func parsePositiveInt(v any) int {
	switch t := v.(type) {
	case int:
		if t > 0 {
			return t
		}
	case int64:
		if t > 0 {
			return int(t)
		}
	case float64:
		i := int(t)
		if float64(i) == t && i > 0 {
			return i
		}
	case string:
		s := strings.TrimSpace(t)
		if s == "" {
			return 0
		}
		i, err := strconv.Atoi(s)
		if err == nil && i > 0 {
			return i
		}
	}
	return 0
}
