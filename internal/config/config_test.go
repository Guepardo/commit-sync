package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	cfg := &Config{MirrorPath: "/tmp/mirror"}
	if err := Save(cfg); err != nil {
		t.Fatal(err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.MirrorPath != cfg.MirrorPath {
		t.Fatalf("expected %q, got %q", cfg.MirrorPath, loaded.MirrorPath)
	}
}

func TestLoadNonExistent(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.MirrorPath != "" {
		t.Fatalf("expected empty, got %q", cfg.MirrorPath)
	}
}

func TestConfigDirCreation(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	cfg := &Config{MirrorPath: "/tmp/mirror"}
	if err := Save(cfg); err != nil {
		t.Fatal(err)
	}

	configDirPath := filepath.Join(dir, ".config", "commit-sync", "config.json")
	if _, err := os.Stat(configDirPath); os.IsNotExist(err) {
		t.Fatal("config file was not created")
	}
}
