// Package mcpcommands implements scheme-2 R0/R1: load MCP prompt-style commands from JSON (bridge / snapshot)
// without running an MCP host. Shape matches TypeScript Command after prompts/list + fetchCommandsForClient mapping.
//
// Contract:
//   - File: JSON array of [types.Command], each suitable for Skill merge (type "prompt", loadedFrom "mcp" for skill listing).
//   - Env: GOU_DEMO_MCP_COMMANDS_JSON — absolute or relative path to that file; unset = no load.
//   - Flag: gou-demo -mcp-commands-json=path (sets effective path over env; see [pui.DemoConfig.MCPCommandsJSONPath]).
//
// MCP **tool definitions** (for assembleToolPool / Options.Tools): see [EnvToolsJSONPath], [LoadToolsFromPath] in tools_load.go
// and gou-demo -mcp-tools-json (requires GOU_DEMO_USE_EMBEDDED_TOOLS_API=1 for [pui.BuildDemoParams]).
//
// Path resolution: tries the path as given, then cwd-relative variants. If the path starts with "goc/"
// (repo-root style) but the process cwd is already .../goc, the first attempt fails; [resolveMCPCommandsPath]
// retries after stripping that prefix so repo-root-style paths still resolve under `cd goc`.
package mcpcommands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"goc/types"
)

// EnvCommandsJSONPath is the environment variable for the MCP commands snapshot file (R0).
const EnvCommandsJSONPath = "GOU_DEMO_MCP_COMMANDS_JSON"

func resolveMCPCommandsPath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", nil
	}
	cwd, errWd := os.Getwd()
	var candidates []string
	add := func(s string) {
		s = filepath.Clean(s)
		for _, existing := range candidates {
			if existing == s {
				return
			}
		}
		candidates = append(candidates, s)
	}
	add(path)
	if errWd == nil {
		if !filepath.IsAbs(path) {
			add(filepath.Join(cwd, path))
		}
		slash := filepath.ToSlash(path)
		for _, prefix := range []string{"goc/", "./goc/"} {
			if after, ok := strings.CutPrefix(slash, prefix); ok && after != "" {
				add(filepath.Join(cwd, filepath.FromSlash(after)))
				break
			}
		}
	}
	for _, c := range candidates {
		st, err := os.Stat(c)
		if err != nil {
			continue
		}
		if st.IsDir() {
			continue
		}
		return c, nil
	}
	if errWd != nil {
		return "", fmt.Errorf("mcpcommands: file not found %q (tried %v)", path, candidates)
	}
	return "", fmt.Errorf("mcpcommands: file not found %q (cwd=%q, tried %v)", path, cwd, candidates)
}

// LoadFromPath reads a JSON array of [types.Command]. Empty path returns (nil, nil).
func LoadFromPath(path string) ([]types.Command, error) {
	resolved, err := resolveMCPCommandsPath(path)
	if err != nil {
		return nil, err
	}
	if resolved == "" {
		return nil, nil
	}
	data, err := os.ReadFile(resolved)
	if err != nil {
		return nil, fmt.Errorf("mcpcommands: read %q: %w", resolved, err)
	}
	var out []types.Command
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("mcpcommands: parse %q: %w", resolved, err)
	}
	return out, nil
}

// LoadFromEnv loads from os.Getenv(EnvCommandsJSONPath). Missing or empty env → (nil, nil).
func LoadFromEnv() ([]types.Command, error) {
	return LoadFromPath(os.Getenv(EnvCommandsJSONPath))
}
