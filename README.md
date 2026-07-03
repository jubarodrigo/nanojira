# Nanojira — Task Management API

Backend de gestão de tarefas para times de operações, desenvolvido como take-home assessment. Inspirado em um Jira simplificado: managers criam e atribuem trabalho; workers executam e atualizam o status; atribuições disparam e-mail real.

## Stack

- **Go 1.25** + **Gin** (HTTP)
- **PostgreSQL** (persistência)
- **Mailpit** (SMTP local para e-mails de atribuição)
- **Zap** (logging estruturado)
- **mockgen** (mocks para testes unitários)

## Arquitetura

```
cmd/api/          → entrypoint, wiring de dependências
internal/
  domain/         → entidades, regras de workflow, erros de domínio
  handler/        → camada HTTP (Gin)
  service/        → casos de uso (um arquivo por operação)
  repository/     → interfaces + implementação Postgres
  email/          → envio SMTP
  config/         → variáveis de ambiente
migrations/       → schema versionado com [goose](https://github.com/pressly/goose) (uma migration por tabela)
mocks/            → gerados via mockgen
```

Padrões aplicados: interfaces, injeção de dependência, separação handler → service → repository, erros com wrap (`fmt.Errorf("context: %w", err)`), respostas de API com `code` + `message` para o cliente explicar falhas ao usuário.

## Executar com Docker (recomendado)

```bash
docker compose up --build
```

Serviços:

| Serviço   | URL                        |
|-----------|----------------------------|
| API       | http://localhost:8080      |
| Mailpit UI| http://localhost:8025      |
| Postgres  | localhost:5432             |

Health check:

```bash
curl http://localhost:8080/health
```

## Executar localmente (sem Docker na API)

```bash
# Subir dependências
docker compose up -d postgres mailpit

# Variáveis (ou copie .env.example)
export DATABASE_URL=postgres://nanojira:nanojira@localhost:5432/nanojira?sslmode=disable
export SMTP_HOST=localhost
export SMTP_PORT=1025

make run
# ou: go run ./cmd/api
```

As migrations rodam automaticamente no boot da API via goose. Para executar manualmente:

```bash
make migrate-up       # aplicar pendentes
make migrate-down     # reverter última migration
make migrate-status   # listar versões aplicadas
```

### Migrations (goose)

Cada arquivo em `migrations/` cobre um contexto isolado:

| Arquivo | Conteúdo |
|---------|----------|
| `00001_enable_pgcrypto.sql` | Extensão para UUIDs |
| `00002_create_users.sql` | Tabela `users` |
| `00003_create_tasks.sql` | Tabela `tasks` + índices |
| `00004_create_assignment_notifications.sql` | Tabela `assignment_notifications` |
| `00005_create_stepback_requests.sql` | Tabela `stepback_requests` |
| `00006_seed_users.sql` | Dados iniciais de usuários |
| `00007_seed_tasks.sql` | Dados iniciais de tarefas |

Se o banco já existia com o schema antigo (arquivo único), recrie o volume antes de subir:

```bash
docker compose down -v && docker compose up --build
```

## Autenticação (simulada)

Contas vêm de outro sistema. Para testes, use o header:

```
X-User-ID: <uuid-do-usuario>
```

### Usuários seed

| ID | Nome | E-mail | Papel |
|----|------|--------|-------|
| `11111111-1111-1111-1111-111111111101` | Alice Manager | alice.manager@example.com | manager |
| `11111111-1111-1111-1111-111111111102` | Bob Manager | bob.manager@example.com | manager |
| `22222222-2222-2222-2222-222222222201` | Carol Worker | carol.worker@example.com | worker |
| `22222222-2222-2222-2222-222222222202` | Dave Worker | dave.worker@example.com | worker |
| `22222222-2222-2222-2222-222222222203` | Eve Worker | eve.worker@example.com | worker |

## API

Base: `/api/v1` — todas as rotas exigem `X-User-ID`.

| Método | Rota | Quem | Descrição |
|--------|------|------|-----------|
| GET | `/tasks` | todos | Lista tarefas (manager: todas; worker: só as atribuídas). Query: `status`, `limit`, `offset` |
| POST | `/tasks` | manager | Cria tarefa (com ou sem `assignee_id`) |
| GET | `/tasks/:id` | todos | Detalhe da tarefa |
| PATCH | `/tasks/:id/assign` | manager | Atribui worker → envia e-mail |
| PATCH | `/tasks/:id/status` | worker / manager | Worker: avança status ou solicita retrocesso. Manager: aprova/rejeita pendência |
| GET | `/tasks/:id/notifications` | todos | Histórico de notificações de atribuição (rastreabilidade) |

### Workflow de status

```
todo → doing → testing → done
         ↕        ↕
       on_hold ←──┘
```

- **Forward** (worker): `PATCH /tasks/:id/status` com `{"status":"..."}` — aplica imediatamente.
- **Backward** (worker): mesmo endpoint com `{"status":"...", "reason":"..."}` — a task **permanece no status atual** e expõe `pending_status_change` até o manager decidir.
- **Aprovação** (manager): `PATCH /tasks/:id/status` com `{"approve_status_change": true}` — move para o status solicitado; `false` rejeita e mantém o atual.
- Managers **não** alteram status diretamente; só aprovam ou rejeitam pendências.

Resposta da task com pendência:

```json
{
  "id": "...",
  "status": "testing",
  "pending_status_change": {
    "id": "...",
    "requested_status": "doing",
    "reason": "Testes falharam na integração",
    "requested_by_id": "...",
    "requested_at": "..."
  }
}
```

### Cenário de teste manual

```bash
# 1. Manager cria tarefa sem responsável
curl -s -X POST http://localhost:8080/api/v1/tasks \
  -H "Content-Type: application/json" \
  -H "X-User-ID: 11111111-1111-1111-1111-111111111101" \
  -d '{"title":"Revisar runbook","description":"Atualizar procedimento de incidentes"}'

# 2. Manager atribui a Carol (e-mail em http://localhost:8025)
curl -s -X PATCH http://localhost:8080/api/v1/tasks/<TASK_ID>/assign \
  -H "Content-Type: application/json" \
  -H "X-User-ID: 11111111-1111-1111-1111-111111111101" \
  -d '{"assignee_id":"22222222-2222-2222-2222-222222222201"}'

# 3. Carol avança status
curl -s -X PATCH http://localhost:8080/api/v1/tasks/<TASK_ID>/status \
  -H "Content-Type: application/json" \
  -H "X-User-ID: 22222222-2222-2222-2222-222222222201" \
  -d '{"status":"doing"}'

# 4. Dave solicita retrocesso (testing → doing) — status permanece "testing"
curl -s -X PATCH http://localhost:8080/api/v1/tasks/<TASK_ID>/status \
  -H "Content-Type: application/json" \
  -H "X-User-ID: 22222222-2222-2222-2222-222222222202" \
  -d '{"status":"doing","reason":"Testes falharam na integração"}'

# 5. Manager aprova a mudança pendente
curl -s -X PATCH http://localhost:8080/api/v1/tasks/<TASK_ID>/status \
  -H "Content-Type: application/json" \
  -H "X-User-ID: 11111111-1111-1111-1111-111111111101" \
  -d '{"approve_status_change":true}'
```

## Testes

```bash
make test          # go test ./...
make mocks         # go generate (mockgen)
```

Testes unitários cobrem: autorização, transições de status, retrocesso pendente, aprovação pelo manager e atribuição com e-mail.

## Makefile

| Comando | Ação |
|---------|------|
| `make run` | API local |
| `make docker-up` | Stack completa |
| `make test` | Testes |
| `make mocks` | Regenerar mocks |
| `make migrate-up` | Aplicar migrations (goose) |
| `make migrate-down` | Reverter última migration |
| `make migrate-status` | Status das migrations |

---

## Reflexões (prompts do assessment)

### 1. Interpretação do cenário e premissas

- O sistema é a **fonte da verdade** para itens de trabalho: quem criou, quem é responsável, em que etapa está.
- **Dois papéis**: manager (visão global, criação, atribuição) e worker (fila própria, progresso).
- Autenticação é externa → simulada via `X-User-ID`.
- “Não retroceder sem bom motivo” → retrocesso via `PATCH status` com motivo; task fica no step atual com `pending_status_change` até aprovação do manager (persistido em `stepback_requests`).
- Managers não executam trabalho alheio → bloqueio de `PATCH status` para managers.
- E-mail de atribuição é **real** (SMTP); localmente via Mailpit. Cada envio é registrado em `assignment_notifications` para rastreabilidade.

### 2. Decisões de design importantes

- **Workflow explícito** no domínio (`ForwardTransitions` / `BackwardTransitions`) — regras testáveis e centralizadas.
- **Um arquivo por caso de uso** na camada `service/` — facilita navegação e testes focados.
- **Erros tipados** (`AppError` com `code`) — cliente pode mapear para mensagens de UI.
- **Interfaces no repository** — Postgres substituível; mocks para testes.
- **Paginação** em listagens (`limit`/`offset`) — preparado para volume crescente.

### 3. Atribuições e notificações

- Atribuição na criação (`assignee_id`) ou via `PATCH .../assign`.
- Ao mudar o responsável, dispara SMTP para o e-mail do worker.
- Falha de e-mail **não persiste** a atribuição silenciosamente — retorna `502 EMAIL_FAILED` (operação falha de forma clara).
- Registro em `assignment_notifications` só após envio bem-sucedido.

### 4. Evolução com time/carga maiores

- Fila assíncrona para e-mails (SQS/RabbitMQ) e workers dedicados.
- Cache de leitura (Redis) para listagens frequentes.
- Índices compostos conforme padrões de query (ex.: `assignee_id + status`).
- Eventos de domínio (`task.assigned`, `task.status_changed`) para integrações.
- Auth real (JWT/OAuth2) no lugar do header.
- Migrations versionadas com goose no boot; em produção rodariam em job separado da API.

### 5. Trade-offs do timebox

- Migrations aplicadas no startup via **goose** (versionadas, com rollback por arquivo).
- Sem testes de integração com Postgres real (só unitários com mocks).
- Sem endpoint separado para listar pendências globais do manager (visível via `GET /tasks` com `pending_status_change`).
- Sem edição de título/descrição após criação.
- Auth minimalista (header) — suficiente para o exercício.

### 6. Antes de produção

- Auth/OAuth integrado ao IdP da empresa.
- Migrations versionadas e rollback.
- Observabilidade (métricas, tracing, alertas de falha de e-mail).
- Rate limiting e validação de input mais rígida.
- Testes de integração e contrato (OpenAPI).
- Secrets em vault; TLS; SMTP autenticado.
- Idempotência em atribuições e retries na fila de e-mail.

---

## Uso de IA

Ferramentas: **Cursor** (scaffolding, boilerplate, testes, README).

- IA gerou estrutura inicial de pastas, handlers e SQL de migration.
- Revisei manualmente: regras de workflow, autorização, tratamento de erros, fluxo de e-mail e testes.
- Sem IA, eu gastaria mais tempo em boilerplate Gin/Postgres e documentação; o desenho de domínio (retrocesso integrado ao status, papéis, traceability) foi intencional.
