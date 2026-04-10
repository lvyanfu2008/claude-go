package builtin

// Tool names mirror TS *Tool/prompt.ts and constants (AgentTool, Bash, Read, …).
const (
	ToolBash          = "Bash"
	ToolRead          = "Read"
	ToolWrite         = "Write"
	ToolEdit          = "Edit"
	ToolGlob          = "Glob"
	ToolGrep          = "Grep"
	ToolWebFetch      = "WebFetch"
	ToolWebSearch     = "WebSearch"
	ToolSendMessage   = "SendMessage"
	ToolAgent         = "Agent"
	ToolExitPlanMode  = "ExitPlanMode"
	ToolNotebookEdit  = "NotebookEdit"
)
