package claudeinit

// phase3Network: TS configureGlobalMTLS, configureGlobalAgents, initSentry, preconnectAnthropicApi, upstreamproxy.
func phase3Network() error {
	// P3e HEAD warm-up (subset of TS preconnectAnthropicApi; OAuth staging URL is explicit_gap).
	firePreconnectAnthropicAPI()
	return nil
}
