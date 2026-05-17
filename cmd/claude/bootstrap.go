package main

import (
	"context"
	"encoding/json"
	"os"
	"strconv"

	"goc/mcp"
	"goc/tools"
	"goc/tools/toolexecution"
)

// BootstrapConfig holds CLI-parsed values that map to module-level bootstrap state.
// Mirrors TS bootstrap/state.ts setters.
type BootstrapConfig struct {
	Print             bool
	Continue          bool
	Resume            string
	ForkSession       bool
	SessionID         string
	SessionName       string
	OutputFormat      string
	InputFormat       string
	MaxTurns          int
	Thinking          string
	AllowedTools      []string
	DisallowedTools   []string
	Tools             []string
	MCPConfig         []string
	AddDir            []string
	Worktree          string
	PluginDir         []string
}

// Bootstrap state mirrors TS bootstrap/state.ts module-level singletons.
var (
	bootstrapIsInteractive  = true
	bootstrapSessionID      string
	bootstrapIsPersisted    = true
	bootstrapOutputFormat   = "text"
	bootstrapMaxTurns       = 0
)

// ApplyBootstrapState applies CLI flags to bootstrap state.
func ApplyBootstrapState(cfg BootstrapConfig) {
	if cfg.Print {
		bootstrapIsInteractive = false
	}
	if cfg.SessionID != "" {
		bootstrapSessionID = cfg.SessionID
		os.Setenv("CLAUDE_SESSION_ID", cfg.SessionID)
	}
	if cfg.MaxTurns > 0 {
		bootstrapMaxTurns = cfg.MaxTurns
		os.Setenv("CLAUDE_CODE_MAX_TURNS", strconv.Itoa(cfg.MaxTurns))
	}
	if cfg.OutputFormat != "" {
		bootstrapOutputFormat = cfg.OutputFormat
	}
	if cfg.Thinking != "" {
		os.Setenv("CLAUDE_CODE_THINKING", cfg.Thinking)
	}
	// Set allowed/disallowed tools
	if len(cfg.AllowedTools) > 0 {
		for _, t := range cfg.AllowedTools {
			existing := os.Getenv("CLAUDE_CODE_ALLOWED_TOOLS")
			if existing != "" {
				os.Setenv("CLAUDE_CODE_ALLOWED_TOOLS", existing+","+t)
			} else {
				os.Setenv("CLAUDE_CODE_ALLOWED_TOOLS", t)
			}
		}
	}
	// Set explicit tools list
	if len(cfg.Tools) > 0 {
		for _, t := range cfg.Tools {
			existing := os.Getenv("CLAUDE_CODE_TOOLS")
			if existing != "" {
				os.Setenv("CLAUDE_CODE_TOOLS", existing+","+t)
			} else {
				os.Setenv("CLAUDE_CODE_TOOLS", t)
			}
		}
	}
	// Additional directories
	if len(cfg.AddDir) > 0 {
		for _, dir := range cfg.AddDir {
			existing := os.Getenv("CLAUDE_CODE_ADDITIONAL_DIRECTORIES")
			if existing != "" {
				os.Setenv("CLAUDE_CODE_ADDITIONAL_DIRECTORIES", existing+":"+dir)
			} else {
				os.Setenv("CLAUDE_CODE_ADDITIONAL_DIRECTORIES", dir)
			}
		}
	}
	// Plugin directories
	if len(cfg.PluginDir) > 0 {
		for _, dir := range cfg.PluginDir {
			existing := os.Getenv("CLAUDE_CODE_PLUGIN_DIR")
			if existing != "" {
				os.Setenv("CLAUDE_CODE_PLUGIN_DIR", existing+":"+dir)
			} else {
				os.Setenv("CLAUDE_CODE_PLUGIN_DIR", dir)
			}
		}
	}
}

// IsInteractive returns whether this is an interactive session.
func IsInteractive() bool {
	return bootstrapIsInteractive
}

// GetSessionID returns the session ID from bootstrap state.
func GetSessionID() string {
	if bootstrapSessionID != "" {
		return bootstrapSessionID
	}
	return os.Getenv("CLAUDE_SESSION_ID")
}

// IsSessionPersistenceEnabled returns whether sessions are persisted to disk.
func IsSessionPersistenceEnabled() bool {
	return bootstrapIsPersisted
}

// mcpConnMgr is the global MCP connection manager for the CLI process.
var mcpConnMgr *mcp.ConnectionManager

// cronScheduler is the global cron job scheduler for the CLI process.
var cronScheduler *tools.CronScheduler

// InitMCP initializes the MCP connection manager and wires it into the tool execution pipeline.
func InitMCP(ctx context.Context) error {
	clientMgr := mcp.NewClientManager()
	mcpConnMgr = mcp.NewConnectionManager(clientMgr)

	// Wire MCP tool dispatcher into the tool execution pipeline.
	// When RunToolUseChan sees an mcp__* tool, it calls this dispatcher.
	toolexecution.MCPToolDispatcher = func(dctx context.Context, toolName string, input json.RawMessage) (string, error) {
		return tools.RunMcpTool(dctx, toolName, input)
	}

	// Set the MCP executor's actual implementation.
	tools.DefaultMCPToolExecutor.ExecuteMCPToolFunc = func(dctx context.Context, fullToolName string, args map[string]interface{}) (string, error) {
		return mcpConnMgr.ExecuteMCPTool(dctx, fullToolName, args)
	}

	// Load MCP configs from --mcp-config CLI flag (JSON arrays).
	for _, raw := range flagMCPConfig {
		if err := loadMCPConfigFromJSON(ctx, mcpConnMgr, raw); err != nil {
			return err
		}
	}

		// Load MCP configs from .mcp.json (unless --strict-mcp-config is set)
		if !flagStrictMCPConfig {
			cwd, _ := os.Getwd()
			if cfg, err := mcp.LoadMcpJsonConfig(cwd); err == nil && cfg != nil {
				servers, err := mcp.ParseMcpServersFromJSON(cfg.McpServers)
				if err == nil {
					for name, serverCfg := range servers {
						mcpConnMgr.AddServer(name, mcp.ScopedMcpServerConfig{
							Config: serverCfg,
							Scope:  mcp.ScopeProject,
						})
					}
				}
			}
		}

	// Connect to all configured servers.
	return mcpConnMgr.StartAll(ctx)
}

// ShutdownMCP disconnects all MCP servers.
func ShutdownMCP() {
	if mcpConnMgr != nil {
		mcpConnMgr.Shutdown()
	}
}

// StartCronScheduler creates and starts the cron scheduler for the given project root.
// The onFire callback is called when a cron job fires.
func StartCronScheduler(ctx context.Context, projectRoot string, onFire func(context.Context, string)) {
	if cronScheduler != nil {
		cronScheduler.Stop()
	}
	cronScheduler = tools.NewCronScheduler(projectRoot, onFire)
	cronScheduler.Start(ctx)
}

// StopCronScheduler stops the cron scheduler if running.
func StopCronScheduler() {
	if cronScheduler != nil {
		cronScheduler.Stop()
		cronScheduler = nil
	}
}

// loadMCPConfigFromJSON parses a JSON MCP server configuration and adds it.
func loadMCPConfigFromJSON(ctx context.Context, cm *mcp.ConnectionManager, raw string) error {
	var wrapper struct {
		McpServers map[string]json.RawMessage `json:"mcpServers"`
	}
	if err := json.Unmarshal([]byte(raw), &wrapper); err != nil {
		// Try parsing as flat server config map directly.
		var flat map[string]json.RawMessage
		if err2 := json.Unmarshal([]byte(raw), &flat); err2 != nil {
			return err
		}
		wrapper.McpServers = flat
	}
	servers, err := mcp.ParseMcpServersFromJSON(wrapper.McpServers)
	if err != nil {
		return err
	}
	for name, cfg := range servers {
		cm.AddServer(name, mcp.ScopedMcpServerConfig{
			Config: cfg,
			Scope:  mcp.ScopeDynamic,
		})
	}
	return nil
}
