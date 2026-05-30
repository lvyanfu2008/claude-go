package integration

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"goc/gou/ink"
	"goc/gou/ink/compo"
	"goc/gou/theme"
)

// RunNewEngine starts the new rendering engine. Activated by GOU_USE_NEW_ENGINE=1.
// Returns true if it handled execution (caller should return).
func RunNewEngine() bool {
	if os.Getenv("GOU_USE_NEW_ENGINE") != "1" {
		return false
	}

	term := ink.NewTerminal()
	store := ink.NewStore()

	pal := theme.ActivePalette()
	if pal == nil {
		pal = &theme.Palette{}
	}

	// Set actual CWD for status line
	if cwd, err := os.Getwd(); err == nil {
		store.SetMeta("cwd", cwd)
	}
	store.SetMeta("model", "Opus 4.7")

	engine := ink.NewEngine(term, store, pal, compo.REPL)
	if err := engine.Start(); err != nil {
		panic(err)
	}
	defer engine.Stop()

	// Seed welcome message
	store.AppendMessage(ink.Message{
		UUID: "welcome", Type: "system",
		ContentBlocks: []ink.ContentBlock{{Type: "informational", Content: "Welcome! Type a message and press Enter."}},
	})

	inputCh := term.ReadStdin()
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	var inputBuf []rune
	cursor := 0

	for {
		select {
		case data, ok := <-inputCh:
			if !ok {
				return true
			}
			s := string(data)
			runes := []rune(s)

			for i := 0; i < len(runes); i++ {
				r := runes[i]
				switch {
				case r == 3: // Ctrl+C
					return true

				case r == 13: // Enter
					if len(inputBuf) > 0 {
						text := string(inputBuf)
						store.AppendMessage(ink.Message{
							UUID: fmt.Sprintf("u%d", time.Now().UnixNano()),
							Type: "user",
							ContentBlocks: []ink.ContentBlock{{Type: "text", Content: text}},
						})
						store.AppendMessage(ink.Message{
							UUID: fmt.Sprintf("a%d", time.Now().UnixNano()),
							Type: "assistant",
							ContentBlocks: []ink.ContentBlock{
								{Type: "text", Content: "You said: **" + text + "**  \n_(API not yet wired — local echo)_"},
							},
						})
						inputBuf = nil
						cursor = 0
						store.SetInputValue("")
						store.SetCursorPos(0)
					}

				case r == 127: // Backspace
					if cursor > 0 && len(inputBuf) > 0 {
						inputBuf = append(inputBuf[:cursor-1], inputBuf[cursor:]...)
						cursor--
						store.SetInputValue(string(inputBuf))
						store.SetCursorPos(cursor)
					}

				case r == '\x1b': // ANSI escape sequence
					if i+1 < len(runes) && runes[i+1] == '[' {
						i += 2
						seq := ""
						for i < len(runes) && runes[i] >= '0' && runes[i] <= '?' {
							seq += string(runes[i])
							i++
						}
						if i < len(runes) {
							term := runes[i]
							switch term {
							case 'D': // Left arrow
								if cursor > 0 {
									cursor--
									store.SetCursorPos(cursor)
								}
							case 'C': // Right arrow
								if cursor < len(inputBuf) {
									cursor++
									store.SetCursorPos(cursor)
								}
							case 'H': // Home
								cursor = 0
								store.SetCursorPos(cursor)
							case 'F': // End
								cursor = len(inputBuf)
								store.SetCursorPos(cursor)
							case '3': // Delete key (CSI 3~)
								if i < len(runes) && runes[i] == '~' {
									if cursor < len(inputBuf) {
										inputBuf = append(inputBuf[:cursor], inputBuf[cursor+1:]...)
										store.SetInputValue(string(inputBuf))
										store.SetCursorPos(cursor)
									}
								}
							}
						}
					}

				default:
					if r >= 32 { // printable char
						inputBuf = append(inputBuf[:cursor], append([]rune{r}, inputBuf[cursor:]...)...)
						cursor++
						store.SetInputValue(string(inputBuf))
						store.SetCursorPos(cursor)
					}
				}
			}

		case <-sigCh:
			return true
		}
	}
}
