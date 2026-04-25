package tools

import (
	"encoding/json"
	"time"

	"goc/types"
)

type AgentToolInput struct {
	Description     string `json:"description"`
	Prompt          string `json:"prompt"`
	SubagentType    string `json:"subagent_type,omitempty"`
	Model           string `json:"model,omitempty"`
	RunInBackground bool   `json:"run_in_background,omitempty"`
	Isolation       string `json:"isolation,omitempty"`
	Cwd             string `json:"cwd,omitempty"`
	Name            string `json:"name,omitempty"`
	TeamName        string `json:"team_name,omitempty"`
	Mode            string `json:"mode,omitempty"`
	Resume          string `json:"resume,omitempty"`
}

type AgentRuntimeConfig struct {
	WorkDir             string
	ProjectRoot         string
	SessionID           string
	TasksDir            string
	AvailableMCPServers []string
	Messages            []types.Message
	SystemPrompt        []string
	// Team identity for multi-agent teams.
	TeamName  string
	AgentName string
	AgentID   string
	// ProgressCallback forwards progress messages (MessageTypeProgress) from the
	// inner agent's query loop in real time so the UI can render them while the
	// agent is still executing, matching TS AgentTool/UI.tsx streaming behavior.
	ProgressCallback func(*types.Message)
}

type AgentDefinition struct {
	AgentType                          string   `json:"agentType"`
	WhenToUse                          string   `json:"whenToUse"`
	Tools                              []string `json:"tools,omitempty"`
	DisallowedTools                    []string `json:"disallowedTools,omitempty"`
	Skills                             []string `json:"skills,omitempty"`
	Source                             string   `json:"source,omitempty"`
	Model                              string   `json:"model,omitempty"`
	Color                              string   `json:"color,omitempty"`
	PermissionMode                     string   `json:"permissionMode,omitempty"`
	MaxTurns                           int      `json:"maxTurns,omitempty"`
	Background                         bool     `json:"background,omitempty"`
	Isolation                          string   `json:"isolation,omitempty"`
	RequiredMcpServers                 []string `json:"requiredMcpServers,omitempty"`
	SystemPrompt                       string   `json:"systemPrompt,omitempty"`
	OmitClaudeMd                       bool     `json:"omitClaudeMd,omitempty"`
	// Memory mirrors TS custom agent frontmatter memory field.
	// Valid values: "user", "project", "local" (AgentMemoryScope).
	Memory string `json:"memory,omitempty"`
	// Hooks mirrors TS custom agent frontmatter hooks — parsed from agent markdown frontmatter hooks field.
	// Stored as json.RawMessage to round-trip through JSON serialization for settings/tool API schemas.
	Hooks json.RawMessage `json:"hooks,omitempty"`
}

type AgentSession struct {
	ID                                 string    `json:"id"`
	Name                               string    `json:"name,omitempty"`
	TeamName                           string    `json:"teamName,omitempty"`
	AgentType                          string    `json:"agentType"`
	Description                        string    `json:"description"`
	Model                              string    `json:"model,omitempty"`
	PermissionMode                     string    `json:"permissionMode,omitempty"`
	MaxTurns                           int       `json:"maxTurns,omitempty"`
	AllowedTools                       []string  `json:"allowedTools,omitempty"`
	Skills                             []string  `json:"skills,omitempty"`
	RequiredMcpServers                 []string  `json:"requiredMcpServers,omitempty"`
	AvailableMcpServers                []string  `json:"availableMcpServers,omitempty"`
	Prompt                             string    `json:"prompt"`
	WorkDir                            string    `json:"workDir"`
	ProjectRoot                        string    `json:"projectRoot"`
	Isolation                          string    `json:"isolation,omitempty"`
	WorktreePath                       string    `json:"worktreePath,omitempty"`
	SystemPrompt                       string    `json:"systemPrompt,omitempty"`
	OmitClaudeMd                       bool      `json:"omitClaudeMd,omitempty"`
	CriticalSystemReminderExperimental string    `json:"criticalSystemReminder_EXPERIMENTAL,omitempty"`
	Memory                             string    `json:"memory,omitempty"`
	CreatedAt                          time.Time `json:"createdAt"`
	UpdatedAt                          time.Time `json:"updatedAt"`
	LastOutput                         string            `json:"lastOutput,omitempty"`
	ProgressMessages                   []json.RawMessage `json:"progressMessages,omitempty"`
	// Hooks mirrors the agent definition's hooks frontmatter; stored for runtime hook execution.
	Hooks json.RawMessage `json:"hooks,omitempty"`
	// Summary holds the latest agent summarization text (set periodically by the summary goroutine).
	Summary string `json:"summary,omitempty"`
}

type AgentToolResponse struct {
	Data AgentToolResponseData `json:"data"`
}

type AgentToolResponseData struct {
	Success      bool   `json:"success"`
	AgentID      string `json:"agent_id,omitempty"`
	Name         string `json:"name,omitempty"`
	AgentType    string `json:"agent_type,omitempty"`
	Message      string `json:"message,omitempty"`
	Output       string `json:"output,omitempty"`
	OutputFile       string            `json:"output_file,omitempty"`
	IsBackground     bool              `json:"is_background,omitempty"`
	WorktreePath     string            `json:"worktree_path,omitempty"`
	ProgressMessages []json.RawMessage `json:"progressMessages,omitempty"`
}

type SendMessageInput struct {
	To      string `json:"to"`
	Message string `json:"message"`
}
