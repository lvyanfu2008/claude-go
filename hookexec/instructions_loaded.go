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
	if HooksDisabled() || ShouldDisableAllHooksIncludingManaged() || ShouldSkipHookDueToTrust() || len(table) == 0 {
		return
	}
	base.HookEventName = hookEventInstructionsLoaded
	in := instructionsLoadedInput{BaseHookInput: base, InstructionsLoadedFields: fields}
	jsonIn, err := marshalHookInput(in)
	if err != nil {
		return
	}
	var hookInput map[string]any
	if err := json.Unmarshal([]byte(jsonIn), &hookInput); err != nil {
		return
	}
	if len(CommandHooksForHookInput(table, hookInput)) == 0 {
		return
	}
	wd := trimOrDot(workDir)
	go func() {
		bg := context.Background()
		if ctx != nil {
			_ = ctx
		}
		_ = ExecuteCommandHooksOutsideREPLParallel(OutsideReplCommandParams{
			Ctx:       bg,
			WorkDir:   wd,
			Hooks:     table,
			JSONInput: jsonIn,
			TimeoutMs: batchTimeoutMs,
		})
	}()
}
