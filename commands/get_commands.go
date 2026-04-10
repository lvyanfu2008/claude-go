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

// LoadAndFilterCommands runs LoadAllCommands then FilterGetCommands (TS getCommands disk + filter path).
func LoadAndFilterCommands(ctx context.Context, cwd string, opts LoadOptions, auth GetCommandsAuth) ([]types.Command, error) {
	all, err := LoadAllCommands(ctx, cwd, opts)
	if err != nil {
		return nil, err
	}
	out := FilterGetCommands(all, auth)
	logLoadAndFilterCommands(len(all), len(out))
	return out, nil
}

// BuiltinCommandNameSet returns names and aliases from the handwritten COMMANDS() assembly
// (TS: builtInCommandNames memo uses flatMap name + aliases). Invalidates when [featuregates.GatesFingerprint] changes.
var builtinNameSetCache sync.Map // string (fingerprint) -> map[string]struct{}

func clearBuiltinNameSetCache() {
	builtinNameSetCache = sync.Map{}
}

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
// name appears in builtinNames (TS: insert after plugin skills, before COMMANDS() block).
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
	return InsertDynamicSkillsBeforeBuiltins(baseFiltered, uniq, BuiltinCommandNameSet())
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
