package state

import "goc/types"

type Chrome struct {
	PermissionMode        types.PermissionMode
	LastEmittedTitlePlain string
	LastMainLoopModel     string
}
