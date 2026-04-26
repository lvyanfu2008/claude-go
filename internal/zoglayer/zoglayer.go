package zoglayer

import (
	"encoding/json"
)

// validators holds tools with a Zog-based input path (GO_TOOL_INPUT_VALIDATOR=zog).
// Tools not listed here fall back to JSON Schema + toolrefine in toolvalidator.
// Keys must match tool.Name() values at validation time. Bash is registered under
// both "Bash" and "BashZog" (bashzog handles its own path; "BashZog" is for the
// pre-rename tool alias).
var validators = map[string]func(json.RawMessage) error{
	"Bash":                validateBashZog,
	"BashZog":             validateBashZog,
	"EnterPlanMode":       validateEnterPlanMode,
	"ExitPlanMode":        validateExitPlanModeZog,
	"Grep":                validateGrepZog,
	"Read":                validateReadZog,
	"Write":               validateWriteZog,
	"Edit":                validateEditZog,
	"Glob":                validateGlobZog,
	"NotebookEdit":        validateNotebookEditZog,
	"TaskStop":            validateTaskStopZog,
	"TodoWrite":           validateTodoWriteZog,
	"WebFetch":            validateWebFetchZog,
	"WebSearch":           validateWebSearchZog,
	"Sleep":               validateSleepZog,
	"EnterWorktree":       validateEnterWorktreeZog,
	"ExitWorktree":        validateExitWorktreeZog,
	"Skill":               validateSkillZog,
	"workflow":            validateWorkflowZog,
	"TaskOutput":          validateTaskOutputZog,
	"ToolSearch":          validateToolSearchZog,
	"CronCreate":          validateCronCreateZog,
	"CronDelete":          validateCronDeleteZog,
	"CronList":            validateCronListZog,
	"SendMessage":         validateSendMessageZog,
	"SendUserMessage":     validateSendUserMessageZog,
	"ListMcpResourcesTool": validateListMcpResourcesZog,
	"ReadMcpResourceTool": validateReadMcpResourceZog,
	"TaskCreate":          validateTaskCreateZog,
	"TaskGet":             validateTaskGetZog,
	"TaskList":            validateTaskListZog,
	"TaskUpdate":          validateTaskUpdateZog,
	"CtxInspect":          validateCtxInspectZog,
	"ListPeers":           validateListPeersZog,
	"Monitor":             validateMonitorZog,
	"PushNotification":    validatePushNotificationZog,
	"SendUserFile":        validateSendUserFileZog,
	"Snip":                validateSnipZog,
	"PowerShell":          validatePowerShellZog,
	"Agent":               validateAgentZog,
	"AskUserQuestion":     validateAskUserQuestionZog,

	// Empty-object tools (no properties)
	"Config":             validateEmptyObjectZog,
	"Tungsten":           validateEmptyObjectZog,
	"SuggestBackgroundPR": validateEmptyObjectZog,
	"WebBrowser":         validateEmptyObjectZog,
	"OverflowTest":       validateEmptyObjectZog,
	"TerminalCapture":    validateEmptyObjectZog,
	"LSP":                validateEmptyObjectZog,
	"TeamCreate":         validateEmptyObjectZog,
	"TeamDelete":         validateEmptyObjectZog,
	"TeamAddMember":      validateTeamAddMemberZog,
	"TeamRemoveMember":   validateTeamRemoveMemberZog,
	"VerifyPlanExecution": validateEmptyObjectZog,
	"REPL":               validateEmptyObjectZog,
	"RemoteTrigger":      validateEmptyObjectZog,
	"SubscribePR":        validateEmptyObjectZog,
	"ReviewArtifact":     validateEmptyObjectZog,
	"TestingPermission":  validateEmptyObjectZog,
}

// Has reports whether toolName uses the Zog validator when Zog mode is on.
func Has(toolName string) bool {
	_, ok := validators[toolName]
	return ok
}

// Validate runs the Zog schema for toolName. Caller must ensure Has(toolName).
func Validate(toolName string, input json.RawMessage) error {
	f := validators[toolName]
	if f == nil {
		return nil
	}
	return f(input)
}
