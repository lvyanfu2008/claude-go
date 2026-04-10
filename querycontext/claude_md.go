package querycontext

import (
	"goc/claudemd"
)

// discoverClaudeMd uses goc/claudemd (TS getMemoryFiles + getClaudeMds parity).
func discoverClaudeMd(primaryCwd string, extraRoots []string) (string, error) {
	return claudemd.BuildClaudeMdString(claudemd.LoadOptions{
		OriginalCwd:           primaryCwd,
		AdditionalWorkingDirs: extraRoots,
	}), nil
}
