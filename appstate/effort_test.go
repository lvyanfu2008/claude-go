package appstate

import (
	"encoding/json"
	"testing"
)

func TestEffortValue_JSON_string(t *testing.T) {
	e := EffortValue{Level: EffortHigh, IsNum: false}
	b, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `"high"` {
		t.Fatalf("got %s", b)
	}
	var got EffortValue
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got.Level != EffortHigh || got.IsNum {
		t.Fatalf("%+v", got)
	}
}

func TestEffortValue_JSON_number(t *testing.T) {
	b := []byte(`0.75`)
	var got EffortValue
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if !got.IsNum || got.Number != 0.75 {
		t.Fatalf("%+v", got)
	}
	out, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != `0.75` {
		t.Fatalf("got %s", out)
	}
}
