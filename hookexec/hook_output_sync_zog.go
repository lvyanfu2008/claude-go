package hookexec

import (
	"encoding/json"
	"fmt"
	"strings"

	z "github.com/Oudwins/zog"
)

// syncHookStdoutParsed mirrors the top-level fields of src/types/hooks.ts syncHookResponseSchema
// plus hookSpecificOutput (validated separately for UserPromptSubmit).
type syncHookStdoutParsed struct {
	Continue           *bool           `json:"continue"`
	SuppressOutput     *bool           `json:"suppressOutput"`
	StopReason         *string         `json:"stopReason"`
	Decision           *string         `json:"decision"`
	Reason             *string         `json:"reason"`
	SystemMessage      *string         `json:"systemMessage"`
	HookSpecificOutput json.RawMessage `json:"hookSpecificOutput"`
}

// hookSpecificLooseParsed is a minimal projection of hookSpecificOutput union members for Zog parse;
// wrong hookEventName for UserPromptSubmit is rejected after parse (see userPromptSubmitAggregates).
type hookSpecificLooseParsed struct {
	HookEventName     *string `json:"hookEventName"`
	AdditionalContext *string `json:"additionalContext"`
}

var syncHookStdoutScalarSchema = z.Struct(z.Shape{
	"Continue":       z.Ptr(z.Bool()),
	"SuppressOutput": z.Ptr(z.Bool()),
	"StopReason":     z.Ptr(z.String()),
	"Decision":       z.Ptr(z.String().OneOf([]string{"approve", "block"})),
	"Reason":         z.Ptr(z.String()),
	"SystemMessage":  z.Ptr(z.String()),
})

var hookSpecificLooseSchema = z.Struct(z.Shape{
	"HookEventName":     z.Ptr(z.String()),
	"AdditionalContext": z.Ptr(z.String()),
})

func parseSyncHookStdoutJSON(trimmed string) (syncHookStdoutParsed, error) {
	var dest syncHookStdoutParsed
	if err := json.Unmarshal([]byte(trimmed), &dest); err != nil {
		return dest, err
	}
	if issues := syncHookStdoutScalarSchema.Validate(&dest); len(issues) > 0 {
		return dest, fmt.Errorf("%v", issues)
	}
	if len(bytesTrimSpaceJSON(dest.HookSpecificOutput)) > 0 {
		var hsoMap map[string]any
		if err := json.Unmarshal(dest.HookSpecificOutput, &hsoMap); err != nil {
			return dest, fmt.Errorf("hookSpecificOutput: %w", err)
		}
		var hso hookSpecificLooseParsed
		if issues := hookSpecificLooseSchema.Parse(hsoMap, &hso); len(issues) > 0 {
			return dest, fmt.Errorf("hookSpecificOutput: %v", issues)
		}
	}
	return dest, nil
}

func bytesTrimSpaceJSON(r json.RawMessage) []byte {
	return []byte(strings.TrimSpace(string(r)))
}

func parsedSyncHookToLegacyTop(p syncHookStdoutParsed) syncUserPromptSubmitJSON {
	out := syncUserPromptSubmitJSON{
		Continue:           p.Continue,
		SuppressOutput:     p.SuppressOutput,
		StopReason:         p.StopReason,
		HookSpecificOutput: p.HookSpecificOutput,
	}
	if p.Decision != nil {
		out.Decision = strings.TrimSpace(*p.Decision)
	}
	if p.Reason != nil {
		out.Reason = strings.TrimSpace(*p.Reason)
	}
	if p.SystemMessage != nil {
		out.SystemMessage = strings.TrimSpace(*p.SystemMessage)
	}
	return out
}
