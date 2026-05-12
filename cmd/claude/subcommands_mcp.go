package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"

	"goc/mcp"
)

// ---- mcp serve ----

var mcpServeDebug bool
var mcpServeVerbose bool

var mcpServeCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the Claude Code MCP server",
	Long:  "Start a local MCP server that exposes Claude Code tools to other MCP clients.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if mcpServeVerbose {
			fmt.Fprintf(os.Stderr, "[mcp serve] starting MCP server (debug=%v)...\n", mcpServeDebug)
		}

		// MCP serve requires a full Claude Code session context.
		// In Go, this is a stub that points users to the TS CLI.
		fmt.Println("MCP serve mode is not yet implemented in Go.")
		fmt.Println("Use the TS CLI for MCP server functionality: claude mcp serve")
		return nil
	},
}

// ---- mcp add-from-claude-desktop ----

var mcpAddFromCDScope string

var mcpAddFromClaudeDesktopCmd = &cobra.Command{
	Use:   "add-from-claude-desktop",
	Short: "Import MCP servers from Claude Desktop config",
	Long: `Import MCP server configurations from the Claude Desktop app.

On macOS, reads from ~/Library/Application Support/Claude/claude_desktop_config.json.
On other platforms, looks for the Claude Desktop config file.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		desktopPath := claudeDesktopConfigPath()
		if desktopPath == "" {
			return fmt.Errorf("Claude Desktop config not found on this platform")
		}

		data, err := os.ReadFile(desktopPath)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("Claude Desktop config not found at %s", desktopPath)
			}
			return fmt.Errorf("read Claude Desktop config: %w", err)
		}

		var desktopCfg struct {
			McpServers map[string]json.RawMessage `json:"mcpServers"`
		}
		if err := json.Unmarshal(data, &desktopCfg); err != nil {
			return fmt.Errorf("parse Claude Desktop config: %w", err)
		}

		if len(desktopCfg.McpServers) == 0 {
			fmt.Println("No MCP servers found in Claude Desktop config.")
			return nil
		}

		cfg, err := loadOrCreateMcpJson()
		if err != nil {
			return err
		}

		imported := 0
		for name, raw := range desktopCfg.McpServers {
			normalized := mcp.NormalizeMcpServerName(name)

			if _, err := mcp.ParseMcpServerConfig(raw); err != nil {
				fmt.Fprintf(os.Stderr, "warning: skipping %q: invalid config: %v\n", name, err)
				continue
			}

			if _, exists := cfg.McpServers[normalized]; exists {
				fmt.Printf("  Skipping %q (already configured)\n", normalized)
				continue
			}

			cfg.McpServers[normalized] = raw
			imported++
			fmt.Printf("  Imported %q\n", normalized)
		}

		if imported == 0 {
			fmt.Println("No new servers to import.")
			return nil
		}

		if err := saveMcpJson(cfg); err != nil {
			return err
		}

		fmt.Printf("\nImported %d server(s) from Claude Desktop to .mcp.json\n", imported)
		return nil
	},
}

// ---- mcp reset-project-choices ----

var mcpResetProjectChoicesCmd = &cobra.Command{
	Use:   "reset-project-choices",
	Short: "Reset approved/rejected project-scoped MCP servers",
	Long: `Clear any approved or rejected server choices stored in the
project's .mcp.json, requiring re-approval on next connection.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd := getCWD()
		cfg, err := mcp.LoadMcpJsonConfig(cwd)
		if err != nil {
			return fmt.Errorf("read .mcp.json: %w", err)
		}

		if cfg == nil || len(cfg.McpServers) == 0 {
			fmt.Println("No MCP servers configured; nothing to reset.")
			return nil
		}

		fmt.Println("Project MCP server configurations in .mcp.json:")
		count := 0
		for name := range cfg.McpServers {
			fmt.Printf("  %s\n", name)
			count++
		}
		fmt.Printf("\n%d server(s) listed. Use 'claude mcp remove <name>' to remove specific servers.\n", count)
		fmt.Println("(Server approval/rejection tracking is not yet implemented in Go)")
		return nil
	},
}

// claudeDesktopConfigPath returns the path to the Claude Desktop config file.
func claudeDesktopConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "Claude", "claude_desktop_config.json")
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			return ""
		}
		return filepath.Join(appData, "Claude", "claude_desktop_config.json")
	default:
		// Linux and others — multiple possible locations
		configDir := os.Getenv("XDG_CONFIG_HOME")
		if configDir == "" {
			configDir = filepath.Join(home, ".config")
		}
		return filepath.Join(configDir, "Claude", "claude_desktop_config.json")
	}
}

func init() {
	mcpServeCmd.Flags().BoolVar(&mcpServeDebug, "debug", false, "Enable debug logging")
	mcpServeCmd.Flags().BoolVar(&mcpServeVerbose, "verbose", false, "Verbose output")
	mcpAddFromClaudeDesktopCmd.Flags().StringVar(&mcpAddFromCDScope, "scope", "project", "Config scope")

	mcpCmd.AddCommand(mcpServeCmd)
	mcpCmd.AddCommand(mcpAddFromClaudeDesktopCmd)
	mcpCmd.AddCommand(mcpResetProjectChoicesCmd)
}
