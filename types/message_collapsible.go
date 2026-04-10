// Message sub-shapes mirroring src/types/message.ts (GroupedToolUseMessage,
// CollapsedReadSearchGroup, system variants). Fields are full TS parity for named
// properties; open-ended TS index signatures are noted per struct.
package types

import "encoding/json"

// StopHookInfo mirrors src/types/message.ts StopHookInfo.
// TS allows [key: string]: unknown — extend via sibling JSON decode or future Extra field.
type StopHookInfo struct {
	Command    *string `json:"command,omitempty"`
	DurationMs *int64  `json:"durationMs,omitempty"`
}

// MemoryAttachment mirrors attachment.memories[] and relevantMemories[] entries in message.ts.
type MemoryAttachment struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	MtimeMs int64  `json:"mtimeMs"`
}

// GitCommitEntry mirrors CollapsedReadSearchGroup.commits[].
type GitCommitEntry struct {
	Sha  string     `json:"sha"`
	Kind CommitKind `json:"kind"`
}

// GitPushEntry mirrors CollapsedReadSearchGroup.pushes[].
type GitPushEntry struct {
	Branch string `json:"branch"`
}

// GitBranchEntry mirrors CollapsedReadSearchGroup.branches[].
type GitBranchEntry struct {
	Ref    string       `json:"ref"`
	Action BranchAction `json:"action"`
}

// GitPrEntry mirrors CollapsedReadSearchGroup.prs[].
type GitPrEntry struct {
	Number int      `json:"number"`
	URL    *string  `json:"url,omitempty"`
	Action PrAction `json:"action"`
}

// CollapsedReadSearchGroup mirrors src/types/message.ts CollapsedReadSearchGroup.
// TS index signature [key: string]: unknown — unknown keys are dropped on stdlib decode unless captured elsewhere.
type CollapsedReadSearchGroup struct {
	Type                  MessageType        `json:"type"`
	UUID                  string             `json:"uuid"`
	Timestamp             json.RawMessage    `json:"timestamp,omitempty"`
	SearchCount           int                `json:"searchCount"`
	ReadCount             int                `json:"readCount"`
	ListCount             int                `json:"listCount"`
	ReplCount             int                `json:"replCount"`
	MemorySearchCount     int                `json:"memorySearchCount"`
	MemoryReadCount       int                `json:"memoryReadCount"`
	MemoryWriteCount      int                `json:"memoryWriteCount"`
	ReadFilePaths         []string           `json:"readFilePaths"`
	SearchArgs            []string           `json:"searchArgs"`
	LatestDisplayHint     *string            `json:"latestDisplayHint,omitempty"`
	Messages              []Message          `json:"messages"`
	DisplayMessage        Message            `json:"displayMessage"`
	McpCallCount          *int               `json:"mcpCallCount,omitempty"`
	McpServerNames        []string           `json:"mcpServerNames,omitempty"`
	BashCount             *int               `json:"bashCount,omitempty"`
	GitOpBashCount        *int               `json:"gitOpBashCount,omitempty"`
	Commits               []GitCommitEntry   `json:"commits,omitempty"`
	Pushes                []GitPushEntry     `json:"pushes,omitempty"`
	Branches              []GitBranchEntry   `json:"branches,omitempty"`
	Prs                   []GitPrEntry       `json:"prs,omitempty"`
	HookTotalMs           *int64             `json:"hookTotalMs,omitempty"`
	HookCount             *int               `json:"hookCount,omitempty"`
	HookInfos             []StopHookInfo     `json:"hookInfos,omitempty"`
	RelevantMemories      []MemoryAttachment `json:"relevantMemories,omitempty"`
	TeamMemorySearchCount *int               `json:"teamMemorySearchCount,omitempty"`
	TeamMemoryReadCount   *int               `json:"teamMemoryReadCount,omitempty"`
	TeamMemoryWriteCount  *int               `json:"teamMemoryWriteCount,omitempty"`
}

// GroupedToolUseMessage mirrors src/types/message.ts GroupedToolUseMessage (intersection with Message).
// Transcript rows often decode as a single types.Message with Type=grouped_tool_use and the same JSON keys (see types.Message).
type GroupedToolUseMessage struct {
	Message
	ToolName       string    `json:"toolName"`
	Messages       []Message `json:"messages"`
	Results        []Message `json:"results"`
	DisplayMessage Message   `json:"displayMessage"`
}

// SystemStopHookSummaryMessage mirrors src/types/message.ts SystemStopHookSummaryMessage.
type SystemStopHookSummaryMessage struct {
	Message
	HookLabel       string         `json:"hookLabel"`
	HookCount       int            `json:"hookCount"`
	TotalDurationMs *int64         `json:"totalDurationMs,omitempty"`
	HookInfos       []StopHookInfo `json:"hookInfos"`
}

// SystemCompactBoundaryMessage mirrors src/types/message.ts SystemCompactBoundaryMessage.
type SystemCompactBoundaryMessage struct {
	Message
	CompactMetadata json.RawMessage `json:"compactMetadata,omitempty"`
}
