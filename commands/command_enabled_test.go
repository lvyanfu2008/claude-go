package commands

import (
	"testing"

	"goc/types"
)

func TestIsCommandEnabledData_DoctorDisabledByEnv(t *testing.T) {
	t.Setenv("DISABLE_DOCTOR_COMMAND", "1")
	cmd := types.Command{CommandBase: types.CommandBase{Name: "doctor"}, Type: "local-jsx"}
	if IsCommandEnabledData(cmd, DefaultConsoleAPIAuth()) {
		t.Fatal("expected doctor disabled")
	}
}

func TestIsCommandEnabledData_TagRequiresAnt(t *testing.T) {
	cmd := types.Command{CommandBase: types.CommandBase{Name: "tag"}, Type: "local-jsx"}
	t.Setenv("USER_TYPE", "")
	if IsCommandEnabledData(cmd, DefaultConsoleAPIAuth()) {
		t.Fatal("expected tag off without USER_TYPE=ant")
	}
	t.Setenv("USER_TYPE", "ant")
	if !IsCommandEnabledData(cmd, DefaultConsoleAPIAuth()) {
		t.Fatal("expected tag on for ant")
	}
}

func TestIsCommandEnabledData_ExtraUsageByTypeAndAuth(t *testing.T) {
	auth := DefaultConsoleAPIAuth()
	auth.ExtraUsageAllowed = true
	jsx := types.Command{CommandBase: types.CommandBase{Name: "extra-usage"}, Type: "local-jsx"}
	loc := types.Command{CommandBase: types.CommandBase{Name: "extra-usage"}, Type: "local"}
	if !IsCommandEnabledData(jsx, auth) || IsCommandEnabledData(loc, auth) {
		t.Fatalf("interactive: want jsx on local off, jsx=%v local=%v",
			IsCommandEnabledData(jsx, auth), IsCommandEnabledData(loc, auth))
	}
	auth.IsNonInteractiveSession = true
	if IsCommandEnabledData(jsx, auth) || !IsCommandEnabledData(loc, auth) {
		t.Fatalf("non-interactive: want jsx off local on, jsx=%v local=%v",
			IsCommandEnabledData(jsx, auth), IsCommandEnabledData(loc, auth))
	}
	t.Setenv("DISABLE_EXTRA_USAGE_COMMAND", "1")
	if IsCommandEnabledData(loc, auth) {
		t.Fatal("expected extra-usage off when DISABLE_EXTRA_USAGE_COMMAND")
	}
}

func TestIsCommandEnabledData_ContextByTypeAndAuth(t *testing.T) {
	auth := DefaultConsoleAPIAuth()
	jsx := types.Command{CommandBase: types.CommandBase{Name: "context"}, Type: "local-jsx"}
	loc := types.Command{CommandBase: types.CommandBase{Name: "context"}, Type: "local"}
	if !IsCommandEnabledData(jsx, auth) || IsCommandEnabledData(loc, auth) {
		t.Fatal("interactive session: want local-jsx only")
	}
	auth.IsNonInteractiveSession = true
	if IsCommandEnabledData(jsx, auth) || !IsCommandEnabledData(loc, auth) {
		t.Fatal("non-interactive: want local only")
	}
}

func TestIsCommandEnabledData_FeedbackBedrock(t *testing.T) {
	t.Setenv("CLAUDE_CODE_USE_BEDROCK", "1")
	cmd := types.Command{CommandBase: types.CommandBase{Name: "feedback"}, Type: "local-jsx"}
	if IsCommandEnabledData(cmd, DefaultConsoleAPIAuth()) {
		t.Fatal("expected feedback off on Bedrock")
	}
}

func TestIsCommandEnabledData_FeedbackDenyPolicy(t *testing.T) {
	t.Setenv("CLAUDE_CODE_USE_BEDROCK", "")
	auth := DefaultConsoleAPIAuth()
	auth.DenyProductFeedback = true
	cmd := types.Command{CommandBase: types.CommandBase{Name: "feedback"}, Type: "local-jsx"}
	if IsCommandEnabledData(cmd, auth) {
		t.Fatal("expected feedback off when DenyProductFeedback")
	}
}

func TestIsCommandEnabledData_RateLimitSubscriber(t *testing.T) {
	cmd := types.Command{CommandBase: types.CommandBase{Name: "rate-limit-options"}, Type: "local-jsx"}
	auth := DefaultConsoleAPIAuth()
	if IsCommandEnabledData(cmd, auth) {
		t.Fatal("expected rate-limit-options off for non-subscriber")
	}
	auth.IsClaudeAISubscriber = true
	if !IsCommandEnabledData(cmd, auth) {
		t.Fatal("expected rate-limit-options on for subscriber")
	}
}

func TestIsCommandEnabledData_RemoteEnvPolicy(t *testing.T) {
	cmd := types.Command{CommandBase: types.CommandBase{Name: "remote-env"}, Type: "local-jsx"}
	auth := DefaultConsoleAPIAuth()
	auth.IsClaudeAISubscriber = true
	if !IsCommandEnabledData(cmd, auth) {
		t.Fatal("expected remote-env when subscriber and policy allow")
	}
	auth.BlockRemoteSessions = true
	if IsCommandEnabledData(cmd, auth) {
		t.Fatal("expected remote-env off when BlockRemoteSessions")
	}
}

func TestIsCommandEnabledData_ThinkbackEnvShim(t *testing.T) {
	cmd := types.Command{CommandBase: types.CommandBase{Name: "think-back"}, Type: "local-jsx"}
	if IsCommandEnabledData(cmd, DefaultConsoleAPIAuth()) {
		t.Fatal("expected think-back off without shim")
	}
	t.Setenv("CLAUDE_CODE_GO_THINKBACK", "1")
	if !IsCommandEnabledData(cmd, DefaultConsoleAPIAuth()) {
		t.Fatal("expected think-back on with CLAUDE_CODE_GO_THINKBACK")
	}
}

func TestIsCommandEnabledData_KeybindingCustomizationEnv(t *testing.T) {
	cmd := types.Command{CommandBase: types.CommandBase{Name: "keybindings-help"}, Type: "prompt"}
	if IsCommandEnabledData(cmd, DefaultConsoleAPIAuth()) {
		t.Fatal("expected keybindings-help off without ant or env")
	}
	t.Setenv("CLAUDE_CODE_GO_KEYBINDING_CUSTOMIZATION", "1")
	if !IsCommandEnabledData(cmd, DefaultConsoleAPIAuth()) {
		t.Fatal("expected keybindings-help on with env shim")
	}
}

func TestIsCommandEnabledData_InstallGitHubAppDisabled(t *testing.T) {
	t.Setenv("DISABLE_INSTALL_GITHUB_APP_COMMAND", "1")
	cmd := types.Command{CommandBase: types.CommandBase{Name: "install-github-app"}, Type: "local-jsx"}
	if IsCommandEnabledData(cmd, DefaultConsoleAPIAuth()) {
		t.Fatal("expected install-github-app off")
	}
}

func TestIsCommandEnabledData_PrivacySettingsConsumer(t *testing.T) {
	cmd := types.Command{CommandBase: types.CommandBase{Name: "privacy-settings"}, Type: "local-jsx"}
	auth := DefaultConsoleAPIAuth()
	if IsCommandEnabledData(cmd, auth) {
		t.Fatal("expected privacy-settings off without consumer subscriber")
	}
	auth.IsConsumerSubscriber = true
	if !IsCommandEnabledData(cmd, auth) {
		t.Fatal("expected privacy-settings on for consumer subscriber")
	}
}

func TestIsCommandEnabledData_UltrareviewEnvShim(t *testing.T) {
	cmd := types.Command{CommandBase: types.CommandBase{Name: "ultrareview"}, Type: "local-jsx"}
	if IsCommandEnabledData(cmd, DefaultConsoleAPIAuth()) {
		t.Fatal("expected ultrareview off without shim")
	}
	t.Setenv("CLAUDE_CODE_GO_ULTRAREVIEW", "1")
	if !IsCommandEnabledData(cmd, DefaultConsoleAPIAuth()) {
		t.Fatal("expected ultrareview on with CLAUDE_CODE_GO_ULTRAREVIEW")
	}
}

func TestIsCommandEnabledData_DreamRemoteMemory(t *testing.T) {
	t.Setenv("CLAUDE_CODE_DISABLE_AUTO_MEMORY", "")
	t.Setenv("CLAUDE_CODE_SIMPLE", "")
	t.Setenv("CLAUDE_CODE_REMOTE", "1")
	t.Setenv("CLAUDE_CODE_REMOTE_MEMORY_DIR", "")
	cmd := types.Command{CommandBase: types.CommandBase{Name: "dream"}, Type: "prompt"}
	if IsCommandEnabledData(cmd, DefaultConsoleAPIAuth()) {
		t.Fatal("expected dream off when remote without CLAUDE_CODE_REMOTE_MEMORY_DIR")
	}
	t.Setenv("CLAUDE_CODE_REMOTE_MEMORY_DIR", "/tmp/mem")
	if !IsCommandEnabledData(cmd, DefaultConsoleAPIAuth()) {
		t.Fatal("expected dream on when remote memory dir set")
	}
}
