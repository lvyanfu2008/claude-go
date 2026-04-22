package toolpool

import "goc/types"

// AssembleToolPoolFromGoWire mirrors assembleToolPool when built-ins come from Go native providers:
// GetTools(permissionContext, ToolSpecsFromGoWire()) then AssembleToolPool (src/tools.ts).
func AssembleToolPoolFromGoWire(permissionContext types.ToolPermissionContextData, mcpTools []types.ToolSpec) ([]types.ToolSpec, error) {
	base := ToolSpecsFromGoWire()
	builtIn := GetTools(permissionContext, base)
	var err error
	builtIn, err = ReplaceBashToolSpecIfZogMode(builtIn)
	if err != nil {
		return nil, err
	}
	return AssembleToolPool(permissionContext, builtIn, mcpTools), nil
}
