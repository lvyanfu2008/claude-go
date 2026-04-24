package pui

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"goc/agents/builtin"
	"goc/tools/skilltools"
	"goc/commands"
	processuserinput "goc/conversation-runtime/process-user-input"
	"goc/gou/conversation"
	"goc/mcpcommands"
	"goc/modelenv"
	"goc/tools/toolpool"
	"goc/tscontext"
	"goc/types"
)

// DefaultMainLoopModelForDemo returns the model id used when DemoConfig.MainLoopModel is empty
// and neither TS bridge nor env ([modelenv.LookupKeys]) supplies one (see [BuildDemoParams]).
func DefaultMainLoopModelForDemo() string {
	return modelenv.DefaultMainLoopModelID
}

// SlashGated is true when the trimmed line starts with "/".
// Callers that inject [ProcessUserInputParams.ProcessSlashCommand] (e.g. gou-demo with [NewSlashResolveProcessSlashCommand])
// should not gate on this — [processuserinput.ProcessUserInput] runs the slash handler instead of slashprepare populating [processuserinput.ProcessUserInputBaseResult.Execution].
func SlashGated(line string) bool {
	return strings.HasPrefix(strings.TrimSpace(line), "/")
}

func isEnvOn(s string) bool {
	v := strings.TrimSpace(strings.ToLower(s))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

// DemoConfig holds TUI-level options for building [processuserinput.ProcessUserInputParams].
// JSON tags match TS-facing names where applicable (mainLoopModel, uuid).
type DemoConfig struct {
	MainLoopModel   string  `json:"mainLoopModel,omitempty"`
	UserMessageUUID *string `json:"uuid,omitempty"`
	// SkipCommands when true leaves Commands empty (unit tests). When false, fills from
	// commands.GetCommands (TS getCommands: includes session dynamic skills).
	SkipCommands bool `json:"-"`
	// SessionID substitutes ${CLAUDE_SESSION_ID} in disk skills; empty defaults to "gou-demo".
	SessionID string `json:"-"`
	// MCPCommands optional appState.mcp.commands slice (prompt tools with loadedFrom mcp for merge tests / future bridge).
	MCPCommands []types.Command `json:"-"`
	// MCPCommandsJSONPath optional path to JSON array of types.Command (scheme-2 R0/R1). When set, overrides
	// GOU_DEMO_MCP_COMMANDS_JSON for this build. Loaded entries are merged after MCPCommands (uniqBy name, cfg wins on duplicate).
	MCPCommandsJSONPath string `json:"-"`
	// Language optional response language (TS settings.language); gou-demo fills from
	// settings merge + CLAUDE_CODE_LANGUAGE (see gou-demo main / apiparity.GouDemo).
	Language string `json:"-"`
	// UseEmbeddedToolsAPI when true (or env GOU_DEMO_USE_EMBEDDED_TOOLS_API=1) builds Options.Tools from
	// the Go tool wire via toolpool.GetTools + AssembleToolPool (TS getTools + assembleToolPool semantics).
	UseEmbeddedToolsAPI bool `json:"-"`
	// MCPToolsJSONPath optional path to JSON array of MCP tool defs (see mcpcommands.LoadToolsFromPath).
	// Merged after env GOU_DEMO_MCP_TOOLS_JSON when both set (cfg path tried first).
	MCPToolsJSONPath string `json:"-"`
	// TSContextBridge optional in-process snapshot. When set, commands
	// and tools are taken from the snapshot (then MCP file/env merged for commands) unless SkipCommands / embedded tools override.
	TSContextBridge *tscontext.Snapshot `json:"-"`
	// IsRemoteMode when true includes /session in GetCommands output (matches src/bootstrap getIsRemoteMode).
	IsRemoteMode bool `json:"-"`
	// PreExpansionInput optional; when non-nil, copied to [processuserinput.ProcessUserInputParams.PreExpansionInput] (TS preExpansionInput).
	PreExpansionInput *string `json:"-"`
	// PermissionMode optional; when non-nil, sets [processuserinput.ProcessUserInputParams.PermissionMode] (TS toolPermissionContext.mode).
	PermissionMode *types.PermissionMode `json:"-"`
	// ToolPermissionContext optional merged alwaysDeny/alwaysAsk (TS appState.toolPermissionContext); when set, filters Agent tool
	// listing in [toolpool.PatchAgentToolDescriptionWithPermission] (filterDeniedAgents parity).
	ToolPermissionContext *types.ToolPermissionContextData `json:"-"`
}

// BuildDemoParams builds params for gou-demo: prompt mode, skip attachments, minimal ToolUseContext.
func BuildDemoParams(line string, store *conversation.Store, cfg DemoConfig) (*processuserinput.ProcessUserInputParams, error) {
	trimmed := strings.TrimSpace(line)
	raw, err := json.Marshal(trimmed)
	if err != nil {
		return nil, err
	}
	msgs := append([]types.Message(nil), store.Messages...)
	var model string
	if m := strings.TrimSpace(cfg.MainLoopModel); m != "" {
		model = m
	} else {
		// Live process env (including values merged from ~/.claude/settings.json and project
		// .claude/settings.go.json by settingsfile) beats TS bridge snapshot for API/model line parity.
		if envModel := modelenv.FirstNonEmpty(); envModel != "" {
			model = envModel
		} else if cfg.TSContextBridge != nil {
			model = strings.TrimSpace(cfg.TSContextBridge.MainLoopModel)
		}
		if model == "" {
			model = modelenv.DefaultMainLoopModelID
		}
	}
	// When UUID is unset, leave nil so process-user-input newUserMessage uses
	// randomUUID() — TS createUserMessage default is crypto.randomUUID().
	var uuidPtr *string
	if cfg.UserMessageUUID != nil && strings.TrimSpace(*cfg.UserMessageUUID) != "" {
		u := strings.TrimSpace(*cfg.UserMessageUUID)
		uuidPtr = &u
	}

	skipAtt := true
	var loaded []types.Command
	if cfg.TSContextBridge != nil && len(bytes.TrimSpace(cfg.TSContextBridge.Commands)) > 2 {
		if err := json.Unmarshal(cfg.TSContextBridge.Commands, &loaded); err != nil {
			return nil, fmt.Errorf("TSContextBridge commands: %w", err)
		}
	} else if !cfg.SkipCommands {
		cwd, errWd := os.Getwd()
		if errWd != nil {
			cwd = "."
		}
		var lc []types.Command
		var errLC error
		if cfg.IsRemoteMode {
			auth := commands.DefaultConsoleAPIAuth()
			auth.IsRemoteMode = true
			lc, errLC = commands.GetCommands(context.Background(), cwd, commands.DefaultLoadOptions(), auth)
		} else {
			lc, errLC = commands.GetCommandsWithDefaults(context.Background(), cwd)
		}
		if errLC == nil {
			loaded = lc
		}
	}
	mcp := cfg.MCPCommands
	path := strings.TrimSpace(cfg.MCPCommandsJSONPath)
	if path == "" {
		path = strings.TrimSpace(os.Getenv(mcpcommands.EnvCommandsJSONPath))
	}
	if path != "" {
		fileMCP, errMCP := mcpcommands.LoadFromPath(path)
		if errMCP != nil {
			return nil, errMCP
		}
		mcp = commands.MergeCommandsUniqByName(mcp, fileMCP)
	}
	merged := commands.MergeCommandsForSkillTool(loaded, mcp)
	cmds := merged
	var listing []types.Command
	if cfg.TSContextBridge != nil && len(bytes.TrimSpace(cfg.TSContextBridge.SkillToolCommands)) > 2 {
		var tsSkills []types.Command
		if err := json.Unmarshal(cfg.TSContextBridge.SkillToolCommands, &tsSkills); err != nil {
			return nil, fmt.Errorf("TSContextBridge skillToolCommands: %w", err)
		}
		listing = commands.SkillListingFromTSPresliced(tsSkills, mcp, commands.FeatureMcpSkills())
	} else {
		listing = commands.SkillListingCommandsForAPI(loaded, mcp, commands.FeatureMcpSkills())
	}
	var toolsRaw json.RawMessage
	var errTools error
	useExport := cfg.UseEmbeddedToolsAPI || isEnvOn(os.Getenv("GOU_DEMO_USE_EMBEDDED_TOOLS_API"))
	if useExport {
		permCtx := types.EmptyToolPermissionContextData()
		if cfg.ToolPermissionContext != nil {
			permCtx = *cfg.ToolPermissionContext
			types.NormalizeToolPermissionContextData(&permCtx)
		}
		var mcpToolSpecs []types.ToolSpec
		toolsPath := strings.TrimSpace(cfg.MCPToolsJSONPath)
		if toolsPath == "" {
			toolsPath = strings.TrimSpace(os.Getenv(mcpcommands.EnvToolsJSONPath))
		}
		if toolsPath != "" {
			loadedTools, errT := mcpcommands.LoadToolsFromPath(toolsPath)
			if errT != nil {
				return nil, errT
			}
			mcpToolSpecs = loadedTools
		}
		assembled, errA := toolpool.AssembleToolPoolFromGoWire(permCtx, mcpToolSpecs)
		if errA != nil {
			return nil, errA
		}
		assembled = toolpool.PatchAgentToolDescriptionWithPermission(assembled, builtin.GetBuiltInAgents(builtin.ConfigFromEnv(), builtin.GuideContext{}), permCtx)
		toolSchemaOpts := toolpool.DefaultToolToAPISchemaOptionsFromEnv()
		toolSchemaOpts.Model = model
		toolsRaw, errTools = toolpool.MarshalToolsAPIDocumentDefinitionsWithOptions(assembled, toolSchemaOpts)
	} else if cfg.TSContextBridge != nil && len(bytes.TrimSpace(cfg.TSContextBridge.Tools)) > 2 {
		toolsRaw = append(json.RawMessage(nil), cfg.TSContextBridge.Tools...)
	} else {
		toolsRaw, errTools = skilltools.GouDemoParityToolsJSON()
	}
	if errTools != nil {
		return nil, errTools
	}
	rc := &types.ProcessUserInputContextData{
		ToolUseContext: types.ToolUseContext{
			Options: types.ToolUseContextOptionsData{
				Commands:      cmds,
				MainLoopModel: model,
				Tools:         toolsRaw,
			},
			Messages: msgs,
		},
	}
	perm := types.PermissionDefault
	if cfg.PermissionMode != nil && *cfg.PermissionMode != "" {
		perm = *cfg.PermissionMode
	}
	if cfg.ToolPermissionContext != nil {
		tpc := *cfg.ToolPermissionContext
		types.NormalizeToolPermissionContextData(&tpc)
		rc.ToolPermissionContext = &tpc
	}
	out := &processuserinput.ProcessUserInputParams{
		Input:                raw,
		Mode:                 types.PromptInputModePrompt,
		Messages:             msgs,
		UUID:                 uuidPtr,
		PermissionMode:       perm,
		Commands:             cmds,
		SkillListingCommands: listing,
		RuntimeContext:       rc,
		SkipAttachments:      &skipAtt,
	}
	if cfg.PreExpansionInput != nil {
		s := *cfg.PreExpansionInput
		out.PreExpansionInput = &s
	}
	return out, nil
}
