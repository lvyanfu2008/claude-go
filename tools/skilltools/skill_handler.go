package skilltools

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"goc/commands"
	"goc/slashresolve"
	"goc/types"
)

// NewSkillMultiMessageHandler returns a MultiMessageToolHandler that expands
// Skill tool invocations into a metadata user message + content user message
// (mirrors TS processPromptSlashCommand → getMessagesForPromptSlashCommand).
//
// The metadata message carries <command-message>/<command-name>/<command-args>
// XML tags; the content message carries the resolved skill text with isMeta: true.
func NewSkillMultiMessageHandler(commandsList []types.Command, sessionID string) func(
	ctx context.Context,
	name, toolUseID string,
	input json.RawMessage,
	assistantUUID string,
) ([]types.Message, bool) {
	sid := strings.TrimSpace(sessionID)
	if sid == "" {
		sid = "gou-demo"
	}

	return func(
		ctx context.Context,
		name, toolUseID string,
		input json.RawMessage,
		assistantUUID string,
	) ([]types.Message, bool) {
		_ = ctx
		if name != SkillToolName() {
			return nil, false
		}

		var in struct {
			Skill string `json:"skill"`
			Args  string `json:"args"`
		}
		if err := json.Unmarshal(input, &in); err != nil {
			return nil, false
		}
		trimmed := strings.TrimSpace(in.Skill)
		if trimmed == "" {
			return nil, false
		}
		normalized := trimmed
		if strings.HasPrefix(normalized, "/") {
			normalized = normalized[1:]
		}

		found := commands.FindCommand(normalized, commandsList)
		if found == nil {
			return nil, false
		}
		if found.DisableModelInvocation != nil && *found.DisableModelInvocation {
			return nil, false
		}
		if found.Type != "prompt" {
			return nil, false
		}

		cwd := "."
		if wd, err := os.Getwd(); err == nil {
			cwd = wd
		}

		var res types.SlashResolveResult
		if found.SkillRoot != nil && strings.TrimSpace(*found.SkillRoot) != "" {
			var err error
			res, err = slashresolve.ResolveDiskSkill(*found, in.Args, sid)
			if err != nil {
				return nil, false
			}
		} else if slashresolve.IsBundledPrompt(*found) {
			var err error
			res, err = slashresolve.ResolveBundledSkill(*found, in.Args, sid, &slashresolve.BundledResolveOptions{
				Cwd: cwd,
			})
			if err != nil {
				return nil, false
			}
		} else {
			return nil, false
		}

		// Build metadata user message (XML tags, mirrors TS formatSlashCommandLoadingMetadata).
		metadataParts := []string{
			fmt.Sprintf("<command-message>%s</command-message>", normalized),
			fmt.Sprintf("<command-name>/%s</command-name>", normalized),
		}
		args := strings.TrimSpace(in.Args)
		if args != "" {
			metadataParts = append(metadataParts, fmt.Sprintf("<command-args>%s</command-args>", args))
		}
		metadataContent := strings.Join(metadataParts, "\n")

		metadataMsg := newUserMessage(metadataContent, nil, nil)
		if metadataMsg.UUID == "" {
			return nil, false
		}

		// Build content user message with resolved skill text (isMeta: true).
		trueVal := true
		contentMsg := newUserMessage(res.UserText, &assistantUUID, &trueVal)
		if contentMsg.UUID == "" {
			return nil, false
		}

		return []types.Message{metadataMsg, contentMsg}, true
	}
}

// newUserMessage creates a user message with the given content, optional UUID, and optional isMeta.
// Mirrors processuserinput.newUserMessage without importing that package.
func newUserMessage(content any, uuidOpt *string, isMeta *bool) types.Message {
	id := randomUUID()
	if uuidOpt != nil && *uuidOpt != "" {
		id = *uuidOpt
	}
	inner := map[string]any{"role": "user", "content": content}
	msgInner, err := json.Marshal(inner)
	if err != nil {
		return types.Message{}
	}
	m := types.Message{
		Type:    types.MessageTypeUser,
		UUID:    id,
		Message: json.RawMessage(msgInner),
	}
	if isMeta != nil {
		m.IsMeta = isMeta
	}
	return m
}

// randomUUID generates a v4 UUID using crypto/rand.
func randomUUID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "00000000-0000-4000-8000-000000000000"
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%s",
		uint32(b[0])<<24|uint32(b[1])<<16|uint32(b[2])<<8|uint32(b[3]),
		uint16(b[4])<<8|uint16(b[5]),
		uint16(b[6])<<8|uint16(b[7]),
		uint16(b[8])<<8|uint16(b[9]),
		hex.EncodeToString(b[10:16]),
	)
}
