package commands

import (
	"context"
	"fmt"
	"sync"

	"goc/commands/featuregates"
	"goc/types"
	"path/filepath"
	"sort"
	"strings"
)

// LoadOptions configures LoadAllCommands. Zero value is safe: mirrors conservative defaults where noted.
// Parity: src/commands.ts loadAllCommands (Promise.all of getSkills, getPluginCommands, getWorkflowCommands) + getSkillDirCommands gates.
type LoadOptions struct {
	// WorkflowScripts enables workflow command listing (Phase P6). Product path: **deferred** — default [DefaultLoadOptions] keeps this false; set true to use [loadWorkflowCommands]. TS historically gated WORKFLOW_SCRIPTS.
	WorkflowScripts bool
	// BareMode when non-nil overrides IsBareMode() env (CLAUDE_CODE_BARE).
	BareMode *bool
	// DisablePolicySkills skips managed policy skills directory (Phase P2 full parity with CLAUDE_CODE_DISABLE_POLICY_SKILLS).
	DisablePolicySkills bool
	// SkillsPluginOnlyLocked mirrors isRestrictedToPluginOnly('skills'): skips user/project/add-dir skills and legacy /commands; managed policy skills dir still loads when policy is not disabled.
	SkillsPluginOnlyLocked bool
	// AddSkillDirs are extra project roots (TS --add-dir); each contributes <root>/.claude/skills.
	AddSkillDirs []string
	// SessionProjectRoot is TS getProjectRoot() for resolveStopBoundary (nested repo / worktree). Empty defaults to cwd when resolving.
	SessionProjectRoot string
	// EnabledSettingSources lists allowed sources (TS getAllowedSettingSources); nil means default CLI (user, project, local). Empty slice = isolation (no user/project/local). policySettings/flagSettings are always enabled in TS getEnabledSettingSources — managed skills dir still obeys DisablePolicySkills / env only.
	EnabledSettingSources []string
	// IncludeConditionalSkills when true returns path-filtered skills; TS getSkillDirCommands returns only unconditional (default false).
	IncludeConditionalSkills bool
}

// DefaultLoadOptions returns LoadOptions for interactive Go TUI / local CLI: non-bare session and
// nil EnabledSettingSources (user + project + local per [LoadOptions.isSettingSourceEnabled], same as TS default allowed sources).
// WorkflowScripts is false (P6 workflow listing deferred — see docs/plans/goc-load-all-commands.md).
func DefaultLoadOptions() LoadOptions {
	f := false
	return LoadOptions{
		BareMode:        &f,
		WorkflowScripts: false,
	}
}

// WorkflowScriptsEnabled is true when FEATURE_WORKFLOW_SCRIPTS is set. Hosts may merge this into [LoadOptions] when explicitly enabling workflow listing; [DefaultLoadOptions] does not use it while P6 is deferred.
func WorkflowScriptsEnabled() bool {
	return IsEnvTruthy("FEATURE_WORKFLOW_SCRIPTS")
}

func (o LoadOptions) sessionRootForBoundary(cwd string) string {
	if s := strings.TrimSpace(o.SessionProjectRoot); s != "" {
		return s
	}
	return cwd
}

func (o LoadOptions) disablePolicySkillsEffective() bool {
	return o.DisablePolicySkills || DisablePolicySkillsEnv()
}

func (o LoadOptions) effectiveBare() bool {
	if o.BareMode != nil {
		return *o.BareMode
	}
	return IsBareMode()
}

func (o LoadOptions) cacheKey(cwd string) string {
	abs, err := filepath.Abs(cwd)
	if err != nil {
		abs = cwd
	}
	var b strings.Builder
	b.WriteString(abs)
	b.WriteByte(0)
	b.WriteString(fmt.Sprintf("wf=%v", o.WorkflowScripts))
	b.WriteByte(0)
	b.WriteString(fmt.Sprintf("bare=%v", o.effectiveBare()))
	b.WriteByte(0)
	b.WriteString(fmt.Sprintf("dp=%v", o.disablePolicySkillsEffective()))
	b.WriteByte(0)
	b.WriteString(fmt.Sprintf("lock=%v", o.SkillsPluginOnlyLocked))
	b.WriteByte(0)
	if len(o.AddSkillDirs) > 0 {
		cp := append([]string(nil), o.AddSkillDirs...)
		sort.Strings(cp)
		b.WriteString(strings.Join(cp, "\x1e"))
	}
	b.WriteByte(0)
	b.WriteString(strings.TrimSpace(o.SessionProjectRoot))
	b.WriteByte(0)
	b.WriteString(fmt.Sprintf("icc=%v", o.IncludeConditionalSkills))
	b.WriteByte(0)
	if o.EnabledSettingSources == nil {
		b.WriteString("ess=nil")
	} else {
		cp := append([]string(nil), o.EnabledSettingSources...)
		sort.Strings(cp)
		b.WriteString(strings.Join(cp, ","))
	}
	b.WriteByte(0)
	b.WriteString(featuregates.GatesFingerprint())
	return b.String()
}

var (
	loadAllMu    sync.Mutex
	loadAllCache map[string][]types.Command
)

// ClearCommandMemoizationCaches mirrors src/commands.ts clearCommandMemoizationCaches: invalidates
// [loadAllCommands] memo only (does not clear dynamic skill session state).
func ClearCommandMemoizationCaches() {
	loadAllMu.Lock()
	defer loadAllMu.Unlock()
	loadAllCache = nil
}

// ClearLoadAllCommandsCache drops cwd/options memoization, builtin name cache, and full dynamic skill
// session (tests and hard reset). For TS parity when only new dynamics load, use [ClearCommandMemoizationCaches].
func ClearLoadAllCommandsCache() {
	ClearCommandMemoizationCaches()
	clearBuiltinNameSetCache()
	ClearDynamicSkills()
}

// LoadAllCommandsAsyncResult is the Promise-like outcome of [LoadAllCommandsAsync].
type LoadAllCommandsAsyncResult struct {
	Commands []types.Command
	Err      error
}

// loadAllCommandsBody mirrors TS loadAllCommands after Promise.all resolves: concat in order.
// getSkills runs as [GetSkillsAsync] in parallel with getPluginCommands and getWorkflowCommands.
func loadAllCommandsBody(ctx context.Context, cwd string, opts LoadOptions) ([]types.Command, loadAllCounts, error) {
	skillsCh := GetSkillsAsync(ctx, cwd, opts)
	var workflow []types.Command
	var pluginCmd []types.Command
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		pluginCmd = loadPluginCommands()
		if pluginCmd == nil {
			pluginCmd = make([]types.Command, 0)
		}
	}()
	go func() {
		defer wg.Done()
		workflow = loadWorkflowCommands(cwd, opts)
		if workflow == nil {
			workflow = make([]types.Command, 0)
		}
	}()
	wg.Wait()

	skillsRes := <-skillsCh
	if skillsRes.Err != nil {
		return nil, loadAllCounts{}, skillsRes.Err
	}
	skillsBatch := skillsRes.Batch

	bundled := skillsBatch.BundledSkills
	builtinPlugin := skillsBatch.BuiltinPluginSkills
	skillDir := skillsBatch.SkillDirCommands
	pluginSkills := skillsBatch.PluginSkills
	builtins := loadBuiltinCommands()

	counts := loadAllCounts{
		Bundled:       len(bundled),
		BuiltinPlugin: len(builtinPlugin),
		SkillDir:      len(skillDir),
		Workflow:      len(workflow),
		PluginCmd:     len(pluginCmd),
		PluginSkills:  len(pluginSkills),
		Builtins:      len(builtins),
	}

	out := make([]types.Command, 0, len(bundled)+len(builtinPlugin)+len(skillDir)+len(workflow)+len(pluginCmd)+len(pluginSkills)+len(builtins))
	out = append(out, bundled...)
	out = append(out, builtinPlugin...)
	out = append(out, skillDir...)
	out = append(out, workflow...)
	out = append(out, pluginCmd...)
	out = append(out, pluginSkills...)
	out = append(out, builtins...)

	return out, counts, nil
}

func loadAllCommandsResolve(ctx context.Context, cwd string, opts LoadOptions) ([]types.Command, error) {
	key := opts.cacheKey(cwd)
	loadAllMu.Lock()
	if loadAllCache == nil {
		loadAllCache = make(map[string][]types.Command)
	}
	if v, ok := loadAllCache[key]; ok {
		out := append([]types.Command(nil), v...)
		loadAllMu.Unlock()
		logLoadAllCommands(cwd, true, loadAllCounts{}, len(out))
		return out, nil
	}
	loadAllMu.Unlock()

	out, counts, err := loadAllCommandsBody(ctx, cwd, opts)
	if err != nil {
		return nil, err
	}

	logLoadAllCommands(cwd, false, counts, len(out))

	copyOut := append([]types.Command(nil), out...)
	loadAllMu.Lock()
	if loadAllCache == nil {
		loadAllCache = make(map[string][]types.Command)
	}
	loadAllCache[key] = append([]types.Command(nil), copyOut...)
	loadAllMu.Unlock()

	return out, nil
}

// LoadAllCommands mirrors src/commands.ts loadAllCommands concat order:
//
//	bundled → builtin-plugin → skillDir → workflow → pluginCommands → pluginSkills → COMMANDS()
//
// Filtering (meetsAvailabilityRequirement, isEnabled) is not applied here — same as TS.
// Internally uses the same async-shaped loading as [LoadAllCommandsAsync]; this call blocks until complete.
func LoadAllCommands(ctx context.Context, cwd string, opts LoadOptions) ([]types.Command, error) {
	return loadAllCommandsResolve(ctx, cwd, opts)
}

// LoadAllCommandsAsync runs [loadAllCommandsResolve] in a goroutine and sends one [LoadAllCommandsAsyncResult] (TS async loadAllCommands / Promise).
func LoadAllCommandsAsync(ctx context.Context, cwd string, opts LoadOptions) <-chan LoadAllCommandsAsyncResult {
	ch := make(chan LoadAllCommandsAsyncResult, 1)
	go func() {
		defer close(ch)
		cmds, err := loadAllCommandsResolve(ctx, cwd, opts)
		ch <- LoadAllCommandsAsyncResult{Commands: cmds, Err: err}
	}()
	return ch
}

// --- Phase P3: bundled skills — handwritten.AssembleBundledSkills, see handwritten/bundled_*.go ---

// --- Phase P4: builtin plugin skills — [BuiltinPluginSkillCommands] / getBuiltinPlugins parity ---

// --- Phase P6: workflows — see workflow_load.go (TS createWorkflowCommand / WORKFLOW_SCRIPTS stub; Go lists YAML/JSON on disk) ---

// --- Phase P5: plugin marketplace (src/utils/plugins/loadPluginCommands.ts) ---

func loadPluginCommands() []types.Command {
	return nil
}

func loadPluginSkills() []types.Command {
	return nil
}

// --- Phase P7: COMMANDS() (src/commands.ts) — handwritten.AssembleBuiltinCommands + z_builtin_table_gen.go ---
