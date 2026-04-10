package claudeinit

import (
	"os"
	"sort"
	"strings"
)

const dumpSchema = "claude-init-dump/v1"

// DumpV1 is JSON emitted by [DumpState] and TS [scripts/dump-init-state.ts].
type DumpV1 struct {
	Schema              string            `json:"schema"`
	Runtime             string            `json:"runtime"`
	Cwd                 string            `json:"cwd"`
	ProjectRoot         string            `json:"projectRoot"`
	GitHubRepo          *string           `json:"gitHubRepo"`
	NodeExtraCaCertsSet bool              `json:"nodeExtraCaCertsSet"`
	EnvAllowlisted      map[string]string `json:"envAllowlisted"`
}

// DumpState builds a redacted snapshot for parity harness (sorted env keys).
func DumpState() DumpV1 {
	cwd, _ := os.Getwd()
	nodeExtra := strings.TrimSpace(os.Getenv("NODE_EXTRA_CA_CERTS")) != ""
	var gh *string
	if s := CachedGitHubRepo(); s != "" {
		gh = &s
	}
	env := map[string]string{}
	for _, e := range os.Environ() {
		k, v, ok := strings.Cut(e, "=")
		if !ok {
			continue
		}
		if allowDumpEnvKey(k) {
			env[k] = v
		}
	}
	return DumpV1{
		Schema:              dumpSchema,
		Runtime:             "go",
		Cwd:                 cwd,
		ProjectRoot:         ProjectRoot(),
		GitHubRepo:          gh,
		NodeExtraCaCertsSet: nodeExtra,
		EnvAllowlisted:      sortedStringMap(env),
	}
}

func sortedStringMap(m map[string]string) map[string]string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make(map[string]string, len(keys))
	for _, k := range keys {
		out[k] = m[k]
	}
	return out
}

func allowDumpEnvKey(k string) bool {
	if strings.HasPrefix(k, "GOU_DEMO_") || strings.HasPrefix(k, "CCB_") {
		return true
	}
	if strings.HasPrefix(k, "FEATURE_") {
		return true
	}
	if strings.HasPrefix(k, "CLAUDE_CODE_LOG_") || strings.HasPrefix(k, "CLAUDE_CODE_DEBUG_") {
		return true
	}
	switch k {
	case "CLAUDE_CODE_SIMPLE", "CLAUDE_CODE_REMOTE", "CLAUDE_CODE_GO_INIT_WALL_MS":
		return true
	}
	// Never dump obvious secret-bearing keys even if prefix matches elsewhere
	if strings.Contains(strings.ToUpper(k), "API_KEY") || strings.Contains(strings.ToUpper(k), "AUTH_TOKEN") || strings.Contains(strings.ToUpper(k), "SECRET") {
		return false
	}
	return false
}
