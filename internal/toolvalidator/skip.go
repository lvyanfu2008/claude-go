package toolvalidator

import (
	"os"
	"strings"
)

// SkipValidation returns true when CCB_ENGINE_SKIP_TOOL_INPUT_SCHEMA=1 (same semantics as toolinput).
func SkipValidation() bool {
	return strings.TrimSpace(os.Getenv("CCB_ENGINE_SKIP_TOOL_INPUT_SCHEMA")) == "1"
}
