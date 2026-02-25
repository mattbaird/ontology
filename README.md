# Signal Discovery Demo

A narrated walkthrough showing how the signal discovery system gives AI agents
the ability to see cross-cutting patterns that experienced property managers
carry as intuition.

Everything you see in the demo is **computed live** — real API calls, real
signal classification, real escalation rules firing. No mocks.

---

## Quick Start

```bash
# 1. Start the server with demo data
go run ./cmd/server --demo

# 2. In a second terminal, run the demo
bash demo/signals.sh
```

Or with Air hot-reload:

```bash
air -- --demo          # terminal 1
bash demo/signals.sh   # terminal 2
```

The demo is interactive — press Enter to advance through each section.

---

## The Scenario

Six tenants live at Sunset Apartments. Each has a different activity profile
seeded into an in-memory activity store:

| Tenant | Profile | Sentiment |
|--------|---------|-----------|
| **Marcus Johnson** | Flight risk. On-time payments mask 3 noise complaints, roommate departure, unanswered outreach, declining portal activity. | Concerning |
| **Jennifer Park** | Late payment pattern. 3 consecutive late payments (6-12 days), late fee assessed. | Concerning |
| **David Kim** | Lease violation. Unauthorized pet discovered by maintenance. Payments on time. | Mixed |
| **Lisa Hernandez** | Guarantor removed. Co-signer parent removed from lease, expiring in 77 days. | Mixed |
| **James Wright** | Model tenant. 5 months on-time payments, proactively asked about renewal. | Positive |
| **Amy Torres** | Recently renewed. 12-month renewal signed with 2% increase, consistent payments. | Positive |

The demo focuses on Marcus Johnson — the tenant that traditional software says
is "fine" but an experienced property manager would flag immediately.

---

## What You'll See

### Act 1 — The Problem

Without signal discovery, an AI agent checks the obvious places — payment
history, lease dates, balances — and concludes Marcus looks fine. It misses
the 3 noise complaints, the roommate who left, the unanswered outreach, and
the portal silence. These are stored in different tables with no connection
between them.

### Act 2 — One Query Sees Everything

The agent calls a single endpoint:

```
GET /v1/activity/entity/person/marcus-johnson
```

Returns a chronological feed of **every** signal — financial, maintenance,
communication, relationship, lifecycle — unified into one timeline with
category, weight, and polarity metadata on each entry.

### Act 3 — Signal Taxonomy

Every event is classified using a taxonomy defined in the CUE ontology:

- **8 categories**: financial, maintenance, communication, compliance, behavioral, market, relationship, lifecycle
- **5 weights**: critical, strong, moderate, weak, info
- **4 polarities**: positive, negative, neutral, contextual

~30 signal registrations map specific event types (e.g., `PaymentRecorded`,
`ComplaintCreated`) to their classification. Escalation rules fire when
dangerous patterns accumulate.

### Act 4 — The Signal Summary

The agent calls:

```
GET /v1/activity/summary/person/marcus-johnson
```

Returns a pre-aggregated assessment:

- **Overall sentiment**: CONCERNING
- Per-category signal counts with dominant polarity and trend direction
- Weight distribution within each category
- **Escalations triggered**: the `maint_complaint_pattern` rule fires because
  3+ maintenance complaints accumulated within 180 days

### Act 5 — Portfolio Screening

The demo calls `GetSignalSummary` for all 6 tenants and ranks them by
computed sentiment. Marcus and Jennifer surface at the top as concerning —
not because anyone hardcoded that ranking, but because the signal engine
computed it from their activity patterns.

### Act 6 — Cross-Entity Search

```
POST /v1/activity/search
{"query": "noise", "entity_type": "person"}
```

Free-text search across all activity. Finds the 3 noise complaints tied to
Marcus without the agent needing to know which table complaints live in.

### Act 7 — The Reasoning Guide

The system generates `gen/agent/SIGNALS.md` — a reasoning guide loaded into
the agent's context that encodes domain expertise:

- Non-renewal predictors (complaint + communication decline + lease expiring)
- How to interpret absence (no maintenance requests from a long-term tenant
  is a red flag, not a good sign)
- When to escalate vs. when to monitor

This is generated from the CUE ontology. When signal definitions change,
the reasoning guide regenerates automatically.

---

## API Endpoints

All served on the main server (default port 8080):

| Method | Path | Description |
|--------|------|-------------|
| GET | `/v1/activity/entity/{entity_type}/{entity_id}` | Full activity timeline for any entity |
| GET | `/v1/activity/summary/{entity_type}/{entity_id}` | Aggregated signal summary with sentiment and escalations |
| POST | `/v1/activity/portfolio` | Portfolio-wide signal screening |
| POST | `/v1/activity/search` | Free-text search across all activity |

---

## Architecture

```
ontology/signals.cue          Signal taxonomy + 30 registrations (source of truth)
        |
        |--> internal/signals/     Registry, classifier, aggregator (Go)
        |--> internal/activity/    Store interface, indexer, query engine
        |--> 4 API endpoints       Activity, Summary, Portfolio, Search
        |
        '--> gen/agent/SIGNALS.md  Agent reasoning guide (auto-generated)
             gen/agent/TOOLS.md    Updated tool definitions
```

The signal system follows the same ontology-first pattern as the rest of
the codebase: CUE defines it, generators derive everything downstream.
