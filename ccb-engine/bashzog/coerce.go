package bashzog

import (
	"encoding/json"
	"math"
	"regexp"
	"strconv"
	"strings"
)

var semanticNumberString = regexp.MustCompile(`^-?\d+(\.\d+)?$`)

// semanticNumber mirrors claude-code src/utils/semanticNumber.ts preprocess for JSON tool input.
func semanticNumber(v any) any {
	if s, ok := v.(string); ok && semanticNumberString.MatchString(strings.TrimSpace(s)) {
		n, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
		if err == nil && !math.IsNaN(n) && !math.IsInf(n, 0) {
			return n
		}
	}
	return v
}

// semanticBoolean mirrors claude-code src/utils/semanticBoolean.ts preprocess.
func semanticBoolean(v any) any {
	if s, ok := v.(string); ok {
		switch strings.TrimSpace(strings.ToLower(s)) {
		case "true":
			return true
		case "false":
			return false
		}
	}
	return v
}

// normalizeJSONValue applies semantic coercions suitable after json.Unmarshal(..., any).
func normalizeJSONValue(key string, v any) any {
	switch key {
	case "timeout":
		return semanticNumber(v)
	case "run_in_background", "dangerouslyDisableSandbox":
		return semanticBoolean(v)
	default:
		return v
	}
}

// parseTimeoutFloat returns timeout in ms if v is a finite JSON number.
func parseTimeoutFloat(v any) (float64, bool, error) {
	switch t := v.(type) {
	case float64:
		if math.IsNaN(t) || math.IsInf(t, 0) {
			return 0, false, errInvalidTimeout
		}
		return t, true, nil
	case int:
		return float64(t), true, nil
	case int64:
		return float64(t), true, nil
	case json.Number:
		f, err := t.Float64()
		if err != nil || math.IsNaN(f) || math.IsInf(f, 0) {
			return 0, false, errInvalidTimeout
		}
		return f, true, nil
	default:
		return 0, false, errInvalidTimeout
	}
}

var errInvalidTimeout = strError("bash: timeout must be a finite number")

type strError string

func (e strError) Error() string { return string(e) }
