package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"goc/claudemd"
)

func TestBuildAutoMemoryPrompt_disabledByRemoteWithoutMemoryDir(t *testing.T) {
	t.Setenv("CLAUDE_CODE_REMOTE", "1")
	t.Setenv("CLAUDE_CODE_REMOTE_MEMORY_DIR", "")
	t.Setenv("CLAUDE_CODE_DISABLE_AUTO_MEMORY", "")
	dir := t.TempDir()
	s := BuildAutoMemoryPrompt(GouDemoSystemOpts{Cwd: dir})
	if s != "" {
		t.Fatalf("expected empty when remote without memory dir, got len=%d", len(s))
	}
}

func TestBuildAutoMemoryPrompt_whenEnabled(t *testing.T) {
	t.Setenv("CLAUDE_CODE_REMOTE", "")
	t.Setenv("CLAUDE_CODE_DISABLE_AUTO_MEMORY", "")
	t.Setenv("CLAUDE_CODE_SIMPLE", "")
	t.Setenv("CLAUDE_CODE_AUTO_MEMORY_ENABLED", "1")
	cfgHome := filepath.Join(t.TempDir(), "claudecfg")
	t.Setenv("CLAUDE_CONFIG_DIR", cfgHome)
	dir := t.TempDir()
	s := BuildAutoMemoryPrompt(GouDemoSystemOpts{Cwd: dir})
	if !strings.Contains(s, "# auto memory") || !strings.Contains(s, "## Types of memory") {
		t.Fatalf("unexpected prompt head: %q", s[:min(120, len(s))])
	}
	if !strings.Contains(s, "MEMORY.md") {
		t.Fatal("expected MEMORY.md index instructions")
	}
	if strings.Contains(s, "/__MEMORY_DIR__/") {
		t.Fatal("placeholder not replaced")
	}
}

func TestBuildAutoMemoryPrompt_searchingPastContext(t *testing.T) {
	t.Setenv("CLAUDE_CODE_REMOTE", "")
	t.Setenv("CLAUDE_CODE_DISABLE_AUTO_MEMORY", "")
	t.Setenv("CLAUDE_CODE_AUTO_MEMORY_ENABLED", "1")
	cfgHome := filepath.Join(t.TempDir(), "claudecfg")
	t.Setenv("CLAUDE_CONFIG_DIR", cfgHome)
	dir := t.TempDir()
	s := BuildAutoMemoryPrompt(GouDemoSystemOpts{
		Cwd:                     dir,
		MemorySearchPastContext: true,
	})
	if !strings.Contains(s, "## Searching past context") {
		t.Fatal(s[:min(200, len(s))])
	}
}

func TestBuildAutoMemoryPrompt_kairosBranch(t *testing.T) {
	t.Setenv("FEATURE_KAIROS", "1")
	t.Setenv("CLAUDE_CODE_GO_KAIROS_ACTIVE", "1")
	t.Setenv("CLAUDE_CODE_REMOTE", "")
	t.Setenv("CLAUDE_CODE_DISABLE_AUTO_MEMORY", "")
	t.Setenv("CLAUDE_CODE_SIMPLE", "")
	t.Setenv("CLAUDE_CODE_AUTO_MEMORY_ENABLED", "1")
	cfgHome := filepath.Join(t.TempDir(), "claudecfg")
	t.Setenv("CLAUDE_CONFIG_DIR", cfgHome)
	dir := t.TempDir()
	s := BuildAutoMemoryPrompt(GouDemoSystemOpts{Cwd: dir, KairosActive: true})
	if !strings.Contains(s, "append-only") || !strings.Contains(s, "## What to log") {
		t.Fatalf("expected KAIROS daily-log shape: %q", s[:min(200, len(s))])
	}
	if strings.Contains(s, "## Types of memory") {
		t.Fatal("individual auto-memory section should not appear in KAIROS mode")
	}
}

func TestBuildAutoMemoryPrompt_kairosTakesPrecedenceOverTeam(t *testing.T) {
	t.Setenv("FEATURE_KAIROS", "1")
	t.Setenv("FEATURE_TEAMMEM", "1")
	t.Setenv("CLAUDE_CODE_GO_KAIROS_ACTIVE", "1")
	t.Setenv("CLAUDE_CODE_REMOTE", "")
	t.Setenv("CLAUDE_CODE_DISABLE_AUTO_MEMORY", "")
	t.Setenv("CLAUDE_CODE_AUTO_MEMORY_ENABLED", "1")
	cfgHome := filepath.Join(t.TempDir(), "claudecfg")
	t.Setenv("CLAUDE_CONFIG_DIR", cfgHome)
	dir := t.TempDir()
	s := BuildAutoMemoryPrompt(GouDemoSystemOpts{Cwd: dir, KairosActive: true})
	if !strings.Contains(s, "append-only") {
		t.Fatal("expected KAIROS prompt when both KAIROS and TEAMMEM are on")
	}
}

func TestBuildAutoMemoryPrompt_teamCombinedBranch(t *testing.T) {
	t.Setenv("FEATURE_TEAMMEM", "1")
	t.Setenv("CLAUDE_CODE_REMOTE", "")
	t.Setenv("CLAUDE_CODE_DISABLE_AUTO_MEMORY", "")
	t.Setenv("CLAUDE_CODE_AUTO_MEMORY_ENABLED", "1")
	cfgHome := filepath.Join(t.TempDir(), "claudecfg")
	t.Setenv("CLAUDE_CONFIG_DIR", cfgHome)
	dir := t.TempDir()
	s := BuildAutoMemoryPrompt(GouDemoSystemOpts{Cwd: dir})
	if !strings.HasPrefix(strings.TrimSpace(s), "# Memory") {
		t.Fatalf("expected combined team header: %q", s[:min(80, len(s))])
	}
	if !strings.Contains(s, "shared team directory") {
		t.Fatal(s[:min(200, len(s))])
	}
	teamDir := strings.TrimSuffix(claudemd.GetTeamMemPath(dir), string(filepath.Separator))
	if st, err := os.Stat(teamDir); err != nil || !st.IsDir() {
		t.Fatalf("EnsureMemoryDirExists(team): stat %q: %v isDir=%v", teamDir, err, st != nil && st.IsDir())
	}
}
