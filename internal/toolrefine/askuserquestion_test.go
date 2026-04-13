package toolrefine

import (
	"encoding/json"
	"testing"
)

func TestValidateAskUserQuestionUniqueness_ok(t *testing.T) {
	raw := json.RawMessage(`{"questions":[{"question":"A?","header":"A","options":[{"label":"x","description":"d"},{"label":"y","description":"d"}],"multiSelect":false}]}`)
	if err := ValidateAskUserQuestionUniqueness(raw); err != nil {
		t.Fatal(err)
	}
}

func TestValidateAskUserQuestionUniqueness_duplicateLabels(t *testing.T) {
	raw := json.RawMessage(`{"questions":[{"question":"A?","header":"A","options":[{"label":"dup","description":"1"},{"label":"dup","description":"2"}],"multiSelect":false}]}`)
	if err := ValidateAskUserQuestionUniqueness(raw); err == nil {
		t.Fatal("expected error")
	}
}
