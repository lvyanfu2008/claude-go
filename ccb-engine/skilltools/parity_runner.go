package skilltools

import (
	"context"
	"encoding/json"
	"path/filepath"
	"sort"
	"strings"

	"goc/ccb-engine/bashzog"
	"goc/ccb-engine/localtools"
	"goc/ccb-engine/paritytools"
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
	// LocalBashDefault enables Bash without CCB_ENGINE_LOCAL_BASH (gou-demo aligns with TS; opt out via GOU_DEMO_NO_LOCAL_BASH).
	LocalBashDefault bool
	// MainLoopModel is optional; when set it drives Read tool_result cyber-risk mitigation (TS shouldIncludeFileReadMitigation).
	MainLoopModel string
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

// Run implements [engine.ToolRunner].
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
	cfg := paritytools.Config{
		Roots:        roots,
		WorkDir:      wd,
		ProjectRoot:  pr,
		SessionID:    strings.TrimSpace(r.SessionID),
		AskAutoFirst: r.AskAutoFirst,
	}
	s, isErr, perr := paritytools.Run(ctx, name, input, cfg)
	if perr == nil || !paritytools.IsNotHandled(perr) {
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
		return localtools.WriteFromJSON(input, roots, st)
	case "Edit":
		return localtools.EditFromJSON(input, roots, st, r.UserModified)
	case "Glob":
		return localtools.GlobFromJSON(ctx, input, roots)
	case "Grep":
		return localtools.GrepFromJSON(ctx, input, roots)
	case "Bash", bashzog.ZogToolName:
		return localtools.BashFromJSON(ctx, input, wd, r.LocalBashDefault)
	}
	if dn := DiscoverSkillsToolNameFromEnv(); dn != "" && name == dn {
		return `{"note":"Go local runner: discover-skills is not implemented; use the Skill tool with a skill name, or enable the TS socket worker for full tool parity."}`, false, nil
	}
	return r.DemoToolRunner.Run(ctx, name, toolUseID, input)
}

// ToolReadMappingRoots supplies absolute roots for Read tool_result mapping (engine.Session).
func (r *ParityToolRunner) ToolReadMappingRoots() []string {
	return r.roots()
}

// ToolReadMappingMemCWD supplies cwd for auto-memory freshness in Read formatter (engine.Session).
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
