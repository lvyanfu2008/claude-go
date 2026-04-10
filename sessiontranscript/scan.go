package sessiontranscript

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"strings"
)

// MaxTranscriptReadBytes matches MAX_TRANSCRIPT_READ_BYTES in sessionStorage.ts (50 MiB).
const MaxTranscriptReadBytes = 50 * 1024 * 1024

// MessageUUIDsFromJSONL returns UUIDs of lines that parse as JSON objects with a string "uuid" field.
func MessageUUIDsFromJSONL(path string) (map[string]struct{}, error) {
	st, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]struct{}{}, nil
		}
		return nil, err
	}
	if st.Size() > MaxTranscriptReadBytes {
		return nil, errTranscriptTooLarge(st.Size())
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	out := make(map[string]struct{})
	s := bufio.NewScanner(bytes.NewReader(data))
	// Lines can be large; allow bigger buffer
	buf := make([]byte, 0, 64*1024)
	s.Buffer(buf, 1024*1024)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" {
			continue
		}
		var obj struct {
			UUID string `json:"uuid"`
		}
		if json.Unmarshal([]byte(line), &obj) != nil || obj.UUID == "" {
			continue
		}
		out[obj.UUID] = struct{}{}
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

type transcriptTooLargeError struct {
	size int64
}

func (e transcriptTooLargeError) Error() string {
	return "sessiontranscript: transcript file exceeds MaxTranscriptReadBytes"
}

func errTranscriptTooLarge(size int64) error {
	return transcriptTooLargeError{size: size}
}
