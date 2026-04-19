package utils

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

// GitExe returns the path to git executable, similar to gitExe() in src/utils/git.ts
func GitExe() string {
	if p, err := exec.LookPath("git"); err == nil {
		return p
	}
	return "git"
}

// FindGitRoot walks up from start until a .git directory or file exists
// Mirrors findGitRoot in src/utils/git.ts
func FindGitRoot(start string) string {
	abs, err := filepath.Abs(start)
	if err != nil {
		return ""
	}
	cur := filepath.Clean(abs)
	root := filepath.VolumeName(cur) + string(filepath.Separator)
	if root == "" {
		root = string(filepath.Separator)
	}

	for cur != root {
		gitPath := filepath.Join(cur, ".git")
		if fi, err := os.Lstat(gitPath); err == nil {
			// .git can be a directory (regular repo) or file (worktree/submodule)
			if fi.IsDir() || fi.Mode().IsRegular() {
				return filepath.Clean(cur)
			}
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			break
		}
		cur = parent
	}

	// Check root directory as well
	gitPath := filepath.Join(root, ".git")
	if fi, err := os.Lstat(gitPath); err == nil {
		if fi.IsDir() || fi.Mode().IsRegular() {
			return filepath.Clean(root)
		}
	}

	return ""
}

// IsNormalizedGitCommand checks if a command is a git command after normalization
// Mirrors isNormalizedGitCommand in src/tools/BashTool/bashPermissions.ts
func IsNormalizedGitCommand(command string) bool {
	// Fast path: catch the most common case before any parsing
	if strings.HasPrefix(command, "git ") || command == "git" {
		return true
	}

	// Simple stripping of safe wrappers (simplified version)
	stripped := stripSafeWrappers(command)

	// Check for direct git command
	if strings.HasPrefix(stripped, "git ") || stripped == "git" {
		return true
	}

	// Check for "xargs git ..." pattern
	if strings.HasPrefix(stripped, "xargs ") && strings.Contains(stripped, " git") {
		return true
	}

	return false
}

// stripSafeWrappers removes safe command wrappers like time, nice, etc.
// Simplified version of stripSafeWrappers in TS
func stripSafeWrappers(command string) string {
	// Remove leading/trailing whitespace
	cmd := strings.TrimSpace(command)

	// Strip environment variables
	cmd = stripEnvVars(cmd)

	// List of safe prefixes to strip
	safePrefixes := []string{
		"time ", "nice ", "nohup ", "stdbuf ", "setsid ",
		"script -q -c '", "script -q -c \"",
	}

	for _, prefix := range safePrefixes {
		if strings.HasPrefix(cmd, prefix) {
			// Remove the prefix
			cmd = strings.TrimPrefix(cmd, prefix)
			// Handle quoted commands
			if strings.HasSuffix(cmd, "'") || strings.HasSuffix(cmd, "\"") {
				cmd = cmd[:len(cmd)-1]
			}
			break
		}
	}

	return strings.TrimSpace(cmd)
}

// stripEnvVars strips environment variable assignments from beginning of command
func stripEnvVars(command string) string {
	cmd := strings.TrimSpace(command)

	// Pattern for env var assignment: VAR=value
	// Keep stripping until we find a non-env-var token
	for {
		// Find first space
		spaceIdx := strings.Index(cmd, " ")
		if spaceIdx == -1 {
			// No space, check if it's just an env var
			if strings.Contains(cmd, "=") {
				return ""
			}
			return cmd
		}

		// Check if part before space looks like env var
		prefix := cmd[:spaceIdx]
		if strings.Contains(prefix, "=") {
			// Strip this env var and continue
			cmd = strings.TrimSpace(cmd[spaceIdx:])
		} else {
			// Not an env var, stop
			break
		}
	}

	return cmd
}

// IsNormalizedCdCommand checks if a command is a cd command after normalization
// Mirrors isNormalizedCdCommand in TS
func IsNormalizedCdCommand(command string) bool {
	cmd := strings.TrimSpace(command)

	// Fast path
	if strings.HasPrefix(cmd, "cd ") || cmd == "cd" {
		return true
	}

	// Check with safe wrapper stripping
	stripped := stripSafeWrappers(cmd)
	return strings.HasPrefix(stripped, "cd ") || stripped == "cd"
}

// IsCurrentDirectoryBareGitRepo checks if cwd looks like a bare/exploited git directory
// Mirrors isCurrentDirectoryBareGitRepo in src/utils/git.ts (security check)
func IsCurrentDirectoryBareGitRepo(cwd string) bool {
	gitPath := filepath.Join(cwd, ".git")

	// Check if .git exists and is valid
	if fi, err := os.Lstat(gitPath); err == nil {
		if fi.IsDir() {
			// Check if .git/HEAD is a regular file
			gitHeadPath := filepath.Join(gitPath, "HEAD")
			if headFi, err := os.Lstat(gitHeadPath); err == nil && headFi.Mode().IsRegular() {
				// Normal repo - .git/HEAD valid, Git won't fall back to cwd
				return false
			}
			// .git exists but no valid HEAD - fall through to bare-repo check
		} else if fi.Mode().IsRegular() {
			// worktree/submodule - Git follows the gitdir reference
			return false
		}
	}

	// No valid .git/HEAD found. Check if cwd has bare git repo indicators.
	// Check HEAD file
	if fi, err := os.Lstat(filepath.Join(cwd, "HEAD")); err == nil && fi.Mode().IsRegular() {
		return true
	}

	// Check objects/ directory
	if fi, err := os.Lstat(filepath.Join(cwd, "objects")); err == nil && fi.IsDir() {
		return true
	}

	// Check refs/ directory
	if fi, err := os.Lstat(filepath.Join(cwd, "refs")); err == nil && fi.IsDir() {
		return true
	}

	return false
}

// NormalizePathForComparison normalizes paths for comparison (Windows: case + backslash)
// Mirrors normalizePathForComparison in src/utils/file.ts
func NormalizePathForComparison(p string) string {
	p = filepath.Clean(p)
	if runtime.GOOS == "windows" {
		return strings.ToLower(strings.ReplaceAll(p, "/", `\`))
	}
	return p
}

// RealPathClean resolves symlinks and cleans path
func RealPathClean(p string) string {
	if r, err := filepath.EvalSymlinks(p); err == nil {
		return filepath.Clean(r)
	}
	return filepath.Clean(p)
}

// GitInsideWorkTree checks if we're inside a git work tree
func GitInsideWorkTree(cwd string) bool {
	git := GitExe()
	cmd := exec.Command(git, "rev-parse", "--is-inside-work-tree")
	cmd.Dir = cwd
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(strings.ToLower(string(output))) == "true"
}

// RunGitCommand runs a git command and returns output
func RunGitCommand(cwd string, args ...string) (string, error) {
	git := GitExe()
	cmd := exec.Command(git, args...)
	cmd.Dir = cwd
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// RunGitCommandsParallel runs multiple git commands in parallel
// Similar to Promise.all in TS getGitStatus
func RunGitCommandsParallel(cwd string, commands [][]string) ([]string, []error) {
	type result struct {
		index int
		value string
		err   error
	}

	results := make([]string, len(commands))
	errors := make([]error, len(commands))
	resultChan := make(chan result, len(commands))

	for i, args := range commands {
		go func(idx int, args []string) {
			value, err := RunGitCommand(cwd, args...)
			resultChan <- result{index: idx, value: value, err: err}
		}(i, args)
	}

	// Use a wait group to ensure all goroutines complete
	for i := 0; i < len(commands); i++ {
		res := <-resultChan
		results[res.index] = res.value
		errors[res.index] = res.err
	}

	return results, errors
}

// ExtractDefaultBranch extracts default branch from symbolic-ref output
func ExtractDefaultBranch(symbolicRef string) string {
	if symbolicRef == "" {
		return ""
	}
	// ref like "origin/main" → "main"
	if i := strings.LastIndex(symbolicRef, "/"); i >= 0 && i+1 < len(symbolicRef) {
		return symbolicRef[i+1:]
	}
	return symbolicRef
}

// CheckCdGitCompoundCommand checks for cd+git across pipe segments
// Security check to prevent bare repo fsmonitor bypass
func CheckCdGitCompoundCommand(segments []string) bool {
	hasCd := false
	hasGit := false

	for _, segment := range segments {
		// Use the same splitting logic as bash_security.go
		subcommands := splitCompoundCommandsSimple(segment)
		for _, sub := range subcommands {
			trimmed := strings.TrimSpace(sub)
			if IsNormalizedCdCommand(trimmed) {
				hasCd = true
			}
			if IsNormalizedGitCommand(trimmed) {
				hasGit = true
			}
		}
	}

	return hasCd && hasGit
}

// splitCompoundCommandsSimple splits by && and ; (simplified version)
func splitCompoundCommandsSimple(segment string) []string {
	var subcommands []string
	var current strings.Builder
	inSingleQuote := false
	inDoubleQuote := false

	runes := []rune(segment)
	for i := 0; i < len(runes); i++ {
		ch := runes[i]

		switch ch {
		case '\'':
			if !inDoubleQuote {
				inSingleQuote = !inSingleQuote
			}
			current.WriteRune(ch)
		case '"':
			if !inSingleQuote {
				inDoubleQuote = !inDoubleQuote
			}
			current.WriteRune(ch)
		case '&':
			// Check for &&
			if i+1 < len(runes) && runes[i+1] == '&' && !inSingleQuote && !inDoubleQuote {
				if current.Len() > 0 {
					subcommands = append(subcommands, current.String())
					current.Reset()
				}
				i++ // Skip next &
			} else {
				current.WriteRune(ch)
			}
		case ';':
			if !inSingleQuote && !inDoubleQuote {
				if current.Len() > 0 {
					subcommands = append(subcommands, current.String())
					current.Reset()
				}
			} else {
				current.WriteRune(ch)
			}
		default:
			current.WriteRune(ch)
		}
	}

	if current.Len() > 0 {
		subcommands = append(subcommands, current.String())
	}

	return subcommands
}

// IsLocalHost checks if host is localhost
func IsLocalHost(host string) bool {
	hostWithoutPort := strings.Split(host, ":")[0]
	return hostWithoutPort == "localhost" || regexp.MustCompile(`^127\.\d{1,3}\.\d{1,3}\.\d{1,3}$`).MatchString(hostWithoutPort)
}