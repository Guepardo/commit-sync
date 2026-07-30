# commit-sync

Sincroniza commits de repositórios Git locais **sem remote** em um único repositório mirror, preservando metadados originais (autor, data, mensagem, árvore de arquivos).

## Problema

Repositórios Git sem remote (criados com `git init` para experimentos ou estudos) ficam isolados — seus commits não são versionados em lugar nenhum. O `commit-sync` descobre esses repositórios e replica os commits em um mirror central, em ordem cronológica linear.

## Arquitetura

```
                           ┌─────────────────────────────┐
                           │   commit-sync set-mirror     │
                           │   ~/meu-mirror               │
                           └────────────┬────────────────┘
                                        │
                                        ▼
                           ┌─────────────────────────────┐
                           │    ~/.config/commit-sync/   │
                           │       config.json           │
                           │   { "mirror_path": "..." }  │
                           └────────────┬────────────────┘
                                        │
              ┌─────────────────────────┼─────────────────────────┐
              │                         │                         │
              ▼                         ▼                         ▼
  ┌─────────────────────┐   ┌─────────────────────┐   ┌─────────────────────┐
  │  scan <root>        │   │  sync <root>        │   │  status             │
  │                     │   │                     │   │                     │
  │  filepath.WalkDir   │   │  scanner.Scan()     │   │  git.PlainOpen()    │
  │     ↓               │   │     ↓               │   │     ↓               │
  │  detecta .git/      │   │  syncer.Sync()      │   │  exibe estado      │
  │     ↓               │   │     ↓               │   │                     │
  │  git.Remotes() == 0 │   │  constrói dedup map │   └─────────────────────┘
  │     ↓               │   │  (lê mirror commits)│
  │  lista resultados   │   │     ↓               │
  └─────────────────────┘   │  itera source repos │
                            │     ↓               │
                            │  ordena por data    │
                            │     ↓               │
                            │  cria commits no    │
                            │  mirror com trailer │
                            │  Mirrored-From:     │
                            │     ↓               │
                            │  atualiza refs/     │
                            │  heads/main         │
                            └─────────────────────┘
```

## Fluxo

1. **`set-mirror <path>`** — salva o caminho do mirror no `config.json`
2. **`scan <root>`** — percorre `<root>` recursivamente atrás de pastas `.git`; filtra apenas repositórios **sem nenhum remote**; exclui o próprio mirror
3. **`sync <root>`** — para cada repositório encontrado, lê os commits da branch default; ignora merge commits e commits já sincronizados (via trailer `Mirrored-From`); ordena tudo por data e recria no mirror em ordem linear
4. **`status`** — exibe o caminho do mirror, branch e total de commits sincronizados

## Instalação

### Binário pré-compilado (recomendado)

Baixe o arquivo correspondente ao seu SO/arquitetura da [página de releases](https://github.com/Guepardo/commit-sync/releases), extraia e coloque o binário no `PATH`:

```bash
# Linux x86-64
curl -LO https://github.com/Guepardo/commit-sync/releases/download/v0.0.1/commit-sync_v0.0.1_linux_amd64.tar.gz
tar xzf commit-sync_v0.0.1_linux_amd64.tar.gz
sudo install commit-sync /usr/local/bin/

# Linux ARM64
curl -LO https://github.com/Guepardo/commit-sync/releases/download/v0.0.1/commit-sync_v0.0.1_linux_arm64.tar.gz
tar xzf commit-sync_v0.0.1_linux_arm64.tar.gz
sudo install commit-sync /usr/local/bin/

# macOS (Intel)
curl -LO https://github.com/Guepardo/commit-sync/releases/download/v0.0.1/commit-sync_v0.0.1_darwin_amd64.tar.gz
tar xzf commit-sync_v0.0.1_darwin_amd64.tar.gz
sudo install commit-sync /usr/local/bin/

# macOS (Apple Silicon)
curl -LO https://github.com/Guepardo/commit-sync/releases/download/v0.0.1/commit-sync_v0.0.1_darwin_arm64.tar.gz
tar xzf commit-sync_v0.0.1_darwin_arm64.tar.gz
sudo install commit-sync /usr/local/bin/

# Windows (PowerShell)
# Baixe https://github.com/Guepardo/commit-sync/releases/download/v0.0.1/commit-sync_v0.0.1_windows_amd64.zip
# Extraia e adicione ao PATH manualmente
```

### Via Go

```bash
go install github.com/Guepardo/commit-sync@latest
```

### Build local

```bash
git clone https://github.com/Guepardo/commit-sync.git
cd commit-sync
go build -o commit-sync .
```

## Uso

```bash
# 1. Configurar o repositório mirror (cria se não existir)
commit-sync set-mirror /caminho/do/mirror

# 2. Escanear diretório em busca de repositórios sem remote
commit-sync scan /caminho/raiz

# 3. Sincronizar (com preview)
commit-sync sync /caminho/raiz --dry-run

# 4. Sincronizar (executar)
commit-sync sync /caminho/raiz

# 5. Verificar status do mirror
commit-sync status
```

### Exemplo completo

```bash
commit-sync set-mirror ~/meus-commits-mirror

commit-sync scan ~/documentos

commit-sync sync ~/documentos --dry-run
commit-sync sync ~/documentos

commit-sync status
```

## Limitações

- Apenas commits da **branch default** de cada repositório
- **Merge commits são ignorados** (apenas commits com 1 parent)
- A topologia original (DAG) é perdida — todos os commits viram uma sequência linear
- Assinaturas GPG não são preservadas (o hash do commit muda)
- Submódulos não são processados

## Como contribuir

```bash
git clone <url>
cd commit-sync
go test ./...
```
