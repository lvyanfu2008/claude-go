package ccbstream

import (
	"bufio"
	"encoding/json"
	"io"
	"os"

	"goc/gou/conversation"
)

// ReplayFile reads NDJSON stream events from path and applies them in order.
func ReplayFile(path string, store *conversation.Store) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return ReplayReader(f, store)
}

// ReplayReader applies each JSON line as StreamEvent (ignores malformed lines).
func ReplayReader(r io.Reader, store *conversation.Store) error {
	s := bufio.NewScanner(r)
	buf := make([]byte, 0, 64*1024)
	s.Buffer(buf, 1024*1024)
	for s.Scan() {
		line := s.Bytes()
		if len(line) == 0 {
			continue
		}
		var ev StreamEvent
		if err := json.Unmarshal(line, &ev); err != nil || ev.Type == "" {
			continue
		}
		Apply(store, ev)
	}
	return s.Err()
}
