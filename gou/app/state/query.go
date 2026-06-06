package state

import (
	"context"
	"time"

	"goc/conversation-runtime/query"
	"goc/gou/pui"
)

type Query struct {
	CCBSend         func(msg interface{})
	CCBInline       bool
	Busy            bool
	BusyStartedAt   time.Time
	LastActivity    time.Time
	SpinnerVerb     string
	SpinnerFrame    int
	SpinnerTokens   int
	PreCompactVerb  string
	Cancel          context.CancelFunc
	LastCtrlC       time.Time
	CtrlCPending    bool
	LastQueryParams *query.QueryParams
	Handoff         pui.ProcessUserInputBaseResultHandoff
}
