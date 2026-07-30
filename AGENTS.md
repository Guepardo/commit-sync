# commit-sync — AGENTS.md

- **Module**: `github.com/allyson/commit-sync` — single module, no monorepo
- **Entrypoint**: `main.go` → `cmd.Execute()`
- **CLI framework**: `github.com/spf13/cobra`
- **Git library**: `github.com/go-git/go-git/v5` (pure Go, no shell `git`)
- **Dedup mechanism**: looks for `Mirrored-From: <source-path> <source-hash>` trailer in mirror commit messages
- **Config**: `~/.config/commit-sync/config.json` (`{"mirror_path":"..."}`), created/read by `internal/config`
- **Mirror repo**: auto-initialized via `go-git` on first sync if missing
- **Core data flow**: `scanner.Scan(root, exclude)` → `syncer.Sync(results)` → writes commits + `mirrorBranch` ref + symbolic HEAD
- **Known quirks**:
  - `sync` runs `scanner.Scan()` internally — no need to call `scan` first except for preview
  - merge commits (`NumParents() > 1`) are silently skipped
  - only the **default branch** of each source repo is inspected
  - commits sorted by `Author.When` (tie-break: hash string)
  - mirror always writes to `refs/heads/main` regardless of source branches
- **Commands**: `set-mirror <path>`, `scan <root>`, `sync <root> [--dry-run]`, `status`

## Commands

| build | `go build -o commit-sync .` |
|---|---|
| test (all) | `go test ./...` |
| test (single pkg) | `go test ./internal/scanner/` |
| test (single func) | `go test -run TestSyncerSkipsMergeCommits ./internal/syncer/` |
| vet | `go vet ./...` |
| complexity | `make complexity` (requires `gocyclo`, pinned in `tools/tools.go`) |
| tidy | `go mod tidy` |

## Testing quirks

- All tests use `t.TempDir()` — no fixtures or outside dependencies
- Syncer tests increment `commitTime` by 1s per commit to guarantee ordering
- `Scan` tests add remotes programmatically via `config.RemoteConfig`
- Mirror ref setup for sync tests uses `plumbing.NewHashReference` + `plumbing.NewSymbolicReference` directly on storer (not the higher-level `git.Repository` API)

## Structural conventions

- `cmd/` — one file per cobra command, `init()` registers subcommands on `rootCmd`
- `internal/` — three packages: `config`, `scanner`, `syncer`
- `tools/tools.go` — build-tag-gated import pin for `gocyclo` (never edit)
- `.gitignore` ignores root binary `commit-sync`, `/tmp/`, `*.test`

## Release

| skill | `use the release skill` — cria tag semântica e push |
|---|---|
| trigger | push de tag `v*` → `.github/workflows/release.yml` compila e publica no GitHub Releases via GoReleaser |
| config | `.goreleaser.yaml` — build cruzado linux/darwin/windows × amd64/arm64 |

Use a **release skill** (`use the release skill`) quando quiser criar uma nova versão. Ela pergunta se o bump é `minir` (patch), `medium` (minor) ou `major`, calcula a próxima tag, cria e faz push. A Action de release cuida do resto.
