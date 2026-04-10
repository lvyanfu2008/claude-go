package claudeinit

import (
	"os"
	"strings"

	"goc/ccb-engine/settingsfile"
)

// applyExtraCACertsUserControlled mirrors TS [applyExtraCACertsFromConfig]: only
// ~/.claude/settings.json and ~/.claude.json env (user-controlled), not project files.
// Call before [settingsfile.EnsureProjectClaudeEnvOnce] so shell/parent NODE_EXTRA_CA_CERTS still wins.
func applyExtraCACertsUserControlled() error {
	if strings.TrimSpace(os.Getenv("NODE_EXTRA_CA_CERTS")) != "" {
		return nil
	}
	settingsEnv, err := settingsfile.ReadUserSettingsEnv()
	if err != nil {
		return err
	}
	globalEnv, err := settingsfile.ReadGlobalClaudeJSONEnv()
	if err != nil {
		return err
	}
	// TS: settingsEnv?.NODE_EXTRA_CA_CERTS || globalEnv?.NODE_EXTRA_CA_CERTS
	path := ""
	if settingsEnv != nil {
		path = strings.TrimSpace(settingsEnv["NODE_EXTRA_CA_CERTS"])
	}
	if path == "" && globalEnv != nil {
		path = strings.TrimSpace(globalEnv["NODE_EXTRA_CA_CERTS"])
	}
	if path == "" {
		return nil
	}
	return os.Setenv("NODE_EXTRA_CA_CERTS", path)
}
