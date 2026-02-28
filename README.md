# Propeller Ontology

An ontology-first architecture for property management where a CUE schema is
the single source of truth and code generators derive everything downstream:
database schemas, HTTP handlers, API definitions, event catalogs, authorization
policies, agent tooling, UI components, state machine tests, a signal discovery
engine, and an interactive REPL.

**~2,500 lines of CUE generate a fully functional REST API with 18 entities,
~85 operations, 13 state machines, 11 domain commands with event recording,
versioned migrations, a two-layer UI generation pipeline, cross-cutting signal
intelligence, and a WebSocket-based query console with read/write support.**

---

## Architecture

```
                          ontology/*.cue
                     (single source of truth)
                              |
  ┌──────────┬──────────┬─────┼─────┬──────────┬──────────┬──────────┐
  v          v          v     v     v          v          v          v
entgen   handlergen  apigen  |  eventgen  authzgen  agentgen  openapigen
  |          |          |    |      |          |          |          |
  v          v          v    |      v          v          v          v
ent/      internal/   gen/   |   internal/  gen/opa/  gen/agent/  gen/openapi/
schema/   handler/    proto/ |   worker/    (Rego)    ONTOLOGY.md openapi.json
          server/            |   events.go            SIGNALS.md
                             |                        TOOLS.md
  ┌──────────┬───────────────┤
  v          v               v
uigen    uirender         testgen     replgen     driftcheck
  |          |               |            |            |
  v          v               v            v            v
gen/ui/   gen/ui/         gen/tests/  internal/    cross-boundary
schema/   components/     (314 test   repl/        CUE validation
          types/           cases)     (PQL engine)
          stores/
          api/
```

One `make generate` command runs the full pipeline. Change a field in CUE and
the database schema, HTTP handlers, proto definitions, event types,
authorization policies, agent context documents, UI components, state machine
tests, and REPL schema registry all update automatically.

### Why CUE

CUE provides types, constraints, defaults, and validation in a single
declarative language. Unlike JSON Schema or protobuf, CUE schemas are
composable — entity definitions, value types, enums, state machine
transitions, and signal registrations all live in the same module and
cross-reference each other. The generators read the CUE evaluation result
directly, so there's no intermediate representation to keep in sync.

---

## Domain Model

18 entities across 5 domains, all defined in `ontology/*.cue`:

| Domain | Entities |
|--------|----------|
| **Person** | Person, Organization, PersonRole |
| **Property** | Portfolio, Property, Building, Space |
| **Lease** | Lease, LeaseSpace, Application |
| **Accounting** | Account, LedgerEntry, JournalEntry, BankAccount, Reconciliation |
| **Jurisdiction** | Jurisdiction, PropertyJurisdiction, JurisdictionRule |

Every entity with a `status` field has a state machine defined in
`ontology/state_machines.cue`. 13 state machines enforce transitions at the
persistence layer via generated Ent hooks — no code path can violate them.

### CQRS: Commands and Events

Domain writes flow through commands defined in `commands/*.cue` and produce
events defined in `events/*.cue`. The CUE definitions specify input shapes,
execution plans, affected entities, and required permissions. The runtime
command layer in `internal/handler/custom_*.go` implements each command as a
handler function following a consistent pattern: parse request, validate state
machine transitions, open an Ent transaction, execute multi-entity mutations,
commit, then record a domain event.

**11 commands across 4 domains:**

| Command | Route | What it does |
|---------|-------|--------------|
| **MoveInTenant** | `POST /v1/leases/{id}/move-in` | Validates lease, sets move-in date, transitions lease to active, transitions spaces to occupied, updates tenant attributes, creates security deposit LedgerEntry |
| **RecordPayment** | `POST /v1/leases/{id}/payments` | Creates JournalEntry + LedgerEntry, updates tenant balance and standing |
| **RenewLease** | `POST /v1/leases/{id}/renew` | Creates new Lease copying terms, copies LeaseSpaces, transitions old lease to renewed |
| **InitiateEviction** | `POST /v1/leases/{id}/evict` | Validates eviction reason, transitions lease to eviction, updates tenant standing |
| **SubmitApplication** | `POST /v1/applications` | Creates Application in submitted status with optional fee recording |
| **ApproveApplication** | `POST /v1/applications/{id}/approve` | Transitions application, records decision metadata |
| **DenyApplication** | `POST /v1/applications/{id}/deny` | Transitions application, requires reason for fair housing compliance |
| **OnboardProperty** | `POST /v1/properties/onboard` | Creates Property in onboarding status with bulk Space creation |
| **PostJournalEntry** | `POST /v1/journal-entries/{id}/post` | Validates balanced lines, transitions to posted, creates LedgerEntries |
| **StartReconciliation** | `POST /v1/reconciliations/{id}/start` | Creates Reconciliation, marks matched LedgerEntries, determines balanced/unbalanced |
| **ApproveReconciliation** | `POST /v1/reconciliations/{id}/approve` | Transitions balanced reconciliation to approved |

**10 domain events** — each command records a typed event via the
`internal/event` package. Events fan out as `ActivityEntry` records (one per
affected entity) through the `activity.Store` interface, feeding the signal
discovery system. Event recording is best-effort and non-blocking — it never
fails a command.

Permissions are defined in `policies/*.cue` with 7 permission groups and
field-level access policies. API response shapes live in `api/v1/*.cue` as
their own anti-corruption layer — they import ontology enums but define their
own response structures.

---

## Code Generators

| Generator | Input | Output | What it does |
|-----------|-------|--------|--------------|
| **entgen** | `ontology/*.cue` | `ent/schema/*.go` | Generates Ent ORM schemas with fields, edges, indexes, validators, and state machine hooks |
| **handlergen** | `ontology/*.cue` + `codegen/apigen.cue` | `internal/handler/gen_*.go`, `internal/server/gen_routes.go` | Generates HTTP handlers for CRUD + state transitions, wired to chi routes |
| **apigen** | `ontology/*.cue` + `codegen/apigen.cue` | `gen/proto/*.proto` | Generates Connect-RPC protobuf service definitions |
| **eventgen** | `ontology/*.cue` | `internal/worker/events.go`, `gen/events_catalog.json` | Generates event type constants and a machine-readable event catalog |
| **authzgen** | `ontology/*.cue` | `gen/opa/*.rego` | Generates OPA/Rego policy scaffolds per entity |
| **agentgen** | `ontology/*.cue` | `gen/agent/ONTOLOGY.md`, `SIGNALS.md`, `TOOLS.md`, `propeller-tools.json` | Generates AI agent context: world model, signal reasoning guide, tool definitions |
| **openapigen** | `ontology/*.cue` + `codegen/apigen.cue` | `gen/openapi/openapi.json` | Generates OpenAPI 3.1 spec |
| **uigen** | `ontology/*.cue` + `codegen/uigen.cue` | `gen/ui/schema/*.json` | Generates framework-agnostic JSON UI schemas (Layer 1) |
| **uirender** | `gen/ui/schema/*.json` | `gen/ui/components/`, `gen/ui/types/`, `gen/ui/stores/`, `gen/ui/api/` | Generates Svelte + Skeleton UI + Tailwind components from UI schemas (Layer 2) |
| **testgen** | `ontology/*.cue` + `codegen/testgen.cue` | `gen/tests/*_test.go` | Generates state machine transition test cases (314 tests across 13 state machines) |
| **replgen** | `ontology/*.cue` | `internal/repl/schema/gen_registry.go`, `internal/repl/executor/gen_dispatch.go` | Generates REPL schema registry and typed entity dispatchers for all 18 entities |

**driftcheck** (`cmd/driftcheck`) validates cross-boundary consistency between
the ontology, commands, events, API definitions, and policies — catching
mismatches before they reach production.

Custom operations are marked `custom: true` in `codegen/apigen.cue` and
hand-written in `internal/handler/custom_*.go`. The generators skip these —
they coexist with generated code without conflict. Domain commands
(MoveInTenant, RecordPayment, etc.) implement multi-entity transactional
mutations following a consistent pattern: parse → validate → `client.Tx()` →
mutations → commit → record event. Events fan out through `internal/event/`
into the activity store, feeding the signal discovery system.

---

## REPL and PQL

The Propeller REPL is a web-based interactive console providing direct access
to the Ent entity layer over WebSocket. PQL (Propeller Query Language) is a
domain-specific query syntax that compiles to Ent query builder calls,
supporting both reads and writes across all 18 entities.

### PQL syntax

**Reads:**

```pql
-- Query entities with filtering
find lease where status = "active" and lease_type = "fixed_term"

-- Field projection
find lease where status = "active" select status, lease_type, base_rent_amount_cents

-- Edge traversal (eager loading)
find lease where status = "active" include lease_spaces, tenant_roles

-- Sorting and pagination
find lease order by base_rent_amount_cents desc limit 25 offset 50

-- Clauses can appear in any order
find lease limit 10 where status = "active" order by created_at desc

-- Get by ID
get lease "550e8400-e29b-41d4-a716-446655440000"

-- Count
count lease where status = "active"
```

**Writes:**

```pql
-- Create entity
create portfolio set name = "West Coast Properties"

-- Update entity
update property "550e8400-..." set name = "Maple Heights", year_built = 2005

-- Delete entity
delete building "550e8400-..."
```

**Meta-commands:**

```pql
:schema lease              -- Show entity schema (fields, edges, state machines)
:help find                 -- Help on a specific topic
:history                   -- Show command history
:env                       -- Session info (ID, mode, timestamps)
:clear                     -- Clear output
```

### Operators

| Operator | Example |
|----------|---------|
| `=`, `!=` | `status = "active"` |
| `>`, `<`, `>=`, `<=` | `year_built >= 2000` |
| `like` | `first_name like "J%"` |
| `in` | `status in ["active", "draft"]` |
| `and`, `or` | `status = "active" and type = "fixed_term"` |

### Planner features

- **Fuzzy suggestions** — misspelled entity or field names produce "did you
  mean...?" errors using Levenshtein distance
- **Type coercion** — UUID fields auto-parse, enum fields validate against
  allowed values, integers downcast correctly
- **Immutability enforcement** — create/update/delete rejected for immutable
  entities (LedgerEntry, etc.) with clear errors
- **Computed field protection** — `id`, `created_at`, `updated_at` cannot be
  set in mutations

### Architecture

```
internal/repl/
  pql/            Lexer, parser, AST (recursive descent, any-order clauses)
  schema/         Entity metadata registry (generated from CUE)
  planner/        AST → QueryPlan with validation (fuzzy suggestions on typos)
  executor/       QueryPlan → Ent calls via generated dispatchers
  autocomplete/   Context-aware completions (<50ms from in-memory registry)
  session/        Session lifecycle (30-min idle timeout, 24-hr expiry)
  wire/           WebSocket protocol (execute, cancel, autocomplete, ping)
  meta/           Meta-commands (:help, :clear, :env, :history, :schema)
```

The pipeline: **PQL text** → lexer → tokens → parser → AST → planner
(validates against schema registry) → QueryPlan → executor (dispatches to
typed Ent queries) → JSON results streamed over WebSocket.

### WebSocket protocol

| Direction | Message types |
|-----------|---------------|
| Client → Server | `execute`, `cancel`, `autocomplete`, `ping` |
| Server → Client | `meta`, `rows`, `done`, `error`, `completions`, `session`, `pong` |

### Endpoints

| Route | Description |
|-------|-------------|
| `GET /api/repl/ws` | WebSocket upgrade for interactive sessions |
| `GET /api/repl/schema` | Full schema registry as JSON |
| `POST /api/repl/session` | Create a new REPL session |

---

## UI Generation

A two-layer pipeline generates a complete Svelte + Skeleton UI + Tailwind
component library from the ontology:

**Layer 1 — `cmd/uigen`**: Reads CUE ontology + `codegen/uigen.cue` overrides
and produces framework-agnostic JSON schemas (one per entity plus a shared
enums schema). These schemas capture field types, validation rules, display
names, enum groupings, and state machine metadata.

**Layer 2 — `cmd/uirender`**: Reads the JSON schemas and generates typed
Svelte components, TypeScript interfaces, API clients, validation functions,
and reactive stores.

### Generated artifacts (~160 files)

| Directory | Contents |
|-----------|----------|
| `gen/ui/schema/` | 19 JSON schemas (18 entities + shared enums) |
| `gen/ui/types/` | TypeScript interfaces + enum types |
| `gen/ui/api/` | Typed API client per entity + base client |
| `gen/ui/validation/` | Validation functions per entity |
| `gen/ui/components/entities/` | Form, Detail, List, StatusBadge, Actions per entity |
| `gen/ui/components/shared/` | 17 shared components (MoneyInput, AddressForm, EnumSelect, etc.) |
| `gen/ui/stores/` | 5 generic stores (entity, entityList, entityMutation, stateMachine, related) |

---

## Signal Discovery System

The signal system gives AI agents the ability to see cross-cutting patterns
across entity boundaries — the kind of intuition an experienced property
manager carries but traditional software siloes into separate tables.

### How it works

`ontology/signals.cue` defines a signal taxonomy:

- **8 categories**: financial, maintenance, communication, compliance, behavioral, market, relationship, lifecycle
- **5 weights**: critical, strong, moderate, weak, info
- **4 polarities**: positive, negative, neutral, contextual
- **~30 signal registrations** mapping event types to classifications
- **Escalation rules** that fire when dangerous patterns accumulate

The runtime engine (`internal/signals/`) classifies incoming events, aggregates
them per entity, computes trends, evaluates escalation rules, and determines
overall sentiment.

### Signal API

| Endpoint | Description |
|----------|-------------|
| `GET /v1/activity/entity/{type}/{id}` | Full activity timeline for any entity — one query sees everything |
| `GET /v1/activity/summary/{type}/{id}` | Aggregated signal summary: sentiment, category breakdown, escalations |
| `POST /v1/activity/portfolio` | Portfolio-wide signal screening across all entities |
| `POST /v1/activity/search` | Free-text search across all activity |

### Agent integration

`cmd/agentgen` produces files that load into an AI agent's context:

- **ONTOLOGY.md** — the complete domain model (entity fields, relationships)
- **SIGNALS.md** — a reasoning guide teaching the agent how to interpret
  signals, recognize cross-category patterns, and understand what absence of
  signals means
- **TOOLS.md** — tool usage documentation
- **propeller-tools.json** — tool schemas the agent can call

This means the agent doesn't need property management expertise baked into its
training. The ontology teaches it what to look for and how to reason about
what it finds.

---

## Database & Migrations

The project uses [Ent](https://entgo.io) as the ORM with SQLite and
[Atlas](https://atlasgo.io) for versioned schema migrations.

```
ontology/*.cue  →  cmd/entgen  →  ent/schema/*.go  →  Atlas  →  ent/migrations/*.sql
```

Atlas reads the Ent schemas directly via the `ent://` URL scheme, diffs
against the current migration directory, and produces explicit `.sql` files.
Migrations are applied at server startup — no auto-migration, full history.

```bash
make migrate-diff      # Generate migration from schema changes
make migrate-apply     # Apply pending migrations
make migrate-status    # Show migration status
```

---

## Running

### Prerequisites

- Go 1.25+
- [CUE](https://cuelang.org/docs/introduction/installation/) v0.15+
- [Atlas](https://atlasgo.io/getting-started#installation) (official binary, not community)

### Development

```bash
make generate          # Run full CUE → code generation pipeline (12 generators)
air                    # Hot-reload server (auto-generates on CUE changes)
air -- --demo          # Same, with seeded signal demo data
```

### Server

```bash
go run ./cmd/server              # REST API on :8080
go run ./cmd/server --demo       # With seeded activity data for 6 demo tenants
```

### Signal Discovery Demo

An interactive terminal walkthrough that hits real API endpoints and shows
the signal engine classifying, aggregating, and escalating live — no mocks:

```bash
go run ./cmd/server --demo       # terminal 1
bash demo/signals.sh             # terminal 2
```

The demo seeds 6 tenants with different activity profiles and walks through
how the signal system surfaces a flight risk that traditional software misses.
Press Enter to advance through each section.

### Build Commands

```bash
cue vet ./ontology/...           # Validate CUE schemas
cue vet ./commands/... ./events/... ./api/... ./policies/...  # Validate all CUE packages
make generate                    # Full generation pipeline (12 generators)
make driftcheck                  # Cross-boundary consistency validation
go build ./...                   # Compile everything
go test ./...                    # Run tests
make ci-check                    # Verify generated code matches ontology (no drift)
```

---

## Project Structure

```
ontology/                CUE domain models (source of truth)
  common.cue             Shared value types (Money, Address, DateRange, etc.)
  base.cue               Base entity types (BaseEntity, StatefulEntity, ImmutableEntity)
  person.cue             Person, Organization, PersonRole
  property.cue           Portfolio, Property, Building, Space
  lease.cue              Lease, LeaseSpace, Application
  accounting.cue         Account, LedgerEntry, JournalEntry, BankAccount, Reconciliation
  jurisdiction.cue       Jurisdiction, PropertyJurisdiction, JurisdictionRule
  relationships.cue      Cross-entity edges (30 relationships)
  state_machines.cue     Transition maps for 13 state machines
  signals.cue            Signal taxonomy, registrations, escalation rules
  config_schema.cue      Configuration schema definitions
commands/                CQRS command definitions
  lease_commands.cue     Lease operations (MoveInTenant, etc.)
  property_commands.cue  Property mutations
  accounting_commands.cue Journal posting, payment recording
  application_commands.cue Application lifecycle
events/                  Domain event definitions
  lease_events.cue       Lease lifecycle events
  property_events.cue    Property events
  accounting_events.cue  Accounting events
  jurisdiction_events.cue Jurisdiction events
api/v1/                  API contract definitions (anti-corruption layer)
  common_api.cue         Shared API types
  lease_api.cue          Lease API responses
  property_api.cue       Property API responses
  person_api.cue         Person API responses
  accounting_api.cue     Accounting API responses
policies/                Permission groups + field-level policies
  permission_groups.cue  7 role-based permission groups
  command_permissions.cue Per-command access rules
  field_policies.cue     Field-level visibility and sensitivity
codegen/                 Generator configuration
  apigen.cue             Service mapping (6 services, ~85 operations)
  entgen.cue             Ontology → Ent type/constraint mappings
  uigen.cue              UI overrides (display names, enum groupings)
  testgen.cue            Test generation configuration
  drift.cue              Cross-boundary drift check rules
cmd/
  entgen/                CUE → Ent schema generator
  handlergen/            CUE → HTTP handler + route generator
  apigen/                CUE → protobuf generator
  eventgen/              CUE → event type generator
  authzgen/              CUE → OPA/Rego generator
  agentgen/              CUE → agent context generator
  openapigen/            CUE → OpenAPI generator
  uigen/                 CUE → JSON UI schema generator (Layer 1)
  uirender/              JSON schema → Svelte component generator (Layer 2)
  testgen/               CUE → state machine test generator
  replgen/               CUE → REPL schema registry + entity dispatcher generator
  driftcheck/            Cross-boundary CUE consistency validator
  server/                Server entrypoint
ent/
  schema/                Generated Ent schemas (18 entities)
  migrations/            Atlas versioned migrations
internal/
  event/                 Domain event recording (Recorder interface, typed constructors)
  handler/               Generated (gen_*.go) + custom (custom_*.go) HTTP handlers
  server/                Route registration and server startup
  repl/                  REPL engine
    pql/                 PQL lexer, parser, AST
    schema/              Schema registry (generated metadata for all entities)
    planner/             AST → QueryPlan with validation
    executor/            QueryPlan → Ent query execution (generated dispatchers)
    autocomplete/        Context-aware completions
    session/             Session lifecycle management
    wire/                WebSocket protocol handler
    meta/                Meta-command execution
  activity/              Activity store interface + in-memory implementation
  signals/               Signal registry, classifier, aggregator
  types/                 Go structs for CUE value types
gen/
  proto/                 Generated protobuf service definitions
  opa/                   Generated OPA/Rego policies (23 files)
  openapi/               Generated OpenAPI spec
  agent/                 Generated agent context (ONTOLOGY.md, SIGNALS.md, TOOLS.md)
  events_catalog.json    Generated event type catalog
  docs/                  Generated documentation (API reference, data dictionary, etc.)
  tests/                 Generated state machine transition tests (20 test files)
  ui/
    schema/              JSON UI schemas (19 files)
    types/               TypeScript interfaces + enums
    api/                 Typed API clients per entity
    validation/          Validation functions per entity
    components/
      entities/          Form, Detail, List, StatusBadge, Actions per entity
      shared/            17 shared components (MoneyInput, AddressForm, etc.)
    stores/              5 generic Svelte stores
specs/                   Design specifications
  repl.md                REPL and PQL specification
  uigen.md               UI generation specification
  signaldiscovery.md     Signal discovery specification
demo/
  signals.sh             Interactive signal discovery walkthrough
  lifecycle.sh           Entity lifecycle demo
```
