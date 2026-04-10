package markdown

// TokenCacheMax mirrors Markdown.tsx TOKEN_CACHE_MAX.
const TokenCacheMax = 500

// TokenCache is an LRU-ish map (promote on hit like TS tokenCache delete+set).
type TokenCache struct {
	max int
	m   map[string][]Token
}

// NewTokenCache returns a cache with at most max entries (oldest evicted).
func NewTokenCache(max int) *TokenCache {
	if max <= 0 {
		max = TokenCacheMax
	}
	return &TokenCache{max: max, m: make(map[string][]Token)}
}

// Get returns a copy of cached tokens.
func (c *TokenCache) Get(key string) ([]Token, bool) {
	toks, ok := c.m[key]
	if !ok {
		return nil, false
	}
	out := make([]Token, len(toks))
	copy(out, toks)
	return out, true
}

// Put stores tokens; promotes key to MRU; evicts FIFO when over capacity.
func (c *TokenCache) Put(key string, tokens []Token) {
	if c.m == nil {
		c.m = make(map[string][]Token)
	}
	delete(c.m, key)
	cp := make([]Token, len(tokens))
	copy(cp, tokens)
	c.m[key] = cp
	for len(c.m) > c.max {
		for k := range c.m {
			delete(c.m, k)
			break
		}
	}
}

// globalCache mirrors module-level tokenCache in Markdown.tsx.
var globalCache = NewTokenCache(TokenCacheMax)

// SetGlobalCacheForTest replaces the package cache (tests only).
func SetGlobalCacheForTest(c *TokenCache) {
	globalCache = c
}
