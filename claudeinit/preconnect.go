package claudeinit

import (
	"context"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

var (
	preconnectMu    sync.Mutex
	preconnectFired bool
)

func firePreconnectAnthropicAPI() {
	preconnectMu.Lock()
	if preconnectFired {
		preconnectMu.Unlock()
		return
	}
	preconnectFired = true
	preconnectMu.Unlock()

	if !shouldPreconnectAnthropic() {
		return
	}
	url := anthropicAPIBaseURLForPreconnect()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
		if err != nil {
			return
		}
		c := &http.Client{Timeout: 10 * time.Second}
		_, _ = c.Do(req)
	}()
}

func preconnectResetForTesting() {
	preconnectMu.Lock()
	defer preconnectMu.Unlock()
	preconnectFired = false
}

func shouldPreconnectAnthropic() bool {
	if envTruthy("CLAUDE_CODE_USE_BEDROCK") || envTruthy("CLAUDE_CODE_USE_VERTEX") || envTruthy("CLAUDE_CODE_USE_FOUNDRY") {
		return false
	}
	if envTruthy("CLAUDE_CODE_USE_OPENAI") || envTruthy("CLAUDE_CODE_INTRANET_MODE") {
		return false
	}
	if proxyEnvSet() {
		return false
	}
	switch strings.TrimSpace(os.Getenv("ANTHROPIC_UNIX_SOCKET")) {
	case "":
	default:
		return false
	}
	if strings.TrimSpace(os.Getenv("CLAUDE_CODE_CLIENT_CERT")) != "" || strings.TrimSpace(os.Getenv("CLAUDE_CODE_CLIENT_KEY")) != "" {
		return false
	}
	return true
}

func proxyEnvSet() bool {
	for _, k := range []string{"HTTPS_PROXY", "https_proxy", "HTTP_PROXY", "http_proxy"} {
		if strings.TrimSpace(os.Getenv(k)) != "" {
			return true
		}
	}
	return false
}

func envTruthy(k string) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(k)))
	switch v {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func anthropicAPIBaseURLForPreconnect() string {
	u := strings.TrimSpace(os.Getenv("ANTHROPIC_BASE_URL"))
	if u == "" {
		return "https://api.anthropic.com"
	}
	if !strings.Contains(u, "://") {
		u = "https://" + u
	}
	return u
}
