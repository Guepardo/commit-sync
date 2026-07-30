package syncer

import (
	"fmt"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

type MirrorRepo struct {
	repo *git.Repository
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

func OpenMirror(path string) (*MirrorRepo, error) {
	repo, err := openOrInitMirror(path)
	if err != nil {
		return nil, err
	}
	return &MirrorRepo{repo: repo}, nil
}

func (m *MirrorRepo) HeadHash() plumbing.Hash {
	head, err := m.repo.Head()
	if err != nil {
		return plumbing.ZeroHash
	}
	return head.Hash()
}

func (m *MirrorRepo) BuildDedupMap() (dedup map[string]bool, srcToMirror map[string]plumbing.Hash, err error) {
	dedup = make(map[string]bool)
	srcToMirror = make(map[string]plumbing.Hash)

	head, err2 := m.repo.Head()
	if err2 != nil {
		return dedup, srcToMirror, nil
	}

	iter, err2 := m.repo.Log(&git.LogOptions{From: head.Hash()})
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

func (m *MirrorRepo) WriteCommits(pending []pendingCommit, lastHash plumbing.Hash) (plumbing.Hash, map[string]plumbing.Hash, error) {
	srcToMirror := make(map[string]plumbing.Hash)

	for _, p := range pending {
		newHash, err := m.writeSingleCommit(p, lastHash)
		if err != nil {
			return plumbing.ZeroHash, nil, err
		}
		lastHash = newHash
		srcToMirror[key(p.path, p.hash.String())] = newHash
	}

	return lastHash, srcToMirror, nil
}

func (m *MirrorRepo) writeSingleCommit(p pendingCommit, lastHash plumbing.Hash) (plumbing.Hash, error) {
	treeHash, err := ensureEmptyTree(m.repo.Storer)
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

	obj := m.repo.Storer.NewEncodedObject()
	if err := commit.Encode(obj); err != nil {
		return plumbing.ZeroHash, fmt.Errorf("encode commit: %w", err)
	}
	newHash, err := m.repo.Storer.SetEncodedObject(obj)
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("store commit: %w", err)
	}

	return newHash, nil
}

func (m *MirrorRepo) UpdateRefs(lastHash plumbing.Hash, branchTips map[string]plumbing.Hash) error {
	storer := m.repo.Storer
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


