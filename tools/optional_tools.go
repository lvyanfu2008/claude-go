package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// TestingPermissionFromJSON matches TS TestingPermissionTool.call output shape.
func TestingPermissionFromJSON(raw []byte) (string, bool, error) {
	_ = raw
	out := map[string]any{"data": "TestingPermission executed successfully"}
	b, _ := json.Marshal(out)
	return string(b), false, nil
}

// SleepFromJSON waits up to duration_seconds (default 1, max 60). Kairos SleepTool is not in-tree; schema matches typical duration payloads.
func SleepFromJSON(ctx context.Context, raw []byte) (string, bool, error) {
	var in struct {
		DurationSeconds float64 `json:"duration_seconds"`
		Seconds         float64 `json:"seconds"`
	}
	_ = json.Unmarshal(raw, &in)
	sec := in.DurationSeconds
	if sec <= 0 {
		sec = in.Seconds
	}
	if sec <= 0 {
		sec = 1
	}
	d := time.Duration(sec * float64(time.Second))
	start := time.Now()
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		elapsed := time.Since(start).Seconds()
		out := map[string]any{
			"data": map[string]any{
				"slept_seconds": elapsed,
				"interrupted":   true,
			},
		}
		b, _ := json.Marshal(out)
		return string(b), false, nil
	case <-timer.C:
	}
	out := map[string]any{
		"data": map[string]any{
			"slept_seconds": sec,
			"interrupted":   false,
		},
	}
	b, _ := json.Marshal(out)
	return string(b), false, nil
}

// ListPeersFromJSON mirrors TS shape { data: { peers: [...] } }.
func ListPeersFromJSON(raw []byte, cfg Config) (string, bool, error) {
	var in struct {
		IncludeSelf bool `json:"include_self"`
	}
	_ = json.Unmarshal(raw, &in)
	peers := make([]map[string]any, 0)
	if in.IncludeSelf {
		addr := strings.TrimSpace(os.Getenv("CLAUDE_CODE_MESSAGING_SOCKET_PATH"))
		if addr == "" {
			addr = strings.TrimSpace(os.Getenv("CLAUDE_CODE_MESSAGING_SOCKET"))
		}
		if addr != "" {
			peers = append(peers, map[string]any{
				"address": "uds:" + addr,
				"name":    "self",
				"cwd":     cfg.WorkDir,
				"pid":     os.Getpid(),
			})
		}
	}
	out := map[string]any{"data": map[string]any{"peers": peers}}
	b, _ := json.Marshal(out)
	return string(b), false, nil
}

// VerifyPlanExecutionFromJSON matches disabled VerifyPlanExecutionTool.js in this tree.
func VerifyPlanExecutionFromJSON(raw []byte) (string, bool, error) {
	var in struct {
		PlanSummary      string `json:"plan_summary"`
		AllStepsComplete bool   `json:"all_steps_completed"`
	}
	if err := json.Unmarshal(raw, &in); err != nil {
		return "", true, err
	}
	out := map[string]any{
		"data": map[string]any{
			"verified": in.AllStepsComplete,
			"summary":  in.PlanSummary,
		},
	}
	b, _ := json.Marshal(out)
	return string(b), false, nil
}

// OverflowTestFromJSON feature tool not wired in Go runner.
func OverflowTestFromJSON(raw []byte) (string, bool, error) {
	_ = raw
	out := map[string]any{
		"data": map[string]any{
			"ok":      false,
			"message": "OverflowTest runtime is not available in this build.",
		},
	}
	b, _ := json.Marshal(out)
	return string(b), false, nil
}

// CtxInspectFromJSON mirrors TS fallback when CONTEXT_COLLAPSE runtime is absent.
func CtxInspectFromJSON(raw []byte) (string, bool, error) {
	_ = raw
	out := map[string]any{
		"data": map[string]any{
			"total_tokens":  0,
			"message_count": 0,
			"summary":       "Context inspection requires the CONTEXT_COLLAPSE runtime.",
		},
	}
	b, _ := json.Marshal(out)
	return string(b), false, nil
}

// TerminalCaptureFromJSON feature tool not wired in Go runner.
func TerminalCaptureFromJSON(raw []byte) (string, bool, error) {
	var in struct {
		Lines   int    `json:"lines"`
		PanelID string `json:"panel_id"`
	}
	_ = json.Unmarshal(raw, &in)
	lines := in.Lines
	if lines <= 0 {
		lines = 50
	}
	terminalsDir := strings.TrimSpace(os.Getenv("CLAUDE_CODE_TERMINALS_DIR"))
	content := ""
	if terminalsDir != "" {
		target := ""
		if strings.TrimSpace(in.PanelID) != "" {
			target = filepath.Join(terminalsDir, strings.TrimSpace(in.PanelID)+".txt")
		} else {
			entries, err := os.ReadDir(terminalsDir)
			if err == nil {
				for _, e := range entries {
					if !e.IsDir() && strings.HasSuffix(e.Name(), ".txt") {
						target = filepath.Join(terminalsDir, e.Name())
						break
					}
				}
			}
		}
		if target != "" {
			if b, err := os.ReadFile(target); err == nil {
				all := strings.Split(string(b), "\n")
				if len(all) > lines {
					all = all[len(all)-lines:]
				}
				content = strings.Join(all, "\n")
			}
		}
	}
	out := map[string]any{
		"data": map[string]any{
			"content":    content,
			"line_count": lines,
		},
	}
	b, _ := json.Marshal(out)
	return string(b), false, nil
}

// LSPFromJSON is not available without an LSP client in Go.
func LSPFromJSON(raw []byte) (string, bool, error) {
	var in struct {
		Operation string `json:"operation"`
		FilePath  string `json:"filePath"`
		Line      int    `json:"line"`
		Character int    `json:"character"`
	}
	_ = json.Unmarshal(raw, &in)
	op := strings.TrimSpace(in.Operation)
	if op == "" {
		op = "hover"
	}
	filePath := strings.TrimSpace(in.FilePath)
	if filePath != "" && !filepath.IsAbs(filePath) {
		if cwd, err := os.Getwd(); err == nil {
			filePath = filepath.Join(cwd, filePath)
		}
	}
	result := "No LSP server available for file type: "
	if filePath != "" {
		if b, err := os.ReadFile(filePath); err == nil {
			result = lspOperationResult(op, string(b), in.Line, in.Character)
		}
	}
	out := map[string]any{
		"data": map[string]any{
			"operation": op,
			"result":    result,
			"filePath":  in.FilePath,
		},
	}
	b, _ := json.Marshal(out)
	return string(b), false, nil
}

func lspOperationResult(operation, content string, line, character int) string {
	lines := strings.Split(content, "\n")
	switch operation {
	case "hover":
		if line <= 0 || line > len(lines) {
			return "No symbol at position"
		}
		l := lines[line-1]
		sym := wordAt(l, character)
		if sym == "" {
			return "No symbol at position"
		}
		return "Hover for `" + sym + "`"
	case "documentSymbol", "workspaceSymbol":
		symbols := collectGoLikeSymbols(lines)
		if len(symbols) == 0 {
			return "No symbols found."
		}
		return strings.Join(symbols, "\n")
	case "goToDefinition", "goToImplementation", "prepareCallHierarchy":
		if line <= 0 || line > len(lines) {
			return "No definition found."
		}
		sym := wordAt(lines[line-1], character)
		if sym == "" {
			return "No definition found."
		}
		for i, l := range lines {
			if strings.Contains(l, "func "+sym+"(") || strings.Contains(l, "type "+sym+" ") || strings.Contains(l, "var "+sym+" ") || strings.Contains(l, "const "+sym+" ") {
				return fmt.Sprintf("%s:%d: definition of %s", "file", i+1, sym)
			}
		}
		return "No definition found."
	case "findReferences", "incomingCalls", "outgoingCalls":
		if line <= 0 || line > len(lines) {
			return "No references found."
		}
		sym := wordAt(lines[line-1], character)
		if sym == "" {
			return "No references found."
		}
		count := 0
		for _, l := range lines {
			count += strings.Count(l, sym)
		}
		if count == 0 {
			return "No references found."
		}
		return fmt.Sprintf("Found %d reference(s) for %s", count, sym)
	default:
		return "Unsupported operation: " + operation
	}
}

func wordAt(line string, character int) string {
	if line == "" {
		return ""
	}
	idx := character - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(line) {
		idx = len(line) - 1
	}
	isWord := func(b byte) bool {
		return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '_'
	}
	if !isWord(line[idx]) {
		return ""
	}
	start, end := idx, idx
	for start > 0 && isWord(line[start-1]) {
		start--
	}
	for end+1 < len(line) && isWord(line[end+1]) {
		end++
	}
	return line[start : end+1]
}

func collectGoLikeSymbols(lines []string) []string {
	re := regexp.MustCompile(`^\s*(func|type|var|const)\s+([A-Za-z_][A-Za-z0-9_]*)`)
	out := make([]string, 0)
	for i, l := range lines {
		m := re.FindStringSubmatch(l)
		if len(m) == 3 {
			out = append(out, fmt.Sprintf("L%d: %s %s", i+1, m[1], m[2]))
		}
	}
	return out
}

// EnterWorktreeFromJSON — git worktree / session integration is TS-only here.
func EnterWorktreeFromJSON(raw []byte, cfg Config) (string, bool, error) {
	var in struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(raw, &in); err != nil {
		return "", true, err
	}
	root := strings.TrimSpace(cfg.ProjectRoot)
	if root == "" {
		root = strings.TrimSpace(cfg.WorkDir)
	}
	if root == "" {
		return "", true, fmt.Errorf("project root is required")
	}
	wtPath, err := createWorktree(root, in.Name)
	if err != nil {
		return "", true, err
	}
	branch := "agent/" + sanitizeName(strings.TrimSpace(in.Name))
	out := map[string]any{
		"data": map[string]any{
			"worktreePath":   wtPath,
			"worktreeBranch": branch,
			"message":        "Created worktree at " + wtPath + " on branch " + branch + ". The session is now working in the worktree. Use ExitWorktree to leave mid-session, or exit the session to be prompted.",
		},
	}
	b, _ := json.Marshal(out)
	return string(b), false, nil
}

// ExitWorktreeFromJSON — git worktree / session integration is TS-only here.
func ExitWorktreeFromJSON(raw []byte, cfg Config) (string, bool, error) {
	var in struct {
		Action string `json:"action"`
	}
	if err := json.Unmarshal(raw, &in); err != nil {
		return "", true, err
	}
	action := strings.TrimSpace(in.Action)
	if action == "" {
		action = "keep"
	}
	root := strings.TrimSpace(cfg.ProjectRoot)
	if root == "" {
		root = strings.TrimSpace(cfg.WorkDir)
	}
	if action == "remove" && strings.TrimSpace(cfg.WorkDir) != "" {
		_ = removeWorktree(root, cfg.WorkDir)
	}
	out := map[string]any{
		"data": map[string]any{
			"action":       action,
			"originalCwd":  root,
			"worktreePath": cfg.WorkDir,
			"message":      "Exited worktree session.",
		},
	}
	b, _ := json.Marshal(out)
	return string(b), false, nil
}

// TeamCreateFromJSON creates a new agent team with proper roster file (TS TeamCreateTool).
func TeamCreateFromJSON(raw []byte, cfg Config) (string, bool, error) {
	var in struct {
		TeamName    string `json:"team_name"`
		Description string `json:"description,omitempty"`
	}
	if err := json.Unmarshal(raw, &in); err != nil {
		return "", true, err
	}
	team := strings.TrimSpace(in.TeamName)
	if team == "" {
		return "", true, fmt.Errorf("team_name is required for TeamCreate")
	}

	// Check if team already exists
	existing, err := readTeamFile(team)
	if err != nil {
		return "", true, fmt.Errorf("read team: %w", err)
	}
	if existing != nil {
		out := map[string]any{
			"data": map[string]any{
				"team_name":      team,
				"team_file_path": getTeamFilePath(team),
				"lead_agent_id":  existing.LeadAgentID,
				"message":        "Team already exists",
				"member_count":   len(existing.Members),
			},
		}
		b, _ := json.Marshal(out)
		return string(b), false, nil
	}

	leadID := "team-lead@" + team
	sessionID := strings.TrimSpace(cfg.SessionID)
	tf := &TeamFile{
		Name:          team,
		Description:   strings.TrimSpace(in.Description),
		CreatedAt:     time.Now().UnixMilli(),
		LeadAgentID:   leadID,
		LeadSessionID: sessionID,
		Members: []TeamFileMember{
			{
				AgentID:    leadID,
				Name:       "lead",
				AgentType:  "general-purpose",
				JoinedAt:   time.Now().UnixMilli(),
				Subscriptions: []string{},
				IsActive:   true,
				SessionID:  sessionID,
			},
		},
	}

	var memberSubs []string
	for _, m := range tf.Members {
		memberSubs = append(memberSubs, m.AgentID)
	}
	tf.Members[0].Subscriptions = memberSubs

	teamPath := getTeamFilePath(team)
	if err := writeTeamFile(team, tf); err != nil {
		return "", true, fmt.Errorf("write team file: %w", err)
	}

	// Ensure inbox directory exists
	_ = ensureInboxDir(team)

	out := map[string]any{
		"data": map[string]any{
			"team_name":      team,
			"team_file_path": teamPath,
			"lead_agent_id":  leadID,
			"message":        "Team created",
			"member_count":   len(tf.Members),
		},
	}
	b, _ := json.Marshal(out)
	return string(b), false, nil
}

// TeamDeleteFromJSON deletes an agent team and its inboxes (TS TeamDeleteTool).
func TeamDeleteFromJSON(raw []byte, cfg Config) (string, bool, error) {
	var in struct {
		TeamName string `json:"team_name"`
	}
	_ = json.Unmarshal(raw, &in)
	team := strings.TrimSpace(in.TeamName)
	if team == "" {
		team = strings.TrimSpace(getenv("CLAUDE_CODE_TEAM_NAME"))
	}

	if team == "" {
		out := map[string]any{
			"data": map[string]any{
				"success":   true,
				"message":   "No team name found, nothing to clean up",
				"team_name": nil,
			},
		}
		b, _ := json.Marshal(out)
		return string(b), false, nil
	}

	tf, err := readTeamFile(team)
	if err != nil || tf == nil {
		// Try old path for backwards compatibility
		root := strings.TrimSpace(cfg.ProjectRoot)
		if root == "" {
			root = strings.TrimSpace(cfg.WorkDir)
		}
		oldDir := filepath.Join(root, ".claude", ".gou-team")
		_ = os.RemoveAll(oldDir)

		out := map[string]any{
			"data": map[string]any{
				"success":   true,
				"message":   "Team cleaned up",
				"team_name": team,
			},
		}
		b, _ := json.Marshal(out)
		return string(b), false, nil
	}

	// Remove the entire team directory (roster + inboxes)
	teamDir := getTeamDir(team)
	if err := os.RemoveAll(teamDir); err != nil {
		return "", true, fmt.Errorf("remove team dir: %w", err)
	}

	out := map[string]any{
		"data": map[string]any{
			"success":      true,
			"message":      "Team deleted",
			"team_name":    team,
			"member_count": len(tf.Members),
		},
	}
	b, _ := json.Marshal(out)
	return string(b), false, nil
}

// TeamAddMemberFromJSON adds a member to an existing team roster.
func TeamAddMemberFromJSON(raw []byte, cfg Config) (string, bool, error) {
	var in struct {
		TeamName string `json:"team_name"`
		AgentID  string `json:"agent_id"`
		Name     string `json:"name"`
	}
	if err := json.Unmarshal(raw, &in); err != nil {
		return "", true, err
	}
	team := strings.TrimSpace(in.TeamName)
	if team == "" {
		return "", true, fmt.Errorf("team_name is required")
	}
	agentID := strings.TrimSpace(in.AgentID)
	if agentID == "" {
		return "", true, fmt.Errorf("agent_id is required")
	}
	member := TeamFileMember{
		AgentID:       agentID,
		Name:          strings.TrimSpace(in.Name),
		JoinedAt:      time.Now().UnixMilli(),
		Subscriptions: []string{},
		IsActive:      true,
		SessionID:     strings.TrimSpace(cfg.SessionID),
	}
	if err := addTeamMember(team, member); err != nil {
		return "", true, fmt.Errorf("add team member: %w", err)
	}
	out := map[string]any{
		"data": map[string]any{
			"success":   true,
			"team_name": team,
			"agent_id":  agentID,
			"message":   "Member added to team",
		},
	}
	b, _ := json.Marshal(out)
	return string(b), false, nil
}

// TeamRemoveMemberFromJSON removes a member from a team roster.
func TeamRemoveMemberFromJSON(raw []byte, cfg Config) (string, bool, error) {
	var in struct {
		TeamName string `json:"team_name"`
		AgentID  string `json:"agent_id"`
	}
	if err := json.Unmarshal(raw, &in); err != nil {
		return "", true, err
	}
	team := strings.TrimSpace(in.TeamName)
	if team == "" {
		return "", true, fmt.Errorf("team_name is required")
	}
	agentID := strings.TrimSpace(in.AgentID)
	if agentID == "" {
		return "", true, fmt.Errorf("agent_id is required")
	}
	if err := removeTeamMember(team, agentID); err != nil {
		return "", true, fmt.Errorf("remove team member: %w", err)
	}
	out := map[string]any{
		"data": map[string]any{
			"success":   true,
			"team_name": team,
			"agent_id":  agentID,
			"message":   "Member removed from team",
		},
	}
	b, _ := json.Marshal(out)
	return string(b), false, nil
}

// ConfigFromJSON — ant-only tool.
func ConfigFromJSON(raw []byte, cfg Config) (string, bool, error) {
	var in struct {
		Setting string `json:"setting"`
		Value   any    `json:"value"`
	}
	if err := json.Unmarshal(raw, &in); err != nil {
		return "", true, err
	}
	if strings.TrimSpace(in.Setting) == "" {
		return "", true, fmt.Errorf("setting is required")
	}
	root := strings.TrimSpace(cfg.ProjectRoot)
	if root == "" {
		root = strings.TrimSpace(cfg.WorkDir)
	}
	if root == "" {
		return "", true, fmt.Errorf("project root is required")
	}
	cfgPath := filepath.Join(root, ".claude", ".gou-config.json")
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0o700); err != nil {
		return "", true, err
	}
	cur := map[string]any{}
	if b, err := os.ReadFile(cfgPath); err == nil {
		_ = json.Unmarshal(b, &cur)
	}
	toolCfg, _ := cur["settings"].(map[string]any)
	if toolCfg == nil {
		toolCfg = map[string]any{}
	}
	prev := toolCfg[in.Setting]
	if in.Value == nil {
		out := map[string]any{
			"data": map[string]any{
				"success":   true,
				"operation": "get",
				"setting":   in.Setting,
				"value":     prev,
			},
		}
		b, _ := json.Marshal(out)
		return string(b), false, nil
	}
	toolCfg[in.Setting] = in.Value
	cur["settings"] = toolCfg
	if b, err := json.MarshalIndent(cur, "", "  "); err == nil {
		_ = os.WriteFile(cfgPath, append(b, '\n'), 0o600)
	}
	out := map[string]any{
		"data": map[string]any{
			"success":       true,
			"operation":     "set",
			"setting":       in.Setting,
			"previousValue": prev,
			"newValue":      in.Value,
		},
	}
	b, _ := json.Marshal(out)
	return string(b), false, nil
}

// TungstenFromJSON — ant-only tool.
func TungstenFromJSON(raw []byte) (string, bool, error) {
	_ = raw
	out := map[string]any{
		"data": map[string]any{
			"success": false,
			"error":   "Tungsten runtime is not available in this build.",
		},
	}
	b, _ := json.Marshal(out)
	return string(b), false, nil
}

// SuggestBackgroundPRFromJSON — ant feature tool.
func SuggestBackgroundPRFromJSON(raw []byte) (string, bool, error) {
	out := map[string]any{
		"data": map[string]any{
			"suggested":     false,
			"suggestion_id": "",
			"error":         "SuggestBackgroundPR requires the KAIROS runtime.",
		},
	}
	b, _ := json.Marshal(out)
	return string(b), false, nil
}

// WebBrowserFromJSON — browser automation not in Go runner.
func WebBrowserFromJSON(raw []byte) (string, bool, error) {
	var in struct {
		URL string `json:"url"`
	}
	_ = json.Unmarshal(raw, &in)
	out := map[string]any{
		"data": map[string]any{
			"title":   "",
			"url":     in.URL,
			"content": "Web browser requires the WEB_BROWSER_TOOL runtime.",
		},
	}
	b, _ := json.Marshal(out)
	return string(b), false, nil
}

// RemoteTriggerFromJSON — remote triggers not in Go runner.
func RemoteTriggerFromJSON(raw []byte) (string, bool, error) {
	var in struct {
		Action    string         `json:"action"`
		TriggerID string         `json:"trigger_id"`
		Body      map[string]any `json:"body"`
	}
	if err := json.Unmarshal(raw, &in); err != nil {
		return "", true, err
	}
	action := strings.TrimSpace(in.Action)
	if action == "" {
		return "", true, fmt.Errorf("action is required")
	}
	token := oauthAccessToken()
	if token == "" {
		return "", true, fmt.Errorf("Not authenticated with a claude.ai account. Run /login and try again.")
	}
	org := organizationUUID()
	if org == "" {
		return "", true, fmt.Errorf("Unable to resolve organization UUID.")
	}
	base := triggerAPIBaseURL() + "/v1/code/triggers"
	method := http.MethodGet
	url := base
	var body any
	switch action {
	case "list":
		method = http.MethodGet
		url = base
	case "get":
		if strings.TrimSpace(in.TriggerID) == "" {
			return "", true, fmt.Errorf("get requires trigger_id")
		}
		url = base + "/" + strings.TrimSpace(in.TriggerID)
	case "create":
		if in.Body == nil {
			return "", true, fmt.Errorf("create requires body")
		}
		method = http.MethodPost
		body = in.Body
	case "update":
		if strings.TrimSpace(in.TriggerID) == "" {
			return "", true, fmt.Errorf("update requires trigger_id")
		}
		if in.Body == nil {
			return "", true, fmt.Errorf("update requires body")
		}
		method = http.MethodPost
		url = base + "/" + strings.TrimSpace(in.TriggerID)
		body = in.Body
	case "run":
		if strings.TrimSpace(in.TriggerID) == "" {
			return "", true, fmt.Errorf("run requires trigger_id")
		}
		method = http.MethodPost
		url = base + "/" + strings.TrimSpace(in.TriggerID) + "/run"
		body = map[string]any{}
	default:
		return "", true, fmt.Errorf("unsupported action %q", action)
	}
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return "", true, err
		}
		reqBody = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return "", true, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("anthropic-beta", "ccr-triggers-2026-01-30")
	req.Header.Set("x-organization-uuid", org)
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", true, err
	}
	defer resp.Body.Close()
	rb, _ := io.ReadAll(resp.Body)
	out := map[string]any{
		"data": map[string]any{
			"status": resp.StatusCode,
			"json":   string(rb),
		},
	}
	b, _ := json.Marshal(out)
	return string(b), false, nil
}

func triggerAPIBaseURL() string {
	if v := strings.TrimSpace(os.Getenv("ANTHROPIC_API_BASE_URL")); v != "" {
		return strings.TrimSuffix(v, "/")
	}
	return "https://api.anthropic.com"
}

func oauthAccessToken() string {
	if t := strings.TrimSpace(os.Getenv("CLAUDE_CODE_OAUTH_TOKEN")); t != "" {
		return t
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	raw, err := os.ReadFile(filepath.Join(home, ".claude", ".credentials.json"))
	if err != nil {
		return ""
	}
	var root map[string]json.RawMessage
	if json.Unmarshal(raw, &root) != nil {
		return ""
	}
	var oauth struct {
		AccessToken string `json:"accessToken"`
	}
	if v, ok := root["claudeAiOauth"]; ok {
		_ = json.Unmarshal(v, &oauth)
	}
	return strings.TrimSpace(oauth.AccessToken)
}

func organizationUUID() string {
	if v := strings.TrimSpace(os.Getenv("CLAUDE_CODE_ORGANIZATION_UUID")); v != "" {
		return v
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	raw, err := os.ReadFile(filepath.Join(home, ".claude", "config.json"))
	if err != nil {
		return ""
	}
	var cfg struct {
		OAuthAccount *struct {
			OrganizationUUID string `json:"organizationUuid"`
		} `json:"oauthAccount"`
	}
	if json.Unmarshal(raw, &cfg) != nil || cfg.OAuthAccount == nil {
		return ""
	}
	return strings.TrimSpace(cfg.OAuthAccount.OrganizationUUID)
}

// MonitorFromJSON starts a background monitor and returns TS-compatible payload.
func MonitorFromJSON(ctx context.Context, raw []byte, cfg Config) (string, bool, error) {
	var in struct {
		Command     string `json:"command"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal(raw, &in); err != nil {
		return "", true, err
	}
	in.Command = strings.TrimSpace(in.Command)
	in.Description = strings.TrimSpace(in.Description)
	if in.Command == "" {
		return "", true, fmt.Errorf("command is required")
	}
	if in.Description == "" {
		return "", true, fmt.Errorf("description is required")
	}
	tasksDir := cfg.TasksDir()
	if err := ensureDir(tasksDir); err != nil {
		return "", true, err
	}
	taskID := fmt.Sprintf("monitor-%d", time.Now().UTC().UnixNano())
	outputFile := filepath.Join(tasksDir, taskID+".output")
	writeBackgroundStatus(tasksDir, taskID, "running", "Monitor started", true)
	go func() {
		f, err := os.OpenFile(outputFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
		if err != nil {
			writeBackgroundStatus(tasksDir, taskID, "failed", "Failed opening output file", false)
			return
		}
		defer f.Close()
		cmd := exec.Command("bash", "-lc", in.Command)
		if strings.TrimSpace(cfg.WorkDir) != "" {
			cmd.Dir = cfg.WorkDir
		}
		cmd.Stdout = f
		cmd.Stderr = f
		if err := cmd.Run(); err != nil {
			writeBackgroundStatus(tasksDir, taskID, "failed", "Monitor exited with error: "+err.Error(), false)
			return
		}
		writeBackgroundStatus(tasksDir, taskID, "completed", "Monitor completed", true)
	}()
	out := map[string]any{
		"data": map[string]any{
			"taskId":     taskID,
			"outputFile": outputFile,
		},
	}
	b, _ := json.Marshal(out)
	return string(b), false, nil
}

// WorkflowFromJSON mirrors TS fallback when workflow runtime is absent.
func WorkflowFromJSON(raw []byte) (string, bool, error) {
	var in struct {
		Workflow string `json:"workflow"`
		Args     string `json:"args"`
	}
	if err := json.Unmarshal(raw, &in); err != nil {
		return "", true, err
	}
	out := map[string]any{
		"data": map[string]any{
			"output": "Error: Workflow execution requires the WORKFLOW_SCRIPTS runtime.",
		},
	}
	b, _ := json.Marshal(out)
	return string(b), false, nil
}

// SnipFromJSON — feature import not in Go runner.
func SnipFromJSON(raw []byte) (string, bool, error) {
	var in struct {
		MessageIDs []string `json:"message_ids"`
		Reason     string   `json:"reason"`
	}
	if err := json.Unmarshal(raw, &in); err != nil {
		return "", true, err
	}
	summary := strings.TrimSpace(in.Reason)
	if summary == "" {
		summary = fmt.Sprintf("Snipped %d messages", len(in.MessageIDs))
	}
	out := map[string]any{
		"data": map[string]any{
			"snipped_count": len(in.MessageIDs),
			"summary":       summary,
		},
	}
	b, _ := json.Marshal(out)
	return string(b), false, nil
}

// SendUserFileFromJSON mirrors TS fallback payload for missing KAIROS transport.
func SendUserFileFromJSON(raw []byte) (string, bool, error) {
	var in struct {
		FilePath string `json:"file_path"`
	}
	if err := json.Unmarshal(raw, &in); err != nil {
		return "", true, err
	}
	out := map[string]any{
		"data": map[string]any{
			"sent":      false,
			"file_path": in.FilePath,
			"error":     "SendUserFile requires the KAIROS assistant transport layer.",
		},
	}
	b, _ := json.Marshal(out)
	return string(b), false, nil
}

// PushNotificationFromJSON mirrors TS fallback payload for missing KAIROS transport.
func PushNotificationFromJSON(raw []byte) (string, bool, error) {
	out := map[string]any{
		"data": map[string]any{
			"sent":  false,
			"error": "PushNotification requires the KAIROS transport layer.",
		},
	}
	b, _ := json.Marshal(out)
	return string(b), false, nil
}

// SubscribePRFromJSON mirrors TS fallback payload for missing webhook runtime.
func SubscribePRFromJSON(raw []byte) (string, bool, error) {
	out := map[string]any{
		"data": map[string]any{
			"subscribed":      false,
			"subscription_id": "",
			"error":           "SubscribePR requires the KAIROS GitHub webhook subsystem.",
		},
	}
	b, _ := json.Marshal(out)
	return string(b), false, nil
}
