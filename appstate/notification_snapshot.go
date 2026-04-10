package appstate

import "encoding/json"

// NotificationSnapshot mirrors context/notifications.tsx Notification.
// fold (merge function) is omitted; jsx is opaque (React nodes are not JSON).
type NotificationSnapshot struct {
	Key         string          `json:"key"`
	Invalidates []string        `json:"invalidates,omitempty"`
	Priority    string          `json:"priority"`
	TimeoutMs   *int            `json:"timeoutMs,omitempty"`
	Text        string          `json:"text,omitempty"`
	Color       string          `json:"color,omitempty"`
	JSX         json.RawMessage `json:"jsx,omitempty"`
}
