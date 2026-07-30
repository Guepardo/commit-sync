package scanner

import (
	"io/fs"
	"path/filepath"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func processGitDir(repoDir string, seen map[string]bool) (*ScanResult, error) {
	if seen[repoDir] {
		return nil, nil
	}
	seen[repoDir] = true

	r, err := git.PlainOpen(repoDir)
	if err != nil {
		return nil, nil
	}

	remotes, err := r.Remotes()
	if err != nil || len(remotes) > 0 {
		return nil, nil
	}

	var branches []string
	branchIter, err := r.Branches()
	if err == nil {
		branchIter.ForEach(func(ref *plumbing.Reference) error {
			branches = append(branches, ref.Name().String())
			return nil
		})
	}

	defaultBranch := ""
	commitCount := 0

	head, err := r.Head()
	if err == nil {
		defaultBranch = head.Name().String()
		cIter, err := r.Log(&git.LogOptions{From: head.Hash()})
		if err == nil {
			cIter.ForEach(func(c *object.Commit) error {
				commitCount++
				return nil
			})
		}
	}

	return &ScanResult{
		Path:          repoDir,
		DefaultBranch: defaultBranch,
		Branches:      branches,
		CommitCount:   commitCount,
	}, nil
}

func walkRepos(root, exclude string, seen map[string]bool) ([]ScanResult, error) {
	var results []ScanResult

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() {
			return nil
		}
		if path == exclude {
			return filepath.SkipDir
		}
		if d.Name() != ".git" {
			return nil
		}

		result, err := processGitDir(filepath.Dir(path), seen)
		if err != nil {
			return nil
		}
		if result != nil {
			results = append(results, *result)
		}
		return filepath.SkipDir
	})

	return results, err
}
