package claudeinit

import (
	"context"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// ParsedRepository matches TS ParsedRepository (host may include :port for https).
type ParsedRepository struct {
	Host  string
	Owner string
	Name  string
}

var (
	sshRemoteRe = regexp.MustCompile(`^git@([^:]+):([^/]+)/([^/]+?)(?:\.git)?$`)
	urlRemoteRe = regexp.MustCompile(`^(https?|ssh|git)://(?:[^@]+@)?([^/:]+(?::\d+)?)/([^/]+)/([^/]+?)(?:\.git)?$`)
)

// ParseGitRemote ports TS [parseGitRemote] (github.com + GHE URLs).
func ParseGitRemote(input string) *ParsedRepository {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return nil
	}
	if m := sshRemoteRe.FindStringSubmatch(trimmed); len(m) == 4 {
		host := m[1]
		if !looksLikeRealHostname(host) {
			return nil
		}
		return &ParsedRepository{Host: host, Owner: m[2], Name: m[3]}
	}
	if m := urlRemoteRe.FindStringSubmatch(trimmed); len(m) == 5 {
		protocol := m[1]
		hostWithPort := m[2]
		hostWithoutPort := hostWithPort
		if i := strings.IndexByte(hostWithPort, ':'); i >= 0 {
			hostWithoutPort = hostWithPort[:i]
		}
		if !looksLikeRealHostname(hostWithoutPort) {
			return nil
		}
		host := hostWithoutPort
		if protocol == "https" || protocol == "http" {
			host = hostWithPort
		}
		return &ParsedRepository{Host: host, Owner: m[3], Name: m[4]}
	}
	return nil
}

func looksLikeRealHostname(host string) bool {
	if !strings.Contains(host, ".") {
		return false
	}
	last := host[strings.LastIndex(host, ".")+1:]
	if last == "" {
		return false
	}
	for _, r := range last {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') {
			return false
		}
	}
	return true
}

func gitRemoteOriginURL() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "config", "--get", "remote.origin.url")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// GitHubOwnerSlashName returns "owner/name" for github.com only (TS detectCurrentRepository).
func GitHubOwnerSlashName(parsed *ParsedRepository) string {
	if parsed == nil {
		return ""
	}
	host := parsed.Host
	if i := strings.IndexByte(host, ':'); i >= 0 {
		host = host[:i]
	}
	if host != "github.com" {
		return ""
	}
	return parsed.Owner + "/" + parsed.Name
}
