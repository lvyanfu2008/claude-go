package memoize

import (
	"container/list"
	"sync"
)

// stringLRU is a thread-safe LRU of string->V with a bounded size (TS lru-cache max + peek).
type stringLRU[V any] struct {
	mu   sync.Mutex
	max  int
	li   *list.List
	keys map[string]*list.Element
}

type kvPair[V any] struct {
	k string
	v V
}

func newStringLRU[V any](max int) *stringLRU[V] {
	if max < 1 {
		max = 1
	}
	return &stringLRU[V]{
		max:  max,
		li:   list.New(),
		keys: make(map[string]*list.Element),
	}
}

func (c *stringLRU[V]) getMRU(key string) (V, bool) {
	var z V
	c.mu.Lock()
	defer c.mu.Unlock()
	el, ok := c.keys[key]
	if !ok {
		return z, false
	}
	c.li.MoveToFront(el)
	return el.Value.(kvPair[V]).v, true
}

// peek does not change recency (TS lru-cache peek).
func (c *stringLRU[V]) peek(key string) (V, bool) {
	var z V
	c.mu.Lock()
	defer c.mu.Unlock()
	el, ok := c.keys[key]
	if !ok {
		return z, false
	}
	return el.Value.(kvPair[V]).v, true
}

func (c *stringLRU[V]) set(key string, v V) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.keys[key]; ok {
		c.li.MoveToFront(el)
		el.Value = kvPair[V]{k: key, v: v}
		return
	}
	for c.li.Len() >= c.max {
		back := c.li.Back()
		if back == nil {
			break
		}
		old := back.Value.(kvPair[V])
		delete(c.keys, old.k)
		c.li.Remove(back)
	}
	front := c.li.PushFront(kvPair[V]{k: key, v: v})
	c.keys[key] = front
}

func (c *stringLRU[V]) clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.li.Init()
	c.keys = make(map[string]*list.Element)
}

func (c *stringLRU[V]) size() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.li.Len()
}

func (c *stringLRU[V]) has(key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, ok := c.keys[key]
	return ok
}

func (c *stringLRU[V]) del(key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	el, ok := c.keys[key]
	if !ok {
		return false
	}
	delete(c.keys, key)
	c.li.Remove(el)
	return true
}
