// Command claude is the unified CLI entry point for the Go implementation
// of Claude Code. It provides the same flags and subcommands as the TS reference.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
	"runtime/debug"
	"time"

	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"

	"goc/conversation-runtime/process-user-input"
	"goc/conversation-runtime/query"
	"goc/gou/app"
	"goc/gou/commandqueue"
	"goc/hookexec"
	_ "goc/plugins"
	"goc/sessiontranscript"
	"goc/types"
)

var (
	// Core flags
	flagPrint                 bool
	flagBare                  bool
	flagDebug                 string
	flagDebugFile             string
	flagVerbose               bool
	flagOutputFormat          string
	flagInputFormat           string
	flagIncludePartialMsgs    bool

	// Session flags
	flagContinue       bool
	flagResume         string
	flagForkSession    bool
	flagSessionID      string
	flagSessionName    string
	flagNoPersistence  bool
	flagFromPR         string

	// Model flags
	flagModel          string
	flagFallbackModel  string
	flagEffort         string
	flagMaxTurns       int
	flagThinking       string
	flagMaxBudgetUSD   float64
	flagAgent          string
	flagBetas          []string

	// Permission flags
	flagPermissionMode                string
	flagDangerouslySkipPermissions    bool
	flagAllowedTools                  []string
	flagDisallowedTools               []string
	flagTools                         []string

	// Config flags
	flagSystemPrompt        string
	flagSystemPromptFile    string
	flagAppendSystemPrompt  string
	flagMCPConfig           []string
	flagStrictMCPConfig     bool
	flagSettings            string
	flagAddDir              []string
	flagAgents              string
	flagSettingSources      string
	flagPluginDir           []string
	flagDisableSlashCmds    bool

	// Environment flags
	flagWorktree string
	flagIDE      bool
)

func main() {
	// Fast-path for --version / -v: zero module loading needed
	if len(os.Args) >= 2 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		version := "dev"
		if bi, ok := debug.ReadBuildInfo(); ok {
			v := bi.Main.Version
			if v != "" && v != "(devel)" {
				version = v
			}
		}
		fmt.Printf("%s (Claude Code Go) %s/%s\n", version, runtime.GOOS, runtime.GOARCH)
		return
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "claude [flags] [prompt]",
	Short: "Claude Code - AI coding assistant",
	Long: `Claude Code is an AI-powered coding assistant that helps you
write, debug, and understand code from the terminal.`,
	Args:    cobra.MaximumNArgs(1),
	RunE:    runClaude,
	Version: versionString(),
}

func versionString() string {
	if bi, ok := debug.ReadBuildInfo(); ok {
		v := bi.Main.Version
		if v != "" && v != "(devel)" {
			return v
		}
	}
	return "dev"
}

func init() {
	// Core flags
	rootCmd.Flags().BoolVarP(&flagPrint, "print", "p", false, "Print response and exit (non-interactive mode)")
	rootCmd.Flags().BoolVar(&flagBare, "bare", false, "Minimal/bare output mode")
	rootCmd.Flags().StringVarP(&flagDebug, "debug", "d", "", "Enable debug logging [filter]")
	rootCmd.Flags().StringVar(&flagDebugFile, "debug-file", "", "Write debug logs to a specific file path")
	rootCmd.Flags().BoolVar(&flagVerbose, "verbose", false, "Verbose output")
	rootCmd.Flags().StringVar(&flagOutputFormat, "output-format", "text", "Output format: text, json, stream-json")
	rootCmd.Flags().StringVar(&flagInputFormat, "input-format", "text", "Input format: text, stream-json")
	rootCmd.Flags().BoolVar(&flagIncludePartialMsgs, "include-partial-messages", false, "Include partial message chunks as they arrive (only works with --print and --output-format=stream-json)")

	// Session flags
	rootCmd.Flags().BoolVarP(&flagContinue, "continue", "c", false, "Continue the most recent conversation")
	rootCmd.Flags().StringVarP(&flagResume, "resume", "r", "", "Resume a conversation by session ID")
	rootCmd.Flags().BoolVar(&flagForkSession, "fork-session", false, "Create a new session ID on resume")
	rootCmd.Flags().StringVar(&flagSessionID, "session-id", "", "Use a specific session ID")
	rootCmd.Flags().StringVarP(&flagSessionName, "name", "n", "", "Display name for the session")
	rootCmd.Flags().BoolVar(&flagNoPersistence, "no-session-persistence", false, "Disable session persistence to disk")
	rootCmd.Flags().StringVar(&flagFromPR, "from-pr", "", "Resume a session linked to a PR by PR number/URL")

	// Model flags
	rootCmd.Flags().StringVar(&flagModel, "model", "", "Model to use (e.g. claude-sonnet-4-6)")
	rootCmd.Flags().StringVar(&flagFallbackModel, "fallback-model", "", "Fallback model")
	rootCmd.Flags().StringVar(&flagEffort, "effort", "", "Effort level: low, medium, high, max")
	rootCmd.Flags().IntVar(&flagMaxTurns, "max-turns", 0, "Maximum number of agentic turns (only works with --print)")
	rootCmd.Flags().Float64Var(&flagMaxBudgetUSD, "max-budget-usd", 0, "Maximum dollar amount to spend on API calls (only works with --print)")
	rootCmd.Flags().StringVar(&flagThinking, "thinking", "", "Thinking mode: enabled, adaptive, disabled")
	rootCmd.Flags().StringVar(&flagAgent, "agent", "", "Agent for the current session")
	rootCmd.Flags().StringSliceVar(&flagBetas, "betas", nil, "Beta headers to include in API requests")

	// Permission flags
	rootCmd.Flags().StringVar(&flagPermissionMode, "permission-mode", "", "Permission mode: default, acceptEdits, bypassPermissions, plan, auto")
	rootCmd.Flags().BoolVar(&flagDangerouslySkipPermissions, "dangerously-skip-permissions", false, "Bypass all permission checks")
	rootCmd.Flags().StringSliceVar(&flagAllowedTools, "allowedTools", nil, "Tools to allow")
	rootCmd.Flags().StringSliceVar(&flagDisallowedTools, "disallowedTools", nil, "Tools to disallow")
	rootCmd.Flags().StringSliceVar(&flagTools, "tools", nil, "Specify the list of available tools from the built-in set")

	// Config flags
	rootCmd.Flags().StringVar(&flagSystemPrompt, "system-prompt", "", "Custom system prompt")
	rootCmd.Flags().StringVar(&flagSystemPromptFile, "system-prompt-file", "", "Read system prompt from file")
	rootCmd.Flags().StringVar(&flagAppendSystemPrompt, "append-system-prompt", "", "Append a system prompt to the default system prompt")
	rootCmd.Flags().StringSliceVar(&flagMCPConfig, "mcp-config", nil, "MCP server configs as JSON")
	rootCmd.Flags().BoolVar(&flagStrictMCPConfig, "strict-mcp-config", false, "Only use MCP servers from --mcp-config, ignoring other MCP configurations")
	rootCmd.Flags().StringVar(&flagSettings, "settings", "", "Additional settings file or JSON")
	rootCmd.Flags().StringSliceVar(&flagAddDir, "add-dir", nil, "Additional directories for tool access")
	rootCmd.Flags().StringVar(&flagAgents, "agents", "", "JSON object defining custom agents")
	rootCmd.Flags().StringVar(&flagSettingSources, "setting-sources", "", "Comma-separated list of setting sources to load (user, project, local)")
	rootCmd.Flags().StringSliceVar(&flagPluginDir, "plugin-dir", nil, "Load plugins from a directory for this session only")
	rootCmd.Flags().BoolVar(&flagDisableSlashCmds, "disable-slash-commands", false, "Disable all skills")

	// Environment flags
	rootCmd.Flags().StringVarP(&flagWorktree, "worktree", "w", "", "Create a git worktree for this session")
	rootCmd.Flags().BoolVar(&flagIDE, "ide", false, "Auto-connect to IDE")

	// Register subcommands
	rootCmd.AddCommand(mcpCmd)
	rootCmd.AddCommand(doctorCmd)
	rootCmd.AddCommand(authCmd)
	rootCmd.AddCommand(completionCmd)
	rootCmd.AddCommand(pluginCmd)
	rootCmd.AddCommand(agentsCmd)
	rootCmd.AddCommand(autoModeCmd)
	rootCmd.AddCommand(serverCmd)
	rootCmd.AddCommand(sshCmd)
	rootCmd.AddCommand(openCmd)
	rootCmd.AddCommand(setupTokenCmd)
	rootCmd.AddCommand(installCmd)
	rootCmd.AddCommand(assistantCmd)
	rootCmd.AddCommand(remoteControlCmd)
	rootCmd.AddCommand(upCmd)
	rootCmd.AddCommand(versionCmd)
}

// runClaude is the main command handler.
func runClaude(cmd *cobra.Command, args []string) error {
	// Set environment variables from flags
	if flagVerbose {
		os.Setenv("CLAUDE_CODE_VERBOSE", "1")
	}
	if flagBare || flagPrint {
		os.Setenv("CLAUDE_CODE_SIMPLE", "1")
	}
	if flagPermissionMode != "" {
		os.Setenv("CLAUDE_CODE_PERMISSION_MODE", flagPermissionMode)
	}
	if flagDebug != "" {
		os.Setenv("CLAUDE_CODE_DEBUG", flagDebug)
	}
	if flagDebugFile != "" {
		os.Setenv("CLAUDE_CODE_DEBUG_FILE", flagDebugFile)
	}
	if flagNoPersistence {
		os.Setenv("CLAUDE_CODE_NO_SESSION_PERSISTENCE", "1")
	}
	if flagModel != "" {
		os.Setenv("CLAUDE_CODE_MODEL", flagModel)
	}
	if flagDangerouslySkipPermissions {
		os.Setenv("CLAUDE_CODE_DANGEROUSLY_SKIP_PERMISSIONS", "1")
	}
	if flagEffort != "" {
		os.Setenv("CLAUDE_CODE_EFFORT_LEVEL", flagEffort)
	}
	if flagAgent != "" {
		os.Setenv("CLAUDE_CODE_AGENT", flagAgent)
	}
	if flagDisableSlashCmds {
		os.Setenv("CLAUDE_CODE_DISABLE_SLASH_COMMANDS", "1")
	}
	if flagStrictMCPConfig {
		os.Setenv("CLAUDE_CODE_STRICT_MCP_CONFIG", "1")
	}
	if flagSettingSources != "" {
		os.Setenv("CLAUDE_CODE_SETTING_SOURCES", flagSettingSources)
	}
	if flagAgents != "" {
		os.Setenv("CLAUDE_CODE_AGENTS_JSON", flagAgents)
	}
	if flagIncludePartialMsgs {
		os.Setenv("CLAUDE_CODE_INCLUDE_PARTIAL_MESSAGES", "1")
	}
	if flagMaxBudgetUSD > 0 {
		os.Setenv("CLAUDE_CODE_MAX_BUDGET_USD", fmt.Sprintf("%.2f", flagMaxBudgetUSD))
	}

	// Set bootstrap state from flags
	ApplyBootstrapState(BootstrapConfig{
		Print:             flagPrint,
		Continue:          flagContinue,
		Resume:            flagResume,
		ForkSession:       flagForkSession,
		SessionID:         flagSessionID,
		SessionName:       flagSessionName,
		OutputFormat:      flagOutputFormat,
		InputFormat:       flagInputFormat,
		MaxTurns:          flagMaxTurns,
		Thinking:          flagThinking,
		AllowedTools:      flagAllowedTools,
		DisallowedTools:   flagDisallowedTools,
		Tools:             flagTools,
		MCPConfig:         flagMCPConfig,
		AddDir:            flagAddDir,
		Worktree:          flagWorktree,
		PluginDir:         flagPluginDir,
	})

	if flagVerbose {
		fmt.Fprintf(os.Stderr, "[claude] flags: print=%v continue=%v resume=%q session=%q model=%q effort=%q max-turns=%d\n",
			flagPrint, flagContinue, flagResume, flagSessionID, flagModel, flagEffort, flagMaxTurns)
	}

	// If --print mode, run headless
	if flagPrint {
		prompt := ""
		if len(args) > 0 {
			prompt = args[0]
		}
		return runHeadless(prompt)
	}

	// Default: launch interactive REPL
	return runInteractive(args)
}

func runHeadless(prompt string) error {
	// Read from stdin if no prompt given (pipe mode).
	if prompt == "" && !isatty.IsTerminal(os.Stdin.Fd()) {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("read stdin: %w", err)
		}
		prompt = string(data)
	}

	if prompt == "" {
		return fmt.Errorf("no prompt provided (use --print 'your prompt' or pipe input)")
	}

	fmt.Fprintf(os.Stderr, "[claude] headless mode: processing prompt (%d chars)...\n", len(prompt))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Initialize MCP if configured.
	if len(flagMCPConfig) > 0 {
		if err := InitMCP(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "[claude] MCP init warning: %v\n", err)
		}
		defer ShutdownMCP()
		defer StopCronScheduler()
	}

	// Resolve working directory and session ID for hook execution.
	cwd, _ := os.Getwd()
	// Start cron scheduler for background cron jobs.
	StartCronScheduler(ctx, cwd, func(fireCtx context.Context, prompt string) {
		fmt.Fprintf(os.Stderr, "[claude] cron fire: %s\n", prompt)
	})
	sessionID := GetSessionID()
	transcriptPath := sessiontranscript.TranscriptPath(sessionID, cwd, "", sessiontranscript.ConfigHomeDir())

	// Build base hook input (mirrors TS toolUseContext injection).
	pm := flagPermissionMode
	if pm == "" {
		pm = string(types.PermissionDefault)
	}
	baseHookInput := hookexec.BaseHookInput{
		SessionID:      sessionID,
		TranscriptPath: transcriptPath,
		Cwd:            cwd,
		PermissionMode: pm,
		HookEventName:  "UserPromptSubmit",
	}

	// Load merged hooks and wire UserPromptSubmit hook runner (mirrors TS processUserInput internals).
	mergedHooks, _ := hookexec.MergedHooksForCwd(cwd)
	baseRunner := hookexec.MakeUserPromptSubmitHookRunner(mergedHooks, cwd, baseHookInput)

	inputJSON, _ := json.Marshal(prompt)
	p := &processuserinput.ProcessUserInputParams{
		Input:                 json.RawMessage(inputJSON),
		Mode:                  types.PromptInputModePrompt,
		PermissionMode:        types.PermissionMode(pm),
		GetAttachmentMessages: processuserinput.NewDefaultGetAttachmentMessages(cwd),
	}
	// Adapter: hookexec returns func(ctx, inputMessage); ExecuteUserPromptSubmitHooks expects func(ctx, *ProcessUserInputParams, inputMessage).
	if baseRunner != nil {
		p.ExecuteUserPromptSubmitHooks = func(ctx context.Context, _ *processuserinput.ProcessUserInputParams, inputMessage string) ([]types.AggregatedHookResult, error) {
			return baseRunner(ctx, inputMessage)
		}
	}

	// Defer SessionEnd hooks for cleanup on shutdown (TS parity: executeSessionEndHooks).
	sessionEndReason := "error"
	defer func() {
		bgCtx, bgCancel := context.WithTimeout(context.Background(), time.Duration(hookexec.SessionEndHookTimeoutMs)*time.Millisecond)
		defer bgCancel()
		hookexec.RunSessionEndHooks(bgCtx, mergedHooks, cwd, baseHookInput, sessionEndReason, sessionID)
	}()

	// Process user input (includes hook execution inline, mirrors TS processUserInput flow).
	result, err := processuserinput.ProcessUserInput(ctx, p)
	if err != nil {
		return fmt.Errorf("process prompt: %w", err)
	}
	if !result.ShouldQuery {
		sessionEndReason = "normal"
		// Hook may have blocked the query — print hook messages if any.
		for _, msg := range result.Messages {
			if msg.Type == "system" {
				var sys struct{ Content string }
				json.Unmarshal([]byte(msg.Content), &sys)
				if sys.Content != "" {
					fmt.Println(sys.Content)
				}
			}
		}
		return nil
	}

	// Build query params.
	var maxTurns *int
	if flagMaxTurns > 0 {
		maxTurns = &flagMaxTurns
	}
	qparams := query.QueryParams{
		Messages: result.Messages,
		MaxTurns: maxTurns,
	}

	if flagSystemPrompt != "" {
		qparams.SystemPrompt = query.SystemPrompt{flagSystemPrompt}
	}
	if flagSystemPromptFile != "" {
		data, err := os.ReadFile(flagSystemPromptFile)
		if err != nil {
			return fmt.Errorf("read system prompt file: %w", err)
		}
		qparams.SystemPrompt = query.SystemPrompt{string(data)}
	}
	// Append system prompt if set
	if flagAppendSystemPrompt != "" {
		promptData := os.Getenv("CLAUDE_CODE_APPEND_SYSTEM_PROMPT")
		if promptData == "" {
			promptData = flagAppendSystemPrompt
		} else {
			promptData = promptData + "\n" + flagAppendSystemPrompt
		}
		os.Setenv("CLAUDE_CODE_APPEND_SYSTEM_PROMPT", promptData)
	}

	// Run the query and collect assistant text.
	for yield, err := range query.Query(ctx, qparams) {
		if err != nil {
			return fmt.Errorf("query: %w", err)
		}
		if yield.Message != nil && yield.Message.Type == "assistant" {
			var blocks []types.MessageContentBlock
			if err := json.Unmarshal([]byte(yield.Message.Content), &blocks); err == nil {
				for _, block := range blocks {
					if block.Type == "text" {
						fmt.Print(block.Text)
					}
				}
			}
		}
		if yield.Terminal != nil {
			break
		}
	}

	fmt.Println()
	// Drain command queue: keep processing background agent notifications
	// until no more pending results (matches TS print.ts do-while loop).
	msgs := result.Messages
	for commandqueue.HasPendingNotifications() {
		notifications := commandqueue.DrainCommandQueue()
		for _, n := range notifications {
			content, _ := json.Marshal([]map[string]any{{
				"type": "text",
				"text": "<system-reminder>\n" + n.Value + "\n</system-reminder>",
			}})
			msgs = append(msgs, types.Message{
				Type:    "user",
				Content: content,
			})
		}

		qparams2 := qparams
		qparams2.Messages = msgs

		for yield, err := range query.Query(ctx, qparams2) {
			if err != nil {
				return fmt.Errorf("query (notification drain): %w", err)
			}
			if yield.Message != nil && yield.Message.Type == "assistant" {
				var blocks []json.RawMessage
				if err := json.Unmarshal(yield.Message.Content, &blocks); err == nil {
					for _, block := range blocks {
						var b struct {
							Type string `json:"type"`
							Text string `json:"text"`
						}
						if json.Unmarshal(block, &b) == nil && b.Type == "text" {
							fmt.Print(b.Text)
						}
					}
				}
			}
			if yield.Terminal != nil {
				break
			}
		}
		fmt.Println()
	}

	return nil
}

func runInteractive(args []string) error {
	cwd, _ := os.Getwd()
	sessionID := GetSessionID()

	pm := flagPermissionMode
	if pm == "" {
		pm = string(types.PermissionDefault)
	}

	cfg := app.Config{
		SessionID:      sessionID,
		PermissionMode: types.PermissionMode(pm),
		CWD:            cwd,
	}

	// Start cron scheduler for background cron jobs.
	StartCronScheduler(context.Background(), cwd, func(fireCtx context.Context, prompt string) {
		fmt.Fprintf(os.Stderr, "[claude] cron fire: %s\n", prompt)
	})
	defer StopCronScheduler()

	// Load merged hooks and defer SessionEnd hooks for interactive mode.
	mergedHooks, _ := hookexec.MergedHooksForCwd(cwd)
	transcriptPath := sessiontranscript.TranscriptPath(sessionID, cwd, "", sessiontranscript.ConfigHomeDir())
	baseHookInput := hookexec.BaseHookInput{
		SessionID:      sessionID,
		TranscriptPath: transcriptPath,
		Cwd:            cwd,
		PermissionMode: pm,
	}
	sessionEndReason := "error"
	defer func() {
		bgCtx, bgCancel := context.WithTimeout(context.Background(), time.Duration(hookexec.SessionEndHookTimeoutMs)*time.Millisecond)
		defer bgCancel()
		hookexec.RunSessionEndHooks(bgCtx, mergedHooks, cwd, baseHookInput, sessionEndReason, sessionID)
	}()

	err := app.Run(cfg)
	if err == nil {
		sessionEndReason = "normal"
	}
	return err
}
