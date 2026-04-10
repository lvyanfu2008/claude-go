package appstate

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestFileHistoryState_emptyJSON(t *testing.T) {
	f := EmptyFileHistoryState()
	b, err := json.Marshal(f)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"snapshots":[]`) || !strings.Contains(string(b), `"trackedFiles":[]`) {
		t.Fatalf("%s", b)
	}
}

func TestFileHistoryState_snapshotMapNotNull(t *testing.T) {
	ts := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	name := "bk"
	f := FileHistoryState{
		Snapshots: []FileHistorySnapshot{
			{
				MessageID: "mid",
				TrackedFileBackups: map[string]FileHistoryBackup{
					"a.go": {BackupFileName: &name, Version: 1, BackupTime: ts},
				},
				Timestamp: ts,
			},
		},
		TrackedFiles:     []string{"a.go"},
		SnapshotSequence: 1,
	}
	b, err := json.Marshal(f)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"trackedFileBackups":{`) {
		t.Fatalf("%s", b)
	}
	var back FileHistoryState
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}
	if back.Snapshots[0].TrackedFileBackups == nil {
		t.Fatal("expected map after unmarshal")
	}
}
