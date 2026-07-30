---
name: release
description: Cria release com versionamento semântico (tag + push)
compatibility: opencode
---

# Skill: release — Criar Release

## Visão Geral

Cria uma nova release seguindo versionamento semântico: pergunta o tipo de bump, calcula a próxima versão, cria a tag e faz push.

## Fluxo

### 1. Validar working tree

```bash
git status --porcelain
```

Se sujo, avisar e abortar — o usuário precisa commitar ou stash primeiro.

### 2. Descobrir última tag

Pegar a tag mais recente ordenada por versão semântica:

```bash
git tag --list 'v*' --sort=-version:refname | head -1
```

Se não existir nenhuma tag, assumir `v0.1.0` como base.

### 3. Perguntar tipo de bump

Usar a `question` tool com três opções — exibir a versão atual no prompt:

| Opção | Parte do semver | Exemplo (v1.2.3 →) |
|---|---|---|
| **minir** (patch) | `z` em `x.y.z` | `v1.2.4` |
| **medium** (minor) | `y` em `x.y.z` | `v1.3.0` |
| **major** | `x` em `x.y.z` | `v2.0.0` |

Usar labels em pt-BR: `minir`, `medium`, `major`. Descrever o significado de cada um.

### 4. Calcular nova versão

Fazer o bump semântico: extrair os três números da tag, incrementar o componente escolhido e zerar os da direita.

### 5. Criar tag local

```bash
git tag v<nova-versao>
```

### 6. Fazer push da tag

```bash
git push origin v<nova-versao>
```

### 7. Mostrar resumo e sugerir próximos passos

Exibir algo como:

```
✓ Release v1.3.0 criada com sucesso
  → git push origin v1.3.0 (ok)
  → A GitHub Action de release será disparada automaticamente
  → Acompanhe em: https://github.com/allyson/commit-sync/actions
```
