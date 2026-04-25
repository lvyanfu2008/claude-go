package tools

import "goc/types"

// Config is passed from [skilltools.ParityToolRunner] into unconditional tool runners.
type Config struct {
	Roots        []string
	WorkDir      string
	ProjectRoot  string
	SessionID    string
	AskAutoFirst bool // when true, AskUserQuestion picks the first option per question (gou-demo default)
	// Messages carries the parent conversation messages (needed for fork subagent).
	Messages []types.Message
	// SystemPrompt carries the parent's rendered system prompt parts (needed for fork subagent).
	SystemPrompt []string
}
