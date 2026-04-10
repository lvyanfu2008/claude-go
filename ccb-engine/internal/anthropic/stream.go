package anthropic

// Streaming Messages API (SSE) parsing is not implemented in this milestone.
// Non-streaming CreateMessage in client.go is used by the engine; see spec/protocol-v1.md
// for StreamEvent shapes the TS client will eventually consume from a socket.
//
// Future: POST /v1/messages with "stream":true, parse event: / data: frames per Anthropic docs.
