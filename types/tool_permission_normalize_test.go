package types

import (
	"testing"
)

func TestNormalizeToolPermissionContextData_ruleMaps(t *testing.T) {
	d := ToolPermissionContextData{
		Mode:                             PermissionDefault,
		IsBypassPermissionsModeAvailable: false,
	}
	NormalizeToolPermissionContextData(&d)
	if len(d.AlwaysAllowRules) == 0 || string(d.AlwaysAllowRules) != "{}" {
		t.Fatalf("alwaysAllow: %q", d.AlwaysAllowRules)
	}
	if len(d.AdditionalWorkingDirectories) == 0 || string(d.AdditionalWorkingDirectories) != "{}" {
		t.Fatalf("awd: %q", d.AdditionalWorkingDirectories)
	}
}
