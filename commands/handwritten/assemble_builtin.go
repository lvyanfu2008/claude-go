package handwritten

import (
	"goc/commands/featuregates"
	"goc/types"
)

// AssembleBuiltinCommands mirrors src/commands.ts COMMANDS() membership order for listing metadata.
func AssembleBuiltinCommands() []types.Command {
	def := builtinDefaultParsed
	iThink := indexFirstName(def, "think-back")
	iLogout := indexFirstName(def, "logout")
	if iThink < 0 || iLogout < 0 || iLogout+2 > len(def) {
		panic("handwritten: builtin default snapshot missing think-back/logout/login tail")
	}

	pre := def[:iThink]
	mid := def[iThink:iLogout]
	auth := def[iLogout : iLogout+2]
	tail := def[iLogout+2:]

	out := make([]types.Command, 0, len(def)+48)
	out = append(out, pre...)
	out = append(out, optionalBuiltinAfterVim()...)
	out = append(out, mid...)
	if !featuregates.IsUsing3PServicesFromEnv() {
		out = append(out, auth...)
	}
	if len(tail) >= 1 {
		out = append(out, tail[0])
	}
	out = append(out, optionalBuiltinPeers()...)
	if len(tail) >= 2 {
		out = append(out, tail[1])
	}
	out = append(out, optionalBuiltinWorkflows()...)
	out = append(out, optionalBuiltinTorch()...)
	if featuregates.UserTypeAnt() && !featuregates.IsDemo() {
		out = append(out, cloneCommands(internalOnlyParsed)...)
	}
	return out
}
