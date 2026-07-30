package syncer

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage"

	"github.com/Guepardo/commit-sync/internal/scanner"
)

const mirrorBranch = "refs/heads/main"

var mirroredFromRe = regexp.MustCompile(`^Mirrored-From: (.+) ([a-f0-9]+)$`)

type pendingCommit struct {
	path         string
	sourceBranch string
	hash         plumbing.Hash
	author       object.Signature
	committer    object.Signature
	tree         plumbing.Hash
	msg          string
	when         time.Time
	srcStorer    storage.Storer
}

func parseMirroredFrom(msg string) (path, hash string, ok bool) {
	msg = strings.TrimRight(msg, "\n")
	lines := strings.Split(msg, "\n")
	if len(lines) == 0 {
		return "", "", false
	}
	lastLine := strings.TrimSpace(lines[len(lines)-1])
	matches := mirroredFromRe.FindStringSubmatch(lastLine)
	if matches == nil {
		return "", "", false
	}
	return matches[1], matches[2], true
}

func appendMirroredFrom(msg, path, hash string) string {
	return msg + "\n\nMirrored-From: " + path + " " + hash + "\n"
}

func buildDedupMap(repo *git.Repository) (dedup map[string]bool, srcToMirror map[string]plumbing.Hash, err error) {
	dedup = make(map[string]bool)
	srcToMirror = make(map[string]plumbing.Hash)

	head, err2 := repo.Head()
	if err2 != nil {
		return dedup, srcToMirror, nil
	}

	iter, err2 := repo.Log(&git.LogOptions{From: head.Hash()})
	if err2 != nil {
		return nil, nil, fmt.Errorf("log mirror: %w", err2)
	}
	defer iter.Close()

	err2 = iter.ForEach(func(c *object.Commit) error {
		if p, h, ok := parseMirroredFrom(c.Message); ok {
			k := key(p, h)
			dedup[k] = true
			srcToMirror[k] = c.Hash
		}
		return nil
	})
	return dedup, srcToMirror, err2
}

func key(path, hash string) string {
	return path + ":" + hash
}

func initMirror(path string) (*git.Repository, error) {
	repo, err := git.PlainInit(path, false)
	if err != nil {
		return nil, fmt.Errorf("init mirror: %w", err)
	}
	return repo, nil
}

func openOrInitMirror(path string) (*git.Repository, error) {
	repo, err := git.PlainOpen(path)
	if err == nil {
		return repo, nil
	}

	if err == git.ErrRepositoryNotExists {
		return initMirror(path)
	}
	return nil, fmt.Errorf("open mirror: %w", err)
}

type Syncer struct {
	mirrorPath string
	dryRun     bool
}

func New(mirrorPath string) *Syncer {
	return &Syncer{mirrorPath: mirrorPath}
}

func (s *Syncer) SetDryRun(dry bool) {
	s.dryRun = dry
}

func (s *Syncer) Sync(repos []scanner.ScanResult) (int, error) {
	mirrorRepo, err := openOrInitMirror(s.mirrorPath)
	if err != nil {
		return 0, fmt.Errorf("mirror: %w", err)
	}

	dedup, existingSrcToMirror, err := buildDedupMap(mirrorRepo)
	if err != nil {
		return 0, fmt.Errorf("dedup: %w", err)
	}

	pending, sourceTips := collectPending(repos, dedup)
	if len(pending) == 0 && len(sourceTips) == 0 {
		return 0, nil
	}

	sort.SliceStable(pending, func(i, j int) bool {
		if !pending[i].when.Equal(pending[j].when) {
			return pending[i].when.Before(pending[j].when)
		}
		return pending[i].hash.String() < pending[j].hash.String()
	})

	if s.dryRun {
		return len(pending), nil
	}

	var lastHash plumbing.Hash
	head, err := mirrorRepo.Head()
	if err == nil {
		lastHash = head.Hash()
	}

	newSrcToMirror := make(map[string]plumbing.Hash)
	if len(pending) > 0 {
		lastHash, newSrcToMirror, err = writeCommits(mirrorRepo.Storer, pending, lastHash)
		if err != nil {
			return 0, err
		}
	}

	for k, v := range newSrcToMirror {
		existingSrcToMirror[k] = v
	}

	branchTips := make(map[string]plumbing.Hash)
	for skey, srcTip := range sourceTips {
		parts := strings.SplitN(skey, ":", 2)
		if len(parts) != 2 {
			continue
		}
		path, branch := parts[0], parts[1]
		tipKey := key(path, srcTip.String())
		mirrorHash, ok := existingSrcToMirror[tipKey]
		if !ok {
			continue
		}
		basename := filepath.Base(path)
		shortBranch := strings.TrimPrefix(branch, "refs/heads/")
		mirrorRef := fmt.Sprintf("refs/heads/source/%s/%s", basename, shortBranch)
		branchTips[mirrorRef] = mirrorHash
	}

	if lastHash != plumbing.ZeroHash {
		if err := updateMirrorRefs(mirrorRepo.Storer, lastHash, branchTips); err != nil {
			return 0, err
		}
	}

	return len(pending), nil
}
