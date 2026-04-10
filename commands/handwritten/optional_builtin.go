package handwritten

import (
	"goc/commands/featuregates"
	"goc/types"
)

// Optional builtins after /vim and before /think-back (src/commands.ts COMMANDS spread order).
func optionalBuiltinAfterVim() []types.Command {
	var out []types.Command
	if featuregates.Feature("CCR_REMOTE_SETUP") {
		out = append(out, cmdWebSetup())
	}
	if featuregates.Feature("FORK_SUBAGENT") {
		out = append(out, cmdFork())
	}
	if featuregates.Feature("BUDDY") {
		out = append(out, cmdBuddy())
	}
	if featuregates.Feature("PROACTIVE") || featuregates.Feature("KAIROS") {
		out = append(out, cmdProactive())
	}
	if featuregates.Feature("KAIROS") || featuregates.Feature("KAIROS_BRIEF") {
		out = append(out, cmdBrief())
	}
	if featuregates.Feature("KAIROS") {
		out = append(out, cmdAssistant())
	}
	if featuregates.Feature("BRIDGE_MODE") {
		out = append(out, cmdBridge())
	}
	if featuregates.Feature("DAEMON") && featuregates.Feature("BRIDGE_MODE") {
		out = append(out, cmdRemoteControlServer())
	}
	if featuregates.Feature("VOICE_MODE") {
		out = append(out, cmdVoice())
	}
	return out
}

func optionalBuiltinPeers() []types.Command {
	if !featuregates.Feature("UDS_INBOX") {
		return nil
	}
	return []types.Command{cmdPeers()}
}

func optionalBuiltinWorkflows() []types.Command {
	if !featuregates.Feature("WORKFLOW_SCRIPTS") {
		return nil
	}
	return []types.Command{cmdWorkflows()}
}

func optionalBuiltinTorch() []types.Command {
	if !featuregates.Feature("TORCH") {
		return nil
	}
	return []types.Command{cmdTorch()}
}

func cmdWebSetup() types.Command {
	return types.Command{
		CommandBase: types.CommandBase{
			Name:         "web-setup",
			Description:  "Setup Claude Code on the web (requires connecting your GitHub account)",
			Availability: []types.CommandAvailability{types.CommandAvailabilityClaudeAI},
		},
		Type: "local-jsx",
	}
}

func cmdFork() types.Command {
	return types.Command{
		CommandBase: types.CommandBase{
			Name:        "fork",
			Description: "Fork subagent (metadata only in Go listing)",
		},
		Type: "local-jsx",
	}
}

func cmdBuddy() types.Command {
	return types.Command{
		CommandBase: types.CommandBase{
			Name:         "buddy",
			Description:  "Hatch a coding companion · pet, off",
			ArgumentHint: ptrStr("[pet|off]"),
			Immediate:    ptrBool(true),
		},
		Type: "local-jsx",
	}
}

func cmdProactive() types.Command {
	return types.Command{
		CommandBase: types.CommandBase{
			Name:        "proactive",
			Description: "Proactive assistant (metadata only in Go listing)",
		},
		Type: "local-jsx",
	}
}

func cmdBrief() types.Command {
	return types.Command{
		CommandBase: types.CommandBase{
			Name:        "brief",
			Description: "Toggle brief-only mode",
			Immediate:   ptrBool(true),
		},
		Type: "local-jsx",
	}
}

func cmdAssistant() types.Command {
	return types.Command{
		CommandBase: types.CommandBase{
			Name:        "assistant",
			Description: "Assistant install wizard (metadata only in Go listing)",
		},
		Type: "local-jsx",
	}
}

func cmdBridge() types.Command {
	return types.Command{
		CommandBase: types.CommandBase{
			Name:         "remote-control",
			Description:  "Connect this terminal for remote-control sessions",
			Aliases:      strSlice("rc"),
			ArgumentHint: ptrStr("[name]"),
			Immediate:    ptrBool(true),
		},
		Type: "local-jsx",
	}
}

func cmdRemoteControlServer() types.Command {
	return types.Command{
		CommandBase: types.CommandBase{
			Name:        "remote-control-server",
			Description: "Remote control server (DAEMON + BRIDGE_MODE; metadata only in Go listing)",
		},
		Type: "local-jsx",
	}
}

func cmdVoice() types.Command {
	return types.Command{
		CommandBase: types.CommandBase{
			Name:         "voice",
			Description:  "Toggle voice mode",
			Availability: []types.CommandAvailability{types.CommandAvailabilityClaudeAI},
		},
		Type:                   "local",
		SupportsNonInteractive: ptrBool(false),
	}
}

func cmdPeers() types.Command {
	return types.Command{
		CommandBase: types.CommandBase{
			Name:        "peers",
			Description: "UDS inbox peers (metadata only in Go listing)",
		},
		Type: "local-jsx",
	}
}

func cmdWorkflows() types.Command {
	return types.Command{
		CommandBase: types.CommandBase{
			Name:        "workflows",
			Description: "Workflow scripts (WORKFLOW_SCRIPTS; metadata only in Go listing)",
		},
		Type: "local-jsx",
	}
}

func cmdTorch() types.Command {
	return types.Command{
		CommandBase: types.CommandBase{
			Name:        "torch",
			Description: "Torch (TORCH feature; metadata only in Go listing)",
		},
		Type: "local-jsx",
	}
}
