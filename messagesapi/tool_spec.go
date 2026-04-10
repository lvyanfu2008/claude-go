package messagesapi

// ToolSpec is the minimal tool surface needed for normalizeMessagesForAPI (name + optional aliases).
type ToolSpec struct {
	Name    string
	Aliases []string
}
