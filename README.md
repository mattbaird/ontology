# Propeller Ontology

An ontology-first architecture for property management where a CUE schema is
the single source of truth and code generators derive everything downstream:
database schemas, HTTP handlers, API definitions, event catalogs, authorization
policies, agent tooling, and a signal discovery engine.

**~2,000 lines of CUE generate a fully functional REST API with 15 entities,
~70 operations, state machines, versioned migrations, and cross-cutting signal
intelligence.**

---

## Architecture

```
                          ontology/*.cue
                     (single source of truth)
                              |
        ┌─────────┬──────────┼──────────┬──────────┬──────────┐
        v         v          v          v          v          v
    cmd/entgen  cmd/handlergen  cmd/apigen  cmd/eventgen  cmd/authzgen  cmd/agentgen
        |         |          |          |          |          |
        v         v          v          v          v          v
  ent/schema/  internal/   gen/proto/  internal/  gen/opa/   gen/agent/
  (Ent ORM)   handler/    (protobuf)  worker/    (Rego)     ONTOLOGY.md
              server/                 events.go             SIGNALS.md
              (HTTP)                                        TOOLS.md
                                                            propeller-tools.json
```

One `make generate` command runs the full pipeline. Change a field in CUE and
the database schema, HTTP handlers, proto definitions, event types,
authorization policies, and agent context documents all update automatically.

### Why CUE

CUE provides types, constraints, defaults, and validation in a single
declarative language. Unlike JSON Schema or protobuf, CUE schemas are
composable — entity definitions, value types, enums, state machine
transitions, and signal registrations all live in the same module and
cross-reference each other. The generators read the CUE evaluation result
directly, so there's no intermediate representation to keep in sync.

---

## Domain Model

15 entities across 4 domains, all defined in `ontology/*.cue`:

| Domain | Entities |
|--------|----------|
| **Person** | Person, Organization, PersonRole |
| **Property** | Portfolio, Property, Building, Space |
| **Lease** | Lease, LeaseSpace, Application |
| **Accounting** | Account, LedgerEntry, JournalEntry, BankAccount, Reconciliation |

Every entity with a `status` field has a state machine defined in
`ontology/state_machines.cue`. Transitions are enforced at the persistence
layer via generated Ent hooks — no code path can violate them.

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

Custom operations (journal posting, payment recording, application approval,
etc.) are marked `custom: true` in `codegen/apigen.cue` and hand-written in
`internal/handler/custom_*.go`. The generators skip these — they coexist with
generated code without conflict.

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

`cmd/agentgen` produces three files that load into an AI agent's context:

- **ONTOLOGY.md** — the complete domain model (entity fields, relationships)
- **SIGNALS.md** — a reasoning guide teaching the agent how to interpret
  signals, recognize cross-category patterns, and understand what absence of
  signals means
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
make generate          # Run full CUE → code generation pipeline
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
make generate                    # Full generation pipeline
go build ./...                   # Compile everything
go test ./...                    # Run tests
make ci-check                    # Verify generated code matches ontology (no drift)
```

---

## Project Structure

```
ontology/                CUE domain models (source of truth)
  common.cue             Shared value types (Money, Address, DateRange, etc.)
  person.cue             Person, Organization, PersonRole
  property.cue           Portfolio, Property, Building, Space
  lease.cue              Lease, LeaseSpace, Application
  accounting.cue         Account, LedgerEntry, JournalEntry, BankAccount, Reconciliation
  relationships.cue      Cross-entity edges
  state_machines.cue     Transition maps for every status enum
  signals.cue            Signal taxonomy, registrations, escalation rules
codegen/
  apigen.cue             Service mapping (4 services, ~70 operations)
cmd/
  entgen/                CUE → Ent schema generator
  handlergen/            CUE → HTTP handler + route generator
  apigen/                CUE → protobuf generator
  eventgen/              CUE → event type generator
  authzgen/              CUE → OPA/Rego generator
  agentgen/              CUE → agent context generator
  openapigen/            CUE → OpenAPI generator
  server/                Server entrypoint
ent/
  schema/                Generated Ent schemas
  migrations/            Atlas versioned migrations
internal/
  handler/               Generated (gen_*.go) + custom (custom_*.go) HTTP handlers
  server/                Route registration and server startup
  activity/              Activity store interface + in-memory implementation
  signals/               Signal registry, classifier, aggregator
  types/                 Go structs for CUE value types
gen/
  proto/                 Generated protobuf service definitions
  opa/                   Generated OPA/Rego policies
  openapi/               Generated OpenAPI spec
  agent/                 Generated agent context documents
  events_catalog.json    Generated event type catalog
```
