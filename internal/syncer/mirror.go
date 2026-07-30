package syncer

import (
	"fmt"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage"
)

var cachedEmptyTree plumbing.Hash

func ensureEmptyTree(storer storage.Storer) (plumbing.Hash, error) {
	if cachedEmptyTree != plumbing.ZeroHash {
		_, err := storer.EncodedObject(plumbing.TreeObject, cachedEmptyTree)
		if err == nil {
			return cachedEmptyTree, nil
		}
	}
	tree := &object.Tree{}
	obj := storer.NewEncodedObject()
	if err := tree.Encode(obj); err != nil {
		return plumbing.ZeroHash, fmt.Errorf("encode empty tree: %w", err)
	}
	hash, err := storer.SetEncodedObject(obj)
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("store empty tree: %w", err)
	}
	cachedEmptyTree = hash
	return hash, nil
}

func writeSingleCommit(storer storage.Storer, p pendingCommit, lastHash plumbing.Hash) (plumbing.Hash, error) {
	treeHash, err := ensureEmptyTree(storer)
	if err != nil {
		return plumbing.ZeroHash, err
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
		TreeHash:     treeHash,
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

func writeCommits(storer storage.Storer, pending []pendingCommit, lastHash plumbing.Hash) (plumbing.Hash, map[string]plumbing.Hash, error) {
	srcToMirror := make(map[string]plumbing.Hash)

	for _, p := range pending {
		newHash, err := writeSingleCommit(storer, p, lastHash)
		if err != nil {
			return plumbing.ZeroHash, nil, err
		}
		lastHash = newHash
		srcToMirror[key(p.path, p.hash.String())] = newHash
	}

	return lastHash, srcToMirror, nil
}

func updateMirrorRefs(storer storage.Storer, lastHash plumbing.Hash, branchTips map[string]plumbing.Hash) error {
	ref := plumbing.NewHashReference(plumbing.ReferenceName(mirrorBranch), lastHash)
	if err := storer.SetReference(ref); err != nil {
		return fmt.Errorf("set ref: %w", err)
	}

	headRef := plumbing.NewSymbolicReference(plumbing.HEAD, plumbing.ReferenceName(mirrorBranch))
	if err := storer.SetReference(headRef); err != nil {
		return fmt.Errorf("set HEAD: %w", err)
	}

	for mirrorRef, hash := range branchTips {
		br := plumbing.NewHashReference(plumbing.ReferenceName(mirrorRef), hash)
		if err := storer.SetReference(br); err != nil {
			return fmt.Errorf("set %s: %w", mirrorRef, err)
		}
	}

	return nil
}
