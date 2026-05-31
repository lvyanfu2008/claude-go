package core

// Short aliases for modifier constants defined in mouse.go.
// ModCtrl=8, ModAlt=4, ModShift=2.
const (
	Ctrl  = ModCtrl
	Alt   = ModAlt
	Shift = ModShift
	Meta  Modifier = 1 << 4 // 16
)

// ParsedKey holds the result of parsing a raw keyboard input sequence.
type ParsedKey struct {
	Key   string
	Mod   Modifier
	Runes []rune
}

// KeyboardParser decodes raw terminal bytes into ParsedKey values.
// It handles plain keys, Ctrl+letter, CSI sequences (with modifiers),
// SS3 sequences (F1-F4), and the Kitty extended protocol (CSI u).
type KeyboardParser struct{}

// NewKeyboardParser creates a new KeyboardParser.
func NewKeyboardParser() *KeyboardParser {
	return &KeyboardParser{}
}

// Parse decodes a raw byte sequence into a ParsedKey.
func (p *KeyboardParser) Parse(raw []byte) ParsedKey {
	if len(raw) == 0 {
		return ParsedKey{}
	}
	b := raw[0]

	switch b {
	case 13: // CR
		return ParsedKey{Key: "enter"}
	case 9: // TAB
		return ParsedKey{Key: "tab"}
	case 127: // DEL
		return ParsedKey{Key: "backspace"}
	case 27: // ESC — may start an escape sequence
		if len(raw) == 1 {
			return ParsedKey{Key: "esc"}
		}
		return p.parseEscape(raw[1:])
	}

	// Ctrl+letter (bytes 1-26, excluding those handled above: 9, 13, 27)
	if b >= 1 && b <= 26 {
		key := string(rune('a' + b - 1))
		return ParsedKey{Key: key, Mod: Ctrl}
	}

	// Printable characters (including multi-byte UTF-8)
	runes := []rune(string(raw))
	return ParsedKey{Runes: runes}
}

// parseEscape handles bytes following ESC (0x1B).
func (p *KeyboardParser) parseEscape(rest []byte) ParsedKey {
	if len(rest) == 0 {
		return ParsedKey{Key: "esc"}
	}
	switch rest[0] {
	case '[':
		return p.parseCSI(rest[1:])
	case 'O':
		return p.parseSS3(rest[1:])
	default:
		return ParsedKey{Key: "esc", Runes: []rune(string(rest))}
	}
}

// parseCSI handles Control Sequence Introducer sequences (ESC [ ... final).
func (p *KeyboardParser) parseCSI(rest []byte) ParsedKey {
	if len(rest) == 0 {
		return ParsedKey{}
	}
	// Kitty protocol: CSI number u or CSI number;modifier u
	if rest[len(rest)-1] == 'u' {
		return p.parseKitty(rest[:len(rest)-1])
	}
	// CSI number ~ (tilde sequences: PgUp, PgDn, Delete, Insert, etc.)
	if rest[len(rest)-1] == '~' {
		return p.parseCSITilde(rest[:len(rest)-1])
	}
	// Simple CSI: final byte is a command letter (A-F, H, Z, etc.)
	final := rest[len(rest)-1]
	params := rest[:len(rest)-1]
	mod, param1 := parseCSIParams(params)
	// Convert protocol bitmask (1=Shift,2=Alt,4=Ctrl) to Modifier values
	// by shifting left by 1 (ModShift=2, ModAlt=4, ModCtrl=8).
	mod = mod << 1

	switch final {
	case 'A':
		return ParsedKey{Key: "up", Mod: mod}
	case 'B':
		return ParsedKey{Key: "down", Mod: mod}
	case 'C':
		return ParsedKey{Key: "right", Mod: mod}
	case 'D':
		return ParsedKey{Key: "left", Mod: mod}
	case 'H':
		return ParsedKey{Key: "home", Mod: mod}
	case 'F':
		return ParsedKey{Key: "end", Mod: mod}
	case 'Z':
		return ParsedKey{Key: "tab", Mod: Shift}
	default:
		_ = param1
		return ParsedKey{}
	}
}

// parseCSITilde handles CSI tilde sequences like ESC [ n ~.
func (p *KeyboardParser) parseCSITilde(params []byte) ParsedKey {
	mod, param1 := parseCSIParams(params)
	mod = mod << 1 // convert protocol bitmask to Modifier values
	switch param1 {
	case 2:
		return ParsedKey{Key: "insert", Mod: mod}
	case 3:
		return ParsedKey{Key: "delete", Mod: mod}
	case 5:
		return ParsedKey{Key: "pgup", Mod: mod}
	case 6:
		return ParsedKey{Key: "pgdn", Mod: mod}
	case 7:
		return ParsedKey{Key: "home", Mod: mod}
	case 8:
		return ParsedKey{Key: "end", Mod: mod}
	case 11:
		return ParsedKey{Key: "f1", Mod: mod}
	case 12:
		return ParsedKey{Key: "f2", Mod: mod}
	case 13:
		return ParsedKey{Key: "f3", Mod: mod}
	case 14:
		return ParsedKey{Key: "f4", Mod: mod}
	case 15:
		return ParsedKey{Key: "f5", Mod: mod}
	case 17:
		return ParsedKey{Key: "f6", Mod: mod}
	case 18:
		return ParsedKey{Key: "f7", Mod: mod}
	case 19:
		return ParsedKey{Key: "f8", Mod: mod}
	case 20:
		return ParsedKey{Key: "f9", Mod: mod}
	case 21:
		return ParsedKey{Key: "f10", Mod: mod}
	case 23:
		return ParsedKey{Key: "f11", Mod: mod}
	case 24:
		return ParsedKey{Key: "f12", Mod: mod}
	default:
		return ParsedKey{}
	}
}

// parseSS3 handles SS3 sequences (ESC O final) used for F1-F4 and some
// terminals for Home/End.
func (p *KeyboardParser) parseSS3(rest []byte) ParsedKey {
	if len(rest) == 0 {
		return ParsedKey{}
	}
	switch rest[0] {
	case 'P':
		return ParsedKey{Key: "f1"}
	case 'Q':
		return ParsedKey{Key: "f2"}
	case 'R':
		return ParsedKey{Key: "f3"}
	case 'S':
		return ParsedKey{Key: "f4"}
	case 'H':
		return ParsedKey{Key: "home"}
	case 'F':
		return ParsedKey{Key: "end"}
	default:
		return ParsedKey{}
	}
}

// parseKitty handles the Kitty extended keyboard protocol (CSI ... u).
// The parameter format is: codepoint[;modifier] where modifier is
// 1 + bitmask (1=Shift, 2=Alt, 4=Ctrl, 8=Meta).
func (p *KeyboardParser) parseKitty(params []byte) ParsedKey {
	rawMod, codepoint := parseCSIParams(params)
	// rawMod is the protocol-level modifier value (already had 1 subtracted
	// by parseCSIParams). Convert to our Modifier bitmask.
	var modBits Modifier
	if rawMod&1 != 0 {
		modBits |= Shift
	}
	if rawMod&2 != 0 {
		modBits |= Alt
	}
	if rawMod&4 != 0 {
		modBits |= Ctrl
	}
	if rawMod&8 != 0 {
		modBits |= Meta
	}
	// Kitty PUA range (0xE029-0xE042) encodes Ctrl+letter directly in the
	// codepoint. If no explicit modifier was sent, set Ctrl.
	if modBits == 0 && codepoint >= 0xE029 && codepoint <= 0xE042 {
		modBits = Ctrl
	}
	key := keyNameFromCodepoint(codepoint)
	return ParsedKey{Key: key, Mod: modBits}
}

// parseCSIParams splits CSI parameter bytes on ';' and returns the modifier
// and first integer parameter. The modifier value has 1 subtracted per the
// CSI convention (1=no modifier, 2=Shift, 3=Alt, 4=Alt+Shift, 5=Ctrl, etc.).
// When there is no ';' separator, modifier is left at 0 (no modifier).
func parseCSIParams(params []byte) (mod Modifier, param1 int) {
	s := string(params)
	parts := splitByte(s, ';')
	if len(parts) > 1 {
		param1 = atoi(parts[0])
		mod = Modifier(atoi(parts[1]) - 1)
	} else if len(parts) == 1 {
		param1 = atoi(parts[0])
		// mod stays 0 (zero value of Modifier) — no modifier present
	}
	return
}

// keyNameFromCodepoint returns the key name for a Unicode codepoint,
// handling control characters, common named keys, and the Kitty PUA
// range for Ctrl+letter.
func keyNameFromCodepoint(cp int) string {
	switch cp {
	case 13:
		return "enter"
	case 9:
		return "tab"
	case 32:
		return "space"
	case 127:
		return "backspace"
	case 27:
		return "esc"
	default:
		if cp >= 1 && cp <= 26 {
			return string(rune('a' + cp - 1))
		}
		// Kitty PUA range for Ctrl+letter: codepoint = 0xE000 + 0x28 + ctrl_code
		// where ctrl_code is 1-26 (Ctrl+a through Ctrl+z).
		// 0xE029 = Ctrl+a, 0xE042 = Ctrl+z.
		if cp >= 0xE029 && cp <= 0xE042 {
			ctrlCode := cp - 0xE028
			if ctrlCode >= 1 && ctrlCode <= 26 {
				return string(rune('a' + ctrlCode - 1))
			}
		}
		if cp >= 0x20 && cp <= 0x7E {
			return string(rune(cp))
		}
	}
	return ""
}

// splitByte splits a string on a byte separator.
func splitByte(s string, sep byte) []string {
	var parts []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == sep {
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}
	parts = append(parts, s[start:])
	return parts
}

// atoi parses a decimal integer from a string. Returns 0 on empty input.
func atoi(s string) int {
	n := 0
	for _, r := range s {
		if r >= '0' && r <= '9' {
			n = n*10 + int(r-'0')
		}
	}
	return n
}
