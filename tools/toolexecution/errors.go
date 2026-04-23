package toolexecution

import "errors"

// ErrPipelineNotImplemented is returned by skeleton branches of CheckPermissionsAndCallTool
// until parity work fills them in (toolExecution.ts checkPermissionsAndCallTool).
var ErrPipelineNotImplemented = errors.New("toolexecution: CheckPermissionsAndCallTool not fully implemented")
