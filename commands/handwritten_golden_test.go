package commands

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"

	"goc/commands/handwritten"
	"goc/types"
)

func readTestdata(t *testing.T, name string) []byte {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller")
	}
	dir := filepath.Dir(file)
	p := filepath.Join(dir, "testdata", name)
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read %s: %v", p, err)
	}
	return b
}

func commandNames(cmds []types.Command) []string {
	out := make([]string, len(cmds))
	for i, c := range cmds {
		out[i] = c.Name
	}
	return out
}

func TestHandwrittenBundled_matchesGoldenJSON(t *testing.T) {
	var want []types.Command
	if err := json.Unmarshal(readTestdata(t, "bundled_skills_golden.json"), &want); err != nil {
		t.Fatal(err)
	}
	got := handwritten.AssembleBundledSkills()
	if len(got) < len(want) {
		t.Fatalf("got len %d want at least %d", len(got), len(want))
	}
	gotCore := got[:len(want)]
	if !slices.Equal(commandNames(gotCore), commandNames(want)) {
		t.Fatalf("bundled core names\ngot  %v\nwant %v", commandNames(gotCore), commandNames(want))
	}
}

func TestHandwrittenBuiltin_defaultMatchesGoldenJSON(t *testing.T) {
	var want []types.Command
	if err := json.Unmarshal(readTestdata(t, "builtin_commands_default.json"), &want); err != nil {
		t.Fatal(err)
	}
	got := handwritten.AssembleBuiltinCommands()
	if len(got) != len(want) {
		t.Fatalf("len got %d want %d", len(got), len(want))
	}
	if !slices.Equal(commandNames(got), commandNames(want)) {
		t.Fatalf("builtin names mismatch (first diff index): compare manually")
	}
}

func TestHandwrittenBuiltin_featureBuddyInsertsBuddy(t *testing.T) {
	t.Setenv("FEATURE_BUDDY", "1")
	ClearLoadAllCommandsCache()
	var found bool
	for _, c := range handwritten.AssembleBuiltinCommands() {
		if c.Name == "buddy" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected buddy when FEATURE_BUDDY=1")
	}
}

func TestHandwrittenBuiltin_assume3PHidesAuthCommands(t *testing.T) {
	t.Setenv("CLAUDE_CODE_GO_ASSUME_3P", "1")
	ClearLoadAllCommandsCache()
	var logout, login bool
	for _, c := range handwritten.AssembleBuiltinCommands() {
		switch c.Name {
		case "logout":
			logout = true
		case "login":
			login = true
		}
	}
	if logout || login {
		t.Fatalf("expected no logout/login when CLAUDE_CODE_GO_ASSUME_3P=1 (logout=%v login=%v)", logout, login)
	}
}

func TestHandwrittenBuiltin_userTypeAntAppendsInternal(t *testing.T) {
	t.Setenv("USER_TYPE", "ant")
	t.Setenv("IS_DEMO", "")
	ClearLoadAllCommandsCache()
	base := len(handwritten.AssembleBuiltinCommands())
	t.Setenv("USER_TYPE", "")
	ClearLoadAllCommandsCache()
	without := len(handwritten.AssembleBuiltinCommands())
	if base <= without {
		t.Fatalf("expected more builtins with USER_TYPE=ant, got %d vs %d", base, without)
	}
}

func TestFeatureBuddyTruthyStrings(t *testing.T) {
	t.Setenv("FEATURE_BUDDY", "true")
	ClearLoadAllCommandsCache()
	found := false
	for _, c := range handwritten.AssembleBuiltinCommands() {
		if c.Name == "buddy" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("FEATURE_BUDDY=true should enable buddy")
	}
}
