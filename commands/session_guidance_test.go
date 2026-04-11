package commands

import (
	"strings"
	"testing"

	"goc/types"
)

func TestSessionSpecificGuidance_skillsBullet(t *testing.T) {
	enabled := EnabledToolNames([]string{"Skill", "Bash"})
	cmds := []types.Command{{CommandBase: types.CommandBase{Name: "x", LoadedFrom: ptrStr("skills")}, Type: "prompt"}}
	s := SessionSpecificGuidance(enabled, cmds)
	if s == "" || !strings.Contains(s, "# Session-specific guidance") || !strings.Contains(s, "/<skill-name>") {
		t.Fatalf("%q", s)
	}
}

func TestSessionSpecificGuidanceFull_includesAskUserAndDiscover(t *testing.T) {
	sk := []types.Command{{CommandBase: types.CommandBase{Name: "s", LoadedFrom: ptrStr("skills")}, Type: "prompt"}}
	enabled2 := EnabledToolNames([]string{"Skill", "AskUserQuestion", "DiscoverX"})
	s2 := SessionSpecificGuidanceFull(GouDemoSystemOpts{
		EnabledToolNames:       enabled2,
		SkillToolCommands:      sk,
		DiscoverSkillsToolName: "DiscoverX",
		NonInteractiveSession:  false,
	})
	if !strings.Contains(s2, "AskUserQuestion") || !strings.Contains(s2, "DiscoverX") || !strings.Contains(s2, "Skills relevant to your task") {
		t.Fatalf("%q", s2)
	}
}

func TestSessionSpecificGuidanceFull_agentAndExplore(t *testing.T) {
	en := EnabledToolNames([]string{agentToolName, globToolName, grepToolName})
	s := SessionSpecificGuidanceFull(GouDemoSystemOpts{
		EnabledToolNames:         en,
		ExplorePlanAgentsEnabled: true,
		EmbeddedSearchTools:      false,
		NonInteractiveSession:    false,
	})
	if !strings.Contains(s, "Use the "+agentToolName+" tool with specialized") {
		t.Fatalf("missing agent bullet: %q", s)
	}
	if !strings.Contains(s, "subagent_type="+exploreAgentType) || !strings.Contains(s, "For simple, directed") {
		t.Fatalf("missing explore bullets: %q", s)
	}
}

func TestSessionSpecificGuidance_noSkillTool(t *testing.T) {
	enabled := EnabledToolNames([]string{"Bash"})
	cmds := []types.Command{{CommandBase: types.CommandBase{Name: "x", LoadedFrom: ptrStr("skills")}, Type: "prompt"}}
	if SessionSpecificGuidance(enabled, cmds) != "" {
		t.Fatal()
	}
}
