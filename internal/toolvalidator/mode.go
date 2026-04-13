package toolvalidator

import (
	"os"
	"strings"
)

// EnvToolInputValidator is the env name for switching input validation implementation.
// Values: empty or "jsonschema" (default) — embedded tools_api.json + jsonschema + toolrefine.
// "zog" — use Zog for tools registered in zoglayer; others fall back to the jsonschema path.
const EnvToolInputValidator = "GO_TOOL_INPUT_VALIDATOR"

// InputValidatorMode returns "jsonschema" or "zog".
func InputValidatorMode() string {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(EnvToolInputValidator)))
	switch v {
	case "zog":
		return "zog"
	default:
		return "jsonschema"
	}
}
