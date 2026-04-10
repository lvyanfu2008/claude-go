// Parameter and helper result types for [ProcessUserInput] and the process-user-input CLI JSON body.
package processuserinput

import (
	"encoding/json"

	"goc/types"
	"goc/utils"
)

// ProcessUserInputArgs is the JSON "args" object for the process-user-input CLI (field names match src/conversation-runtime/processUserInput for parity).
// Function-typed fields (setToolJSX, setUserInputOnProcessing, canUseTool) are omitted.
type ProcessUserInputArgs struct {
	// Input is either a JSON string or an array of ContentBlockParam.
	Input json.RawMessage `json:"input"`

	PreExpansionInput *string `json:"preExpansionInput,omitempty"`
	Mode              types.PromptInputMode `json:"mode"`

	PastedContents map[string]utils.PastedContent `json:"pastedContents,omitempty"`
	// Keys are decimal string indices matching TS Record<number, PastedContent>.

	IdeSelection *types.IDESelection `json:"ideSelection,omitempty"`
	Messages     []types.Message     `json:"messages,omitempty"`

	UUID *string `json:"uuid,omitempty"`

	IsAlreadyProcessing *bool `json:"isAlreadyProcessing,omitempty"`
	QuerySource         types.QuerySource `json:"querySource,omitempty"`

	SkipSlashCommands *bool `json:"skipSlashCommands,omitempty"`
	BridgeOrigin        *bool `json:"bridgeOrigin,omitempty"`
	IsMeta              *bool `json:"isMeta,omitempty"`
	SkipAttachments     *bool `json:"skipAttachments,omitempty"`

	// Context is the serializable snapshot; when building [ProcessUserInputParams], copy to [ProcessUserInputParams.RuntimeContext].
	Context types.ProcessUserInputContextData `json:"context"`

	// BridgeAttachmentMessages optional pre-resolved @-attachment messages (skip [GetAttachmentMessages] when non-nil).
	// JSON null or omitted: Go falls back to [GetAttachmentMessages] when that callback is set; CLI typically leaves both unset → no attachments.
	BridgeAttachmentMessages *[]types.Message `json:"bridgeAttachmentMessages,omitempty"`
	StatePatchAck            *StatePatchAck   `json:"statePatchAck,omitempty"`
}

// ProcessTextPromptResult mirrors processTextPrompt() return type (processTextPrompt.ts).
type ProcessTextPromptResult struct {
	Messages    []types.Message `json:"messages"`
	ShouldQuery bool            `json:"shouldQuery"`
}
