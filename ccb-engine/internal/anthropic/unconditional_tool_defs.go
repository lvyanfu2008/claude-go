package anthropic

// Unconditional built-in tools (fixed getAllBaseTools slice, excluding feature spreads).
// Names and input_schema align with TS zod schemas at a practical JSON-schema level.

func agentToolDefinition() ToolDefinition {
	return ToolDefinition{
		Name:        "Agent",
		Description: "Launch a new agent",
		InputSchema: map[string]any{
			"type":                 "object",
			"additionalProperties": false,
			"properties": map[string]any{
				"description": map[string]any{"type": "string", "description": "A short (3-5 word) description of the task"},
				"prompt":      map[string]any{"type": "string", "description": "The task for the agent to perform"},
				"subagent_type": map[string]any{
					"type":        "string",
					"description": "The type of specialized agent to use for this task",
				},
				"model": map[string]any{
					"type": "string",
					"enum": []string{"sonnet", "opus", "haiku"},
					"description": "Optional model override for this agent",
				},
				"run_in_background": map[string]any{"type": "boolean"},
				"name":              map[string]any{"type": "string"},
				"team_name":         map[string]any{"type": "string"},
				"mode":              map[string]any{"type": "string"},
				"isolation":         map[string]any{"type": "string", "enum": []string{"worktree", "remote"}},
				"cwd":               map[string]any{"type": "string"},
			},
			"required": []string{"description", "prompt"},
		},
	}
}

func taskOutputToolDefinition() ToolDefinition {
	return ToolDefinition{
		Name:        "TaskOutput",
		Description: "Read output/logs from a background task",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"task_id": map[string]any{"type": "string", "description": "The task ID to get output from"},
				"block":   map[string]any{"type": "boolean", "description": "Whether to wait for completion", "default": true},
				"timeout": map[string]any{"type": "number", "description": "Max wait time in ms", "default": 30000},
			},
			"required": []string{"task_id"},
		},
	}
}

func exitPlanModeToolDefinition() ToolDefinition {
	return ToolDefinition{
		Name:        "ExitPlanMode",
		Description: "Exit plan mode after completing the plan design",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"allowedPrompts": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"tool":   map[string]any{"type": "string", "enum": []string{"Bash"}},
							"prompt": map[string]any{"type": "string"},
						},
						"required": []string{"tool", "prompt"},
					},
				},
			},
		},
	}
}

func notebookEditToolDefinition() ToolDefinition {
	return ToolDefinition{
		Name:        "NotebookEdit",
		Description: "Edit Jupyter notebook cells (.ipynb)",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"notebook_path": map[string]any{"type": "string", "description": "Absolute path to the .ipynb file"},
				"cell_id":       map[string]any{"type": "string", "description": "Target cell id"},
				"new_source":    map[string]any{"type": "string", "description": "New cell source"},
				"cell_type": map[string]any{
					"type": "string", "enum": []string{"code", "markdown"},
				},
				"edit_mode": map[string]any{
					"type": "string", "enum": []string{"replace", "insert", "delete"},
				},
			},
			"required": []string{"notebook_path", "new_source"},
		},
	}
}

func webFetchToolDefinition() ToolDefinition {
	return ToolDefinition{
		Name:        "WebFetch",
		Description: "Fetch and extract content from a URL",
		InputSchema: map[string]any{
			"type":     "object",
			"required": []string{"url", "prompt"},
			"properties": map[string]any{
				"url":    map[string]any{"type": "string", "description": "The URL to fetch content from"},
				"prompt": map[string]any{"type": "string", "description": "The prompt to run on the fetched content"},
			},
		},
	}
}

func todoWriteToolDefinition() ToolDefinition {
	todoItem := map[string]any{
		"type":     "object",
		"required": []string{"content", "status", "activeForm"},
		"properties": map[string]any{
			"content":    map[string]any{"type": "string"},
			"status":     map[string]any{"type": "string", "enum": []string{"pending", "in_progress", "completed"}},
			"activeForm": map[string]any{"type": "string"},
		},
	}
	return ToolDefinition{
		Name:        "TodoWrite",
		Description: "Manage the session task checklist",
		InputSchema: map[string]any{
			"type":     "object",
			"required": []string{"todos"},
			"properties": map[string]any{
				"todos": map[string]any{"type": "array", "items": todoItem},
			},
		},
	}
}

func webSearchToolDefinition() ToolDefinition {
	return ToolDefinition{
		Name:        "WebSearch",
		Description: "Search the web for current information",
		InputSchema: map[string]any{
			"type":     "object",
			"required": []string{"query"},
			"properties": map[string]any{
				"query":           map[string]any{"type": "string", "minLength": 2},
				"allowed_domains": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
				"blocked_domains": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			},
		},
	}
}

func taskStopToolDefinition() ToolDefinition {
	return ToolDefinition{
		Name:        "TaskStop",
		Description: "Stop a running background task",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"task_id":  map[string]any{"type": "string"},
				"shell_id": map[string]any{"type": "string", "description": "Deprecated: use task_id"},
			},
		},
	}
}

func askUserQuestionToolDefinition() ToolDefinition {
	opt := map[string]any{
		"type":     "object",
		"required": []string{"label", "description"},
		"properties": map[string]any{
			"label":       map[string]any{"type": "string"},
			"description": map[string]any{"type": "string"},
			"preview":     map[string]any{"type": "string"},
		},
	}
	q := map[string]any{
		"type":     "object",
		"required": []string{"question", "header", "options"},
		"properties": map[string]any{
			"question":    map[string]any{"type": "string"},
			"header":      map[string]any{"type": "string"},
			"options":     map[string]any{"type": "array", "minItems": 2, "maxItems": 4, "items": opt},
			"multiSelect": map[string]any{"type": "boolean", "default": false},
		},
	}
	return ToolDefinition{
		Name:        "AskUserQuestion",
		Description: "Ask the user a multiple-choice question in the UI",
		InputSchema: map[string]any{
			"type":     "object",
			"required": []string{"questions"},
			"properties": map[string]any{
				"questions": map[string]any{"type": "array", "minItems": 1, "maxItems": 4, "items": q},
				"answers":   map[string]any{"type": "object", "additionalProperties": map[string]any{"type": "string"}},
				"annotations": map[string]any{
					"type": "object",
					"additionalProperties": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"preview": map[string]any{"type": "string"},
							"notes":   map[string]any{"type": "string"},
						},
					},
				},
				"metadata": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"source": map[string]any{"type": "string"},
					},
				},
			},
		},
	}
}

func enterPlanModeToolDefinition() ToolDefinition {
	return ToolDefinition{
		Name:        "EnterPlanMode",
		Description: "Requests permission to enter plan mode for complex tasks requiring exploration and design",
		InputSchema: map[string]any{
			"type":                 "object",
			"additionalProperties": false,
			"properties":           map[string]any{},
		},
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
		InputSchema: map[string]any{
			"type":     "object",
			"required": []string{"cron", "prompt"},
			"properties": map[string]any{
				"cron":      map[string]any{"type": "string"},
				"prompt":    map[string]any{"type": "string"},
				"recurring": map[string]any{"type": "boolean"},
				"durable":   map[string]any{"type": "boolean"},
			},
		},
	}
}

func cronDeleteToolDefinition() ToolDefinition {
	return ToolDefinition{
		Name:        "CronDelete",
		Description: "Cancel a scheduled cron job",
		InputSchema: map[string]any{
			"type":     "object",
			"required": []string{"id"},
			"properties": map[string]any{
				"id": map[string]any{"type": "string", "description": "Job ID returned by CronCreate"},
			},
		},
	}
}

func cronListToolDefinition() ToolDefinition {
	return ToolDefinition{
		Name:        "CronList",
		Description: "List active cron jobs",
		InputSchema: map[string]any{
			"type":                 "object",
			"additionalProperties": false,
			"properties":           map[string]any{},
		},
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
