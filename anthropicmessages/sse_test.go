package anthropicmessages

import (
	"strings"
	"testing"
)

func TestReadSSE_multipleEvents(t *testing.T) {
	body := "data: {\"type\":\"ping\"}\n\n" +
		"data: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"end_turn\"}}\n\n" +
		"data: {\"type\":\"message_stop\"}\n\n"
	var got []string
	err := ReadSSE(strings.NewReader(body), func(data []byte) error {
		ev, err := DecodeStreamPayload(data)
		if err != nil {
			return err
		}
		got = append(got, ev.Type)
		if ev.Type == "message_stop" {
			return ErrMessageStreamDone
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 || got[0] != "ping" || got[1] != "message_delta" || got[2] != "message_stop" {
		t.Fatalf("got %#v", got)
	}
}
