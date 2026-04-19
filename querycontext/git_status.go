package querycontext

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"goc/utils"
)

// MaxStatusChars matches src/context.ts MAX_STATUS_CHARS.
const MaxStatusChars = 2000

// BuildGitStatusSnapshot mirrors getGitStatus in src/context.ts (git snapshot string or empty).
func BuildGitStatusSnapshot(ctx context.Context, cwd string) string {
	startTime := time.Now()

	if ctx.Err() != nil {
		return ""
	}
	abs, err := filepath.Abs(cwd)
	if err != nil {
		return ""
	}

	// Check if we're in a git repo
	if !utils.GitInsideWorkTree(abs) {
		return ""
	}

	// Run Git commands in parallel like TS Promise.all
	commands := [][]string{
		{"rev-parse", "--abbrev-ref", "HEAD"},                     // branch
		{"symbolic-ref", "-q", "--short", "refs/remotes/origin/HEAD"}, // for default branch
		{"--no-optional-locks", "status", "--short"},              // status
		{"--no-optional-locks", "log", "--oneline", "-n", "5"},    // log
		{"config", "user.name"},                                   // userName
	}

	results, errs := utils.RunGitCommandsParallel(abs, commands)

	// Check for critical errors
	for i, err := range errs {
		if err != nil && i != 1 { // symbolic-ref can fail, that's OK
			// Log error but continue with partial results
			fmt.Printf("Git command failed: %v\n", err)
		}
	}

	branch := results[0]
	mainBranch := utils.ExtractDefaultBranch(results[1])
	status := results[2]
	logOut := results[3]
	userName := results[4]

	// Fallback for default branch if symbolic-ref failed
	if mainBranch == "" {
		mainBranch = defaultBranchName(ctx, abs)
	}

	truncated := status
	if len(truncated) > MaxStatusChars {
		truncated = truncated[:MaxStatusChars] + "\n... (truncated because it exceeds 2k characters. If you need more information, run \"git status\" using BashTool)"
	}
	if truncated == "" {
		truncated = "(clean)"
	}

	parts := []string{
		`This is the git status at the start of the conversation. Note that this status is a snapshot in time, and will not update during the conversation.`,
		`Current branch: ` + branch,
		`Main branch (you will usually use this for PRs): ` + mainBranch,
	}
	if userName != "" {
		parts = append(parts, `Git user: `+userName)
	}
	parts = append(parts, `Status:
`+truncated, `Recent commits:
`+logOut)

	// Log completion time for diagnostics (simplified version of TS logging)
	fmt.Printf("Git status completed in %v\n", time.Since(startTime))

	return strings.Join(parts, "\n\n")
}

func defaultBranchName(ctx context.Context, cwd string) string {
	// Try symbolic-ref first
	ref, err := utils.RunGitCommand(cwd, "symbolic-ref", "-q", "--short", "refs/remotes/origin/HEAD")
	if err == nil && ref != "" {
		return utils.ExtractDefaultBranch(ref)
	}

	// Fallback to checking common branch names
	for _, c := range []string{"main", "master"} {
		sha, err := utils.RunGitCommand(cwd, "rev-parse", "--verify", "refs/remotes/origin/"+c)
		if err == nil && sha != "" {
			return c
		}
	}
	return "main"
}

// runGitOut is kept for backward compatibility
func runGitOut(ctx context.Context, cwd, git string, args ...string) string {
	cmd := exec.CommandContext(ctx, git, args...)
	cmd.Dir = cwd
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return string(output)
}
