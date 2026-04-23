package toolpool

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"strings"

	"goc/ccb-engine/bashzog"
	"goc/ccb-engine/diaglog"
	"goc/commands"
	"goc/commands/featuregates"
	"goc/deferredtoolsdelta"
	"goc/tstenv"
	"goc/types"
	"goc/utils"
)

type goWireToolEntry struct {
	Name     string
	Required bool
	Enabled  func() bool
}

type nativeToolSpecProvider func(name string) (types.ToolSpec, bool, error)

func indexToolSpecsByName(specs []types.ToolSpec) map[string]types.ToolSpec {
	out := make(map[string]types.ToolSpec, len(specs))
	for _, s := range specs {
		out[s.Name] = s
	}
	return out
}

func mustToolSpecByName(index map[string]types.ToolSpec, name string) types.ToolSpec {
	spec, ok := index[name]
	if !ok {
		panic("toolpool: missing embedded schema for tool: " + name)
	}
	return spec
}

// buildGoWireToolSpecsFromExportSpecsWithProvider assembles runtime tools from
// Go registry entries.
//
// Resolution order per tool:
//  1. native provider (Go-owned schema/spec)
//
// Required tools are expected to be covered by the native provider; export
// fallback is not used in this path.
func buildGoWireToolSpecsFromExportSpecsWithProvider(specs []types.ToolSpec, provider nativeToolSpecProvider) []types.ToolSpec {
	_ = specs // kept for call-site compatibility during migration off export inputs
	out := make([]types.ToolSpec, 0, len(goWireBaseTools))
	for _, e := range goWireBaseTools {
		if e.Enabled != nil && !e.Enabled() {
			continue
		}
		if e.Required && provider == nil {
			panic("toolpool: required tool requires native provider: " + e.Name)
		}
		if provider != nil {
			nativeSpec, ok, err := provider(e.Name)
			if err != nil {
				if e.Required {
					panic("toolpool: build native tool spec for " + e.Name + ": " + err.Error())
				}
			} else if ok {
				out = append(out, nativeSpec)
				continue
			}
			if e.Required {
				panic("toolpool: required tool missing native spec: " + e.Name)
			}
			continue
		}
	}
	return out
}

func buildGoWireToolSpecsFromExportSpecs(specs []types.ToolSpec) []types.ToolSpec {
	return buildGoWireToolSpecsFromExportSpecsWithProvider(specs, nativeSpecFromGoProvider)
}

func mustMarshalJSONRaw(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic("toolpool: marshal native schema: " + err.Error())
	}
	return b
}

func nativeReadToolSpec() types.ToolSpec {
	schema := map[string]any{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type":    "object",
		"properties": map[string]any{
			"file_path": map[string]any{
				"type":        "string",
				"description": "The absolute path to the file to read",
			},
			"offset": map[string]any{
				"description": "The line number to start reading from. Only provide if the file is too large to read at once",
				"type":        "integer",
				"minimum":     0,
				"maximum":     int64(9007199254740991),
			},
			"limit": map[string]any{
				"description":      "The number of lines to read. Only provide if the file is too large to read at once.",
				"type":             "integer",
				"exclusiveMinimum": 0,
				"maximum":          int64(9007199254740991),
			},
			"pages": map[string]any{
				"description": "Page range for PDF files (e.g., \"1-5\", \"3\", \"10-20\"). Only applicable to PDF files. Maximum 20 pages per request.",
				"type":        "string",
			},
		},
		"required":             []string{"file_path"},
		"additionalProperties": false,
	}
	return types.ToolSpec{
		Name:            "Read",
		Description:     "Reads a file from the local filesystem. You can access any file directly by using this tool.\nAssume this tool is able to read all files on the machine. If the User provides a path to a file assume that path is valid. It is okay to read a file that does not exist; an error will be returned.\n\nUsage:\n- The file_path parameter must be an absolute path, not a relative path\n- By default, it reads up to 2000 lines starting from the beginning of the file\n- You can optionally specify a line offset and limit (especially handy for long files), but it's recommended to read the whole file by not providing these parameters\n- Results are returned using cat -n format, with line numbers starting at 1\n- This tool allows Claude Code to read images (eg PNG, JPG, etc). When reading an image file the contents are presented visually as Claude Code is a multimodal LLM.\n- This tool can read PDF files (.pdf). For large PDFs (more than 10 pages), you MUST provide the pages parameter to read specific page ranges (e.g., pages: \"1-5\"). Reading a large PDF without the pages parameter will fail. Maximum 20 pages per request.\n- This tool can read Jupyter notebooks (.ipynb files) and returns all cells with their outputs, combining code, text, and visualizations.\n- This tool can only read files, not directories. To read a directory, use an ls command via the Bash tool.\n- You will regularly be asked to read screenshots. If the user provides a path to a screenshot, ALWAYS use this tool to view the file at the path. This tool will work with all temporary file paths.\n- If you read a file that exists but has empty contents you will receive a system reminder warning in place of file contents.",
		InputJSONSchema: mustMarshalJSONRaw(schema),
	}
}

func nativeWriteToolSpec() types.ToolSpec {
	schema := map[string]any{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type":    "object",
		"properties": map[string]any{
			"file_path": map[string]any{
				"type":        "string",
				"description": "The absolute path to the file to write (must be absolute, not relative)",
			},
			"content": map[string]any{
				"type":        "string",
				"description": "The content to write to the file",
			},
		},
		"required":             []string{"file_path", "content"},
		"additionalProperties": false,
	}
	return types.ToolSpec{
		Name:            "Write",
		Description:     "Writes a file to the local filesystem.\n\nUsage:\n- This tool will overwrite the existing file if there is one at the provided path.\n- If this is an existing file, you MUST use the Read tool first to read the file's contents. This tool will fail if you did not read the file first.\n- Prefer the Edit tool for modifying existing files — it only sends the diff. Only use this tool to create new files or for complete rewrites.\n- NEVER create documentation files (*.md) or README files unless explicitly requested by the User.\n- Only use emojis if the user explicitly requests it. Avoid writing emojis to files unless asked.",
		InputJSONSchema: mustMarshalJSONRaw(schema),
	}
}

// getPreReadInstruction mirrors the TypeScript getPreReadInstruction function
func getPreReadInstruction() string {
	return "\n- You must use your `Read` tool at least once in the conversation before editing. This tool will error if you attempt an edit without reading the file. "
}

// getEditToolDescription generates the Edit tool description with dynamic prefix format
func getEditToolDescription() string {
	var prefixFormat string
	if isCompactLinePrefixEnabled() {
		prefixFormat = "line number + tab"
	} else {
		prefixFormat = "spaces + line number + arrow"
	}

	var minimalUniquenessHint string
	if os.Getenv("USER_TYPE") == "ant" {
		minimalUniquenessHint = "\n- Use the smallest old_string that's clearly unique — usually 2-4 adjacent lines is sufficient. Avoid including 10+ lines of context when less uniquely identifies the target."
	}

	return fmt.Sprintf(`Performs exact string replacements in files.

Usage:%s- When editing text from Read tool output, ensure you preserve the exact indentation (tabs/spaces) as it appears AFTER the line number prefix. The line number prefix format is: %s. Everything after that is the actual file content to match. Never include any part of the line number prefix in the old_string or new_string.
- ALWAYS prefer editing existing files in the codebase. NEVER write new files unless explicitly required.
- Only use emojis if the user explicitly requests it. Avoid adding emojis to files unless asked.
- The edit will FAIL if `+"`old_string`"+` is not unique in the file. Either provide a larger string with more surrounding context to make it unique or use `+"`replace_all`"+` to change every instance of `+"`old_string`"+`.%s
- Use `+"`replace_all`"+` for replacing and renaming strings across the file. This parameter is useful if you want to rename a variable for instance.`,
		getPreReadInstruction(),
		prefixFormat,
		minimalUniquenessHint)
}

// isCompactLinePrefixEnabled mirrors the TypeScript isCompactLinePrefixEnabled function
func isCompactLinePrefixEnabled() bool {
	val := strings.ToLower(strings.TrimSpace(os.Getenv("TENGU_COMPACT_LINE_PREFIX_KILLSWITCH")))
	truthy := val == "1" || val == "true" || val == "yes" || val == "on"
	return !truthy
}

func nativeEditToolSpec() types.ToolSpec {
	schema := map[string]any{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type":    "object",
		"properties": map[string]any{
			"file_path": map[string]any{
				"type":        "string",
				"description": "The absolute path to the file to modify",
			},
			"old_string": map[string]any{
				"type":        "string",
				"description": "The text to replace",
			},
			"new_string": map[string]any{
				"type":        "string",
				"description": "The text to replace it with (must be different from old_string)",
			},
			"replace_all": map[string]any{
				"description": "Replace all occurrences of old_string (default false)",
				"default":     false,
				"type":        "boolean",
			},
		},
		"required":             []string{"file_path", "old_string", "new_string"},
		"additionalProperties": false,
	}
	return types.ToolSpec{
		Name:            "Edit",
		Description:     getEditToolDescription(),
		InputJSONSchema: mustMarshalJSONRaw(schema),
	}
}

func nativeGlobToolSpec() types.ToolSpec {
	schema := map[string]any{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type":    "object",
		"properties": map[string]any{
			"pattern": map[string]any{
				"type":        "string",
				"description": "The glob pattern to match files against",
			},
			"path": map[string]any{
				"type":        "string",
				"description": "The directory to search in. If not specified, the current working directory will be used. IMPORTANT: Omit this field to use the default directory. DO NOT enter \"undefined\" or \"null\" - simply omit it for the default behavior. Must be a valid directory path if provided.",
			},
		},
		"required":             []string{"pattern"},
		"additionalProperties": false,
	}
	return types.ToolSpec{
		Name:            "Glob",
		Description:     "- Fast file pattern matching tool that works with any codebase size\n- Supports glob patterns like \"**/*.js\" or \"src/**/*.ts\"\n- Returns matching file paths sorted by modification time\n- Use this tool when you need to find files by name patterns\n- When you are doing an open ended search that may require multiple rounds of globbing and grepping, use the Agent tool instead",
		InputJSONSchema: mustMarshalJSONRaw(schema),
	}
}

func nativeGrepToolSpec() types.ToolSpec {
	schema := map[string]any{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type":    "object",
		"properties": map[string]any{
			"pattern": map[string]any{
				"type":        "string",
				"description": "The regular expression pattern to search for in file contents",
			},
			"path": map[string]any{
				"type":        "string",
				"description": "File or directory to search in (rg PATH). Defaults to current working directory.",
			},
			"glob": map[string]any{
				"type":        "string",
				"description": "Glob pattern to filter files (e.g. \"*.js\", \"*.{ts,tsx}\") - maps to rg --glob",
			},
			"output_mode": map[string]any{
				"type":        "string",
				"description": "Output mode: \"content\" shows matching lines (supports -A/-B/-C context, -n line numbers, head_limit), \"files_with_matches\" shows file paths (supports head_limit), \"count\" shows match counts (supports head_limit). Defaults to \"files_with_matches\".",
				"enum":        []string{"content", "files_with_matches", "count"},
			},
			"-B": map[string]any{
				"type":        "number",
				"description": "Number of lines to show before each match (rg -B). Requires output_mode: \"content\", ignored otherwise.",
			},
			"-A": map[string]any{
				"type":        "number",
				"description": "Number of lines to show after each match (rg -A). Requires output_mode: \"content\", ignored otherwise.",
			},
			"-C": map[string]any{
				"type":        "number",
				"description": "Alias for context.",
			},
			"context": map[string]any{
				"type":        "number",
				"description": "Number of lines to show before and after each match (rg -C). Requires output_mode: \"content\", ignored otherwise.",
			},
			"-n": map[string]any{
				"type":        "boolean",
				"description": "Show line numbers in output (rg -n). Requires output_mode: \"content\", ignored otherwise. Defaults to true.",
			},
			"-i": map[string]any{
				"type":        "boolean",
				"description": "Case insensitive search (rg -i)",
			},
			"type": map[string]any{
				"type":        "string",
				"description": "File type to search (rg --type). Common types: js, py, rust, go, java, etc. More efficient than include for standard file types.",
			},
			"head_limit": map[string]any{
				"type":        "number",
				"description": "Limit output to first N lines/entries, equivalent to \"| head -N\". Works across all output modes: content (limits output lines), files_with_matches (limits file paths), count (limits count entries). Defaults to 250 when unspecified. Pass 0 for unlimited (use sparingly — large result sets waste context).",
			},
			"offset": map[string]any{
				"type":        "number",
				"description": "Skip first N lines/entries before applying head_limit, equivalent to \"| tail -n +N | head -N\". Works across all output modes. Defaults to 0.",
			},
			"multiline": map[string]any{
				"type":        "boolean",
				"description": "Enable multiline mode where . matches newlines and patterns can span lines (rg -U --multiline-dotall). Default: false.",
			},
		},
		"required":             []string{"pattern"},
		"additionalProperties": false,
	}
	return types.ToolSpec{
		Name:            "Grep",
		Description:     "A powerful search tool built on ripgrep\n\n  Usage:\n  - ALWAYS use Grep for search tasks. NEVER invoke `grep` or `rg` as a Bash command. The Grep tool has been optimized for correct permissions and access.\n  - Supports full regex syntax (e.g., \"log.*Error\", \"function\\s+\\w+\")\n  - Filter files with glob parameter (e.g., \"*.js\", \"**/*.tsx\") or type parameter (e.g., \"js\", \"py\", \"rust\")\n  - Output modes: \"content\" shows matching lines, \"files_with_matches\" shows only file paths (default), \"count\" shows match counts\n  - Use Agent tool for open-ended searches requiring multiple rounds\n  - Pattern syntax: Uses ripgrep (not grep) - literal braces need escaping (use `interface\\{\\}` to find `interface{}` in Go code)\n  - Multiline matching: By default patterns match within single lines only. For cross-line patterns like `struct \\{[\\s\\S]*?field`, use `multiline: true`\n",
		InputJSONSchema: mustMarshalJSONRaw(schema),
	}
}

func nativeNotebookEditToolSpec() types.ToolSpec {
	schema := map[string]any{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type":    "object",
		"properties": map[string]any{
			"notebook_path": map[string]any{
				"type":        "string",
				"description": "The absolute path to the Jupyter notebook file to edit (must be absolute, not relative)",
			},
			"cell_id": map[string]any{
				"type":        "string",
				"description": "The ID of the cell to edit. When inserting a new cell, the new cell will be inserted after the cell with this ID, or at the beginning if not specified.",
			},
			"new_source": map[string]any{
				"type":        "string",
				"description": "The new source for the cell",
			},
			"cell_type": map[string]any{
				"type":        "string",
				"description": "The type of the cell (code or markdown). If not specified, it defaults to the current cell type. If using edit_mode=insert, this is required.",
				"enum":        []string{"code", "markdown"},
			},
			"edit_mode": map[string]any{
				"type":        "string",
				"description": "The type of edit to make (replace, insert, delete). Defaults to replace.",
				"enum":        []string{"replace", "insert", "delete"},
			},
		},
		"required":             []string{"notebook_path", "new_source"},
		"additionalProperties": false,
	}
	return types.ToolSpec{
		Name:            "NotebookEdit",
		Description:     "Edit Jupyter notebook cells (.ipynb)",
		InputJSONSchema: mustMarshalJSONRaw(schema),
	}
}

func nativeTaskStopToolSpec() types.ToolSpec {
	schema := map[string]any{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type":    "object",
		"properties": map[string]any{
			"task_id": map[string]any{
				"type":        "string",
				"description": "The ID of the background task to stop",
			},
			"shell_id": map[string]any{
				"type":        "string",
				"description": "Deprecated: use task_id instead",
			},
		},
		"additionalProperties": false,
	}
	return types.ToolSpec{
		Name:            "TaskStop",
		Description:     "Stop a running background task by ID.",
		InputJSONSchema: mustMarshalJSONRaw(schema),
	}
}

func nativeTodoWriteToolSpec() types.ToolSpec {
	schema := map[string]any{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type":    "object",
		"properties": map[string]any{
			"todos": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"content": map[string]any{
							"type":      "string",
							"minLength": 1,
						},
						"status": map[string]any{
							"type": "string",
							"enum": []string{"pending", "in_progress", "completed"},
						},
						"activeForm": map[string]any{
							"type":      "string",
							"minLength": 1,
						},
					},
					"required":             []string{"content", "status", "activeForm"},
					"additionalProperties": false,
				},
				"description": "The updated todo list",
			},
		},
		"required":             []string{"todos"},
		"additionalProperties": false,
	}
	return types.ToolSpec{
		Name:            "TodoWrite",
		Description:     "Manage structured task lists for the current coding session.",
		InputJSONSchema: mustMarshalJSONRaw(schema),
	}
}

func nativeWebFetchToolSpec() types.ToolSpec {
	schema := map[string]any{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type":    "object",
		"properties": map[string]any{
			"url": map[string]any{
				"type":        "string",
				"format":      "uri",
				"description": "The URL to fetch content from",
			},
			"prompt": map[string]any{
				"type":        "string",
				"description": "The prompt to run on the fetched content",
			},
		},
		"required":             []string{"url", "prompt"},
		"additionalProperties": false,
	}
	return types.ToolSpec{
		Name:            "WebFetch",
		Description:     "Fetch and analyze web content for a given URL.",
		InputJSONSchema: mustMarshalJSONRaw(schema),
	}
}

func nativeWebSearchToolSpec() types.ToolSpec {
	schema := map[string]any{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type":    "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type":        "string",
				"minLength":   2,
				"description": "The search query to use",
			},
			"allowed_domains": map[string]any{
				"type":        "array",
				"description": "Only include search results from these domains",
				"items": map[string]any{
					"type": "string",
				},
			},
			"blocked_domains": map[string]any{
				"type":        "array",
				"description": "Never include search results from these domains",
				"items": map[string]any{
					"type": "string",
				},
			},
		},
		"required":             []string{"query"},
		"additionalProperties": false,
	}
	return types.ToolSpec{
		Name:            "WebSearch",
		Description:     "Search the web for up-to-date information.",
		InputJSONSchema: mustMarshalJSONRaw(schema),
	}
}

func nativeEnterPlanModeToolSpec() types.ToolSpec {
	schema := map[string]any{
		"$schema":              "https://json-schema.org/draft/2020-12/schema",
		"type":                 "object",
		"properties":           map[string]any{},
		"additionalProperties": false,
	}
	return types.ToolSpec{
		Name:            "EnterPlanMode",
		Description:     "Enter plan mode for complex implementation tasks.",
		InputJSONSchema: mustMarshalJSONRaw(schema),
	}
}

func nativeExitPlanModeToolSpec() types.ToolSpec {
	schema := map[string]any{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type":    "object",
		"properties": map[string]any{
			"allowedPrompts": map[string]any{
				"type":        "array",
				"description": "Prompt-based permissions needed to implement the plan. These describe categories of actions rather than specific commands.",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"tool": map[string]any{
							"type":        "string",
							"enum":        []string{"Bash"},
							"description": "The tool this prompt applies to",
						},
						"prompt": map[string]any{
							"type":        "string",
							"description": "Semantic description of the action, e.g. \"run tests\", \"install dependencies\"",
						},
					},
					"required":             []string{"tool", "prompt"},
					"additionalProperties": false,
				},
			},
		},
		"additionalProperties": map[string]any{},
	}
	return types.ToolSpec{
		Name:            "ExitPlanMode",
		Description:     "Exit plan mode and request user approval.",
		InputJSONSchema: mustMarshalJSONRaw(schema),
	}
}

func nativeCronCreateToolSpec() types.ToolSpec {
	schema := map[string]any{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type":    "object",
		"properties": map[string]any{
			"cron": map[string]any{
				"type":        "string",
				"description": "Standard 5-field cron expression in local time: \"M H DoM Mon DoW\" (e.g. \"*/5 * * * *\" = every 5 minutes, \"30 14 28 2 *\" = Feb 28 at 2:30pm local once).",
			},
			"prompt": map[string]any{
				"type":        "string",
				"description": "The prompt to enqueue at each fire time.",
			},
			"recurring": map[string]any{
				"type":        "boolean",
				"description": "true (default) = fire on every cron match until deleted or auto-expired after 7 days. false = fire once at the next match, then auto-delete. Use false for \"remind me at X\" one-shot requests with pinned minute/hour/dom/month.",
			},
			"durable": map[string]any{
				"type":        "boolean",
				"description": "true = persist to .claude/scheduled_tasks.json and survive restarts. false (default) = in-memory only, dies when this Claude session ends. Use true only when the user asks the task to survive across sessions.",
			},
		},
		"required":             []string{"cron", "prompt"},
		"additionalProperties": false,
	}
	return types.ToolSpec{
		Name:            "CronCreate",
		Description:     "Schedule prompts using cron expressions.",
		InputJSONSchema: mustMarshalJSONRaw(schema),
	}
}

func nativeCronDeleteToolSpec() types.ToolSpec {
	schema := map[string]any{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type":    "object",
		"properties": map[string]any{
			"id": map[string]any{
				"type":        "string",
				"description": "Job ID returned by CronCreate.",
			},
		},
		"required":             []string{"id"},
		"additionalProperties": false,
	}
	return types.ToolSpec{
		Name:            "CronDelete",
		Description:     "Cancel a scheduled cron job.",
		InputJSONSchema: mustMarshalJSONRaw(schema),
	}
}

func nativeCronListToolSpec() types.ToolSpec {
	schema := map[string]any{
		"$schema":              "https://json-schema.org/draft/2020-12/schema",
		"type":                 "object",
		"properties":           map[string]any{},
		"additionalProperties": false,
	}
	return types.ToolSpec{
		Name:            "CronList",
		Description:     "List all scheduled cron jobs.",
		InputJSONSchema: mustMarshalJSONRaw(schema),
	}
}

func nativeAskUserQuestionToolSpec() types.ToolSpec {
	schema := map[string]any{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type":    "object",
		"properties": map[string]any{
			"questions": map[string]any{
				"type":     "array",
				"minItems": 1,
				"maxItems": 4,
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"question": map[string]any{
							"type":        "string",
							"description": "The complete question to ask the user. Should be clear, specific, and end with a question mark. Example: \"Which library should we use for date formatting?\" If multiSelect is true, phrase it accordingly, e.g. \"Which features do you want to enable?\"",
						},
						"header": map[string]any{
							"type":        "string",
							"description": "Very short label displayed as a chip/tag (max 12 chars). Examples: \"Auth method\", \"Library\", \"Approach\".",
						},
						"options": map[string]any{
							"type":        "array",
							"minItems":    2,
							"maxItems":    4,
							"description": "The available choices for this question. Must have 2-4 options. Each option should be a distinct, mutually exclusive choice (unless multiSelect is enabled). There should be no 'Other' option, that will be provided automatically.",
							"items": map[string]any{
								"type": "object",
								"properties": map[string]any{
									"label": map[string]any{
										"type":        "string",
										"description": "The display text for this option that the user will see and select. Should be concise (1-5 words) and clearly describe the choice.",
									},
									"description": map[string]any{
										"type":        "string",
										"description": "Explanation of what this option means or what will happen if chosen. Useful for providing context about trade-offs or implications.",
									},
									"preview": map[string]any{
										"type":        "string",
										"description": "Optional preview content rendered when this option is focused. Use for mockups, code snippets, or visual comparisons that help users compare options. See the tool description for the expected content format.",
									},
								},
								"required":             []string{"label", "description"},
								"additionalProperties": false,
							},
						},
						"multiSelect": map[string]any{
							"type":        "boolean",
							"default":     false,
							"description": "Set to true to allow the user to select multiple options instead of just one. Use when choices are not mutually exclusive.",
						},
					},
					"required":             []string{"question", "header", "options", "multiSelect"},
					"additionalProperties": false,
				},
				"description": "Questions to ask the user (1-4 questions)",
			},
			"answers": map[string]any{
				"type":        "object",
				"description": "User answers collected by the permission component",
				"propertyNames": map[string]any{
					"type": "string",
				},
				"additionalProperties": map[string]any{
					"type": "string",
				},
			},
			"annotations": map[string]any{
				"type":        "object",
				"description": "Optional per-question annotations from the user (e.g., notes on preview selections). Keyed by question text.",
				"propertyNames": map[string]any{
					"type": "string",
				},
				"additionalProperties": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"preview": map[string]any{
							"type":        "string",
							"description": "The preview content of the selected option, if the question used previews.",
						},
						"notes": map[string]any{
							"type":        "string",
							"description": "Free-text notes the user added to their selection.",
						},
					},
					"additionalProperties": false,
				},
			},
			"metadata": map[string]any{
				"type":        "object",
				"description": "Optional metadata for tracking and analytics purposes. Not displayed to user.",
				"properties": map[string]any{
					"source": map[string]any{
						"type":        "string",
						"description": "Optional identifier for the source of this question (e.g., \"remember\" for /remember command). Used for analytics tracking.",
					},
				},
				"additionalProperties": false,
			},
		},
		"required":             []string{"questions"},
		"additionalProperties": false,
	}
	return types.ToolSpec{
		Name:            "AskUserQuestion",
		Description:     "Ask the user clarifying multiple-choice questions during execution.",
		InputJSONSchema: mustMarshalJSONRaw(schema),
	}
}

func nativeEnterWorktreeToolSpec() types.ToolSpec {
	schema := map[string]any{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type":    "object",
		"properties": map[string]any{
			"name": map[string]any{
				"type":        "string",
				"description": "Optional name for the worktree. Each \"/\"-separated segment may contain only letters, digits, dots, underscores, and dashes; max 64 chars total. A random name is generated if not provided.",
			},
		},
		"additionalProperties": false,
	}
	return types.ToolSpec{
		Name:            "EnterWorktree",
		Description:     "Enter an isolated worktree session.",
		InputJSONSchema: mustMarshalJSONRaw(schema),
	}
}

func nativeExitWorktreeToolSpec() types.ToolSpec {
	schema := map[string]any{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type":    "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type":        "string",
				"enum":        []string{"keep", "remove"},
				"description": "\"keep\" leaves the worktree and branch on disk; \"remove\" deletes both.",
			},
			"discard_changes": map[string]any{
				"type":        "boolean",
				"description": "Required true when action is \"remove\" and the worktree has uncommitted files or unmerged commits. The tool will refuse and list them otherwise.",
			},
		},
		"required":             []string{"action"},
		"additionalProperties": false,
	}
	return types.ToolSpec{
		Name:            "ExitWorktree",
		Description:     "Exit current worktree session.",
		InputJSONSchema: mustMarshalJSONRaw(schema),
	}
}

func nativeSkillToolSpec() types.ToolSpec {
	schema := map[string]any{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type":    "object",
		"properties": map[string]any{
			"skill": map[string]any{
				"type":        "string",
				"description": "The skill name. E.g., \"commit\", \"review-pr\", or \"pdf\"",
			},
			"args": map[string]any{
				"type":        "string",
				"description": "Optional arguments for the skill",
			},
		},
		"required":             []string{"skill"},
		"additionalProperties": false,
	}
	return types.ToolSpec{
		Name:            "Skill",
		Description:     commands.SkillToolDescriptionPrompt,
		InputJSONSchema: mustMarshalJSONRaw(schema),
	}
}

// nativeWorkflowToolSpec mirrors claude-code-best/src/tools/WorkflowTool/WorkflowTool.ts inputSchema
// (workflow required, args optional). Used for LLM tool schema parity with the TypeScript Zod shape.
func nativeWorkflowToolSpec() types.ToolSpec {
	schema := map[string]any{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type":    "object",
		"properties": map[string]any{
			"workflow": map[string]any{
				"type":        "string",
				"description": "Name of the workflow to execute",
			},
			"args": map[string]any{
				"type":        "string",
				"description": "Arguments to pass to the workflow",
			},
		},
		"required":             []string{"workflow"},
		"additionalProperties": false,
	}
	return types.ToolSpec{
		Name: "workflow",
		Description: `Use the Workflow tool to execute user-defined workflow scripts located in .claude/workflows/. Workflows are YAML or Markdown files that define a sequence of steps for common development tasks.

Guidelines:
- Specify the workflow name to execute (must match a file in .claude/workflows/)
- Optionally pass arguments that the workflow can use
- Workflows run in the context of the current project`,
		InputJSONSchema: mustMarshalJSONRaw(schema),
	}
}

func nativeTaskOutputToolSpec() types.ToolSpec {
	schema := map[string]any{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type":    "object",
		"properties": map[string]any{
			"task_id": map[string]any{
				"type":        "string",
				"description": "The task ID to get output from",
			},
			"block": map[string]any{
				"type":        "boolean",
				"default":     true,
				"description": "Whether to wait for completion",
			},
			"timeout": map[string]any{
				"type":        "number",
				"default":     30000,
				"minimum":     0,
				"maximum":     600000,
				"description": "Max wait time in ms",
			},
		},
		"required":             []string{"task_id", "block", "timeout"},
		"additionalProperties": false,
	}
	return types.ToolSpec{
		Name:            "TaskOutput",
		Description:     "Read output from a running or completed background task.",
		InputJSONSchema: mustMarshalJSONRaw(schema),
	}
}

func nativeToolSearchToolSpec() types.ToolSpec {
	schema := map[string]any{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type":    "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "Query to find deferred tools. Use \"select:<tool_name>\" for direct selection, or keywords to search.",
			},
			"max_results": map[string]any{
				"type":        "number",
				"default":     5,
				"description": "Maximum number of results to return (default: 5)",
			},
		},
		"required":             []string{"query", "max_results"},
		"additionalProperties": false,
	}
	return types.ToolSpec{
		Name:            "ToolSearch",
		Description:     deferredtoolsdelta.ToolSearchToolDescription(),
		InputJSONSchema: mustMarshalJSONRaw(schema),
	}
}

func nativeAgentToolSpec() types.ToolSpec {
	properties := map[string]any{
		"description": map[string]any{
			"type":        "string",
			"description": "A short (3-5 word) description of the task",
		},
		"prompt": map[string]any{
			"type":        "string",
			"description": "The task for the agent to perform",
		},
		"subagent_type": map[string]any{
			"type":        "string",
			"description": "The type of specialized agent to use for this task",
		},
		"model": map[string]any{
			"type":        "string",
			"description": "Optional model override for this agent. Takes precedence over the agent definition's model frontmatter. If omitted, uses the agent definition's model, or inherits from the parent.",
			"enum":        []string{"sonnet", "opus", "haiku"},
		},
		"isolation": map[string]any{
			"type":        "string",
			"description": "Isolation mode. \"worktree\" creates a temporary git worktree so the agent works on an isolated copy of the repo.",
			"enum":        []string{"worktree"},
		},
		"cwd": map[string]any{
			"type":        "string",
			"description": "Absolute path to run the agent in. Overrides the working directory for all filesystem and shell operations within this agent. Mutually exclusive with isolation: \"worktree\".",
		},
	}

	// Conditionally add run_in_background parameter based on environment and feature flags.
	// Mirrors src/tools/AgentTool/AgentTool.tsx inputSchema: include only if background
	// tasks are not disabled and isForkSubagentEnabled() is false (forkSubagent.ts).
	backgroundDisabled := utils.IsEnvTruthy("CLAUDE_CODE_DISABLE_BACKGROUND_TASKS")
	forkEnabled := commands.ForkSubagentEnabled(commands.GouDemoSystemOpts{})
	includeRunInBackground := !backgroundDisabled && !forkEnabled
	if utils.IsEnvTruthy("CLAUDE_CODE_GO_DEBUG_AGENT_TOOL_SCHEMA") {
		diaglog.Line("[agent-tool-schema] CLAUDE_CODE_DISABLE_BACKGROUND_TASKS=%q backgroundDisabled=%v forkEnabled=%v run_in_background_in_schema=%v",
			strings.TrimSpace(os.Getenv("CLAUDE_CODE_DISABLE_BACKGROUND_TASKS")),
			backgroundDisabled, forkEnabled, includeRunInBackground)
	}
	if includeRunInBackground {
		properties["run_in_background"] = map[string]any{
			"type":        "boolean",
			"description": "Set to true to run this agent in the background. You will be notified when it completes.",
		}
	}

	schema := map[string]any{
		"$schema":              "https://json-schema.org/draft/2020-12/schema",
		"type":                 "object",
		"properties":           properties,
		"required":             []string{"description", "prompt"},
		"additionalProperties": false,
	}
	return types.ToolSpec{
		Name:            "Agent",
		Description:     AgentToolDescription(),
		InputJSONSchema: mustMarshalJSONRaw(schema),
	}
}

func nativeEmptyObjectSchemaToolSpec(name, fallbackDescription string) types.ToolSpec {
	schema := map[string]any{
		"$schema":              "https://json-schema.org/draft/2020-12/schema",
		"type":                 "object",
		"properties":           map[string]any{},
		"additionalProperties": false,
	}
	return types.ToolSpec{
		Name:            name,
		Description:     fallbackDescription,
		InputJSONSchema: mustMarshalJSONRaw(schema),
	}
}

func nativeSendMessageToolSpec() types.ToolSpec {
	schema := map[string]any{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type":    "object",
		"properties": map[string]any{
			"to":      map[string]any{"type": "string"},
			"message": map[string]any{"type": "string"},
		},
		"required":             []string{"to", "message"},
		"additionalProperties": false,
	}
	return types.ToolSpec{
		Name:            "SendMessage",
		Description:     "Send a message to a teammate or broadcast target.",
		InputJSONSchema: mustMarshalJSONRaw(schema),
	}
}

func nativeSendUserMessageToolSpec() types.ToolSpec {
	schema := map[string]any{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type":    "object",
		"properties": map[string]any{
			"message": map[string]any{"type": "string"},
			"attachments": map[string]any{
				"type":  "array",
				"items": map[string]any{"type": "string"},
			},
			"status": map[string]any{"type": "string", "enum": []string{"normal", "proactive"}},
		},
		"required":             []string{"message", "status"},
		"additionalProperties": false,
	}
	return types.ToolSpec{
		Name:            "SendUserMessage",
		Description:     "Send a user-visible message with optional attachments.",
		InputJSONSchema: mustMarshalJSONRaw(schema),
	}
}

func nativeListMcpResourcesToolSpec() types.ToolSpec {
	schema := map[string]any{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type":    "object",
		"properties": map[string]any{
			"server": map[string]any{"type": "string"},
		},
		"additionalProperties": false,
	}
	return types.ToolSpec{
		Name:            "ListMcpResourcesTool",
		Description:     "List resources from connected MCP servers.",
		InputJSONSchema: mustMarshalJSONRaw(schema),
	}
}

func nativeReadMcpResourceToolSpec() types.ToolSpec {
	schema := map[string]any{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type":    "object",
		"properties": map[string]any{
			"server": map[string]any{"type": "string"},
			"uri":    map[string]any{"type": "string"},
		},
		"required":             []string{"server", "uri"},
		"additionalProperties": false,
	}
	return types.ToolSpec{
		Name:            "ReadMcpResourceTool",
		Description:     "Read a specific resource from an MCP server.",
		InputJSONSchema: mustMarshalJSONRaw(schema),
	}
}

func nativeTaskCreateToolSpec() types.ToolSpec {
	schema := map[string]any{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type":    "object",
		"properties": map[string]any{
			"subject":     map[string]any{"type": "string"},
			"description": map[string]any{"type": "string"},
			"activeForm":  map[string]any{"type": "string"},
			"metadata":    map[string]any{"type": "object", "additionalProperties": map[string]any{}},
		},
		"required":             []string{"subject"},
		"additionalProperties": false,
	}
	return types.ToolSpec{
		Name:            "TaskCreate",
		Description:     "Create a task in Todo v2.",
		InputJSONSchema: mustMarshalJSONRaw(schema),
	}
}

func nativeTaskGetToolSpec() types.ToolSpec {
	schema := map[string]any{
		"$schema":              "https://json-schema.org/draft/2020-12/schema",
		"type":                 "object",
		"properties":           map[string]any{"taskId": map[string]any{"type": "string"}},
		"required":             []string{"taskId"},
		"additionalProperties": false,
	}
	return types.ToolSpec{
		Name:            "TaskGet",
		Description:     "Get a task by ID in Todo v2.",
		InputJSONSchema: mustMarshalJSONRaw(schema),
	}
}

func nativeTaskListToolSpec() types.ToolSpec {
	return nativeEmptyObjectSchemaToolSpec("TaskList", "List tasks in Todo v2.")
}

func nativeTaskUpdateToolSpec() types.ToolSpec {
	schema := map[string]any{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type":    "object",
		"properties": map[string]any{
			"taskId":       map[string]any{"type": "string"},
			"subject":      map[string]any{"type": "string"},
			"description":  map[string]any{"type": "string"},
			"activeForm":   map[string]any{"type": "string"},
			"status":       map[string]any{"type": "string"},
			"owner":        map[string]any{"type": "string"},
			"addBlocks":    map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"addBlockedBy": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"metadata":     map[string]any{"type": "object", "additionalProperties": map[string]any{}},
		},
		"required":             []string{"taskId"},
		"additionalProperties": false,
	}
	return types.ToolSpec{
		Name:            "TaskUpdate",
		Description:     "Update task fields in Todo v2.",
		InputJSONSchema: mustMarshalJSONRaw(schema),
	}
}

func nativeSleepToolSpec() types.ToolSpec {
	schema := map[string]any{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type":    "object",
		"properties": map[string]any{
			"duration_seconds": map[string]any{
				"type":        "number",
				"description": "How long to wait in seconds. The user can interrupt the sleep at any time.",
			},
		},
		"required":             []string{"duration_seconds"},
		"additionalProperties": false,
	}
	return types.ToolSpec{
		Name:            "Sleep",
		Description:     commands.SleepToolPrompt(),
		InputJSONSchema: mustMarshalJSONRaw(schema),
	}
}

func nativeCtxInspectToolSpec() types.ToolSpec {
	schema := map[string]any{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type":    "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "Optional query to filter context entries. If omitted, returns a summary of all context.",
			},
		},
		"additionalProperties": false,
	}
	return types.ToolSpec{
		Name:            "CtxInspect",
		Description:     "Inspect the current conversation context. Shows token usage, message count, and a breakdown of what's consuming context space.\n\nUse this to understand your context budget before deciding whether to snip old messages or adjust your approach.",
		InputJSONSchema: mustMarshalJSONRaw(schema),
	}
}

func nativeListPeersToolSpec() types.ToolSpec {
	schema := map[string]any{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type":    "object",
		"properties": map[string]any{
			"include_self": map[string]any{
				"type":        "boolean",
				"description": "Whether to include the current session in the list. Defaults to false.",
			},
		},
		"additionalProperties": false,
	}
	var builder strings.Builder
	builder.WriteString("List active Claude Code sessions that can receive messages via SendMessage.\n\n")
	builder.WriteString("Returns an array of peers with their addresses. Use these addresses as the ")
	builder.WriteString("`to` field in SendMessage:\n")
	builder.WriteString("- `\"uds:/path/to.sock\"` — local sessions on the same machine (Unix Domain Socket)\n")
	builder.WriteString("- `\"bridge:session_...\"` — remote sessions via Remote Control\n\n")
	builder.WriteString("Use this tool to discover messaging targets before sending cross-session messages. Only running sessions with active messaging sockets are returned.")

	return types.ToolSpec{
		Name:            "ListPeers",
		Description:     builder.String(),
		InputJSONSchema: mustMarshalJSONRaw(schema),
	}
}

func nativeMonitorToolSpec() types.ToolSpec {
	schema := map[string]any{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type":    "object",
		"properties": map[string]any{
			"command": map[string]any{
				"type":        "string",
				"description": "The shell command to run as a long-running monitor.",
			},
			"description": map[string]any{
				"type":        "string",
				"description": `Clear, concise description of what this monitor watches. Used as the label in the background tasks UI.`,
			},
		},
		"required":             []string{"command", "description"},
		"additionalProperties": false,
	}
	return types.ToolSpec{
		Name: "Monitor",
		Description: `Use Monitor to start a long-running background process that streams output (watching logs, polling APIs, tailing files, etc.). The command runs in the background and you receive a notification when it exits. Use the Read tool with the output file path to check its output at any time.
Guidelines:
- Use Monitor for commands that produce ongoing streaming output: ` + "`tail -f`" + `, log watchers, file watchers, API polling loops, ` + "`watch`" + ` commands
- Do NOT use Monitor for one-shot commands that finish quickly — use Bash for those
- Do NOT use Monitor for commands that need interactive input — they will hang
- The description should clearly explain what is being monitored
- You'll get a task notification when the monitor process exits (stream ends, script fails, or killed)
- To check output at any time, use Read on the output file path returned by this tool

Examples:
- Watching a log file: command="tail -f /var/log/app.log", description="Watch app log for errors"
- Polling an API: command="while true; do curl -s http://localhost:3000/health; sleep 5; done", description="Poll health endpoint every 5s"
- Watching for file changes: command="inotifywait -m -r ./src", description="Watch src directory for file changes"`,
		InputJSONSchema: mustMarshalJSONRaw(schema),
	}
}

func nativePushNotificationToolSpec() types.ToolSpec {
	schema := map[string]any{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type":    "object",
		"properties": map[string]any{
			"title": map[string]any{
				"type":        "string",
				"description": "Title of the push notification.",
			},
			"body": map[string]any{
				"type":        "string",
				"description": "Body text of the push notification.",
			},
			"priority": map[string]any{
				"type":        "string",
				"enum":        []string{"normal", "high"},
				"description": `Notification priority. Use "high" for blockers or permission prompts.`,
			},
		},
		"required":             []string{"title", "body"},
		"additionalProperties": false,
	}
	return types.ToolSpec{
		Name: "PushNotification",
		Description: `Send a push notification to the user's mobile device via Remote Control.

Use this when:
- A long-running task completes and the user may not be watching
- A permission prompt is waiting and you need user input
- Something urgent requires the user's attention

Requires Remote Control to be configured. Respects user notification settings (taskCompleteNotifEnabled, inputNeededNotifEnabled, agentPushNotifEnabled).`,
		InputJSONSchema: mustMarshalJSONRaw(schema),
	}
}

func nativeSendUserFileToolSpec() types.ToolSpec {
	schema := map[string]any{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type":    "object",
		"properties": map[string]any{
			"file_path": map[string]any{
				"type":        "string",
				"description": "Absolute path to the file to send to the user.",
			},
			"description": map[string]any{
				"type":        "string",
				"description": "Optional description of the file being sent.",
			},
		},
		"required":             []string{"file_path"},
		"additionalProperties": false,
	}
	return types.ToolSpec{
		Name: "SendUserFile",
		Description: `Send a file to the user's device. Use this in assistant mode when the user requests a file or when a file is relevant to the conversation.

Guidelines:
- Use absolute paths
- The file must exist and be readable
- Large files may take time to transfer`,
		InputJSONSchema: mustMarshalJSONRaw(schema),
	}
}

func nativeSnipToolSpec() types.ToolSpec {
	schema := map[string]any{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type":    "object",
		"properties": map[string]any{
			"message_ids": map[string]any{
				"type":        "array",
				"items":       map[string]any{"type": "string"},
				"description": "IDs of the messages to snip from history. Snipped messages are replaced with a short summary.",
			},
			"reason": map[string]any{
				"type":        "string",
				"description": "Why these messages are being snipped. Used in the summary replacement.",
			},
		},
		"required":             []string{"message_ids"},
		"additionalProperties": false,
	}
	return types.ToolSpec{
		Name:            "Snip",
		Description:     "Snip messages from your conversation history to free up context window space. Snipped messages are replaced with a compact summary so you retain awareness of what happened without the full content.\n\nUse this when:\n- Your context is getting full and you need to make room\n- Earlier messages contain large tool outputs you no longer need in full\n- You want to compact a long exploration sequence into a summary\n\nGuidelines:\n- Only snip messages you're confident you won't need verbatim again\n- The summary replacement preserves key facts (file paths, decisions, errors found)\n- You cannot un-snip — the original content is gone from context.",
		InputJSONSchema: mustMarshalJSONRaw(schema),
	}
}

func nativePowerShellToolSpec() types.ToolSpec {
	schema := map[string]any{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type":    "object",
		"properties": map[string]any{
			"command":           map[string]any{"type": "string"},
			"timeout":           map[string]any{"type": "number"},
			"run_in_background": map[string]any{"type": "boolean"},
		},
		"required":             []string{"command"},
		"additionalProperties": false,
	}
	return types.ToolSpec{
		Name:            "PowerShell",
		Description:     "Execute a PowerShell command in the local environment.",
		InputJSONSchema: mustMarshalJSONRaw(schema),
	}
}

func nativeSpecFromGoProvider(name string) (types.ToolSpec, bool, error) {
	switch name {
	case "Bash":
		spec, err := bashzog.BashToolSpec()
		if err != nil {
			return types.ToolSpec{}, false, err
		}
		return spec, true, nil
	case "Read":
		return nativeReadToolSpec(), true, nil
	case "Write":
		return nativeWriteToolSpec(), true, nil
	case "Edit":
		return nativeEditToolSpec(), true, nil
	case "Glob":
		return nativeGlobToolSpec(), true, nil
	case "Grep":
		return nativeGrepToolSpec(), true, nil
	case "NotebookEdit":
		return nativeNotebookEditToolSpec(), true, nil
	case "TaskStop":
		return nativeTaskStopToolSpec(), true, nil
	case "TodoWrite":
		return nativeTodoWriteToolSpec(), true, nil
	case "WebFetch":
		return nativeWebFetchToolSpec(), true, nil
	case "WebSearch":
		return nativeWebSearchToolSpec(), true, nil
	case "EnterPlanMode":
		return nativeEnterPlanModeToolSpec(), true, nil
	case "ExitPlanMode":
		return nativeExitPlanModeToolSpec(), true, nil
	case "AskUserQuestion":
		return nativeAskUserQuestionToolSpec(), true, nil
	case "EnterWorktree":
		return nativeEnterWorktreeToolSpec(), true, nil
	case "ExitWorktree":
		return nativeExitWorktreeToolSpec(), true, nil
	case "Skill":
		return nativeSkillToolSpec(), true, nil
	case "TaskOutput":
		return nativeTaskOutputToolSpec(), true, nil
	case "ToolSearch":
		return nativeToolSearchToolSpec(), true, nil
	case "Agent":
		return nativeAgentToolSpec(), true, nil
	case "CronCreate":
		return nativeCronCreateToolSpec(), true, nil
	case "CronDelete":
		return nativeCronDeleteToolSpec(), true, nil
	case "CronList":
		return nativeCronListToolSpec(), true, nil
	case "SendMessage":
		return nativeSendMessageToolSpec(), true, nil
	case "SendUserMessage":
		return nativeSendUserMessageToolSpec(), true, nil
	case "ListMcpResourcesTool":
		return nativeListMcpResourcesToolSpec(), true, nil
	case "ReadMcpResourceTool":
		return nativeReadMcpResourceToolSpec(), true, nil
	case "TaskCreate":
		return nativeTaskCreateToolSpec(), true, nil
	case "TaskGet":
		return nativeTaskGetToolSpec(), true, nil
	case "TaskList":
		return nativeTaskListToolSpec(), true, nil
	case "TaskUpdate":
		return nativeTaskUpdateToolSpec(), true, nil
	case "Sleep":
		return nativeSleepToolSpec(), true, nil
	case "PowerShell":
		return nativePowerShellToolSpec(), true, nil
	case "Config":
		return nativeEmptyObjectSchemaToolSpec("Config", "Config tool settings management."), true, nil
	case "Tungsten":
		return nativeEmptyObjectSchemaToolSpec("Tungsten", "Tungsten tool invocation."), true, nil
	case "SuggestBackgroundPR":
		return nativeEmptyObjectSchemaToolSpec("SuggestBackgroundPR", "Suggest background pull-request workflow."), true, nil
	case "WebBrowser":
		return nativeEmptyObjectSchemaToolSpec("WebBrowser", "Web browser automation tool."), true, nil
	case "OverflowTest":
		return nativeEmptyObjectSchemaToolSpec("OverflowTest", "Internal overflow stress-test tool."), true, nil
	case "CtxInspect":
		return nativeCtxInspectToolSpec(), true, nil
	case "TerminalCapture":
		return nativeEmptyObjectSchemaToolSpec("TerminalCapture", "Capture terminal state for diagnostics."), true, nil
	case "LSP":
		return nativeEmptyObjectSchemaToolSpec("LSP", "Language Server Protocol helper tool."), true, nil
	case "ListPeers":
		return nativeListPeersToolSpec(), true, nil
	case "TeamCreate":
		return nativeEmptyObjectSchemaToolSpec("TeamCreate", "Create an agent team context."), true, nil
	case "TeamDelete":
		return nativeEmptyObjectSchemaToolSpec("TeamDelete", "Delete an agent team context."), true, nil
	case "VerifyPlanExecution":
		return nativeEmptyObjectSchemaToolSpec("VerifyPlanExecution", "Verify a plan execution result."), true, nil
	case "REPL":
		return nativeEmptyObjectSchemaToolSpec("REPL", "Run command(s) in REPL mode."), true, nil
	case "workflow":
		return nativeWorkflowToolSpec(), true, nil
	case "RemoteTrigger":
		return nativeEmptyObjectSchemaToolSpec("RemoteTrigger", "Trigger remote agent or workflow actions."), true, nil
	case "Monitor":
		return nativeMonitorToolSpec(), true, nil
	case "SendUserFile":
		return nativeSendUserFileToolSpec(), true, nil
	case "PushNotification":
		return nativePushNotificationToolSpec(), true, nil
	case "SubscribePR":
		return nativeEmptyObjectSchemaToolSpec("SubscribePR", "Subscribe to PR updates."), true, nil
	case "ReviewArtifact":
		return nativeEmptyObjectSchemaToolSpec("ReviewArtifact", "Review generated artifact content."), true, nil
	case "Snip":
		return nativeSnipToolSpec(), true, nil
	default:
		return types.ToolSpec{}, false, nil
	}
}

func alwaysEnabled() bool { return true }

func envTruthyWire(v string) bool {
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

func isNodeEnvTest() bool {
	return envTruthyWire(stringsToLowerTrim(os.Getenv("NODE_ENV")))
}

func stringsToLowerTrim(v string) string {
	return strings.TrimSpace(strings.ToLower(v))
}

// goWireBaseTools mirrors TS getAllBaseTools structure and ordering.
//
// Required entries:
//   - must be present in runtime output when enabled
//   - are guarded by tests to ensure native provider coverage
//
// Optional entries:
//   - included only when enabled and schema/spec is available
//   - may fall back to export during migration
var goWireBaseTools = []goWireToolEntry{
	{Name: "Agent", Required: true, Enabled: alwaysEnabled},
	{Name: "AskUserQuestion", Required: true, Enabled: alwaysEnabled},
	{Name: "Bash", Required: true, Enabled: alwaysEnabled},
	{Name: "CronCreate", Required: true, Enabled: alwaysEnabled},
	{Name: "CronDelete", Required: true, Enabled: alwaysEnabled},
	{Name: "CronList", Required: true, Enabled: alwaysEnabled},
	{Name: "Edit", Required: true, Enabled: alwaysEnabled},
	{Name: "EnterPlanMode", Required: true, Enabled: alwaysEnabled},
	{Name: "EnterWorktree", Required: true, Enabled: alwaysEnabled},
	{Name: "ExitPlanMode", Required: true, Enabled: alwaysEnabled},
	{Name: "ExitWorktree", Required: true, Enabled: alwaysEnabled},
	{Name: "Glob", Required: true, Enabled: func() bool { return !EmbeddedSearchToolsActive() }},
	{Name: "Grep", Required: true, Enabled: func() bool { return !EmbeddedSearchToolsActive() }},
	{Name: "NotebookEdit", Required: true, Enabled: alwaysEnabled},
	{Name: "Read", Required: true, Enabled: alwaysEnabled},
	{Name: "Skill", Required: true, Enabled: alwaysEnabled},
	{Name: "TaskOutput", Required: true, Enabled: alwaysEnabled},
	{Name: "TaskStop", Required: true, Enabled: alwaysEnabled},
	{Name: "TodoWrite", Required: true, Enabled: alwaysEnabled},
	{Name: "ToolSearch", Required: true, Enabled: tstenv.ToolSearchEnabledOptimistic},
	{Name: "WebFetch", Required: true, Enabled: alwaysEnabled},
	{Name: "WebSearch", Required: true, Enabled: alwaysEnabled},
	{Name: "Write", Required: true, Enabled: alwaysEnabled},

	{Name: "Config", Required: false, Enabled: featuregates.UserTypeAnt},
	{Name: "Tungsten", Required: false, Enabled: featuregates.UserTypeAnt},
	{Name: "SuggestBackgroundPR", Required: false, Enabled: func() bool { return featuregates.UserTypeAnt() }},
	{Name: "WebBrowser", Required: false, Enabled: func() bool { return featuregates.Feature("WEB_BROWSER_TOOL") }},
	{Name: "TaskCreate", Required: false, Enabled: commands.TodoV2Enabled},
	{Name: "TaskGet", Required: false, Enabled: commands.TodoV2Enabled},
	{Name: "TaskUpdate", Required: false, Enabled: commands.TodoV2Enabled},
	{Name: "TaskList", Required: false, Enabled: commands.TodoV2Enabled},
	{Name: "OverflowTest", Required: false, Enabled: func() bool { return featuregates.Feature("OVERFLOW_TEST_TOOL") }},
	{Name: "CtxInspect", Required: false, Enabled: alwaysEnabled},
	{Name: "TerminalCapture", Required: false, Enabled: func() bool { return featuregates.Feature("TERMINAL_PANEL") }},
	{Name: "LSP", Required: false, Enabled: func() bool { return commands.IsEnvTruthy("ENABLE_LSP_TOOL") }},
	{Name: "SendMessage", Required: false, Enabled: alwaysEnabled},
	{Name: "ListPeers", Required: false, Enabled: alwaysEnabled},
	{Name: "TeamCreate", Required: false, Enabled: commands.AgentSwarmsEnabled},
	{Name: "TeamDelete", Required: false, Enabled: commands.AgentSwarmsEnabled},
	{Name: "VerifyPlanExecution", Required: false, Enabled: func() bool { return commands.IsEnvTruthy("CLAUDE_CODE_VERIFY_PLAN") }},
	{Name: "REPL", Required: false, Enabled: featuregates.UserTypeAnt},
	{Name: "workflow", Required: false, Enabled: alwaysEnabled},
	{Name: "Sleep", Required: false, Enabled: alwaysEnabled},
	{Name: "RemoteTrigger", Required: false, Enabled: func() bool { return featuregates.Feature("AGENT_TRIGGERS_REMOTE") }},
	{Name: "Monitor", Required: false, Enabled: func() bool { return featuregates.Feature("MONITOR_TOOL") }},
	{Name: "SendUserMessage", Required: false, Enabled: alwaysEnabled},
	{Name: "SendUserFile", Required: false, Enabled: alwaysEnabled},
	{Name: "PushNotification", Required: false, Enabled: alwaysEnabled},
	{Name: "SubscribePR", Required: false, Enabled: func() bool { return featuregates.Feature("KAIROS_GITHUB_WEBHOOKS") }},
	{Name: "ReviewArtifact", Required: false, Enabled: func() bool { return featuregates.Feature("REVIEW_ARTIFACT") }},
	{Name: "PowerShell", Required: false, Enabled: func() bool { return runtime.GOOS == "windows" }},
	{Name: "Snip", Required: false, Enabled: alwaysEnabled},
	{Name: "TestingPermission", Required: false, Enabled: isNodeEnvTest},
	{Name: "ListMcpResourcesTool", Required: false, Enabled: alwaysEnabled},
	{Name: "ReadMcpResourceTool", Required: false, Enabled: alwaysEnabled},
}

// ToolSpecsFromGoWire returns model-facing tool specs from the Go-owned base
// registry using only native provider implementations, with no dependency on embedded JSON.
func ToolSpecsFromGoWire() []types.ToolSpec {
	return buildGoWireToolSpecsFromExportSpecsWithProvider(nil, nativeSpecFromGoProvider)
}

// DiscoverSkillsToolSpecFromEnv returns a model tool spec when CLAUDE_CODE_DISCOVER_SKILLS_TOOL_NAME is set.
// The tool name is taken from that env var; input schema matches the historical DiscoverSkills ToolDefinition.
func DiscoverSkillsToolSpecFromEnv() (types.ToolSpec, bool) {
	name := strings.TrimSpace(os.Getenv("CLAUDE_CODE_DISCOVER_SKILLS_TOOL_NAME"))
	if name == "" {
		return types.ToolSpec{}, false
	}
	schema, err := json.Marshal(discoverSkillsInputSchemaObject())
	if err != nil {
		return types.ToolSpec{}, false
	}
	return types.ToolSpec{
		Name:            name,
		Description:     "Search and discover skills relevant to the current task when surfaced skills are insufficient.",
		InputJSONSchema: schema,
	}, true
}

func discoverSkillsInputSchemaObject() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"description": map[string]any{
				"type":        "string",
				"description": "Specific description of what you are doing or need skills for",
			},
		},
		"required": []string{"description"},
	}
}
