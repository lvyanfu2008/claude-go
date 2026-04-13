package sessiontranscript

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"goc/types"
)

const defaultVersion = "2.1.888"

// TeamInfo mirrors sessionStorage TeamInfo.
type TeamInfo struct {
	TeamName  string
	AgentName string
}

// SessionMetadata optional lines written when materializing a new transcript file (TS reAppendSessionMetadata subset).
type SessionMetadata struct {
	LastPrompt  string
	CustomTitle string
	Tag         string
	AgentName   string
	AgentColor  string
	Mode        string
}

// Store holds session transcript persistence state (TS Project + globals subset).
type Store struct {
	mu sync.Mutex

	SessionID         string
	OriginalCwd       string
	SessionProjectDir string // optional override (TS getSessionProjectDir)
	ConfigHome        string // optional; default ConfigHomeDir()
	Cwd               string // stamp on each line; default OriginalCwd
	UserType          string // default "external"
	Entrypoint        string
	Version           string
	PromptID          string // optional, stamped on user messages
	PlanSlug          string // optional

	SkipPersistence bool
	InitialMetadata *SessionMetadata // written once when creating new file before first messages

	// Sidechain / agent path (RecordSidechainTranscript)
	AgentSubdir string

	// FileHistorySnapshotOnUser when true, may append a TS-shaped JSONL row
	// {type:"file-history-snapshot",...} immediately before a newly persisted non-meta user
	// message (sessionStorage.insertFileHistorySnapshot shape; empty trackedFileBackups when
	// Go has no file-backup pipeline). Skipped when file checkpointing is disabled (same as TS
	// fileHistoryEnabled() false — see [fileCheckpointingDisabled]).
	FileHistorySnapshotOnUser bool
	// FileHistorySnapshotOnce when true, append at most one such stub line per Store lifetime
	// (first eligible user only). TS writes one snapshot per fileHistoryMakeSnapshot call when
	// checkpointing is on; empty Go stubs every turn inflate JSONL — use Once for demo parity.
	FileHistorySnapshotOnce bool

	fileHistoryStubEmitted bool

	// currentSessionLastPrompt mirrors TS sessionStorage currentSessionLastPrompt
	// (tail last-prompt JSONL rows for resume picker / lite metadata).
	currentSessionLastPrompt string

	// TranscriptFile when non-empty overrides the computed JSONL path (tests).
	TranscriptFile string

	// uuid cache: path -> ids (invalidated when file size changes)
	uuidCachePath string
	uuidCache     map[string]struct{}
	uuidCacheSize int64
}

func (s *Store) configHome() string {
	if strings.TrimSpace(s.ConfigHome) != "" {
		return filepath.Clean(s.ConfigHome)
	}
	return ConfigHomeDir()
}

func (s *Store) cwdStamp() string {
	if strings.TrimSpace(s.Cwd) != "" {
		return filepath.Clean(s.Cwd)
	}
	if strings.TrimSpace(s.OriginalCwd) != "" {
		return filepath.Clean(s.OriginalCwd)
	}
	wd, _ := os.Getwd()
	return wd
}

func (s *Store) userType() string {
	if strings.TrimSpace(s.UserType) != "" {
		return s.UserType
	}
	if v := strings.TrimSpace(os.Getenv("USER_TYPE")); v != "" {
		return v
	}
	return "external"
}

func (s *Store) version() string {
	if strings.TrimSpace(s.Version) != "" {
		return s.Version
	}
	return defaultVersion
}

// TranscriptPath returns the main session JSONL path.
func (s *Store) TranscriptPath() string {
	if strings.TrimSpace(s.TranscriptFile) != "" {
		return filepath.Clean(s.TranscriptFile)
	}
	return TranscriptPath(s.SessionID, s.OriginalCwd, s.SessionProjectDir, s.configHome())
}

// sidechainPath returns JSONL path for RecordSidechainTranscript.
func (s *Store) sidechainPath(agentID string) string {
	return AgentTranscriptPath(s.SessionID, s.OriginalCwd, s.SessionProjectDir, s.configHome(), agentID, s.AgentSubdir)
}

func (s *Store) shouldSkipPersistence() bool {
	if s.SkipPersistence {
		return true
	}
	return envTruthy("CLAUDE_CODE_SKIP_PROMPT_HISTORY")
}

// RecordOpts options for RecordTranscript.
type RecordOpts struct {
	StartingParentUUID string
	Team               *TeamInfo
	// AllMessages when non-nil is used for REPL id collection (TS cleanMessagesForLogging third arg).
	AllMessages []types.Message
}

// IsCompactBoundaryMessage mirrors messages.ts isCompactBoundaryMessage.
func IsCompactBoundaryMessage(m types.Message) bool {
	return m.Type == types.MessageTypeSystem && m.Subtype != nil && *m.Subtype == "compact_boundary"
}

// IsChainParticipant mirrors sessionStorage.ts isChainParticipant.
func IsChainParticipant(m types.Message) bool {
	return m.Type != types.MessageTypeProgress
}

func gitBranchForDir(dir string) string {
	d := strings.TrimSpace(dir)
	if d == "" {
		return ""
	}
	cmd := exec.Command("git", "-C", d, "rev-parse", "--abbrev-ref", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func messageToMap(m types.Message) (map[string]any, error) {
	raw, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	var o map[string]any
	if err := json.Unmarshal(raw, &o); err != nil {
		return nil, err
	}
	return o, nil
}

func isUserMetaMessage(m types.Message) bool {
	return m.Type == types.MessageTypeUser && m.IsMeta != nil && *m.IsMeta
}

// appendTSFileHistorySnapshotLine mirrors sessionStorage FileHistorySnapshotMessage JSON shape.
func fileCheckpointingDisabled() bool {
	return envTruthy("CLAUDE_CODE_DISABLE_FILE_CHECKPOINTING")
}

func appendTSFileHistorySnapshotLine(path, messageID string, isSnapshotUpdate bool) error {
	messageID = strings.TrimSpace(messageID)
	if messageID == "" {
		return nil
	}
	ts := time.Now().UTC().Format(time.RFC3339Nano)
	snap := map[string]any{
		"messageId":          messageID,
		"trackedFileBackups": map[string]any{},
		"timestamp":          ts,
	}
	entry := map[string]any{
		"type":             "file-history-snapshot",
		"messageId":        messageID,
		"snapshot":         snap,
		"isSnapshotUpdate": isSnapshotUpdate,
	}
	return appendJSONL(path, entry)
}

func appendJSONL(path string, obj map[string]any) error {
	line, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.Write(append(line, '\n')); err != nil {
		return err
	}
	return nil
}

func writeMetadataLines(path, sessionID string, meta *SessionMetadata) error {
	if meta == nil {
		return nil
	}
	type kv struct {
		key string
		obj map[string]any
	}
	var lines []map[string]any
	if meta.LastPrompt != "" {
		lines = append(lines, map[string]any{"type": "last-prompt", "lastPrompt": meta.LastPrompt, "sessionId": sessionID})
	}
	if meta.CustomTitle != "" {
		lines = append(lines, map[string]any{"type": "custom-title", "customTitle": meta.CustomTitle, "sessionId": sessionID})
	}
	if meta.Tag != "" {
		lines = append(lines, map[string]any{"type": "tag", "tag": meta.Tag, "sessionId": sessionID})
	}
	if meta.AgentName != "" {
		lines = append(lines, map[string]any{"type": "agent-name", "agentName": meta.AgentName, "sessionId": sessionID})
	}
	if meta.AgentColor != "" {
		lines = append(lines, map[string]any{"type": "agent-color", "agentColor": meta.AgentColor, "sessionId": sessionID})
	}
	if meta.Mode != "" {
		lines = append(lines, map[string]any{"type": "mode", "mode": meta.Mode, "sessionId": sessionID})
	}
	for _, o := range lines {
		if err := appendJSONL(path, o); err != nil {
			return err
		}
	}
	return nil
}

func hasUserOrAssistant(msgs []types.Message) bool {
	for _, m := range msgs {
		if m.Type == types.MessageTypeUser || m.Type == types.MessageTypeAssistant {
			return true
		}
	}
	return false
}

// RecordTranscript mirrors recordTranscript (main chain, isSidechain=false).
func (s *Store) RecordTranscript(ctx context.Context, messages []types.Message, opts RecordOpts) (lastUUID string, err error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.shouldSkipPersistence() {
		return "", nil
	}
	sid := strings.TrimSpace(s.SessionID)
	if sid == "" {
		return "", nil
	}
	if !IsValidUUID(sid) {
		return "", fmt.Errorf("sessiontranscript: SessionID %q is not a valid UUID", sid)
	}

	path := s.TranscriptPath()
	all := opts.AllMessages
	if len(all) == 0 {
		all = messages
	}
	cleaned := CleanMessagesForLogging(messages, all, s.userType())

	existing, err := s.loadUUIDSet(path)
	if err != nil {
		return "", err
	}

	var newMsgs []types.Message
	var startingParent = strings.TrimSpace(opts.StartingParentUUID)
	seenNew := false
	for _, m := range cleaned {
		if m.UUID == "" {
			continue
		}
		if _, ok := existing[m.UUID]; ok {
			if !seenNew && IsChainParticipant(m) {
				startingParent = m.UUID
			}
		} else {
			newMsgs = append(newMsgs, m)
			seenNew = true
		}
	}

	if len(newMsgs) == 0 {
		if startingParent != "" {
			return startingParent, nil
		}
		return "", nil
	}

	// Materialize: first user/assistant in this write batch
	_, statErr := os.Stat(path)
	fileMissing := statErr != nil && os.IsNotExist(statErr)
	if fileMissing && hasUserOrAssistant(newMsgs) {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return "", err
		}
		if s.InitialMetadata != nil {
			if err := writeMetadataLines(path, s.SessionID, s.InitialMetadata); err != nil {
				return "", err
			}
			if lp := strings.TrimSpace(s.InitialMetadata.LastPrompt); lp != "" {
				s.currentSessionLastPrompt = FlattenLastPromptCache(lp)
			}
		}
	}

	parentUUID := startingParent
	gitBr := gitBranchForDir(s.cwdStamp())
	slug := strings.TrimSpace(s.PlanSlug)
	entry := strings.TrimSpace(s.Entrypoint)
	if entry == "" {
		entry = strings.TrimSpace(os.Getenv("CLAUDE_CODE_ENTRYPOINT"))
	}

	var lastRecorded string
	for _, m := range newMsgs {
		if m.UUID == "" {
			continue
		}
		line, err := messageToMap(m)
		if err != nil {
			return "", err
		}

		isCompact := IsCompactBoundaryMessage(m)
		effectiveParent := parentUUID
		if m.Type == types.MessageTypeUser && m.SourceToolAssistantUUID != nil && strings.TrimSpace(*m.SourceToolAssistantUUID) != "" {
			effectiveParent = strings.TrimSpace(*m.SourceToolAssistantUUID)
		}

		if isCompact {
			line["parentUuid"] = nil
			if parentUUID != "" {
				line["logicalParentUuid"] = parentUUID
			}
		} else if effectiveParent != "" {
			line["parentUuid"] = effectiveParent
		} else {
			line["parentUuid"] = nil
		}

		line["isSidechain"] = false
		if opts.Team != nil {
			if opts.Team.TeamName != "" {
				line["teamName"] = opts.Team.TeamName
			}
			if opts.Team.AgentName != "" {
				line["agentName"] = opts.Team.AgentName
			}
		}
		if m.Type == types.MessageTypeUser && strings.TrimSpace(s.PromptID) != "" {
			line["promptId"] = strings.TrimSpace(s.PromptID)
		}
		line["userType"] = s.userType()
		if entry != "" {
			line["entrypoint"] = entry
		}
		line["cwd"] = s.cwdStamp()
		line["sessionId"] = s.SessionID
		line["timestamp"] = time.Now().UTC().Format(time.RFC3339Nano)
		line["version"] = s.version()
		if gitBr != "" {
			line["gitBranch"] = gitBr
		}
		if slug != "" {
			line["slug"] = slug
		}

		if s.FileHistorySnapshotOnUser && m.Type == types.MessageTypeUser && !isUserMetaMessage(m) && !fileCheckpointingDisabled() {
			if !s.FileHistorySnapshotOnce || !s.fileHistoryStubEmitted {
				if err := appendTSFileHistorySnapshotLine(path, m.UUID, false); err != nil {
					return "", err
				}
				s.fileHistoryStubEmitted = true
			}
		}
		if err := appendJSONL(path, line); err != nil {
			return "", err
		}
		existing[m.UUID] = struct{}{}
		if IsChainParticipant(m) {
			parentUUID = m.UUID
			lastRecorded = m.UUID
		}
	}

	// TS insertMessageChain: cache last prompt from this batch; reAppendSessionMetadata
	// appends last-prompt at EOF whenever new lines land and cache is set.
	if text := FirstMeaningfulUserMessageTextContent(cleaned); text != "" {
		s.currentSessionLastPrompt = FlattenLastPromptCache(text)
	}
	if len(newMsgs) > 0 && strings.TrimSpace(s.currentSessionLastPrompt) != "" {
		if err := appendJSONL(path, map[string]any{
			"type":       "last-prompt",
			"lastPrompt": s.currentSessionLastPrompt,
			"sessionId":  s.SessionID,
		}); err != nil {
			return "", err
		}
	}

	s.uuidCachePath = path
	s.uuidCache = existing
	if st, err := os.Stat(path); err == nil {
		s.uuidCacheSize = st.Size()
	}

	if lastRecorded != "" {
		return lastRecorded, nil
	}
	if startingParent != "" {
		return startingParent, nil
	}
	return "", nil
}

func (s *Store) loadUUIDSet(path string) (map[string]struct{}, error) {
	if st, err := os.Stat(path); err == nil {
		if s.uuidCachePath == path && s.uuidCache != nil && st.Size() == s.uuidCacheSize {
			cp := make(map[string]struct{}, len(s.uuidCache))
			for k := range s.uuidCache {
				cp[k] = struct{}{}
			}
			return cp, nil
		}
	}
	ids, err := MessageUUIDsFromJSONL(path)
	if err != nil {
		return nil, err
	}
	return ids, nil
}

// RecordSidechainTranscript mirrors recordSidechainTranscript.
func (s *Store) RecordSidechainTranscript(ctx context.Context, agentID string, messages []types.Message, startingParentUUID string) error {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.shouldSkipPersistence() || strings.TrimSpace(agentID) == "" {
		return nil
	}
	if sid := strings.TrimSpace(s.SessionID); sid == "" || !IsValidUUID(sid) {
		return fmt.Errorf("sessiontranscript: RecordSidechainTranscript needs valid SessionID UUID")
	}
	path := s.sidechainPath(agentID)
	cleaned := CleanMessagesForLogging(messages, messages, s.userType())
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	parentUUID := strings.TrimSpace(startingParentUUID)
	gitBr := gitBranchForDir(s.cwdStamp())
	slug := strings.TrimSpace(s.PlanSlug)
	entry := strings.TrimSpace(s.Entrypoint)
	if entry == "" {
		entry = strings.TrimSpace(os.Getenv("CLAUDE_CODE_ENTRYPOINT"))
	}

	for _, m := range cleaned {
		if m.UUID == "" {
			continue
		}
		line, err := messageToMap(m)
		if err != nil {
			return err
		}
		isCompact := IsCompactBoundaryMessage(m)
		effectiveParent := parentUUID
		if m.Type == types.MessageTypeUser && m.SourceToolAssistantUUID != nil && strings.TrimSpace(*m.SourceToolAssistantUUID) != "" {
			effectiveParent = strings.TrimSpace(*m.SourceToolAssistantUUID)
		}
		if isCompact {
			line["parentUuid"] = nil
			if parentUUID != "" {
				line["logicalParentUuid"] = parentUUID
			}
		} else if effectiveParent != "" {
			line["parentUuid"] = effectiveParent
		} else {
			line["parentUuid"] = nil
		}
		line["isSidechain"] = true
		line["agentId"] = agentID
		line["userType"] = s.userType()
		if entry != "" {
			line["entrypoint"] = entry
		}
		line["cwd"] = s.cwdStamp()
		line["sessionId"] = s.SessionID
		line["timestamp"] = time.Now().UTC().Format(time.RFC3339Nano)
		line["version"] = s.version()
		if gitBr != "" {
			line["gitBranch"] = gitBr
		}
		if slug != "" {
			line["slug"] = slug
		}
		if err := appendJSONL(path, line); err != nil {
			return err
		}
		if IsChainParticipant(m) {
			parentUUID = m.UUID
		}
	}
	return nil
}

// ClearMessageUUIDCache drops in-memory UUID cache (TS clearSessionMessagesCache).
func (s *Store) ClearMessageUUIDCache() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.uuidCache = nil
	s.uuidCachePath = ""
	s.uuidCacheSize = 0
}
