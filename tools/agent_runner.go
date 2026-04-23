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

	processuserinput "goc/conversation-runtime/process-user-input"
	"goc/conversation-runtime/query"
	"goc/sessiontranscript"
	"goc/toolexecution"
	"goc/toolpool"
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

func executeAgent(ctx context.Context, cfg AgentRuntimeConfig, s *AgentSession, prompt string, history []types.Message) string {
	if ctx == nil {
		ctx = context.Background()
	}
	msg := strings.TrimSpace(prompt)
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
	msgs := append(append([]types.Message{}, history...), pr.Messages...)

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

	qdeps := query.ProductionDeps()
	toolCfg := Config{
		Roots:        []string{cfg.ProjectRoot},
		WorkDir:      s.WorkDir,
		ProjectRoot:  cfg.ProjectRoot,
		SessionID:    cfg.SessionID,
		AskAutoFirst: true,
	}
	qdeps.ToolexecutionDeps = toolexecution.ExecutionDeps{
		InvokeTool: func(ctx context.Context, name, _ string, input json.RawMessage) (string, bool, error) {
			return Run(ctx, name, input, toolCfg)
		},
		MainLoopModel: tc.Options.MainLoopModel,
	}

	// Build system prompt components
	systemPromptParts := []string{}
	
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
	
	// Add skills information if present
	if len(s.Skills) > 0 {
		skillsText := "Available skills: " + strings.Join(s.Skills, ", ")
		systemPromptParts = append(systemPromptParts, skillsText)
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
	var assistantChunks []string
	for y, qerr := range query.Query(ctx, qp) {
		if qerr != nil {
			return runAgentNow(s, msg)
		}
		if y.Message == nil || y.Message.Type != types.MessageTypeAssistant {
			continue
		}
		if text := assistantMessageText(*y.Message); strings.TrimSpace(text) != "" {
			assistantChunks = append(assistantChunks, text)
		}
	}
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

func persistSidechain(cfg AgentRuntimeConfig, session *AgentSession, userPrompt, assistantOutput string) {
	if strings.TrimSpace(cfg.SessionID) == "" {
		return
	}
	st := &sessiontranscript.Store{
		SessionID:   cfg.SessionID,
		OriginalCwd: cfg.ProjectRoot,
		Cwd:         session.WorkDir,
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	userMsg := types.Message{
		Type: types.MessageTypeUser,
		UUID: fmt.Sprintf("agent-user-%d", time.Now().UnixNano()),
	}
	assistantMsg := types.Message{
		Type:      types.MessageTypeAssistant,
		UUID:      fmt.Sprintf("agent-assistant-%d", time.Now().UnixNano()),
		Timestamp: &now,
	}
	userContent, _ := json.Marshal(map[string]any{"role": "user", "content": userPrompt})
	asstContent, _ := json.Marshal(map[string]any{"role": "assistant", "content": assistantOutput})
	userMsg.Message = userContent
	assistantMsg.Message = asstContent
	_ = st.RecordSidechainTranscript(nil, session.ID, []types.Message{userMsg, assistantMsg}, "")
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
