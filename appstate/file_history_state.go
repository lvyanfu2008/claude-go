package appstate

import (
	"encoding/json"
	"time"
)

// FileHistoryBackup mirrors src/utils/fileHistory.ts FileHistoryBackup (backupFileName: string | null).
type FileHistoryBackup struct {
	BackupFileName *string   `json:"backupFileName"`
	Version        int       `json:"version"`
	BackupTime     time.Time `json:"backupTime"`
}

// FileHistorySnapshot mirrors src/utils/fileHistory.ts FileHistorySnapshot.
type FileHistorySnapshot struct {
	MessageID          string                       `json:"messageId"`
	TrackedFileBackups map[string]FileHistoryBackup `json:"trackedFileBackups"`
	Timestamp          time.Time                    `json:"timestamp"`
}

// FileHistoryState mirrors src/utils/fileHistory.ts FileHistoryState (trackedFiles Set → []string).
type FileHistoryState struct {
	Snapshots        []FileHistorySnapshot `json:"snapshots"`
	TrackedFiles     []string              `json:"trackedFiles"`
	SnapshotSequence int                   `json:"snapshotSequence"`
}

// EmptyFileHistoryState matches getDefaultAppState fileHistory (empty snapshots, no tracked files).
func EmptyFileHistoryState() FileHistoryState {
	return FileHistoryState{
		Snapshots:        []FileHistorySnapshot{},
		TrackedFiles:     []string{},
		SnapshotSequence: 0,
	}
}

func normalizeSnapshotMaps(snaps []FileHistorySnapshot) []FileHistorySnapshot {
	if snaps == nil {
		return []FileHistorySnapshot{}
	}
	out := make([]FileHistorySnapshot, len(snaps))
	for i := range snaps {
		s := snaps[i]
		if s.TrackedFileBackups == nil {
			s.TrackedFileBackups = make(map[string]FileHistoryBackup)
		}
		out[i] = s
	}
	return out
}

// MarshalJSON uses [] for nil snapshots/trackedFiles and {} for nil per-snapshot maps.
func (f FileHistoryState) MarshalJSON() ([]byte, error) {
	type snap struct {
		MessageID          string                       `json:"messageId"`
		TrackedFileBackups map[string]FileHistoryBackup `json:"trackedFileBackups"`
		Timestamp          time.Time                    `json:"timestamp"`
	}
	snaps := f.Snapshots
	if snaps == nil {
		snaps = []FileHistorySnapshot{}
	}
	tf := f.TrackedFiles
	if tf == nil {
		tf = []string{}
	}
	list := make([]snap, len(snaps))
	for i := range snaps {
		m := snaps[i].TrackedFileBackups
		if m == nil {
			m = make(map[string]FileHistoryBackup)
		}
		list[i] = snap{
			MessageID:          snaps[i].MessageID,
			TrackedFileBackups: m,
			Timestamp:          snaps[i].Timestamp,
		}
	}
	return json.Marshal(struct {
		Snapshots        []snap   `json:"snapshots"`
		TrackedFiles     []string `json:"trackedFiles"`
		SnapshotSequence int      `json:"snapshotSequence"`
	}{Snapshots: list, TrackedFiles: tf, SnapshotSequence: f.SnapshotSequence})
}

// UnmarshalJSON normalizes nil slices and per-snapshot maps.
func (f *FileHistoryState) UnmarshalJSON(data []byte) error {
	var raw struct {
		Snapshots        []FileHistorySnapshot `json:"snapshots"`
		TrackedFiles     []string              `json:"trackedFiles"`
		SnapshotSequence int                   `json:"snapshotSequence"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*f = FileHistoryState{
		Snapshots:        normalizeSnapshotMaps(raw.Snapshots),
		TrackedFiles:     raw.TrackedFiles,
		SnapshotSequence: raw.SnapshotSequence,
	}
	if f.TrackedFiles == nil {
		f.TrackedFiles = []string{}
	}
	return nil
}
