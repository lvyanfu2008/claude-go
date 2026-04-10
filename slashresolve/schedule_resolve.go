package slashresolve

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"goc/claudeinit"
	"goc/commands/featuregates"
	"goc/types"
)

func scheduleUserTimezone() string {
	if tz := strings.TrimSpace(os.Getenv("TZ")); tz != "" {
		return tz
	}
	loc := time.Now().Location()
	if loc != nil {
		s := loc.String()
		if s != "" && s != "Local" {
			return s
		}
	}
	return "Local"
}

func gitOriginHTTPSURL(cwd string) *string {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "config", "--get", "remote.origin.url")
	if strings.TrimSpace(cwd) != "" {
		cmd.Dir = cwd
	}
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	raw := strings.TrimSpace(string(out))
	if raw == "" {
		return nil
	}
	parsed := claudeinit.ParseGitRemote(raw)
	if parsed == nil {
		return nil
	}
	host := parsed.Host
	u := fmt.Sprintf("https://%s/%s/%s", host, parsed.Owner, parsed.Name)
	return &u
}

func inGitRepo(cwd string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--git-dir")
	if strings.TrimSpace(cwd) != "" {
		cmd.Dir = cwd
	}
	return cmd.Run() == nil
}

// resolveSchedule mirrors src/skills/bundled/scheduleRemoteAgents.ts getPromptForCommand.
func resolveSchedule(args string, opt *BundledResolveOptions) (types.SlashResolveResult, error) {
	cwd := ""
	if opt != nil {
		cwd = opt.Cwd
	}
	if cwd == "" {
		cwd = "."
	}
	cwd, _ = filepath.Abs(cwd)

	token := scheduleOAuthToken()
	if token == "" {
		return types.SlashResolveResult{
			UserText: scheduleAuthRequiredMsg,
			Source:   types.SlashResolveBundledEmbed,
		}, nil
	}
	org := scheduleOrganizationUUID()
	if org == "" {
		return types.SlashResolveResult{
			UserText: "Unable to resolve organization UUID for the Environment API. Set CLAUDE_CODE_ORGANIZATION_UUID, or ensure ~/.claude/config.json contains oauthAccount.organizationUuid (same sources as Claude Code TS).",
			Source:   types.SlashResolveBundledEmbed,
		}, nil
	}

	envs, err := fetchScheduleEnvironments(token, org)
	if err != nil {
		return types.SlashResolveResult{
			UserText: scheduleFetchFailMsg,
			Source:   types.SlashResolveBundledEmbed,
		}, nil
	}

	var createdName, createdID string
	if len(envs) == 0 {
		created, cerr := createDefaultCloudEnvironment(token, org, "claude-code-default")
		if cerr != nil {
			return types.SlashResolveResult{
				UserText: scheduleNoEnvMsg,
				Source:   types.SlashResolveBundledEmbed,
			}, nil
		}
		createdName = created.Name
		createdID = created.EnvironmentID
		envs = []environmentResource{created}
	}

	var setupNotes []string
	if !inGitRepo(cwd) {
		setupNotes = append(setupNotes,
			`Not in a git repo — you'll need to specify a repo URL manually (or skip repos entirely).`,
		)
	}
	setupNotes = append(setupNotes,
		`No MCP connectors — connect at https://claude.ai/settings/connectors if needed.`,
	)

	connectorsInfo := "No connected MCP connectors found. The user may need to connect servers at https://claude.ai/settings/connectors"

	var gitURL *string
	if u := gitOriginHTTPSURL(cwd); u != nil {
		gitURL = u
	}

	lines := []string{"Available environments:"}
	for _, env := range envs {
		lines = append(lines, fmt.Sprintf("- %s (id: %s, kind: %s)", env.Name, env.EnvironmentID, env.Kind))
	}
	environmentsInfo := strings.Join(lines, "\n")

	prompt := buildSchedulePrompt(SchedulePromptOpts{
		UserTimezone:              scheduleUserTimezone(),
		ConnectorsInfo:            connectorsInfo,
		GitRepoURL:                gitURL,
		EnvironmentsInfo:          environmentsInfo,
		CreatedEnvironmentName:    createdName,
		CreatedEnvironmentID:      createdID,
		SetupNotes:                setupNotes,
		NeedsGitHubAccessReminder: false,
		WebSetupEnabled:           featuregates.Feature("TENGU_COBALT_LANTERN"),
		UserArgs:                  args,
	})

	return types.SlashResolveResult{
		UserText: prompt,
		Source:   types.SlashResolveBundledEmbed,
	}, nil
}
