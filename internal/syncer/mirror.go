package syncer

import (
	"fmt"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage"
)

func writeSingleCommit(storer storage.Storer, p pendingCommit, lastHash plumbing.Hash) (plumbing.Hash, error) {
	msg := appendMirroredFrom(p.msg, p.path, p.hash.String())

	var parentHashes []plumbing.Hash
	if lastHash != plumbing.ZeroHash {
		parentHashes = []plumbing.Hash{lastHash}
	}

	commit := &object.Commit{
		Author:       p.author,
		Committer:    p.committer,
		Message:      msg,
		TreeHash:     p.tree,
		ParentHashes: parentHashes,
	}

	obj := storer.NewEncodedObject()
	if err := commit.Encode(obj); err != nil {
		return plumbing.ZeroHash, fmt.Errorf("encode commit: %w", err)
	}
	newHash, err := storer.SetEncodedObject(obj)
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("store commit: %w", err)
	}

	return newHash, nil
}

func writeCommits(storer storage.Storer, pending []pendingCommit, lastHash plumbing.Hash) (plumbing.Hash, error) {
	for _, p := range pending {
		newHash, err := writeSingleCommit(storer, p, lastHash)
		if err != nil {
			return plumbing.ZeroHash, err
		}
		lastHash = newHash
	}

	return lastHash, nil
}

func updateMirrorRefs(storer storage.Storer, lastHash plumbing.Hash) error {
	ref := plumbing.NewHashReference(plumbing.ReferenceName(mirrorBranch), lastHash)
	if err := storer.SetReference(ref); err != nil {
		return fmt.Errorf("set ref: %w", err)
	}

	headRef := plumbing.NewSymbolicReference(plumbing.HEAD, plumbing.ReferenceName(mirrorBranch))
	if err := storer.SetReference(headRef); err != nil {
		return fmt.Errorf("set HEAD: %w", err)
	}

	return nil
}
