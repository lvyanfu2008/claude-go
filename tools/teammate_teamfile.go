package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// TeamFileMember mirrors TS TeamFile.members[] entry.
type TeamFileMember struct {
	AgentID          string `json:"agentId"`
	Name             string `json:"name"`
	AgentType        string `json:"agentType,omitempty"`
	Model            string `json:"model,omitempty"`
	Prompt           string `json:"prompt,omitempty"`
	Color            string `json:"color,omitempty"`
	PlanModeRequired bool   `json:"planModeRequired,omitempty"`
	JoinedAt         int64  `json:"joinedAt"`
	TmuxPaneID       string `json:"tmuxPaneId"`
	CWD              string `json:"cwd"`
	WorktreePath     string `json:"worktreePath,omitempty"`
	SessionID        string `json:"sessionId,omitempty"`
	Subscriptions    []string `json:"subscriptions"`
	BackendType      string `json:"backendType,omitempty"`
	IsActive         bool   `json:"isActive,omitempty"`
}

// TeamFile mirrors TS TeamFile type.
type TeamFile struct {
	Name          string           `json:"name"`
	Description   string           `json:"description,omitempty"`
	CreatedAt     int64            `json:"createdAt"`
	LeadAgentID   string           `json:"leadAgentId"`
	LeadSessionID string           `json:"leadSessionId,omitempty"`
	HiddenPaneIDs []string         `json:"hiddenPaneIds,omitempty"`
	Members       []TeamFileMember `json:"members"`
}

// getTeamDir returns the team directory path.
func getTeamDir(teamName string) string {
	safe := sanitizePathComponent(teamName)
	return filepath.Join(getTeamsDir(), safe)
}

// getTeamFilePath returns the path to a team's JSON roster file.
func getTeamFilePath(teamName string) string {
	return filepath.Join(getTeamDir(teamName), "team.json")
}

// readTeamFile reads a team's JSON roster file.
func readTeamFile(teamName string) (*TeamFile, error) {
	p := getTeamFilePath(teamName)
	b, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read team file: %w", err)
	}
	var tf TeamFile
	if err := json.Unmarshal(b, &tf); err != nil {
		return nil, fmt.Errorf("parse team file: %w", err)
	}
	return &tf, nil
}

// writeTeamFile writes a team's JSON roster file, creating directories as needed.
func writeTeamFile(teamName string, tf *TeamFile) error {
	dir := getTeamDir(teamName)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create team dir: %w", err)
	}
	b, err := json.MarshalIndent(tf, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal team file: %w", err)
	}
	return os.WriteFile(getTeamFilePath(teamName), append(b, '\n'), 0o600)
}

// addTeamMember adds a member to the team roster, creating the team file if needed.
func addTeamMember(teamName string, member TeamFileMember) error {
	tf, err := readTeamFile(teamName)
	if err != nil {
		return err
	}
	if tf == nil {
		tf = &TeamFile{
			Name:        teamName,
			CreatedAt:   time.Now().UnixMilli(),
			LeadAgentID: member.AgentID,
			Members:     []TeamFileMember{},
		}
	}

	// Replace if already exists
	for i, m := range tf.Members {
		if m.AgentID == member.AgentID {
			tf.Members[i] = member
			return writeTeamFile(teamName, tf)
		}
	}

	tf.Members = append(tf.Members, member)
	return writeTeamFile(teamName, tf)
}

// removeTeamMember removes a member from the team roster by agent ID.
func removeTeamMember(teamName, agentID string) error {
	tf, err := readTeamFile(teamName)
	if err != nil {
		return err
	}
	if tf == nil {
		return nil
	}
	filtered := make([]TeamFileMember, 0, len(tf.Members))
	for _, m := range tf.Members {
		if m.AgentID != agentID {
			filtered = append(filtered, m)
		}
	}
	tf.Members = filtered
	return writeTeamFile(teamName, tf)
}

// findTeamMemberByName searches all team files for a member by name.
// Returns the team name and member, or nil if not found.
func findTeamMemberByName(name string) (teamName string, member *TeamFileMember, err error) {
	teamsDir := getTeamsDir()
	entries, err := os.ReadDir(teamsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil, nil
		}
		return "", nil, fmt.Errorf("read teams dir: %w", err)
	}

	name = strings.TrimSpace(name)
	lower := strings.ToLower(name)

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		tf, err := readTeamFile(entry.Name())
		if err != nil || tf == nil {
			continue
		}
		for _, m := range tf.Members {
			if strings.EqualFold(strings.TrimSpace(m.Name), lower) ||
				strings.EqualFold(strings.TrimSpace(m.AgentID), lower) {
				return entry.Name(), &m, nil
			}
		}
	}
	return "", nil, nil
}

// listTeamNames returns all known team names.
func listTeamNames() ([]string, error) {
	teamsDir := getTeamsDir()
	entries, err := os.ReadDir(teamsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	return names, nil
}
