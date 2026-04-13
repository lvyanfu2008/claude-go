package anthropicmessages

import (
	"bytes"
	"io"
)

// MaxStreamBodyLogBytes caps raw SSE bytes logged for one stream when
// CLAUDE_CODE_LOG_API_RESPONSE_BODY is on (avoids unbounded memory).
const MaxStreamBodyLogBytes = 32 << 20

// NewStreamBodyReadTee wraps rc so each Read copies up to max bytes into buf
// (for apilog). If buf is nil or max <= 0, returns rc unchanged.
func NewStreamBodyReadTee(rc io.ReadCloser, buf *bytes.Buffer, max int) io.ReadCloser {
	if buf == nil || max <= 0 {
		return rc
	}
	return &streamReadTee{rc: rc, buf: buf, max: max}
}

// streamReadTee tees raw response bytes into buf up to max (for apilog).
type streamReadTee struct {
	rc  io.ReadCloser
	buf *bytes.Buffer
	max int
}

func (s *streamReadTee) Read(p []byte) (int, error) {
	n, err := s.rc.Read(p)
	if n > 0 && s.buf != nil && s.buf.Len() < s.max {
		room := s.max - s.buf.Len()
		if room > 0 {
			if n <= room {
				_, _ = s.buf.Write(p[:n])
			} else {
				_, _ = s.buf.Write(p[:room])
			}
		}
	}
	return n, err
}

func (s *streamReadTee) Close() error {
	return s.rc.Close()
}
