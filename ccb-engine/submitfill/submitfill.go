// Package submitfill optionally builds system prompt + user-context reminder when the HTTP/socket
// SubmitUserTurn payload has an empty system string (TS client often omits system).
//
// Contract (payload fields on SubmitUserTurn, optional):
//   - fetch_system_prompt_if_empty (bool): when true, and system is empty after trim, run
//     [querycontext.FetchSystemPromptParts] like gou-demo streaming parity / socket hosts.
//   - cwd (string): working directory for context discovery; default os.Getwd().
//   - extra_claude_md_roots ([]string): passed as [querycontext.FetchOpts.ExtraClaudeMdRoots]
//     (still requires CLAUDE_CODE_ADDITIONAL_DIRECTORIES_CLAUDE_MD=1 for extra roots in claudemd).
//   - custom_system_prompt, append_system_prompt (string): same semantics as ToolUseContext options.
//
// Environment (opt-in default when payload flag absent):
//   - CCB_ENGINE_FETCH_SYSTEM_PROMPT_IF_EMPTY=1|true: same as fetch_system_prompt_if_empty=true.
//
// When a user-context reminder is produced, it is prepended then merged with the following user
// message via a second [messagesapi.NormalizeMessagesForAPI] pass (see [goc/gou/ccbhydrate.PrependUserMessageJSON]).
package submitfill

import (
	"context"
	"encoding/json"
	"os"
	"slices"
	"strings"

	"goc/ccb-engine/settingsfile"
	"goc/commands"
	"goc/gou/ccbhydrate"
	"goc/modelenv"
	"goc/querycontext"
)

// Options configures optional system/message fill for SubmitUserTurn.
type Options struct {
	FetchIfEmpty bool
	Cwd          string
	// ToolsJSON Anthropic-style tools array; used only to derive enabled tool names for session guidance.
	ToolsJSON json.RawMessage
	// ExtraClaudeMdRoots optional extra project roots for CLAUDE.md (with env gate in claudemd).
	ExtraClaudeMdRoots []string
	CustomSystemPrompt string
	AppendSystemPrompt string
	// ModelID when empty defaults to the same model env chain as HTTP ([modelenv.FirstNonEmpty])
	// or a built-in demo default.
	ModelID string
}

func envTruthy(key string) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

// FetchDesired returns true when payload requests fill or CCB_ENGINE_FETCH_SYSTEM_PROMPT_IF_EMPTY is set.
func FetchDesired(payloadFlag bool) bool {
	return payloadFlag || envTruthy("CCB_ENGINE_FETCH_SYSTEM_PROMPT_IF_EMPTY")
}

func toolNamesFromToolsJSON(raw json.RawMessage) map[string]struct{} {
	var defs []struct {
		Name string `json:"name"`
	}
	_ = json.Unmarshal(raw, &defs)
	m := make(map[string]struct{}, len(defs))
	for _, t := range defs {
		if strings.TrimSpace(t.Name) != "" {
			m[strings.TrimSpace(t.Name)] = struct{}{}
		}
	}
	return m
}

func mergedSystemLocale() (lang, outputStyleName, outputStylePrompt string) {
	projRoot := settingsfile.ProjectRootLastResolved()
	locLang, locStyleKey, _ := settingsfile.MergeGouDemoLocalePrefs(projRoot, true)
	lang = strings.TrimSpace(os.Getenv("CLAUDE_CODE_LANGUAGE"))
	if lang == "" {
		lang = locLang
	}
	on, op := commands.ResolveGouDemoOutputStyle(
		os.Getenv("CLAUDE_CODE_OUTPUT_STYLE_NAME"),
		os.Getenv("CLAUDE_CODE_OUTPUT_STYLE_PROMPT"),
		locStyleKey,
	)
	return lang, on, op
}

func defaultModelID(fallback string) string {
	if m := modelenv.FirstNonEmpty(); m != "" {
		return m
	}
	if strings.TrimSpace(fallback) != "" {
		return strings.TrimSpace(fallback)
	}
	return "claude-sonnet-4-20250514"
}

// ApplyIfEmpty when system is empty and opts.FetchIfEmpty, builds system from querycontext and
// prepends user-context reminder to messages when non-empty. Otherwise returns inputs unchanged.
func ApplyIfEmpty(system string, messages json.RawMessage, opts Options) (outSystem string, outMsgs json.RawMessage, err error) {
	outSystem = system
	outMsgs = messages
	if strings.TrimSpace(system) != "" || !opts.FetchIfEmpty {
		return outSystem, outMsgs, nil
	}
	cwd := strings.TrimSpace(opts.Cwd)
	if cwd == "" {
		var errWd error
		cwd, errWd = os.Getwd()
		if errWd != nil {
			cwd = "."
		}
	}
	lang, outName, outPrompt := mergedSystemLocale()
	gouOpts := commands.GouDemoSystemOpts{
		EnabledToolNames:       toolNamesFromToolsJSON(opts.ToolsJSON),
		SkillToolCommands:      nil,
		ModelID:                defaultModelID(opts.ModelID),
		Cwd:                    cwd,
		Language:               lang,
		DiscoverSkillsToolName: strings.TrimSpace(os.Getenv("CLAUDE_CODE_DISCOVER_SKILLS_TOOL_NAME")),
		NonInteractiveSession:  envTruthy("GOU_DEMO_NON_INTERACTIVE"),
		OutputStyleName:        outName,
		OutputStylePrompt:      outPrompt,
	}
	commands.ApplyGouDemoRuntimeEnv(&gouOpts)
	customSys := strings.TrimSpace(opts.CustomSystemPrompt)
	appendSys := strings.TrimSpace(opts.AppendSystemPrompt)
	extra := slices.Clone(opts.ExtraClaudeMdRoots)
	partsRes, errParts := querycontext.FetchSystemPromptParts(context.Background(), querycontext.FetchOpts{
		CustomSystemPrompt: customSys,
		Gou:                gouOpts,
		ExtraClaudeMdRoots: extra,
	})
	if errParts != nil {
		outSystem = commands.BuildGouDemoSystemPrompt(gouOpts)
		if appendSys != "" {
			outSystem = strings.TrimSpace(outSystem + "\n\n" + appendSys)
		}
		return outSystem, outMsgs, nil
	}
	reminder := querycontext.FormatUserContextReminder(partsRes.UserContext)
	if strings.TrimSpace(reminder) != "" {
		outMsgs, err = ccbhydrate.PrependUserMessageJSON(outMsgs, reminder)
		if err != nil {
			return "", messages, err
		}
	}
	var base []string
	if customSys != "" {
		base = []string{customSys}
	} else {
		base = slices.Clone(partsRes.DefaultSystemPrompt)
	}
	if appendSys != "" {
		base = append(base, appendSys)
	}
	fullParts := querycontext.AppendSystemContextParts(base, partsRes.SystemContext)
	outSystem = strings.Join(fullParts, "\n\n")
	return outSystem, outMsgs, nil
}
