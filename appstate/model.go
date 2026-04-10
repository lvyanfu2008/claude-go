package appstate

// ModelSetting mirrors src/utils/model/model.ts ModelSetting (ModelName | ModelAlias | null).
// Nil means TS null (session/product default model). Non-nil string is full id or alias token.
type ModelSetting = *string
