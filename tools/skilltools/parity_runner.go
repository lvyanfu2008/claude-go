package skilltools

import (
	"context"
	"encoding/json"
	"path/filepath"
	"sort"
	"strings"

	"goc/ccb-engine/bashzog"
	"goc/tools/localtools"
	"goc/tools"
	"goc/types"
)

// ParityToolRunner runs core filesystem/search tools in Go, then delegates Skill (disk + embedded bundled)
// to [DemoToolRunner]; other names fall through to stub via Demo.
type ParityToolRunner struct {
	DemoToolRunner
	WorkDir    string
	ExtraRoots []string
	// ProjectRoot is the repo / project directory (for .claude paths: todos, cron, plan mode, task files).
	ProjectRoot string
	// ReadFileState mirrors TS toolUseContext.readFileState (nil → lazy per runner; gou-demo sets a session-scoped pointer on model).
	ReadFileState *localtools.ReadFileState
	UserModified  bool
	// AskAutoFirst makes AskUserQuestion pick the first option per question (gou-demo default).
	AskAutoFirst bool
	// LocalBashDefault enables Bash by default (gou-demo aligns with TS; opt out via GOU_DEMO_NO_LOCAL_BASH).
	LocalBashDefault bool
	// MainLoopModel is optional; when set it drives Read tool_result cyber-risk mitigation (TS shouldIncludeFileReadMitigation).
	MainLoopModel string
	// Messages is a snapshot of conversation messages at runner construction time.
	// For the fork subagent Agent tool, use MessagesFunc for the latest messages.
	Messages []types.Message
	// SystemPrompt is the parent's rendered system prompt parts, forwarded to Agent tool
	// for the fork subagent path (cache-identical API prefixes).
	SystemPrompt []string
	// MessagesFunc returns the current conversation messages at tool dispatch time.
	// Used by the Agent tool (fork subagent) to access the in-progress assistant message.
	// Takes precedence over Messages when set.
	MessagesFunc func() []types.Message
	// ProgressCallback forwards agent progress messages in real time to the UI.
	// Set by gou-demo to forward progress from inner agent query loops via ccbSend.
	ProgressCallback func(*types.Message)
	// WriteDeps holds optional callbacks for Write tool parity features.
	// When nil or individual callbacks are nil, the corresponding TS feature is skipped.
	WriteDeps *localtools.WriteDeps
	// EditDeps holds optional callbacks for Edit tool parity features.
	// When nil or individual callbacks are nil, the corresponding TS feature is skipped.
	EditDeps *localtools.EditDeps
	// ToolPermission is the parent's permission context propagated to child agents
	// for bubble-mode permission enforcement (TS PermissionUpdate parity).
	ToolPermission *types.ToolPermissionContextData
}

func (r *ParityToolRunner) roots() []string {
	m := map[string]struct{}{}
	add := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" {
			return
		}
		a, err := filepath.Abs(s)
		if err != nil {
			return
		}
		m[a] = struct{}{}
	}
	add(r.WorkDir)
	for _, e := range r.ExtraRoots {
		add(e)
	}
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	if len(out) == 0 {
		if wd, err := filepath.Abs("."); err == nil {
			out = []string{wd}
		}
	}
	return out
}

// Run implements [toolstub.ToolRunner].
func (r *ParityToolRunner) Run(ctx context.Context, name, toolUseID string, input json.RawMessage) (string, bool, error) {
	if name == "REPL" {
		return r.runREPLTool(ctx, toolUseID, input)
	}
	return r.dispatchTool(ctx, name, toolUseID, input)
}

func (r *ParityToolRunner) dispatchTool(ctx context.Context, name, toolUseID string, input json.RawMessage) (string, bool, error) {
	roots := r.roots()
	wd := strings.TrimSpace(r.WorkDir)
	if wd == "" && len(roots) > 0 {
		wd = roots[0]
	}
	pr := strings.TrimSpace(r.ProjectRoot)
	if pr == "" && len(roots) > 0 {
		pr = roots[0]
	}
	msgs := r.Messages
	if r.MessagesFunc != nil {
		msgs = r.MessagesFunc()
	}
	cfg := tools.Config{
		Roots:             roots,
		WorkDir:           wd,
		ProjectRoot:       pr,
		SessionID:         strings.TrimSpace(r.SessionID),
		AskAutoFirst:      r.AskAutoFirst,
		Messages:          msgs,
		SystemPrompt:      r.SystemPrompt,
		MainLoopModel:     r.MainLoopModel,
		ProgressCallback:  r.ProgressCallback,
		ToolPermission:    r.ToolPermission,
	}
	s, isErr, perr := tools.Run(ctx, name, input, cfg)
	if perr == nil || !tools.IsNotHandled(perr) {
		return s, isErr, perr
	}
	if r.ReadFileState == nil {
		r.ReadFileState = localtools.NewReadFileState()
	}
	st := r.ReadFileState
	switch name {
	case "Read":
		// Return raw tool output JSON (TS tool.call `data`). toolexecution maps to tool_result.content
		// while embedding this string as structured toolUseResult (see syntheticToolMessageAfterInvoke).
		return localtools.ReadFromJSON(input, roots, st, nil)
	case "Write":
		return localtools.WriteFromJSONDeps(input, roots, st, r.WriteDeps)
	case "Edit":
		return localtools.EditFromJSONDeps(input, roots, st, r.UserModified, r.EditDeps)
	case "Glob":
		return localtools.GlobFromJSON(ctx, input, roots)
	case "Grep":
		return localtools.GrepFromJSON(ctx, input, roots)
	case "Bash", bashzog.ZogToolName:
		tasksDir := computeTasksDir(pr, r.SessionID)
		return localtools.BashFromJSON(ctx, input, wd, r.LocalBashDefault, tasksDir)
	}
	if dn := DiscoverSkillsToolNameFromEnv(); dn != "" && name == dn {
		return discoverSkillsFromJSON(input, r.Commands)
	}
	return r.DemoToolRunner.Run(ctx, name, toolUseID, input)
}

// ToolReadMappingRoots supplies absolute roots for Read tool_result mapping.
func (r *ParityToolRunner) ToolReadMappingRoots() []string {
	return r.roots()
}

// ToolReadMappingMemCWD supplies cwd for auto-memory freshness in Read formatter.
func (r *ParityToolRunner) ToolReadMappingMemCWD() string {
	wd := strings.TrimSpace(r.WorkDir)
	if wd != "" {
		return wd
	}
	rs := r.roots()
	if len(rs) > 0 {
		return rs[0]
	}
	return ""
}

// discoverSkillsFromJSON lists loaded prompt-type commands (skills) with name, description, and
// argument hint, optionally filtered by the input description field. Returns JSON matching TS
// DiscoverSkills tool output shape.
func discoverSkillsFromJSON(raw json.RawMessage, commands []types.Command) (string, bool, error) {
	var in struct {
		Description string `json:"description"`
	}
	_ = json.Unmarshal(raw, &in)
	query := strings.ToLower(strings.TrimSpace(in.Description))

	type skillEntry struct {
		Name         string `json:"name"`
		Description  string `json:"description"`
		ArgumentHint string `json:"argumentHint,omitempty"`
	}
	var skills []skillEntry
	for _, cmd := range commands {
		if cmd.Type != "prompt" {
			continue
		}
		if cmd.DisableModelInvocation != nil && *cmd.DisableModelInvocation {
			continue
		}
		name := strings.TrimSpace(cmd.Name)
		if name == "" {
			continue
		}
		desc := strings.TrimSpace(cmd.Description)
		argHint := ""
		if cmd.ArgumentHint != nil {
			argHint = strings.TrimSpace(*cmd.ArgumentHint)
		}
		if query != "" {
			if !strings.Contains(strings.ToLower(name), query) &&
				!strings.Contains(strings.ToLower(desc), query) &&
				!strings.Contains(strings.ToLower(argHint), query) {
				continue
			}
		}
		skills = append(skills, skillEntry{
			Name:         name,
			Description:  desc,
			ArgumentHint: argHint,
		})
	}
	if skills == nil {
		skills = make([]skillEntry, 0)
	}
	out := map[string]any{
		"data": map[string]any{
			"skills": skills,
			"count":  len(skills),
		},
	}
	b, _ := json.Marshal(out)
	return string(b), false, nil
}

// computeTasksDir derives the background-task directory from project root and session ID.
func computeTasksDir(projectRoot, sessionID string) string {
	pr := strings.TrimSpace(projectRoot)
	if pr == "" {
		return ""
	}
	sid := strings.TrimSpace(sessionID)
	if sid == "" {
		sid = "default-session"
	}
	return filepath.Join(pr, ".claude", ".gou-tasks", sid, "tasks")
}
