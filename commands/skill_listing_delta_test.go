package commands

import (
	"strings"
	"testing"

	"goc/types"
)

func TestAppendSkillListingForAPI_noSkillTool(t *testing.T) {
	sent := map[string]struct{}{}
	cmds := []types.Command{{CommandBase: types.CommandBase{Name: "a", LoadedFrom: ptrStr("skills")}, Type: "prompt"}}
	_, ok := AppendSkillListingForAPI(cmds, false, sent, nil)
	if ok || len(sent) != 0 {
		t.Fatal()
	}
}

func TestAppendSkillListingForAPI_incremental(t *testing.T) {
	sent := map[string]struct{}{}
	cmds := []types.Command{
		{CommandBase: types.CommandBase{Name: "one", LoadedFrom: ptrStr("skills")}, Type: "prompt"},
		{CommandBase: types.CommandBase{Name: "two", LoadedFrom: ptrStr("skills")}, Type: "prompt"},
	}
	txt, ok := AppendSkillListingForAPI(cmds, true, sent, nil)
	if !ok || !strings.Contains(txt, "one") || !strings.Contains(txt, "two") {
		t.Fatalf("ok=%v txt=%q", ok, preview(txt, 200))
	}
	if len(sent) != 2 {
		t.Fatalf("sent=%v", sent)
	}
	txt2, ok2 := AppendSkillListingForAPI(cmds, true, sent, nil)
	if ok2 || txt2 != "" {
		t.Fatalf("second should be empty ok2=%v", ok2)
	}
	cmds2 := append(cmds, types.Command{CommandBase: types.CommandBase{Name: "three", LoadedFrom: ptrStr("skills")}, Type: "prompt"})
	txt3, ok3 := AppendSkillListingForAPI(cmds2, true, sent, nil)
	if !ok3 || !strings.Contains(txt3, "three") || strings.Contains(txt3, "one") {
		// delta should only list new skill
		t.Fatalf("ok3=%v txt3=%q", ok3, preview(txt3, 300))
	}
}

func preview(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}
