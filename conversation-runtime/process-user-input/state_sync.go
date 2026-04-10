package processuserinput

// StatePatch models Go->TS state sync proposals for hook-related runtime state.
type StatePatch struct {
	Op      string         `json:"op"`
	Payload map[string]any `json:"payload,omitempty"`
}

// StatePatchBatch is emitted by Go and applied by TS with version checks.
type StatePatchBatch struct {
	PatchID     string       `json:"patchId"`
	BaseVersion int          `json:"baseVersion"`
	Patches     []StatePatch `json:"patches"`
}

// StatePatchAck is sent by TS on subsequent requests after apply attempt.
type StatePatchAck struct {
	PatchID    string `json:"patchId"`
	Applied    bool   `json:"applied"`
	Reason     string `json:"reason,omitempty"`
	NewVersion int    `json:"newVersion"`
}
