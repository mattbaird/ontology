# Propeller REPL Specification

**Version**: 1.0  
**Status**: Draft  
**Last Updated**: 2026-02-27  
**Stack**: Svelte + Skeleton UI + Tailwind CSS (frontend) · Go + Ent (backend)

---

## 1. Purpose

The Propeller REPL is a web-based interactive console that provides script-based and direct access to the Ent entity layer. It is the primary tool for querying, mutating, exploring, and debugging the Propeller domain model at runtime.

Two user modes serve different audiences:

- **Dev Mode** — full entity layer access, raw queries, schema introspection, script execution, state machine debugging. For engineers building and maintaining Propeller.
- **Operator Mode** — curated query templates, natural-language-ish entity lookups, read-heavy workflows with guarded writes. For property managers, support staff, and power users who need direct data access without writing code.

The REPL is not a replacement for the Propeller UI. It is a power-user tool that exposes the entity graph directly, the way `psql` exposes Postgres or Rails console exposes ActiveRecord — but through a modern browser interface with autocomplete, output rendering, and session management.

---

## 2. Architecture Overview

```
┌─────────────────────────────────────────────────────┐
│                  Browser (Svelte)                     │
│  ┌──────────┐  ┌──────────┐  ┌────────────────────┐ │
│  │  Editor   │  │  Output  │  │  Inspector Panel   │ │
│  │  Panel    │  │  Panel   │  │  (schema/graph/    │ │
│  │          │  │          │  │   state machines)   │ │
│  └────┬─────┘  └────▲─────┘  └────────────────────┘ │
│       │              │                                │
│       ▼              │                                │
│  ┌────────────────────────┐                          │
│  │   REPL Client Runtime  │                          │
│  │   - parser / tokenizer │                          │
│  │   - autocomplete       │                          │
│  │   - session state      │                          │
│  │   - output formatter   │                          │
│  └────────┬───────────────┘                          │
└───────────┼──────────────────────────────────────────┘
            │  WebSocket (persistent)
            │  + HTTP/2 (file uploads, large results)
            ▼
┌───────────────────────────────────────────────────────┐
│                REPL Backend (Go)                       │
│  ┌──────────────┐  ┌────────────┐  ┌───────────────┐ │
│  │  Session Mgr  │  │  Executor  │  │  Sandbox      │ │
│  │  - auth       │  │  - parse   │  │  - tx scope   │ │
│  │  - mode       │  │  - plan    │  │  - read-only  │ │
│  │  - history    │  │  - execute │  │  - dry-run    │ │
│  └──────────────┘  └─────┬──────┘  └───────────────┘ │
│                          │                             │
│                          ▼                             │
│  ┌─────────────────────────────────────────────────┐  │
│  │                 Ent Client                       │  │
│  │  - entity queries    - edge traversal            │  │
│  │  - mutations         - state transitions         │  │
│  │  - aggregations      - schema introspection      │  │
│  └─────────────────────────┬───────────────────────┘  │
│                            │                           │
│                            ▼                           │
│                     ┌──────────────┐                   │
│                     │   Postgres   │                   │
│                     └──────────────┘                   │
└───────────────────────────────────────────────────────┘
```

### 2.1 Key Design Decisions

1. **Not a general-purpose scripting language.** The REPL defines a domain-specific query/command syntax purpose-built for Ent entity access. It is not embedded JavaScript, Python, or Lua. It is closer to GraphQL meets `jq` meets a property-management-aware shell.

2. **WebSocket for interactivity, HTTP for bulk.** The editor session uses a persistent WebSocket for keystroke-level autocomplete and streaming results. Large exports and file operations use standard HTTP/2 endpoints.

3. **Every mutation goes through commands.** The REPL does not allow raw `UPDATE` against entities. All writes flow through the command layer defined in the ontology spec. This ensures state machine enforcement, event emission, and permission checks are never bypassed.

4. **Transaction sandbox by default.** Dev mode mutations run inside a database transaction that is held open until the user explicitly commits or rolls back. This allows exploratory writes without consequences.

5. **Operator mode is a restricted view, not a separate system.** Both modes hit the same backend. Operator mode applies allowlists (which entities, which fields, which commands) enforced server-side. The UI adapts presentation but the security boundary is backend.

---

## 3. Query Language — `PQL` (Propeller Query Language)

PQL is the REPL's input language. It is designed to be readable by non-engineers while remaining precise enough for complex entity traversal. PQL compiles to Ent query builder calls on the backend.

### 3.1 Core Syntax

```
<verb> <entity_type> [<filters>] [<projections>] [<modifiers>]
```

**Verbs** map directly to operations:

| Verb | Operation | Mode |
|------|-----------|------|
| `find` | Query entities, return list | Both |
| `get` | Query single entity by ID | Both |
| `count` | Count matching entities | Both |
| `describe` | Show entity schema and state machine | Dev |
| `explain` | Show query plan (Ent → SQL) | Dev |
| `run` | Execute a command (write) | Both (guarded in operator) |
| `history` | Show entity audit trail | Both |
| `traverse` | Walk entity graph from a starting node | Both |
| `aggregate` | Group-by with aggregation functions | Dev |
| `diff` | Compare entity state across two points in time | Dev |
| `watch` | Subscribe to entity changes (live tail) | Dev |

### 3.2 Querying Entities

```pql
-- Find all active leases for a property
find lease where property_id = "prop_123" and status = "active"

-- Project specific fields
find lease where status = "active" select id, start_date, monthly_rent

-- Nested field access
find person where contact_methods.type = "email" select name, contact_methods

-- Pagination
find lease where status = "active" limit 25 offset 50

-- Sorting
find lease where property_id = "prop_123" order by start_date desc

-- Get single entity
get lease "lease_abc123"

-- Count
count space where property_id = "prop_456" and status = "vacant"
```

### 3.3 Edge Traversal

PQL uses dot-path syntax to traverse Ent edges. Edge names match the relationship definitions in `ontology/relationships.cue`.

```pql
-- Traverse: lease → spaces
find lease where id = "lease_123" include spaces

-- Deep traversal: lease → spaces → property
find lease where status = "active" include spaces.property

-- Reverse traversal: person → roles → lease
find person where id = "person_456" include roles.lease

-- Filter on related entities
find lease where spaces.property.address.state = "CA" and status = "active"

-- Multi-edge include
find property where id = "prop_789" include spaces, buildings, jurisdiction_rules
```

### 3.4 Aggregations (Dev Mode)

```pql
-- Total monthly rent by property
aggregate lease
  where status = "active"
  group by property_id
  select property_id, sum(monthly_rent) as total_rent, count() as lease_count

-- Vacancy rate by portfolio
aggregate space
  group by property.portfolio_id
  select property.portfolio_id,
    count(status = "vacant") as vacant,
    count() as total,
    ratio(status = "vacant") as vacancy_rate
```

### 3.5 Executing Commands

All writes flow through the command layer. The REPL `run` verb maps to command execution.

```pql
-- Execute a command
run MoveInTenant {
  lease_id: "lease_123",
  person_id: "person_456",
  space_id: "space_789",
  move_in_date: "2026-03-01",
  deposit_amount: { amount: 250000, currency: "USD" }
}

-- Dry-run (validates without executing)
run MoveInTenant { ... } --dry-run

-- Show what a command will touch before running
explain run MoveInTenant { ... }
```

In **Operator mode**, `run` requires confirmation and only a curated subset of commands is available. The backend enforces this — the UI simply doesn't autocomplete unavailable commands.

### 3.6 Schema Introspection (Dev Mode)

```pql
-- Describe an entity's fields, edges, and constraints
describe lease

-- Show state machine for an entity
describe lease states

-- Show all commands that touch an entity
describe lease commands

-- Show all events emitted by an entity
describe lease events

-- List all entity types
describe entities

-- Show relationships between two entities
describe edges lease person
```

### 3.7 History and Diff (Dev Mode)

```pql
-- Audit trail for an entity
history lease "lease_123"

-- Filtered history
history lease "lease_123" since "2026-01-01" actions [create, update]

-- Compare entity state at two points
diff lease "lease_123" at "2026-01-01" vs "2026-02-01"

-- Diff against current
diff lease "lease_123" at "2026-01-01" vs now
```

### 3.8 Live Watching (Dev Mode)

```pql
-- Subscribe to all changes on an entity
watch lease "lease_123"

-- Watch all events for a property
watch events where property_id = "prop_456"

-- Watch state transitions
watch transitions lease where property_id = "prop_456"
```

Watch streams results to the output panel in real time via the WebSocket connection. `Ctrl+C` or clicking the stop button ends the subscription.

### 3.9 Scripting

PQL supports multi-statement scripts with variables, pipes, and control flow for complex operations.

```pql
-- Variable binding
let active_leases = find lease where property_id = "prop_123" and status = "active"

-- Pipe results to next operation
find lease where status = "active" and end_date < "2026-04-01"
  | select id, end_date
  | order by end_date asc

-- Iterate over results
let expiring = find lease where end_date < "2026-04-01" and status = "active"
for lease in expiring {
  run SendRenewalNotice {
    lease_id: lease.id,
    notice_type: "60_day"
  }
}

-- Conditional logic
let lease = get lease "lease_123"
if lease.status == "active" and lease.end_date < "2026-06-01" {
  run InitiateRenewal { lease_id: lease.id, proposed_term_months: 12 }
}

-- Functions (dev mode only)
fn vacancy_report(portfolio_id) {
  let props = find property where portfolio_id = portfolio_id include spaces
  for prop in props {
    let vacant = count space where property_id = prop.id and status = "vacant"
    let total = count space where property_id = prop.id
    emit { property: prop.name, vacant: vacant, total: total, rate: vacant / total }
  }
}

vacancy_report("port_001")
```

### 3.10 Script Files

Scripts can be saved, loaded, and shared.

```pql
-- Save current script
:save "monthly_vacancy_report"

-- Load and run a saved script
:load "monthly_vacancy_report"

-- List saved scripts
:scripts

-- Export script to file
:export "monthly_vacancy_report" --format pql
```

---

## 4. Frontend — Browser UI

### 4.1 Layout

The REPL UI is a three-panel layout built with Svelte + Skeleton UI + Tailwind CSS.

```
┌──────────────────────────────────────────────────────────────┐
│  [Mode: Dev ▾]  [Session: #a1b2c3]  [Tx: open ●]  [⚙ ☰]   │
├─────────────────────────┬────────────────────────────────────┤
│                         │                                    │
│    EDITOR PANEL         │         OUTPUT PANEL               │
│                         │                                    │
│  Line-numbered editor   │  Results table / tree / JSON       │
│  with syntax            │  Entity cards                      │
│  highlighting,          │  Streaming output                  │
│  autocomplete,          │  Error messages                    │
│  multi-line scripts     │  Charts (aggregations)             │
│                         │  Diff views                        │
│                         │                                    │
├─────────────────────────┴────────────────────────────────────┤
│                    INSPECTOR PANEL (collapsible)              │
│                                                              │
│  [Schema] [Graph] [State Machine] [History] [Saved Scripts]  │
│                                                              │
│  Context-sensitive: shows relevant info based on what's      │
│  selected in editor or output.                               │
└──────────────────────────────────────────────────────────────┘
```

### 4.2 Editor Panel

The editor panel is the primary input surface. It functions as both a single-line REPL prompt and a multi-line script editor.

**Capabilities:**

- **Syntax highlighting** for PQL keywords, entity types, field names, string literals, numbers, operators.
- **Autocomplete** with four providers:
  - **Entity types** — triggered after verbs (`find `, `get `, `describe `). Source: schema registry.
  - **Field names** — triggered after entity type or `where`/`select` clauses. Source: entity schema for the current type in context.
  - **Edge names** — triggered after `.` in traversal paths. Source: relationship definitions.
  - **Command names** — triggered after `run`. Source: command registry. In operator mode, only the allowed subset appears.
  - **Saved scripts** — triggered after `:load`.
- **Multi-cursor editing** for batch operations.
- **Inline error markers** — syntax errors underlined in red with hover tooltip before execution.
- **Execution**: `Cmd+Enter` to execute current statement or selected block. `Shift+Enter` for newline without executing.
- **History navigation**: `Up/Down` arrows cycle through previous commands (per-session and cross-session).
- **Script mode toggle**: switch between single-line prompt and full editor with line numbers, folding, and a run button.

### 4.3 Output Panel

All results render in the output panel. The panel supports multiple output formats and auto-selects the best format based on result type.

| Result Type | Default Rendering | Alt Renderings |
|---|---|---|
| Entity list | Paginated table with sortable columns | JSON, entity cards, CSV export |
| Single entity | Entity card with all fields + edges | JSON, raw Ent struct |
| Count / scalar | Large number display | — |
| Aggregation | Table + auto-chart (bar/line) | JSON, CSV export |
| Command result | Success/failure banner + affected entities | JSON, event log |
| Schema (describe) | Formatted type definition with constraints | CUE source, Ent schema |
| State machine | Visual state diagram (SVG) | Transition table |
| Diff | Side-by-side diff with changed fields highlighted | Unified diff |
| History | Timeline with expandable entries | Table |
| Watch stream | Scrolling log with entity cards | JSON stream |
| Error | Red banner with error details + suggested fix | Stack trace (dev) |

**Output features:**

- **Click-to-inspect**: Click any entity ID in the output to populate the inspector panel with that entity's details.
- **Pin results**: Pin an output block so it persists as you run new queries. Useful for comparing results.
- **Copy/Export**: Every output block has copy-as-JSON and export-as-CSV buttons.
- **Result pagination**: Large result sets paginate with "load more" rather than loading everything.
- **Streaming**: Long-running queries and `watch` streams render incrementally.

### 4.4 Inspector Panel

The inspector panel is a collapsible bottom panel with tabbed views. It provides contextual, read-only reference information.

**Tabs:**

| Tab | Content | Trigger |
|---|---|---|
| **Schema** | Entity field definitions, types, constraints, CUE source | Clicking entity type in editor/output |
| **Graph** | Visual entity relationship graph (interactive, d3-based) | Navigating to an entity, clicking "show graph" |
| **State Machine** | Visual state diagram for current entity type | Selecting a stateful entity type |
| **History** | Audit trail for currently inspected entity | Clicking entity ID in output |
| **Saved Scripts** | List of saved scripts with preview, search, and run | `:scripts` command or tab click |

The inspector panel updates automatically based on cursor position in the editor and selection in the output. For example, placing the cursor on `find lease` highlights the Lease schema tab. Clicking a specific lease ID in the output loads that entity's history.

### 4.5 Toolbar

The top toolbar shows session state and mode controls:

- **Mode toggle**: Switch between Dev and Operator mode. Switching modes clears the command allowlist cache and reloads autocomplete providers.
- **Session indicator**: Shows current session ID. Sessions persist across browser refreshes for 24 hours.
- **Transaction indicator** (Dev mode only): Shows transaction state — `idle` (no open tx), `open` (uncommitted changes), `dirty` (changes pending). Click to commit/rollback.
- **Settings gear**: Theme (light/dark/system), font size, key bindings (default/vim/emacs), result format defaults.
- **Hamburger menu**: Session history, saved scripts, keyboard shortcuts reference, documentation link.

---

## 5. Operator Mode

Operator mode is a restricted, friendlier view of the same system. The restrictions are enforced on the backend — the frontend adapts presentation.

### 5.1 Restrictions

| Capability | Dev Mode | Operator Mode |
|---|---|---|
| `find` / `get` / `count` | All entities, all fields | Allowlisted entities and fields |
| `run` (commands) | All commands, dry-run optional | Allowlisted commands, confirmation required |
| `describe` | Full schema, CUE source | Simplified field descriptions |
| `explain` | Query plan | Not available |
| `aggregate` | Unrestricted | Not available |
| `diff` | Unrestricted | Not available |
| `watch` | Unrestricted | Not available |
| `history` | Full audit trail | Own actions only |
| `traverse` | Unlimited depth | Max 2 levels |
| Scripting | Full (variables, loops, functions) | Variables and pipes only (no loops, no functions) |
| Transaction sandbox | Available | Not available — all commands execute immediately |
| Raw JSON output | Available | Hidden by default, available via toggle |
| Field-level access | All fields including internal | PII fields masked, internal fields hidden |

### 5.2 Operator Allowlist

The operator allowlist is defined per-role using the permission policies from `policies/*.cue`. The REPL backend loads the allowlist for the authenticated user's role at session start.

```cue
// policies/repl_operator.cue

#OperatorAccess: {
    "property_manager": {
        entities: ["lease", "space", "property", "person", "work_order", "application"]
        commands: ["CreateWorkOrder", "AssignVendor", "ApproveApplication", "SendNotice"]
        max_results: 500
        traverse_depth: 2
    }
    "accountant": {
        entities: ["lease", "ledger_entry", "journal_entry", "bank_account", "account"]
        commands: ["PostJournalEntry", "ReconcileTransaction", "GenerateStatement"]
        max_results: 1000
        traverse_depth: 2
    }
    "leasing_agent": {
        entities: ["lease", "application", "person", "space", "property"]
        commands: ["CreateApplication", "ScreenApplicant", "ApproveLease", "SendOffer"]
        max_results: 200
        traverse_depth: 1
    }
}
```

### 5.3 Operator UX Adaptations

- **Query templates**: The editor shows a template picker above the input. Templates are parameterized queries like "Find all vacant spaces in [property]" or "Show expiring leases in next [N] days". Clicking a template populates the editor with the PQL, with fillable placeholders highlighted.
- **Field labels**: Instead of raw field names (`monthly_rent`), operator mode shows human labels ("Monthly Rent") in output tables. Labels come from `uigen.cue` field metadata.
- **Confirmation dialogs**: Every `run` command in operator mode shows a confirmation dialog listing what the command will do, which entities it will affect, and requires an explicit "Execute" click.
- **Simplified errors**: Error messages in operator mode omit stack traces and Ent internals. They show plain-language descriptions with suggested fixes.

---

## 6. Backend

### 6.1 Session Management

Each REPL session maps to a WebSocket connection with the following state:

```go
type Session struct {
    ID            string
    UserID        string
    Mode          Mode          // Dev | Operator
    Permissions   *Allowlist    // loaded from policies at session start
    Tx            *ent.Tx       // open transaction (dev mode only, nil if idle)
    TxStarted     time.Time     // for timeout enforcement
    History       []HistoryEntry
    Variables     map[string]any // PQL variable bindings
    WatchSubs     []Subscription
    CreatedAt     time.Time
    LastActiveAt  time.Time
}
```

**Session lifecycle:**

1. **Open** — user loads REPL page, authenticates, WebSocket connects. Backend creates session, loads allowlist.
2. **Active** — user sends PQL statements over WebSocket. Backend parses, plans, executes, streams results back.
3. **Idle timeout** — 30 minutes of no activity. Backend sends warning at 25 minutes. At 30, open transactions are rolled back, watch subscriptions are cancelled, session is suspended but preserved.
4. **Resume** — reconnecting within 24 hours restores session (history, variables). Open transactions are NOT restored.
5. **Close** — explicit close or 24-hour expiry. All state is cleaned up. History is persisted to the user's session archive.

### 6.2 PQL Parser and Executor

The PQL parser is a hand-written recursive descent parser in Go. It produces an AST that the executor converts to Ent query builder calls.

```
PQL Statement
    │
    ▼
  Parser (PQL text → AST)
    │
    ▼
  Planner (AST → Query Plan)
    │  - resolves entity types against schema registry
    │  - validates field names and types
    │  - checks permissions (operator allowlist)
    │  - applies traversal depth limits
    │  - estimates result size
    │
    ▼
  Executor (Query Plan → Ent calls → Results)
    │  - builds Ent predicates from filters
    │  - executes within session transaction (if open)
    │  - enforces result limits
    │  - streams large results
    │
    ▼
  Formatter (Results → Wire format)
       - serializes to JSON
       - applies field masking (operator mode)
       - applies PII redaction
```

### 6.3 PQL-to-Ent Mapping

PQL compiles to Ent query builder calls. The mapping is direct:

| PQL | Ent Query Builder |
|---|---|
| `find lease` | `client.Lease.Query()` |
| `where status = "active"` | `.Where(lease.StatusEQ("active"))` |
| `where monthly_rent > 2000` | `.Where(lease.MonthlyRentGT(2000))` |
| `include spaces` | `.WithSpaces()` |
| `include spaces.property` | `.WithSpaces(func(q *ent.SpaceQuery) { q.WithProperty() })` |
| `select id, start_date` | `.Select(lease.FieldID, lease.FieldStartDate)` |
| `order by start_date desc` | `.Order(ent.Desc(lease.FieldStartDate))` |
| `limit 25 offset 50` | `.Limit(25).Offset(50)` |
| `count space where ...` | `client.Space.Query().Where(...).Count(ctx)` |

### 6.4 Command Execution

`run` statements do NOT compile to Ent mutations directly. They dispatch to the command handler layer, which performs validation, state machine checks, event emission, and then calls Ent.

```go
func (e *Executor) RunCommand(ctx context.Context, sess *Session, cmd *CommandAST) (*CommandResult, error) {
    // 1. Resolve command handler from registry
    handler, ok := e.commandRegistry.Get(cmd.Name)
    if !ok {
        return nil, ErrUnknownCommand(cmd.Name)
    }

    // 2. Check operator allowlist
    if sess.Mode == ModeOperator {
        if !sess.Permissions.AllowsCommand(cmd.Name) {
            return nil, ErrCommandNotAllowed(cmd.Name)
        }
    }

    // 3. Build command payload from PQL arguments
    payload, err := cmd.BuildPayload(handler.Schema())
    if err != nil {
        return nil, err
    }

    // 4. Validate payload against CUE schema
    if err := handler.Validate(payload); err != nil {
        return nil, ErrValidationFailed(err)
    }

    // 5. Dry-run check
    if cmd.DryRun {
        return handler.DryRun(ctx, payload)
    }

    // 6. Execute within session transaction (dev) or new transaction (operator)
    var tx *ent.Tx
    if sess.Tx != nil {
        tx = sess.Tx
    } else {
        tx, err = e.client.Tx(ctx)
        if err != nil {
            return nil, err
        }
        defer tx.Commit() // operator mode: auto-commit
    }

    result, err := handler.Execute(ctx, tx, payload)
    if err != nil {
        if sess.Tx == nil {
            tx.Rollback()
        }
        return nil, err
    }

    return result, nil
}
```

### 6.5 Transaction Sandbox (Dev Mode)

In Dev mode, users can open an explicit transaction that wraps multiple operations. This allows exploratory writes that can be rolled back.

```pql
-- Open a transaction
:begin

-- Run some mutations
run MoveInTenant { ... }
run PostJournalEntry { ... }

-- Inspect results within the transaction
find lease where id = "lease_123"

-- Decide: commit or rollback
:commit    -- persist changes
:rollback  -- discard everything since :begin
```

**Safety constraints:**

- Max transaction duration: **10 minutes**. After 10 minutes, the backend automatically rolls back and notifies the user. This prevents long-held locks.
- Max one open transaction per session.
- `watch` subscriptions see uncommitted changes within the same session but other sessions do not.
- Transaction state is shown in the toolbar: green dot (committed), yellow dot (open/dirty), no dot (idle).

### 6.6 Autocomplete Backend

The autocomplete system has a dedicated endpoint that responds within **50ms** for keystroke-level responsiveness.

```go
type AutocompleteRequest struct {
    Text     string    // full editor content
    Cursor   int       // cursor position
    Mode     Mode      // Dev | Operator
    Session  string    // for variable completions
}

type AutocompleteResponse struct {
    Completions []Completion
    // each completion includes:
    // - label (display text)
    // - insert_text (what gets inserted)
    // - kind (entity, field, edge, command, keyword, variable, script)
    // - detail (type info, description)
    // - documentation (longer help text, shown in tooltip)
}
```

The autocomplete engine does contextual parsing of the partial PQL statement to determine which provider to activate:

1. **Start of statement** → verbs (`find`, `get`, `run`, `count`, `describe`, ...)
2. **After verb** → entity types (filtered by operator allowlist in operator mode)
3. **After `where`** → field names for current entity type
4. **After field name** → operators (`=`, `!=`, `>`, `<`, `>=`, `<=`, `like`, `in`)
5. **After `.`** → edge names for current entity type
6. **After `run`** → command names (filtered by allowlist)
7. **After `run CommandName {`** → command payload fields
8. **After `:load`** → saved script names
9. **Anywhere** → session variables (prefixed with `let`)

### 6.7 Result Streaming

Large result sets are streamed over the WebSocket connection to avoid memory pressure and enable progressive rendering.

```
Client                          Server
  │                               │
  │── PQL: find lease limit 500 ──▶│
  │                               │── query Ent
  │                               │
  │◀── { type: "meta",           │
  │      total: 347,              │
  │      schema: [...] }          │
  │                               │
  │◀── { type: "rows",           │
  │      data: [row1..row50] }    │  ← batch 1
  │                               │
  │◀── { type: "rows",           │
  │      data: [row51..row100] }  │  ← batch 2
  │                               │
  │         ...                   │
  │                               │
  │◀── { type: "done" }          │
```

Batch size is adaptive: starts at 50 rows, scales down if rows are wide (many fields, nested edges). The frontend renders each batch as it arrives.

---

## 7. Permissions and Security

### 7.1 Authentication

The REPL uses the same authentication as the Propeller UI. Session tokens carry the user's identity and role.

### 7.2 Authorization Layers

```
Request arrives
    │
    ▼
Layer 1: Mode check
    │  - Is this verb available in the user's mode?
    │  - e.g., `aggregate` blocked in operator mode
    │
    ▼
Layer 2: Entity allowlist (operator mode)
    │  - Is this entity type in the user's allowlist?
    │  - e.g., `find journal_entry` blocked for leasing_agent
    │
    ▼
Layer 3: Field allowlist (operator mode)
    │  - Are all requested fields accessible?
    │  - e.g., SSN field blocked, internal audit fields hidden
    │
    ▼
Layer 4: Command allowlist (operator mode)
    │  - Is this command in the user's allowlist?
    │
    ▼
Layer 5: Standard Propeller RBAC
    │  - Does this user have permission to access this
    │    specific data? (portfolio-scoped, property-scoped, etc.)
    │
    ▼
Layer 6: PII masking
       - Mask sensitive fields in output regardless of allowlist
       - SSN → "***-**-1234", bank account → "****5678"
       - Dev mode: full access with audit log
```

### 7.3 Audit Trail

Every REPL interaction is logged:

```go
type AuditEntry struct {
    Timestamp   time.Time
    SessionID   string
    UserID      string
    Mode        Mode
    Statement   string      // the PQL statement
    QueryPlan   string      // what the planner produced
    EntitiesRead  []string  // entity IDs read
    EntitiesMutated []string // entity IDs written
    Duration    time.Duration
    Error       string      // empty if success
}
```

All mutations are logged with full before/after snapshots. In operator mode, even reads are logged if they access PII-flagged fields.

### 7.4 Rate Limiting

| Operation | Dev Mode | Operator Mode |
|---|---|---|
| Queries per minute | 120 | 60 |
| Mutations per minute | 30 | 10 |
| Max result set | 10,000 rows | Per-role (200–1000) |
| Max traversal depth | Unlimited | 2 |
| Max concurrent watch subs | 10 | 0 |
| Max script length | 500 statements | 20 statements |

---

## 8. Built-in Commands (REPL Meta-Commands)

Meta-commands are prefixed with `:` and control the REPL itself rather than querying the entity layer.

| Command | Description | Mode |
|---|---|---|
| `:begin` | Open a transaction sandbox | Dev |
| `:commit` | Commit open transaction | Dev |
| `:rollback` | Rollback open transaction | Dev |
| `:mode dev` / `:mode operator` | Switch modes | Both (requires auth) |
| `:save "name"` | Save current editor content as named script | Both |
| `:load "name"` | Load saved script into editor | Both |
| `:scripts` | List saved scripts | Both |
| `:delete "name"` | Delete saved script | Both |
| `:export --format csv\|json` | Export last result to downloadable file | Both |
| `:history` | Show command history for current session | Both |
| `:history --all` | Show command history across sessions | Both |
| `:clear` | Clear output panel | Both |
| `:env` | Show session info (user, mode, tx state, vars) | Both |
| `:vars` | List all bound variables | Both |
| `:set <key> <value>` | Set session config (result_limit, format, etc.) | Both |
| `:help` | Show help / keyboard shortcuts | Both |
| `:help <verb>` | Show detailed help for a verb | Both |

---

## 9. Keyboard Shortcuts

| Shortcut | Action |
|---|---|
| `Cmd+Enter` | Execute current statement / selected block |
| `Shift+Enter` | Insert newline (no execute) |
| `Up` / `Down` | History navigation (single-line mode) |
| `Cmd+L` | Clear output |
| `Cmd+S` | Save current script |
| `Cmd+P` | Open script picker (fuzzy search) |
| `Cmd+K` | Open command palette |
| `Cmd+/` | Toggle comment on selected lines |
| `Cmd+D` | Describe entity type at cursor |
| `Cmd+.` | Show autocomplete |
| `Escape` | Cancel running query / close panels |
| `Cmd+Shift+T` | Toggle transaction sandbox (:begin / :commit) |
| `Cmd+Shift+I` | Toggle inspector panel |

---

## 10. Error Handling

### 10.1 Error Categories

| Category | Example | Presentation |
|---|---|---|
| **Syntax** | `find lease whre status = "active"` (typo) | Red underline in editor + error in output with "Did you mean `where`?" |
| **Schema** | `find lease where nonexistent_field = "x"` | Error with list of valid fields for that entity |
| **Permission** | Operator trying `aggregate` | "This operation is not available in Operator mode" |
| **Validation** | `run MoveInTenant { deposit_amount: -100 }` | Command validation error with field-level details |
| **State machine** | Trying to move in a tenant on a terminated lease | "Cannot transition lease from `terminated` to `active`. Valid transitions: none." |
| **Runtime** | Database timeout, connection error | Retry suggestion + option to re-run |
| **Limit** | Result set exceeds max | "Query returned 15,000 rows (limit: 10,000). Add filters or use `limit`." |

### 10.2 Suggested Fixes

The error system includes a suggestion engine. Every error type maps to zero or more fix suggestions that are displayed as clickable actions in the output panel.

```
ERROR: Unknown entity type "lese"

  Did you mean:
    → lease    (click to fix)
    → lease_space    (click to fix)
```

```
ERROR: Field "rent" not found on entity "lease"

  Available fields matching "rent":
    → monthly_rent (Money)
    → security_deposit (Money)
    → rent_escalation_schedule ([]EscalationStep)
```

---

## 11. Saved Scripts and Templates

### 11.1 Script Storage

Saved scripts are stored per-user with optional sharing.

```go
type SavedScript struct {
    ID          string
    Name        string
    UserID      string
    SharedWith  []string    // user IDs or "public" or role names
    Content     string      // PQL source
    Description string
    Tags        []string
    Parameters  []Parameter // named parameters for template scripts
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

type Parameter struct {
    Name        string
    Type        string    // string, number, date, entity_id
    Required    bool
    Default     any
    Description string
}
```

### 11.2 Built-in Templates (Operator Mode)

The system ships with built-in templates for common property management tasks:

| Template | Parameters | PQL |
|---|---|---|
| Vacant spaces | `property_id` | `find space where property_id = $property_id and status = "vacant" select name, type, square_footage` |
| Expiring leases | `days_ahead`, `portfolio_id` | `find lease where end_date < today() + $days_ahead and status = "active" include spaces.property select ...` |
| Rent roll | `property_id` | `find lease where property_id = $property_id and status = "active" include person, spaces select person.name, spaces.name, monthly_rent, start_date, end_date` |
| Overdue balances | `property_id`, `min_amount` | `find ledger_entry where property_id = $property_id and balance > $min_amount and due_date < today() include person` |
| Work order status | `property_id`, `status` | `find work_order where property_id = $property_id and status = $status order by created_at desc` |
| Lease timeline | `lease_id` | `history lease $lease_id` |
| Application pipeline | `property_id` | `find application where property_id = $property_id order by status, created_at` |

Templates appear in the editor panel's template picker. Clicking one populates the editor with fillable parameter fields highlighted.

---

## 12. Wire Protocol

### 12.1 WebSocket Messages

Client → Server:

```typescript
type ClientMessage =
  | { type: "execute"; id: string; pql: string }
  | { type: "cancel"; id: string }
  | { type: "autocomplete"; text: string; cursor: number }
  | { type: "meta"; command: string; args: Record<string, any> }
  | { type: "ping" }
```

Server → Client:

```typescript
type ServerMessage =
  | { type: "meta"; id: string; total?: number; schema?: FieldDef[] }
  | { type: "rows"; id: string; data: any[] }
  | { type: "done"; id: string; duration_ms: number }
  | { type: "error"; id: string; error: ErrorPayload }
  | { type: "completions"; items: Completion[] }
  | { type: "watch_event"; subscription: string; data: any }
  | { type: "tx_state"; state: "idle" | "open" | "dirty" }
  | { type: "session"; session_id: string; mode: string; user: string }
  | { type: "pong" }
```

### 12.2 HTTP Endpoints

| Endpoint | Method | Purpose |
|---|---|---|
| `/api/repl/session` | POST | Create session (returns WebSocket URL) |
| `/api/repl/export` | POST | Export result as CSV/JSON file download |
| `/api/repl/scripts` | GET/POST/PUT/DELETE | CRUD for saved scripts |
| `/api/repl/templates` | GET | List available templates for current user role |
| `/api/repl/schema` | GET | Full schema registry (for offline autocomplete) |

---

## 13. Implementation Phases

### Phase 1: Foundation
- PQL parser (find, get, count with where/select/order/limit)
- Ent query builder mapping for core entity types
- WebSocket session management
- Editor panel with syntax highlighting
- Output panel with table rendering
- Basic autocomplete (entity types, fields)
- Environment indicator (local/staging/production badge)

### Phase 2: Commands and Dev Tools
- Command execution via `run`
- Transaction sandbox (:begin/:commit/:rollback)
- `describe` verb (schema introspection)
- `explain` verb (query plan)
- Inspector panel (schema tab, state machine tab)
- Tenant scope toggle (dev mode)

### Phase 3: Operator Mode
- Operator allowlist enforcement
- Tenant scoping (locked to assigned portfolios/properties)
- Template system
- Confirmation dialogs for mutations
- PII masking
- Human-readable field labels
- Simplified error messages

### Phase 4: CUE and Scripting
- `:cue` meta-command (validate, resolve, unify)
- CUE autocomplete provider
- CUE hot-reload in local dev mode (`--watch`)
- Script mode (variables, pipes, control flow)
- Saved scripts with version history
- Version commands (:versions, :diff, :restore, :pin)

### Phase 5: Advanced Queries
- Edge traversal (`include` with dot-paths)
- `history` and `diff` verbs
- `aggregate` verb with auto-charting
- Graph visualization (inspector graph tab)
- Autocomplete for edges and command payloads

### Phase 6: Local Dev and Collaboration
- `propeller repl-server` CLI command
- `:connect` command for environment switching
- Production confirmation safeguards
- Shared scripts (cross-user)
- Cross-session history
- `watch` verb (live subscriptions)
- `fn` definitions (dev mode)
- Keyboard shortcut customization
- Vim/Emacs key bindings

---

## 14. CUE Expression Evaluation

The REPL supports evaluating raw CUE expressions against the live ontology. This lets developers test constraint changes, validate data against schemas, and explore the type system interactively.

### 14.1 The `:cue` Meta-Command

```pql
-- Evaluate a CUE expression against the ontology
:cue #Lease & { status: "active", monthly_rent: { amount: -500, currency: "USD" } }

-- Output:
-- ERROR: monthly_rent.amount: invalid value -500 (out of bound >0)
--   constraint from: ontology/common.cue:#PositiveMoney.amount
```

```pql
-- Check if a value satisfies a type
:cue #SpaceType & "garage"

-- Output:
-- ✓ "garage" satisfies #SpaceType
```

```pql
-- Explore type definitions
:cue #LeaseStatus

-- Output:
-- "draft" | "pending_approval" | "active" | "expired" |
-- "terminated" | "renewed" | "month_to_month"
```

```pql
-- Unify two schemas (test composition)
:cue #BaseEntity & #StatefulEntity & { status: #LeaseStatus }

-- Output (resolved struct):
-- {
--     id:         string
--     created_at: time.Time
--     updated_at: time.Time
--     created_by: string
--     updated_by: string
--     status:     "draft" | "pending_approval" | "active" | ...
-- }
```

```pql
-- Test a command payload against its schema
:cue #MoveInTenantPayload & {
    lease_id: "lease_123",
    person_id: "person_456",
    move_in_date: "not-a-date"
}

-- Output:
-- ERROR: move_in_date: invalid value "not-a-date" (does not match time.Format)
--   constraint from: commands/lease.cue:#MoveInTenantPayload.move_in_date
```

### 14.2 CUE Evaluation Modes

| Mode | Syntax | Purpose |
|---|---|---|
| **Validate** | `:cue <type> & <value>` | Check a value against a schema, report errors |
| **Resolve** | `:cue <type>` | Show the fully resolved type after all unification |
| **Unify** | `:cue <type1> & <type2>` | Compose two schemas and show the result |
| **Diff** | `:cue diff <type> --since <commit>` | Show what changed in a type definition (dev mode) |

### 14.3 CUE Context

The `:cue` evaluator loads from the same CUE module that drives code generation. The evaluation context includes:

- All packages under `ontology/` (entity types, common types, state machines, relationships)
- All packages under `commands/` (command schemas)
- All packages under `events/` (event schemas)
- All packages under `api/v1/` (API contract types)
- All packages under `policies/` (permission definitions)

The REPL backend keeps a cached CUE runtime instance that reloads when the ontology changes (file watcher in dev, deploy event in production).

### 14.4 CUE Autocomplete

The `:cue` command gets its own autocomplete provider that suggests:

- Type names from the ontology (`#Lease`, `#Person`, `#PositiveMoney`, ...)
- Package-qualified names (`ontology.#Lease`, `commands.#MoveInTenantPayload`)
- Field names within a struct context
- Enum values when the cursor is inside a constrained field

### 14.5 Availability

CUE evaluation is **Dev mode only**. Operators have no need to interact with the type system directly.

---

## 15. Multi-Tenant Scoping

### 15.1 Operator Mode — Tenant-Scoped by Default

Operators are always scoped to their assigned tenant (the portfolios/properties their role grants access to). This is not optional — it mirrors the same data boundary enforced in the Propeller UI.

### 15.2 Dev Mode — Tenant Toggle

Developers see a **tenant scope toggle** in the toolbar:

```
[Scope: All Tenants ▾]   →   dropdown:
                               ● All Tenants (full database)
                               ○ Portfolio: Acme Properties
                               ○ Portfolio: Sunset Management
                               ○ Property: 123 Main St
                               ○ Custom filter...
```

**Behavior:**

- **All Tenants** — no tenant filter applied. Queries see the full database. This is the default for dev mode. Useful for cross-tenant analysis, debugging, and system-level queries.
- **Scoped to portfolio/property** — every PQL query automatically injects a tenant filter. The filter is applied at the Ent query level before execution, not as a PQL rewrite. This simulates what an operator for that portfolio would see.
- **Custom filter** — arbitrary PQL predicate applied to every query (e.g., `where portfolio_id in ["port_001", "port_002"]`).

The scope is shown in the toolbar and prepended to every output block header so there's no ambiguity about what data context produced a result.

### 15.3 Backend Enforcement

```go
type TenantScope struct {
    Mode       string    // "all" | "portfolio" | "property" | "custom"
    EntityIDs  []string  // portfolio or property IDs when scoped
    CustomPred string    // raw PQL predicate for custom mode
}

// Applied in the Planner, before Ent query building
func (p *Planner) ApplyTenantScope(plan *QueryPlan, scope *TenantScope) {
    if scope.Mode == "all" {
        return // no filter
    }
    // Inject predicate that scopes to the tenant boundary
    plan.InjectPredicate(scope.ToPredicate())
}
```

The scope toggle state is stored on the session and persists across queries within a session.

---

## 16. Script Versioning

Saved scripts maintain version history. Every save creates a new version.

### 16.1 Version Model

```go
type ScriptVersion struct {
    ScriptID    string
    Version     int         // monotonically increasing
    Content     string      // PQL source at this version
    Message     string      // optional commit message
    AuthorID    string      // who saved this version
    CreatedAt   time.Time
    Diff        string      // unified diff from previous version (computed)
}
```

### 16.2 Version Commands

```pql
-- Save with version message
:save "monthly_vacancy_report" --message "Added portfolio filter parameter"

-- List versions of a script
:versions "monthly_vacancy_report"

-- Output:
-- v3  2026-02-27  matt   "Added portfolio filter parameter"
-- v2  2026-02-20  matt   "Fixed date range calculation"
-- v1  2026-02-15  matt   "Initial version"

-- Load a specific version
:load "monthly_vacancy_report" --version 2

-- Diff between versions
:diff "monthly_vacancy_report" v1 v3

-- Restore a previous version (creates a new version with old content)
:restore "monthly_vacancy_report" --version 1

-- Pin a version (prevent accidental overwrite of a known-good script)
:pin "monthly_vacancy_report" --version 2
```

### 16.3 Version Retention

- All versions are retained for **90 days**.
- Pinned versions are retained indefinitely.
- The latest version is always retained regardless of age.
- Shared scripts retain all versions for as long as the script exists.

### 16.4 Inspector Integration

The **Saved Scripts** tab in the inspector panel shows version history inline. Clicking a version loads a diff view in the output panel. Double-clicking loads that version into the editor.

---

## 17. Local Development Mode

The REPL supports connecting to a local Propeller instance for engineers running the system on their development machine.

### 17.1 Connection Configuration

```pql
-- Connect to local instance (from browser REPL)
:connect localhost:8080

-- Connect to a named environment
:connect staging
:connect production

-- Show current connection
:env

-- Output:
-- Connection: localhost:8080
-- Database: propeller_dev
-- User: matt@appfolio.com
-- Mode: Dev
-- Tenant Scope: All Tenants
-- Session: #a1b2c3
```

### 17.2 How It Works

The REPL frontend is a static Svelte app that connects to a REPL backend via WebSocket. The backend URL is configurable:

- **Production/staging** — REPL frontend is served from the Propeller deployment, WebSocket connects to the same origin. Authentication goes through the standard auth flow.
- **Local dev** — engineer runs `propeller repl-server` which starts the REPL backend connected to their local Postgres. The browser REPL connects to `localhost:8080` (or configured port). Auth is bypassed in local mode (dev mode always, no operator restrictions).

```bash
# Start local REPL backend
$ propeller repl-server --port 8080 --db "postgres://localhost:5432/propeller_dev"

# With CUE hot-reload (watches ontology files for changes)
$ propeller repl-server --port 8080 --db "..." --watch ./ontology
```

### 17.3 CUE Hot-Reload

In local mode with `--watch`, the REPL backend watches the CUE source files. When an ontology file changes:

1. CUE module is re-evaluated
2. Schema registry is refreshed
3. Autocomplete providers are rebuilt
4. Connected browser sessions receive a `schema_reload` event
5. The inspector panel refreshes if showing schema/state machine tabs

This makes the REPL the primary feedback loop for ontology development: edit a `.cue` file, see the updated types and constraints reflected immediately in the REPL.

### 17.4 Environment Indicator

The toolbar shows a colored environment badge to prevent accidental production operations:

| Environment | Badge | Color |
|---|---|---|
| Local | `LOCAL` | Green |
| Staging | `STAGING` | Yellow |
| Production | `PRODUCTION` | Red |

Production environments additionally require a confirmation step for any `run` command, even in dev mode.

---

## 18. Agent Integration

_Deferred. Not in scope for V1. The agent runtime is a separate surface. If needed later, a `ask` verb or `:agent` meta-command can route natural language queries to the agent runtime from within the REPL._

---

## 19. Updated Meta-Commands (Complete Reference)

| Command | Description | Mode |
|---|---|---|
| `:begin` | Open a transaction sandbox | Dev |
| `:commit` | Commit open transaction | Dev |
| `:rollback` | Rollback open transaction | Dev |
| `:mode dev` / `:mode operator` | Switch modes | Both (requires auth) |
| `:save "name"` | Save current editor content as named script | Both |
| `:save "name" --message "msg"` | Save with version message | Both |
| `:load "name"` | Load saved script into editor | Both |
| `:load "name" --version N` | Load specific version of a script | Both |
| `:scripts` | List saved scripts | Both |
| `:delete "name"` | Delete saved script | Both |
| `:versions "name"` | List version history of a script | Both |
| `:diff "name" vN vM` | Diff two versions of a script | Both |
| `:restore "name" --version N` | Restore a previous version | Both |
| `:pin "name" --version N` | Pin a version (prevent pruning) | Both |
| `:export --format csv\|json` | Export last result to downloadable file | Both |
| `:history` | Show command history for current session | Both |
| `:history --all` | Show command history across sessions | Both |
| `:clear` | Clear output panel | Both |
| `:env` | Show session info (connection, user, mode, tx state, vars) | Both |
| `:vars` | List all bound variables | Both |
| `:set <key> <value>` | Set session config (result_limit, format, etc.) | Both |
| `:connect <target>` | Connect to a different environment | Dev |
| `:cue <expression>` | Evaluate CUE expression against ontology | Dev |
| `:help` | Show help / keyboard shortcuts | Both |
| `:help <verb>` | Show detailed help for a verb | Both |