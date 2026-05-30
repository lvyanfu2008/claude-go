package ink

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/term"
)

type Terminal struct {
	stdin         *os.File
	stdout        *os.File
	prevState     *term.State
	width, height int
	resizeCh      chan [2]int
	restoreFuncs  []func()
}

func NewTerminal() *Terminal {
	return &Terminal{
		stdin:    os.Stdin,
		stdout:   os.Stdout,
		resizeCh: make(chan [2]int, 8),
	}
}

func (t *Terminal) Init() error {
	fd := int(t.stdin.Fd())
	state, err := term.MakeRaw(fd)
	if err != nil {
		return fmt.Errorf("make raw: %w", err)
	}
	t.prevState = state
	t.restoreFuncs = append(t.restoreFuncs, func() {
		term.Restore(fd, state)
	})

	// alternate screen
	fmt.Fprint(t.stdout, "\x1b[?1049h")
	t.restoreFuncs = append(t.restoreFuncs, func() {
		fmt.Fprint(t.stdout, "\x1b[?1049l")
	})

	// hide cursor
	fmt.Fprint(t.stdout, "\x1b[?25l")
	t.restoreFuncs = append(t.restoreFuncs, func() {
		fmt.Fprint(t.stdout, "\x1b[?25h")
	})

	// enable SGR mouse tracking
	fmt.Fprint(t.stdout, "\x1b[?1000h\x1b[?1002h\x1b[?1006h")
	t.restoreFuncs = append(t.restoreFuncs, func() {
		fmt.Fprint(t.stdout, "\x1b[?1006l\x1b[?1002l\x1b[?1000l")
	})

	w, h, err := term.GetSize(fd)
	if err == nil {
		t.width, t.height = w, h
	}

	go t.handleSignals()
	return nil
}

func (t *Terminal) handleSignals() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGWINCH)
	for range sigCh {
		w, h, err := term.GetSize(int(t.stdin.Fd()))
		if err != nil {
			continue
		}
		t.width, t.height = w, h
		select {
		case t.resizeCh <- [2]int{w, h}:
		default:
		}
	}
}

func (t *Terminal) Size() (int, int) { return t.width, t.height }

func (t *Terminal) ResizeEvents() <-chan [2]int { return t.resizeCh }

func (t *Terminal) Shutdown() {
	for i := len(t.restoreFuncs) - 1; i >= 0; i-- {
		t.restoreFuncs[i]()
	}
	signal.Stop(make(chan os.Signal, 1))
}

func (t *Terminal) Write(data []byte) (int, error) {
	return t.stdout.Write(data)
}
