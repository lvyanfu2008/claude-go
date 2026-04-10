package commands

import (
	"os"
	"strings"

	"goc/commands/featuregates"
	"goc/types"
)

func is3PProviderEnv() bool {
	return IsEnvTruthy("CLAUDE_CODE_USE_BEDROCK") ||
		IsEnvTruthy("CLAUDE_CODE_USE_VERTEX") ||
		IsEnvTruthy("CLAUDE_CODE_USE_FOUNDRY")
}

func goDreamSkillEnabled() bool {
	if IsEnvTruthy("CLAUDE_CODE_DISABLE_AUTO_MEMORY") {
		return false
	}
	if IsEnvTruthy("CLAUDE_CODE_SIMPLE") {
		return false
	}
	if IsEnvTruthy("CLAUDE_CODE_REMOTE") && strings.TrimSpace(os.Getenv("CLAUDE_CODE_REMOTE_MEMORY_DIR")) == "" {
		return false
	}
	return true
}

func goKeybindingCustomizationEnabled() bool {
	if IsEnvTruthy("CLAUDE_CODE_GO_KEYBINDING_CUSTOMIZATION") {
		return true
	}
	return featuregates.UserTypeAnt()
}

func goThinkbackEnabled() bool {
	return IsEnvTruthy("CLAUDE_CODE_GO_THINKBACK")
}

func goUltrareviewEnabled() bool {
	return IsEnvTruthy("CLAUDE_CODE_GO_ULTRAREVIEW")
}

func extraUsageGateOK(auth GetCommandsAuth) bool {
	if IsEnvTruthy("DISABLE_EXTRA_USAGE_COMMAND") {
		return false
	}
	return auth.ExtraUsageAllowed
}

// IsCommandEnabledData mirrors src/types/command.ts isCommandEnabled for static / JSON-backed commands.
// Embedded manifests do not carry runtime isEnabled() — default true (same as TS when field absent).
// GrowthBook-only gates (think-back, ultrareview) use CLAUDE_CODE_GO_* env shims when the host has no GB.
func IsCommandEnabledData(cmd types.Command, auth GetCommandsAuth) bool {
	switch cmd.Name {
	case "session":
		return auth.IsRemoteMode
	case "fast":
		return !IsEnvTruthy("CLAUDE_CODE_DISABLE_FAST_MODE")
	case "install-github-app":
		return !IsEnvTruthy("DISABLE_INSTALL_GITHUB_APP_COMMAND")
	case "doctor":
		return !IsEnvTruthy("DISABLE_DOCTOR_COMMAND")
	case "tag":
		return featuregates.UserTypeAnt()
	case "rate-limit-options":
		return auth.IsClaudeAISubscriber
	case "remote-env":
		return auth.IsClaudeAISubscriber && !auth.BlockRemoteSessions
	case "privacy-settings":
		return auth.IsConsumerSubscriber
	case "think-back", "thinkback-play":
		return goThinkbackEnabled()
	case "feedback":
		if is3PProviderEnv() {
			return false
		}
		if IsEnvTruthy("DISABLE_FEEDBACK_COMMAND") || IsEnvTruthy("DISABLE_BUG_COMMAND") {
			return false
		}
		if IsEnvTruthy("CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC") {
			return false
		}
		if featuregates.UserTypeAnt() {
			return false
		}
		if auth.DenyProductFeedback {
			return false
		}
		return true
	case "ultrareview":
		return goUltrareviewEnabled()
	case "keybindings":
		if cmd.Type != "local" {
			return true
		}
		return goKeybindingCustomizationEnabled()
	case "keybindings-help":
		return goKeybindingCustomizationEnabled()
	case "dream", "remember":
		if cmd.Type != "prompt" {
			return true
		}
		return goDreamSkillEnabled()
	case "extra-usage":
		if !extraUsageGateOK(auth) {
			return false
		}
		if cmd.Type == "local-jsx" {
			return !auth.IsNonInteractiveSession
		}
		if cmd.Type == "local" {
			return auth.IsNonInteractiveSession
		}
		return true
	case "context":
		if cmd.Type == "local-jsx" {
			return !auth.IsNonInteractiveSession
		}
		if cmd.Type == "local" {
			return auth.IsNonInteractiveSession
		}
		return true
	default:
		return true
	}
}
