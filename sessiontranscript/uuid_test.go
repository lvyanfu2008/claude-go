package sessiontranscript

import "testing"

func TestIsValidUUID(t *testing.T) {
	if !IsValidUUID("11111111-2222-3333-4444-555555555555") {
		t.Fatal()
	}
	if IsValidUUID("demo") || IsValidUUID("") || IsValidUUID("not-a-uuid") {
		t.Fatal()
	}
}

func TestNewUUID_format(t *testing.T) {
	u := NewUUID()
	if !IsValidUUID(u) {
		t.Fatalf("%q", u)
	}
}
