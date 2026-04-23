package bashzog

import (
	"fmt"

	"goc/commands"
)

// Model-facing Bash tool row (wire name [bashModelWireName]; [BashZogToolSpec] uses [ZogToolName]).
// input_schema mirrors claude-code src/tools/BashTool/BashTool.tsx (model-facing inputSchema):
//   - timeout.describe uses getMaxTimeoutMs from src/tools/BashTool/prompt.ts → [maxBashTimeoutMs]
//   - run_in_background omitted when CLAUDE_CODE_DISABLE_BACKGROUND_TASKS (same as TS module load).
// Schema shape: same pattern as toolpool native specs (map[string]any + json.Marshal).

const bashModelWireName = "Bash"

// Model-facing tool description: [GetSimplePrompt] (src/tools/BashTool/prompt.ts getSimplePrompt).

const bashInputSchemaFieldDescription = `Clear, concise description of what this command does in active voice. Never use words like "complex" or "risk" in the description - just describe what it does.

For simple commands (git, npm, standard CLI tools), keep it brief (5-10 words):
- ls → "List files in current directory"
- git status → "Show working tree status"
- npm install → "Install package dependencies"

For commands that are harder to parse at a glance (piped commands, obscure flags, etc.), add enough context to clarify what it does:
- find . -name "*.tmp" -exec rm {} \; → "Find and delete all .tmp files recursively"
- git reset --hard origin/main → "Discard all local changes and match remote main"
- curl -s url | jq '.data[]' → "Fetch JSON from URL and extract data array elements"`

func bashToolInputSchema() map[string]any {
	maxMs := maxBashTimeoutMs()
	props := map[string]any{
		"command": map[string]any{
			"type":        "string",
			"description": "The command to execute",
		},
		"timeout": map[string]any{
			"description": fmt.Sprintf("Optional timeout in milliseconds (max %d)", maxMs),
			"type":        "number",
		},
		"description": map[string]any{
			"description": bashInputSchemaFieldDescription,
			"type":        "string",
		},
		"dangerouslyDisableSandbox": map[string]any{
			"description": "Set this to true to dangerously override sandbox mode and run commands without sandboxing.",
			"type":        "boolean",
		},
	}
	if !commands.IsEnvTruthy("CLAUDE_CODE_DISABLE_BACKGROUND_TASKS") {
		props["run_in_background"] = map[string]any{
			"description": "Set to true to run this command in the background. Use Read to read the output later.",
			"type":        "boolean",
		}
	}
	return map[string]any{
		"$schema":              "https://json-schema.org/draft/2020-12/schema",
		"type":                 "object",
		"properties":           props,
		"required":             []string{"command"},
		"additionalProperties": false,
	}
}
