package appstate

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestTodosMap_emptyJSON(t *testing.T) {
	var m TodosMap
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `{}` {
		t.Fatalf("%s", b)
	}
}

func TestTodosMap_nilSlicePerAgent(t *testing.T) {
	m := TodosMap{"a": nil}
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"a":[]`) {
		t.Fatalf("%s", b)
	}
}
