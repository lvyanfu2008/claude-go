// Package suggestions provides file indexing and @-mention autocomplete suggestions for the TUI.
package suggestions

import (
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// FileIndex maintains an in-memory list of project files with background refresh.
type FileIndex struct {
	entries         []string
	mu              sync.RWMutex
	cwd             string
	lastRefresh     time.Time
	refreshThrottle time.Duration
	stopCh          chan struct{}
}

// NewFileIndex builds the initial file index from the given working directory.
func NewFileIndex(cwd string) *FileIndex {
	fi := &FileIndex{
		cwd:             cwd,
		refreshThrottle: 5 * time.Second,
		stopCh:          make(chan struct{}),
	}
	fi.refresh()
	go fi.backgroundRefresh()
	return fi
}

// Stop shuts down the background refresh goroutine.
func (fi *FileIndex) Stop() {
	close(fi.stopCh)
}

// Search returns scored file/directory entries matching query. Returns top 15.
func (fi *FileIndex) Search(query string) []ScoredItem {
	fi.mu.RLock()
	defer fi.mu.RUnlock()
	return scoredSearch(fi.entries, query, 15)
}

// GetTopLevelPaths lists directory entries under the given directory path.
func (fi *FileIndex) GetTopLevelPaths(dirPath string) []ScoredItem {
	absPath := fi.resolvePath(dirPath)
	entries, err := os.ReadDir(absPath)
	if err != nil {
		return nil
	}
	var items []ScoredItem
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		rel, err := filepath.Rel(fi.cwd, filepath.Join(absPath, name))
		if err != nil {
			continue
		}
		icon := "F"
		sType := SuggestionTypeFile
		if e.IsDir() {
			icon = "D"
			sType = SuggestionTypeDirectory
			rel += "/"
		}
		items = append(items, ScoredItem{
			Type:  sType,
			Label: filepath.ToSlash(rel),
			Value: filepath.ToSlash(rel),
			Score: 0,
			Icon:  icon,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Type != items[j].Type {
			return items[i].Type == SuggestionTypeDirectory
		}
		return items[i].Label < items[j].Label
	})
	if len(items) > 15 {
		items = items[:15]
	}
	return items
}

func (fi *FileIndex) resolvePath(rel string) string {
	rel = filepath.ToSlash(rel)
	if strings.HasPrefix(rel, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, rel[2:])
	}
	if filepath.IsAbs(rel) {
		return filepath.Clean(rel)
	}
	return filepath.Join(fi.cwd, rel)
}

func (fi *FileIndex) refresh() {
	fi.mu.Lock()
	defer fi.mu.Unlock()
	fi.entries = fi.collectFiles()
	fi.lastRefresh = time.Now()
}

func (fi *FileIndex) collectFiles() []string {
	entries := fi.tryGitLsFiles()
	if entries == nil {
		entries = fi.walkFilesystem()
	}
	return entries
}

func (fi *FileIndex) tryGitLsFiles() []string {
	cmd := exec.Command("git", "ls-files", "--cached", "--others", "--exclude-standard")
	cmd.Dir = fi.cwd
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var entries []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		entries = append(entries, filepath.ToSlash(line))
	}
	if len(entries) == 0 {
		return nil
	}
	return entries
}

func (fi *FileIndex) walkFilesystem() []string {
	var entries []string
	filepath.Walk(fi.cwd, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			base := filepath.Base(path)
			if strings.HasPrefix(base, ".") && base != "." {
				return filepath.SkipDir
			}
			if base == "node_modules" || base == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(fi.cwd, path)
		if err != nil {
			return nil
		}
		entries = append(entries, filepath.ToSlash(rel))
		return nil
	})
	return entries
}

func (fi *FileIndex) backgroundRefresh() {
	ticker := time.NewTicker(fi.refreshThrottle)
	defer ticker.Stop()
	for {
		select {
		case <-fi.stopCh:
			return
		case <-ticker.C:
			fi.refresh()
		}
	}
}

// scoredSearch ranks entries against query using a simple tiered scoring system.
func scoredSearch(entries []string, query string, limit int) []ScoredItem {
	q := strings.ToLower(query)
	type rank struct {
		idx   int
		tier  int
		score float64
	}
	var cand []rank
	for i, entry := range entries {
		lower := strings.ToLower(entry)
		base := filepath.Base(entry)
		lowerBase := strings.ToLower(base)
		var tier int
		var score float64
		switch {
		case q == "":
			tier = 10
		case lower == q:
			tier = 0
		case strings.HasPrefix(lower, q):
			tier = 1
			score = float64(len(lower) - len(q))
		case strings.HasPrefix(lowerBase, q):
			tier = 2
			score = float64(len(lowerBase) - len(q))
		case strings.Contains(lower, q):
			tier = 3
			score = float64(strings.Index(lower, q))
		case strings.Contains(lowerBase, q):
			tier = 4
			score = float64(strings.Index(lowerBase, q))
		case subsequenceMatch(q, lower):
			tier = 5
			score = 2
		default:
			continue
		}
		cand = append(cand, rank{idx: i, tier: tier, score: score})
	}
	sort.SliceStable(cand, func(i, j int) bool {
		if cand[i].tier != cand[j].tier {
			return cand[i].tier < cand[j].tier
		}
		if cand[i].score != cand[j].score {
			return cand[i].score < cand[j].score
		}
		return strings.ToLower(entries[cand[i].idx]) < strings.ToLower(entries[cand[j].idx])
	})
	var out []ScoredItem
	seen := map[string]bool{}
	for _, c := range cand {
		entry := entries[c.idx]
		if seen[entry] {
			continue
		}
		seen[entry] = true
		icon := "F"
		out = append(out, ScoredItem{
			Type:  SuggestionTypeFile,
			Label: entry,
			Value: entry,
			Score: float64(c.tier)*100 + c.score,
			Icon:  icon,
		})
		if len(out) >= limit {
			break
		}
	}
	return out
}

func subsequenceMatch(needle, haystack string) bool {
	if needle == "" {
		return true
	}
	nr := []rune(needle)
	hr := []rune(haystack)
	j := 0
	for i := 0; i < len(hr) && j < len(nr); i++ {
		if hr[i] == nr[j] {
			j++
		}
	}
	return j == len(nr)
}
