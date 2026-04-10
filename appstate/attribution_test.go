package appstate

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestAttributionState_emptyJSONObjects(t *testing.T) {
	a := EmptyAttributionState()
	b, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"fileStates":{}`) || !strings.Contains(string(b), `"sessionBaselines":{}`) {
		t.Fatalf("%s", b)
	}
	var back AttributionState
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}
	if back.FileStates == nil || back.SessionBaselines == nil {
		t.Fatal("maps should be non-nil after unmarshal")
	}
}

func TestAttributionState_unmarshalBareObject(t *testing.T) {
	raw := []byte(`{"surface":"ide","fileStates":null,"sessionBaselines":null,"promptCount":1}`)
	var a AttributionState
	if err := json.Unmarshal(raw, &a); err != nil {
		t.Fatal(err)
	}
	if a.FileStates == nil || a.SessionBaselines == nil {
		t.Fatal("expected empty maps, not nil")
	}
	if a.Surface != "ide" || a.PromptCount != 1 {
		t.Fatalf("%+v", a)
	}
}
