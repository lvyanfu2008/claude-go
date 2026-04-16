package prompt

import (
	"bytes"
	"reflect"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
)

// NormalizeTTYNewlineKey maps TTY quirks to KeyCtrlJ so multiline REPL sees a newline:
// macOS Option+layer may insert NEL / line/paragraph separators instead of Alt+Enter.
func NormalizeTTYNewlineKey(k tea.KeyMsg) tea.KeyMsg {
	if k.Type != tea.KeyRunes || k.Paste || len(k.Runes) != 1 {
		return k
	}
	switch k.Runes[0] {
	case '\u0085', '\u2028', '\u2029':
		return tea.KeyMsg{Type: tea.KeyCtrlJ}
	default:
		return k
	}
}

// SyntheticNewlineFromUnknownMsg returns KeyCtrlJ when msg is bubbletea's unknown CSI sequence
// for kitty keyboard protocol style modified Enter (e.g. \x1b[13;2u for Alt+Enter) that the
// library does not map to KeyMsg.
func SyntheticNewlineFromUnknownMsg(msg tea.Msg) (tea.KeyMsg, bool) {
	k, ok := SyntheticTTYKeyFromUnknownMsg(msg)
	if !ok || k.Type != tea.KeyCtrlJ {
		return tea.KeyMsg{}, false
	}
	return k, true
}

// SyntheticTTYKeyFromUnknownMsg maps bubbletea's unknown CSI (Kitty keyboard protocol) to KeyMsg:
//   - Ctrl+letter (a–z) → KeyCtrlA…KeyCtrlZ (e.g. \x1b[111;5u = ctrl+o, modifier 5 = 1+ctrl)
//   - modified Enter → KeyCtrlJ (newline in REPL prompt)
func SyntheticTTYKeyFromUnknownMsg(msg tea.Msg) (tea.KeyMsg, bool) {
	b, ok := bubbleteaUnknownCSIBytes(msg)
	if !ok {
		return tea.KeyMsg{}, false
	}
	if k, mod, ok := parseKittyCSIKeyU(b); ok {
		if kt, ok := kittyCtrlLetterKeyType(k, mod); ok {
			return tea.KeyMsg{Type: kt}, true
		}
	}
	if isKittyModifiedEnterCSI(b) {
		return tea.KeyMsg{Type: tea.KeyCtrlJ}, true
	}
	return tea.KeyMsg{}, false
}

// kittyCtrlLetterKeyType maps Kitty CSI key code + modifier to ctrl+a…ctrl+z (modifier 5 = 1+ctrl).
func kittyCtrlLetterKeyType(keyCode, modEnc int) (tea.KeyType, bool) {
	if modEnc != 5 {
		return 0, false
	}
	if keyCode < 97 || keyCode > 122 {
		return 0, false
	}
	return tea.KeyType(keyCode - 96), true
}

// parseKittyCSIKeyU parses CSI … u sequences (e.g. \x1b[99;5u or \x1b[99;5:1u).
func parseKittyCSIKeyU(b []byte) (keyCode int, modEnc int, ok bool) {
	if len(b) < 7 || b[0] != '\x1b' || b[1] != '[' || !bytes.HasSuffix(b, []byte("u")) {
		return 0, 0, false
	}
	core := b[2 : len(b)-1]
	semi := bytes.IndexByte(core, ';')
	if semi < 0 {
		return 0, 0, false
	}
	keyStr := string(core[:semi])
	rest := core[semi+1:]
	modPart := rest
	if colon := bytes.IndexByte(rest, ':'); colon >= 0 {
		modPart = rest[:colon]
	}
	k, err1 := strconv.Atoi(keyStr)
	m, err2 := strconv.Atoi(string(modPart))
	if err1 != nil || err2 != nil {
		return 0, 0, false
	}
	return k, m, true
}

func bubbleteaUnknownCSIBytes(msg tea.Msg) ([]byte, bool) {
	t := reflect.TypeOf(msg)
	if t == nil || t.Kind() != reflect.Slice || t.Elem().Kind() != reflect.Uint8 {
		return nil, false
	}
	if t.Name() != "unknownCSISequenceMsg" || t.PkgPath() != "github.com/charmbracelet/bubbletea" {
		return nil, false
	}
	return reflect.ValueOf(msg).Bytes(), true
}

// isKittyModifiedEnterCSI reports CSI "u" (kitty / xterm style) for key 13 (Enter) with a modifier.
// Plain \x1b[13u without ";N" is left alone (may duplicate bare Enter handling elsewhere).
func isKittyModifiedEnterCSI(b []byte) bool {
	if len(b) < 7 || b[0] != '\x1b' || b[1] != '[' {
		return false
	}
	if !bytes.HasSuffix(b, []byte("u")) {
		return false
	}
	// \x1b[13;Nu or \x1b[13:Nu — must include a modifier field
	if bytes.HasPrefix(b[2:], []byte("13;")) || bytes.HasPrefix(b[2:], []byte("13:")) {
		return true
	}
	return false
}
