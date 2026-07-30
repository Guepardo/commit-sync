---
name: complexity-refactor
description: Roda o quality gate de complexidade e sugere refatorações baseadas no primeiro item mais complexo
compatibility: opencode
---

# Skill: complexity-refactor — Quality Gate de Complexidade

## Visão Geral

Pega o primeiro item do `make complexity` (que lista os 20 métodos mais complexos via `gocyclo`), analisa a função sob a ótica das três causas de complexidade (Change Amplification, Cognitive Load, Unknown Unknowns) e propõe refatorações concretas.

O agente **nunca modifica código automaticamente** — apenas analisa e sugere. A decisão de refatorar é do usuário.

## Fluxo

### 1. Rodar o quality gate

```bash
make complexity
```

Extrair a **primeira linha** (sem o cabeçalho). Exemplo de saída:

```
    10  main(*main.compose) (*main.compose, error)
     9  main(*main.compose).Run main.go:48:1
     8  main.main main.go:12:1
```

O alvo da vez é o primeiro item: `main(*main.compose)` com complexidade 10.

### 2. Identificar o arquivo e a função

A primeira linha tem o formato:

```
<complexity> <package>(<receiver>).<function> <file>:<line>:<col>
```

Extrair:
- `complexity` (número inteiro)
- `package` (ex: `main`)
- `receiver` (se houver, ex: `*main.compose`)
- `function` (ex: `Run`)
- `file` (caminho absoluto ou relativo)
- `line`

Ler o arquivo na linha indicada para entender o contexto completo da função (cabeçalho + body).

### 3. Analisar a função

Avaliar a função contra as **três causas de complexidade**:

| Causa | Pergunta |
|---|---|
| **Change Amplification** | Quantos arquivos precisariam mudar se eu alterar esta função? |
| **Cognitive Load** | Quanto contexto o leitor precisa carregar? Ela depende de muitos módulos externos? Parâmetros demais? |
| **Unknown Unknowns** | Existem pré-condições implícitas? Ordem obrigatória de chamadas? Efeitos colaterais invisíveis? |

### 4. Verificar Deep Modules

A função atual é um **módulo profundo** ou raso?

- Interface pequena e implementação complexa → **profundo** ✅
- Interface grande e pouca lógica → **raso** ❌

### 5. Propor refatorações

Listar uma ou mais sugestões priorizando mudanças que:

1. Reduzam dependências
2. Diminuam conhecimento necessário para usar o módulo
3. Escondam detalhes internos
4. Centralizem regras de negócio
5. Reduzam efeitos colaterais
6. Tornem APIs menores
7. Eliminem duplicação de conhecimento
8. Aumentem previsibilidade

Cada sugestão deve responder:

- Por que isso reduz complexidade?
- Qual causa ataca (Change Amplification / Cognitive Load / Unknown Unknowns)?
- A mudança cria um módulo mais profundo?
- Ela apenas move complexidade de lugar?

### 6. Validar contra as perguntas do guia

Antes de finalizar, responder:

- Esta mudança reduz o custo cognitivo?
- Ela diminui Change Amplification?
- Ela reduz Unknown Unknowns?
- Ela cria um módulo mais profundo?
- Ela melhora o encapsulamento?
- A API ficou mais simples?
- O código ficou mais previsível?
- Um novo desenvolvedor entenderia isso mais rapidamente?

Se a maioria das respostas for **"não"**, a refatoração provavelmente não vale a pena.

### 7. Verificação pós-refatoração

**Este passo aplica-se apenas se o usuário optar por implementar as sugestões.**

Após a refatoração manual, o usuário deve rodar para garantir que nada quebrou:

```bash
go vet ./...
go test ./...
make complexity
```

Comparar a nova saída do `make complexity` com a anterior — a função refatorada deve ter sumido do topo da lista ou reduzido significativamente sua complexidade.

Incluir no resumo um bloco de verificação:

```
### Pós-refatoração (executar manualmente)
- [ ] go vet ./...
- [ ] go test ./...
- [ ] make complexity (confirmar redução)
```

### 8. Exibir resumo

```
## Análise de Complexidade

### Função: <package>.<function>
### Arquivo: <file>:<linha>
### Complexidade: <N>

### Diagnóstico
<explicação do porquê essa função é complexa>

### Causas identificadas
- [Change Amplification | Cognitive Load | Unknown Unknowns]: <detalhe>

### Sugestões de refatoração

1. **<título breve>**
   - <descrição>
   - Ataca: <causa(s)>
   - Custo-benefício: <alto/médio/baixo>
   - ⚠️ Apenas move complexidade? sim/não

2. **<título breve>**
   ...
```

## Regras

- **Nunca modificar código.** Apenas analisar e sugerir.
- Se a complexidade for aceitável (ex: switch de casos simples), dizer que não vale a pena refatorar.
- Sempre incluir o trecho relevante da função na análise.
- Considerar que o usuário vai rodar a skill novamente para o próximo item.
