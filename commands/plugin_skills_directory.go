package commands

import (
	"context"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"goc/ccb-engine/diaglog"
	"goc/types"
)

// LoadSkillsFromDirectory mirrors src/utils/plugins/loadPluginCommands.ts loadSkillsFromDirectory:
// either skillsPath/SKILL.md (direct skill folder) or subdirectories each containing SKILL.md.
// loadedPaths dedupes by resolved real path (TS isDuplicatePath). pluginManifest may be nil/empty → {}.
func LoadSkillsFromDirectory(
	ctx context.Context,
	skillsPath string,
	pluginName string,
	sourceName string,
	pluginPath string,
	pluginManifest json.RawMessage,
	loadedPaths map[string]struct{},
) ([]types.Command, error) {
	if loadedPaths == nil {
		loadedPaths = make(map[string]struct{})
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	var skills []types.Command
	var dedupMu sync.Mutex

	directSkillPath := filepath.Join(skillsPath, "SKILL.md")
	directBody, err := os.ReadFile(directSkillPath)
	if err != nil && !os.IsNotExist(err) {
		diaglog.Line("[goc/commands] LoadSkillsFromDirectory: failed to read %s: %v", directSkillPath, err)
		return skills, nil
	}
	if err == nil {
		dedupMu.Lock()
		dup := isPluginSkillDuplicatePath(directSkillPath, loadedPaths)
		dedupMu.Unlock()
		if dup {
			return skills, nil
		}
		skillName := pluginName + ":" + filepath.Base(skillsPath)
		skillRoot := filepath.Dir(directSkillPath)
		cmd, err := commandFromPluginSkill(skillName, skillRoot, directSkillPath, directBody, sourceName, pluginPath, pluginManifest)
		if err != nil {
			diaglog.Line("[goc/commands] LoadSkillsFromDirectory: %v", err)
			return skills, nil
		}
		skills = append(skills, cmd)
		return skills, nil
	}

	entries, err := os.ReadDir(skillsPath)
	if err != nil {
		if !os.IsNotExist(err) {
			diaglog.Line("[goc/commands] LoadSkillsFromDirectory: readdir %s: %v", skillsPath, err)
		}
		return skills, nil
	}

	var skillsMu sync.Mutex
	var wg sync.WaitGroup

	for _, ent := range entries {
		if err := ctx.Err(); err != nil {
			wg.Wait()
			return nil, err
		}
		skillDirPath := filepath.Join(skillsPath, ent.Name())
		if !ent.IsDir() {
			if ent.Type()&fs.ModeSymlink == 0 {
				continue
			}
			st, err := os.Stat(skillDirPath)
			if err != nil || !st.IsDir() {
				continue
			}
		}

		entryName := ent.Name()
		skillFilePath := filepath.Join(skillDirPath, "SKILL.md")
		wg.Add(1)
		go func(entryName, skillFilePath, skillDirPath string) {
			defer wg.Done()
			if err := ctx.Err(); err != nil {
				return
			}
			content, err := os.ReadFile(skillFilePath)
			if err != nil {
				if !os.IsNotExist(err) {
					diaglog.Line("[goc/commands] LoadSkillsFromDirectory: read %s: %v", skillFilePath, err)
				}
				return
			}
			dedupMu.Lock()
			dup := isPluginSkillDuplicatePath(skillFilePath, loadedPaths)
			dedupMu.Unlock()
			if dup {
				return
			}
			skillName := pluginName + ":" + entryName
			cmd, err := commandFromPluginSkill(skillName, skillDirPath, skillFilePath, content, sourceName, pluginPath, pluginManifest)
			if err != nil {
				diaglog.Line("[goc/commands] LoadSkillsFromDirectory: %s: %v", skillFilePath, err)
				return
			}
			skillsMu.Lock()
			skills = append(skills, cmd)
			skillsMu.Unlock()
		}(entryName, skillFilePath, skillDirPath)
	}

	wg.Wait()

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	return skills, nil
}

// LoadSkillsFromDirectoryAsyncResult is the Promise-like outcome of [LoadSkillsFromDirectoryAsync].
type LoadSkillsFromDirectoryAsyncResult struct {
	Commands []types.Command
	Err      error
}

// LoadSkillsFromDirectoryAsync runs [LoadSkillsFromDirectory] in a goroutine (TS async loadSkillsFromDirectory).
func LoadSkillsFromDirectoryAsync(
	ctx context.Context,
	skillsPath string,
	pluginName string,
	sourceName string,
	pluginPath string,
	pluginManifest json.RawMessage,
	loadedPaths map[string]struct{},
) <-chan LoadSkillsFromDirectoryAsyncResult {
	ch := make(chan LoadSkillsFromDirectoryAsyncResult, 1)
	go func() {
		defer close(ch)
		cmds, err := LoadSkillsFromDirectory(ctx, skillsPath, pluginName, sourceName, pluginPath, pluginManifest, loadedPaths)
		ch <- LoadSkillsFromDirectoryAsyncResult{Commands: cmds, Err: err}
	}()
	return ch
}

func resolvePathForPluginDedup(p string) string {
	abs, err := filepath.Abs(p)
	if err != nil {
		abs = p
	}
	if r, err := filepath.EvalSymlinks(abs); err == nil {
		return filepath.Clean(r)
	}
	return filepath.Clean(abs)
}

// isPluginSkillDuplicatePath mirrors isDuplicatePath(fs, filePath, loadedPaths): true = skip (already seen).
func isPluginSkillDuplicatePath(markdownPath string, loadedPaths map[string]struct{}) bool {
	key := resolvePathForPluginDedup(markdownPath)
	if _, ok := loadedPaths[key]; ok {
		return true
	}
	loadedPaths[key] = struct{}{}
	return false
}

func normalizePluginManifest(m json.RawMessage) json.RawMessage {
	if len(strings.TrimSpace(string(m))) == 0 {
		return json.RawMessage(`{}`)
	}
	return m
}

func substituteCLAUDEPluginRoot(s, pluginPath string) string {
	if pluginPath == "" {
		return s
	}
	root := filepath.ToSlash(filepath.Clean(pluginPath))
	return strings.ReplaceAll(s, "${CLAUDE_PLUGIN_ROOT}", root)
}

func commandFromPluginSkill(
	skillName string,
	skillRoot string,
	markdownPath string,
	body []byte,
	repository string,
	pluginPath string,
	pluginManifest json.RawMessage,
) (types.Command, error) {
	cmd, err := commandFromSkillMarkdown(skillName, skillRoot, markdownPath, body, repository, "plugin", "Plugin skill")
	if err != nil {
		return types.Command{}, err
	}
	pluginSrc := "plugin"
	cmd.Source = &pluginSrc
	loadPm := "loading"
	cmd.ProgressMessage = &loadPm
	if len(cmd.AllowedTools) > 0 {
		for i := range cmd.AllowedTools {
			cmd.AllowedTools[i] = substituteCLAUDEPluginRoot(cmd.AllowedTools[i], pluginPath)
		}
	}
	type pluginInfoShape struct {
		PluginManifest json.RawMessage `json:"pluginManifest"`
		Repository     string          `json:"repository"`
	}
	raw, err := json.Marshal(pluginInfoShape{
		PluginManifest: normalizePluginManifest(pluginManifest),
		Repository:     repository,
	})
	if err != nil {
		return types.Command{}, err
	}
	cmd.PluginInfo = raw
	return cmd, nil
}
