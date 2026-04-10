package localturn

import "testing"

func TestValidateParams_empty(t *testing.T) {
	if err := validateParams(Params{}); err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateParams_textOK(t *testing.T) {
	if err := validateParams(Params{Text: "hi"}); err != nil {
		t.Fatal(err)
	}
}

func TestValidateParams_messagesBytesOK(t *testing.T) {
	if err := validateParams(Params{Messages: []byte(`[{"role":"user","content":"x"}]`)}); err != nil {
		t.Fatal(err)
	}
}
