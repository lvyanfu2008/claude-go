// Mirrors src/utils/attachments.ts attachment discriminator types used from processUserInput.
package types

// AgentMentionAttachment mirrors src/utils/attachments.ts AgentMentionAttachment.
type AgentMentionAttachment struct {
	Type      string `json:"type"` // agent_mention
	AgentType string `json:"agentType"`
}
