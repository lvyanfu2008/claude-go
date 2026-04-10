// Data shapes used by src/commands.ts (getSkills, command list assembly). Command definitions live in command.go.
package types

// SkillsLoadResult mirrors the return type of getSkills() in src/commands.ts.
type SkillsLoadResult struct {
	SkillDirCommands    []Command `json:"skillDirCommands"`
	PluginSkills        []Command `json:"pluginSkills"`
	BundledSkills       []Command `json:"bundledSkills"`
	BuiltinPluginSkills []Command `json:"builtinPluginSkills"`
}
