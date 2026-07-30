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


