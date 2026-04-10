package querycontext

import (
	"bytes"
	"context"
	"os/exec"
	"path/filepath"
	"strings"
)

// MaxStatusChars matches src/context.ts MAX_STATUS_CHARS.
const MaxStatusChars = 2000

// BuildGitStatusSnapshot mirrors getGitStatus in src/context.ts (git snapshot string or empty).
func BuildGitStatusSnapshot(ctx context.Context, cwd string) string {
	if ctx.Err() != nil {
		return ""
	}
	abs, err := filepath.Abs(cwd)
	if err != nil {
		return ""
	}
	git := gitExe()
	if git == "" {
		return ""
	}
	// TS: getIsGit first
	if !gitInsideWorkTree(ctx, git, abs) {
		return ""
	}

	branch := strings.TrimSpace(runGitOut(ctx, abs, git, "rev-parse", "--abbrev-ref", "HEAD"))
	mainBranch := defaultBranchName(ctx, git, abs)
	status := strings.TrimSpace(runGitOut(ctx, abs, git, "--no-optional-locks", "status", "--short"))
	logOut := strings.TrimSpace(runGitOut(ctx, abs, git, "--no-optional-locks", "log", "--oneline", "-n", "5"))
	userName := strings.TrimSpace(runGitOut(ctx, abs, git, "config", "user.name"))

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
	return strings.Join(parts, "\n\n")
}

func gitExe() string {
	if p, err := exec.LookPath("git"); err == nil {
		return p
	}
	return ""
}

func gitInsideWorkTree(ctx context.Context, git, cwd string) bool {
	out := runGitOut(ctx, cwd, git, "rev-parse", "--is-inside-work-tree")
	return strings.TrimSpace(strings.ToLower(out)) == "true"
}

func defaultBranchName(ctx context.Context, git, cwd string) string {
	// Prefer symbolic-ref for origin/HEAD (TS computeDefaultBranch).
	ref := strings.TrimSpace(runGitOut(ctx, cwd, git, "symbolic-ref", "-q", "--short", "refs/remotes/origin/HEAD"))
	if ref != "" {
		// ref like "origin/main" → "main"
		if i := strings.LastIndex(ref, "/"); i >= 0 && i+1 < len(ref) {
			return ref[i+1:]
		}
		return ref
	}
	for _, c := range []string{"main", "master"} {
		sha := strings.TrimSpace(runGitOut(ctx, cwd, git, "rev-parse", "--verify", "refs/remotes/origin/"+c))
		if sha != "" {
			return c
		}
	}
	return "main"
}

func runGitOut(ctx context.Context, cwd, git string, args ...string) string {
	var buf bytes.Buffer
	c := exec.CommandContext(ctx, git, args...)
	c.Dir = cwd
	c.Stdout = &buf
	c.Stderr = nil
	_ = c.Run()
	return buf.String()
}
