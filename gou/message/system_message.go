// Package message implements TS-style message rendering for Go TUI.
// SystemMessageRenderer handles system messages.
package message

import (
	"encoding/json"
	"fmt"
	"strings"

	"goc/types"
)

// SystemMessageRenderer renders system messages.
type SystemMessageRenderer struct{}

// CanRender checks if this renderer can handle the message.
func (r *SystemMessageRenderer) CanRender(msg *types.Message) bool {
	return msg.Type == types.MessageTypeSystem
}

// Render renders a system message.
func (r *SystemMessageRenderer) Render(msg *types.Message, ctx *RenderContext) ([]string, error) {
	if !r.CanRender(msg) {
		return nil, fmt.Errorf("SystemMessageRenderer cannot render message type: %s", msg.Type)
	}

	// Determine subtype
	subtype := "generic"
	if msg.Subtype != nil {
		subtype = *msg.Subtype
	}

	switch subtype {
	case "compact_boundary":
		return r.renderCompactBoundary(msg, ctx)
	case "local_command":
		return r.renderLocalCommand(msg, ctx)
	case "informational":
		return r.renderInformational(msg, ctx)
	case "api_error":
		return r.renderApiError(msg, ctx)
	case "stop_hook_summary":
		return r.renderStopHookSummary(msg, ctx)
	default:
		return r.renderGenericSystemMessage(msg, ctx)
	}
}

// Measure measures a system message.
func (r *SystemMessageRenderer) Measure(msg *types.Message, ctx *RenderContext) (int, error) {
	if !r.CanRender(msg) {
		return 0, fmt.Errorf("SystemMessageRenderer cannot measure message type: %s", msg.Type)
	}

	// Determine subtype
	subtype := "generic"
	if msg.Subtype != nil {
		subtype = *msg.Subtype
	}

	switch subtype {
	case "compact_boundary":
		return 1, nil
	case "local_command":
		return r.measureLocalCommand(msg, ctx)
	case "informational":
		return r.measureInformational(msg, ctx)
	case "api_error":
		return r.measureApiError(msg, ctx)
	case "stop_hook_summary":
		return r.measureStopHookSummary(msg, ctx)
	default:
		return r.measureGenericSystemMessage(msg, ctx)
	}
}

// renderCompactBoundary renders a compact boundary message.
func (r *SystemMessageRenderer) renderCompactBoundary(msg *types.Message, ctx *RenderContext) ([]string, error) {
	return []string{"---"}, nil
}

// renderLocalCommand renders a local command message.
func (r *SystemMessageRenderer) renderLocalCommand(msg *types.Message, ctx *RenderContext) ([]string, error) {
	// Extract content
	content := ""
	if msg.Content != nil {
		// Try to parse as string
		content = string(msg.Content)
		// Remove quotes if present
		content = strings.Trim(content, `"`)
	}

	if content == "" {
		return []string{"[Local command]"}, nil
	}

	// Render as user text message
	// Delegate to UserTextMessage renderer
	userRenderer := &UserMessageRenderer{}
	return userRenderer.renderTextBlock(map[string]interface{}{
		"type": "text",
		"text": content,
	}, ctx)
}

// measureLocalCommand measures a local command message.
func (r *SystemMessageRenderer) measureLocalCommand(msg *types.Message, ctx *RenderContext) (int, error) {
	content := ""
	if msg.Content != nil {
		content = string(msg.Content)
		content = strings.Trim(content, `"`)
	}

	if content == "" {
		return 1, nil
	}

	// Simple line count estimation
	return len(strings.Split(content, "\n")), nil
}

// renderInformational renders an informational system message.
func (r *SystemMessageRenderer) renderInformational(msg *types.Message, ctx *RenderContext) ([]string, error) {
	content := ""
	if msg.Content != nil {
		content = string(msg.Content)
		content = strings.Trim(content, `"`)
	}

	if content == "" {
		return []string{"ℹ [System message]"}, nil
	}

	// Add info icon
	lines := strings.Split(content, "\n")
	if len(lines) > 0 {
		lines[0] = "ℹ " + lines[0]
	}
	return lines, nil
}

// measureInformational measures an informational system message.
func (r *SystemMessageRenderer) measureInformational(msg *types.Message, ctx *RenderContext) (int, error) {
	content := ""
	if msg.Content != nil {
		content = string(msg.Content)
		content = strings.Trim(content, `"`)
	}

	if content == "" {
		return 1, nil
	}

	return len(strings.Split(content, "\n")), nil
}

// renderApiError renders an API error message.
func (r *SystemMessageRenderer) renderApiError(msg *types.Message, ctx *RenderContext) ([]string, error) {
	content := ""
	if msg.Content != nil {
		content = string(msg.Content)
		content = strings.Trim(content, `"`)
	}

	if content == "" {
		return []string{"✗ [API error]"}, nil
	}

	lines := strings.Split(content, "\n")
	if len(lines) > 0 {
		lines[0] = "✗ " + lines[0]
	}
	return lines, nil
}

// measureApiError measures an API error message.
func (r *SystemMessageRenderer) measureApiError(msg *types.Message, ctx *RenderContext) (int, error) {
	content := ""
	if msg.Content != nil {
		content = string(msg.Content)
		content = strings.Trim(content, `"`)
	}

	if content == "" {
		return 1, nil
	}

	return len(strings.Split(content, "\n")), nil
}

// renderStopHookSummary renders a stop hook summary message.
func (r *SystemMessageRenderer) renderStopHookSummary(msg *types.Message, ctx *RenderContext) ([]string, error) {
	// Similar to TS StopHookSummaryMessage component
	// Extract hook summary from message
	summary := "Hook executed"
	if msg.Content != nil {
		var text string
		if err := json.Unmarshal(msg.Content, &text); err == nil && text != "" {
			if len(text) > 60 {
				summary = text[:60] + "..."
			} else {
				summary = text
			}
		}
	}
	return []string{"🪝 " + summary}, nil
}

// measureStopHookSummary measures a stop hook summary message.
func (r *SystemMessageRenderer) measureStopHookSummary(msg *types.Message, ctx *RenderContext) (int, error) {
	// Stop hook summary is always 1 line
	return 1, nil
}

// renderGenericSystemMessage renders a generic system message.
func (r *SystemMessageRenderer) renderGenericSystemMessage(msg *types.Message, ctx *RenderContext) ([]string, error) {
	content := ""
	if msg.Content != nil {
		content = string(msg.Content)
		content = strings.Trim(content, `"`)
	}

	if content == "" {
		return []string{"[System]"}, nil
	}

	return strings.Split(content, "\n"), nil
}

// measureGenericSystemMessage measures a generic system message.
func (r *SystemMessageRenderer) measureGenericSystemMessage(msg *types.Message, ctx *RenderContext) (int, error) {
	content := ""
	if msg.Content != nil {
		content = string(msg.Content)
		content = strings.Trim(content, `"`)
	}

	if content == "" {
		return 1, nil
	}

	return len(strings.Split(content, "\n")), nil
}