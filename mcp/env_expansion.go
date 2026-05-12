package mcp

import (
	"os"
	"regexp"
	"strings"
)

// envVarPattern matches ${VAR} or ${VAR:-default}.
var envVarPattern = regexp.MustCompile(`\$\{(\w+)(?::-([^}]*))?\}`)

// ExpandEnvVars expands ${VAR} and ${VAR:-default} patterns in a string.
// Mirrors TS services/mcp/envExpansion.ts.
func ExpandEnvVars(s string) string {
	return envVarPattern.ReplaceAllStringFunc(s, func(match string) string {
		parts := envVarPattern.FindStringSubmatch(match)
		if len(parts) < 2 {
			return match
		}
		name := parts[1]
		defaultVal := ""
		if len(parts) >= 3 {
			defaultVal = parts[2]
		}
		if val, ok := os.LookupEnv(name); ok && val != "" {
			return val
		}
		return defaultVal
	})
}

// ExpandMapEnvVars expands env vars in all values of a string map.
func ExpandMapEnvVars(m map[string]string) map[string]string {
	if m == nil {
		return nil
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = ExpandEnvVars(v)
	}
	return out
}

// ExpandSliceEnvVars expands env vars in all elements of a string slice.
func ExpandSliceEnvVars(s []string) []string {
	if s == nil {
		return nil
	}
	out := make([]string, len(s))
	for i, v := range s {
		out[i] = ExpandEnvVars(v)
	}
	return out
}

// ExpandStringMapWithEnv expands env vars in a nested map structure.
func ExpandStringMapWithEnv(m map[string]interface{}) map[string]interface{} {
	if m == nil {
		return nil
	}
	out := make(map[string]interface{}, len(m))
	for k, v := range m {
		switch vv := v.(type) {
		case string:
			out[k] = ExpandEnvVars(vv)
		case map[string]interface{}:
			out[k] = ExpandStringMapWithEnv(vv)
		case []interface{}:
			out[k] = expandSliceWithEnv(vv)
		default:
			out[k] = v
		}
	}
	return out
}

func expandSliceWithEnv(s []interface{}) []interface{} {
	out := make([]interface{}, len(s))
	for i, v := range s {
		switch vv := v.(type) {
		case string:
			out[i] = ExpandEnvVars(vv)
		case map[string]interface{}:
			out[i] = ExpandStringMapWithEnv(vv)
		default:
			out[i] = v
		}
	}
	return out
}

// expandEnvLiteral provides direct variable lookup (no default syntax).
func expandEnvLiteral(s string) string {
	return os.Expand(s, func(key string) string {
		return os.Getenv(key)
	})
}

// HasEnvVarRef returns true if the string contains any ${...} reference.
func HasEnvVarRef(s string) bool {
	return strings.Contains(s, "${")
}
