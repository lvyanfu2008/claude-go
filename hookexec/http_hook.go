package hookexec

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"goc/tools/hookstypes"
)

// DefaultHttpHookTimeoutMs mirrors HTTP_HOOK_DEFAULT_TIMEOUT_MS in TS execHttpHook.ts (10 minutes).
const DefaultHttpHookTimeoutMs = 10 * 60 * 1000

// parseHookTimeoutMs extracts the timeout in milliseconds from a HookCommand.
// Returns DefaultHttpHookTimeoutMs if not set or invalid.
func parseHookTimeoutMs(h hookstypes.HookCommand) int {
	if h.Timeout > 0 {
		ms := int(h.Timeout * 1000)
		if ms > 30*60*1000 {
			return 30 * 60 * 1000
		}
		if ms < 1000 {
			return 1000
		}
		return ms
	}
	return DefaultHttpHookTimeoutMs
}

// RunHttpHook executes an HTTP hook: POSTs the JSON input to the configured URL.
// Mirrors TS execHttpHook in execHttpHook.ts.
//
// Returns an OutsideReplHookResult. On success, stdout contains the response body.
// On blocking JSON response (decision: "block"), Blocked is true.
func RunHttpHook(ctx context.Context, workDir string, h hookstypes.HookCommand, jsonInput string, allowedURLs []string) OutsideReplHookResult {
	res := OutsideReplHookResult{
		HookType: "http",
		Command:  "[http] " + h.URL,
	}

	// URL allowlist check (mirrors TS allowedHttpHookUrls).
	// undefined = no restriction; [] = block all; non-empty = must match.
	if err := checkURLAllowlist(h.URL, allowedURLs); err != nil {
		res.ErrorMessage = err.Error()
		return res
	}

	timeoutMs := parseHookTimeoutMs(h)
	cctx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	start := time.Now()

	// Build headers with env var interpolation and sanitization.
	headers := buildHookHeaders(h.Headers, h.AllowedEnvVars)

	// SSRF-guarded HTTP client.
	client := ssrfGuardedHTTPClient()

	req, err := http.NewRequestWithContext(cctx, http.MethodPost, h.URL, bytes.NewReader([]byte(jsonInput)))
	if err != nil {
		res.ErrorMessage = fmt.Sprintf("http hook: create request: %v", err)
		res.DurationMs = time.Since(start).Milliseconds()
		return res
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		if cctx.Err() != nil {
			res.Cancelled = true
			res.ErrorMessage = "http hook: cancelled or timed out"
		} else {
			res.ErrorMessage = fmt.Sprintf("http hook: %v", err)
		}
		res.DurationMs = time.Since(start).Milliseconds()
		return res
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024))
	if err != nil {
		res.ErrorMessage = fmt.Sprintf("http hook: read response: %v", err)
		res.DurationMs = time.Since(start).Milliseconds()
		return res
	}

	res.Stdout = string(body)
	res.Output = string(body)
	res.HTTPStatusCode = resp.StatusCode
	res.DurationMs = time.Since(start).Milliseconds()

	// 2xx = success; others = non-blocking error.
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		res.Succeeded = true
		// Check for JSON "decision": "block" in response.
		if blocked, _ := parseHookBlocked(string(body)); blocked {
			res.Blocked = true
			res.Succeeded = false
		}
	} else {
		res.ErrorMessage = fmt.Sprintf("http hook: HTTP %d: %s", resp.StatusCode, truncateString(string(body), 200))
	}

	return res
}

// checkURLAllowlist validates an HTTP hook URL against the configured allowlist.
// Mirrors TS allowedHttpHookUrls check in execHttpHook.ts.
func checkURLAllowlist(targetURL string, allowed []string) error {
	if allowed == nil {
		// undefined = no restriction
		return nil
	}
	if len(allowed) == 0 {
		return fmt.Errorf("http hook: no allowed URLs configured (allowlist is empty)")
	}
	for _, pattern := range allowed {
		if urlMatchesPattern(targetURL, pattern) {
			return nil
		}
	}
	return fmt.Errorf("http hook: URL %q not in allowlist", targetURL)
}

// urlMatchesPattern checks if a URL matches a glob-style pattern (* wildcard).
func urlMatchesPattern(targetURL, pattern string) bool {
	// Convert glob pattern to regex: escape regex metachars, replace * with .*
	escaped := regexp.QuoteMeta(pattern)
	escaped = strings.ReplaceAll(escaped, `\*`, ".*")
	re, err := regexp.Compile("^" + escaped + "$")
	if err != nil {
		return false
	}
	return re.MatchString(targetURL)
}

// buildHookHeaders builds HTTP headers with env var interpolation and sanitization.
// Mirrors TS execHttpHook header processing (interpolateEnvVars + sanitizeHeaderValue).
func buildHookHeaders(headers map[string]string, allowedEnvVars []string) map[string]string {
	if len(headers) == 0 {
		return nil
	}
	result := make(map[string]string, len(headers))
	for k, v := range headers {
		interpolated := interpolateEnvVars(v, allowedEnvVars)
		result[k] = sanitizeHeaderValue(interpolated)
	}
	return result
}

var envVarPattern = regexp.MustCompile(`\$\{?([A-Za-z_][A-Za-z0-9_]*)\}?`)

// interpolateEnvVars replaces $VAR and ${VAR} in a string with env values.
// Only variables in the allowlist are interpolated; others become empty strings.
func interpolateEnvVars(s string, allowedEnvVars []string) string {
	allowed := make(map[string]bool, len(allowedEnvVars))
	for _, v := range allowedEnvVars {
		allowed[v] = true
	}
	return envVarPattern.ReplaceAllStringFunc(s, func(match string) string {
		// Extract the variable name from $VAR or ${VAR}.
		name := match
		name = strings.TrimPrefix(name, "$")
		name = strings.TrimPrefix(name, "{")
		name = strings.TrimSuffix(name, "}")
		if allowed[name] {
			return osGetenv(name)
		}
		return ""
	})
}

// sanitizeHeaderValue strips CR, LF, and NUL bytes to prevent HTTP header injection.
// Mirrors TS sanitizeHeaderValue in execHttpHook.ts.
func sanitizeHeaderValue(v string) string {
	v = strings.ReplaceAll(v, "\r", "")
	v = strings.ReplaceAll(v, "\n", "")
	v = strings.ReplaceAll(v, "\x00", "")
	return v
}

// ssrfGuardedHTTPClient returns an *http.Client with SSRF protection.
// The custom DialContext resolves hostnames and blocks private/link-local IPs,
// except loopback addresses which are allowed (for local dev).
func ssrfGuardedHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 0, // timeout is handled per-request via context
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				host, port, err := net.SplitHostPort(addr)
				if err != nil {
					host = addr
					port = "80"
				}
				ips, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
				if err != nil {
					return nil, err
				}
				for _, ip := range ips {
					if err := checkSSRF(ip); err != nil {
						return nil, err
					}
				}
				d := net.Dialer{}
				return d.DialContext(ctx, network, net.JoinHostPort(ips[0].String(), port))
			},
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // maxRedirects: 0
		},
	}
}

// checkSSRF validates that an IP address is not in a private/link-local range.
// Loopback addresses (127.0.0.0/8, ::1) are allowed for local dev.
// Mirrors TS ssrfGuardedLookup in ssrfGuard.ts.
func checkSSRF(ip net.IP) error {
	// Allow loopback.
	if ip.IsLoopback() {
		return nil
	}
	// Block private, link-local, unspecified, multicast.
	if ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsUnspecified() || ip.IsMulticast() {
		return fmt.Errorf("http hook: SSRF guard blocked IP %s", ip.String())
	}
	return nil
}

// osGetenv is a package-level variable for testing; defaults to os.Getenv.
var osGetenv = os.Getenv

// checkSSRFURL validates a URL before making the request (pre-flight check).
// Returns an error if the URL has a blocked host.
func checkSSRFURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("http hook: invalid URL: %v", err)
	}
	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("http hook: empty host in URL")
	}
	ips, err := net.DefaultResolver.LookupIP(context.Background(), "ip", host)
	if err != nil {
		return fmt.Errorf("http hook: DNS lookup failed for %q: %v", host, err)
	}
	for _, ip := range ips {
		if err := checkSSRF(ip); err != nil {
			return err
		}
	}
	return nil
}
