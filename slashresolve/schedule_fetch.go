package slashresolve

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// environmentResource mirrors TS EnvironmentResource (subset used by /schedule).
type environmentResource struct {
	Kind          string `json:"kind"`
	EnvironmentID string `json:"environment_id"`
	Name          string `json:"name"`
	CreatedAt     string `json:"created_at"`
	State         string `json:"state"`
}

type environmentListResponse struct {
	Environments []environmentResource `json:"environments"`
}

func scheduleAPIBaseURL() string {
	if v := strings.TrimSpace(os.Getenv("ANTHROPIC_API_BASE_URL")); v != "" {
		return strings.TrimSuffix(v, "/")
	}
	return "https://api.anthropic.com"
}

func scheduleOAuthToken() string {
	if t := strings.TrimSpace(os.Getenv("CLAUDE_CODE_OAUTH_TOKEN")); t != "" {
		return t
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	path := filepath.Join(home, ".claude", ".credentials.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var root map[string]json.RawMessage
	if json.Unmarshal(raw, &root) != nil {
		return ""
	}
	oauthRaw, ok := root["claudeAiOauth"]
	if !ok {
		return ""
	}
	var oauth struct {
		AccessToken string `json:"accessToken"`
	}
	if json.Unmarshal(oauthRaw, &oauth) != nil {
		return ""
	}
	return strings.TrimSpace(oauth.AccessToken)
}

func scheduleOrganizationUUID() string {
	if u := strings.TrimSpace(os.Getenv("CLAUDE_CODE_ORGANIZATION_UUID")); u != "" {
		return u
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	cfgPath := filepath.Join(home, ".claude", "config.json")
	raw, err := os.ReadFile(cfgPath)
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

// fetchScheduleEnvironments mirrors fetchEnvironments() in environments.ts (GET /v1/environment_providers).
func fetchScheduleEnvironments(token, orgUUID string) ([]environmentResource, error) {
	if token == "" {
		return nil, fmt.Errorf("no oauth token")
	}
	if orgUUID == "" {
		return nil, fmt.Errorf("no organization uuid")
	}
	url := scheduleAPIBaseURL() + "/v1/environment_providers"
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("x-organization-uuid", orgUUID)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("environment_providers: %s: %s", resp.Status, string(bytes.TrimSpace(body)))
	}
	var parsed environmentListResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}
	return parsed.Environments, nil
}

// createDefaultCloudEnvironment mirrors createDefaultCloudEnvironment in environments.ts (POST .../cloud/create).
func createDefaultCloudEnvironment(token, orgUUID, name string) (environmentResource, error) {
	var zero environmentResource
	if token == "" || orgUUID == "" {
		return zero, fmt.Errorf("missing auth")
	}
	url := scheduleAPIBaseURL() + "/v1/environment_providers/cloud/create"
	payload := map[string]any{
		"name":        name,
		"kind":        "anthropic_cloud",
		"description": "",
		"config": map[string]any{
			"environment_type": "anthropic",
			"cwd":              "/home/user",
			"init_script":      nil,
			"environment":      map[string]any{},
			"languages": []map[string]string{
				{"name": "python", "version": "3.11"},
				{"name": "node", "version": "20"},
			},
			"network_config": map[string]any{
				"allowed_hosts":        []any{},
				"allow_default_hosts":  true,
			},
		},
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return zero, err
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return zero, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("anthropic-beta", "ccr-byoc-2025-07-29")
	req.Header.Set("x-organization-uuid", orgUUID)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return zero, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return zero, fmt.Errorf("create env: %s: %s", resp.Status, string(bytes.TrimSpace(body)))
	}
	var out environmentResource
	if err := json.Unmarshal(body, &out); err != nil {
		return zero, err
	}
	return out, nil
}
