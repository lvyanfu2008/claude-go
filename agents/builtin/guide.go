package builtin

import (
	"fmt"
	"strings"
)

const (
	claudeCodeDocsMapURL = "https://code.claude.com/docs/en/claude_code_docs_map.md"
	cdpDocsMapURL        = "https://platform.claude.com/llms.txt"
	defaultIssuesExplainer3P =
		"their cloud provider's documentation and support channels for Claude"
)

// GuideCommand is a slash / prompt command for guide context (TS commands filter).
type GuideCommand struct {
	Name        string
	Description string
	Type        string // e.g. "prompt"
	Source      string // e.g. "plugin" or ""
}

// GuideAgentRef is a non-built-in agent (TS activeAgents filter source !== 'built-in').
type GuideAgentRef struct {
	AgentType string
	WhenToUse string
}

// GuideMCPClient names a configured MCP server.
type GuideMCPClient struct {
	Name string
}

// GuideContext is optional context for the claude-code-guide system prompt (TS toolUseContext.options).
type GuideContext struct {
	Commands     []GuideCommand // TS `commands`; section 1 uses all with type === 'prompt' (includes plugins)
	CustomAgents []GuideAgentRef
	MCPClients   []GuideMCPClient
	SettingsJSON string // pretty-printed settings object, or empty
}

func claudeCodeGuideBasePrompt(embeddedSearch bool) string {
	localSearchHint := fmt.Sprintf("%s, %s, and %s", ToolRead, ToolGlob, ToolGrep)
	if embeddedSearch {
		localSearchHint = fmt.Sprintf("%s, `find`, and `grep`", ToolRead)
	}
	return fmt.Sprintf(`You are the Claude guide agent. Your primary responsibility is helping users understand and use Claude Code, the Claude Agent SDK, and the Claude API (formerly the Anthropic API) effectively.

**Your expertise spans three domains:**

1. **Claude Code** (the CLI tool): Installation, configuration, hooks, skills, MCP servers, keyboard shortcuts, IDE integrations, settings, and workflows.

2. **Claude Agent SDK**: A framework for building custom AI agents based on Claude Code technology. Available for Node.js/TypeScript and Python.

3. **Claude API**: The Claude API (formerly known as the Anthropic API) for direct model interaction, tool use, and integrations.

**Documentation sources:**

- **Claude Code docs** (%s): Fetch this for questions about the Claude Code CLI tool, including:
  - Installation, setup, and getting started
  - Hooks (pre/post command execution)
  - Custom skills
  - MCP server configuration
  - IDE integrations (VS Code, JetBrains)
  - Settings files and configuration
  - Keyboard shortcuts and hotkeys
  - Subagents and plugins
  - Sandboxing and security

- **Claude Agent SDK docs** (%s): Fetch this for questions about building agents with the SDK, including:
  - SDK overview and getting started (Python and TypeScript)
  - Agent configuration + custom tools
  - Session management and permissions
  - MCP integration in agents
  - Hosting and deployment
  - Cost tracking and context management
  Note: Agent SDK docs are part of the Claude API documentation at the same URL.

- **Claude API docs** (%s): Fetch this for questions about the Claude API (formerly the Anthropic API), including:
  - Messages API and streaming
  - Tool use (function calling) and Anthropic-defined tools (computer use, code execution, web search, text editor, bash, programmatic tool calling, tool search tool, context editing, Files API, structured outputs)
  - Vision, PDF support, and citations
  - Extended thinking and structured outputs
  - MCP connector for remote MCP servers
  - Cloud provider integrations (Bedrock, Vertex AI, Foundry)

**Approach:**
1. Determine which domain the user's question falls into
2. Use %s to fetch the appropriate docs map
3. Identify the most relevant documentation URLs from the map
4. Fetch the specific documentation pages
5. Provide clear, actionable guidance based on official documentation
6. Use %s if docs don't cover the topic
7. Reference local project files (CLAUDE.md, .claude/ directory) when relevant using %s

**Guidelines:**
- Always prioritize official documentation over assumptions
- Keep responses concise and actionable
- Include specific examples or code snippets when helpful
- Reference exact documentation URLs in your responses
- Help users discover features by proactively suggesting related commands, shortcuts, or capabilities

Complete the user's request by providing accurate, documentation-based guidance.`,
		claudeCodeDocsMapURL,
		cdpDocsMapURL,
		cdpDocsMapURL,
		ToolWebFetch,
		ToolWebSearch,
		localSearchHint,
	)
}

func guideFeedbackLine(cfg Config) string {
	if cfg.Using3PServices {
		exp := strings.TrimSpace(cfg.IssuesExplainer)
		if exp == "" {
			exp = defaultIssuesExplainer3P
		}
		return fmt.Sprintf("- When you cannot find an answer or the feature doesn't exist, direct the user to %s", exp)
	}
	return "- When you cannot find an answer or the feature doesn't exist, direct the user to use /feedback to report a feature request or bug"
}

// ClaudeCodeGuideSystemPrompt builds the claude-code-guide system prompt (TS getSystemPrompt on CLAUDE_CODE_GUIDE_AGENT).
func ClaudeCodeGuideSystemPrompt(cfg Config, ctx GuideContext) string {
	base := claudeCodeGuideBasePrompt(cfg.EmbeddedSearchTools)
	feedback := guideFeedbackLine(cfg)
	baseWithFeedback := base + "\n" + feedback

	var sections []string
	var customSkillLines []string
	for _, cmd := range ctx.Commands {
		if cmd.Type != "prompt" {
			continue
		}
		customSkillLines = append(customSkillLines, fmt.Sprintf("- /%s: %s", cmd.Name, cmd.Description))
	}
	if len(customSkillLines) > 0 {
		sections = append(sections, "**Available custom skills in this project:**\n"+strings.Join(customSkillLines, "\n"))
	}
	if len(ctx.CustomAgents) > 0 {
		var lines []string
		for _, a := range ctx.CustomAgents {
			lines = append(lines, fmt.Sprintf("- %s: %s", a.AgentType, a.WhenToUse))
		}
		sections = append(sections, "**Available custom agents configured:**\n"+strings.Join(lines, "\n"))
	}
	if len(ctx.MCPClients) > 0 {
		var lines []string
		for _, c := range ctx.MCPClients {
			lines = append(lines, "- "+c.Name)
		}
		sections = append(sections, "**Configured MCP servers:**\n"+strings.Join(lines, "\n"))
	}
	var pluginLines []string
	for _, cmd := range ctx.Commands {
		if cmd.Type == "prompt" && cmd.Source == "plugin" {
			pluginLines = append(pluginLines, fmt.Sprintf("- /%s: %s", cmd.Name, cmd.Description))
		}
	}
	if len(pluginLines) > 0 {
		sections = append(sections, "**Available plugin skills:**\n"+strings.Join(pluginLines, "\n"))
	}
	if strings.TrimSpace(ctx.SettingsJSON) != "" {
		sections = append(sections, "**User's settings.json:**\n```json\n"+strings.TrimSpace(ctx.SettingsJSON)+"\n```")
	}
	if len(sections) == 0 {
		return baseWithFeedback
	}
	return baseWithFeedback + `

---

# User's Current Configuration

The user has the following custom setup in their environment:

` + strings.Join(sections, "\n\n") + `

When answering questions, consider these configured features and proactively suggest them when relevant.`
}

func guideTools(embeddedSearch bool) []string {
	if embeddedSearch {
		return []string{ToolBash, ToolRead, ToolWebFetch, ToolWebSearch}
	}
	return []string{ToolGlob, ToolGrep, ToolRead, ToolWebFetch, ToolWebSearch}
}
