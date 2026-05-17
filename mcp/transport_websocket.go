package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
)

// WebSocketTransport implements transport.Interface over a WebSocket connection.
type WebSocketTransport struct {
	url     string
	headers map[string]string

	mu        sync.Mutex
	conn      *websocket.Conn
	sessionID string
	closed    bool

	// Response routing: maps request ID -> response channel.
	pending  map[interface{}]chan *transport.JSONRPCResponse
	pendMu   sync.Mutex
	notifyFn func(mcp.JSONRPCNotification)

	readDone chan struct{}
}

// NewWebSocketTransport creates a WebSocket transport for MCP.
func NewWebSocketTransport(url string, headers map[string]string) *WebSocketTransport {
	return &WebSocketTransport{
		url:      url,
		headers:  headers,
		pending:  make(map[interface{}]chan *transport.JSONRPCResponse),
		readDone: make(chan struct{}),
	}
}

// Start dials the WebSocket server and begins the read loop.
func (t *WebSocketTransport) Start(ctx context.Context) error {
	dialer := &websocket.Dialer{
		Subprotocols: []string{"mcp"},
	}
	header := http.Header{}
	for k, v := range t.headers {
		header.Set(k, v)
	}

	conn, resp, err := dialer.DialContext(ctx, t.url, header)
	if err != nil {
		return fmt.Errorf("websocket dial: %w", err)
	}
	if resp != nil && resp.Body != nil {
		resp.Body.Close()
	}

	t.mu.Lock()
	t.conn = conn
	t.closed = false
	t.mu.Unlock()

	// Extract session ID from response headers.
	if resp != nil {
		if sid := resp.Header.Get("Mcp-Session-Id"); sid != "" {
			t.sessionID = sid
		}
	}

	go t.readLoop()
	return nil
}

func (t *WebSocketTransport) readLoop() {
	defer close(t.readDone)
	for {
		t.mu.Lock()
		conn := t.conn
		t.mu.Unlock()
		if conn == nil {
			return
		}

		_, message, err := conn.ReadMessage()
		if err != nil {
			return // connection closed
		}

		// Try to parse as JSON-RPC response first.
		var resp transport.JSONRPCResponse
		if err := json.Unmarshal(message, &resp); err == nil && resp.JSONRPC != "" {
			t.pendMu.Lock()
			ch, ok := t.pending[resp.ID]
			if ok {
				delete(t.pending, resp.ID)
			}
			t.pendMu.Unlock()
			if ok {
				ch <- &resp
			}
			continue
		}

		// Try to parse as JSON-RPC notification.
		var notif mcp.JSONRPCNotification
		if err := json.Unmarshal(message, &notif); err == nil && notif.Method != "" {
			t.pendMu.Lock()
			fn := t.notifyFn
			t.pendMu.Unlock()
			if fn != nil {
				fn(notif)
			}
		}
	}
}

// SendRequest sends a JSON-RPC request and waits for a response.
func (t *WebSocketTransport) SendRequest(ctx context.Context, req transport.JSONRPCRequest) (*transport.JSONRPCResponse, error) {
	t.mu.Lock()
	conn := t.conn
	t.mu.Unlock()
	if conn == nil {
		return nil, fmt.Errorf("websocket: not connected")
	}

	ch := make(chan *transport.JSONRPCResponse, 1)
	t.pendMu.Lock()
	t.pending[req.ID] = ch
	t.pendMu.Unlock()

	defer func() {
		t.pendMu.Lock()
		delete(t.pending, req.ID)
		t.pendMu.Unlock()
	}()

	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	t.mu.Lock()
	err = conn.WriteMessage(websocket.TextMessage, body)
	t.mu.Unlock()
	if err != nil {
		return nil, fmt.Errorf("websocket write: %w", err)
	}

	select {
	case resp := <-ch:
		return resp, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(30 * time.Second):
		return nil, fmt.Errorf("websocket: request timeout")
	}
}

// SendNotification sends a JSON-RPC notification.
func (t *WebSocketTransport) SendNotification(ctx context.Context, notif mcp.JSONRPCNotification) error {
	t.mu.Lock()
	conn := t.conn
	t.mu.Unlock()
	if conn == nil {
		return fmt.Errorf("websocket: not connected")
	}

	body, err := json.Marshal(notif)
	if err != nil {
		return err
	}

	t.mu.Lock()
	err = conn.WriteMessage(websocket.TextMessage, body)
	t.mu.Unlock()
	if err != nil {
		return fmt.Errorf("websocket write: %w", err)
	}
	return nil
}

// SetNotificationHandler sets the handler for incoming notifications.
func (t *WebSocketTransport) SetNotificationHandler(fn func(mcp.JSONRPCNotification)) {
	t.pendMu.Lock()
	t.notifyFn = fn
	t.pendMu.Unlock()
}

// Close closes the WebSocket connection.
func (t *WebSocketTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed {
		return nil
	}
	t.closed = true
	if t.conn != nil {
		_ = t.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		_ = t.conn.Close()
		t.conn = nil
	}
	return nil
}

// GetSessionId returns the MCP session ID.
func (t *WebSocketTransport) GetSessionId() string {
	return t.sessionID
}
