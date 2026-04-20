package compactservice

import (
	"strings"

	"goc/types"
)

// HookResultMessage mirrors the TS union HookResultMessage (attachment or system-variant rows produced by hook phases).
// Using types.Message keeps wire parity without introducing a second union.
type HookResultMessage = types.Message

// CompactionResult mirrors CompactionResult in services/compact/compact.ts.
// Field order follows TS; new fields must be added, not embedded as a raw blob.
type CompactionResult struct {
	BoundaryMarker         types.Message
	SummaryMessages        []types.Message // TS UserMessage[] — content is isCompactSummary=true user rows
	Attachments            []types.Message // TS AttachmentMessage[]
	HookResults            []HookResultMessage
	MessagesToKeep         []types.Message // optional suffix preserved across compaction
	UserDisplayMessage     string          // surfaced in CLI when set
	PreCompactTokenCount   int
	PostCompactTokenCount  int // compact-call TOTAL usage (input + cache + output), TS semantics
	TruePostCompactTokenCount int // rough estimate of the resulting conversation size
	CompactionUsage        *TokenUsage
}

// BuildPostCompactMessages mirrors buildPostCompactMessages in TS.
// Order: boundaryMarker, summaryMessages, messagesToKeep, attachments, hookResults.
func BuildPostCompactMessages(r CompactionResult) []types.Message {
	out := make([]types.Message, 0, 1+len(r.SummaryMessages)+len(r.MessagesToKeep)+len(r.Attachments)+len(r.HookResults))
	out = append(out, r.BoundaryMarker)
	out = append(out, r.SummaryMessages...)
	out = append(out, r.MessagesToKeep...)
	out = append(out, r.Attachments...)
	out = append(out, r.HookResults...)
	return out
}

// MergeHookInstructions mirrors mergeHookInstructions in TS.
// Empty strings normalize to "" (TS returns undefined; Go callers check empty string).
func MergeHookInstructions(userInstructions, hookInstructions string) string {
	if strings.TrimSpace(hookInstructions) == "" {
		return userInstructions
	}
	if strings.TrimSpace(userInstructions) == "" {
		return hookInstructions
	}
	return userInstructions + "\n\n" + hookInstructions
}
