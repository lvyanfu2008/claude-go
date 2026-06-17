package workflow

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sync"
)

// Journal records agent call results for resume/cache-hit logic.
type Journal struct {
	RunID   string          `json:"runId"`
	Entries []JournalEntry  `json:"entries"`
	index   map[string]int  // hash → position in Entries
	mu      sync.Mutex
}

// JournalEntry is a single cached agent result.
type JournalEntry struct {
	Hash   string          `json:"hash"`
	Result json.RawMessage `json:"result"`
}

// NewJournal creates an empty journal for a workflow run.
func NewJournal(runID string) *Journal {
	return &Journal{
		RunID: runID,
		index: make(map[string]int),
	}
}

// Lookup returns a cached entry by hash, or nil if not found.
func (j *Journal) Lookup(hash string) *JournalEntry {
	j.mu.Lock()
	defer j.mu.Unlock()
	if idx, ok := j.index[hash]; ok {
		if idx < len(j.Entries) && j.Entries[idx].Hash == hash {
			entry := j.Entries[idx]
			return &entry
		}
	}
	return nil
}

// Record stores a new agent result in the journal.
func (j *Journal) Record(hash string, result json.RawMessage) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if _, ok := j.index[hash]; ok {
		return // Already recorded
	}
	idx := len(j.Entries)
	j.Entries = append(j.Entries, JournalEntry{Hash: hash, Result: result})
	j.index[hash] = idx
}

// HashAgentCall computes a deterministic hash for an agent(prompt, opts) call.
func HashAgentCall(prompt string, opts *AgentOpts) string {
	h := sha256.New()
	h.Write([]byte(prompt))
	if opts != nil {
		data, _ := json.Marshal(opts)
		h.Write(data)
	}
	return fmt.Sprintf("%x", h.Sum(nil))[:16]
}
