package hookexec

import (
	"context"
	"encoding/json"
	"strings"
)

const hookEventInstructionsLoaded = "InstructionsLoaded"

// InstructionsLoadedFields are event-specific stdin fields (merged with BaseHookInput).
type InstructionsLoadedFields struct {
	FilePath        string   `json:"file_path"`
	MemoryType      string   `json:"memory_type"`
	LoadReason      string   `json:"load_reason"`
	Globs           []string `json:"globs,omitempty"`
	TriggerFilePath string   `json:"trigger_file_path,omitempty"`
	ParentFilePath  string   `json:"parent_file_path,omitempty"`
}

type instructionsLoadedInput struct {
	BaseHookInput
	InstructionsLoadedFields
}

// HasInstructionsLoaded returns true when merged hooks include at least one InstructionsLoaded command hook.
func HasInstructionsLoaded(table HooksTable) bool {
	for _, mg := range table[hookEventInstructionsLoaded] {
		for _, raw := range mg.Hooks {
			var h commandHook
			if err := json.Unmarshal(raw, &h); err != nil {
				continue
			}
			if strings.TrimSpace(h.Type) == "command" && strings.TrimSpace(h.Command) != "" {
				return true
			}
		}
	}
	return false
}

// FireInstructionsLoaded runs all matching InstructionsLoaded command hooks fire-and-forget (TS executeInstructionsLoadedHooks).
func FireInstructionsLoaded(ctx context.Context, table HooksTable, workDir string, base BaseHookInput, fields InstructionsLoadedFields, batchTimeoutMs int) {
	if HooksDisabled() || len(table) == 0 {
		return
	}
	base.HookEventName = hookEventInstructionsLoaded
	in := instructionsLoadedInput{BaseHookInput: base, InstructionsLoadedFields: fields}
	matchers := table[hookEventInstructionsLoaded]
	if len(matchers) == 0 {
		return
	}
	jsonIn, err := marshalHookInput(in)
	if err != nil {
		return
	}
	hooks := collectMatchingCommands(matchers, fields.LoadReason)
	if len(hooks) == 0 {
		return
	}
	wd := trimOrDot(workDir)
	for _, h := range hooks {
		h := h
		payload := jsonIn
		go func() {
			bg := context.Background()
			_ = ctx
			ms := hookTimeoutMS(h, batchTimeoutMs)
			_, _, _, _ = RunCommandHook(bg, wd, h.Command, payload, ms)
		}()
	}
}

func collectMatchingCommands(matchers []MatcherGroup, matchQuery string) []commandHook {
	var out []commandHook
	for _, mg := range matchers {
		if !MatchesPattern(matchQuery, mg.Matcher) {
			continue
		}
		for _, raw := range mg.Hooks {
			var h commandHook
			if err := json.Unmarshal(raw, &h); err != nil {
				continue
			}
			if strings.TrimSpace(h.Type) != "command" || strings.TrimSpace(h.Command) == "" {
				continue
			}
			out = append(out, h)
		}
	}
	return out
}

func matchingCommandHooks(table HooksTable, event, matchQuery string) []commandHook {
	matchers := table[event]
	return collectMatchingCommands(matchers, matchQuery)
}
