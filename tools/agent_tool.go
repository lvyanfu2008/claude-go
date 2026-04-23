package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

func getenv(k string) string { return os.Getenv(k) }

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
	defs := LoadAgentDefinitionsForCwd(cfg.ProjectRoot)
	selected := ResolveAgentDefinition(defs, in.SubagentType)
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
	s := &AgentSession{
		ID:                                 agentID,
		Name:                               name,
		AgentType:                          selected.AgentType,
		Description:                        strings.TrimSpace(in.Description),
		Model:                              model,
		PermissionMode:                     selected.PermissionMode,
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
		OmitClaudeMd:                       selected.OmitClaudeMd,
		CriticalSystemReminderExperimental: selected.CriticalSystemReminderExperimental,
		CreatedAt:                          time.Now().UTC(),
		UpdatedAt:                          time.Now().UTC(),
	}
	if s.Description == "" {
		s.Description = selected.WhenToUse
	}
	if s.Isolation == "" {
		s.Isolation = strings.TrimSpace(selected.Isolation)
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

	agentSessionsMu.Lock()
	agentSessions[s.ID] = s
	agentByName[s.Name] = s.ID
	agentSessionsMu.Unlock()
	persistAgentMetadata(cfg, s)

	if in.RunInBackground || selected.Background {
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
			output := executeAgent(context.Background(), cfg, s, in.Prompt, nil)
			_, _ = writeBackgroundOutput(cfg.TasksDir, s.ID, output)
			persistSidechain(cfg, s, in.Prompt, output)
			persistAgentMetadata(cfg, s)
			if isTaskStopRequested(cfg.TasksDir, s.ID) {
				writeBackgroundStatus(cfg.TasksDir, s.ID, "stopped", "Agent stopped", false)
				return
			}
			writeBackgroundStatus(cfg.TasksDir, s.ID, "completed", "Agent completed in background", true)
		}()
		resp, _ := json.Marshal(AgentToolResponse{
			Data: AgentToolResponseData{
				Success:      true,
				AgentID:      s.ID,
				Name:         s.Name,
				AgentType:    s.AgentType,
				Message:      "Agent started in background",
				OutputFile:   outFile,
				IsBackground: true,
				WorktreePath: s.WorktreePath,
			},
		})
		return string(resp), false, nil
	}

	output := executeAgent(context.Background(), cfg, s, in.Prompt, nil)
	persistSidechain(cfg, s, in.Prompt, output)
	resp, _ := json.Marshal(AgentToolResponse{
		Data: AgentToolResponseData{
			Success:      true,
			AgentID:      s.ID,
			Name:         s.Name,
			AgentType:    s.AgentType,
			Message:      "Agent completed",
			Output:       output,
			WorktreePath: s.WorktreePath,
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
			in.Prompt = fmt.Sprintf("resume from transcript with %d messages", len(history))
		} else {
			in.Prompt = "resume"
		}
	}
	output := executeAgent(context.Background(), cfg, s, in.Prompt, history)
	persistSidechain(cfg, s, in.Prompt, output)
	persistAgentMetadata(cfg, s)
	resp, _ := json.Marshal(AgentToolResponse{
		Data: AgentToolResponseData{
			Success:      true,
			AgentID:      s.ID,
			Name:         s.Name,
			AgentType:    s.AgentType,
			Message:      "Agent resumed",
			Output:       output,
			WorktreePath: s.WorktreePath,
		},
	})
	return string(resp), false, nil
}

func RunSendMessageTool(raw []byte, cfg AgentRuntimeConfig) (string, bool, error) {
	var in SendMessageInput
	if err := json.Unmarshal(raw, &in); err != nil {
		return "", true, err
	}
	if strings.TrimSpace(in.To) == "" {
		return "", true, fmt.Errorf("to is required")
	}
	agentSessionsMu.RLock()
	id := strings.TrimSpace(in.To)
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
			if persisted := loadAgentMetadataByName(cfg, strings.TrimSpace(in.To)); persisted != nil {
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
			"data": map[string]any{"success": false, "message": "SendMessage target not found", "to": in.To},
		})
		return string(resp), false, nil
	}
	history := loadSidechainMessages(cfg, s.ID)
	output := executeAgent(context.Background(), cfg, s, in.Message, history)
	persistSidechain(cfg, s, in.Message, output)
	persistAgentMetadata(cfg, s)
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
