package workflow

import (
	"math"
	"sync"
)

// BudgetTracker tracks token spending for a workflow execution.
// Methods spent() and remaining() are exposed to JS via the budget global.
type BudgetTracker struct {
	total  int64
	spent  int64
	mu     sync.Mutex
}

// NewBudgetTracker creates a BudgetTracker. total=0 means no budget (infinite).
func NewBudgetTracker(total int64) *BudgetTracker {
	return &BudgetTracker{total: total}
}

// Total returns the budget cap (0 = unlimited).
func (b *BudgetTracker) Total() int64 {
	return b.total
}

// Spent returns the current spent token count.
func (b *BudgetTracker) Spent() int64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.spent
}

// Remaining returns max(0, total - spent), or MaxInt64 if no budget set.
func (b *BudgetTracker) Remaining() int64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.total <= 0 {
		return math.MaxInt64
	}
	remaining := b.total - b.spent
	if remaining < 0 {
		return 0
	}
	return remaining
}

// IsExhausted reports whether the budget has been exhausted.
func (b *BudgetTracker) IsExhausted() bool {
	if b.total <= 0 {
		return false
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.spent >= b.total
}

// AddSpent increments the spent counter atomically.
func (b *BudgetTracker) AddSpent(n int64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.spent += n
}
