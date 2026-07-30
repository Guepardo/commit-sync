# commit-sync

Sincroniza commits de repositórios Git locais **sem remote** em um único repositório mirror, preservando metadados originais (autor, data, mensagem, árvore de arquivos).

## Problema

Repositórios Git sem remote (criados com `git init` para experimentos ou estudos) ficam isolados — seus commits não são versionados em lugar nenhum. O `commit-sync` descobre esses repositórios e replica os commits em um mirror central, em ordem cronológica linear.

## Como funciona

1. **Varredura** — percorre um diretório raiz recursivamente atrás de pastas `.git`
2. **Filtro** — mantém apenas repositórios **sem nenhum remote** configurado; exclui o próprio mirror
3. **Deduplicação** — cada commit espelhado recebe um trailer `Mirrored-From: <path> <hash>` na mensagem; na próxima execução commits já sincronizados são ignorados
4. **Sincronização** — commits não-merge são ordenados por data e recriados no mirror em uma branch linear `main`, preservando author, committer, data, mensagem e conteúdo dos arquivos

## Instalação

```bash
# Com Go instalado
go install github.com/allyson/commit-sync@latest

# Ou build local
git clone <url>
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
