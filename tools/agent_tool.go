package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"goc/claudemd"
	"goc/commands"
	"goc/tools/toolpool"
	"goc/types"
)

func getenv(k string) string { return os.Getenv(k) }

// resolveAgentWorkDirFromCwd mirrors TS: cwd must be an absolute path to a directory
// (mutually exclusive with isolation "worktree" / "remote" — checked in [RunAgentTool]).
func resolveAgentWorkDirFromCwd(cwd string) (string, error) {
	clean := filepath.Clean(strings.TrimSpace(cwd))
	if clean == "" || clean == "." {
		return "", fmt.Errorf("cwd is empty")
	}
	if !filepath.IsAbs(clean) {
		return "", fmt.Errorf("cwd must be an absolute path, got %q", cwd)
	}
	st, err := os.Stat(clean)
	if err != nil {
		return "", fmt.Errorf("cwd: %w", err)
	}
	if !st.IsDir() {
		return "", fmt.Errorf("cwd is not a directory: %q", clean)
	}
	return clean, nil
}

func RunAgentTool(raw []byte, cfg AgentRuntimeConfig) (string, bool, error) {
	var in AgentToolInput
	if err := json.Unmarshal(raw, &in); err != nil {
		return "", true, err
	}
	if strings.TrimSpace(in.Resume) != "" {
		return ResumeAgentTool(raw, cfg)
	}
	if strings.TrimSpace(in.Prompt) == "" {
		return "", true, fmt.Errorf("prompt is required")
	}

	messages := cfg.Messages
	forkEnabled := isForkSubagentEnabled()
	isForkPath := strings.TrimSpace(in.SubagentType) == "" && forkEnabled

	// Recursive fork guard
	if isForkPath && isInForkChild(messages) {
		return "", true, fmt.Errorf("Fork is not available inside a forked worker. Complete your task directly using your tools.")
	}

	var selected AgentDefinition
	if isForkPath {
		selected = ForkAgentDef()
	} else {
		defs := LoadAgentDefinitionsForCwd(cfg.ProjectRoot)
		selected = ResolveAgentDefinition(defs, in.SubagentType)
	}
	if !AgentMeetsRequiredMCPServers(selected, cfg.AvailableMCPServers) {
		resp, _ := json.Marshal(AgentToolResponse{
			Data: AgentToolResponseData{
				Success:   false,
				AgentType: selected.AgentType,
				Message:   fmt.Sprintf("required MCP servers unavailable: %s", strings.Join(selected.RequiredMcpServers, ", ")),
			},
		})
		return string(resp), false, nil
	}
	model := strings.TrimSpace(in.Model)
	if model == "" {
		model = selected.Model
	}
	agentID := fmt.Sprintf("agent-%d", time.Now().UnixNano())
	name := strings.TrimSpace(in.Name)
	if name == "" {
		name = strings.ToLower(strings.TrimSpace(selected.AgentType))
	}
	pm := selected.PermissionMode
	if m := strings.TrimSpace(in.Mode); m != "" {
		pm = m
	}
	s := &AgentSession{
		ID:                                 agentID,
		Name:                               name,
		TeamName:                           strings.TrimSpace(in.TeamName),
		AgentType:                          selected.AgentType,
		Description:                        strings.TrimSpace(in.Description),
		Model:                              model,
		PermissionMode:                     pm,
		MaxTurns:                           selected.MaxTurns,
		AllowedTools:                       ResolveAllowedTools(selected, availableAgentToolNames()),
		Skills:                             append([]string(nil), selected.Skills...),
		RequiredMcpServers:                 append([]string(nil), selected.RequiredMcpServers...),
		AvailableMcpServers:                append([]string(nil), cfg.AvailableMCPServers...),
		Prompt:                             in.Prompt,
		WorkDir:                            cfg.WorkDir,
		ProjectRoot:                        cfg.ProjectRoot,
		Isolation:                          strings.TrimSpace(in.Isolation),
		SystemPrompt:                       selected.SystemPrompt,
		Memory:                            selected.Memory,
		OmitClaudeMd:                       selected.OmitClaudeMd,
		Hooks:                             selected.Hooks,
		CreatedAt:                          time.Now().UTC(),
		UpdatedAt:                          time.Now().UTC(),
	}
	if s.Description == "" {
		s.Description = selected.WhenToUse
	}
	if s.Isolation == "" {
		s.Isolation = strings.TrimSpace(selected.Isolation)
	}
	if s.TeamName == "" {
		s.TeamName = strings.TrimSpace(cfg.TeamName)
	}

	// Append agent memory prompt if memory scope is configured and auto-memory is enabled.
	if s.Memory != "" && claudemd.IsAutoMemoryEnabled() {
		scope := claudemd.AgentMemoryScope(s.Memory)
		memPrompt := claudemd.LoadAgentMemoryPrompt(s.AgentType, scope)
		if memPrompt != "" {
			if s.SystemPrompt != "" {
				s.SystemPrompt += "\n\n" + memPrompt
			} else {
				s.SystemPrompt = memPrompt
			}
		}
	}

	// Register the spawned agent with the team roster if team context is set.
	if s.TeamName != "" {
		_ = addTeamMember(s.TeamName, TeamFileMember{
			AgentID:   s.ID,
			Name:      s.Name,
			AgentType: s.AgentType,
			Model:     s.Model,
			Prompt:    s.Prompt,
			JoinedAt:  time.Now().UnixMilli(),
			CWD:       s.WorkDir,
			SessionID: cfg.SessionID,
			IsActive:  true,
		})
		// Subscribe to all existing team members by default.
		if tf, err := readTeamFile(s.TeamName); err == nil && tf != nil {
			var subs []string
			for _, m := range tf.Members {
				if m.AgentID != s.ID {
					subs = append(subs, m.AgentID)
				}
			}
			if len(subs) > 0 {
				_ = addTeamMember(s.TeamName, TeamFileMember{
					AgentID:       s.ID,
					Name:          s.Name,
					JoinedAt:      time.Now().UnixMilli(),
					Subscriptions: subs,
					IsActive:      true,
				})
			}
		}
	}

	cwdTrim := strings.TrimSpace(in.Cwd)
	if cwdTrim != "" {
		if s.Isolation == "worktree" {
			return "", true, fmt.Errorf("cwd is mutually exclusive with isolation %q", "worktree")
		}
		if s.Isolation == "remote" {
			return "", true, fmt.Errorf("cwd is mutually exclusive with isolation %q", "remote")
		}
	}

	switch s.Isolation {
	case "":
	case "worktree":
		wp, err := createWorktree(cfg.ProjectRoot, name)
		if err != nil {
			return "", true, err
		}
		s.WorktreePath = wp
		s.WorkDir = wp
	case "remote":
		if err := requireRemoteBackend(); err != nil {
			resp, _ := json.Marshal(AgentToolResponse{
				Data: AgentToolResponseData{
					Success:   false,
					AgentID:   s.ID,
					AgentType: s.AgentType,
					Message:   err.Error(),
				},
			})
			return string(resp), false, nil
		}
	default:
		return "", true, fmt.Errorf("unsupported isolation %q", s.Isolation)
	}

	if s.Isolation != "worktree" && cwdTrim != "" {
		abs, err := resolveAgentWorkDirFromCwd(cwdTrim)
		if err != nil {
			return "", true, err
		}
		s.WorkDir = abs
	}

	agentSessionsMu.Lock()
	agentSessions[s.ID] = s
	agentByName[s.Name] = s.ID
	agentSessionsMu.Unlock()
	persistAgentMetadata(cfg, s)

	// Persist fork system prompt so ResumeAgentTool can reconstruct cache-identical prefixes.
	if isForkPath {
		s.ForkSystemPrompt = cfg.SystemPrompt
	}

	// Build fork execution options if this is a fork path.
	var forkOpts *executeAgentOpts
	if isForkPath {
		opts := &executeAgentOpts{
			ForkSystemPrompt: cfg.SystemPrompt,
		}
		var parentAssistantMsg types.Message
		for i := len(cfg.Messages) - 1; i >= 0; i-- {
			if cfg.Messages[i].Type == types.MessageTypeAssistant {
				parentAssistantMsg = cfg.Messages[i]
				break
			}
		}
		forkMsgs := buildForkedMessages(in.Prompt, parentAssistantMsg)
		// Load and prepend any existing sidechain history (resume scenario).
		if existing := loadSidechainMessages(cfg, agentID); len(existing) > 0 {
			forkMsgs = append(existing, forkMsgs...)
		}
		opts.ForkMessages = forkMsgs
		forkOpts = opts
	}

	if in.RunInBackground || selected.Background || toolpool.CoordinatorMergeFilterActive() || commands.ProactiveModeActive() {
		outFile, err := writeBackgroundOutput(cfg.TasksDir, s.ID, "Agent started")
		if err != nil {
			return "", true, err
		}
		writeBackgroundStatus(cfg.TasksDir, s.ID, "running", "Agent started in background", true)
		go func() {
			defer func() {
				if r := recover(); r != nil {
					msg := fmt.Sprintf("Agent failed in background: %v", r)
					_, _ = writeBackgroundOutput(cfg.TasksDir, s.ID, msg)
					writeBackgroundStatus(cfg.TasksDir, s.ID, "failed", msg, false)
				}
			}()
			if isTaskStopRequested(cfg.TasksDir, s.ID) {
				msg := "Agent stopped before execution"
				_, _ = writeBackgroundOutput(cfg.TasksDir, s.ID, msg)
				writeBackgroundStatus(cfg.TasksDir, s.ID, "stopped", msg, false)
				return
			}
			var output string
			if forkOpts != nil {
				output = executeAgentWithOpts(context.Background(), cfg, s, in.Prompt, nil, *forkOpts)
			} else {
				output = executeAgent(context.Background(), cfg, s, in.Prompt, nil)
			}
			_, _ = writeBackgroundOutput(cfg.TasksDir, s.ID, output)
			persistAgentMetadata(cfg, s)
			if isTaskStopRequested(cfg.TasksDir, s.ID) {
				writeBackgroundStatus(cfg.TasksDir, s.ID, "stopped", "Agent stopped", false)
				return
			}
			writeBackgroundStatus(cfg.TasksDir, s.ID, "completed", "Agent completed in background", true)
		}()
		resp, _ := json.Marshal(AgentToolResponse{
			Data: AgentToolResponseData{
				Success:          true,
				AgentID:          s.ID,
				Name:             s.Name,
				AgentType:        s.AgentType,
				Message:          "Agent started in background",
				OutputFile:       outFile,
				IsBackground:     true,
				WorktreePath:     s.WorktreePath,
				ProgressMessages: s.ProgressMessages,
			},
		})
		return string(resp), false, nil
	}

	var output string
	if forkOpts != nil {
		output = executeAgentWithOpts(context.Background(), cfg, s, in.Prompt, nil, *forkOpts)
	} else {
		output = executeAgent(context.Background(), cfg, s, in.Prompt, nil)
	}
	resp, _ := json.Marshal(AgentToolResponse{
		Data: AgentToolResponseData{
			Success:      true,
			AgentID:      s.ID,
			Name:         s.Name,
			AgentType:    s.AgentType,
			Message:      "Agent completed",
			Output:           output,
			WorktreePath:     s.WorktreePath,
			ProgressMessages: s.ProgressMessages,
		},
	})
	return string(resp), false, nil
}

func ResumeAgentTool(raw []byte, cfg AgentRuntimeConfig) (string, bool, error) {
	var in AgentToolInput
	if err := json.Unmarshal(raw, &in); err != nil {
		return "", true, err
	}
	target := strings.TrimSpace(in.Resume)
	if target == "" {
		return "", true, fmt.Errorf("resume is required")
	}
	agentSessionsMu.RLock()
	id := target
	if mapped, ok := agentByName[target]; ok {
		id = mapped
	}
	s, ok := agentSessions[id]
	agentSessionsMu.RUnlock()
	if !ok {
		if persisted := loadAgentMetadata(cfg, id); persisted != nil {
			s = persisted
			ok = true
			agentSessionsMu.Lock()
			agentSessions[s.ID] = s
			if strings.TrimSpace(s.Name) != "" {
				agentByName[s.Name] = s.ID
			}
			agentSessionsMu.Unlock()
		}
		if !ok {
			if persisted := loadAgentMetadataByName(cfg, target); persisted != nil {
				s = persisted
				ok = true
				agentSessionsMu.Lock()
				agentSessions[s.ID] = s
				if strings.TrimSpace(s.Name) != "" {
					agentByName[s.Name] = s.ID
				}
				agentSessionsMu.Unlock()
			}
		}
	}
	if !ok {
		resp, _ := json.Marshal(AgentToolResponse{
			Data: AgentToolResponseData{Success: false, Message: "Agent not found", AgentID: target},
		})
		return string(resp), false, nil
	}
	history := loadSidechainMessages(cfg, s.ID)
	if strings.TrimSpace(in.Prompt) == "" {
		if len(history) > 0 {
			var sb strings.Builder
			sb.WriteString("You are resuming an agent session with previously recorded context.\n")
			sb.WriteString(fmt.Sprintf("Previous session recorded %d messages.\n", len(history)))
			if s.Summary != "" {
				sb.WriteString(fmt.Sprintf("Last known state: %s\n", s.Summary))
			}
			if s.LastOutput != "" {
				sb.WriteString(fmt.Sprintf("Last output: %s\n", s.LastOutput))
			}
			sb.WriteString(fmt.Sprintf("Current time: %s\n", time.Now().UTC().Format(time.RFC3339)))
			sb.WriteString("Continue your work where you left off. Review the conversation history and pick up the task.")
			in.Prompt = sb.String()
		} else {
			in.Prompt = "resume"
		}
	}
	var output string
	if len(s.ForkSystemPrompt) > 0 {
		output = executeAgentWithOpts(context.Background(), cfg, s, in.Prompt, history, executeAgentOpts{
			ForkSystemPrompt: s.ForkSystemPrompt,
		})
	} else {
		output = executeAgent(context.Background(), cfg, s, in.Prompt, history)
	}
	persistAgentMetadata(cfg, s)
	resp, _ := json.Marshal(AgentToolResponse{
		Data: AgentToolResponseData{
			Success:      true,
			AgentID:      s.ID,
			Name:         s.Name,
			AgentType:    s.AgentType,
			Message:          "Agent resumed",
			Output:           output,
			WorktreePath:     s.WorktreePath,
			ProgressMessages: s.ProgressMessages,
		},
	})
	return string(resp), false, nil
}

// isSubscribedTo checks if a team member's subscriptions include the given sender identity.
func isSubscribedTo(member TeamFileMember, senderIdentity string) bool {
	if len(member.Subscriptions) == 0 {
		return true // no subscriptions means subscribe to all
	}
	for _, sub := range member.Subscriptions {
		if sub == senderIdentity {
			return true
		}
	}
	return false
}

func RunSendMessageTool(raw []byte, cfg AgentRuntimeConfig) (string, bool, error) {
	var in SendMessageInput
	if err := json.Unmarshal(raw, &in); err != nil {
		return "", true, err
	}
	if strings.TrimSpace(in.To) == "" {
		return "", true, fmt.Errorf("to is required")
	}

	// Resolve team context
	teamName := strings.TrimSpace(getenv("CLAUDE_CODE_TEAM_NAME"))
	senderName := strings.TrimSpace(getenv("CLAUDE_CODE_AGENT_NAME"))
	senderID := strings.TrimSpace(getenv("CLAUDE_CODE_AGENT_ID"))
	if senderName == "" {
		senderName = senderID
	}
	if senderName == "" {
		senderName = "unknown"
	}

	to := strings.TrimSpace(in.To)

	// Broadcast to all team members (*)
	if to == "*" {
		if teamName == "" {
			resp, _ := json.Marshal(map[string]any{
				"data": map[string]any{"success": false, "message": "Broadcast requires team context (CLAUDE_CODE_TEAM_NAME)"},
			})
			return string(resp), false, nil
		}
		tf, err := readTeamFile(teamName)
		if err != nil || tf == nil {
			resp, _ := json.Marshal(map[string]any{
				"data": map[string]any{"success": false, "message": "Team not found for broadcast", "team": teamName},
			})
			return string(resp), false, nil
		}
		broadcastCount := 0
		for _, m := range tf.Members {
			if m.AgentID == senderID || m.Name == senderName {
				continue // skip sender
			}
			targetName := m.Name
			if targetName == "" {
				targetName = m.AgentID
			}
			// Only deliver if target subscribes to sender
			if !isSubscribedTo(m, senderID) && !isSubscribedTo(m, senderName) {
				continue
			}
			if err := writeToMailbox(targetName, teamName, senderName, in.Message); err == nil {
				broadcastCount++
			}
		}
		resp, _ := json.Marshal(map[string]any{
			"data": map[string]any{
				"success": true,
				"message": fmt.Sprintf("Broadcast delivered to %d team members", broadcastCount),
				"count":   broadcastCount,
			},
		})
		return string(resp), false, nil
	}

	// Single-target delivery: write to mailbox with subscription check
	if teamName != "" {
		shouldDeliver := true
		if tf, err := readTeamFile(teamName); err == nil && tf != nil {
			for _, m := range tf.Members {
				if m.Name == to || m.AgentID == to {
					if !isSubscribedTo(m, senderID) && !isSubscribedTo(m, senderName) {
						shouldDeliver = false
					}
					break
				}
			}
		}
		if shouldDeliver {
			_ = writeToMailbox(to, teamName, senderName, in.Message)
		}
	}

	// Look up and run the target agent session
	agentSessionsMu.RLock()
	id := to
	if mapped, ok := agentByName[id]; ok {
		id = mapped
	}
	s, ok := agentSessions[id]
	agentSessionsMu.RUnlock()
	if !ok {
		if persisted := loadAgentMetadata(cfg, id); persisted != nil {
			s = persisted
			ok = true
			agentSessionsMu.Lock()
			agentSessions[s.ID] = s
			if strings.TrimSpace(s.Name) != "" {
				agentByName[s.Name] = s.ID
			}
			agentSessionsMu.Unlock()
		}
		if !ok {
			if persisted := loadAgentMetadataByName(cfg, to); persisted != nil {
				s = persisted
				ok = true
				agentSessionsMu.Lock()
				agentSessions[s.ID] = s
				if strings.TrimSpace(s.Name) != "" {
					agentByName[s.Name] = s.ID
				}
				agentSessionsMu.Unlock()
			}
		}
	}
	if !ok {
		resp, _ := json.Marshal(map[string]any{
			"data": map[string]any{"success": false, "message": "SendMessage target not found", "to": to},
		})
		return string(resp), false, nil
	}
	history := loadSidechainMessages(cfg, s.ID)
	output := executeAgent(context.Background(), cfg, s, in.Message, history)
	persistAgentMetadata(cfg, s)

	// Mark mailbox messages as read after processing
	if teamName != "" && s.Name != "" {
		_ = markMessagesAsRead(s.Name, teamName)
	}

	resp, _ := json.Marshal(map[string]any{
		"data": map[string]any{
			"success":  true,
			"agent_id": s.ID,
			"message":  "Message delivered",
			"output":   output,
		},
	})
	return string(resp), false, nil
}
