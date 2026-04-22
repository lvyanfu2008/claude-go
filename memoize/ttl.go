package memoize

import (
	"sync"
	"time"
)

// DefaultCacheLifetime matches TS default 5 * 60 * 1000 ms in memoizeWithTTL.
const DefaultCacheLifetime = 5 * time.Minute

// MemoizeWithTTL implements src/utils/memoize.ts memoizeWithTTL: stale-while-revalidate, identity-guarded.
func MemoizeWithTTL[Arg any, R any](f func(Arg) R, key func(Arg) string, cacheLifetime time.Duration) (func(Arg) R, func()) {
	if cacheLifetime <= 0 {
		cacheLifetime = DefaultCacheLifetime
	}
	var mu sync.Mutex
	cache := make(map[string]*entryTTL[R])

	memo := func(arg Arg) R {
		k := key(arg)
		now := time.Now()

		mu.Lock()
		c := cache[k]
		if c == nil {
			v := f(arg)
			cache[k] = &entryTTL[R]{value: v, at: now, refreshing: false}
			mu.Unlock()
			return v
		}

		if now.Sub(c.at) > cacheLifetime && !c.refreshing {
			c.refreshing = true
			probe := c
			argCopy := arg
			mu.Unlock()

			go func() {
				defer func() {
					if rec := recover(); rec != nil {
						mu.Lock()
						if cur, ok := cache[k]; ok && cur == probe {
							delete(cache, k)
						}
						mu.Unlock()
					}
				}()
				newV := f(argCopy)
				mu.Lock()
				defer mu.Unlock()
				if cur, ok := cache[k]; ok && cur == probe {
					cur.value = newV
					cur.at = time.Now()
					cur.refreshing = false
				}
			}()
			return c.value
		}

		v := c.value
		mu.Unlock()
		return v
	}

	clear := func() {
		mu.Lock()
		defer mu.Unlock()
		for k := range cache {
			delete(cache, k)
		}
	}
	return memo, clear
}

type entryTTL[R any] struct {
	value      R
	at         time.Time
	refreshing bool
}
