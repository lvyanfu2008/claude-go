package claudemd

import "os"

// userMemoryEnabled combines isSettingSourceEnabled('userSettings') with optional hard opt-out env.
func userMemoryEnabled() bool {
	return IsSettingSourceEnabled(SourceUserSettings) && !truthy(os.Getenv("CLAUDE_CODE_DISABLE_USER_MEMORY"))
}

func projectMemoryEnabled() bool {
	return IsSettingSourceEnabled(SourceProjectSettings) && !truthy(os.Getenv("CLAUDE_CODE_DISABLE_PROJECT_MEMORY"))
}

func localMemoryEnabled() bool {
	return IsSettingSourceEnabled(SourceLocalSettings) && !truthy(os.Getenv("CLAUDE_CODE_DISABLE_LOCAL_MEMORY"))
}
