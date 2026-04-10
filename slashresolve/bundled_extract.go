package slashresolve

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Memoize extracted reference dirs per process (matches TS bundledSkills closure memoization).
var bundledExtractMemo sync.Map // skillName -> extracted dir string

type bundledFileEntry struct {
	relPath string
	name    string // path inside embed FS
}

var verifyBundledFiles = []bundledFileEntry{
	{"examples/cli.md", "verify/examples/cli.md"},
	{"examples/server.md", "verify/examples/server.md"},
}

func materializeVerifySkillDir() (string, error) {
	if v, ok := bundledExtractMemo.Load("verify"); ok {
		return v.(string), nil
	}
	base, err := os.MkdirTemp("", "claude-go-bundled-verify-*")
	if err != nil {
		return "", err
	}
	for _, fe := range verifyBundledFiles {
		b, err := fs.ReadFile(bundledData, filepath.Join("bundleddata", fe.name))
		if err != nil {
			os.RemoveAll(base)
			return "", fmt.Errorf("bundled verify %s: %w", fe.name, err)
		}
		target := filepath.Join(base, filepath.FromSlash(fe.relPath))
		if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
			os.RemoveAll(base)
			return "", err
		}
		if err := os.WriteFile(target, b, 0o600); err != nil {
			os.RemoveAll(base)
			return "", err
		}
	}
	bundledExtractMemo.Store("verify", base)
	return base, nil
}

func readVerifySkillBody() (string, error) {
	b, err := fs.ReadFile(bundledData, filepath.Join("bundleddata", "verify", "SKILL.md"))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}
