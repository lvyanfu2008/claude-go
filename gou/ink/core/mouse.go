package core

import "strconv"

// Modifier represents keyboard modifier keys.
type Modifier uint8

const (
	ModNone  Modifier = 0
	ModShift Modifier = 1 << iota
	ModAlt
	ModCtrl
)

// MouseEventType describes the kind of mouse action.
type MouseEventType uint8

const (
	MousePress   MouseEventType = iota
	MouseRelease
	MouseMove
	MouseWheel
)

// MouseEvent holds the decoded state of an SGR 1006 mouse sequence.
type MouseEvent struct {
	Type   MouseEventType
	Button int // 0=left, 1=middle, 2=right; wheel: -1=up, 1=down
	X, Y   int // 1-indexed cell coordinates
	Mod    Modifier
}

// DecodeMouse decodes an SGR 1006 mouse sequence.
// Format: ESC [ < Btn ; X ; Y M/m
func DecodeMouse(raw []byte) (MouseEvent, bool) {
	if !IsMouseEvent(raw) {
		return MouseEvent{}, false
	}

	// Strip the prefix "\x1b[<" and work with the rest.
	rest := raw[3:]

	// Parse button number.
	btn, rest, ok := parseInt(rest)
	if !ok {
		return MouseEvent{}, false
	}

	// Expect ';'
	if len(rest) == 0 || rest[0] != ';' {
		return MouseEvent{}, false
	}
	rest = rest[1:]

	// Parse X.
	x, rest, ok := parseInt(rest)
	if !ok {
		return MouseEvent{}, false
	}

	// Expect ';'
	if len(rest) == 0 || rest[0] != ';' {
		return MouseEvent{}, false
	}
	rest = rest[1:]

	// Parse Y.
	y, rest, ok := parseInt(rest)
	if !ok {
		return MouseEvent{}, false
	}

	// Expect 'M' (press/motion/wheel) or 'm' (release).
	if len(rest) == 0 || (rest[0] != 'M' && rest[0] != 'm') {
		return MouseEvent{}, false
	}
	suffix := rest[0]
	_ = suffix

	ev := MouseEvent{X: x, Y: y}

	// Determine event type and button from the encoded button number.
	switch {
	case btn >= 64:
		// Wheel event.
		ev.Type = MouseWheel
		switch btn {
		case 64:
			ev.Button = -1 // up
		case 65:
			ev.Button = 1 // down
		default:
			ev.Button = 0
		}
		// Wheel doesn't carry a meaningful button; extract modifiers.
		ev.Mod = extractMod(btn)

	case btn >= 32:
		// Motion event.
		ev.Type = MouseMove
		ev.Button = int(btn & 0x1F)
		ev.Mod = extractMod(btn)

	case suffix == 'm':
		ev.Type = MouseRelease
		ev.Button = int(btn & 0x03)
		ev.Mod = extractMod(btn)

	default:
		// Press ('M').
		ev.Type = MousePress
		ev.Button = int(btn & 0x03)
		ev.Mod = extractMod(btn)
	}

	return ev, true
}

// IsMouseEvent returns true if raw starts an SGR mouse sequence (ESC [ <).
func IsMouseEvent(raw []byte) bool {
	return len(raw) >= 4 && raw[0] == '\x1b' && raw[1] == '[' && raw[2] == '<'
}

// IsBracketedPasteStart returns true for \x1b[200~
func IsBracketedPasteStart(raw []byte) bool {
	return len(raw) >= 6 && string(raw[:6]) == "\x1b[200~"
}

// IsBracketedPasteEnd returns true for \x1b[201~
func IsBracketedPasteEnd(raw []byte) bool {
	return len(raw) >= 6 && string(raw[:6]) == "\x1b[201~"
}

// parseInt reads a non-negative integer from the front of buf,
// stopping at the first non-digit byte. It returns the integer,
// the remaining slice, and whether a number was found.
func parseInt(buf []byte) (int, []byte, bool) {
	if len(buf) == 0 || buf[0] < '0' || buf[0] > '9' {
		return 0, buf, false
	}
	i := 0
	for i < len(buf) && buf[i] >= '0' && buf[i] <= '9' {
		i++
	}
	n, _ := strconv.Atoi(string(buf[:i]))
	return n, buf[i:], true
}

// extractMod extracts modifier flags from the upper bits of an SGR button code.
// bit 2 = shift, bit 3 = alt, bit 4 = ctrl.
func extractMod(btn int) Modifier {
	var m Modifier
	if btn&0x04 != 0 {
		m |= ModShift
	}
	if btn&0x08 != 0 {
		m |= ModAlt
	}
	if btn&0x10 != 0 {
		m |= ModCtrl
	}
	return m
}
