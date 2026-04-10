package querycontext

import (
	"os"
	"strings"
)

// BuildUserContext mirrors src/context.ts getUserContext (no memoization; host may cache).
func BuildUserContext(cwd string, extraClaudeMdRoots []string) (map[string]string, error) {
	shouldDisableClaudeMd := IsEnvTruthy(os.Getenv("CLAUDE_CODE_DISABLE_CLAUDE_MDS")) ||
		(BareModeFromEnv() && len(extraClaudeMdRoots) == 0)

	out := map[string]string{
		"currentDate": `Today's date is ` + LocalISODate() + `.`,
	}

	if shouldDisableClaudeMd {
		return out, nil
	}

	claudeMd, err := discoverClaudeMd(cwd, extraClaudeMdRoots)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(claudeMd) != "" {
		out["claudeMd"] = claudeMd
	}
	return out, nil
}
