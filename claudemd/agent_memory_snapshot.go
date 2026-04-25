package claudemd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	snapshotBase = "agent-memory-snapshots"
	snapshotJSON = "snapshot.json"
	syncedJSON   = ".snapshot-synced.json"
)

// snapshotMeta mirrors TS snapshotMetaSchema.
type snapshotMeta struct {
	UpdatedAt string `json:"updatedAt"`
}

// syncedMeta mirrors TS SyncedMeta.
type syncedMeta struct {
	SyncedFrom string `json:"syncedFrom"`
}

// GetSnapshotDirForAgent mirrors TS getSnapshotDirForAgent.
// Returns the path to the snapshot directory for an agent in the current project.
// e.g., <cwd>/.claude/agent-memory-snapshots/<agentType>/
func GetSnapshotDirForAgent(agentType string) string {
	cwd, _ := os.Getwd()
	return filepath.Join(cwd, ".claude", snapshotBase, agentType)
}

func getSnapshotJSONPath(agentType string) string {
	return filepath.Join(GetSnapshotDirForAgent(agentType), snapshotJSON)
}

func getSyncedJSONPath(agentType string, scope AgentMemoryScope) string {
	return filepath.Join(GetAgentMemoryDir(agentType, scope), syncedJSON)
}

// readSnapshotMeta reads the snapshot metadata JSON file.
func readSnapshotMeta(agentType string) (*snapshotMeta, error) {
	path := getSnapshotJSONPath(agentType)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var meta snapshotMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, err
	}
	if strings.TrimSpace(meta.UpdatedAt) == "" {
		return nil, os.ErrNotExist
	}
	return &meta, nil
}

// readSyncedMeta reads the synced metadata JSON file.
func readSyncedMeta(agentType string, scope AgentMemoryScope) (*syncedMeta, error) {
	path := getSyncedJSONPath(agentType, scope)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var meta syncedMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, err
	}
	if strings.TrimSpace(meta.SyncedFrom) == "" {
		return nil, os.ErrNotExist
	}
	return &meta, nil
}

// copySnapshotToLocal mirrors TS copySnapshotToLocal.
// Copies all files from the snapshot directory to the local agent memory directory,
// excluding the snapshot.json metadata file.
func copySnapshotToLocal(agentType string, scope AgentMemoryScope) error {
	snapshotMemDir := GetSnapshotDirForAgent(agentType)
	localMemDir := GetAgentMemoryDir(agentType, scope)

	if err := os.MkdirAll(localMemDir, 0o700); err != nil {
		return err
	}

	entries, err := os.ReadDir(snapshotMemDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() || entry.Name() == snapshotJSON {
			continue
		}
		data, err := os.ReadFile(filepath.Join(snapshotMemDir, entry.Name()))
		if err != nil {
			continue
		}
		_ = os.WriteFile(filepath.Join(localMemDir, entry.Name()), data, 0o600)
	}
	return nil
}

// saveSyncedMeta mirrors TS saveSyncedMeta.
// Writes the synced metadata file to track which snapshot was last synced.
func saveSyncedMeta(agentType string, scope AgentMemoryScope, snapshotTimestamp string) error {
	syncedPath := getSyncedJSONPath(agentType, scope)
	localMemDir := GetAgentMemoryDir(agentType, scope)
	if err := os.MkdirAll(localMemDir, 0o700); err != nil {
		return err
	}
	meta := syncedMeta{SyncedFrom: snapshotTimestamp}
	data, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	return os.WriteFile(syncedPath, data, 0o600)
}

// CheckAgentMemorySnapshotResult mirrors TS checkAgentMemorySnapshot return type.
type CheckAgentMemorySnapshotResult struct {
	Action            string // "none", "initialize", or "prompt-update"
	SnapshotTimestamp string
}

// CheckAgentMemorySnapshot mirrors TS checkAgentMemorySnapshot.
// Checks if a snapshot exists and whether it's newer than what we last synced.
func CheckAgentMemorySnapshot(agentType string, scope AgentMemoryScope) CheckAgentMemorySnapshotResult {
	snapshotMeta, err := readSnapshotMeta(agentType)
	if err != nil {
		return CheckAgentMemorySnapshotResult{Action: "none"}
	}

	localMemDir := GetAgentMemoryDir(agentType, scope)
	hasLocalMemory := false
	if entries, err := os.ReadDir(localMemDir); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
				hasLocalMemory = true
				break
			}
		}
	}

	if !hasLocalMemory {
		return CheckAgentMemorySnapshotResult{
			Action:            "initialize",
			SnapshotTimestamp: snapshotMeta.UpdatedAt,
		}
	}

	syncedMeta, err := readSyncedMeta(agentType, scope)
	if err != nil {
		// No sync record: prompt update
		return CheckAgentMemorySnapshotResult{
			Action:            "prompt-update",
			SnapshotTimestamp: snapshotMeta.UpdatedAt,
		}
	}

	// Compare timestamps
	snapshotTime, err1 := time.Parse(time.RFC3339Nano, snapshotMeta.UpdatedAt)
	syncedTime, err2 := time.Parse(time.RFC3339Nano, syncedMeta.SyncedFrom)
	if err1 == nil && err2 == nil && snapshotTime.After(syncedTime) {
		return CheckAgentMemorySnapshotResult{
			Action:            "prompt-update",
			SnapshotTimestamp: snapshotMeta.UpdatedAt,
		}
	}

	// Fallback to string comparison if time parsing fails
	if snapshotMeta.UpdatedAt > syncedMeta.SyncedFrom {
		return CheckAgentMemorySnapshotResult{
			Action:            "prompt-update",
			SnapshotTimestamp: snapshotMeta.UpdatedAt,
		}
	}

	return CheckAgentMemorySnapshotResult{Action: "none"}
}

// InitializeFromSnapshot mirrors TS initializeFromSnapshot.
// Initializes local agent memory from a snapshot (first-time setup).
func InitializeFromSnapshot(agentType string, scope AgentMemoryScope, snapshotTimestamp string) error {
	if err := copySnapshotToLocal(agentType, scope); err != nil {
		return err
	}
	return saveSyncedMeta(agentType, scope, snapshotTimestamp)
}

// ReplaceFromSnapshot mirrors TS replaceFromSnapshot.
// Replaces local agent memory with the snapshot contents.
func ReplaceFromSnapshot(agentType string, scope AgentMemoryScope, snapshotTimestamp string) error {
	localMemDir := GetAgentMemoryDir(agentType, scope)

	// Remove existing .md files before copying to avoid orphans
	if entries, err := os.ReadDir(localMemDir); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
				_ = os.Remove(filepath.Join(localMemDir, entry.Name()))
			}
		}
	}

	if err := copySnapshotToLocal(agentType, scope); err != nil {
		return err
	}
	return saveSyncedMeta(agentType, scope, snapshotTimestamp)
}

// MarkSnapshotSynced mirrors TS markSnapshotSynced.
// Marks the current snapshot as synced without changing local memory.
func MarkSnapshotSynced(agentType string, scope AgentMemoryScope, snapshotTimestamp string) error {
	return saveSyncedMeta(agentType, scope, snapshotTimestamp)
}
