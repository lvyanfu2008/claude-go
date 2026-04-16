package prompt

import (
	"bytes"
	"reflect"

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
	b, ok := bubbleteaUnknownCSIBytes(msg)
	if !ok || !isKittyModifiedEnterCSI(b) {
		return tea.KeyMsg{}, false
	}
	return tea.KeyMsg{Type: tea.KeyCtrlJ}, true
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
