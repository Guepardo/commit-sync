package syncer

import (
	"fmt"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage"
)

func copyObject(srcStorer, dstStorer storage.Storer, hash plumbing.Hash, seen map[plumbing.Hash]bool) error {
	if seen[hash] {
		return nil
	}
	seen[hash] = true

	_, err := dstStorer.EncodedObject(plumbing.AnyObject, hash)
	if err == nil {
		return nil
	}
	if err != plumbing.ErrObjectNotFound {
		return err
	}

	obj, err := srcStorer.EncodedObject(plumbing.AnyObject, hash)
	if err != nil {
		return fmt.Errorf("read %s from source: %w", hash, err)
	}

	if obj.Type() == plumbing.TreeObject {
		tree := &object.Tree{}
		if err := tree.Decode(obj); err != nil {
			return fmt.Errorf("decode tree %s: %w", hash, err)
		}
		for _, entry := range tree.Entries {
			if err := copyObject(srcStorer, dstStorer, entry.Hash, seen); err != nil {
				return err
			}
		}
	}

	if _, err := dstStorer.SetEncodedObject(obj); err != nil {
		return fmt.Errorf("store %s: %w", hash, err)
	}

	return nil
}

func writeSingleCommit(storer storage.Storer, p pendingCommit, lastHash plumbing.Hash, seen map[plumbing.Hash]bool) (plumbing.Hash, error) {
	if err := copyObject(p.srcStorer, storer, p.tree, seen); err != nil {
		return plumbing.ZeroHash, fmt.Errorf("copy objects for %s: %w", p.hash, err)
	}

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
	seen := make(map[plumbing.Hash]bool)
	for _, p := range pending {
		newHash, err := writeSingleCommit(storer, p, lastHash, seen)
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
