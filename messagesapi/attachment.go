package messagesapi

import (
	"encoding/json"
	"fmt"
	"strings"

	"goc/types"
)

// normalizeAttachmentForAPI mirrors src/utils/messages.ts normalizeAttachmentForAPI (subset + common types).
func normalizeAttachmentForAPI(att json.RawMessage, opts Options, uuidGen func() string) ([]types.Message, error) {
	if len(att) == 0 {
		return nil, nil
	}
	var head struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(att, &head); err != nil {
		return nil, err
	}
	if opts.AgentSwarmsEnabled {
		switch head.Type {
		case "teammate_mailbox":
			return normalizeAttachmentTeammateMailbox(att, uuidGen)
		case "team_context":
			return normalizeAttachmentTeamContext(att, uuidGen)
		}
	}
	if opts.ExperimentalSkillSearch && head.Type == "skill_discovery" {
		return normalizeAttachmentSkillDiscovery(att, uuidGen)
	}
	switch head.Type {
	case "directory":
		return normalizeAttachmentDirectory(att, uuidGen)
	case "edited_text_file":
		return normalizeAttachmentEditedTextFile(att, uuidGen)
	case "file":
		return normalizeAttachmentFile(att, uuidGen)
	case "compact_file_reference":
		return normalizeAttachmentCompactFileRef(att, uuidGen)
	case "pdf_reference":
		return normalizeAttachmentPDFRef(att, uuidGen)
	case "selected_lines_in_ide":
		return normalizeAttachmentSelectedLines(att, uuidGen)
	case "opened_file_in_ide":
		return normalizeAttachmentOpenedFile(att, uuidGen)
	case "plan_file_reference":
		return normalizeAttachmentPlanFile(att, uuidGen)
	case "invoked_skills":
		return normalizeAttachmentInvokedSkills(att, uuidGen)
	case "todo_reminder":
		return normalizeAttachmentTodoReminder(att, uuidGen)
	case "nested_memory":
		return normalizeAttachmentNestedMemory(att, uuidGen)
	case "relevant_memories":
		return normalizeAttachmentRelevantMemories(att, uuidGen)
	case "task_reminder":
		if !opts.EnableTaskReminder {
			return nil, nil
		}
		return normalizeAttachmentTaskReminder(att, uuidGen)
	case "dynamic_skill":
		return normalizeAttachmentDynamicSkill()
	case "skill_listing":
		return normalizeAttachmentSkillListing(att, uuidGen)
	case "queued_command":
		return normalizeAttachmentQueuedCommand(att, uuidGen)
	case "output_style":
		return normalizeAttachmentOutputStyle(att, uuidGen)
	case "diagnostics":
		return normalizeAttachmentDiagnostics(att, uuidGen)
	case "plan_mode":
		return normalizeAttachmentPlanMode(att, opts, uuidGen)
	case "plan_mode_reentry":
		return normalizeAttachmentPlanModeReentry(att, uuidGen)
	case "plan_mode_exit":
		return normalizeAttachmentPlanModeExit(att, uuidGen)
	case "auto_mode":
		return normalizeAttachmentAutoMode(att, uuidGen)
	case "auto_mode_exit":
		return normalizeAttachmentAutoModeExit(uuidGen)
	case "critical_system_reminder":
		return normalizeAttachmentCriticalSystemReminder(att, uuidGen)
	case "mcp_resource":
		return normalizeAttachmentMCPResource(att, uuidGen)
	case "agent_mention":
		return normalizeAttachmentAgentMention(att, uuidGen)
	case "task_status":
		return normalizeAttachmentTaskStatus(att, uuidGen)
	case "async_hook_response":
		return normalizeAttachmentAsyncHookResponse(att, uuidGen)
	case "token_usage":
		return normalizeAttachmentTokenUsage(att, uuidGen)
	case "budget_usd":
		return normalizeAttachmentBudgetUSD(att, uuidGen)
	case "output_token_usage":
		return normalizeAttachmentOutputTokenUsage(att, uuidGen)
	case "hook_blocking_error":
		return normalizeAttachmentHookBlockingError(att, uuidGen)
	case "hook_success":
		return normalizeAttachmentHookSuccess(att, uuidGen)
	case "hook_additional_context":
		return normalizeAttachmentHookAdditionalContext(att, uuidGen)
	case "hook_stopped_continuation":
		return normalizeAttachmentHookStoppedContinuation(att, uuidGen)
	case "compaction_reminder":
		return normalizeAttachmentCompactionReminder(uuidGen)
	case "context_efficiency":
		return normalizeAttachmentContextEfficiency(opts, uuidGen)
	case "date_change":
		return normalizeAttachmentDateChange(att, uuidGen)
	case "ultrathink_effort":
		return normalizeAttachmentUltrathinkEffort(att, uuidGen)
	case "deferred_tools_delta":
		return normalizeAttachmentDeferredToolsDelta(att, uuidGen)
	case "agent_listing_delta":
		return normalizeAttachmentAgentListingDelta(att, uuidGen)
	case "mcp_instructions_delta":
		return normalizeAttachmentMCPInstructionsDelta(att, uuidGen)
	case "companion_intro":
		return normalizeAttachmentCompanionIntro(att, uuidGen)
	case "verify_plan_reminder":
		return normalizeAttachmentVerifyPlanReminder(opts, uuidGen)
	default:
		if isLegacyAttachmentType(head.Type) || isNoopAttachmentType(head.Type) {
			return nil, nil
		}
		return nil, nil
	}
}

func wrapMessagesInSystemReminder(msgs []types.Message) []types.Message {
	out := make([]types.Message, len(msgs))
	for i := range msgs {
		out[i] = ensureSystemReminderWrap(msgs[i])
	}
	return out
}

func createToolUseMetaMessage(toolName string, input map[string]any, uuidGen func() string) types.Message {
	text := fmt.Sprintf("Called the %s tool with the following input: %s", toolName, jsonStringify(input))
	return createUserMessageString(text, uuidGen(), "", true)
}

func createBashResultUserMessage(stdout, stderr string, interrupted bool, uuidGen func() string) types.Message {
	stdout = strings.TrimRight(strings.TrimLeft(stdout, "\n\t "), " \t")
	stderr = strings.TrimSpace(stderr)
	var parts []string
	if stdout != "" {
		parts = append(parts, stdout)
	}
	errMsg := stderr
	if interrupted {
		if errMsg != "" {
			errMsg += "\n"
		}
		errMsg += "<error>Command was aborted before completion</error>"
	}
	if errMsg != "" {
		parts = append(parts, errMsg)
	}
	body := strings.Join(parts, "\n")
	text := "Result of calling the " + bashToolName + " tool:\n" + body
	return createUserMessageString(text, uuidGen(), "", true)
}

func normalizeAttachmentDirectory(att json.RawMessage, uuidGen func() string) ([]types.Message, error) {
	var a struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(att, &a); err != nil {
		return nil, err
	}
	cmd := fmt.Sprintf("ls %s", shellQuoteSingle(a.Path))
	msgs := []types.Message{
		createToolUseMetaMessage(bashToolName, map[string]any{
			"command":     cmd,
			"description": fmt.Sprintf("Lists files in %s", a.Path),
		}, uuidGen),
		createBashResultUserMessage(a.Content, "", false, uuidGen),
	}
	return wrapMessagesInSystemReminder(msgs), nil
}

func normalizeAttachmentEditedTextFile(att json.RawMessage, uuidGen func() string) ([]types.Message, error) {
	var a struct {
		Filename string `json:"filename"`
		Snippet  string `json:"snippet"`
	}
	if err := json.Unmarshal(att, &a); err != nil {
		return nil, err
	}
	text := fmt.Sprintf(
		"Note: %s was modified, either by the user or by a linter. This change was intentional, so make sure to take it into account as you proceed (ie. don't revert it unless the user asks you to). Don't tell the user this, since they are already aware. Here are the relevant changes (shown with line numbers):\n%s",
		a.Filename, a.Snippet,
	)
	msgs := []types.Message{createUserMessageString(text, uuidGen(), "", true)}
	return wrapMessagesInSystemReminder(msgs), nil
}

func normalizeAttachmentFile(att json.RawMessage, uuidGen func() string) ([]types.Message, error) {
	var a struct {
		Filename  string          `json:"filename"`
		Content   json.RawMessage `json:"content"`
		Truncated bool            `json:"truncated"`
	}
	if err := json.Unmarshal(att, &a); err != nil {
		return nil, err
	}
	var fc struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(a.Content, &fc); err != nil {
		return nil, err
	}
	msgs := []types.Message{
		createToolUseMetaMessage(fileReadToolName, map[string]any{"file_path": a.Filename}, uuidGen),
		createFileReadResultUserMessage(a.Filename, a.Content, uuidGen),
	}
	if a.Truncated {
		note := fmt.Sprintf(
			"Note: The file %s was too large and has been truncated to the first %d lines. Don't tell the user about this truncation. Use %s to read more of the file if you need.",
			a.Filename, maxLinesToRead, fileReadToolName,
		)
		msgs = append(msgs, createUserMessageString(note, uuidGen(), "", true))
	}
	return wrapMessagesInSystemReminder(msgs), nil
}

func createFileReadResultUserMessage(filename string, fileContent json.RawMessage, uuidGen func() string) types.Message {
	// Mirror TS: jsonStringify(result.content) when not plain string
	var asStr string
	if err := json.Unmarshal(fileContent, &asStr); err == nil {
		text := "Result of calling the " + fileReadToolName + " tool:\n" + asStr
		return createUserMessageString(text, uuidGen(), "", true)
	}
	text := "Result of calling the " + fileReadToolName + " tool:\n" + string(fileContent)
	return createUserMessageString(text, uuidGen(), "", true)
}

func normalizeAttachmentCompactFileRef(att json.RawMessage, uuidGen func() string) ([]types.Message, error) {
	var a struct {
		Filename string `json:"filename"`
	}
	if err := json.Unmarshal(att, &a); err != nil {
		return nil, err
	}
	text := fmt.Sprintf(
		"Note: %s was read before the last conversation was summarized, but the contents are too large to include. Use %s tool if you need to access it.",
		a.Filename, fileReadToolName,
	)
	msgs := []types.Message{createUserMessageString(text, uuidGen(), "", true)}
	return wrapMessagesInSystemReminder(msgs), nil
}

func normalizeAttachmentPDFRef(att json.RawMessage, uuidGen func() string) ([]types.Message, error) {
	var a struct {
		Filename  string `json:"filename"`
		PageCount int    `json:"pageCount"`
		FileSize  int64  `json:"fileSize"`
	}
	if err := json.Unmarshal(att, &a); err != nil {
		return nil, err
	}
	text := fmt.Sprintf(
		"PDF file: %s (%d pages, %s). "+
			"This PDF is too large to read all at once. You MUST use the %s tool with the pages parameter "+
			"to read specific page ranges (e.g., pages: \"1-5\"). Do NOT call %s without the pages parameter "+
			"or it will fail. Start by reading the first few pages to understand the structure, then read more as needed. "+
			"Maximum 20 pages per request.",
		a.Filename, a.PageCount, formatFileSize(a.FileSize),
		fileReadToolName, fileReadToolName,
	)
	msgs := []types.Message{createUserMessageString(text, uuidGen(), "", true)}
	return wrapMessagesInSystemReminder(msgs), nil
}

func normalizeAttachmentSelectedLines(att json.RawMessage, uuidGen func() string) ([]types.Message, error) {
	var a struct {
		Filename  string `json:"filename"`
		LineStart int    `json:"lineStart"`
		LineEnd   int    `json:"lineEnd"`
		Content   string `json:"content"`
	}
	if err := json.Unmarshal(att, &a); err != nil {
		return nil, err
	}
	maxSelectionLength := 2000
	content := a.Content
	if len(content) > maxSelectionLength {
		content = content[:maxSelectionLength] + "\n... (truncated)"
	}
	text := fmt.Sprintf(
		"The user selected the lines %d to %d from %s:\n%s\n\nThis may or may not be related to the current task.",
		a.LineStart, a.LineEnd, a.Filename, content,
	)
	msgs := []types.Message{createUserMessageString(text, uuidGen(), "", true)}
	return wrapMessagesInSystemReminder(msgs), nil
}

func normalizeAttachmentOpenedFile(att json.RawMessage, uuidGen func() string) ([]types.Message, error) {
	var a struct {
		Filename string `json:"filename"`
	}
	if err := json.Unmarshal(att, &a); err != nil {
		return nil, err
	}
	text := fmt.Sprintf("The user opened the file %s in the IDE. This may or may not be related to the current task.", a.Filename)
	msgs := []types.Message{createUserMessageString(text, uuidGen(), "", true)}
	return wrapMessagesInSystemReminder(msgs), nil
}

func normalizeAttachmentPlanFile(att json.RawMessage, uuidGen func() string) ([]types.Message, error) {
	var a struct {
		PlanFilePath string `json:"planFilePath"`
		PlanContent  string `json:"planContent"`
	}
	if err := json.Unmarshal(att, &a); err != nil {
		return nil, err
	}
	text := fmt.Sprintf(
		"A plan file exists from plan mode at: %s\n\nPlan contents:\n\n%s\n\nIf this plan is relevant to the current work and not already complete, continue working on it.",
		a.PlanFilePath, a.PlanContent,
	)
	msgs := []types.Message{createUserMessageString(text, uuidGen(), "", true)}
	return wrapMessagesInSystemReminder(msgs), nil
}

func normalizeAttachmentInvokedSkills(att json.RawMessage, uuidGen func() string) ([]types.Message, error) {
	var a struct {
		Skills []struct {
			Name        string `json:"name"`
			Path        string `json:"path"`
			Content     string `json:"content"`
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
		lines = append(lines, fmt.Sprintf("### Skill: %s\nPath: %s\n\n%s", s.Name, s.Path, s.Content))
	}
	body := strings.Join(lines, "\n\n---\n\n")
	text := "The following skills were invoked in this session. Continue to follow these guidelines:\n\n" + body
	msgs := []types.Message{createUserMessageString(text, uuidGen(), "", true)}
	return wrapMessagesInSystemReminder(msgs), nil
}

func normalizeAttachmentTodoReminder(att json.RawMessage, uuidGen func() string) ([]types.Message, error) {
	var a struct {
		Content []struct {
			Status  string `json:"status"`
			Content string `json:"content"`
		} `json:"content"`
	}
	if err := json.Unmarshal(att, &a); err != nil {
		return nil, err
	}
	var items []string
	for i, todo := range a.Content {
		items = append(items, fmt.Sprintf("%d. [%s] %s", i+1, todo.Status, todo.Content))
	}
	todoItems := strings.Join(items, "\n")
	message := "The TodoWrite tool hasn't been used recently. If you're working on tasks that would benefit from tracking progress, consider using the TodoWrite tool to track progress. Also consider cleaning up the todo list if has become stale and no longer matches what you are working on. Only use it if it's relevant to the current work. This is just a gentle reminder - ignore if not applicable. Make sure that you NEVER mention this reminder to the user\n"
	if len(todoItems) > 0 {
		message += "\n\nHere are the existing contents of your todo list:\n\n[" + todoItems + "]"
	}
	msgs := []types.Message{createUserMessageString(message, uuidGen(), "", true)}
	return wrapMessagesInSystemReminder(msgs), nil
}

func normalizeAttachmentNestedMemory(att json.RawMessage, uuidGen func() string) ([]types.Message, error) {
	var a struct {
		Content struct {
			Path    string `json:"path"`
			Content string `json:"content"`
		} `json:"content"`
	}
	if err := json.Unmarshal(att, &a); err != nil {
		return nil, err
	}
	text := fmt.Sprintf("Contents of %s:\n\n%s", a.Content.Path, a.Content.Content)
	msgs := []types.Message{createUserMessageString(text, uuidGen(), "", true)}
	return wrapMessagesInSystemReminder(msgs), nil
}

func normalizeAttachmentRelevantMemories(att json.RawMessage, uuidGen func() string) ([]types.Message, error) {
	var a struct {
		Memories []struct {
			Header  *string `json:"header"`
			Path    string  `json:"path"`
			MtimeMs int64   `json:"mtimeMs"`
			Content string  `json:"content"`
		} `json:"memories"`
	}
	if err := json.Unmarshal(att, &a); err != nil {
		return nil, err
	}
	var msgs []types.Message
	for _, m := range a.Memories {
		header := ""
		if m.Header != nil && *m.Header != "" {
			header = *m.Header
		} else {
			header = fmt.Sprintf("Memory: %s:", m.Path)
		}
		text := header + "\n\n" + m.Content
		msgs = append(msgs, createUserMessageString(text, uuidGen(), "", true))
	}
	return wrapMessagesInSystemReminder(msgs), nil
}

func normalizeAttachmentTaskReminder(att json.RawMessage, uuidGen func() string) ([]types.Message, error) {
	var a struct {
		Content []struct {
			ID      int    `json:"id"`
			Status  string `json:"status"`
			Subject string `json:"subject"`
		} `json:"content"`
	}
	if err := json.Unmarshal(att, &a); err != nil {
		return nil, err
	}
	var items []string
	for _, task := range a.Content {
		items = append(items, fmt.Sprintf("#%d. [%s] %s", task.ID, task.Status, task.Subject))
	}
	taskItems := strings.Join(items, "\n")
	message := "The task tools haven't been used recently. If you're working on tasks that would benefit from tracking progress, consider using TaskCreate to add new tasks and TaskUpdate to update task status (set to in_progress when starting, completed when done). Also consider cleaning up the task list if it has become stale. Only use these if relevant to the current work. This is just a gentle reminder - ignore if not applicable. Make sure that you NEVER mention this reminder to the user\n"
	if len(taskItems) > 0 {
		message += "\n\nHere are the existing tasks:\n\n" + taskItems
	}
	msgs := []types.Message{createUserMessageString(message, uuidGen(), "", true)}
	return wrapMessagesInSystemReminder(msgs), nil
}
