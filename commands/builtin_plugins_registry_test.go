package commands

import (
	"os"
	"path/filepath"
	"testing"

	"goc/types"
)

func TestGetBuiltinPlugins_emptyRegistry(t *testing.T) {
	t.Cleanup(ClearBuiltinPluginsRegistry)
	ClearBuiltinPluginsRegistry()

	en, dis, err := GetBuiltinPlugins(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(en) != 0 || len(dis) != 0 {
		t.Fatalf("want empty, got enabled=%d disabled=%d", len(en), len(dis))
	}
}

func TestBuiltinPluginSkillCommands_respectsEnabledPlugins(t *testing.T) {
	t.Cleanup(ClearBuiltinPluginsRegistry)
	ClearBuiltinPluginsRegistry()

	root := t.TempDir()
	cl := filepath.Join(root, ".claude")
	if err := os.MkdirAll(cl, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cl, "settings.json"), []byte(`{"enabledPlugins":{"demo-plugin@builtin":false}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	RegisterBuiltinPlugin(BuiltinPluginDefinition{
		Name:        "demo-plugin",
		Description: "test",
		Skills: []types.Command{
			{CommandBase: types.CommandBase{Name: "demo-skill", Description: "x"}, Type: "prompt"},
		},
	})

	sk, err := BuiltinPluginSkillCommands(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(sk) != 0 {
		t.Fatalf("expected no skills when plugin disabled, got %d", len(sk))
	}

	en, dis, err := GetBuiltinPlugins(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(en) != 0 || len(dis) != 1 {
		t.Fatalf("enabled=%d disabled=%d", len(en), len(dis))
	}
	if dis[0].Name != "demo-plugin" || dis[0].Enabled {
		t.Fatalf("%+v", dis[0])
	}
}

func TestBuiltinPluginSkillCommands_defaultEnabled(t *testing.T) {
	t.Cleanup(ClearBuiltinPluginsRegistry)
	ClearBuiltinPluginsRegistry()

	root := t.TempDir()
	RegisterBuiltinPlugin(BuiltinPluginDefinition{
		Name:        "p2",
		Description: "y",
		Skills: []types.Command{
			{CommandBase: types.CommandBase{Name: "s2", Description: "z"}, Type: "prompt"},
		},
	})

	sk, err := BuiltinPluginSkillCommands(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(sk) != 1 || sk[0].Name != "s2" {
		t.Fatalf("%+v", sk)
	}
	if sk[0].Source == nil || *sk[0].Source != "bundled" {
		t.Fatalf("source: %+v", sk[0].Source)
	}
}

func TestIsBuiltinPluginID(t *testing.T) {
	if !IsBuiltinPluginID("foo@builtin") {
		t.Fatal()
	}
	if IsBuiltinPluginID("foo@marketplace") {
		t.Fatal()
	}
}
