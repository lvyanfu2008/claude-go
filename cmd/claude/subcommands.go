package main

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	"goc/mcp"
)

// ---- helpers ----

func getCWD() string {
	cwd, err := os.Getwd()
	if err != nil {
		return "unknown"
	}
	return cwd
}

// loadOrCreateMcpJson loads .mcp.json, returning an empty config if none exists.
func loadOrCreateMcpJson() (*mcp.McpJsonConfig, error) {
	cwd := getCWD()
	cfg, err := mcp.LoadMcpJsonConfig(cwd)
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		cfg = &mcp.McpJsonConfig{McpServers: make(map[string]json.RawMessage)}
	}
	return cfg, nil
}

// saveMcpJson is a thin wrapper around mcp.SaveMcpJsonConfig.
func saveMcpJson(cfg *mcp.McpJsonConfig) error {
	return mcp.SaveMcpJsonConfig(getCWD(), cfg)
}

// configToRawJSON serializes a McpServerConfig to its raw JSON representation for storage.
func configToRawJSON(cfg mcp.McpServerConfig) (json.RawMessage, error) {
	return json.Marshal(cfg)
}

// ---- mcp subcommand ----

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Manage MCP servers",
	Long:  "Add, remove, list, and configure MCP (Model Context Protocol) servers.",
}

var mcpAddTransport string
var mcpAddScope string
var mcpAddEnv []string
var mcpAddHeader []string

var mcpAddCmd = &cobra.Command{
	Use:   "add <name> <command-or-url> [args...]",
	Short: "Add an MCP server (stdio, SSE, or HTTP)",
	Long: `Add an MCP server to the project .mcp.json.

For stdio servers:    claude mcp add myserver -- node server.js --port 3000
For SSE/HTTP servers: claude mcp add myserver https://example.com/mcp`,
	Args: cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		commandOrURL := args[1]
		extraArgs := args[2:]

		cfg, err := loadOrCreateMcpJson()
		if err != nil {
			return err
		}

		normalizedName := mcp.NormalizeMcpServerName(name)

		var serverCfg mcp.McpServerConfig

		if strings.Contains(commandOrURL, "://") {
			if mcpAddTransport == "sse" || strings.HasPrefix(commandOrURL, "http") {
				headers := parseKeyValuePairs(mcpAddHeader)
				serverCfg = mcp.McpSSEServerConfig{
					Type:    "sse",
					URL:     commandOrURL,
					Headers: headers,
				}
			} else if mcpAddTransport == "ws" || strings.HasPrefix(commandOrURL, "ws") {
				headers := parseKeyValuePairs(mcpAddHeader)
				serverCfg = mcp.McpWebSocketServerConfig{
					Type:    "ws",
					URL:     commandOrURL,
					Headers: headers,
				}
			} else {
				headers := parseKeyValuePairs(mcpAddHeader)
				serverCfg = mcp.McpHTTPServerConfig{
					Type:    "http",
					URL:     commandOrURL,
					Headers: headers,
				}
			}
		} else {
			serverCfg = mcp.McpStdioServerConfig{
				Type:    "stdio",
				Command: commandOrURL,
				Args:    extraArgs,
				Env:     parseKeyValuePairs(mcpAddEnv),
			}
		}

		raw, err := configToRawJSON(serverCfg)
		if err != nil {
			return fmt.Errorf("serialize config: %w", err)
		}
		cfg.McpServers[normalizedName] = raw

		if err := saveMcpJson(cfg); err != nil {
			return err
		}

		fmt.Printf("Added MCP server %q (%s) to .mcp.json\n", normalizedName, commandOrURL)
		if mcpAddScope != "" {
			fmt.Printf("  Scope: %s\n", mcpAddScope)
		}
		return nil
	},
}

var mcpRemoveScope string

var mcpRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove an MCP server",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		normalizedName := mcp.NormalizeMcpServerName(name)

		cfg, err := loadOrCreateMcpJson()
		if err != nil {
			return err
		}

		if _, ok := cfg.McpServers[normalizedName]; !ok {
			return fmt.Errorf("server %q not found in .mcp.json", normalizedName)
		}

		delete(cfg.McpServers, normalizedName)

		if err := saveMcpJson(cfg); err != nil {
			return err
		}

		fmt.Printf("Removed MCP server %q from .mcp.json\n", normalizedName)
		return nil
	},
}

var mcpListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured MCP servers",
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd := getCWD()
		cfg, err := mcp.LoadMcpJsonConfig(cwd)
		if err != nil {
			return fmt.Errorf("read .mcp.json: %w", err)
		}

		fmt.Println("MCP servers in .mcp.json:")
		if cfg == nil || len(cfg.McpServers) == 0 {
			fmt.Println("  (no servers configured)")
			return nil
		}

		for name, raw := range cfg.McpServers {
			serverCfg, err := mcp.ParseMcpServerConfig(raw)
			if err != nil {
				fmt.Printf("  %s: (parse error: %v)\n", name, err)
				continue
			}
			desc := describeServerConfig(serverCfg)
			fmt.Printf("  %s: %s\n", name, desc)
		}
		return nil
	},
}

var mcpGetCmd = &cobra.Command{
	Use:   "get <name>",
	Short: "Show details of an MCP server",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		normalizedName := mcp.NormalizeMcpServerName(name)

		cfg, err := loadOrCreateMcpJson()
		if err != nil {
			return err
		}

		raw, ok := cfg.McpServers[normalizedName]
		if !ok {
			return fmt.Errorf("server %q not found in .mcp.json", normalizedName)
		}

		serverCfg, err := mcp.ParseMcpServerConfig(raw)
		if err != nil {
			return fmt.Errorf("parse config: %w", err)
		}

		fmt.Printf("MCP server: %s\n", normalizedName)
		fmt.Println(strings.Repeat("-", 40))
		pretty, _ := json.MarshalIndent(serverCfg, "", "  ")
		fmt.Println(string(pretty))
		return nil
	},
}

var mcpAddJSONScope string

var mcpAddJSONCmd = &cobra.Command{
	Use:   "add-json <name> <json-config>",
	Short: "Add an MCP server via JSON config string",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		configJSON := args[1]
		normalizedName := mcp.NormalizeMcpServerName(name)

		var raw json.RawMessage
		if err := json.Unmarshal([]byte(configJSON), &raw); err != nil {
			return fmt.Errorf("invalid JSON config: %w", err)
		}

		// Validate it parses as a server config
		if _, err := mcp.ParseMcpServerConfig(raw); err != nil {
			return fmt.Errorf("invalid MCP server config: %w", err)
		}

		cfg, err := loadOrCreateMcpJson()
		if err != nil {
			return err
		}

		cfg.McpServers[normalizedName] = raw

		if err := saveMcpJson(cfg); err != nil {
			return err
		}

		fmt.Printf("Added MCP server %q from JSON config to .mcp.json\n", normalizedName)
		return nil
	},
}

// describeServerConfig returns a short human-readable summary of a server config.
func describeServerConfig(cfg mcp.McpServerConfig) string {
	switch c := cfg.(type) {
	case mcp.McpStdioServerConfig:
		return fmt.Sprintf("stdio: %s %s", c.Command, strings.Join(c.Args, " "))
	case mcp.McpSSEServerConfig:
		return fmt.Sprintf("sse: %s", c.URL)
	case mcp.McpSSEIDEServerConfig:
		return fmt.Sprintf("sse-ide: %s (%s)", c.URL, c.IDEName)
	case mcp.McpHTTPServerConfig:
		return fmt.Sprintf("http: %s", c.URL)
	case mcp.McpWebSocketServerConfig:
		return fmt.Sprintf("ws: %s", c.URL)
	case mcp.McpWebSocketIDEServerConfig:
		return fmt.Sprintf("ws-ide: %s (%s)", c.URL, c.IDEName)
	case mcp.McpSdkServerConfig:
		return fmt.Sprintf("sdk: %s", c.Name)
	case mcp.McpClaudeAIProxyServerConfig:
		return fmt.Sprintf("claudeai-proxy: %s", c.ID)
	default:
		return "unknown type"
	}
}

// parseKeyValuePairs parses "key=value" strings into a map.
func parseKeyValuePairs(pairs []string) map[string]string {
	result := make(map[string]string, len(pairs))
	for _, p := range pairs {
		parts := strings.SplitN(p, "=", 2)
		if len(parts) == 2 {
			result[parts[0]] = parts[1]
		}
	}
	return result
}

// ---- doctor subcommand ----

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check installation health",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Claude Code Go - Health Check")
		fmt.Println("==============================")
		fmt.Printf("  OS:            %s/%s\n", runtime.GOOS, runtime.GOARCH)
		fmt.Printf("  Go version:    %s\n", runtime.Version())
		fmt.Printf("  CWD:           %s\n", getCWD())

		cwd := getCWD()
		if _, err := os.Stat(cwd + "/CLAUDE.md"); err == nil {
			fmt.Println("  CLAUDE.md:     found")
		} else {
			fmt.Println("  CLAUDE.md:     not found")
		}

		if _, err := os.Stat(cwd + "/.harness"); err == nil {
			fmt.Println("  .harness/:      found")
		} else {
			fmt.Println("  .harness/:      not found")
		}

		if _, err := os.Stat(cwd + "/.git"); err == nil {
			fmt.Println("  Git repo:      yes")
		} else {
			fmt.Println("  Git repo:      no")
		}

		if os.Getenv("ANTHROPIC_API_KEY") != "" || os.Getenv("ANTHROPIC_AUTH_TOKEN") != "" {
			fmt.Println("  API key:       set")
		} else {
			fmt.Println("  API key:       not set")
		}

		// Check .mcp.json
		if _, err := os.Stat(cwd + "/.mcp.json"); err == nil {
			fmt.Println("  .mcp.json:     found")
		} else {
			fmt.Println("  .mcp.json:     not found")
		}

		return nil
	},
}

// ---- auth subcommand ----

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage authentication",
}

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Log in to Anthropic",
	Long: `Authenticate with Anthropic to use Claude Code.

If ANTHROPIC_API_KEY is already set, no action is needed.
Otherwise, set it manually or use 'claude auth login' to open
the Anthropic Console for an API key.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if os.Getenv("ANTHROPIC_API_KEY") != "" {
			fmt.Println("Already authenticated (ANTHROPIC_API_KEY is set).")
			return nil
		}
		if os.Getenv("ANTHROPIC_AUTH_TOKEN") != "" {
			fmt.Println("Already authenticated (ANTHROPIC_AUTH_TOKEN is set).")
			return nil
		}

		fmt.Println("To authenticate, set your API key:")
		fmt.Println("  1. Visit https://console.anthropic.com/")
		fmt.Println("  2. Generate an API key")
		fmt.Println("  3. Run: export ANTHROPIC_API_KEY='your-key'")
		fmt.Println("")
		fmt.Println("Or add it to your shell profile (~/.zshrc, ~/.bashrc) for persistence.")
		return nil
	},
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show authentication status",
	RunE: func(cmd *cobra.Command, args []string) error {
		switch {
		case os.Getenv("ANTHROPIC_API_KEY") != "":
			key := os.Getenv("ANTHROPIC_API_KEY")
			masked := key[:4] + "..." + key[len(key)-4:]
			fmt.Printf("Authenticated (ANTHROPIC_API_KEY=%s)\n", masked)
		case os.Getenv("ANTHROPIC_AUTH_TOKEN") != "":
			fmt.Println("Authenticated (ANTHROPIC_AUTH_TOKEN is set)")
		default:
			fmt.Println("Not authenticated.")
			fmt.Println("Set ANTHROPIC_API_KEY or ANTHROPIC_AUTH_TOKEN to authenticate.")
		}
		return nil
	},
}

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Clear local authentication",
	RunE: func(cmd *cobra.Command, args []string) error {
		if os.Getenv("ANTHROPIC_API_KEY") == "" && os.Getenv("ANTHROPIC_AUTH_TOKEN") == "" {
			fmt.Println("Not currently authenticated.")
			return nil
		}
		fmt.Println("To log out, unset the following environment variables:")
		if os.Getenv("ANTHROPIC_API_KEY") != "" {
			fmt.Println("  unset ANTHROPIC_API_KEY")
		}
		if os.Getenv("ANTHROPIC_AUTH_TOKEN") != "" {
			fmt.Println("  unset ANTHROPIC_AUTH_TOKEN")
		}
		fmt.Println("")
		fmt.Println("You can remove them from your shell profile for a permanent logout.")
		return nil
	},
}

// ---- completion subcommand ----

var completionCmd = &cobra.Command{
	Use:       "completion [bash|zsh|fish|powershell]",
	Short:     "Generate shell completion script",
	Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return rootCmd.GenBashCompletion(os.Stdout)
		case "zsh":
			return rootCmd.GenZshCompletion(os.Stdout)
		case "fish":
			return rootCmd.GenFishCompletion(os.Stdout, true)
		case "powershell":
			return rootCmd.GenPowerShellCompletion(os.Stdout)
		}
		return nil
	},
}

// ---- init registrations ----

func init() {
	// mcp flags
	mcpAddCmd.Flags().StringVar(&mcpAddTransport, "transport", "", "Transport type: stdio, sse, http, ws")
	mcpAddCmd.Flags().StringVar(&mcpAddScope, "scope", "project", "Config scope: project, user, local")
	mcpAddCmd.Flags().StringSliceVar(&mcpAddEnv, "env", nil, "Environment variables (KEY=VALUE)")
	mcpAddCmd.Flags().StringSliceVar(&mcpAddHeader, "header", nil, "HTTP headers (KEY=VALUE)")
	mcpRemoveCmd.Flags().StringVar(&mcpRemoveScope, "scope", "project", "Config scope")
	mcpAddJSONCmd.Flags().StringVar(&mcpAddJSONScope, "scope", "project", "Config scope")

	mcpCmd.AddCommand(mcpAddCmd)
	mcpCmd.AddCommand(mcpRemoveCmd)
	mcpCmd.AddCommand(mcpListCmd)
	mcpCmd.AddCommand(mcpGetCmd)
	mcpCmd.AddCommand(mcpAddJSONCmd)
}

func init() {
	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authStatusCmd)
	authCmd.AddCommand(authLogoutCmd)
}
