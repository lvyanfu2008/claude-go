package commands

import (
	"context"
	"sync"

	"goc/ccb-engine/diaglog"
	"goc/types"
)

// SkillsBatch mirrors the return type of src/commands.ts async function getSkills(cwd).
type SkillsBatch struct {
	SkillDirCommands    []types.Command
	PluginSkills        []types.Command
	BundledSkills       []types.Command
	BuiltinPluginSkills []types.Command
}

// SkillsAsyncResult is the Promise-like outcome of [GetSkillsAsync] (single send, then channel close).
type SkillsAsyncResult struct {
	Batch SkillsBatch
	Err   error
}

func emptySkillsBatch() SkillsBatch {
	return SkillsBatch{
		SkillDirCommands:    make([]types.Command, 0),
		PluginSkills:        make([]types.Command, 0),
		BundledSkills:       make([]types.Command, 0),
		BuiltinPluginSkills: make([]types.Command, 0),
	}
}

// getSkillsWork is the synchronous body of getSkills (parallel skill dir + plugin skills, then bundled + builtin-plugin).
func getSkillsWork(ctx context.Context, cwd string, opts LoadOptions) SkillsBatch {
	if err := ctx.Err(); err != nil {
		return emptySkillsBatch()
	}

	var skillDir []types.Command
	var pluginSkills []types.Command
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		cmds, err := loadSkillDirCommands(cwd, opts)
		if err != nil {
			diaglog.Line("[goc/commands] GetSkills: skill directory commands failed to load, continuing without them: %v", err)
			skillDir = make([]types.Command, 0)
			return
		}
		if cmds == nil {
			skillDir = make([]types.Command, 0)
		} else {
			skillDir = cmds
		}
	}()

	go func() {
		defer wg.Done()
		cmds := loadPluginSkills()
		if cmds == nil {
			pluginSkills = make([]types.Command, 0)
		} else {
			pluginSkills = cmds
		}
	}()

	wg.Wait()

	bundled := loadBundledSkills()
	if bundled == nil {
		bundled = make([]types.Command, 0)
	}

	builtinPlugin, err := BuiltinPluginSkillCommands(cwd)
	if err != nil {
		diaglog.Line("[goc/commands] GetSkills: builtin plugin skills failed to load, continuing without them: %v", err)
		builtinPlugin = make([]types.Command, 0)
	}

	diaglog.Line("[goc/commands] GetSkills returning: %d skill dir commands, %d plugin skills, %d bundled skills, %d builtin plugin skills",
		len(skillDir), len(pluginSkills), len(bundled), len(builtinPlugin))

	return SkillsBatch{
		SkillDirCommands:    skillDir,
		PluginSkills:        pluginSkills,
		BundledSkills:       bundled,
		BuiltinPluginSkills: builtinPlugin,
	}
}

// GetSkillsAsync mirrors async getSkills: runs [getSkillsWork] in a goroutine and delivers one [SkillsAsyncResult] on a buffered channel (Promise-shaped API for Go).
func GetSkillsAsync(ctx context.Context, cwd string, opts LoadOptions) <-chan SkillsAsyncResult {
	ch := make(chan SkillsAsyncResult, 1)
	go func() {
		defer close(ch)
		if err := ctx.Err(); err != nil {
			ch <- SkillsAsyncResult{Err: err}
			return
		}
		ch <- SkillsAsyncResult{Batch: getSkillsWork(ctx, cwd, opts)}
	}()
	return ch
}

// GetSkills blocks on the same work as getSkills (sync convenience wrapper around [getSkillsWork]).
func GetSkills(ctx context.Context, cwd string, opts LoadOptions) SkillsBatch {
	return getSkillsWork(ctx, cwd, opts)
}
