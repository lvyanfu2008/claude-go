package appstate

import "encoding/json"

// EmptySettingsJSON is the default merged settings object (TS getInitialSettings() fallback {}).
var EmptySettingsJSON = json.RawMessage(`{}`)
