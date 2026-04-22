package memoize

// LRUMemo exposes the cache surface of TS memoizeWithLRU.
type LRUMemo[R any] struct {
	Clear  func()
	Size   func() int
	Delete func(key string) bool
	// Get returns a cached value without promoting recency (TS cache.peek for observation).
	Get func(key string) (R, bool)
	Has func(key string) bool
}

// MemoizeWithLRU mirrors src/utils/memoize.ts memoizeWithLRU (default max=100 in TS if needed check — TS default 100).
func MemoizeWithLRU[Arg any, R any](f func(Arg) R, keyFn func(Arg) string, maxCacheSize int) (func(Arg) R, LRUMemo[R]) {
	if maxCacheSize < 1 {
		maxCacheSize = 100
	}
	c := newStringLRU[R](maxCacheSize)
	memo := func(arg Arg) R {
		k := keyFn(arg)
		if v, ok := c.getMRU(k); ok {
			return v
		}
		r := f(arg)
		c.set(k, r)
		return r
	}
	ctl := LRUMemo[R]{
		Clear:  c.clear,
		Size:   c.size,
		Delete: c.del,
		Get:    c.peek,
		Has:    c.has,
	}
	return memo, ctl
}
