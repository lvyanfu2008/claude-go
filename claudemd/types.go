// Package claudemd ports src/utils/claudemd.ts memory discovery and getClaudeMds for Go-only runtimes.
package claudemd

// MemoryType mirrors src/utils/memory/types.ts MemoryType used by claudemd.
type MemoryType string

const (
	MemoryManaged MemoryType = "Managed"
	MemoryUser    MemoryType = "User"
	MemoryProject MemoryType = "Project"
	MemoryLocal   MemoryType = "Local"
	MemoryAutoMem MemoryType = "AutoMem"
	MemoryTeamMem MemoryType = "TeamMem"
)

// MemoryFileInfo mirrors claudemd.ts MemoryFileInfo (fields used by getMemoryFiles / getClaudeMds).
type MemoryFileInfo struct {
	Path    string
	Type    MemoryType
	Content string
	Parent  string
	Globs   []string
}
