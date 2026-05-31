package core

import "testing"

func TestParseBasicKeys(t *testing.T) {
	tests := []struct {
		name string
		raw  []byte
		want ParsedKey
	}{
		{"enter", []byte("\r"), ParsedKey{Key: "enter"}},
		{"tab", []byte("\t"), ParsedKey{Key: "tab"}},
		{"backspace", []byte{127}, ParsedKey{Key: "backspace"}},
		{"ctrl+c", []byte{3}, ParsedKey{Key: "c", Mod: Ctrl}},
		{"escape", []byte{27}, ParsedKey{Key: "esc"}},
		{"letter a", []byte("a"), ParsedKey{Runes: []rune("a")}},
		{"letter A", []byte("A"), ParsedKey{Runes: []rune("A")}},
		{"unicode", []byte("世"), ParsedKey{Runes: []rune("世")}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewKeyboardParser()
			got := p.Parse(tt.raw)
			if got.Key != tt.want.Key {
				t.Errorf("Key: got %q, want %q", got.Key, tt.want.Key)
			}
			if got.Mod != tt.want.Mod {
				t.Errorf("Mod: got %v, want %v", got.Mod, tt.want.Mod)
			}
			if tt.want.Runes != nil {
				if string(got.Runes) != string(tt.want.Runes) {
					t.Errorf("Runes: got %q, want %q", string(got.Runes), string(tt.want.Runes))
				}
			}
		})
	}
}

func TestParseCSI(t *testing.T) {
	tests := []struct {
		name string
		raw  []byte
		want ParsedKey
	}{
		{"up", []byte("\x1b[A"), ParsedKey{Key: "up"}},
		{"down", []byte("\x1b[B"), ParsedKey{Key: "down"}},
		{"right", []byte("\x1b[C"), ParsedKey{Key: "right"}},
		{"left", []byte("\x1b[D"), ParsedKey{Key: "left"}},
		{"home", []byte("\x1b[H"), ParsedKey{Key: "home"}},
		{"end", []byte("\x1b[F"), ParsedKey{Key: "end"}},
		{"pgup", []byte("\x1b[5~"), ParsedKey{Key: "pgup"}},
		{"pgdn", []byte("\x1b[6~"), ParsedKey{Key: "pgdn"}},
		{"delete", []byte("\x1b[3~"), ParsedKey{Key: "delete"}},
		{"f1", []byte("\x1bOP"), ParsedKey{Key: "f1"}},
		{"f2", []byte("\x1bOQ"), ParsedKey{Key: "f2"}},
		{"shift+up", []byte("\x1b[1;2A"), ParsedKey{Key: "up", Mod: Shift}},
		{"ctrl+up", []byte("\x1b[1;5A"), ParsedKey{Key: "up", Mod: Ctrl}},
		{"alt+up", []byte("\x1b[1;3A"), ParsedKey{Key: "up", Mod: Alt}},
		{"ctrl+shift+up", []byte("\x1b[1;6A"), ParsedKey{Key: "up", Mod: Ctrl | Shift}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewKeyboardParser()
			got := p.Parse(tt.raw)
			if got.Key != tt.want.Key {
				t.Errorf("Key: got %q, want %q", got.Key, tt.want.Key)
			}
			if got.Mod != tt.want.Mod {
				t.Errorf("Mod: got 0x%x, want 0x%x", got.Mod, tt.want.Mod)
			}
		})
	}
}

func TestParseKitty(t *testing.T) {
	tests := []struct {
		name string
		raw  []byte
		want ParsedKey
	}{
		{"ctrl+o kitty", []byte("\x1b[57399u"), ParsedKey{Key: "o", Mod: Ctrl}},
		{"ctrl+enter kitty", []byte("\x1b[13;5u"), ParsedKey{Key: "enter", Mod: Ctrl}},
		{"alt+enter kitty", []byte("\x1b[13;3u"), ParsedKey{Key: "enter", Mod: Alt}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewKeyboardParser()
			got := p.Parse(tt.raw)
			if got.Key != tt.want.Key || got.Mod != tt.want.Mod {
				t.Errorf("got {%q 0x%x}, want {%q 0x%x}", got.Key, got.Mod, tt.want.Key, tt.want.Mod)
			}
		})
	}
}

func TestParseMultiByteUTF8(t *testing.T) {
	p := NewKeyboardParser()
	got := p.Parse([]byte("hello"))
	if string(got.Runes) != "hello" {
		t.Errorf("expected hello, got %q", string(got.Runes))
	}
}
