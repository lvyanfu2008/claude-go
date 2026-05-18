package bashtool

import (
	"context"
	"encoding/json"
	"os"
	"strings"

	"goc/tools/localtools"
)

// BashSecurityEnabled checks whether the CLAUDE_CODE_BASH_SECURITY env var is set.
func BashSecurityEnabled() bool {
	return isTruthy("CLAUDE_CODE_BASH_SECURITY")
}

// isTruthy reports whether an env var is set to a truthy value.
func isTruthy(key string) bool {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return false
	}
	switch strings.ToLower(v) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

// ValidateBashCommand runs all security validators against a command.
// Returns the aggregated validation result.
func ValidateBashCommand(command string) *ValidationResult {
	return RunAllValidators(command)
}

// RunBashWithSecurity validates a bash command and executes it if safe.
// Returns the tool output JSON string, isError flag, and error.
func RunBashWithSecurity(ctx context.Context, input json.RawMessage, workDir, tasksDir string, localBashDefault bool) (string, bool, error) {
	if BashSecurityEnabled() {
		var in struct {
			Command string `json:"command"`
		}
		if err := json.Unmarshal(input, &in); err == nil && strings.TrimSpace(in.Command) != "" {
			if result := ValidateBashSecurity(in.Command); result != nil {
				// Return a safe error message in tool result format.
				out := map[string]any{
					"data": map[string]any{
						"stdout": "",
						"stderr": "Bash security validation failed: " + result.Reason,
						"exitCode": 1,
					},
				}
				b, _ := json.Marshal(out)
				return string(b), true, nil
			}
		}
	}

	// Delegate to the existing localtools Bash execution.
	return localtools.BashFromJSON(ctx, input, workDir, localBashDefault, tasksDir)
}
