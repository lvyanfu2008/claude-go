// Mirrors src/tools/shared/gitOperationTracking.ts (exported string unions).
package types

// CommitKind mirrors gitOperationTracking.ts CommitKind.
type CommitKind string

const (
	CommitKindCommitted    CommitKind = "committed"
	CommitKindAmended      CommitKind = "amended"
	CommitKindCherryPicked CommitKind = "cherry-picked"
)

// BranchAction mirrors gitOperationTracking.ts BranchAction.
type BranchAction string

const (
	BranchActionMerged  BranchAction = "merged"
	BranchActionRebased BranchAction = "rebased"
)

// PrAction mirrors gitOperationTracking.ts PrAction.
type PrAction string

const (
	PrActionCreated   PrAction = "created"
	PrActionEdited    PrAction = "edited"
	PrActionMerged    PrAction = "merged"
	PrActionCommented PrAction = "commented"
	PrActionClosed    PrAction = "closed"
	PrActionReady     PrAction = "ready"
)
