package messagesapi

// Mirrors src/utils/messagesLiteralConstants.ts and src/constants/messages.ts.
const (
	syntheticModel            = "<synthetic>"
	noContentMessage          = "(no content)"
	toolReferenceTurnBoundary = "Tool loaded."
)

// Tool names (TS BashTool.name, FileReadTool, constants).
const (
	bashToolName           = "Bash"
	fileReadToolName       = "Read"
	fileWriteToolName      = "Write"
	exitPlanModeV2ToolName = "ExitPlanMode"
	fileEditToolName       = "Edit"
	askUserQuestionToolName = "AskUserQuestion"
	agentToolName          = "Agent"
	sendMessageToolName    = "SendMessage"
	taskOutputToolName     = "TaskOutput"
)

// snipNudgeText mirrors src/services/compact/snipCompact.ts SNIP_NUDGE_TEXT (empty when unset).
const snipNudgeText = ""

// API_IMAGE_MAX_BASE64_SIZE from src/constants/apiLimits.ts
const apiImageMaxBase64Size = 5 * 1024 * 1024

// PDF limits for user-facing error strings (src/constants/apiLimits.ts).
const (
	apiPDFMaxPages     = 100
	pdfTargetRawSize   = 20 * 1024 * 1024
	maxLinesToRead     = 2000 // src/tools/FileReadTool/prompt.ts MAX_LINES_TO_READ
)
