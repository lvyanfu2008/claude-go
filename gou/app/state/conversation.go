package state

import (
	"goc/appstate"
	"goc/gou/conversation"
	"goc/gou/messagerow"
	"goc/sessiontranscript"
	"goc/tools/localtools"
	"goc/tscontext"
)

type Conversation struct {
	Store               *conversation.Store
	Transcript          *sessiontranscript.Store
	ResolvedToolIDs     map[string]struct{}
	GroupedAgentLookups *messagerow.GroupedAgentLookups
	ReadFileState       *localtools.ReadFileState
	AppStateStore       *appstate.Store
	TSBridge            *tscontext.Snapshot
}
