package tools

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"goc/commands"
	processuserinput "goc/conversation-runtime/process-user-input"
	"goc/conversation-runtime/query"
	"goc/hookexec"
	"goc/sessiontranscript"
	"goc/slashresolve"
	"goc/tools/toolexecution"
	"goc/tools/toolpool"
	"goc/types"
)

var (
	agentSessionsMu sync.RWMutex
	agentSessions   = map[string]*AgentSession{}
	agentByName     = map[string]string{}
)

func agentMetaDir(cfg AgentRuntimeConfig) string {
	return filepath.Join(cfg.ProjectRoot, ".claude", ".gou-agents")
}

func persistAgentMetadata(cfg AgentRuntimeConfig, s *AgentSession) {
	if strings.TrimSpace(cfg.ProjectRoot) == "" || s == nil {
		return
	}
	_ = os.MkdirAll(agentMetaDir(cfg), 0o700)
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(filepath.Join(agentMetaDir(cfg), s.ID+".json"), append(b, '\n'), 0o600)
}

func loadAgentMetadata(cfg AgentRuntimeConfig, id string) *AgentSession {
	if strings.TrimSpace(cfg.ProjectRoot) == "" || strings.TrimSpace(id) == "" {
		return nil
	}
	b, err := os.ReadFile(filepath.Join(agentMetaDir(cfg), id+".json"))
	if err != nil {
		return nil
	}
	var s AgentSession
	if json.Unmarshal(b, &s) != nil {
		return nil
	}
	return &s
}

func loadAgentMetadataByName(cfg AgentRuntimeConfig, name string) *AgentSession {
	name = strings.TrimSpace(name)
	if name == "" || strings.TrimSpace(cfg.ProjectRoot) == "" {
		return nil
	}
	entries, err := os.ReadDir(agentMetaDir(cfg))
	if err != nil {
		return nil
	}
	for _, ent := range entries {
		if ent.IsDir() || !strings.HasSuffix(ent.Name(), ".json") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(agentMetaDir(cfg), ent.Name()))
		if err != nil {
			continue
		}
		var s AgentSession
		if json.Unmarshal(b, &s) != nil {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(s.Name), name) {
			return &s
		}
	}
	return nil
}

func runAgentNow(s *AgentSession, message string) string {
	msg := strings.TrimSpace(message)
	if msg == "" {
		msg = s.Prompt
	}
	out := fmt.Sprintf("agent[%s] completed: %s", s.AgentType, msg)
	s.LastOutput = out
	s.UpdatedAt = time.Now().UTC()
	return out
}

var (
	agentToolsJSONOnce sync.Once
	agentToolsJSON     []byte
	agentToolNames     []string
)

func loadAgentToolsJSON() []byte {
	agentToolsJSONOnce.Do(func() {
		// Use Go native tool specs instead of reading from JSON file
		specs := toolpool.ToolSpecsFromGoWire()
		var apiSpecs []toolpool.APIToolDefinition
		opts := toolpool.DefaultToolToAPISchemaOptionsFromEnv()
		
		for _, spec := range specs {
			apiSpec := toolpool.ToolToAPISchema(spec, opts)
			apiSpecs = append(apiSpecs, apiSpec)
			agentToolNames = append(agentToolNames, spec.Name)
		}
		
		// Marshal to JSON for compatibility with existing code
		if b, err := json.Marshal(apiSpecs); err == nil {
			agentToolsJSON = b
		}
	})
	return agentToolsJSON
}

func availableAgentToolNames() []string {
	_ = loadAgentToolsJSON()
	return append([]string(nil), agentToolNames...)
}

func filterToolsForAgent(raw json.RawMessage, allowed []string) json.RawMessage {
	if len(raw) == 0 {
		return nil
	}
	if len(allowed) == 0 {
		return raw
	}
	allowedSet := map[string]struct{}{}
	for _, n := range allowed {
		if strings.TrimSpace(n) != "" {
			allowedSet[n] = struct{}{}
		}
	}
	var defs []map[string]any
	if err := json.Unmarshal(raw, &defs); err != nil {
		return raw
	}
	out := make([]map[string]any, 0, len(defs))
	for _, d := range defs {
		name, _ := d["name"].(string)
		if _, ok := allowedSet[name]; ok {
			out = append(out, d)
		}
	}
	if b, err := json.Marshal(out); err == nil {
		return b
	}
	return raw
}

// executeAgentOpts carries optional overrides for the fork path.
type executeAgentOpts struct {
	// ForkSystemPrompt is the parent's system prompt to use instead of building
	// one from the agent definition (cache-identical API prefixes on fork).
	ForkSystemPrompt []string
	// ForkMessages are pre-built messages (from buildForkedMessages) to use
	// instead of calling ProcessTextPrompt on the prompt string.
	ForkMessages []types.Message
}

func executeAgent(ctx context.Context, cfg AgentRuntimeConfig, s *AgentSession, prompt string, history []types.Message) string {
	return executeAgentWithOpts(ctx, cfg, s, prompt, history, executeAgentOpts{})
}

func executeAgentWithOpts(ctx context.Context, cfg AgentRuntimeConfig, s *AgentSession, prompt string, history []types.Message, opts executeAgentOpts) string {
	if ctx == nil {
		ctx = context.Background()
	}

	// Register agent frontmatter hooks as session-scoped hooks (TS registerFrontmatterHooks).
	if len(s.Hooks) > 0 {
		hookexec.RegisterFrontmatterHooks(s.ID, s.Hooks, true)
	}
	// Cleanup session hooks when agent finishes (TS clearSessionHooks).
	defer hookexec.ClearAgentSessionHooks(s.ID)

	// Reapply tool_result content replacements to sidechain history on resume,
	// restoring tool_result content that was compacted during the prior execution.
	// TS parity: ReapplyToolResultReplacementsFromState in query.ts resume path.
	if len(s.ContentReplacementState) > 0 && len(history) > 0 {
		history = query.ReapplyToolResultReplacementsFromState(history, s.ContentReplacementState)
	}

	var msgs []types.Message
	msg := strings.TrimSpace(prompt)
	if len(opts.ForkMessages) > 0 {
		// Fork path: use pre-built messages from buildForkedMessages
		msgs = append(append([]types.Message{}, history...), opts.ForkMessages...)
		msg = prompt
	} else {
		if msg == "" {
			msg = s.Prompt
		}
		if msg == "" {
			msg = "continue"
		}

		var pm *types.PermissionMode
		if strings.TrimSpace(s.PermissionMode) != "" {
			mode := types.PermissionMode(s.PermissionMode)
			pm = &mode
		}
		pr, err := processuserinput.ProcessTextPrompt(msg, nil, nil, nil, nil, nil, pm, nil, nil)
		if err != nil || len(pr.Messages) == 0 {
			return runAgentNow(s, msg)
		}
		msgs = append(append([]types.Message{}, history...), pr.Messages...)
	}

	// Preload skill messages and insert between history and current prompt.
	if skillMsgs := preloadAgentSkills(ctx, s, s.WorkDir, cfg.SessionID); len(skillMsgs) > 0 {
		offset := len(history)
		combined := make([]types.Message, 0, len(msgs)+len(skillMsgs))
		combined = append(combined, msgs[:offset]...)
		combined = append(combined, skillMsgs...)
		combined = append(combined, msgs[offset:]...)
		msgs = combined
	}

	// --- Incremental sidechain persistence ---
	// Create a Store once and persist each chain-participant message as it is yielded,
	// matching TS recordSidechainTranscript call-per-turn behavior.
	var sidechainStore *sessiontranscript.Store
	var sidechainParentUUID string
	if strings.TrimSpace(cfg.SessionID) != "" && sessiontranscript.IsValidUUID(cfg.SessionID) {
		sidechainStore = &sessiontranscript.Store{
			SessionID:   cfg.SessionID,
			OriginalCwd: cfg.ProjectRoot,
			Cwd:         s.WorkDir,
		}
		// Persist initial messages on first execution (history is empty).
		// Skip meta messages (skills, system-injected) since they are regenerated on resume.
		if len(history) == 0 {
			var initialMsgs []types.Message
			for _, m := range msgs {
				if m.Type == types.MessageTypeProgress || (m.IsMeta != nil && *m.IsMeta) {
					continue
				}
				initialMsgs = append(initialMsgs, m)
			}
			if len(initialMsgs) > 0 {
				_ = sidechainStore.RecordSidechainTranscript(nil, s.ID, initialMsgs, "")
				// TS parity: also write sidechain messages to main session JSONL.
				rOpts := sessiontranscript.RecordOpts{IsSidechain: true}
				if s.TeamName != "" || s.Name != "" {
					rOpts.Team = &sessiontranscript.TeamInfo{TeamName: s.TeamName, AgentName: s.Name}
				}
				_, _ = sidechainStore.RecordTranscript(nil, initialMsgs, rOpts)
				for _, m := range initialMsgs {
					if sessiontranscript.IsChainParticipant(m) {
						sidechainParentUUID = m.UUID
					}
				}
			}
		} else {
			// Resume path: use the last chain participant UUID from history as the
			// starting parent UUID so new yield messages are chained correctly.
			for i := len(history) - 1; i >= 0; i-- {
				if sessiontranscript.IsChainParticipant(history[i]) {
					sidechainParentUUID = history[i].UUID
					break
				}
			}
		}
	}

	tc := types.ToolUseContext{}
	tc.Options.MainLoopModel = strings.TrimSpace(s.Model)
	tc.Options.IsNonInteractiveSession = true
	if tj := loadAgentToolsJSON(); len(tj) > 0 {
		tc.Options.Tools = filterToolsForAgent(json.RawMessage(tj), s.AllowedTools)
	}
	if s.ID != "" {
		id := s.ID
		tc.AgentID = &id
	}
	if s.AgentType != "" {
		at := s.AgentType
		tc.AgentType = &at
	}
	tc.ConversationID = &cfg.SessionID
	tc.ContentReplacementState = s.ContentReplacementState

	qdeps := query.ProductionDeps()
	toolCfg := Config{
		Roots:        []string{cfg.ProjectRoot},
		WorkDir:      s.WorkDir,
		ProjectRoot:  cfg.ProjectRoot,
		SessionID:    cfg.SessionID,
		AskAutoFirst: true,
		TeamName:     s.TeamName,
		AgentName:    s.Name,
		AgentID:      s.ID,
	}
	pretoolHook := hookexec.AgentPreToolUseHookFromSession(s.ID, s.WorkDir)
	qdeps.ToolexecutionDeps = toolexecution.ExecutionDeps{
		InvokeTool: func(ctx context.Context, name, _ string, input json.RawMessage) (string, bool, error) {
			return Run(ctx, name, input, toolCfg)
		},
		MainLoopModel:  tc.Options.MainLoopModel,
		PreToolUseHook: pretoolHook,
	}

	// Wrap Autocompact to capture updated ContentReplacementState so it can be
	// persisted back to AgentSession for the next resume (TS applyAutocompactSideEffects).
	var captureCRS json.RawMessage
	origAC := qdeps.Autocompact
	if origAC != nil {
		qdeps.Autocompact = func(ctx context.Context, in *query.AutocompactInput) (*query.AutocompactResult, error) {
			res, err := origAC(ctx, in)
			if err == nil && res != nil && len(res.UpdatedContentReplacementState) > 0 {
				captureCRS = append(json.RawMessage(nil), res.UpdatedContentReplacementState...)
			}
			return res, err
		}
	}

	// Build system prompt components
	var systemPromptParts []string
	if len(opts.ForkSystemPrompt) > 0 {
		// Fork path: inherit parent's rendered system prompt for cache-identical prefixes
		systemPromptParts = opts.ForkSystemPrompt
	} else {
		systemPromptParts = []string{}

		// Add agent description (whenToUse)
		if strings.TrimSpace(s.Description) != "" {
			systemPromptParts = append(systemPromptParts, s.Description)
		}

		// Add agent's custom system prompt
		if strings.TrimSpace(s.SystemPrompt) != "" {
			systemPromptParts = append(systemPromptParts, s.SystemPrompt)
		}

		// Add critical system reminder if present
		if strings.TrimSpace(s.CriticalSystemReminderExperimental) != "" {
			systemPromptParts = append(systemPromptParts, s.CriticalSystemReminderExperimental)
		}

		// Add MCP server information if present
		if len(s.AvailableMcpServers) > 0 {
			mcpText := "Available MCP servers: " + strings.Join(s.AvailableMcpServers, ", ")
			systemPromptParts = append(systemPromptParts, mcpText)
			if len(s.RequiredMcpServers) > 0 {
				requiredMcpText := "Required MCP servers for this agent: " + strings.Join(s.RequiredMcpServers, ", ")
				systemPromptParts = append(systemPromptParts, requiredMcpText)
			}
		}

		// Inject unread teammate mailbox messages into the agent's context
		agentName := strings.TrimSpace(s.Name)
		teamName := strings.TrimSpace(s.TeamName)
		if agentName != "" && teamName != "" {
			if msgs, err := readMailbox(agentName, teamName); err == nil && len(msgs) > 0 {
				formatted := formatTeammateMessages(msgs)
				if formatted != "" {
					systemPromptParts = append(systemPromptParts,
						"You have unread teammate messages:\n"+formatted)
				}
				// Mark messages as read after injecting into context
				_ = markMessagesAsRead(agentName, teamName)
			}
		}
	}
	
	qp := query.QueryParams{
		Messages:        msgs,
		SystemPrompt:    query.AsSystemPrompt(systemPromptParts),
		ToolUseContext:  tc,
		QuerySource:     types.QuerySource("agent"),
		StreamingParity: true,
		Deps:            &qdeps,
	}
	if s.MaxTurns > 0 {
		mt := s.MaxTurns
		qp.MaxTurns = &mt
	}
	processuserinput.ApplyQueryHostEnvGates(&qp)

	// Shared message buffer for summarization goroutine.
	msgBuf := &sharedMessageBuffer{}
	msgBuf.Append(msgs...)

	// Start background summarization when the agent has a progress callback.
	var stopSummary func()
	if cfg.ProgressCallback != nil && s.ID != "" && s.AgentType != "" {
		stopSummary = startAgentSummarization(s, cfg, msgBuf)
	}

	var assistantChunks []string
	var progressMsgs []json.RawMessage
	for y, qerr := range query.Query(ctx, qp) {
		if qerr != nil {
			if stopSummary != nil {
				stopSummary()
			}
			return runAgentNow(s, msg)
		}
		if y.Message == nil {
			continue
		}
		// Feed yield messages into the shared buffer for summarization.
		msgBuf.Append(*y.Message)

		// Incremental sidechain persistence: persist every chain-participant message
		// as it is yielded from the query loop, matching TS recordSidechainTranscript.
		if sidechainStore != nil && y.Message.Type != types.MessageTypeProgress {
			_ = sidechainStore.RecordSidechainTranscript(nil, s.ID, []types.Message{*y.Message}, sidechainParentUUID)
			// TS parity: also write sidechain messages to main session JSONL.
			rOpts := sessiontranscript.RecordOpts{StartingParentUUID: sidechainParentUUID, IsSidechain: true}
			if s.TeamName != "" || s.Name != "" {
				rOpts.Team = &sessiontranscript.TeamInfo{TeamName: s.TeamName, AgentName: s.Name}
			}
			_, _ = sidechainStore.RecordTranscript(nil, []types.Message{*y.Message}, rOpts)
			if sessiontranscript.IsChainParticipant(*y.Message) {
				sidechainParentUUID = y.Message.UUID
			}
		}

		if y.Message.Type == types.MessageTypeProgress {
			if b, err := json.Marshal(y.Message); err == nil {
				progressMsgs = append(progressMsgs, b)
			}
			if cfg.ProgressCallback != nil {
				cfg.ProgressCallback(y.Message)
			}
			continue
		}
		if y.Message.Type != types.MessageTypeAssistant {
			continue
		}
		if text := assistantMessageText(*y.Message); strings.TrimSpace(text) != "" {
			assistantChunks = append(assistantChunks, text)
		}
	}
	// Stop summarization when the agent's main query loop completes.
	if stopSummary != nil {
		stopSummary()
	}
	// Persist updated ContentReplacementState for next resume.
	if len(captureCRS) > 0 {
		s.ContentReplacementState = captureCRS
	}
	s.ProgressMessages = progressMsgs
	if len(assistantChunks) == 0 {
		return runAgentNow(s, msg)
	}
	out := strings.Join(assistantChunks, "\n")
	s.LastOutput = out
	s.UpdatedAt = time.Now().UTC()
	return out
}

func assistantMessageText(m types.Message) string {
	if len(m.Message) == 0 {
		return ""
	}
	var payload struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(m.Message, &payload); err != nil {
		return ""
	}
	parts := make([]string, 0, len(payload.Content))
	for _, c := range payload.Content {
		if c.Type == "text" && strings.TrimSpace(c.Text) != "" {
			parts = append(parts, c.Text)
		}
	}
	return strings.Join(parts, "\n")
}

// preloadAgentSkills loads skill content for each skill name in s.Skills and returns
// isMeta user messages that are prepended to the agent's conversation context.
// TS parity: runAgent.ts lines 583-651 (skills preloading)
func preloadAgentSkills(ctx context.Context, s *AgentSession, workDir string, sessionID string) []types.Message {
	if len(s.Skills) == 0 {
		return nil
	}

	cmds, err := commands.LoadAllCommands(ctx, workDir, commands.DefaultLoadOptions())
	if err != nil {
		return nil
	}

	var skillMsgs []types.Message
	for _, skillName := range s.Skills {
		cmd := resolveAgentSkillName(skillName, cmds)
		if cmd == nil || cmd.Type != "prompt" {
			continue
		}

		content := getAgentSkillContent(cmd, sessionID)
		if content == "" {
			continue
		}

		// TS formatSkillLoadingMetadata: <command-message>name</command-message>\n<command-name>name</command-name>\n<skill-format>true</skill-format>
		metadata := fmt.Sprintf(
			"<command-message>%s</command-message>\n<command-name>%s</command-name>\n<skill-format>true</skill-format>\n\n%s",
			cmd.Name, cmd.Name, content,
		)

		trueVal := true
		msgInner, _ := json.Marshal(map[string]any{"role": "user", "content": metadata})
		msg := types.Message{
			Type:    types.MessageTypeUser,
			UUID:    sessiontranscript.NewUUID(),
			Message: json.RawMessage(msgInner),
			IsMeta:  &trueVal,
		}
		skillMsgs = append(skillMsgs, msg)
	}

	return skillMsgs
}

// resolveAgentSkillName resolves a skill name from agent frontmatter against loaded commands.
// TS parity: runAgent.ts resolveSkillName (3 strategies: exact, pluginPrefix:name, :name suffix)
func resolveAgentSkillName(name string, cmds []types.Command) *types.Command {
	// Strategy 1: exact match on Name or Aliases
	for _, cmd := range cmds {
		if strings.EqualFold(cmd.Name, name) {
			return &cmd
		}
		for _, alias := range cmd.Aliases {
			if strings.EqualFold(alias, name) {
				return &cmd
			}
		}
	}

	// Strategy 2: pluginPrefix:name match (e.g., "notion:read" matches command named "read")
	if colonIdx := strings.Index(name, ":"); colonIdx > 0 {
		suffix := name[colonIdx+1:]
		for _, cmd := range cmds {
			if strings.EqualFold(cmd.Name, suffix) {
				return &cmd
			}
			for _, alias := range cmd.Aliases {
				if strings.EqualFold(alias, suffix) {
					return &cmd
				}
			}
		}
	}

	// Strategy 3: :name suffix match
	if strings.HasPrefix(name, ":") {
		suffix := name[1:]
		for _, cmd := range cmds {
			if strings.EqualFold(cmd.Name, suffix) {
				return &cmd
			}
			for _, alias := range cmd.Aliases {
				if strings.EqualFold(alias, suffix) {
					return &cmd
				}
			}
		}
	}

	return nil
}

// getAgentSkillContent retrieves the body text of a skill command.
// TS parity: getPromptForCommand for skill preloading (bundled and disk-based)
func getAgentSkillContent(cmd *types.Command, sessionID string) string {
	// Bundled skills: resolve via slashresolve.ResolveBundledSkill
	if slashresolve.IsBundledPrompt(*cmd) {
		result, err := slashresolve.ResolveBundledSkill(*cmd, "", sessionID, nil)
		if err == nil {
			return result.UserText
		}
		return ""
	}

	// Disk-based skills (SKILL.md directories): read body via SplitYAMLFrontmatter
	if cmd.SkillRoot != nil && *cmd.SkillRoot != "" {
		mdPath := filepath.Join(*cmd.SkillRoot, "SKILL.md")
		raw, err := os.ReadFile(mdPath)
		if err != nil {
			return ""
		}
		_, body, ok := commands.SplitYAMLFrontmatter(raw)
		if !ok {
			return ""
		}
		return string(body)
	}

	return ""
}

func writeBackgroundOutput(tasksDir, agentID, output string) (string, error) {
	if err := os.MkdirAll(tasksDir, 0o700); err != nil {
		return "", err
	}
	p := filepath.Join(tasksDir, agentID+".output")
	if err := os.WriteFile(p, []byte(output), 0o600); err != nil {
		return "", err
	}
	return p, nil
}

func writeBackgroundStatus(tasksDir, agentID, status, message string, success bool) {
	if strings.TrimSpace(tasksDir) == "" || strings.TrimSpace(agentID) == "" {
		return
	}
	_ = os.MkdirAll(tasksDir, 0o700)
	payload := map[string]any{
		"task_id":   agentID,
		"status":    status,
		"success":   success,
		"message":   message,
		"updatedAt": time.Now().UTC().Format(time.RFC3339Nano),
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return
	}
	_ = os.WriteFile(filepath.Join(tasksDir, agentID+".status.json"), append(b, '\n'), 0o600)
}

func isTaskStopRequested(tasksDir, agentID string) bool {
	if strings.TrimSpace(tasksDir) == "" || strings.TrimSpace(agentID) == "" {
		return false
	}
	_, err := os.Stat(filepath.Join(tasksDir, agentID+".stop"))
	return err == nil
}

func loadSidechainMessages(cfg AgentRuntimeConfig, agentID string) []types.Message {
	if strings.TrimSpace(cfg.SessionID) == "" || strings.TrimSpace(agentID) == "" {
		return nil
	}
	p := sessiontranscript.AgentTranscriptPath(
		cfg.SessionID,
		cfg.ProjectRoot,
		"",
		sessiontranscript.ConfigHomeDir(),
		agentID,
		"",
	)
	f, err := os.Open(p)
	if err != nil {
		return nil
	}
	defer f.Close()
	var out []types.Message
	sc := bufio.NewScanner(f)
	buf := make([]byte, 0, 64*1024)
	sc.Buffer(buf, 4*1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var m types.Message
		if err := json.Unmarshal([]byte(line), &m); err == nil && strings.TrimSpace(m.UUID) != "" {
			out = append(out, m)
		}
	}
	return out
}
