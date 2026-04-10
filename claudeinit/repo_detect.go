package claudeinit

import "sync"

var (
	repoMu           sync.Mutex
	repoResolved     bool
	cachedGitHubRepo string
)

func startDetectRepositoryBackground() {
	go resolveGitHubRepo()
}

func resolveGitHubRepo() {
	repoMu.Lock()
	defer repoMu.Unlock()
	if repoResolved {
		return
	}
	repoResolved = true

	u, err := gitRemoteOriginURL()
	if err != nil || u == "" {
		cachedGitHubRepo = ""
		return
	}
	p := ParseGitRemote(u)
	cachedGitHubRepo = GitHubOwnerSlashName(p)
}

func resetRepoDetectForTesting() {
	repoMu.Lock()
	defer repoMu.Unlock()
	repoResolved = false
	cachedGitHubRepo = ""
}

// CachedGitHubRepo returns "owner/name" for github.com after [resolveGitHubRepo] ran.
func CachedGitHubRepo() string {
	resolveGitHubRepo()
	repoMu.Lock()
	defer repoMu.Unlock()
	return cachedGitHubRepo
}
