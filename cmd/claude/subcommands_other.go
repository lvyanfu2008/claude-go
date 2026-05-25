package main

import (
	"fmt"
	"os"
	"runtime"

	"goc/tools"

	"github.com/spf13/cobra"
)

// ---- agents ----

var agentsSettingSources string

var agentsCmd = &cobra.Command{
	Use:   "agents",
	Short: "List configured custom agents",
	Long:  "List the custom agents configured in settings files and --agents flag.",
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, _ := os.Getwd()
		report := tools.LoadAgentDefinitionsReport(cwd)
		fmt.Printf("Configured agents (%d total, %d active):\n", len(report.AllAgents), len(report.ActiveAgents))
		for _, a := range report.ActiveAgents {
			fmt.Printf("  %-25s  source=%-14s  model=%-8s  tools=%d\n", a.AgentType, a.Source, a.Model, len(a.Tools))
		}
		if len(report.FailedFiles) > 0 {
			fmt.Println("\nFailed:")
			for _, f := range report.FailedFiles {
				fmt.Printf("  %s: %s\n", f.Path, f.Error)
			}
		}
		return nil
	},
}

// ---- auto-mode ----

var autoModeCmd = &cobra.Command{
	Use:   "auto-mode",
	Short: "Auto mode classifier configuration",
	Long:  "View defaults, config, and get AI feedback on auto-mode rules.",
}

var autoModeDefaultsCmd = &cobra.Command{
	Use:   "defaults",
	Short: "Print default auto-mode env/allow/deny rules as JSON",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Auto-mode defaults not yet implemented in Go.")
		return nil
	},
}

var autoModeConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Print effective auto-mode config as JSON",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Auto-mode config not yet implemented in Go.")
		return nil
	},
}

var autoModeCritiqueModel string

var autoModeCritiqueCmd = &cobra.Command{
	Use:   "critique",
	Short: "Get AI feedback on custom auto-mode rules",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Auto-mode critique not yet implemented in Go.")
		return nil
	},
}

// ---- server ----

var serverPort int
var serverHost string
var serverAuthToken string
var serverUnix string
var serverWorkspace string
var serverIdleTimeout int
var serverMaxSessions int

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start a Claude Code session server",
	Long:  "Start an HTTP server that accepts remote Claude Code connections.",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Server mode is not yet implemented in Go.")
		fmt.Println("Use the TS CLI: claude server --port <port>")
		return nil
	},
}

// ---- ssh ----

var sshPermissionMode string
var sshDangerouslySkipPermissions bool
var sshLocal bool

var sshCmd = &cobra.Command{
	Use:   "ssh <host> [dir]",
	Short: "Run Claude Code on a remote host via SSH",
	Long:  "Deploy and run Claude Code on a remote Linux host over SSH.",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		host := args[0]
		dir := ""
		if len(args) > 1 {
			dir = args[1]
		}
		fmt.Printf("SSH remote to %q (dir=%q) not yet implemented in Go.\n", host, dir)
		return nil
	},
}

// ---- open ----

var openPrint bool
var openOutputFormat string

var openCmd = &cobra.Command{
	Use:   "open <cc-url>",
	Short: "Connect to a Claude Code server",
	Long:  "Connect to a remote Claude Code server using a cc:// URL.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		url := args[0]
		fmt.Printf("Opening connection to %q... (not yet implemented in Go)\n", url)
		return nil
	},
}

// ---- setup-token ----

var setupTokenCmd = &cobra.Command{
	Use:   "setup-token",
	Short: "Set up a long-lived authentication token",
	Long: `Configure a long-lived authentication token for programmatic access.
Requires a Claude subscription.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Token setup:")
		fmt.Println("  1. Set ANTHROPIC_API_KEY for API-key-backed access")
		fmt.Println("  2. Or set ANTHROPIC_AUTH_TOKEN for OAuth-token-backed access")
		fmt.Println("")
		fmt.Println("For Anthropic Console API keys, visit: https://console.anthropic.com/")
		return nil
	},
}

// ---- install ----

var installForce bool

var installCmd = &cobra.Command{
	Use:   "install [target]",
	Short: "Install Claude Code native build",
	Long:  "Install Claude Code. Target can be 'stable', 'latest', or a specific version.",
	RunE: func(cmd *cobra.Command, args []string) error {
		target := "stable"
		if len(args) > 0 {
			target = args[0]
		}
		fmt.Printf("Installing Claude Code %s...\n", target)
		fmt.Println("Native build installation is only available via the TS CLI.")
		fmt.Println("Go build: cd claude-go && go build -o /usr/local/bin/claude ./cmd/claude")
		return nil
	},
}

// ---- assistant ----

var assistantCmd = &cobra.Command{
	Use:   "assistant [sessionId]",
	Short: "Attach as a client to a running bridge session",
	Long:  "Attach the REPL as a client to a running bridge session.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			fmt.Printf("Attaching to session %q... (not yet implemented in Go)\n", args[0])
		} else {
			fmt.Println("Discovering sessions... (not yet implemented in Go)")
		}
		return nil
	},
}

// ---- remote-control ----

var remoteControlCmd = &cobra.Command{
	Use:     "remote-control",
	Aliases: []string{"rc", "remote", "sync", "bridge"},
	Short:   "Serve local machine as bridge environment",
	Long:    "Connect local environment for remote-control sessions via claude.ai/code.",
	Hidden:  true,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Remote control mode not yet implemented in Go.")
		return nil
	},
}

// ---- up ----

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Initialize or upgrade local dev environment",
	Long:  "Run the '# claude up' section from the nearest HARNESS.md.",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("'claude up' not yet implemented in Go.")
		return nil
	},
}

// ---- version (as subcommand) ----

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("%s (Claude Code Go) %s/%s\n", versionString(), runtime.GOOS, runtime.GOARCH)
	},
}

func init() {
	// agents flags
	agentsCmd.Flags().StringVar(&agentsSettingSources, "setting-sources", "", "Setting sources to search")

	// auto-mode flags
	autoModeCritiqueCmd.Flags().StringVar(&autoModeCritiqueModel, "model", "", "Model to use for critique")
	autoModeCmd.AddCommand(autoModeDefaultsCmd)
	autoModeCmd.AddCommand(autoModeConfigCmd)
	autoModeCmd.AddCommand(autoModeCritiqueCmd)

	// server flags
	serverCmd.Flags().IntVar(&serverPort, "port", 0, "HTTP port (0 = random)")
	serverCmd.Flags().StringVar(&serverHost, "host", "0.0.0.0", "Bind address")
	serverCmd.Flags().StringVar(&serverAuthToken, "auth-token", "", "Bearer token for authentication")
	serverCmd.Flags().StringVar(&serverUnix, "unix", "", "Listen on Unix domain socket")
	serverCmd.Flags().StringVar(&serverWorkspace, "workspace", "", "Default workspace directory")
	serverCmd.Flags().IntVar(&serverIdleTimeout, "idle-timeout", 600000, "Idle timeout in ms")
	serverCmd.Flags().IntVar(&serverMaxSessions, "max-sessions", 32, "Max concurrent sessions")

	// ssh flags
	sshCmd.Flags().StringVar(&sshPermissionMode, "permission-mode", "", "Permission mode for remote session")
	sshCmd.Flags().BoolVar(&sshDangerouslySkipPermissions, "dangerously-skip-permissions", false, "Skip all permissions")
	sshCmd.Flags().BoolVar(&sshLocal, "local", false, "Spawn locally (e2e test mode)")

	// open flags
	openCmd.Flags().BoolVarP(&openPrint, "print", "p", false, "Print mode (headless)")
	openCmd.Flags().StringVar(&openOutputFormat, "output-format", "text", "Output format")

	// install flags
	installCmd.Flags().BoolVar(&installForce, "force", false, "Force install")
}
