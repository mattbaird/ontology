# V3.1 Post-Implementation Summary

**Date:** February 27, 2026
**Scope:** Full V3.1 spec implementation across 6 phases, 161 files changed

---

## What Changed

V3.1 reframes the CUE ontology from a **contract generator** into a **shared vocabulary and constraint system**. Previously, the ontology generated everything downstream in a single pipeline. Now, six separate CUE packages independently reference ontology types via cross-package imports, and the CUE compiler catches inconsistencies at build time.

The golden rule: **the ontology defines what things ARE, commands define what you DO, the API defines what consumers SEE, events define what the system HEARS.**

### New CUE Packages

| Package | Files | Purpose |
|---------|-------|---------|
| `commands/` | 4 | 10 CQRS commands with `_affects`, `_requires_permission`, `_jurisdiction_checks` metadata |
| `events/` | 4 | 10 domain events with typed payloads (projections, not entity dumps) |
| `api/v1/` | 5 | API response contracts — anti-corruption layer that imports ontology enums but defines own shapes |
| `policies/` | 3 | 7 permission groups with inheritance, command permissions, field-level visibility |
| `codegen/` | 3 new | Configuration for entgen, testgen, and drift detection |

All packages import the ontology via:
```cue
import "github.com/matthewbaird/ontology/ontology:propeller"
```

Commands use ontology types for field validation (e.g., `propeller.#PositiveMoney`, `propeller.#Address`). Events and API contracts import ontology enums but define their own response shapes. Policies reference command names and entity fields.

### New Tools

| Tool | What it does |
|------|-------------|
| `cmd/testgen` | Reads `#StateMachines` from ontology, generates 314 state machine transition tests (108 positive, 206 negative) covering all 13 state machines |
| `cmd/driftcheck` | Validates all 6 CUE packages in one pass — catches cross-boundary inconsistencies at build time |

Both integrated into the Makefile. `testgen` runs as part of `make generate`. `driftcheck` runs via `make driftcheck`.

### Ontology Refinements

**`close()` on entity types.** Every entity definition (except base types and JournalEntry — see Gotchas) is wrapped in `close({...})`. This prevents unknown fields from being added to values of that type. Demonstrated in `demo/v31_safety.sh`.

**`strings.MinRunes(1)` replaces `!=""`.** Required string fields now use `strings.MinRunes(1)` for semantic clarity. Generators detect both patterns via updated `hasNonEmpty()`.

**Domain attributes.** `#BaseEntity` now carries `@immutable()` on `id` and `@computed()` on `audit`, documenting field semantics in the ontology itself.

**Expanded enums.** `#SpaceType` expanded from 10 to 18 values. `#OntologyRelationship` gained a `via?` field.

### Unified State Machines

Replaced 13 individual `#XTransitions` definitions with a single `#StateMachines` map:

```cue
#StateMachine: {
    [string]: [...string]  // from_state -> [valid_target_states]
}

#StateMachines: {
    lease:       #StateMachine & { draft: ["pending_approval", ...], ... }
    space:       #StateMachine & { ... }
    // ...13 total
}
```

All 5 generators that read transitions (`entgen`, `handlergen`, `uigen`, `eventgen`, `agentgen`) updated to use:
```go
val.LookupPath(cue.ParsePath("#StateMachines." + toSnake(entName)))
```

The hardcoded `entityTransitionMap` variables were removed from all generators.

### Generator Pipeline

The full pipeline now runs 11 generators (was 9):

```
make generate
```

Produces: 18 Ent schemas, 6 handler files + routes, 6 proto files, 59 event types, OPA policies for 18 entities, agent context docs, OpenAPI spec (296KB, 76 paths, 49 schemas), 18 UI schemas, 159 Svelte components, 314 state machine tests.

### JournalEntry Fix

The JournalEntry CRUD handlers (`CreateJournalEntry`, `GetJournalEntry`, `ListJournalEntries`) were not being generated, causing a build failure in `internal/server/gen_routes.go`. Root cause: `close()` + conditional blocks on `#JournalEntry` caused the CUE Go API to hide all fields, making the entity invisible to generators. Fixed by removing `close()` from `#JournalEntry` (see Gotchas).

---

## Why These Changes

### The Drift Problem

Before V3.1, nothing prevented a developer from adding a lease type to the API that didn't exist in the ontology, or creating a command that referenced an entity type nobody defined. The ontology generated downstream code, but there was no mechanism to validate that separately-authored code (commands, events, API contracts) stayed consistent with the ontology.

### The V3.1 Answer

CUE's type system becomes the enforcement mechanism. Commands import `propeller.#LeaseType` — if they try to use `"triple_super_net"`, the CUE compiler rejects it. If someone removes `"section_8"` from the ontology, every downstream package that references it must be updated or `cue vet` fails.

This is **build-time safety**, not runtime validation. The `make driftcheck` command validates the entire dependency graph in one pass.

### CQRS Separation

Commands and events are now first-class CUE definitions, not generated artifacts. This means:

- Commands declare what they affect (`_affects: ["lease", "space", "person_role"]`) and what permissions they require
- Events define their own payload shapes (projections of entity state, not full entity dumps)
- The API anti-corruption layer defines response shapes that import ontology enums but own their serialization format (dollars not cents, ISO dates, denormalized fields)

Each concern is authored independently but validated together.

### Permission Policies

Seven business-defined permission groups (`organization_admin` through `viewer`) with inheritance chains. Field-level visibility policies control who sees `person.ssn_last_four`, `lease.security_deposit`, `bank_account.routing_number`. These are CUE definitions — they validate against the ontology and can be checked for consistency.

---

## Gotchas Discovered

### `close()` + Conditional Blocks

CUE's `close()` combined with `if` conditional blocks (e.g., `if status == "posted" { ... }`) causes the Go CUE API to report an error that makes all fields inaccessible via `Fields()`. This breaks every generator that introspects entity fields.

**Affected:** `#JournalEntry` (the only entity where close() completely hid the `id` field).

**Fix:** Removed `close()` from `#JournalEntry`. Other entities with `close()` + conditionals still work because the error doesn't propagate the same way — the difference is likely JournalEntry's nested `#JournalLine` list type compounding the evaluation issue.

**Documented in:** `ontology/accounting.cue` with a NOTE comment.

### Hidden Fields Not Accessible from Go API

CUE hidden fields (`_state_machines`) are not accessible from the Go CUE API — `LookupPath` with `cue.Hid()` fails regardless of package path. The unified state machine map uses `#StateMachines` (a definition) instead.

### `close()` on Derived Types

`#NonNegativeMoney: #Money & close({amount_cents: >=0})` fails because `close()` on the derived type prevents the `currency` field inherited from `#Money`. Derived types should NOT be independently closed — they inherit closure from their parent.

### `hasNonEmpty()` Updated

Both `cmd/entgen/main.go` and `cmd/uigen/main.go` had `hasNonEmpty()` functions that only detected `!=""` (NotEqualOp). Updated to also detect `strings.MinRunes(1)` (CallOp pattern). Both patterns now produce `.NotEmpty()` in Ent schemas and `required: true` in UI schemas.

---

## Demo

```bash
bash demo/v31_safety.sh
```

A narrated walkthrough (no server needed) that demonstrates:

1. **close() rejection** — adds `favorite_color` to a `#Person` value, CUE rejects it
2. **Cross-package enum safety** — tries invalid `propeller.#LeaseType & "triple_super_net"`, CUE rejects it
3. **Unified drift check** — validates 32 CUE files across 6 packages in one pass
4. **Test generation** — 314 tests auto-derived from 13 state machines, with per-entity breakdown
5. **Full generation pipeline** — all 11 generators from one CUE ontology

---

## Verification Commands

```bash
cue vet ./ontology/... ./commands/... ./events/... ./api/... ./policies/... ./codegen/...
make generate        # 11 generators, 18 entities
make driftcheck      # Cross-boundary consistency
go build ./...       # Clean build including server
```
