package processuserinput

import (
	"testing"

	"goc/types"
)

func TestProcessTextPrompt_tenguInputPromptLog(t *testing.T) {
	var got map[string]any
	log := func(name string, payload map[string]any) {
		if name == "tengu_input_prompt" {
			got = payload
		}
	}
	_, err := ProcessTextPrompt("this sucks", nil, nil, nil, nil, nil, nil, nil, log)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("expected tengu_input_prompt")
	}
	if got["is_negative"] != true {
		t.Fatalf("is_negative=%v", got["is_negative"])
	}
	if got["is_keep_going"] != false {
		t.Fatalf("is_keep_going=%v", got["is_keep_going"])
	}
}

func TestProcessTextPrompt_blocksFirstTextForKeywords(t *testing.T) {
	var got map[string]any
	log := func(name string, payload map[string]any) {
		if name == "tengu_input_prompt" {
			got = payload
		}
	}
	blocks := []types.ContentBlockParam{
		{Type: "text", Text: "continue"},
		{Type: "text", Text: "more"},
	}
	_, err := ProcessTextPrompt("", blocks, nil, nil, nil, nil, nil, nil, log)
	if err != nil {
		t.Fatal(err)
	}
	if got["is_keep_going"] != true {
		t.Fatalf("first text block should drive keywords: %+v", got)
	}
}
