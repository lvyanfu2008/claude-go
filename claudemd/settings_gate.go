package claudemd

import (
	"os"

	"goc/claudebase"
)

// userMemoryEnabled combines isSettingSourceEnabled('userSettings') with optional hard opt-out env.
func userMemoryEnabled() bool {
	return IsSettingSourceEnabled(SourceUserSettings) && !claudebase.Truthy(os.Getenv("CLAUDE_CODE_DISABLE_USER_MEMORY"))
}

func projectMemoryEnabled() bool {
	return IsSettingSourceEnabled(SourceProjectSettings) && !claudebase.Truthy(os.Getenv("CLAUDE_CODE_DISABLE_PROJECT_MEMORY"))
}

func localMemoryEnabled() bool {
	return IsSettingSourceEnabled(SourceLocalSettings) && !claudebase.Truthy(os.Getenv("CLAUDE_CODE_DISABLE_LOCAL_MEMORY"))
}
