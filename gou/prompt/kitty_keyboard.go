// Kitty keyboard protocol (terminal-side): opt-in disambiguated key reporting.
// https://sw.kovidgoyal.net/kitty/keyboard-protocol/
//
// Quickstart: emit CSI > 1 u after startup (or when entering the alternate screen);
// emit CSI < u on exit to restore the previous keyboard mode.
package prompt

import (
	"os"
	"runtime"
)

const (
	// KittyKeyboardProtocolEnable is CSI > 1 u — push keyboard protocol state so
	// non-text keys (including modified Enter) use CSI … u forms.
	KittyKeyboardProtocolEnable = "\x1b[>1u"
	// KittyKeyboardProtocolDisable is CSI < u — pop / restore prior keyboard mode.
	KittyKeyboardProtocolDisable = "\x1b[<u"
)

// WriteKittyKeyboardProtocolEnable writes [KittyKeyboardProtocolEnable] to the controlling TTY.
// Use after the terminal is in raw/fullscreen mode (e.g. Bubble Tea Init), including with alt screen.
func WriteKittyKeyboardProtocolEnable() error {
	return writeToTTY(KittyKeyboardProtocolEnable)
}

// WriteKittyKeyboardProtocolDisable writes [KittyKeyboardProtocolDisable] to the controlling TTY.
// Call on program exit if [WriteKittyKeyboardProtocolEnable] was used.
func WriteKittyKeyboardProtocolDisable() error {
	return writeToTTY(KittyKeyboardProtocolDisable)
}

func writeToTTY(s string) error {
	f, err := openTTYOut()
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(s)
	return err
}

func openTTYOut() (*os.File, error) {
	if runtime.GOOS == "windows" {
		return os.OpenFile("CONOUT$", os.O_WRONLY, 0)
	}
	return os.OpenFile("/dev/tty", os.O_WRONLY, 0)
}
