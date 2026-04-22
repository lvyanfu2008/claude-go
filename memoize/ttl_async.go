package memoize

import (
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/singleflight"
)

// MemoizeWithTTLAsync mirrors src/utils/memoize.ts memoizeWithTTLAsync (SWR, cold coalescing via
// [singleflight.Group], identity-safe across [clear] with a generation counter).
func MemoizeWithTTLAsync[Arg any, R any](f func(Arg) (R, error), key func(Arg) string, cacheLifetime time.Duration) (func(Arg) (R, error), func()) {
	if cacheLifetime <= 0 {
		cacheLifetime = DefaultCacheLifetime
	}
	var (
		mu     sync.Mutex
		cache  = make(map[string]*entryTTL[R])
		flight singleflight.Group
		vers   int64 // increment on every clear; stale writes skip if version moved
	)

	memo := func(arg Arg) (R, error) {
		var z R
		k := key(arg)
		now := time.Now()

		mu.Lock()
		c := cache[k]
		if c == nil {
			vgen := atomic.LoadInt64(&vers)
			mu.Unlock()
			anyV, err, _ := flight.Do(k, func() (any, error) {
				return f(arg)
			})
			if err != nil {
				return z, err
			}
			rr := anyV.(R)
			if atomic.LoadInt64(&vers) != vgen {
				// [clear] happened during f — do not re-populate a cleared cache (TS inFlight/finally).
				return rr, nil
			}
			now2 := time.Now()
			mu.Lock()
			if atomic.LoadInt64(&vers) != vgen {
				mu.Unlock()
				return rr, nil
			}
			if cache[k] == nil {
				cache[k] = &entryTTL[R]{value: rr, at: now2, refreshing: false}
			} else {
				rr = cache[k].value
			}
			mu.Unlock()
			return rr, nil
		}

		if now.Sub(c.at) > cacheLifetime && !c.refreshing {
			c.refreshing = true
			probe := c
			vgen := atomic.LoadInt64(&vers)
			argCopy := arg
			mu.Unlock()
			go func() {
				newR, err := f(argCopy)
				if atomic.LoadInt64(&vers) != vgen {
					return
				}
				mu.Lock()
				defer mu.Unlock()
				if err != nil {
					if cur, ok := cache[k]; ok && cur == probe {
						delete(cache, k)
					}
					return
				}
				if cur, ok := cache[k]; ok && cur == probe {
					cur.value = newR
					cur.at = time.Now()
					cur.refreshing = false
				}
			}()
			return c.value, nil
		}
		v := c.value
		mu.Unlock()
		return v, nil
	}

	clear := func() {
		atomic.AddInt64(&vers, 1)
		mu.Lock()
		for k2 := range cache {
			delete(cache, k2)
		}
		mu.Unlock()
		// singleflight has no public reset; a bumped version prevents stale writes; next Do is fresh.
		flight = singleflight.Group{}
	}
	return memo, clear
}
