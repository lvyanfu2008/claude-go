package messagesapi

import (
	"encoding/json"
	"fmt"
	"strings"

	"goc/types"
)

// teammateMessageXMLTag mirrors src/constants/xml.ts TEAMMATE_MESSAGE_TAG.
const teammateMessageXMLTag = "teammate-message"

// formatTeammateMessages mirrors src/utils/teammateMailbox.ts formatTeammateMessages.
func formatTeammateMessages(messages []struct {
	From      string `json:"from"`
	Text      string `json:"text"`
	Timestamp string `json:"timestamp"`
	Color     string `json:"color,omitempty"`
	Summary   string `json:"summary,omitempty"`
}) string {
	var b strings.Builder
	for i, m := range messages {
		if i > 0 {
			b.WriteString("\n\n")
		}
		colorAttr := ""
		if m.Color != "" {
			colorAttr = fmt.Sprintf(` color="%s"`, m.Color)
		}
		summaryAttr := ""
		if m.Summary != "" {
			summaryAttr = fmt.Sprintf(` summary="%s"`, m.Summary)
		}
		fmt.Fprintf(&b, `<%s teammate_id="%s"%s%s>
%s
</%s>`,
			teammateMessageXMLTag, m.From, colorAttr, summaryAttr, m.Text, teammateMessageXMLTag)
	}
	return b.String()
}

// normalizeAttachmentTeammateMailbox mirrors TS normalizeAttachmentForAPI (isAgentSwarmsEnabled branch).
// TS does not wrap with wrapMessagesInSystemReminder for this type.
func normalizeAttachmentTeammateMailbox(att json.RawMessage, uuidGen func() string) ([]types.Message, error) {
	var a struct {
		Messages []struct {
			From      string `json:"from"`
			Text      string `json:"text"`
			Timestamp string `json:"timestamp"`
			Color     string `json:"color,omitempty"`
			Summary   string `json:"summary,omitempty"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(att, &a); err != nil {
		return nil, err
	}
	body := formatTeammateMessages(a.Messages)
	m := createUserMessageString(body, uuidGen(), "", true)
	syncTopLevelContent(&m)
	return []types.Message{m}, nil
}

func normalizeAttachmentTeamContext(att json.RawMessage, uuidGen func() string) ([]types.Message, error) {
	var a struct {
		TeamName        string `json:"teamName"`
		AgentName       string `json:"agentName"`
		TeamConfigPath  string `json:"teamConfigPath"`
		TaskListPath    string `json:"taskListPath"`
	}
	if err := json.Unmarshal(att, &a); err != nil {
		return nil, err
	}
	jsonExample := "```json\n{\n  \"to\": \"team-lead\",\n  \"message\": \"Your message here\",\n  \"summary\": \"Brief 5-10 word preview\"\n}\n```"
	content := fmt.Sprintf(
		"<system-reminder>\n# Team Coordination\n\nYou are a teammate in team \"%s\".\n\n**Your Identity:**\n- Name: %s\n\n**Team Resources:**\n- Team config: %s\n- Task list: %s\n\n**Team Leader:** The team lead's name is \"team-lead\". Send updates and completion notifications to them.\n\nRead the team config to discover your teammates' names. Check the task list periodically. Create new tasks when work should be divided. Mark tasks resolved when complete.\n\n**IMPORTANT:** Always refer to teammates by their NAME (e.g., \"team-lead\", \"analyzer\", \"researcher\"), never by UUID. When messaging, use the name directly:\n\n%s\n\n</system-reminder>",
		a.TeamName, a.AgentName, a.TeamConfigPath, a.TaskListPath, jsonExample,
	)
	m := createUserMessageString(content, uuidGen(), "", true)
	syncTopLevelContent(&m)
	return []types.Message{m}, nil
}

func normalizeAttachmentSkillDiscovery(att json.RawMessage, uuidGen func() string) ([]types.Message, error) {
	var a struct {
		Skills []struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		} `json:"skills"`
	}
	if err := json.Unmarshal(att, &a); err != nil {
		return nil, err
	}
	if len(a.Skills) == 0 {
		return nil, nil
	}
	var lines []string
	for _, s := range a.Skills {
		lines = append(lines, fmt.Sprintf("- %s: %s", s.Name, s.Description))
	}
	body := "Skills relevant to your task:\n\n" + strings.Join(lines, "\n") + "\n\n" +
		"These skills encode project-specific conventions. " +
		`Invoke via Skill("<name>") for complete instructions.`
	msgs := []types.Message{createUserMessageString(body, uuidGen(), "", true)}
	return wrapMessagesInSystemReminder(msgs), nil
}
