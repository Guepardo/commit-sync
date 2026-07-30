package syncer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"

	"github.com/Guepardo/commit-sync/internal/scanner"
)

func initRepo(t *testing.T, path string) *git.Repository {
	t.Helper()
	repo, err := git.PlainInit(path, false)
	if err != nil {
		t.Fatal(err)
	}
	return repo
}

var commitTime = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

func commitToRepo(t *testing.T, repo *git.Repository, msg, fileName, content string) *object.Commit {
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

	commitTime = commitTime.Add(time.Second)
	hash, err := w.Commit(msg, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Author",
			Email: "author@test.com",
			When:  commitTime,
		},
		Committer: &object.Signature{
			Name:  "Committer",
			Email: "committer@test.com",
			When:  commitTime,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	c, err := repo.CommitObject(hash)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestSyncerSyncsCommitsToMirror(t *testing.T) {
	dir := t.TempDir()

	sourceDir := filepath.Join(dir, "source")
	sourceRepo := initRepo(t, sourceDir)
	c1 := commitToRepo(t, sourceRepo, "first commit", "a.txt", "hello")
	c2 := commitToRepo(t, sourceRepo, "second commit", "b.txt", "world")

	mirrorDir := filepath.Join(dir, "mirror")
	initRepo(t, mirrorDir)

	s := New(mirrorDir)
	repos := []scanner.ScanResult{
		{Path: sourceDir, DefaultBranch: "refs/heads/main"},
	}
	n, err := s.Sync(repos)
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("expected 2 synced, got %d", n)
	}

	mirrorRepo, err := git.PlainOpen(mirrorDir)
	if err != nil {
		t.Fatal(err)
	}

	head, err := mirrorRepo.Head()
	if err != nil {
		t.Fatal(err)
	}

	commitCount := 0
	var commits []*object.Commit
	iter, _ := mirrorRepo.Log(&git.LogOptions{From: head.Hash()})
	iter.ForEach(func(c *object.Commit) error {
		commitCount++
		commits = append(commits, c)
		return nil
	})
	iter.Close()

	if commitCount != 2 {
		t.Fatalf("expected 2 mirror commits, got %d", commitCount)
	}

	lastCommit := commits[0]
	if !strings.Contains(lastCommit.Message, c2.Message) {
		t.Fatalf("expected c2 message in lastCommit: c2.Message=%q lastCommit.Message=%q", c2.Message, lastCommit.Message)
	}
	if !strings.Contains(lastCommit.Message, c2.Hash.String()) {
		t.Fatalf("expected mirrored trailer for second commit")
	}

	firstCommit := commits[1]
	if !strings.Contains(firstCommit.Message, strings.TrimRight(c1.Message, "\n")) {
		t.Fatalf("c1.Message=%q firstCommit.Message=%q", c1.Message, firstCommit.Message)
	}
	if _, _, ok := parseMirroredFrom(firstCommit.Message); !ok {
		t.Fatalf("expected mirrored trailer in first commit")
	}
}

func TestSyncerSkipsAlreadySyncedCommits(t *testing.T) {
	dir := t.TempDir()

	sourceDir := filepath.Join(dir, "source")
	sourceRepo := initRepo(t, sourceDir)
	commitToRepo(t, sourceRepo, "first", "a.txt", "a")
	commitToRepo(t, sourceRepo, "second", "b.txt", "b")

	mirrorDir := filepath.Join(dir, "mirror")
	s := New(mirrorDir)
	repos := []scanner.ScanResult{
		{Path: sourceDir, DefaultBranch: "refs/heads/main"},
	}
	n, err := s.Sync(repos)
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("expected 2 first sync, got %d", n)
	}

	n, err = s.Sync(repos)
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("expected 0 new on second sync, got %d", n)
	}
}

func TestSyncerSyncsOnlyNewCommits(t *testing.T) {
	dir := t.TempDir()

	sourceDir := filepath.Join(dir, "source")
	sourceRepo := initRepo(t, sourceDir)
	commitToRepo(t, sourceRepo, "first", "a.txt", "a")

	mirrorDir := filepath.Join(dir, "mirror")
	s := New(mirrorDir)
	repos := []scanner.ScanResult{
		{Path: sourceDir, DefaultBranch: "refs/heads/main"},
	}
	s.Sync(repos)

	commitToRepo(t, sourceRepo, "third", "c.txt", "c")

	n, err := s.Sync(repos)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("expected 1 new commit on second sync, got %d", n)
	}
}

func TestSyncerSkipsMergeCommits(t *testing.T) {
	dir := t.TempDir()

	sourceDir := filepath.Join(dir, "source")
	sourceRepo := initRepo(t, sourceDir)

	commitToRepo(t, sourceRepo, "first", "a.txt", "a")

	mainHead, _ := sourceRepo.Head()

	refName := plumbing.ReferenceName("refs/heads/feature")
	ref := plumbing.NewHashReference(refName, mainHead.Hash())
	sourceRepo.Storer.SetReference(ref)

	w, _ := sourceRepo.Worktree()
	w.Checkout(&git.CheckoutOptions{Branch: refName})
	commitToRepo(t, sourceRepo, "feature work", "b.txt", "b")
	featureHead, _ := sourceRepo.Head()

	w.Checkout(&git.CheckoutOptions{Branch: plumbing.ReferenceName("refs/heads/main")})
	mainCommit, _ := sourceRepo.CommitObject(mainHead.Hash())

	mergeCommit := &object.Commit{
		Author:       object.Signature{Name: "Merger", Email: "m@t.com"},
		Committer:    object.Signature{Name: "Merger", Email: "m@t.com"},
		Message:      "merge feature",
		TreeHash:     mainCommit.TreeHash,
		ParentHashes: []plumbing.Hash{mainHead.Hash(), featureHead.Hash()},
	}
	obj := sourceRepo.Storer.NewEncodedObject()
	if err := mergeCommit.Encode(obj); err != nil {
		t.Fatal(err)
	}
	mergeHash, err := sourceRepo.Storer.SetEncodedObject(obj)
	if err != nil {
		t.Fatal(err)
	}

	mainRef := plumbing.NewHashReference(plumbing.ReferenceName("refs/heads/main"), mergeHash)
	sourceRepo.Storer.SetReference(mainRef)
	headRef2 := plumbing.NewSymbolicReference(plumbing.HEAD, plumbing.ReferenceName("refs/heads/main"))
	sourceRepo.Storer.SetReference(headRef2)

	mirrorDir := filepath.Join(dir, "mirror")
	s := New(mirrorDir)
	repos := []scanner.ScanResult{
		{Path: sourceDir, DefaultBranch: "refs/heads/main"},
	}
	_, syncErr := s.Sync(repos)
	if syncErr != nil {
		t.Fatal(syncErr)
	}

	mirrorRepo, _ := git.PlainOpen(mirrorDir)
	head, err := mirrorRepo.Head()
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	iter, _ := mirrorRepo.Log(&git.LogOptions{From: head.Hash()})
	iter.ForEach(func(c *object.Commit) error {
		count++
		return nil
	})
	iter.Close()

	if count != 2 {
		t.Fatalf("expected 2 mirror commits (first + feature, no merge), got %d", count)
	}
}

func TestSyncerDryRun(t *testing.T) {
	dir := t.TempDir()

	sourceDir := filepath.Join(dir, "source")
	sourceRepo := initRepo(t, sourceDir)
	commitToRepo(t, sourceRepo, "first", "a.txt", "a")
	commitToRepo(t, sourceRepo, "second", "b.txt", "b")

	mirrorDir := filepath.Join(dir, "mirror")
	initRepo(t, mirrorDir)

	s := New(mirrorDir)
	s.SetDryRun(true)
	repos := []scanner.ScanResult{
		{Path: sourceDir, DefaultBranch: "refs/heads/main"},
	}
	n, err := s.Sync(repos)
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("expected 2 reported, got %d", n)
	}

	mirrorRepo, _ := git.PlainOpen(mirrorDir)
	_, err = mirrorRepo.Head()
	if err == nil {
		t.Fatal("expected no HEAD after dry run (mirror should be empty)")
	}
}

func TestSyncerPreservesMetadata(t *testing.T) {
	dir := t.TempDir()

	sourceDir := filepath.Join(dir, "source")
	sourceRepo := initRepo(t, sourceDir)
	orig := commitToRepo(t, sourceRepo, "my message", "f.txt", "data")

	mirrorDir := filepath.Join(dir, "mirror")
	s := New(mirrorDir)
	repos := []scanner.ScanResult{
		{Path: sourceDir, DefaultBranch: "refs/heads/main"},
	}
	s.Sync(repos)

	mirrorRepo, _ := git.PlainOpen(mirrorDir)
	head, _ := mirrorRepo.Head()
	mirrorCommit, _ := mirrorRepo.CommitObject(head.Hash())

	if mirrorCommit.Author.Name != orig.Author.Name {
		t.Fatalf("author name: expected %q, got %q", orig.Author.Name, mirrorCommit.Author.Name)
	}
	if mirrorCommit.Author.Email != orig.Author.Email {
		t.Fatalf("author email: expected %q, got %q", orig.Author.Email, mirrorCommit.Author.Email)
	}
	if !mirrorCommit.Author.When.Equal(orig.Author.When) {
		t.Fatalf("author time: expected %v, got %v", orig.Author.When, mirrorCommit.Author.When)
	}
	if mirrorCommit.Committer.Name != orig.Committer.Name {
		t.Fatalf("committer name: expected %q, got %q", orig.Committer.Name, mirrorCommit.Committer.Name)
	}
	if mirrorCommit.TreeHash != orig.TreeHash {
		t.Fatalf("tree hash differs")
	}
}

func TestSyncerMirrorInitCreatesRepo(t *testing.T) {
	dir := t.TempDir()

	mirrorDir := filepath.Join(dir, "mirror")

	s := New(mirrorDir)
	s.Sync(nil)

	repo, err := git.PlainOpen(mirrorDir)
	if err != nil {
		t.Fatalf("mirror should have been initialized: %v", err)
	}
	_ = repo
}

func TestParseMirroredFrom(t *testing.T) {
	msg := "some commit message\n\nMirrored-From: /home/user/repo abc123def\n"
	path, hash, ok := parseMirroredFrom(msg)
	if !ok {
		t.Fatal("expected ok")
	}
	if path != "/home/user/repo" {
		t.Fatalf("expected path /home/user/repo, got %q", path)
	}
	if hash != "abc123def" {
		t.Fatalf("expected hash abc123def, got %q", hash)
	}
}

func TestParseMirroredFromNoTrailer(t *testing.T) {
	_, _, ok := parseMirroredFrom("normal message")
	if ok {
		t.Fatal("expected not ok")
	}
}

func TestAppendMirroredFrom(t *testing.T) {
	msg := appendMirroredFrom("hello", "/p", "abc")
	if !strings.Contains(msg, "Mirrored-From: /p abc") {
		t.Fatalf("trailer not found in: %q", msg)
	}
}
