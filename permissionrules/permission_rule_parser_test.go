package permissionrules

import "testing"

func TestPermissionRuleValueFromString(t *testing.T) {
	t.Parallel()
	v := PermissionRuleValueFromString("Bash")
	if v.ToolName != "Bash" || v.RuleContent != nil {
		t.Fatalf("%+v", v)
	}
	v = PermissionRuleValueFromString("Bash(npm install)")
	if v.ToolName != "Bash" || v.RuleContent == nil || *v.RuleContent != "npm install" {
		t.Fatalf("%+v", v)
	}
	v = PermissionRuleValueFromString("Bash()")
	if v.ToolName != "Bash" || v.RuleContent != nil {
		t.Fatalf("empty content -> whole tool: %+v", v)
	}
	v = PermissionRuleValueFromString("Bash(*)")
	if v.ToolName != "Bash" || v.RuleContent != nil {
		t.Fatalf("wildcard content -> whole tool: %+v", v)
	}
	v = PermissionRuleValueFromString("(foo)")
	if v.ToolName != "(foo)" || v.RuleContent != nil {
		t.Fatalf("malformed: %+v", v)
	}
}

func TestUnescapeRuleContent(t *testing.T) {
	t.Parallel()
	// Same character sequence as TS rawContent before unescape: print\(1\)
	got := UnescapeRuleContent(`python -c "print\(1\)"`)
	want := `python -c "print(1)"`
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
