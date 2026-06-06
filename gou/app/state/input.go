package state

import (
	"goc/gou/prompt"
	"goc/gou/suggestions"
	"goc/types"
)

type Input struct {
	PR                prompt.Model
	SlashCommands     []types.Command
	SlashCommandsOnce bool
	SlashPicker       interface{}
	SlashResultPanel  *string
	SuggestionEngine  *suggestions.SuggestionEngine
	Suggestions       []suggestions.ScoredItem
	SelectedSuggIdx   int
	SuggVisible       bool
	SkillListingSent  map[string]struct{}
}
