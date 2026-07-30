package syncer

import (
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"

	"github.com/Guepardo/commit-sync/internal/scanner"
)

func collectFromRepo(r scanner.ScanResult, dedup map[string]bool) []pendingCommit {
	repo, err := git.PlainOpen(r.Path)
	if err != nil {
		return nil
	}

	head, err := repo.Head()
	if err != nil {
		return nil
	}

	iter, err := repo.Log(&git.LogOptions{From: head.Hash()})
	if err != nil {
		return nil
	}
	defer iter.Close()

	var pending []pendingCommit
	iter.ForEach(func(c *object.Commit) error {
		if dedup[key(r.Path, c.Hash.String())] {
			return nil
		}
		if c.NumParents() > 1 {
			return nil
		}
		pending = append(pending, pendingCommit{
			path:      r.Path,
			hash:      c.Hash,
			author:    c.Author,
			committer: c.Committer,
			tree:      c.TreeHash,
			msg:       c.Message,
			when:      c.Author.When,
		})
		return nil
	})

	return pending
}

func collectPending(repos []scanner.ScanResult, dedup map[string]bool) []pendingCommit {
	var all []pendingCommit
	for _, r := range repos {
		all = append(all, collectFromRepo(r, dedup)...)
	}
	return all
}
