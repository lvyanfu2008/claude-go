package bashtool

import "regexp"

// DestructivePattern represents a known destructive command pattern.
// Mirrors TS src/tools/BashTool/destructiveCommandWarning.ts
type DestructivePattern struct {
	Pattern *regexp.Regexp
	Warning string
}

// destructivePatterns holds the known destructive command patterns.
// These are informational only — they don't affect permissions.
var destructivePatterns = []DestructivePattern{
	// Git data loss
	{regexp.MustCompile(`git\s+reset\s+--hard`), "git reset --hard will discard all uncommitted changes."},
	{regexp.MustCompile(`git\s+push\s+--force`), "git push --force can overwrite remote history."},
	{regexp.MustCompile(`git\s+clean\s+-f`), "git clean -f will permanently delete untracked files."},
	{regexp.MustCompile(`git\s+checkout\s+\.`), "git checkout . will discard all local changes."},
	{regexp.MustCompile(`git\s+restore\s+\.`), "git restore . will discard all local changes."},
	{regexp.MustCompile(`git\s+stash\s+(drop|clear)`), "git stash drop/clear will permanently remove stashed changes."},
	{regexp.MustCompile(`git\s+branch\s+-D`), "git branch -D will force-delete the branch."},

	// Git safety bypass
	{regexp.MustCompile(`--no-verify`), "Using --no-verify bypasses pre-commit and pre-push hooks."},
	{regexp.MustCompile(`--no-gpg-sign`), "Using --no-gpg-sign bypasses commit signing."},

	// File deletion
	{regexp.MustCompile(`rm\s+-rf`), "rm -rf permanently deletes files and directories recursively."},
	{regexp.MustCompile(`rm\s+-r\s`), "rm -r recursively deletes directories."},
	{regexp.MustCompile(`rm\s+-f\s`), "rm -f forcibly deletes files without confirmation."},

	// Database destruction
	{regexp.MustCompile(`(?i)DROP\s+(TABLE|DATABASE|SCHEMA)`), "DROP TABLE/DATABASE/SCHEMA will permanently delete database objects."},
	{regexp.MustCompile(`(?i)TRUNCATE\s+(TABLE\s+)?`), "TRUNCATE will delete all rows from a table."},
	{regexp.MustCompile(`(?i)DELETE\s+FROM`), "DELETE FROM will remove rows. Verify the WHERE clause."},

	// Infrastructure
	{regexp.MustCompile(`kubectl\s+delete`), "kubectl delete will remove Kubernetes resources."},
	{regexp.MustCompile(`terraform\s+destroy`), "terraform destroy will destroy all managed infrastructure."},
}

// GetDestructiveCommandWarning checks a command against known destructive patterns.
// Returns the warning message if a destructive pattern is detected, or empty string.
func GetDestructiveCommandWarning(command string) string {
	for _, dp := range destructivePatterns {
		if dp.Pattern.MatchString(command) {
			return dp.Warning
		}
	}
	return ""
}
