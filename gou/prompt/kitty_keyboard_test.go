package prompt

import "testing"

func TestKittyKeyboardProtocolSequences(t *testing.T) {
	t.Parallel()
	if KittyKeyboardProtocolEnable != "\x1b[>1u" {
		t.Fatalf("enable: %q", KittyKeyboardProtocolEnable)
	}
	if KittyKeyboardProtocolDisable != "\x1b[<u" {
		t.Fatalf("disable: %q", KittyKeyboardProtocolDisable)
	}
}
