package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"goc/agents/builtin"
	"goc/claudemd"
	"goc/commands"
	"goc/memdir"
	"goc/tools/agentcolor"
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

	// Load agents from plugins (mirrors TS loadPluginAgents).
	// Plugin agents take priority over builtins but not over user/project/policy/flag agents.
	pluginAgents := loadPluginAgents()
	all = append(all, pluginAgents...)

	loadOrder := []struct {
		source string
		dir    string
	}{
		{source: "userSettings", dir: filepath.Join(commands.ClaudeConfigHome(), "agents")},
		{source: "projectSettings", dir: filepath.Join(strings.TrimSpace(cwd), ".harness", "agents")},
		{source: "policySettings", dir: filepath.Join(commands.ManagedFilePath(), ".harness", "agents")},
	}

	for _, item := range loadOrder {
		entries, ferrs := loadAgentMarkdownDir(item.dir, item.source)
		all = append(all, entries...)
		failed = append(failed, ferrs...)
	}

	// Load agents from --agents CLI flag (CLAUDE_CODE_AGENTS_JSON env var).
	// TS-v2 parses this as parseAgentsFromJson(parsed, 'flagSettings') and
	// merges into allAgents. Flag agents take highest priority in dedup.
	jsonAgents, jsonFailed := loadAgentsFromJSONEnv()
	all = append(all, jsonAgents...)
	failed = append(failed, jsonFailed...)

	active := dedupeAgentsByTypeOrder(all)

	// Initialize agent colors from active agent definitions (mirrors TS loadAgentsDir.ts).
	colorSetters := make([]agentcolor.AgentColorSetter, 0, len(active))
	for _, a := range active {
		if a.Color != "" {
			colorSetters = append(colorSetters, agentcolor.AgentColorSetter{
				AgentType: a.AgentType,
				Color:     a.Color,
			})
		}
	}
	agentcolor.InitAgentColors(colorSetters)

	// Initialize agent memory snapshots for custom agents with user memory scope
	// (mirrors TS loadAgentsDir.ts initializeAgentMemorySnapshots).
	if memdir.IsAutoMemoryEnabled() {
		initializeAgentMemorySnapshots(active)
	}

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
			Color:                              b.Color,
			PermissionMode:                     b.PermissionMode,
			Background:                         b.Background,
			SystemPrompt:                       b.SystemPrompt,
			OmitClaudeMd:                       b.OmitClaudeMd,
			Hooks:                              b.Hooks,
			SystemPromptFn: func(params AgentSystemPromptParams) string {
				if b.SystemPromptFn != nil {
					return b.SystemPromptFn(builtin.SystemPromptParams{
						SessionID:           params.SessionID,
						MemoryContent:       params.MemoryContent,
						TeamMembers:         params.TeamMembers,
						AvailableMCPServers: params.AvailableMCPServers,
						Model:               params.Model,
						SkillsContent:       params.SkillsContent,
						WorkDir:             params.WorkDir,
					})
				}
				return ""
			},
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
	background := false
	switch v := fm["background"].(type) {
	case bool:
		background = v
	case string:
		background = strings.EqualFold(strings.TrimSpace(v), "true")
	}
	color, _ := fm["color"].(string)
	color = strings.TrimSpace(color)
	if color != "" && !agentcolor.IsValidColorName(color) {
		return AgentDefinition{}, false, `invalid "color" value: must be one of: red, blue, green, yellow, purple, orange, pink, cyan`
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
	memory, _ := fm["memory"].(string)
	memory = strings.TrimSpace(memory)
	if memory != "" && memory != "user" && memory != "project" && memory != "local" {
		return AgentDefinition{}, false, `invalid "memory" value: must be "user", "project", or "local"`
	}
	hooks := parseHooksFromFrontmatter(fm)

	return AgentDefinition{
		AgentType:                          name,
		WhenToUse:                          desc,
		Tools:                              tools,
		DisallowedTools:                    disallowed,
		Skills:                             skills,
		Source:                             source,
		Model:                              model,
		Color:                             color,
		PermissionMode:                     permMode,
		MaxTurns:                           maxTurns,
		Background:                         background,
		OmitClaudeMd:                       omitClaudeMd,
		Isolation:                          iso,
		RequiredMcpServers:                 requiredMcp,
		SystemPrompt:                       systemPrompt,
		Memory:                            memory,
		Hooks:                             hooks,
	}, true, ""
}

// parseJSONRawField marshals a frontmatter value to json.RawMessage if present.
func parseJSONRawField(fm map[string]interface{}, key string) json.RawMessage {
	v, ok := fm[key]
	if !ok || v == nil {
		return nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	return json.RawMessage(b)
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

// loadAgentsFromJSONEnv reads CLAUDE_CODE_AGENTS_JSON (set by --agents CLI flag)
// and parses agents with source "flagSettings". Mirrors TS parseAgentsFromJson.
func loadAgentsFromJSONEnv() ([]AgentDefinition, []AgentLoadFailure) {
	raw := strings.TrimSpace(os.Getenv("CLAUDE_CODE_AGENTS_JSON"))
	if raw == "" {
		return nil, nil
	}
	var doc map[string]any
	if err := json.Unmarshal([]byte(raw), &doc); err != nil {
		return nil, []AgentLoadFailure{{Path: "CLAUDE_CODE_AGENTS_JSON", Error: fmt.Sprintf("invalid JSON: %v", err)}}
	}
	var agents []AgentDefinition
	var failed []AgentLoadFailure
	for name, def := range doc {
		a, ok := parseAgentJSON(name, def)
		if ok {
			agents = append(agents, a)
		} else {
			failed = append(failed, AgentLoadFailure{Path: fmt.Sprintf("CLAUDE_CODE_AGENTS_JSON[%s]", name), Error: "failed to parse agent JSON"})
		}
	}
	return agents, failed
}

// parseAgentJSON parses a single agent JSON definition. Mirrors TS parseAgentFromJson.
func parseAgentJSON(name string, v any) (AgentDefinition, bool) {
	def, ok := v.(map[string]any)
	if !ok {
		return AgentDefinition{}, false
	}

	desc, _ := def["description"].(string)
	desc = strings.TrimSpace(desc)
	prompt, _ := def["prompt"].(string)
	prompt = strings.TrimSpace(prompt)
	if desc == "" || prompt == "" {
		return AgentDefinition{}, false
	}

	model, _ := def["model"].(string)
	model = strings.TrimSpace(model)

	permMode, _ := def["permissionMode"].(string)
	permMode = strings.TrimSpace(permMode)

	maxTurns := parsePositiveInt(def["maxTurns"])

	bg := false
	switch v := def["background"].(type) {
	case bool:
		bg = v
	case string:
		bg = strings.EqualFold(strings.TrimSpace(v), "true")
	}

	memory, _ := def["memory"].(string)
	memory = strings.TrimSpace(memory)
	if memory != "" && memory != "user" && memory != "project" && memory != "local" {
		memory = ""
	}

	iso, _ := def["isolation"].(string)
	iso = strings.TrimSpace(iso)
	if iso != "" && iso != "worktree" && iso != "remote" {
		iso = ""
	}

	tools := parseToolList(def["tools"])
	disallowedTools := parseToolList(def["disallowedTools"])
	skills := parseToolList(def["skills"])

	effort, _ := def["effort"].(string)
	effort = strings.TrimSpace(effort)
	initialPrompt, _ := def["initialPrompt"].(string)
	initialPrompt = strings.TrimSpace(initialPrompt)

	var mcpServers json.RawMessage
	if ms, ok := def["mcpServers"]; ok && ms != nil {
		if b, err := json.Marshal(ms); err == nil {
			mcpServers = json.RawMessage(b)
		}
	}

	var hooks json.RawMessage
	if h, ok := def["hooks"]; ok && h != nil {
		b, err := json.Marshal(h)
		if err == nil {
			var raw map[string]any
			if json.Unmarshal(b, &raw) == nil {
				valid := true
				for key := range raw {
					if !hookstypes.KnownHookEvent(key) {
						valid = false
						break
					}
				}
				if valid {
					hooks = json.RawMessage(b)
				}
			}
		}
	}

	return AgentDefinition{
		AgentType:       name,
		WhenToUse:       desc,
		SystemPrompt:    prompt,
		Tools:           tools,
		DisallowedTools: disallowedTools,
		Skills:          skills,
		Source:          "flagSettings",
		Model:           model,
		PermissionMode:  permMode,
		MaxTurns:        maxTurns,
		Background:      bg,
		Isolation:       iso,
		McpServers:      mcpServers,
		Effort:          effort,
		InitialPrompt:   initialPrompt,
		Memory:          memory,
		Hooks:           hooks,
	}, true
}

// loadPluginAgents mirrors TS src/utils/plugins/loadPluginAgents.ts loadPluginAgents.
// Loads agent definitions from enabled plugins' agentsPath / agentsPaths directories.
// Returns nil until the plugin store/cache infrastructure is implemented (P4 dependency).
// TODO: Implement walkPluginMarkdown + loadAgentsFromDirectory for plugin agent .md files.
func loadPluginAgents() []AgentDefinition {
	return nil
}

// initializeAgentMemorySnapshots mirrors TS loadAgentsDir.ts initializeAgentMemorySnapshots.
// For custom agents with user memory scope, checks if a project snapshot exists and either
// initializes local memory from snapshot or marks a pending snapshot update.
func initializeAgentMemorySnapshots(active []AgentDefinition) {
	for i := range active {
		a := &active[i]
		if a.Memory != "user" || a.Source == "built-in" {
			continue
		}
		result := memdir.CheckAgentMemorySnapshot(a.AgentType, memdir.AgentMemoryUser)
		switch result.Action {
		case "initialize":
			_ = memdir.InitializeFromSnapshot(a.AgentType, memdir.AgentMemoryUser, result.SnapshotTimestamp)
		case "prompt-update":
			b, err := json.Marshal(map[string]string{
				"snapshotTimestamp": result.SnapshotTimestamp,
			})
			if err == nil {
				a.PendingSnapshotUpdate = json.RawMessage(b)
			}
		}
	}
}
