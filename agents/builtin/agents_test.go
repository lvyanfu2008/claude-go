package builtin

import (
	"os"
	"strings"
	"testing"
)

// --- agent_line.go ---

func TestFormatAgentLine(t *testing.T) {
	tests := []struct {
		name string
		arg  BuiltinAgent
		want string
	}{
		{
			name: "wildcard tools",
			arg: BuiltinAgent{
				AgentType: "test-agent",
				WhenToUse: "does something",
				Tools:     []string{"*"},
			},
			// Wildcard ["*"] is formatted as "*", not "All tools" (that only
			// appears when Tools is nil/empty and DisallowedTools is also empty).
			want: "- test-agent: does something (Tools: *)",
		},
		{
			name: "explicit tools",
			arg: BuiltinAgent{
				AgentType: "reader",
				WhenToUse: "reads files",
				Tools:     []string{"Read", "Glob"},
			},
			want: "- reader: reads files (Tools: Read, Glob)",
		},
		{
			name: "disallowed tools only",
			arg: BuiltinAgent{
				AgentType:       "safe",
				WhenToUse:       "read-only",
				DisallowedTools: []string{"Edit", "Write"},
			},
			want: "- safe: read-only (Tools: All tools except Edit, Write)",
		},
		{
			name: "both allow and deny with overlap",
			arg: BuiltinAgent{
				AgentType:       "mixed",
				WhenToUse:       "overlap case",
				Tools:           []string{"Read", "Write", "Edit"},
				DisallowedTools: []string{"Edit"},
			},
			want: "- mixed: overlap case (Tools: Read, Write)",
		},
		{
			name: "all tools denied",
			arg: BuiltinAgent{
				AgentType:       "blocked",
				WhenToUse:       "nothing",
				Tools:           []string{"Read", "Write"},
				DisallowedTools: []string{"Read", "Write"},
			},
			want: "- blocked: nothing (Tools: None)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatAgentLine(tt.arg)
			if got != tt.want {
				t.Errorf("FormatAgentLine() = %q, want %q", got, tt.want)
			}
		})
	}
}

// --- config.go ---

func TestConfigFromEnv(t *testing.T) {
	// Save env and restore after test.
	save := func(k string) func() {
		v, ok := os.LookupEnv(k)
		return func() {
			if ok {
				os.Setenv(k, v)
			} else {
				os.Unsetenv(k)
			}
		}
	}
	defer save("FEATURE_CHICAGO_MCP")()
	defer save("CLAUDE_CODE_GO_EMBEDDED_SEARCH_TOOLS")()
	defer save("USER_TYPE")()
	defer save("CLAUDE_CODE_NONINTERACTIVE")()
	defer save("HEADLESS")()
	defer save("CLAUDE_CODE_ENTRYPOINT")()
	defer save("CLAUDE_CODE_USE_BEDROCK")()
	defer save("CLAUDE_CODE_ISSUES_EXPLAINER")()

	os.Unsetenv("FEATURE_CHICAGO_MCP")
	os.Unsetenv("CLAUDE_CODE_GO_EMBEDDED_SEARCH_TOOLS")
	os.Unsetenv("USER_TYPE")
	os.Unsetenv("CLAUDE_CODE_NONINTERACTIVE")
	os.Unsetenv("HEADLESS")
	os.Unsetenv("CLAUDE_CODE_ENTRYPOINT")
	os.Unsetenv("CLAUDE_CODE_USE_BEDROCK")
	os.Unsetenv("CLAUDE_CODE_ISSUES_EXPLAINER")

	t.Run("defaults", func(t *testing.T) {
		cfg := ConfigFromEnv()
		if cfg.EmbeddedSearchTools {
			t.Error("EmbeddedSearchTools should be false by default")
		}
		if cfg.UserTypeAnt {
			t.Error("UserTypeAnt should be false by default")
		}
		if cfg.NonInteractive {
			t.Error("NonInteractive should be false by default")
		}
		if cfg.Entrypoint != "" {
			t.Errorf("Entrypoint should be empty, got %q", cfg.Entrypoint)
		}
		if cfg.Using3PServices {
			t.Error("Using3PServices should be false by default")
		}
	})

	t.Run("embedded search via feature flag", func(t *testing.T) {
		os.Setenv("FEATURE_CHICAGO_MCP", "1")
		defer os.Unsetenv("FEATURE_CHICAGO_MCP")
		cfg := ConfigFromEnv()
		if !cfg.EmbeddedSearchTools {
			t.Error("EmbeddedSearchTools should be true when FEATURE_CHICAGO_MCP=1")
		}
	})

	t.Run("embedded search via Go-specific env", func(t *testing.T) {
		os.Setenv("CLAUDE_CODE_GO_EMBEDDED_SEARCH_TOOLS", "1")
		defer os.Unsetenv("CLAUDE_CODE_GO_EMBEDDED_SEARCH_TOOLS")
		cfg := ConfigFromEnv()
		if !cfg.EmbeddedSearchTools {
			t.Error("EmbeddedSearchTools should be true when CLAUDE_CODE_GO_EMBEDDED_SEARCH_TOOLS=1")
		}
	})

	t.Run("ant user", func(t *testing.T) {
		os.Setenv("USER_TYPE", "ant")
		defer os.Unsetenv("USER_TYPE")
		cfg := ConfigFromEnv()
		if !cfg.UserTypeAnt {
			t.Error("UserTypeAnt should be true when USER_TYPE=ant")
		}
	})

	t.Run("noninteractive", func(t *testing.T) {
		os.Setenv("CLAUDE_CODE_NONINTERACTIVE", "1")
		defer os.Unsetenv("CLAUDE_CODE_NONINTERACTIVE")
		cfg := ConfigFromEnv()
		if !cfg.NonInteractive {
			t.Error("NonInteractive should be true")
		}
	})

	t.Run("noninteractive via HEADLESS", func(t *testing.T) {
		os.Setenv("HEADLESS", "1")
		defer os.Unsetenv("HEADLESS")
		cfg := ConfigFromEnv()
		if !cfg.NonInteractive {
			t.Error("NonInteractive should be true when HEADLESS=1")
		}
	})

	t.Run("entrypoint", func(t *testing.T) {
		os.Setenv("CLAUDE_CODE_ENTRYPOINT", "sdk-py")
		defer os.Unsetenv("CLAUDE_CODE_ENTRYPOINT")
		cfg := ConfigFromEnv()
		if cfg.Entrypoint != "sdk-py" {
			t.Errorf("Entrypoint = %q, want %q", cfg.Entrypoint, "sdk-py")
		}
	})

	t.Run("3P service", func(t *testing.T) {
		os.Setenv("CLAUDE_CODE_USE_BEDROCK", "1")
		defer os.Unsetenv("CLAUDE_CODE_USE_BEDROCK")
		cfg := ConfigFromEnv()
		if !cfg.Using3PServices {
			t.Error("Using3PServices should be true when CLAUDE_CODE_USE_BEDROCK=1")
		}
	})

	t.Run("issues explainer", func(t *testing.T) {
		os.Setenv("CLAUDE_CODE_ISSUES_EXPLAINER", "custom channel")
		defer os.Unsetenv("CLAUDE_CODE_ISSUES_EXPLAINER")
		cfg := ConfigFromEnv()
		if cfg.IssuesExplainer != "custom channel" {
			t.Errorf("IssuesExplainer = %q, want %q", cfg.IssuesExplainer, "custom channel")
		}
	})
}

func TestAreExplorePlanAgentsEnabled(t *testing.T) {
	save := func(k string) func() {
		v, ok := os.LookupEnv(k)
		return func() {
			if ok {
				os.Setenv(k, v)
			} else {
				os.Unsetenv(k)
			}
		}
	}
	defer save("FEATURE_BUILTIN_EXPLORE_PLAN_AGENTS")()
	defer save("CLAUDE_CODE_TENGU_AMBER_STOAT")()

	t.Run("feature flag off", func(t *testing.T) {
		os.Unsetenv("FEATURE_BUILTIN_EXPLORE_PLAN_AGENTS")
		os.Unsetenv("CLAUDE_CODE_TENGU_AMBER_STOAT")
		if AreExplorePlanAgentsEnabled() {
			t.Error("should be false when feature flag is unset")
		}
	})

	t.Run("feature flag on, no tengu gate", func(t *testing.T) {
		os.Setenv("FEATURE_BUILTIN_EXPLORE_PLAN_AGENTS", "1")
		os.Unsetenv("CLAUDE_CODE_TENGU_AMBER_STOAT")
		if !AreExplorePlanAgentsEnabled() {
			t.Error("should be true (default true when tengu env unset)")
		}
	})

	t.Run("feature flag on, tengu gate false", func(t *testing.T) {
		os.Setenv("FEATURE_BUILTIN_EXPLORE_PLAN_AGENTS", "1")
		os.Setenv("CLAUDE_CODE_TENGU_AMBER_STOAT", "0")
		if AreExplorePlanAgentsEnabled() {
			t.Error("should be false when tengu gate is 0")
		}
	})
}

func TestIncludeVerificationAgent(t *testing.T) {
	save := func(k string) func() {
		v, ok := os.LookupEnv(k)
		return func() {
			if ok {
				os.Setenv(k, v)
			} else {
				os.Unsetenv(k)
			}
		}
	}
	defer save("FEATURE_VERIFICATION_AGENT")()
	defer save("CLAUDE_CODE_TENGU_HIVE_EVIDENCE")()

	t.Run("feature off", func(t *testing.T) {
		os.Unsetenv("FEATURE_VERIFICATION_AGENT")
		os.Unsetenv("CLAUDE_CODE_TENGU_HIVE_EVIDENCE")
		if includeVerificationAgent() {
			t.Error("should be false when VERIFICATION_AGENT feature is off")
		}
	})

	t.Run("feature on but gate off", func(t *testing.T) {
		os.Setenv("FEATURE_VERIFICATION_AGENT", "1")
		os.Unsetenv("CLAUDE_CODE_TENGU_HIVE_EVIDENCE")
		if includeVerificationAgent() {
			t.Error("should be false when tengu_hive_evidence is off")
		}
	})

	t.Run("both on", func(t *testing.T) {
		os.Setenv("FEATURE_VERIFICATION_AGENT", "1")
		os.Setenv("CLAUDE_CODE_TENGU_HIVE_EVIDENCE", "1")
		if !includeVerificationAgent() {
			t.Error("should be true when both gates are on")
		}
	})
}

func TestDisableAllBuiltinsForSDK(t *testing.T) {
	save := func(k string) func() {
		v, ok := os.LookupEnv(k)
		return func() {
			if ok {
				os.Setenv(k, v)
			} else {
				os.Unsetenv(k)
			}
		}
	}
	defer save("CLAUDE_AGENT_SDK_DISABLE_BUILTIN_AGENTS")()

	t.Run("env not set", func(t *testing.T) {
		os.Unsetenv("CLAUDE_AGENT_SDK_DISABLE_BUILTIN_AGENTS")
		if disableAllBuiltinsForSDK(Config{NonInteractive: true}) {
			t.Error("should be false when env is not set")
		}
	})

	t.Run("env set but not noninteractive", func(t *testing.T) {
		os.Setenv("CLAUDE_AGENT_SDK_DISABLE_BUILTIN_AGENTS", "1")
		if disableAllBuiltinsForSDK(Config{NonInteractive: false}) {
			t.Error("should be false when not noninteractive")
		}
	})

	t.Run("both set", func(t *testing.T) {
		os.Setenv("CLAUDE_AGENT_SDK_DISABLE_BUILTIN_AGENTS", "1")
		if !disableAllBuiltinsForSDK(Config{NonInteractive: true}) {
			t.Error("should be true when both conditions met")
		}
	})
}

func TestIsNonSdkEntrypoint(t *testing.T) {
	tests := []struct {
		entry string
		want  bool
	}{
		{"", true},
		{"cli", true},
		{"sdk-ts", false},
		{"sdk-py", false},
		{"sdk-cli", false},
		{"daemon", true},
	}
	for _, tt := range tests {
		t.Run(tt.entry, func(t *testing.T) {
			got := isNonSdkEntrypoint(tt.entry)
			if got != tt.want {
				t.Errorf("isNonSdkEntrypoint(%q) = %v, want %v", tt.entry, got, tt.want)
			}
		})
	}
}

// --- agents.go ---

func TestGetBuiltInAgents_basic(t *testing.T) {
	// Save and clear relevant env vars.
	save := func(k string) func() {
		v, ok := os.LookupEnv(k)
		return func() {
			if ok {
				os.Setenv(k, v)
			} else {
				os.Unsetenv(k)
			}
		}
	}
	defer save("FEATURE_BUILTIN_EXPLORE_PLAN_AGENTS")()
	defer save("FEATURE_VERIFICATION_AGENT")()
	defer save("CLAUDE_CODE_TENGU_HIVE_EVIDENCE")()
	defer save("CLAUDE_CODE_ENTRYPOINT")()
	defer save("CLAUDE_AGENT_SDK_DISABLE_BUILTIN_AGENTS")()
	defer save("CLAUDE_CODE_NONINTERACTIVE")()
	defer save("USER_TYPE")()

	os.Unsetenv("FEATURE_BUILTIN_EXPLORE_PLAN_AGENTS")
	os.Unsetenv("FEATURE_VERIFICATION_AGENT")
	os.Unsetenv("CLAUDE_CODE_TENGU_HIVE_EVIDENCE")
	os.Unsetenv("CLAUDE_CODE_ENTRYPOINT")
	os.Unsetenv("CLAUDE_AGENT_SDK_DISABLE_BUILTIN_AGENTS")
	os.Unsetenv("CLAUDE_CODE_NONINTERACTIVE")
	os.Unsetenv("USER_TYPE")

	t.Run("minimal set", func(t *testing.T) {
		cfg := Config{
			UserTypeAnt:    false,
			Entrypoint:     "cli",
			Using3PServices: false,
		}
		agents := GetBuiltInAgents(cfg, GuideContext{})
		// Should have: general-purpose, statusline-setup, claude-code-guide.
		// (Explore, Plan, Verification all gated off.)
		if len(agents) != 3 {
			t.Fatalf("expected 3 agents (general, statusline, guide), got %d: %+v", len(agents), agentTypes(agents))
		}
		assertAgentType(t, agents, "general-purpose")
		assertAgentType(t, agents, "statusline-setup")
		assertAgentType(t, agents, "claude-code-guide")
	})

	t.Run("SDK entrypoint disables guide", func(t *testing.T) {
		cfg := Config{
			Entrypoint: "sdk-py",
		}
		agents := GetBuiltInAgents(cfg, GuideContext{})
		for _, a := range agents {
			if a.AgentType == "claude-code-guide" {
				t.Error("claude-code-guide should be excluded for SDK entrypoints")
			}
		}
	})

	t.Run("SDK disable all builtins", func(t *testing.T) {
		os.Setenv("CLAUDE_AGENT_SDK_DISABLE_BUILTIN_AGENTS", "1")
		defer os.Unsetenv("CLAUDE_AGENT_SDK_DISABLE_BUILTIN_AGENTS")
		cfg := Config{NonInteractive: true}
		agents := GetBuiltInAgents(cfg, GuideContext{})
		if len(agents) != 0 {
			t.Errorf("expected 0 agents when SDK disable is on, got %d", len(agents))
		}
	})

	t.Run("with explore and plan", func(t *testing.T) {
		os.Setenv("FEATURE_BUILTIN_EXPLORE_PLAN_AGENTS", "1")
		defer os.Unsetenv("FEATURE_BUILTIN_EXPLORE_PLAN_AGENTS")
		cfg := Config{UserTypeAnt: false, Entrypoint: "cli"}
		agents := GetBuiltInAgents(cfg, GuideContext{})
		assertAgentType(t, agents, "Explore")
		assertAgentType(t, agents, "Plan")
	})

	t.Run("with verification", func(t *testing.T) {
		os.Setenv("FEATURE_VERIFICATION_AGENT", "1")
		os.Setenv("CLAUDE_CODE_TENGU_HIVE_EVIDENCE", "1")
		defer os.Unsetenv("FEATURE_VERIFICATION_AGENT")
		defer os.Unsetenv("CLAUDE_CODE_TENGU_HIVE_EVIDENCE")
		cfg := Config{Entrypoint: "cli"}
		agents := GetBuiltInAgents(cfg, GuideContext{})
		assertAgentType(t, agents, "verification")
	})

	t.Run("explore model for ant vs non-ant", func(t *testing.T) {
		os.Setenv("FEATURE_BUILTIN_EXPLORE_PLAN_AGENTS", "1")
		defer os.Unsetenv("FEATURE_BUILTIN_EXPLORE_PLAN_AGENTS")

		nonAnt := GetBuiltInAgents(Config{UserTypeAnt: false, Entrypoint: "cli"}, GuideContext{})
		exploreNonAnt := findAgent(t, nonAnt, "Explore")
		if exploreNonAnt.Model != "haiku" {
			t.Errorf("non-ant Explore model = %q, want %q", exploreNonAnt.Model, "haiku")
		}

		ant := GetBuiltInAgents(Config{UserTypeAnt: true, Entrypoint: "cli"}, GuideContext{})
		exploreAnt := findAgent(t, ant, "Explore")
		if exploreAnt.Model != "inherit" {
			t.Errorf("ant Explore model = %q, want %q", exploreAnt.Model, "inherit")
		}
	})

	t.Run("statusline-setup properties", func(t *testing.T) {
		agents := GetBuiltInAgents(Config{Entrypoint: "cli"}, GuideContext{})
		sl := findAgent(t, agents, "statusline-setup")
		if sl.Model != "sonnet" {
			t.Errorf("statusline model = %q, want %q", sl.Model, "sonnet")
		}
		if sl.Color != "orange" {
			t.Errorf("statusline color = %q, want %q", sl.Color, "orange")
		}
		if len(sl.Tools) != 2 || sl.Tools[0] != "Read" || sl.Tools[1] != "Edit" {
			t.Errorf("statusline tools = %v, want [Read Edit]", sl.Tools)
		}
	})

	t.Run("verification properties", func(t *testing.T) {
		os.Setenv("FEATURE_VERIFICATION_AGENT", "1")
		os.Setenv("CLAUDE_CODE_TENGU_HIVE_EVIDENCE", "1")
		defer os.Unsetenv("FEATURE_VERIFICATION_AGENT")
		defer os.Unsetenv("CLAUDE_CODE_TENGU_HIVE_EVIDENCE")
		agents := GetBuiltInAgents(Config{Entrypoint: "cli"}, GuideContext{})
		v := findAgent(t, agents, "verification")
		if v.Color != "red" {
			t.Errorf("verification color = %q, want %q", v.Color, "red")
		}
		if !v.Background {
			t.Error("verification.Background should be true")
		}
		if v.CriticalSystemReminderExperimental == "" {
			t.Error("verification should have CriticalSystemReminderExperimental")
		}
	})
}

func TestGetBuiltInAgents_sourceAndBaseDir(t *testing.T) {
	agents := GetBuiltInAgents(Config{Entrypoint: "cli"}, GuideContext{})
	for _, a := range agents {
		if a.Source != "built-in" {
			t.Errorf("agent %q Source = %q, want %q", a.AgentType, a.Source, "built-in")
		}
		if a.BaseDir != "built-in" {
			t.Errorf("agent %q BaseDir = %q, want %q", a.AgentType, a.BaseDir, "built-in")
		}
	}
}

func TestGetBuiltInAgents_guideTools(t *testing.T) {
	t.Run("non-embedded", func(t *testing.T) {
		agents := GetBuiltInAgents(Config{EmbeddedSearchTools: false, Entrypoint: "cli"}, GuideContext{})
		g := findAgent(t, agents, "claude-code-guide")
		expected := []string{"Glob", "Grep", "Read", "WebFetch", "WebSearch"}
		if !stringSliceEq(g.Tools, expected) {
			t.Errorf("guide tools (non-embedded) = %v, want %v", g.Tools, expected)
		}
	})

	t.Run("embedded", func(t *testing.T) {
		agents := GetBuiltInAgents(Config{EmbeddedSearchTools: true, Entrypoint: "cli"}, GuideContext{})
		g := findAgent(t, agents, "claude-code-guide")
		expected := []string{"Bash", "Read", "WebFetch", "WebSearch"}
		if !stringSliceEq(g.Tools, expected) {
			t.Errorf("guide tools (embedded) = %v, want %v", g.Tools, expected)
		}
	})
}

func TestGetBuiltInAgents_permissionMode(t *testing.T) {
	cfg := Config{Entrypoint: "cli"}
	agents := GetBuiltInAgents(cfg, GuideContext{})
	g := findAgent(t, agents, "claude-code-guide")
	if g.PermissionMode != "dontAsk" {
		t.Errorf("guide permissionMode = %q, want %q", g.PermissionMode, "dontAsk")
	}
}

// --- guide.go ---

func TestGuideFeedbackLine(t *testing.T) {
	t.Run("3P service", func(t *testing.T) {
		got := guideFeedbackLine(Config{Using3PServices: true, IssuesExplainer: ""})
		if !strings.Contains(got, "cloud provider's documentation") {
			t.Errorf("3P feedback should mention cloud provider, got: %s", got)
		}
	})

	t.Run("3P service with custom explainer", func(t *testing.T) {
		got := guideFeedbackLine(Config{Using3PServices: true, IssuesExplainer: "custom help page"})
		if !strings.Contains(got, "custom help page") {
			t.Errorf("3P feedback should use custom explainer, got: %s", got)
		}
	})

	t.Run("non-3P service", func(t *testing.T) {
		got := guideFeedbackLine(Config{Using3PServices: false})
		if !strings.Contains(got, "/feedback") {
			t.Errorf("non-3P feedback should mention /feedback, got: %s", got)
		}
	})
}

func TestGuideTools(t *testing.T) {
	t.Run("non-embedded", func(t *testing.T) {
		tools := guideTools(false)
		expected := []string{"Glob", "Grep", "Read", "WebFetch", "WebSearch"}
		if !stringSliceEq(tools, expected) {
			t.Errorf("guideTools(false) = %v, want %v", tools, expected)
		}
	})

	t.Run("embedded", func(t *testing.T) {
		tools := guideTools(true)
		expected := []string{"Bash", "Read", "WebFetch", "WebSearch"}
		if !stringSliceEq(tools, expected) {
			t.Errorf("guideTools(true) = %v, want %v", tools, expected)
		}
	})
}

func TestClaudeCodeGuideSystemPrompt(t *testing.T) {
	t.Run("base prompt without context", func(t *testing.T) {
		p := ClaudeCodeGuideSystemPrompt(Config{}, GuideContext{})
		if p == "" {
			t.Fatal("system prompt should not be empty")
		}
		if !strings.Contains(p, "Claude guide agent") {
			t.Error("prompt should mention 'Claude guide agent'")
		}
		if !strings.Contains(p, "code.claude.com") {
			t.Error("prompt should reference Claude Code docs URL")
		}
		if !strings.Contains(p, "WebFetch") {
			t.Error("prompt should mention WebFetch")
		}
	})

	t.Run("base prompt with embedded search", func(t *testing.T) {
		p := ClaudeCodeGuideSystemPrompt(Config{EmbeddedSearchTools: true}, GuideContext{})
		if !strings.Contains(p, "`find`") {
			t.Error("embedded prompt should mention `find`")
		}
	})

	t.Run("with custom skills", func(t *testing.T) {
		ctx := GuideContext{
			Commands: []GuideCommand{
				{Name: "test", Description: "runs tests", Type: "prompt"},
			},
		}
		p := ClaudeCodeGuideSystemPrompt(Config{}, ctx)
		if !strings.Contains(p, "/test") {
			t.Error("prompt should list the custom skill")
		}
	})

	t.Run("with custom agents", func(t *testing.T) {
		ctx := GuideContext{
			CustomAgents: []GuideAgentRef{
				{AgentType: "my-agent", WhenToUse: "does things"},
			},
		}
		p := ClaudeCodeGuideSystemPrompt(Config{}, ctx)
		if !strings.Contains(p, "my-agent") {
			t.Error("prompt should list the custom agent")
		}
	})

	t.Run("with MCP clients", func(t *testing.T) {
		ctx := GuideContext{
			MCPClients: []GuideMCPClient{
				{Name: "my-mcp-server"},
			},
		}
		p := ClaudeCodeGuideSystemPrompt(Config{}, ctx)
		if !strings.Contains(p, "my-mcp-server") {
			t.Error("prompt should list the MCP server")
		}
	})

	t.Run("with settings JSON", func(t *testing.T) {
		ctx := GuideContext{
			SettingsJSON: `{"key": "value"}`,
		}
		p := ClaudeCodeGuideSystemPrompt(Config{}, ctx)
		if !strings.Contains(p, `"key": "value"`) {
			t.Error("prompt should include settings JSON")
		}
	})

	t.Run("with plugin commands", func(t *testing.T) {
		ctx := GuideContext{
			Commands: []GuideCommand{
				{Name: "my-plugin", Description: "plugin skill", Type: "prompt", Source: "plugin"},
			},
		}
		p := ClaudeCodeGuideSystemPrompt(Config{}, ctx)
		if !strings.Contains(p, "/my-plugin") {
			t.Error("prompt should list the plugin skill")
		}
	})

	t.Run("with 3P feedback line", func(t *testing.T) {
		p := ClaudeCodeGuideSystemPrompt(
			Config{Using3PServices: true, IssuesExplainer: ""},
			GuideContext{},
		)
		if strings.Contains(p, "/feedback") {
			t.Error("3P prompt should NOT mention /feedback")
		}
		if !strings.Contains(p, "cloud provider") {
			t.Error("3P prompt should redirect to cloud provider")
		}
	})

	t.Run("all context sections", func(t *testing.T) {
		ctx := GuideContext{
			Commands: []GuideCommand{
				{Name: "skill1", Description: "first skill", Type: "prompt"},
				{Name: "plugin1", Description: "plugin skill", Type: "prompt", Source: "plugin"},
			},
			CustomAgents: []GuideAgentRef{
				{AgentType: "agent1", WhenToUse: "does work"},
			},
			MCPClients: []GuideMCPClient{
				{Name: "mcp1"},
			},
			SettingsJSON: `{"theme": "dark"}`,
		}
		p := ClaudeCodeGuideSystemPrompt(Config{}, ctx)
		if !strings.Contains(p, "User's Current Configuration") {
			t.Error("prompt should include the User's Current Configuration section")
		}
	})
}

func TestClaudeCodeGuideBasePrompt(t *testing.T) {
	t.Run("non-embedded", func(t *testing.T) {
		p := claudeCodeGuideBasePrompt(false)
		if !strings.Contains(p, "Glob") || !strings.Contains(p, "Grep") {
			t.Error("non-embedded prompt should mention Glob and Grep")
		}
	})

	t.Run("embedded", func(t *testing.T) {
		p := claudeCodeGuideBasePrompt(true)
		if !strings.Contains(p, "`find`") || !strings.Contains(p, "`grep`") {
			t.Error("embedded prompt should mention `find` and `grep`")
		}
	})
}

// --- prompts_general.go ---

func TestGeneralPurposeSystemPrompt(t *testing.T) {
	p := generalPurposeSystemPrompt()
	if p == "" {
		t.Fatal("system prompt should not be empty")
	}
	if !strings.Contains(p, "Claude Code") {
		t.Error("prompt should mention Claude Code")
	}
	if !strings.Contains(p, "multi-step") {
		t.Error("prompt should mention multi-step research")
	}
}

// --- prompts_explore_plan.go ---

func TestExploreSystemPrompt(t *testing.T) {
	t.Run("non-embedded", func(t *testing.T) {
		p := exploreSystemPrompt(false)
		if !strings.Contains(p, "Glob") {
			t.Error("non-embedded prompt should mention Glob")
		}
		if strings.Contains(p, "`find`") {
			t.Error("non-embedded prompt should not mention `find`")
		}
		if !strings.Contains(p, "READ-ONLY") {
			t.Error("prompt should contain READ-ONLY warning")
		}
	})

	t.Run("embedded", func(t *testing.T) {
		p := exploreSystemPrompt(true)
		if !strings.Contains(p, "`find`") {
			t.Error("embedded prompt should mention `find` via Bash")
		}
		if !strings.Contains(p, "`grep`") {
			t.Error("embedded prompt should mention `grep` via Bash")
		}
	})
}

func TestPlanSystemPrompt(t *testing.T) {
	t.Run("non-embedded", func(t *testing.T) {
		p := planSystemPrompt(false)
		if !strings.Contains(p, "Glob") {
			t.Error("non-embedded prompt should mention Glob")
		}
		if !strings.Contains(p, "Critical Files for Implementation") {
			t.Error("plan prompt should mention Critical Files for Implementation")
		}
	})

	t.Run("embedded", func(t *testing.T) {
		p := planSystemPrompt(true)
		if !strings.Contains(p, "`find`") {
			t.Error("embedded prompt should mention `find`")
		}
	})
}

func TestExploreMinQueries(t *testing.T) {
	if ExploreMinQueries != 3 {
		t.Errorf("ExploreMinQueries = %d, want 3", ExploreMinQueries)
	}
}

// --- statusline.go ---

func TestStatuslineSystemPrompt(t *testing.T) {
	p := StatuslineSystemPrompt()
	if p == "" {
		t.Fatal("statusline prompt should not be empty")
	}
	if !strings.Contains(p, "status line setup agent") {
		t.Error("prompt should mention status line setup agent")
	}
	if !strings.Contains(p, "settings.json") {
		t.Error("prompt should mention settings.json")
	}
	if !strings.Contains(p, "PS1") {
		t.Error("prompt should mention PS1")
	}
}

// --- verification.go ---

func TestVerificationSystemPrompt(t *testing.T) {
	t.Run("contains key content", func(t *testing.T) {
		p := VerificationSystemPrompt()
		if p == "" {
			t.Fatal("verification prompt should not be empty")
		}
		if !strings.Contains(p, "verification specialist") {
			t.Error("prompt should mention verification specialist")
		}
		if !strings.Contains(p, "VERDICT:") {
			t.Error("prompt should mention VERDICT format")
		}
	})

	t.Run("tool name substitution", func(t *testing.T) {
		p := VerificationSystemPrompt()
		if !strings.Contains(p, "Bash") {
			t.Error("prompt should contain 'Bash' after substitution")
		}
		if !strings.Contains(p, "WebFetch") {
			t.Error("prompt should contain 'WebFetch' after substitution")
		}
	})
}

// --- helpers ---

func agentTypes(agents []BuiltinAgent) []string {
	out := make([]string, len(agents))
	for i, a := range agents {
		out[i] = a.AgentType
	}
	return out
}

func findAgent(t *testing.T, agents []BuiltinAgent, agentType string) BuiltinAgent {
	t.Helper()
	for _, a := range agents {
		if a.AgentType == agentType {
			return a
		}
	}
	t.Fatalf("agent %q not found in list: %v", agentType, agentTypes(agents))
	return BuiltinAgent{}
}

func assertAgentType(t *testing.T, agents []BuiltinAgent, agentType string) {
	t.Helper()
	for _, a := range agents {
		if a.AgentType == agentType {
			return
		}
	}
	t.Errorf("expected agent %q not found in list: %v", agentType, agentTypes(agents))
}

func stringSliceEq(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// --- config.go — direct tests for helpers ---

func TestEnvTruthy(t *testing.T) {
	tests := []struct {
		name string
		val  string
		want bool
	}{
		{"unset", "", false},
		{"zero", "0", false},
		{"false", "false", false},
		{"no", "no", false},
		{"off", "off", false},
		{"uppercase TRUE", "TRUE", true},
		{"mixed True", "True", true},
		{"one", "1", true},
		{"yes", "yes", true},
		{"on", "on", true},
		{"whitespace around 1", " 1 ", true},
		{"whitespace around true", " true ", true},
		{"garbage", "garbage", false},
		{"empty spaces", "   ", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("TEST_ENV_TRUTHY", tt.val)
			defer os.Unsetenv("TEST_ENV_TRUTHY")
			if got := envTruthy("TEST_ENV_TRUTHY"); got != tt.want {
				t.Errorf("envTruthy(%q) = %v, want %v", tt.val, got, tt.want)
			}
		})
	}
}

func TestEnvUsing3P(t *testing.T) {
	save := func(k string) func() {
		v, ok := os.LookupEnv(k)
		return func() {
			if ok {
				os.Setenv(k, v)
			} else {
				os.Unsetenv(k)
			}
		}
	}
	defer save("CLAUDE_CODE_USE_BEDROCK")()
	defer save("CLAUDE_CODE_USE_VERTEX")()
	defer save("CLAUDE_CODE_USE_FOUNDRY")()

	t.Run("none set", func(t *testing.T) {
		os.Unsetenv("CLAUDE_CODE_USE_BEDROCK")
		os.Unsetenv("CLAUDE_CODE_USE_VERTEX")
		os.Unsetenv("CLAUDE_CODE_USE_FOUNDRY")
		if envUsing3P() {
			t.Error("envUsing3P should be false when no 3P env is set")
		}
	})

	t.Run("bedrock", func(t *testing.T) {
		os.Unsetenv("CLAUDE_CODE_USE_VERTEX")
		os.Unsetenv("CLAUDE_CODE_USE_FOUNDRY")
		os.Setenv("CLAUDE_CODE_USE_BEDROCK", "1")
		if !envUsing3P() {
			t.Error("envUsing3P should be true for BEDROCK")
		}
	})

	t.Run("vertex", func(t *testing.T) {
		os.Unsetenv("CLAUDE_CODE_USE_BEDROCK")
		os.Unsetenv("CLAUDE_CODE_USE_FOUNDRY")
		os.Setenv("CLAUDE_CODE_USE_VERTEX", "1")
		if !envUsing3P() {
			t.Error("envUsing3P should be true for VERTEX")
		}
	})

	t.Run("foundry", func(t *testing.T) {
		os.Unsetenv("CLAUDE_CODE_USE_BEDROCK")
		os.Unsetenv("CLAUDE_CODE_USE_VERTEX")
		os.Setenv("CLAUDE_CODE_USE_FOUNDRY", "1")
		if !envUsing3P() {
			t.Error("envUsing3P should be true for FOUNDRY")
		}
	})
}

func TestAreExplorePlanAgentsEnabled_tenguGateOn(t *testing.T) {
	save := func(k string) func() {
		v, ok := os.LookupEnv(k)
		return func() {
			if ok {
				os.Setenv(k, v)
			} else {
				os.Unsetenv(k)
			}
		}
	}
	defer save("FEATURE_BUILTIN_EXPLORE_PLAN_AGENTS")()
	defer save("CLAUDE_CODE_TENGU_AMBER_STOAT")()

	t.Run("tengu gate truthy", func(t *testing.T) {
		os.Setenv("FEATURE_BUILTIN_EXPLORE_PLAN_AGENTS", "1")
		os.Setenv("CLAUDE_CODE_TENGU_AMBER_STOAT", "yes")
		if !AreExplorePlanAgentsEnabled() {
			t.Error("should be true when tengu gate is 'yes'")
		}
	})

	t.Run("tengu gate falsey via 'no'", func(t *testing.T) {
		os.Setenv("FEATURE_BUILTIN_EXPLORE_PLAN_AGENTS", "1")
		os.Setenv("CLAUDE_CODE_TENGU_AMBER_STOAT", "no")
		if AreExplorePlanAgentsEnabled() {
			t.Error("should be false when tengu gate is 'no'")
		}
	})
}

// --- agent_line.go — uncovered edge case ---

func TestFormatAgentToolsDescription_defaultAllTools(t *testing.T) {
	// When neither Tools nor DisallowedTools is set, the description should be "All tools".
	a := BuiltinAgent{
		AgentType: "default-tools",
		WhenToUse: "unspecified tools",
	}
	want := "All tools"
	got := formatAgentToolsDescription(a)
	if got != want {
		t.Errorf("formatAgentToolsDescription() = %q, want %q", got, want)
	}
}

// --- agents.go — claudeCodeGuideWhenToUse ---

func TestClaudeCodeGuideWhenToUse(t *testing.T) {
	result := claudeCodeGuideWhenToUse()
	if result == "" {
		t.Fatal("claudeCodeGuideWhenToUse() should not be empty")
	}
	if !strings.Contains(result, "Claude Code") {
		t.Error("should mention Claude Code")
	}
	if !strings.Contains(result, "Claude Agent SDK") {
		t.Error("should mention Claude Agent SDK")
	}
	if !strings.Contains(result, "Claude API") {
		t.Error("should mention Claude API")
	}
	if !strings.Contains(result, ToolSendMessage) {
		t.Error("should mention SendMessage tool")
	}
}
