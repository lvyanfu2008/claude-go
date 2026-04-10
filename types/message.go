// Package types mirrors src/types/message.ts (conversation message shapes).
// Path rule: src/… in TS ↔ go/… in Go.
//
// When adding fields, prefer full parity with the TS type (see .cursor/rules/go-ts-struct-parity.mdc).
// PermissionMode and ImagePasteIDs are Go extensions (CLI / permission paths), not in every TS excerpt of this type.
package types

import "encoding/json"

// MessageType mirrors src/types/message.ts MessageType.
type MessageType string

const (
	MessageTypeUser                MessageType = "user"
	MessageTypeAssistant           MessageType = "assistant"
	MessageTypeSystem              MessageType = "system"
	MessageTypeAttachment          MessageType = "attachment"
	MessageTypeProgress            MessageType = "progress"
	MessageTypeGroupedToolUse      MessageType = "grouped_tool_use"
	MessageTypeCollapsedReadSearch MessageType = "collapsed_read_search"
)

// Message mirrors src/types/message.ts Message (base; TS allows extra keys via index signature).
// UserMessage, AssistantMessage, AttachmentMessage, SystemMessage in TS are this shape with a narrowed type literal.
type Message struct {
	Type             MessageType     `json:"type"`
	UUID             string          `json:"uuid"`
	IsMeta           *bool           `json:"isMeta,omitempty"`
	PermissionMode   *string         `json:"permissionMode,omitempty"`
	ImagePasteIDs    []int           `json:"imagePasteIds,omitempty"`
	IsCompactSummary *bool           `json:"isCompactSummary,omitempty"`
	ToolUseResult    json.RawMessage `json:"toolUseResult,omitempty"`
	// SourceToolAssistantUUID links user tool_result blocks to the assistant turn (TS UserMessage).
	SourceToolAssistantUUID   *string         `json:"sourceToolAssistantUUID,omitempty"`
	IsVirtual                 *bool           `json:"isVirtual,omitempty"`
	// IsApiErrorMessage is set on synthetic assistant API error rows (normalizeMessagesForAPI strips them).
	IsApiErrorMessage *bool `json:"isApiErrorMessage,omitempty"`
	IsVisibleInTranscriptOnly *bool           `json:"isVisibleInTranscriptOnly,omitempty"`
	Attachment                json.RawMessage `json:"attachment,omitempty"`
	Message                   json.RawMessage `json:"message,omitempty"`
	Content                   json.RawMessage `json:"content,omitempty"`
	// Data is set for type "progress" (see ProgressMessage in TS).
	Data json.RawMessage `json:"data,omitempty"`
	// Subtype / Level / Timestamp are used by system messages (informational, local_command, …).
	Subtype   *string `json:"subtype,omitempty"`
	Level     *string `json:"level,omitempty"`
	Timestamp *string `json:"timestamp,omitempty"`

	// --- grouped_tool_use (GroupedToolUseMessage ∩ Message, message.ts) ---
	ToolName       string    `json:"toolName,omitempty"`
	Messages       []Message `json:"messages,omitempty"`
	Results        []Message `json:"results,omitempty"`
	DisplayMessage *Message  `json:"displayMessage,omitempty"`

	// --- collapsed_read_search (CollapsedReadSearchGroup ∩ Message) ---
	// Note: TS uses timestamp?: unknown; Message.Timestamp *string shares json "timestamp" when it is a string.
	SearchCount           int                `json:"searchCount,omitempty"`
	ReadCount             int                `json:"readCount,omitempty"`
	ListCount             int                `json:"listCount,omitempty"`
	ReplCount             int                `json:"replCount,omitempty"`
	MemorySearchCount     int                `json:"memorySearchCount,omitempty"`
	MemoryReadCount       int                `json:"memoryReadCount,omitempty"`
	MemoryWriteCount      int                `json:"memoryWriteCount,omitempty"`
	ReadFilePaths         []string           `json:"readFilePaths,omitempty"`
	SearchArgs            []string           `json:"searchArgs,omitempty"`
	LatestDisplayHint     *string            `json:"latestDisplayHint,omitempty"`
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

// UserMessage mirrors src/types/message.ts UserMessage (type "user").
type UserMessage Message

// AssistantMessage mirrors src/types/message.ts AssistantMessage (type "assistant").
type AssistantMessage Message

// AttachmentMessage mirrors src/types/message.ts AttachmentMessage (type "attachment"; attachment object required in TS).
type AttachmentMessage Message

// SystemMessage mirrors src/types/message.ts SystemMessage (type "system").
type SystemMessage Message

// ProgressMessage mirrors src/types/message.ts ProgressMessage (type "progress"; data present).
type ProgressMessage Message
