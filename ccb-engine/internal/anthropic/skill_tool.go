package anthropic

import (
	"encoding/json"

	"goc/agents/builtin"
	"goc/modelenv"
	"goc/toolpool"
	"goc/types"
)

// SkillToolName matches src/tools/SkillTool/constants.ts SKILL_TOOL_NAME.
const SkillToolName = "Skill"

// skillToolDescriptionPrompt matches getPrompt() in src/tools/SkillTool/prompt.ts (must stay in sync with goc/commands/skill_listing_messages.go SkillToolDescriptionPrompt).
const skillToolDescriptionPrompt = `Execute a skill within the main conversation

When users ask you to perform tasks, check if any of the available skills match. Skills provide specialized capabilities and domain knowledge.

When users reference a "slash command" or "/<something>" (e.g., "/commit", "/review-pr"), they are referring to a skill. Use this tool to invoke it.

How to invoke:
- Use this tool with the skill name and optional arguments
- Examples:
  - ` + "`skill: \"pdf\"`" + ` - invoke the pdf skill
  - ` + "`skill: \"commit\", args: \"-m 'Fix bug'\"`" + ` - invoke with arguments
  - ` + "`skill: \"review-pr\", args: \"123\"`" + ` - invoke with arguments
  - ` + "`skill: \"ms-office-suite:pdf\"`" + ` - invoke using fully qualified name

Important:
- Available skills are listed in system-reminder messages in the conversation
- When a skill matches the user's request, this is a BLOCKING REQUIREMENT: invoke the relevant Skill tool BEFORE generating any other response about the task
- NEVER mention a skill without actually calling this tool
- Do not invoke a skill that is already running
- Do not use this tool for built-in CLI commands (like /help, /clear, etc.)
- If you see a <command-name> tag in the current conversation turn, the skill has ALREADY been loaded - follow the instructions directly instead of calling this tool again
`

// SkillToolDefinition matches the Skill tool registered for the Messages API (zod inputSchema in SkillTool.ts).
func SkillToolDefinition() ToolDefinition {
	return ToolDefinition{
		Name:        SkillToolName,
		Description: skillToolDescriptionPrompt,
		InputSchema: mustExportInputSchema(SkillToolName),
	}
}

// GouDemoDefaultTools is Skill + echo_stub (stub keeps engine wiring tests familiar).
func GouDemoDefaultTools() []ToolDefinition {
	out := make([]ToolDefinition, 0, 2)
	out = append(out, SkillToolDefinition())
	out = append(out, DefaultStubTools()...)
	return out
}

// GouDemoDefaultToolsJSON marshals [GouDemoDefaultTools].
func GouDemoDefaultToolsJSON() (json.RawMessage, error) {
	return json.Marshal(GouDemoDefaultTools())
}

// GouParityToolsJSON returns model-facing tools[]: Go tool wire (see [toolpool.AssembleToolPoolFromGoWire]) + built-in
// agent description patch, optional DiscoverSkills ([toolpool.DiscoverSkillsToolSpecFromEnv]), and [DefaultStubTools] stubs.
func GouParityToolsJSON() (json.RawMessage, error) {
	assembled, err := toolpool.AssembleToolPoolFromGoWire(types.EmptyToolPermissionContextData(), nil)
	if err != nil {
		return nil, err
	}
	assembled = toolpool.PatchAgentToolDescriptionWithBuiltins(assembled, builtin.GetBuiltInAgents(builtin.ConfigFromEnv(), builtin.GuideContext{}))
	if ds, ok := toolpool.DiscoverSkillsToolSpecFromEnv(); ok {
		assembled = toolpool.UniqByName(append([]types.ToolSpec{ds}, assembled...))
	}
	stubs, err := toolDefinitionsToSpecs(DefaultStubTools())
	if err != nil {
		return nil, err
	}
	assembled = toolpool.UniqByName(append(assembled, stubs...))
	opts := toolpool.DefaultToolToAPISchemaOptionsFromEnv()
	opts.Model = modelenv.ResolveWithFallback("claude-sonnet-4-20250514")
	return toolpool.MarshalToolsAPIDocumentDefinitionsWithOptions(assembled, opts)
}

func toolDefinitionsToSpecs(defs []ToolDefinition) ([]types.ToolSpec, error) {
	out := make([]types.ToolSpec, 0, len(defs))
	for _, d := range defs {
		raw, err := json.Marshal(d.InputSchema)
		if err != nil {
			return nil, err
		}
		out = append(out, types.ToolSpec{
			Name:            d.Name,
			Description:     d.Description,
			InputJSONSchema: raw,
		})
	}
	return out, nil
}
