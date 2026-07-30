package scanner

import (
	"fmt"
	"path/filepath"
	"sort"
)

type ScanResult struct {
	Path          string
	DefaultBranch string
	Branches      []string
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

	seen := make(map[string]bool)
	results, err := walkRepos(root, exclude, seen)
	if err != nil {
		return nil, fmt.Errorf("walk: %w", err)
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Path < results[j].Path
	})

	return results, nil
}
