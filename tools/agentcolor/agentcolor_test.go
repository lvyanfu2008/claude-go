package agentcolor

import (
	"testing"
)

func TestIsValidColorName(t *testing.T) {
	tests := []struct {
		name  string
		valid bool
	}{
		{"red", true},
		{"blue", true},
		{"green", true},
		{"yellow", true},
		{"purple", true},
		{"orange", true},
		{"pink", true},
		{"cyan", true},
		{"", false},
		{"unknown", false},
		{"RED", false},
		{"Red", false},
		{"magenta", false},
	}
	for _, tc := range tests {
		got := IsValidColorName(tc.name)
		if got != tc.valid {
			t.Errorf("IsValidColorName(%q) = %v, want %v", tc.name, got, tc.valid)
		}
	}
}

func TestThemeColorKey(t *testing.T) {
	if ColorRed.ThemeColorKey() != "red_FOR_SUBAGENTS_ONLY" {
		t.Errorf("red theme key = %q", ColorRed.ThemeColorKey())
	}
	if ColorBlue.ThemeColorKey() != "blue_FOR_SUBAGENTS_ONLY" {
		t.Errorf("blue theme key = %q", ColorBlue.ThemeColorKey())
	}
	if ColorCyan.ThemeColorKey() != "cyan_FOR_SUBAGENTS_ONLY" {
		t.Errorf("cyan theme key = %q", ColorCyan.ThemeColorKey())
	}
}

func TestThemeColorKeyFor(t *testing.T) {
	if got := ThemeColorKeyFor("red"); got != "red_FOR_SUBAGENTS_ONLY" {
		t.Errorf("ThemeColorKeyFor(red) = %q", got)
	}
	if got := ThemeColorKeyFor("unknown"); got != "" {
		t.Errorf("ThemeColorKeyFor(unknown) = %q, want empty", got)
	}
}

func TestSetAndGetAgentColor(t *testing.T) {
	ResetColorsForTest()

	// Initially nil
	if c := GetAgentColorName("test-agent"); c != nil {
		t.Fatalf("expected nil, got %v", *c)
	}
	if c := GetThemeColorKey("test-agent"); c != nil {
		t.Fatalf("expected nil, got %v", *c)
	}

	// Set a color
	blue := ColorBlue
	SetAgentColor("test-agent", &blue)

	c := GetAgentColorName("test-agent")
	if c == nil || *c != ColorBlue {
		t.Fatalf("expected blue, got %v", c)
	}
	tk := GetThemeColorKey("test-agent")
	if tk == nil || *tk != "blue_FOR_SUBAGENTS_ONLY" {
		t.Fatalf("expected blue_FOR_SUBAGENTS_ONLY, got %v", tk)
	}

	// Delete
	SetAgentColor("test-agent", nil)
	if c := GetAgentColorName("test-agent"); c != nil {
		t.Fatalf("expected nil after delete, got %v", *c)
	}
}

func TestSetAgentColorName(t *testing.T) {
	ResetColorsForTest()

	if !SetAgentColorName("agent-a", "orange") {
		t.Fatal("SetAgentColorName(orange) should succeed")
	}
	c := GetAgentColorName("agent-a")
	if c == nil || *c != ColorOrange {
		t.Fatalf("expected orange, got %v", c)
	}

	if SetAgentColorName("agent-b", "invalid") {
		t.Fatal("SetAgentColorName(invalid) should return false")
	}
	if c := GetAgentColorName("agent-b"); c != nil {
		t.Fatalf("expected nil for invalid color, got %v", *c)
	}
}

func TestDeleteAgentColor(t *testing.T) {
	ResetColorsForTest()

	red := ColorRed
	SetAgentColor("del-agent", &red)
	if GetAgentColorName("del-agent") == nil {
		t.Fatal("expected color after set")
	}
	DeleteAgentColor("del-agent")
	if GetAgentColorName("del-agent") != nil {
		t.Fatal("expected nil after delete")
	}
}

func TestGeneralPurposeReturnsNil(t *testing.T) {
	ResetColorsForTest()

	if tk := GetThemeColorKey("general-purpose"); tk != nil {
		t.Fatalf("general-purpose should return nil, got %v", *tk)
	}
}

func TestInitAgentColors(t *testing.T) {
	ResetColorsForTest()

	agents := []AgentColorSetter{
		{AgentType: "statusline-setup", Color: "orange"},
		{AgentType: "verification", Color: "red"},
		{AgentType: "no-color", Color: ""},
		{AgentType: "explorer", Color: "purple"},
	}
	InitAgentColors(agents)

	tests := []struct {
		agentType string
		want      *ColorName
	}{
		{"statusline-setup", colorPtr(ColorOrange)},
		{"verification", colorPtr(ColorRed)},
		{"no-color", nil},
		{"explorer", colorPtr(ColorPurple)},
		{"unknown", nil},
	}
	for _, tc := range tests {
		got := GetAgentColorName(tc.agentType)
		if tc.want == nil {
			if got != nil {
				t.Errorf("agent %q: expected nil, got %v", tc.agentType, *got)
			}
		} else {
			if got == nil {
				t.Errorf("agent %q: expected %v, got nil", tc.agentType, *tc.want)
			} else if *got != *tc.want {
				t.Errorf("agent %q: expected %v, got %v", tc.agentType, *tc.want, *got)
			}
		}
	}
}

func colorPtr(c ColorName) *ColorName { return &c }
