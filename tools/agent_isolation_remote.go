package tools

import "fmt"

type RemoteIsolationResult struct {
	Accepted bool
	Message  string
}

func startRemoteIsolation() RemoteIsolationResult {
	return RemoteIsolationResult{
		Accepted: false,
		Message:  "remote isolation requested but no remote execution backend is configured in this Go runtime",
	}
}

func requireRemoteBackend() error {
	r := startRemoteIsolation()
	if r.Accepted {
		return nil
	}
	return fmt.Errorf("%s", r.Message)
}
