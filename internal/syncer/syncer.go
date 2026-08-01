package syncer

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"

	"github.com/Guepardo/commit-sync/internal/scanner"
)

const mirrorBranch = "refs/heads/main"

var mirroredFromRe = regexp.MustCompile(`^Mirrored-From: (.+) ([a-f0-9]+)$`)

type pendingCommit struct {
	path      string
	hash      plumbing.Hash
	author    object.Signature
	committer object.Signature
	msg       string
	when      time.Time
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

func key(path, hash string) string {
	return path + ":" + hash
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
	mirror, err := OpenMirror(s.mirrorPath)
	if err != nil {
		return 0, fmt.Errorf("mirror: %w", err)
	}

	dedup, existingSrcToMirror, err := mirror.BuildDedupMap()
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

	return mirror.Apply(pending, sourceTips, existingSrcToMirror)
}

func buildBranchTips(sourceTips, existingSrcToMirror map[string]plumbing.Hash) map[string]plumbing.Hash {
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
	return branchTips
}
