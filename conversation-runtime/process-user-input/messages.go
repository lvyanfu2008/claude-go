package processuserinput

import (
	"encoding/json"
	"strings"
	"time"

	"goc/types"
	"goc/utils"
)

func nowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}

// getContentText mirrors src/utils/messageContentText.ts getContentText for string | []ContentBlockParam.
func getContentText(content string, blocks []types.ContentBlockParam) *string {
	if content != "" {
		return &content
	}
	if len(blocks) == 0 {
		return nil
	}
	var sb strings.Builder
	for _, b := range blocks {
		if b.Type == "text" && b.Text != "" {
			if sb.Len() > 0 {
				sb.WriteByte('\n')
			}
			sb.WriteString(b.Text)
		}
	}
	s := strings.TrimSpace(sb.String())
	if s == "" {
		return nil
	}
	return &s
}

// newUserMessage mirrors createUserMessage (src/utils/messages.ts) for the shapes used by processUserInput.
func newUserMessage(content any, uuidOpt *string, isMeta *bool, permissionMode *types.PermissionMode) (types.Message, error) {
	id := randomUUID()
	if uuidOpt != nil && *uuidOpt != "" {
		id = *uuidOpt
	}
	inner := map[string]any{"role": "user", "content": content}
	msgInner, err := json.Marshal(inner)
	if err != nil {
		return types.Message{}, err
	}
	m := types.Message{
		Type:    types.MessageTypeUser,
		UUID:    id,
		Message: json.RawMessage(msgInner),
	}
	if isMeta != nil {
		m.IsMeta = isMeta
	}
	if permissionMode != nil {
		pm := string(*permissionMode)
		m.PermissionMode = &pm
	}
	return m, nil
}

func newSystemInformationalMessage(content, level string) types.Message {
	ts := nowRFC3339()
	id := randomUUID()
	return types.Message{
		Type:      types.MessageTypeSystem,
		UUID:      id,
		Content:   json.RawMessage(mustMarshalJSONString(content)),
		Subtype:   strPtr("informational"),
		Level:     strPtr(level),
		Timestamp: &ts,
		IsMeta:    boolPtr(false),
	}
}

func newCommandInputMessage(content string) types.Message {
	ts := nowRFC3339()
	id := randomUUID()
	return types.Message{
		Type:      types.MessageTypeSystem,
		UUID:      id,
		Content:   json.RawMessage(mustMarshalJSONString(content)),
		Subtype:   strPtr("local_command"),
		Level:     strPtr("info"),
		Timestamp: &ts,
		IsMeta:    boolPtr(false),
	}
}

func newHookAdditionalContextAttachment(parts []string, toolUseID, hookEvent string) (types.Message, error) {
	att := map[string]any{
		"type":      "hook_additional_context",
		"content":   parts,
		"hookName":  "UserPromptSubmit",
		"toolUseID": toolUseID,
		"hookEvent": hookEvent,
	}
	raw, err := json.Marshal(att)
	if err != nil {
		return types.Message{}, err
	}
	return types.Message{
		Type:       types.MessageTypeAttachment,
		UUID:       randomUUID(),
		Attachment: json.RawMessage(raw),
	}, nil
}

func mustMarshalJSONString(s string) []byte {
	b, err := json.Marshal(s)
	if err != nil {
		return []byte(`""`)
	}
	return b
}

func strPtr(s string) *string { return &s }
func boolPtr(b bool) *bool    { return &b }

func getUserPromptSubmitHookBlockingMessage(blockingError types.HookBlockingError) string {
	return "UserPromptSubmit operation blocked by hook:\n" + blockingError.BlockingError
}

// mergeHookSuccessMessage appends hook attachment message handling hook_success truncation (mirrors TS).
func mergeHookSuccessMessage(dst *[]types.Message, hookMsg json.RawMessage) error {
	var probe struct {
		Attachment struct {
			Type    string          `json:"type"`
			Content json.RawMessage `json:"content"`
		} `json:"attachment"`
	}
	if err := json.Unmarshal(hookMsg, &probe); err != nil {
		return err
	}
	if probe.Attachment.Type == "hook_success" {
		var contentStr string
		if err := json.Unmarshal(probe.Attachment.Content, &contentStr); err == nil && contentStr != "" {
			trunc := applyTruncation(contentStr)
			var full map[string]any
			_ = json.Unmarshal(hookMsg, &full)
			if att, ok := full["attachment"].(map[string]any); ok {
				att["content"] = trunc
			}
			out, err := json.Marshal(full)
			if err != nil {
				return err
			}
			var m types.Message
			if err := json.Unmarshal(out, &m); err != nil {
				return err
			}
			*dst = append(*dst, m)
			return nil
		}
		return nil
	}
	var m types.Message
	if err := json.Unmarshal(hookMsg, &m); err != nil {
		return err
	}
	*dst = append(*dst, m)
	return nil
}

// messageJSONType returns the `type` field of a serialized message (user | assistant | system | attachment | progress | …).
func messageJSONType(raw json.RawMessage) string {
	var x struct {
		Type string `json:"type"`
	}
	_ = json.Unmarshal(raw, &x)
	return x.Type
}

func isValidImagePaste(c utils.PastedContent) bool {
	return c.Type == "image" && len(c.Content) > 0
}
