package messagesapi

import (
	"encoding/json"

	"goc/appstate"
	"goc/types"
)

const (
	turnsBetweenPlanAttachments     = 5
	fullReminderEveryNAttachments   = 5
	turnsBetweenAutoModeAttachments = 5
)

// BuildDynamicAttachments returns attachments that should be injected based on
// the current appstate (plan mode, auto mode, exit/reentry flags).
// Mirrors TS getPlanModeAttachments + getAutoModeAttachments.
func BuildDynamicAttachments(state *appstate.AppState, messages []types.Message, opts Options) []json.RawMessage {
	if state == nil {
		return nil
	}

	var result []json.RawMessage

	// 1. Plan mode exit attachment (one-shot)
	if state.NeedsPlanModeExitAttachment {
		result = append(result, buildPlanModeExitAttachment(state))
	}

	// 2. Auto mode exit attachment (one-shot)
	if state.NeedsAutoModeExitAttachment {
		result = append(result, buildAutoModeExitAttachment())
	}

	// 3. Plan mode reentry (one-shot)
	if state.HasExitedPlanMode && planFileExists(state) {
		result = append(result, buildPlanModeReentryAttachment(state))
	}

	// 4. Plan mode main attachment (throttled)
	if state.ToolPermissionContext.Mode == types.PermissionPlan {
		att := buildPlanModeAttachment(state, messages, opts)
		if att != nil {
			result = append(result, att)
		}
	}

	// 5. Auto mode attachment (throttled)
	if appstate.GlobalAutoModeState.IsActive() {
		att := buildAutoModeAttachment(messages)
		if att != nil {
			result = append(result, att)
		}
	}

	return result
}

// planFileExists checks if a plan file already exists on disk.
func planFileExists(state *appstate.AppState) bool {
	return false
}

// planFilePath extracts the plan file path from appstate.
func planFilePath(state *appstate.AppState) string {
	return "plan.md"
}

// buildPlanModeExitAttachment creates a plan_mode_exit attachment.
func buildPlanModeExitAttachment(state *appstate.AppState) json.RawMessage {
	att := map[string]any{
		"type":         "plan_mode_exit",
		"planExists":   planFileExists(state),
		"planFilePath": planFilePath(state),
	}
	raw, _ := json.Marshal(att)
	return raw
}

// buildAutoModeExitAttachment creates an auto_mode_exit attachment.
func buildAutoModeExitAttachment() json.RawMessage {
	raw, _ := json.Marshal(map[string]any{"type": "auto_mode_exit"})
	return raw
}

// buildPlanModeReentryAttachment creates a plan_mode_reentry attachment (one-shot).
func buildPlanModeReentryAttachment(state *appstate.AppState) json.RawMessage {
	att := map[string]any{
		"type":         "plan_mode_reentry",
		"planFilePath": planFilePath(state),
	}
	raw, _ := json.Marshal(att)
	return raw
}

// buildPlanModeAttachment creates a plan_mode attachment (throttled full/sparse).
// Returns nil if throttling says to skip this turn.
func buildPlanModeAttachment(state *appstate.AppState, messages []types.Message, opts Options) json.RawMessage {
	turnCount, found := countTurnsSinceAttachment(messages, "plan_mode", "plan_mode_reentry")
	if found && turnCount < turnsBetweenPlanAttachments {
		return nil
	}

	attachmentCount := countPlanModeAttachments(messages) + 1
	reminderType := "sparse"
	if attachmentCount%fullReminderEveryNAttachments == 1 {
		reminderType = "full"
	}

	att := map[string]any{
		"type":         "plan_mode",
		"reminderType": reminderType,
		"isSubAgent":   false,
		"planFilePath": planFilePath(state),
		"planExists":   planFileExists(state),
	}
	raw, _ := json.Marshal(att)
	return raw
}

// buildAutoModeAttachment creates an auto_mode attachment (throttled full/sparse).
// Returns nil if throttling says to skip this turn.
func buildAutoModeAttachment(messages []types.Message) json.RawMessage {
	turnCount, found := countTurnsSinceAttachment(messages, "auto_mode")
	if found && turnCount < turnsBetweenAutoModeAttachments {
		return nil
	}

	attachmentCount := countAttachmentsOfType(messages, "auto_mode") + 1
	reminderType := "sparse"
	if attachmentCount%fullReminderEveryNAttachments == 1 {
		reminderType = "full"
	}

	raw, _ := json.Marshal(map[string]any{
		"type":         "auto_mode",
		"reminderType": reminderType,
	})
	return raw
}

// countTurnsSinceAttachment walks messages backwards counting human turns
// until it finds an attachment of the given types. Returns (turnCount, found).
func countTurnsSinceAttachment(messages []types.Message, attachmentTypes ...string) (int, bool) {
	turnCount := 0
	typeSet := make(map[string]struct{}, len(attachmentTypes))
	for _, t := range attachmentTypes {
		typeSet[t] = struct{}{}
	}

	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		if msg.Type == types.MessageTypeAttachment {
			var head struct {
				Type string `json:"type"`
			}
			if len(msg.Attachment) > 0 {
				if err := json.Unmarshal(msg.Attachment, &head); err == nil {
					if _, ok := typeSet[head.Type]; ok {
						return turnCount, true
					}
				}
			}
			continue
		}
		if msg.Type == types.MessageTypeUser {
			if isTruthy(msg.IsMeta) {
				continue
			}
			turnCount++
		}
	}
	return turnCount, false
}

// countPlanModeAttachments counts plan_mode + plan_mode_reentry attachments in the message history.
func countPlanModeAttachments(messages []types.Message) int {
	return countAttachmentsOfType(messages, "plan_mode") + countAttachmentsOfType(messages, "plan_mode_reentry")
}

// countAttachmentsOfType counts attachments of a specific type.
func countAttachmentsOfType(messages []types.Message, attType string) int {
	count := 0
	for _, msg := range messages {
		if msg.Type == types.MessageTypeAttachment {
			var head struct {
				Type string `json:"type"`
			}
			if len(msg.Attachment) > 0 {
				if err := json.Unmarshal(msg.Attachment, &head); err == nil && head.Type == attType {
					count++
				}
			}
		}
	}
	return count
}
