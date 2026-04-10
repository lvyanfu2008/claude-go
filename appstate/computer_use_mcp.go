package appstate

// ComputerUseMcpAllowedApp mirrors AppStateStore computerUseMcpState.allowedApps[].
type ComputerUseMcpAllowedApp struct {
	BundleID    string `json:"bundleId"`
	DisplayName string `json:"displayName"`
	GrantedAt   int64  `json:"grantedAt"`
}

// ComputerUseMcpGrantFlags mirrors computerUseMcpState.grantFlags.
type ComputerUseMcpGrantFlags struct {
	ClipboardRead   bool `json:"clipboardRead"`
	ClipboardWrite  bool `json:"clipboardWrite"`
	SystemKeyCombos bool `json:"systemKeyCombos"`
}

// ComputerUseMcpLastScreenshotDims mirrors computerUseMcpState.lastScreenshotDims.
type ComputerUseMcpLastScreenshotDims struct {
	Width         int  `json:"width"`
	Height        int  `json:"height"`
	DisplayWidth  int  `json:"displayWidth"`
	DisplayHeight int  `json:"displayHeight"`
	DisplayID     *int `json:"displayId,omitempty"`
	OriginX       *int `json:"originX,omitempty"`
	OriginY       *int `json:"originY,omitempty"`
}

// ComputerUseMcpState mirrors AppStateStore.ts computerUseMcpState (JSON-safe).
// TS hiddenDuringTurn is ReadonlySet<string>; serialized as a string slice.
type ComputerUseMcpState struct {
	AllowedApps            []ComputerUseMcpAllowedApp        `json:"allowedApps,omitempty"`
	GrantFlags             *ComputerUseMcpGrantFlags         `json:"grantFlags,omitempty"`
	LastScreenshotDims     *ComputerUseMcpLastScreenshotDims `json:"lastScreenshotDims,omitempty"`
	HiddenDuringTurn       []string                          `json:"hiddenDuringTurn,omitempty"`
	SelectedDisplayID      *int                              `json:"selectedDisplayId,omitempty"`
	DisplayPinnedByModel   *bool                             `json:"displayPinnedByModel,omitempty"`
	DisplayResolvedForApps string                            `json:"displayResolvedForApps,omitempty"`
}
