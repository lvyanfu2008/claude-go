package types

import (
	"encoding/json"
	"testing"
)

func TestSystemContextUserContext_JSON(t *testing.T) {
	sys := SystemContext{
		GitStatus:    strPtr("branch: main"),
		CacheBreaker: strPtr("[CACHE_BREAKER: x]"),
	}
	b, err := json.Marshal(sys)
	if err != nil {
		t.Fatal(err)
	}
	var sys2 SystemContext
	if err := json.Unmarshal(b, &sys2); err != nil {
		t.Fatal(err)
	}
	if sys2.GitStatus == nil || *sys2.GitStatus != "branch: main" {
		t.Fatalf("%+v", sys2)
	}

	usr := UserContext{CurrentDate: "Today's date is 2026-04-08."}
	b2, _ := json.Marshal(usr)
	var usr2 UserContext
	if err := json.Unmarshal(b2, &usr2); err != nil || usr2.CurrentDate != usr.CurrentDate {
		t.Fatalf("%+v err=%v", usr2, err)
	}
}

func strPtr(s string) *string { return &s }
