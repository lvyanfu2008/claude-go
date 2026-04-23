package anthropic

import (
	"encoding/json"

	"goc/agents/builtin"
	"goc/commands"
	"goc/modelenv"
	"goc/toolpool"
	"goc/types"
)

// SkillToolName matches src/tools/SkillTool/constants.ts SKILL_TOOL_NAME.
const SkillToolName = "Skill"

// SkillToolDefinition matches the Skill tool registered for the Messages API (zod inputSchema in SkillTool.ts).
func SkillToolDefinition() ToolDefinition {
	return ToolDefinition{
		Name:        SkillToolName,
		Description: commands.SkillToolDescriptionPrompt,
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
