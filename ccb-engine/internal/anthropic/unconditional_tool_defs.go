package anthropic

// Unconditional built-in tools (fixed getAllBaseTools slice, excluding feature spreads).
// Whenever a tool appears in embedded commands/data/tools_api.json (TS export: toolToAPISchema),
// InputSchema is taken from that file so Go JSON Schema validation matches TS Zod wire shapes.

func agentToolDefinition() ToolDefinition {
	return ToolDefinition{
		Name:        "Agent",
		Description: "Launch a new agent",
		InputSchema: mustExportInputSchema("Agent"),
	}
}

func taskOutputToolDefinition() ToolDefinition {
	return ToolDefinition{
		Name:        "TaskOutput",
		Description: "Read output/logs from a background task",
		InputSchema: mustExportInputSchema("TaskOutput"),
	}
}

func exitPlanModeToolDefinition() ToolDefinition {
	return ToolDefinition{
		Name:        "ExitPlanMode",
		Description: "Exit plan mode after completing the plan design",
		InputSchema: mustExportInputSchema("ExitPlanMode"),
	}
}

func notebookEditToolDefinition() ToolDefinition {
	return ToolDefinition{
		Name:        "NotebookEdit",
		Description: "Edit Jupyter notebook cells (.ipynb)",
		InputSchema: mustExportInputSchema("NotebookEdit"),
	}
}

func webFetchToolDefinition() ToolDefinition {
	return ToolDefinition{
		Name:        "WebFetch",
		Description: "Fetch and extract content from a URL",
		InputSchema: mustExportInputSchema("WebFetch"),
	}
}

func todoWriteToolDefinition() ToolDefinition {
	return ToolDefinition{
		Name:        "TodoWrite",
		Description: "Manage the session task checklist",
		InputSchema: mustExportInputSchema("TodoWrite"),
	}
}

func webSearchToolDefinition() ToolDefinition {
	return ToolDefinition{
		Name:        "WebSearch",
		Description: "Search the web for current information",
		InputSchema: mustExportInputSchema("WebSearch"),
	}
}

func taskStopToolDefinition() ToolDefinition {
	return ToolDefinition{
		Name:        "TaskStop",
		Description: "Stop a running background task",
		InputSchema: mustExportInputSchema("TaskStop"),
	}
}

func askUserQuestionToolDefinition() ToolDefinition {
	return ToolDefinition{
		Name:        "AskUserQuestion",
		Description: "Ask the user a multiple-choice question in the UI",
		InputSchema: mustExportInputSchema("AskUserQuestion"),
	}
}

func enterPlanModeToolDefinition() ToolDefinition {
	return ToolDefinition{
		Name:        "EnterPlanMode",
		Description: "Requests permission to enter plan mode for complex tasks requiring exploration and design",
		InputSchema: mustExportInputSchema("EnterPlanMode"),
	}
}

func sendMessageToolDefinition() ToolDefinition {
	return ToolDefinition{
		Name:        "SendMessage",
		Description: "Send a message to a teammate or broadcast",
		InputSchema: map[string]any{
			"type":     "object",
			"required": []string{"to", "message"},
			"properties": map[string]any{
				"to":      map[string]any{"type": "string"},
				"summary": map[string]any{"type": "string"},
				"message": map[string]any{
					"description": "Plain string or structured shutdown/plan message object",
				},
			},
		},
	}
}

func cronCreateToolDefinition() ToolDefinition {
	return ToolDefinition{
		Name:        "CronCreate",
		Description: "Schedule a recurring or one-shot prompt",
		InputSchema: mustExportInputSchema("CronCreate"),
	}
}

func cronDeleteToolDefinition() ToolDefinition {
	return ToolDefinition{
		Name:        "CronDelete",
		Description: "Cancel a scheduled cron job",
		InputSchema: mustExportInputSchema("CronDelete"),
	}
}

func cronListToolDefinition() ToolDefinition {
	return ToolDefinition{
		Name:        "CronList",
		Description: "List active cron jobs",
		InputSchema: mustExportInputSchema("CronList"),
	}
}

func sendUserMessageToolDefinition() ToolDefinition {
	return ToolDefinition{
		Name:        "SendUserMessage",
		Description: "Send a message to the user",
		InputSchema: map[string]any{
			"type":     "object",
			"required": []string{"message", "status"},
			"properties": map[string]any{
				"message":     map[string]any{"type": "string"},
				"attachments": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
				"status":      map[string]any{"type": "string", "enum": []string{"normal", "proactive"}},
			},
		},
	}
}

func briefAliasToolDefinition() ToolDefinition {
	d := sendUserMessageToolDefinition()
	d.Name = "Brief"
	d.Description = "Send a message to the user (legacy alias)"
	return d
}

func listMcpResourcesToolDefinition() ToolDefinition {
	return ToolDefinition{
		Name:        "ListMcpResourcesTool",
		Description: "List resources from connected MCP servers",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"server": map[string]any{"type": "string", "description": "Optional server name filter"},
			},
		},
	}
}

func readMcpResourceToolDefinition() ToolDefinition {
	return ToolDefinition{
		Name:        "ReadMcpResourceTool",
		Description: "Read a specific MCP resource by URI",
		InputSchema: map[string]any{
			"type":     "object",
			"required": []string{"server", "uri"},
			"properties": map[string]any{
				"server": map[string]any{"type": "string"},
				"uri":    map[string]any{"type": "string"},
			},
		},
	}
}

func ctxInspectToolDefinition() ToolDefinition {
	return ToolDefinition{
		Name:        "CtxInspect",
		Description: "Inspect the current context window contents and token usage. Shows token usage, message count, and a breakdown of what's consuming context space. Use this to understand your context budget before deciding whether to snip old messages or adjust your approach.",
		InputSchema: mustExportInputSchema("CtxInspect"),
	}
}

func listPeersToolDefinition() ToolDefinition {
	return ToolDefinition{
		Name:        "ListPeers",
		Description: "List connected peer sessions in the current workspace. Shows other Claude sessions that you can communicate with using the SendMessage tool.",
		InputSchema: mustExportInputSchema("ListPeers"),
	}
}

func monitorToolDefinition() ToolDefinition {
	return ToolDefinition{
		Name:        "Monitor",
		Description: "Monitor long-running task output by running shell commands continuously in the background. Use this for monitoring logs, watching file changes, or observing system processes. The monitor runs until manually stopped and streams output updates.",
		InputSchema: mustExportInputSchema("Monitor"),
	}
}

func pushNotificationToolDefinition() ToolDefinition {
	return ToolDefinition{
		Name:        "PushNotification",
		Description: "Send a push notification to the user's device. Use this to alert the user when important tasks complete, errors occur, or when you need their attention. Notifications appear outside the chat interface and can be seen even when the user is not actively looking at the conversation.",
		InputSchema: mustExportInputSchema("PushNotification"),
	}
}

func sendUserFileToolDefinition() ToolDefinition {
	return ToolDefinition{
		Name:        "SendUserFile",
		Description: "Send a file artifact to the user for download. Use this when you need to provide the user with files you've created or modified, such as generated reports, processed data, or configuration files. The file will be made available for download in the user interface.",
		InputSchema: mustExportInputSchema("SendUserFile"),
	}
}

func sleepToolDefinition() ToolDefinition {
	return ToolDefinition{
		Name:        "Sleep",
		Description: "Wait for a short duration before continuing.",
		InputSchema: mustExportInputSchema("Sleep"),
	}
}

func snipToolDefinition() ToolDefinition {
	return ToolDefinition{
		Name:        "Snip",
		Description: "Snip messages from conversation history to free up context window space. Snipped messages are replaced with a compact summary so you retain awareness of what happened without the full content. Use this when: your context is getting full and you need to make room, earlier messages contain large tool outputs you no longer need in full, you want to compact a long exploration sequence into a summary. Guidelines: only snip messages you're confident you won't need verbatim again, the summary replacement preserves key facts (file paths, decisions, errors found), you cannot un-snip — the original content is gone from context.",
		InputSchema: mustExportInputSchema("Snip"),
	}
}

func workflowToolDefinition() ToolDefinition {
	return ToolDefinition{
		Name:        "workflow",
		Description: "Execute a user-defined workflow script from .claude/workflows/. Workflows are YAML or Markdown files that define a sequence of steps for common development tasks. Specify the workflow name to execute (must match a file in .claude/workflows/). Optionally pass arguments that the workflow can use. Workflows run in the context of the current project.",
		InputSchema: mustExportInputSchema("workflow"),
	}
}
