package toolpool

import "goc/types"

// AssembleToolPoolFromEmbedded mirrors assembleToolPool when built-ins come from embedded
// tools_api.json: GetTools(permissionContext, embedded) then AssembleToolPool (src/tools.ts).
func AssembleToolPoolFromEmbedded(permissionContext types.ToolPermissionContextData, mcpTools []types.ToolSpec) ([]types.ToolSpec, error) {
	base := ToolSpecsFromGoWire()
	builtIn := GetTools(permissionContext, base)
	var err error
	builtIn, err = ReplaceBashToolSpecIfZogMode(builtIn)
	if err != nil {
		return nil, err
	}
	return AssembleToolPool(permissionContext, builtIn, mcpTools), nil
}
