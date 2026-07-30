package scanner

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func initRepo(t *testing.T, path string) *git.Repository {
	t.Helper()
	repo, err := git.PlainInit(path, false)
	if err != nil {
		t.Fatal(err)
	}
	return repo
}

func addRemote(t *testing.T, repo *git.Repository, name, url string) {
	t.Helper()
	_, err := repo.CreateRemote(&config.RemoteConfig{
		Name: name,
		URLs: []string{url},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func commitToRepo(t *testing.T, repo *git.Repository, msg, fileName, content string) {
	t.Helper()
	w, err := repo.Worktree()
	if err != nil {
		t.Fatal(err)
	}

	f, err := os.Create(filepath.Join(w.Filesystem.Root(), fileName))
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString(content)
	f.Close()

	_, err = w.Add(fileName)
	if err != nil {
		t.Fatal(err)
	}

	_, err = w.Commit(msg, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test",
			Email: "test@test.com",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestScanFindsRepoWithoutRemote(t *testing.T) {
	dir := t.TempDir()

	repoA := filepath.Join(dir, "repo-a")
	r := initRepo(t, repoA)
	commitToRepo(t, r, "init", "f.txt", "content")

	results, err := Scan(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Path != repoA {
		t.Fatalf("expected %q, got %q", repoA, results[0].Path)
	}
}

func TestScanSkipsRepoWithRemote(t *testing.T) {
	dir := t.TempDir()

	repoA := filepath.Join(dir, "repo-a")
	r := initRepo(t, repoA)
	addRemote(t, r, "origin", "https://example.com/repo.git")

	results, err := Scan(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
	}
}

func TestScanSkipsExcludedMirror(t *testing.T) {
	dir := t.TempDir()

	mirror := filepath.Join(dir, "mirror")
	r1 := initRepo(t, mirror)
	commitToRepo(t, r1, "init", "f.txt", "content")

	repoA := filepath.Join(dir, "repo-a")
	r2 := initRepo(t, repoA)
	commitToRepo(t, r2, "init", "f.txt", "content")

	results, err := Scan(dir, mirror)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Path != repoA {
		t.Fatalf("expected %q, got %q", repoA, results[0].Path)
	}
}

func TestScanCountsCommits(t *testing.T) {
	dir := t.TempDir()

	repoA := filepath.Join(dir, "repo-a")
	r := initRepo(t, repoA)
	commitToRepo(t, r, "first", "a.txt", "a")
	commitToRepo(t, r, "second", "b.txt", "b")
	commitToRepo(t, r, "third", "c.txt", "c")

	results, err := Scan(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].CommitCount != 3 {
		t.Fatalf("expected 3 commits, got %d", results[0].CommitCount)
	}
}

func TestScanFindsMultipleRepos(t *testing.T) {
	dir := t.TempDir()

	repoA := filepath.Join(dir, "repo-a")
	r1 := initRepo(t, repoA)
	commitToRepo(t, r1, "init", "f.txt", "content")

	repoB := filepath.Join(dir, "sub", "repo-b")
	r2 := initRepo(t, repoB)
	commitToRepo(t, r2, "init", "f.txt", "content")

	results, err := Scan(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
}

func TestScanIgnoresPermissionError(t *testing.T) {
	dir := t.TempDir()

	restricted := filepath.Join(dir, "restricted")
	os.MkdirAll(restricted, 0000)
	defer os.Chmod(restricted, 0755)

	repoA := filepath.Join(dir, "repo-a")
	r := initRepo(t, repoA)
	commitToRepo(t, r, "init", "f.txt", "content")

	results, err := Scan(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Path != repoA {
		t.Fatalf("expected %q, got %q", repoA, results[0].Path)
	}
}
