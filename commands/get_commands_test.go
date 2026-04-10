package commands

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"goc/types"
)

func TestFilterGetCommands_DropsClaudeAIOnlyWhenNotSubscriber(t *testing.T) {
	claudeAI := types.CommandAvailabilityClaudeAI
	cmd := types.Command{
		CommandBase: types.CommandBase{
			Name:         "sub-only",
			Description:  "x",
			Availability: []types.CommandAvailability{claudeAI},
		},
		Type: "prompt",
	}
	auth := GetCommandsAuth{IsClaudeAISubscriber: false, IsUsing3PServices: false, IsFirstPartyAnthropicBaseURL: true}
	out := FilterGetCommands([]types.Command{cmd}, auth)
	if len(out) != 0 {
		t.Fatalf("expected drop, got %d", len(out))
	}
	auth2 := GetCommandsAuth{IsClaudeAISubscriber: true, IsUsing3PServices: false, IsFirstPartyAnthropicBaseURL: true}
	out2 := FilterGetCommands([]types.Command{cmd}, auth2)
	if len(out2) != 1 {
		t.Fatalf("expected keep for subscriber, got %d", len(out2))
	}
}

func TestFilterGetCommands_KeepsEmptyAvailability(t *testing.T) {
	cmd := types.Command{
		CommandBase: types.CommandBase{Name: "u", Description: "d"},
		Type:        "local-jsx",
	}
	auth := GetCommandsAuth{}
	out := FilterGetCommands([]types.Command{cmd}, auth)
	if len(out) != 1 || out[0].Name != "u" {
		t.Fatalf("got %#v", out)
	}
}

func TestFilterGetCommands_FastDisabledByEnv(t *testing.T) {
	t.Setenv("CLAUDE_CODE_DISABLE_FAST_MODE", "1")
	fast := types.Command{
		CommandBase: types.CommandBase{
			Name:        "fast",
			Description: "Toggle fast mode",
		},
		Type: "local-jsx",
	}
	auth := GetCommandsAuth{IsFirstPartyAnthropicBaseURL: true}
	out := FilterGetCommands([]types.Command{fast}, auth)
	if len(out) != 0 {
		t.Fatalf("expected fast dropped when CLAUDE_CODE_DISABLE_FAST_MODE=1, got %#v", out)
	}
}

func TestFilterGetCommands_SessionRequiresRemoteMode(t *testing.T) {
	session := types.Command{
		CommandBase: types.CommandBase{Name: "session", Description: "Show remote session URL and QR code"},
		Type:        "local-jsx",
	}
	authLocal := GetCommandsAuth{IsFirstPartyAnthropicBaseURL: true}
	out := FilterGetCommands([]types.Command{session}, authLocal)
	if len(out) != 0 {
		t.Fatalf("expected session dropped when not remote, got %#v", out)
	}
	authRemote := GetCommandsAuth{IsFirstPartyAnthropicBaseURL: true, IsRemoteMode: true}
	out2 := FilterGetCommands([]types.Command{session}, authRemote)
	if len(out2) != 1 || out2[0].Name != "session" {
		t.Fatalf("expected session kept in remote mode, got %#v", out2)
	}
}

func TestLoadAndFilterCommands_IncludesBuiltins(t *testing.T) {
	ClearLoadAllCommandsCache()
	tmp := t.TempDir()
	f := false
	all, err := LoadAndFilterCommands(context.Background(), tmp, LoadOptions{BareMode: &f}, DefaultConsoleAPIAuth())
	if err != nil {
		t.Fatal(err)
	}
	if len(all) < 40 {
		t.Fatalf("expected filtered builtins+empty skills, got %d", len(all))
	}
}

func TestLoadAndFilterCommands_OmitsSessionUnlessRemoteMode(t *testing.T) {
	ClearLoadAllCommandsCache()
	tmp := t.TempDir()
	f := false
	auth := DefaultConsoleAPIAuth()
	all, err := LoadAndFilterCommands(context.Background(), tmp, LoadOptions{BareMode: &f}, auth)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range all {
		if c.Name == "session" {
			t.Fatal("expected /session omitted when IsRemoteMode is false (TS getCommands + session.isEnabled)")
		}
	}
	authRemote := auth
	authRemote.IsRemoteMode = true
	ClearLoadAllCommandsCache()
	remoteList, err := LoadAndFilterCommands(context.Background(), tmp, LoadOptions{BareMode: &f}, authRemote)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, c := range remoteList {
		if c.Name == "session" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected /session when IsRemoteMode is true")
	}
}

func TestInsertDynamicSkillsBeforeBuiltins(t *testing.T) {
	builtinNames := map[string]struct{}{"builtinA": {}, "builtinB": {}}
	base := []types.Command{
		{CommandBase: types.CommandBase{Name: "skillX", Description: "1"}, Type: "prompt"},
		{CommandBase: types.CommandBase{Name: "builtinA", Description: "2"}, Type: "local-jsx"},
		{CommandBase: types.CommandBase{Name: "builtinB", Description: "3"}, Type: "local-jsx"},
	}
	dyn := []types.Command{
		{CommandBase: types.CommandBase{Name: "dyn1", Description: "d"}, Type: "prompt"},
	}
	out := InsertDynamicSkillsBeforeBuiltins(base, dyn, builtinNames)
	if len(out) != 4 {
		t.Fatalf("len=%d want 4: %#v", len(out), namesOf(out))
	}
	if out[0].Name != "skillX" || out[1].Name != "dyn1" || out[2].Name != "builtinA" {
		t.Fatalf("order: %v", namesOf(out))
	}
}

func TestInsertDynamicSkillsBeforeBuiltins_AppendWhenNoBuiltinInBase(t *testing.T) {
	builtinNames := map[string]struct{}{"onlyBuiltin": {}}
	base := []types.Command{
		{CommandBase: types.CommandBase{Name: "orphan", Description: "1"}, Type: "prompt"},
	}
	dyn := []types.Command{
		{CommandBase: types.CommandBase{Name: "dyn1", Description: "d"}, Type: "prompt"},
	}
	out := InsertDynamicSkillsBeforeBuiltins(base, dyn, builtinNames)
	if len(out) != 2 || out[0].Name != "orphan" || out[1].Name != "dyn1" {
		t.Fatalf("got %#v", namesOf(out))
	}
}

func TestUniqueDynamicSkillsForGetCommands_SkipsNamePresentInBase(t *testing.T) {
	base := []types.Command{
		{CommandBase: types.CommandBase{Name: "same", Description: "b"}, Type: "prompt"},
	}
	dynamic := []types.Command{
		{CommandBase: types.CommandBase{Name: "same", Description: "d"}, Type: "prompt"},
		{CommandBase: types.CommandBase{Name: "new", Description: "n"}, Type: "prompt"},
	}
	auth := DefaultConsoleAPIAuth()
	out := UniqueDynamicSkillsForGetCommands(dynamic, base, auth)
	if len(out) != 1 || out[0].Name != "new" {
		t.Fatalf("got %#v", out)
	}
}

func TestGetCommands_includesSessionDynamicSkills(t *testing.T) {
	ClearLoadAllCommandsCache()
	tmp := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(tmp, "cfg"))
	if err := os.MkdirAll(filepath.Join(tmp, "cfg", "skills"), 0o755); err != nil {
		t.Fatal(err)
	}
	repo := filepath.Join(tmp, "repo")
	pkg := filepath.Join(repo, "src", "pkg")
	skillsRoot := filepath.Join(pkg, ".claude", "skills")
	skillDir := filepath.Join(skillsRoot, "sessdyn")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\ndescription: from session\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	f := false
	opts := LoadOptions{BareMode: &f}
	if err := AddSkillDirectories([]string{skillsRoot}, opts); err != nil {
		t.Fatal(err)
	}
	out, err := GetCommands(context.Background(), repo, opts, DefaultConsoleAPIAuth())
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, c := range out {
		if c.Name == "sessdyn" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected GetCommands to merge session dynamic skills (TS getDynamicSkills + getCommands)")
	}
}

func TestGetCommandsWithDynamicSkills_UsesBuiltinBoundaryFromEmbed(t *testing.T) {
	ClearLoadAllCommandsCache()
	tmp := t.TempDir()
	f := false
	base, err := LoadAndFilterCommands(context.Background(), tmp, LoadOptions{BareMode: &f}, DefaultConsoleAPIAuth())
	if err != nil {
		t.Fatal(err)
	}
	idx := indexFirstBuiltinCommandInLoadAllOrder(base, BuiltinCommandNameSet())
	if idx < 0 {
		t.Fatal("no builtin boundary in filtered list")
	}
	dyn := []types.Command{
		{CommandBase: types.CommandBase{Name: "sessionDyn", Description: "x"}, Type: "prompt"},
	}
	out := GetCommandsWithDynamicSkills(base, dyn, DefaultConsoleAPIAuth())
	di := -1
	for i, c := range out {
		if c.Name == "sessionDyn" {
			di = i
			break
		}
	}
	if di < 0 {
		t.Fatal("missing dynamic")
	}
	// dynamic sits immediately before first builtin name from COMMANDS()
	if di != idx {
		t.Fatalf("want dynamic@%d before builtin@%d, got dynamic@%d len=%d", idx, idx, di, len(out))
	}
}

func namesOf(cmds []types.Command) []string {
	s := make([]string, len(cmds))
	for i, c := range cmds {
		s[i] = c.Name
	}
	return s
}
