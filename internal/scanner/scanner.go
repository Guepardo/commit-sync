package scanner

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

type ScanResult struct {
	Path          string
	DefaultBranch string
	CommitCount   int
}

func Scan(root string, exclude string) ([]ScanResult, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve root: %w", err)
	}

	exclude, err = filepath.Abs(exclude)
	if err != nil {
		exclude = ""
	}

	var results []ScanResult
	seen := make(map[string]bool)

	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if !d.IsDir() {
			return nil
		}

		if path == exclude {
			return filepath.SkipDir
		}

		if d.Name() == ".git" {
			repoDir := filepath.Dir(path)

			if seen[repoDir] {
				return filepath.SkipDir
			}
			seen[repoDir] = true

			r, err := git.PlainOpen(repoDir)
			if err != nil {
				return filepath.SkipDir
			}

			remotes, err := r.Remotes()
			if err != nil || len(remotes) > 0 {
				return filepath.SkipDir
			}

			branch := ""
			commitCount := 0

			head, err := r.Head()
			if err == nil {
				branch = head.Name().String()
				cIter, err := r.Log(&git.LogOptions{From: head.Hash()})
				if err == nil {
					cIter.ForEach(func(c *object.Commit) error {
						commitCount++
						return nil
					})
				}
			}

			results = append(results, ScanResult{
				Path:          repoDir,
				DefaultBranch: branch,
				CommitCount:   commitCount,
			})

			return filepath.SkipDir
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("walk: %w", err)
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Path < results[j].Path
	})

	return results, nil
}
