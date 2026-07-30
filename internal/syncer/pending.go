package syncer

import (
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"

	"github.com/Guepardo/commit-sync/internal/scanner"
)

func collectFromRepo(r scanner.ScanResult, dedup map[string]bool, seen map[string]bool) ([]pendingCommit, map[string]plumbing.Hash) {
	repo, err := git.PlainOpen(r.Path)
	if err != nil {
		return nil, nil
	}

	branches := r.Branches
	if len(branches) == 0 {
		head, err := repo.Head()
		if err != nil {
			return nil, nil
		}
		branches = []string{head.Name().String()}
	}

	sourceTips := make(map[string]plumbing.Hash)

	var pending []pendingCommit
	for _, branch := range branches {
		ref, err := repo.Reference(plumbing.ReferenceName(branch), true)
		if err != nil {
			continue
		}
		sourceTips[r.Path+":"+branch] = ref.Hash()

		iter, err := repo.Log(&git.LogOptions{From: ref.Hash()})
		if err != nil {
			continue
		}
		iter.ForEach(func(c *object.Commit) error {
			k := key(r.Path, c.Hash.String())
			if dedup[k] || seen[k] {
				return nil
			}
			if c.NumParents() > 1 {
				return nil
			}
			seen[k] = true
			pending = append(pending, pendingCommit{
				path:         r.Path,
				sourceBranch: branch,
				hash:         c.Hash,
				author:       c.Author,
				committer:    c.Committer,
				tree:         c.TreeHash,
				msg:          c.Message,
				when:         c.Author.When,
				srcStorer:    repo.Storer,
			})
			return nil
		})
		iter.Close()
	}

	return pending, sourceTips
}

func collectPending(repos []scanner.ScanResult, dedup map[string]bool) ([]pendingCommit, map[string]plumbing.Hash) {
	var all []pendingCommit
	allTips := make(map[string]plumbing.Hash)
	seen := make(map[string]bool)
	for _, r := range repos {
		pending, tips := collectFromRepo(r, dedup, seen)
		all = append(all, pending...)
		for k, v := range tips {
			allTips[k] = v
		}
	}
	return all, allTips
}
