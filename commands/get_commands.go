package commands

import (
	"context"
	"sync"

	"goc/commands/featuregates"
	"goc/types"
)

// GetCommandsAuth carries flags for TS getCommands() filtering (meetsAvailabilityRequirement).
// Derive from the same sources as src/utils/auth.ts isClaudeAISubscriber / isUsing3PServices
// and src/utils/model/providers.ts isFirstPartyAnthropicBaseUrl when wiring gou or other hosts.
//
// IsRemoteMode mirrors src/bootstrap/state getIsRemoteMode() for commands gated at list time
// (e.g. /session — src/commands/session/index.ts isEnabled).
//
// IsNonInteractiveSession mirrors getIsNonInteractiveSession() for /context and /extra-usage pairs.
// ExtraUsageAllowed mirrors isOverageProvisioningAllowed() when the host cannot compute billing (default false in [DefaultConsoleAPIAuth]).
// IsConsumerSubscriber mirrors isConsumerSubscriber() for /privacy-settings.
// BlockRemoteSessions when true matches !isPolicyAllowed('allow_remote_sessions') for /remote-env.
// DenyProductFeedback when true matches !isPolicyAllowed('allow_product_feedback') for /feedback.
type GetCommandsAuth struct {
	IsClaudeAISubscriber         bool
	IsUsing3PServices            bool
	IsFirstPartyAnthropicBaseURL bool
	IsRemoteMode                 bool
	IsNonInteractiveSession      bool
	ExtraUsageAllowed            bool
	IsConsumerSubscriber         bool
	BlockRemoteSessions          bool
	DenyProductFeedback          bool
}

// FilterGetCommands mirrors src/commands.ts getCommands filtering on the loaded list:
//
//	allCommands.filter(c => meetsAvailabilityRequirement(c) && isCommandEnabled(c))
func FilterGetCommands(all []types.Command, auth GetCommandsAuth) []types.Command {
	out := make([]types.Command, 0, len(all))
	for _, cmd := range all {
		if !MeetsAvailabilityRequirement(cmd, auth.IsClaudeAISubscriber, auth.IsUsing3PServices, auth.IsFirstPartyAnthropicBaseURL) {
			continue
		}
		if !IsCommandEnabledData(cmd, auth) {
			continue
		}
		out = append(out, cmd)
	}
	return out
}

// LoadAndFilterCommands runs [LoadAllCommandsAsync] then FilterGetCommands (TS getCommands: loadAllCommands + filter only, no dynamic merge).
func LoadAndFilterCommands(ctx context.Context, cwd string, opts LoadOptions, auth GetCommandsAuth) ([]types.Command, error) {
	res := <-LoadAllCommandsAsync(ctx, cwd, opts)
	if res.Err != nil {
		return nil, res.Err
	}
	all := res.Commands
	out := FilterGetCommands(all, auth)
	logLoadAndFilterCommands(len(all), len(out))
	return out, nil
}

// GetCommands mirrors src/commands.ts export async function getCommands(cwd): loadAllCommands → filter →
// getDynamicSkills → dedupe → insert before first builtin.
func GetCommands(ctx context.Context, cwd string, opts LoadOptions, auth GetCommandsAuth) ([]types.Command, error) {
	base, err := LoadAndFilterCommands(ctx, cwd, opts, auth)
	if err != nil {
		return nil, err
	}
	return GetCommandsWithDynamicSkills(base, GetDynamicSkills(), auth), nil
}

// GetCommandsWithDefaults matches the TS call shape getCommands(cwd) when the host’s implicit runtime
// matches [DefaultLoadOptions] and [DefaultConsoleAPIAuth]: direct 1P API, non-remote, interactive, etc.
// TS still reads auth and isEnabled gates from global bootstrap state; any deviation (claude.ai subscriber,
// Bedrock/Vertex, remote mode, policy blocks, overage billing) requires the full [GetCommands] signature
// with an explicit [GetCommandsAuth] and, if needed, non-default [LoadOptions].
func GetCommandsWithDefaults(ctx context.Context, cwd string) ([]types.Command, error) {
	return GetCommands(ctx, cwd, DefaultLoadOptions(), DefaultConsoleAPIAuth())
}

// Built-in name caches invalidate when [featuregates.GatesFingerprint] changes.
var (
	builtinNameSetCache        sync.Map // fingerprint -> map[string]struct{} (name + aliases)
	builtinPrimaryNameSetCache sync.Map // fingerprint -> map[string]struct{} (primary names only)
)

func clearBuiltinNameSetCache() {
	builtinNameSetCache = sync.Map{}
	builtinPrimaryNameSetCache = sync.Map{}
}

// BuiltinCommandPrimaryNameSet returns only primary names from the handwritten COMMANDS() assembly.
// Matches src/commands.ts getCommands: builtInNames = new Set(COMMANDS().map(c => c.name)) (aliases excluded).
func BuiltinCommandPrimaryNameSet() map[string]struct{} {
	fp := featuregates.GatesFingerprint()
	if v, ok := builtinPrimaryNameSetCache.Load(fp); ok {
		return v.(map[string]struct{})
	}
	cmds := loadBuiltinCommands()
	m := make(map[string]struct{}, len(cmds))
	for _, c := range cmds {
		if c.Name != "" {
			m[c.Name] = struct{}{}
		}
	}
	builtinPrimaryNameSetCache.Store(fp, m)
	return m
}

// BuiltinCommandNameSet returns names and aliases (TS builtInCommandNames memo: flatMap name + aliases).
func BuiltinCommandNameSet() map[string]struct{} {
	fp := featuregates.GatesFingerprint()
	if v, ok := builtinNameSetCache.Load(fp); ok {
		return v.(map[string]struct{})
	}
	cmds := loadBuiltinCommands()
	m := make(map[string]struct{}, len(cmds)*2)
	for _, c := range cmds {
		if c.Name != "" {
			m[c.Name] = struct{}{}
		}
		for _, a := range c.Aliases {
			if a != "" {
				m[a] = struct{}{}
			}
		}
	}
	builtinNameSetCache.Store(fp, m)
	return m
}

// UniqueDynamicSkillsForGetCommands mirrors the TS getCommands branch that builds uniqueDynamicSkills:
// dynamic entries not already in base, and passing meetsAvailabilityRequirement + isCommandEnabled.
func UniqueDynamicSkillsForGetCommands(dynamic []types.Command, base []types.Command, auth GetCommandsAuth) []types.Command {
	if len(dynamic) == 0 {
		return nil
	}
	baseNames := make(map[string]struct{}, len(base))
	for _, c := range base {
		baseNames[c.Name] = struct{}{}
	}
	var out []types.Command
	for _, s := range dynamic {
		if _, dup := baseNames[s.Name]; dup {
			continue
		}
		if !MeetsAvailabilityRequirement(s, auth.IsClaudeAISubscriber, auth.IsUsing3PServices, auth.IsFirstPartyAnthropicBaseURL) {
			continue
		}
		if !IsCommandEnabledData(s, auth) {
			continue
		}
		out = append(out, s)
	}
	return out
}

// indexFirstBuiltinCommandInLoadAllOrder returns the index in base of the first slot that belongs to
// the COMMANDS() tail of [LoadAllCommands] (last len(loadBuiltinCommands()) entries), not earlier copies
// of the same name (e.g. "update-config" appears in bundled skills and in builtins).
func indexFirstBuiltinCommandInLoadAllOrder(base []types.Command, builtinNames map[string]struct{}) int {
	builtins := loadBuiltinCommands()
	if len(builtins) == 0 {
		return len(base)
	}
	tailStart := len(base) - len(builtins)
	if tailStart < 0 {
		tailStart = 0
	}
	for i := tailStart; i < len(base); i++ {
		if _, ok := builtinNames[base[i].Name]; ok {
			return i
		}
	}
	for i, c := range base {
		if _, ok := builtinNames[c.Name]; ok {
			return i
		}
	}
	return -1
}

// InsertDynamicSkillsBeforeBuiltins inserts uniqueDynamic immediately before the first command whose
// primary name appears in builtinNames (TS getCommands: builtInNames from COMMANDS().map(c => c.name) only).
// If no such command exists, appends at the end (TS: [...base, ...unique]).
func InsertDynamicSkillsBeforeBuiltins(base []types.Command, uniqueDynamic []types.Command, builtinNames map[string]struct{}) []types.Command {
	if len(uniqueDynamic) == 0 {
		return base
	}
	insert := indexFirstBuiltinCommandInLoadAllOrder(base, builtinNames)
	if insert < 0 {
		out := make([]types.Command, 0, len(base)+len(uniqueDynamic))
		out = append(out, base...)
		out = append(out, uniqueDynamic...)
		return out
	}
	out := make([]types.Command, 0, len(base)+len(uniqueDynamic))
	out = append(out, base[:insert]...)
	out = append(out, uniqueDynamic...)
	out = append(out, base[insert:]...)
	return out
}

// GetCommandsWithDynamicSkills is TS getCommands(baseFiltered + dynamic) in one step after base is already filtered.
func GetCommandsWithDynamicSkills(baseFiltered []types.Command, dynamic []types.Command, auth GetCommandsAuth) []types.Command {
	uniq := UniqueDynamicSkillsForGetCommands(dynamic, baseFiltered, auth)
	if len(uniq) == 0 {
		return baseFiltered
	}
	return InsertDynamicSkillsBeforeBuiltins(baseFiltered, uniq, BuiltinCommandPrimaryNameSet())
}

// LoadAndGetCommandsWithDynamic runs LoadAndFilterCommands then GetCommandsWithDynamicSkills.
func LoadAndGetCommandsWithDynamic(ctx context.Context, cwd string, opts LoadOptions, auth GetCommandsAuth, dynamic []types.Command) ([]types.Command, error) {
	base, err := LoadAndFilterCommands(ctx, cwd, opts, auth)
	if err != nil {
		return nil, err
	}
	return GetCommandsWithDynamicSkills(base, dynamic, auth), nil
}

// DefaultConsoleAPIAuth assumes a direct 1P Anthropic API user (not claude.ai subscriber, not Bedrock/Vertex).
// IsRemoteMode is false, so /session is omitted from getCommands-style lists (matches TS when getIsRemoteMode() is false).
// Useful for local dev slash lists when real auth is unknown; production should pass measured flags.
func DefaultConsoleAPIAuth() GetCommandsAuth {
	return GetCommandsAuth{
		IsClaudeAISubscriber:         false,
		IsUsing3PServices:            false,
		IsFirstPartyAnthropicBaseURL: true,
		IsRemoteMode:                 false,
		IsNonInteractiveSession:      false,
		ExtraUsageAllowed:            false,
		IsConsumerSubscriber:         false,
		BlockRemoteSessions:          false,
		DenyProductFeedback:          false,
	}
}
