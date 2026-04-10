package handwritten

// Builtin assembly still contains a /session row in the frozen JSON (TS loadAllCommands / COMMANDS() shape).
// Hosts must use commands.FilterGetCommands / LoadAndFilterCommands with GetCommandsAuth matching TS getCommands
// gates (IsRemoteMode for /session, subscriber flags, policy mirrors, etc.; see commands.IsCommandEnabledData).

import (
	"encoding/json"

	"goc/types"
)

var (
	builtinDefaultParsed []types.Command
	internalOnlyParsed   []types.Command
)

func init() {
	if err := json.Unmarshal([]byte(builtinCommandsDefaultJSON), &builtinDefaultParsed); err != nil {
		panic("handwritten: builtin default JSON: " + err.Error())
	}
	if err := json.Unmarshal([]byte(internalOnlyCommandsJSON), &internalOnlyParsed); err != nil {
		panic("handwritten: internal-only JSON: " + err.Error())
	}
}

func cloneCommands(in []types.Command) []types.Command {
	return append([]types.Command(nil), in...)
}

func indexFirstName(cmds []types.Command, name string) int {
	for i, c := range cmds {
		if c.Name == name {
			return i
		}
	}
	return -1
}
