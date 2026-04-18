package prompt

import (
	"bytes"
	"reflect"
	"strconv"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
)

// NormalizeTTYNewlineKey maps TTY quirks to ctrl+j so multiline REPL sees a newline:
// macOS Option+layer may insert NEL / line/paragraph separators instead of Alt+Enter.
func NormalizeTTYNewlineKey(k tea.KeyPressMsg) tea.KeyPressMsg {
	key := k.Key()
	var r rune
	if key.Text != "" {
		r, _ = utf8.DecodeRuneInString(key.Text)
	} else {
		r = key.Code
	}
	switch r {
	case '\u0085', '\u2028', '\u2029':
		return tea.KeyPressMsg(tea.Key{Code: 'j', Mod: tea.ModCtrl})
	default:
		return k
	}
}

// SyntheticNewlineFromUnknownMsg returns a ctrl+j key when msg is bubbletea's unknown CSI sequence
// for kitty keyboard protocol style modified Enter (e.g. \x1b[13;2u for Alt+Enter) that the
// library does not map to KeyPressMsg.
func SyntheticNewlineFromUnknownMsg(msg tea.Msg) (tea.KeyPressMsg, bool) {
	k, ok := SyntheticTTYKeyFromUnknownMsg(msg)
	if !ok {
		return tea.KeyPressMsg{}, false
	}
	kk := k.Key()
	if kk.Code != 'j' || !kk.Mod.Contains(tea.ModCtrl) {
		return tea.KeyPressMsg{}, false
	}
	return k, true
}

// SyntheticTTYKeyFromUnknownMsg maps bubbletea's unknown CSI (Kitty keyboard protocol) to KeyPressMsg:
//   - Ctrl+letter (a–z) → ctrl+a…ctrl+z (e.g. \x1b[111;5u = ctrl+o, modifier 5 = 1+ctrl)
//   - modified Enter → ctrl+j (newline in REPL prompt)
func SyntheticTTYKeyFromUnknownMsg(msg tea.Msg) (tea.KeyPressMsg, bool) {
	b, ok := bubbleteaUnknownCSIBytes(msg)
	if !ok {
		return tea.KeyPressMsg{}, false
	}
	if k, mod, ok := parseKittyCSIKeyU(b); ok {
		if kp, ok := kittyCtrlLetterKeyPress(k, mod); ok {
			return kp, true
		}
	}
	if isKittyModifiedEnterCSI(b) {
		return tea.KeyPressMsg(tea.Key{Code: 'j', Mod: tea.ModCtrl}), true
	}
	return tea.KeyPressMsg{}, false
}

// kittyCtrlLetterKeyPress maps Kitty CSI key code + modifier to ctrl+a…ctrl+z (modifier 5 = 1+ctrl).
func kittyCtrlLetterKeyPress(keyCode, modEnc int) (tea.KeyPressMsg, bool) {
	if modEnc != 5 {
		return tea.KeyPressMsg{}, false
	}
	if keyCode < 97 || keyCode > 122 {
		return tea.KeyPressMsg{}, false
	}
	return tea.KeyPressMsg(tea.Key{Code: rune(keyCode), Mod: tea.ModCtrl}), true
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
	if t.Name() != "unknownCSISequenceMsg" || t.PkgPath() != "charm.land/bubbletea/v2" {
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
