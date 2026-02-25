# Propeller Ontological Architecture Specification

**Version:** 1.0
**Date:** February 24, 2026
**Author:** Matthew Baird, CTO — AppFolio
**Status:** Draft — For Engineering Leadership Review

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [Architectural Thesis](#2-architectural-thesis)
3. [The Ontology Layer — CUE as Canonical Domain Model](#3-the-ontology-layer)
4. [Domain Models](#4-domain-models)
   - 4.1 Person Model
   - 4.2 Property Model
   - 4.3 Lease Model
   - 4.4 Accounting Model
5. [Schema Projections](#5-schema-projections)
   - 5.1 Postgres via Ent
   - 5.2 Graph Projection
   - 5.3 Search Index
6. [API Contracts — Generated from Ontology](#6-api-contracts)
7. [Event Schemas — Ontologically Typed](#7-event-schemas)
8. [Permission Model — Derived from Relationships](#8-permission-model)
9. [Agent World Model — Ontology as Context](#9-agent-world-model)
10. [Codegen Pipeline](#10-codegen-pipeline)
11. [Implementation Sequence](#11-implementation-sequence)

---

## 1. Executive Summary

Propeller's architecture is organized around a single generative principle: **the ontology is the system**. Every schema, API, event, permission rule, and agent capability is derived from a canonical domain model defined in CUE. Nothing is hand-written that can be generated. Nothing is implicit that can be made explicit.

This document specifies how four core domain models — Person, Property, Lease, and Accounting — are defined once in CUE and then projected into five downstream representations:

```
Ontology (canonical domain model — CUE)
  ├── Schema projections (Postgres via Ent, graph DB, search index)
  ├── API contracts (Connect-RPC services, OpenAPI, generated from ontology)
  ├── Event schemas (ontologically typed, schema-registered)
  ├── Permission model (derived from relationship graph)
  └── Agent world model (ontology as context for LLM reasoning)
```

The key architectural bet: **in an agentic system, the ontology isn't documentation — it's infrastructure.** An AI agent reasoning about a lease renewal needs to know that a Lease connects a Unit to one or more Tenants, that a Lease has a state machine with valid transitions, that activating a Lease changes the Unit's status and creates LedgerEntries. If this knowledge is scattered across code, the agent is blind. If it's centralized in a formal ontology, the agent can reason about the system as fluently as a domain expert.

### Why Now

The December 2025 threshold crossing in agentic coding capabilities created a window where rebuilding from scratch is faster than retrofitting. Propeller's ontological architecture is designed to be implemented by AI agents operating against these specifications. The ontology serves double duty: it defines the system for the agents that build it, and it defines the system for the agents that run it.

### Scope

This spec covers the ontology layer and its five projections for four domain models. It does not cover:

- UI components or frontend architecture
- Infrastructure provisioning (Kubernetes, networking)
- Third-party integrations (payment processors, e-signing, MLS feeds)
- The remaining ~86 modules in the Propeller roadmap

These are addressed in separate specifications that reference the ontology defined here.

---

## 2. Architectural Thesis

### 2.1 The Generative Kernel Pattern

Traditional systems are built bottom-up: database schema first, then ORM models, then API handlers, then documentation. Each layer is hand-written, creating multiple independent sources of truth about what a "Lease" is. Over time, these diverge. The database says a Lease has 47 columns. The API says it has 32 fields. The documentation says it has 28. The agent's tool description says it has 15. No single artifact tells you the complete truth.

Propeller inverts this. The CUE ontology is the **generative kernel** — the single artifact from which all representations are derived:

```
CUE Ontology (source of truth)
  │
  ├─ cue vet                    → Validate internal consistency
  │
  ├─ cmd/entgen                 → Ent schemas (fields, edges, indexes, policies, hooks)
  │    └─ ent generate          → Go types, CRUD, graph traversal, migrations
  │
  ├─ cmd/apigen                 → Connect-RPC proto services
  │    └─ buf generate          → Go server/client, TypeScript SDK
  │
  ├─ cmd/eventgen               → Event payload schemas (JSON Schema / Avro)
  │    └─ schema registry       → Runtime contract enforcement
  │
  ├─ cmd/authzgen               → OPA/Rego policy scaffolds
  │    └─ relationship graph    → Runtime permission evaluation
  │
  └─ cmd/agentgen               → Agent tool definitions + TOOLS.md
       └─ system prompt         → Agent's understanding of the domain
```

The rule: **no hand-written type that represents a domain entity.** If a struct, proto message, API field, event payload, or tool parameter describes a domain concept, it must be generated from the CUE ontology. Hand-written code exists only for business logic within generated scaffolds.

### 2.2 Why CUE

CUE was chosen over Protobuf, JSON Schema, and OWL/SHACL for specific reasons:

| Capability | CUE | Protobuf | JSON Schema | OWL/SHACL |
|---|---|---|---|---|
| Cross-field constraints | ✅ Native | ❌ | ⚠️ Limited | ✅ |
| State machine definitions | ✅ Native | ❌ | ❌ | ⚠️ |
| Conditional required fields | ✅ Native | ❌ | ⚠️ | ✅ |
| Default values with constraints | ✅ Native | ❌ (proto3) | ✅ | ✅ |
| Go ecosystem integration | ✅ Native | ✅ | ⚠️ | ❌ |
| Code generation | ✅ Export to JSON/YAML/Proto | ✅ Native | ⚠️ | ❌ |
| Human readability | ✅ Excellent | ⚠️ | ⚠️ | ❌ |
| Team learning curve | ⚠️ Moderate | ✅ Low | ✅ Low | ❌ High |
| Closed-world assumption | ✅ Yes | ✅ Yes | ✅ Yes | ❌ Open-world |

The critical differentiator: CUE can express that "a fixed-term Lease MUST have an end date" and "a NNN Lease MUST include property tax, insurance, and utilities in CAM terms" directly in the schema. In Protobuf, these rules live in disconnected validation code. In an agentic system, disconnected rules are invisible rules — the agent doesn't know they exist until it violates them.

CUE also generates Protobuf, so we get CUE's expressiveness for ontology definition AND Protobuf's ecosystem for wire format and service definitions. The two are complementary, not competing.

### 2.3 Why Ent

Ent (entgo.io) is the persistence layer because it thinks in terms of entities and relationships — the same mental model as the ontology:

- **Schema-as-graph**: entities (nodes) with typed fields and edges (relationships)
- **Privacy framework**: authorization policies defined per-entity, evaluated as graph traversals
- **Hooks**: lifecycle callbacks for state machine enforcement, event emission, and cross-field validation
- **Code generation**: full Go CRUD + typed graph traversal from schema definitions
- **Migration generation**: database migrations derived from schema changes

Ent schemas are generated from CUE via `cmd/entgen`. Ent then generates Go code from those schemas. Two layers of generation, one source of truth.

### 2.4 Design Principles

1. **Ontology is infrastructure, not documentation.** The CUE files are not aspirational descriptions — they are the executable specification that the system enforces.

2. **Constraints belong in the ontology, not in code.** If a business rule can be expressed as a constraint on entity structure or relationships, it goes in CUE. Code handles procedural logic; the ontology handles structural truth.

3. **State machines are first-class.** Every entity with a `status` field has an explicit state machine defined in CUE. Valid transitions are generated into Ent hooks that reject invalid transitions at the persistence layer. No code path — human, agent, or migration — can violate the state machine.

4. **Relationships carry semantics.** An edge between entities isn't just a foreign key — it has a relationship type (`manages`, `leased_to`, `billed_to`) that drives authorization, event routing, and agent reasoning.

5. **Events are ontological statements.** A domain event is a typed assertion that "Entity X underwent State Change Y in relationship to Entity Z." Events are schema-registered and validated against the ontology.

6. **The agent and the human see the same API.** There is no separate "agent API." The same Connect-RPC services serve the web UI, mobile app, and AI agents. Agent tool definitions are projections of the same API operations with added LLM-specific guidance.

---

## 3. The Ontology Layer

### 3.1 File Structure

```
propeller/
├── ontology/
│   ├── common.cue              # Shared types: Money, Address, DateRange, AuditMetadata
│   ├── person.cue              # Person model: individuals, organizations, roles, contacts
│   ├── property.cue            # Property model: portfolios, properties, units, amenities
│   ├── lease.cue               # Lease model: leases, terms, charges, applications
│   ├── accounting.cue          # Accounting model: chart of accounts, ledger, journals, bank accounts
│   ├── relationships.cue       # Cross-model relationships and reference types
│   └── state_machines.cue      # All entity state machines in one place
├── codegen/
│   ├── entgen.cue              # Ontology → Ent schema mapping
│   ├── apigen.cue              # Ontology → API service mapping
│   ├── eventgen.cue            # Ontology → Event schema mapping
│   └── authzgen.cue            # Ontology → Permission policy mapping
├── cmd/
│   ├── entgen/main.go          # CUE → Ent schema generator
│   ├── apigen/main.go          # CUE → API contract generator
│   ├── eventgen/main.go        # CUE → Event schema generator
│   ├── authzgen/main.go        # CUE → OPA policy generator
│   └── agentgen/main.go        # CUE → Agent tool definition generator
└── Makefile                    # Pipeline orchestration
```

### 3.2 Common Types

These types are shared across all four domain models. They establish the foundational vocabulary of the ontology.

```cue
// ontology/common.cue
package propeller

import "time"

// ─── Monetary ────────────────────────────────────────────────────────────────

// Money represents a monetary amount. All calculations use integer cents
// to eliminate floating-point errors in financial operations.
#Money: {
    amount_cents: int
    currency:     =~"^[A-Z]{3}$" & *"USD"  // ISO 4217, defaults to USD
}

#NonNegativeMoney: #Money & {
    amount_cents: >= 0
}

#PositiveMoney: #Money & {
    amount_cents: > 0
}

// ─── Temporal ────────────────────────────────────────────────────────────────

#DateRange: {
    start: time.Time
    end?:  time.Time  // Open-ended if unset
    // CONSTRAINT: end must be after start
    if end != _|_ {
        end: time.Time  // Runtime validation ensures end > start
    }
}

// ─── Geographic ──────────────────────────────────────────────────────────────

#Address: {
    line1:       string & !=""
    line2?:      string
    city:        string & !=""
    state:       =~"^[A-Z]{2}$"
    postal_code: =~"^[0-9]{5}(-[0-9]{4})?$"
    country:     =~"^[A-Z]{2}$" & *"US"
    latitude?:   float & >= -90 & <= 90
    longitude?:  float & >= -180 & <= 180
    county?:     string  // Important for tax jurisdictions
}

// ─── Identity ────────────────────────────────────────────────────────────────

// EntityRef is the universal relationship primitive. Every cross-entity
// reference in the ontology uses this type, ensuring that relationships
// are always typed and semantically meaningful.
#EntityRef: {
    entity_type:  #EntityType
    entity_id:    string & !=""
    relationship: #RelationshipType
}

#EntityType:
    "person" | "organization" | "portfolio" | "property" | "unit" |
    "lease" | "work_order" | "vendor" | "ledger_entry" | "journal_entry" |
    "account" | "bank_account" | "application" | "inspection" | "document"

#RelationshipType:
    "belongs_to" | "contains" | "managed_by" | "owned_by" |
    "leased_to" | "occupied_by" | "reported_by" | "assigned_to" |
    "billed_to" | "paid_by" | "performed_by" | "approved_by" |
    "guarantor_for" | "emergency_contact_for" | "employed_by" |
    "related_to" | "parent_of" | "child_of"

// ─── Audit ───────────────────────────────────────────────────────────────────

// AuditMetadata is attached to every domain entity. It provides full
// traceability for every change, which is critical for:
// - Compliance (trust accounting, fair housing)
// - Agent accountability (which agent made this change, under what authority)
// - Debugging (correlation IDs trace chains of related changes)
#AuditMetadata: {
    created_by:     string & !=""    // User ID, agent ID, or "system"
    updated_by:     string & !=""
    created_at:     time.Time
    updated_at:     time.Time
    source:         "user" | "agent" | "import" | "system" | "migration"
    correlation_id?: string          // Links related changes across entities
    agent_goal_id?:  string          // If source == "agent", which goal triggered this
}

// ─── Contact ─────────────────────────────────────────────────────────────────

#ContactMethod: {
    type:      "email" | "phone" | "sms" | "mail" | "portal"
    value:     string & !=""
    primary:   bool | *false
    verified:  bool | *false
    opt_out:   bool | *false  // Communication preference
    label?:    string         // "work", "home", "mobile", etc.
}
```

---

## 4. Domain Models

### 4.1 Person Model

The Person model represents all human and organizational actors in the system. This is intentionally a unified model — a single person can be a tenant, a property owner, a vendor contact, and an emergency contact simultaneously. The ontology captures these as relationships, not separate entities.

```cue
// ontology/person.cue
package propeller

import "time"

// ─── Person ──────────────────────────────────────────────────────────────────
// A Person is any individual who interacts with the property management system.
// Roles (tenant, owner, manager, vendor contact) are relationships, not types.
// This prevents the "same person, three records" problem that plagues legacy PM systems.

#Person: {
    id:           string & !=""
    first_name:   string & !=""
    last_name:    string & !=""
    display_name: string | *"\(first_name) \(last_name)"
    
    date_of_birth?: time.Time  // Required for tenant screening
    ssn_last_four?: =~"^[0-9]{4}$"  // Stored encrypted, only last 4 in domain model
    
    contact_methods: [...#ContactMethod] & [_, ...]  // At least one contact method
    preferred_contact: "email" | "sms" | "phone" | "mail" | "portal" | *"email"
    
    // Communication preferences — drives agent behavior
    language_preference: =~"^[a-z]{2}$" & *"en"  // ISO 639-1
    timezone?: string  // IANA timezone
    do_not_contact: bool | *false  // Legal hold — agent must respect this
    
    // Identity verification state
    identity_verified: bool | *false
    verification_method?: "manual" | "id_check" | "credit_check" | "ssn_verify"
    verified_at?: time.Time
    
    // Tags for flexible categorization (not roles — those are relationships)
    tags?: [...string]
    
    // CONSTRAINTS:
    
    // SMS preference requires a phone contact method
    if preferred_contact == "sms" {
        _has_phone: true & or([ for c in contact_methods if c.type == "phone" || c.type == "sms" { true } ])
    }
    
    audit: #AuditMetadata
}

// ─── Organization ────────────────────────────────────────────────────────────
// An Organization represents a business entity — management company, vendor,
// corporate tenant, property owner LLC, etc.

#Organization: {
    id:         string & !=""
    legal_name: string & !=""
    dba_name?:  string  // "Doing Business As"
    
    org_type: "management_company" | "ownership_entity" | "vendor" |
              "corporate_tenant" | "government_agency" | "hoa" |
              "investment_fund" | "other"
    
    tax_id?: string           // EIN / Tax ID — stored encrypted
    tax_id_type?: "ein" | "ssn" | "itin" | "foreign"
    
    status: "active" | "inactive" | "suspended" | "dissolved"
    
    address?: #Address
    contact_methods?: [...#ContactMethod]
    
    // Regulatory
    state_of_incorporation?: =~"^[A-Z]{2}$"
    formation_date?: time.Time
    
    // For management companies — drives trust accounting requirements
    management_license?: string
    license_state?: =~"^[A-Z]{2}$"
    license_expiry?: time.Time
    
    // Relationships to people (officers, contacts, etc.)
    // These are edges in Ent, not embedded data
    
    audit: #AuditMetadata
}

// ─── PersonRole ──────────────────────────────────────────────────────────────
// Roles are relationships between a Person and other entities, not properties
// of the Person. A PersonRole captures the context-specific attributes that
// apply when a Person acts in a particular capacity.

#PersonRole: {
    id:        string & !=""
    person_id: string & !=""
    role_type: "tenant" | "owner" | "property_manager" | "maintenance_tech" |
               "leasing_agent" | "accountant" | "vendor_contact" |
               "guarantor" | "emergency_contact" | "authorized_occupant"
    
    // Scope — what entity this role applies to
    scope_type: "organization" | "portfolio" | "property" | "unit" | "lease"
    scope_id:   string & !=""
    
    status: "active" | "inactive" | "pending" | "terminated"
    
    effective: #DateRange
    
    // Role-specific attributes stored as structured JSON
    // (The CUE constraint system ensures type safety)
    attributes?: #RoleAttributes
    
    audit: #AuditMetadata
}

// Role-specific attribute sets
#RoleAttributes: #TenantAttributes | #OwnerAttributes | #ManagerAttributes | #GuarantorAttributes

#TenantAttributes: {
    _type:              "tenant"
    standing:           "good" | "late" | "collections" | "eviction" | *"good"
    screening_status:   "not_started" | "in_progress" | "approved" | "denied" | "conditional"
    screening_date?:    time.Time
    current_balance?:   #Money
    move_in_date?:      time.Time
    move_out_date?:     time.Time
    pet_count?:         int & >= 0
    vehicle_count?:     int & >= 0
}

#OwnerAttributes: {
    _type:              "owner"
    ownership_percent:  float & > 0 & <= 100
    distribution_method: "ach" | "check" | "hold" | *"ach"
    management_fee_percent?: float & >= 0 & <= 100
    tax_reporting:      "1099" | "k1" | "none" | *"1099"
    reserve_amount?:    #NonNegativeMoney
}

#ManagerAttributes: {
    _type:              "manager"
    license_number?:    string
    license_state?:     =~"^[A-Z]{2}$"
    approval_limit?:    #NonNegativeMoney  // Max they can approve without escalation
    can_sign_leases:    bool | *false
    can_approve_expenses: bool | *true
}

#GuarantorAttributes: {
    _type:              "guarantor"
    guarantee_type:     "full" | "partial" | "conditional"
    guarantee_amount?:  #PositiveMoney     // For partial guarantees
    guarantee_term?:    #DateRange
    credit_score?:      int & >= 300 & <= 850
}
```

### 4.2 Property Model

```cue
// ontology/property.cue
package propeller

import "time"

// ─── Portfolio ───────────────────────────────────────────────────────────────
// The top-level organizational grouping. A management company may manage
// multiple portfolios for different ownership entities.

#Portfolio: {
    id:       string & !=""
    name:     string & !=""
    owner_id: string & !=""  // Organization ID of the ownership entity
    
    management_type: "self_managed" | "third_party" | "hybrid"
    
    // Trust accounting — drives major architectural decisions downstream
    requires_trust_accounting: bool
    trust_bank_account_id?:    string
    
    status: "active" | "inactive" | "onboarding" | "offboarding"
    
    // Financial settings at portfolio level
    default_late_fee_policy?:   string  // Reference to fee schedule
    default_payment_methods?:   [...("ach" | "credit_card" | "check" | "cash" | "money_order")]
    fiscal_year_start_month:    int & >= 1 & <= 12 & *1  // January default
    
    // CONSTRAINT: Trust accounting requires a linked bank account
    if requires_trust_accounting {
        trust_bank_account_id: string & !=""
    }
    
    audit: #AuditMetadata
}

// ─── Property ────────────────────────────────────────────────────────────────

#Property: {
    id:           string & !=""
    portfolio_id: string & !=""
    name:         string & !=""
    address:      #Address
    
    property_type: "single_family" | "multi_family" | "commercial_office" |
                   "commercial_retail" | "mixed_use" | "industrial" |
                   "affordable_housing" | "student_housing" | "senior_living" |
                   "vacation_rental" | "mobile_home_park"
    
    status: "active" | "inactive" | "under_renovation" | "for_sale" | "onboarding"
    
    // Physical
    year_built:           int & >= 1800 & <= 2030
    total_square_footage: float & > 0
    total_units:          int & >= 1
    lot_size_sqft?:       float & > 0
    stories?:             int & >= 1
    parking_spaces?:      int & >= 0
    
    // Regulatory — these drive business rules across the entire system
    jurisdiction_id?:      string  // Links to local ordinance rules
    rent_controlled:       bool | *false
    compliance_programs?:  [...("LIHTC" | "Section8" | "HUD" | "HOME" | "RAD" | "VASH" | "PBV")]
    requires_lead_disclosure: bool | *false  // Pre-1978 buildings
    
    // Financial — property-level overrides
    chart_of_accounts_id?: string  // If different from portfolio default
    bank_account_id?:      string  // If different from portfolio trust account
    
    // Insurance
    insurance_policy_number?: string
    insurance_expiry?:        time.Time
    
    // CONSTRAINTS:
    
    // Single-family = exactly 1 unit
    if property_type == "single_family" {
        total_units: 1
    }
    
    // Affordable housing MUST specify compliance programs
    if property_type == "affordable_housing" {
        compliance_programs: [_, ...]  // At least one
    }
    
    // Rent control requires jurisdiction
    if rent_controlled {
        jurisdiction_id: string & !=""
    }
    
    // Pre-1978 buildings require lead disclosure
    if year_built < 1978 {
        requires_lead_disclosure: true
    }
    
    audit: #AuditMetadata
}

// ─── Unit ────────────────────────────────────────────────────────────────────

#Unit: {
    id:          string & !=""
    property_id: string & !=""
    unit_number: string & !=""  // "101", "A", "Suite 200", etc.
    
    unit_type: "residential" | "commercial_office" | "commercial_retail" |
               "storage" | "parking" | "common_area"
    
    status: "vacant" | "occupied" | "notice_given" | "make_ready" |
            "down" | "model" | "reserved"
    
    // Physical
    square_footage:  float & > 0
    bedrooms?:       int & >= 0
    bathrooms?:      float & >= 0
    floor?:          int
    
    // Features
    amenities?:      [...string]
    floor_plan?:     string  // Reference to floor plan template
    ada_accessible:  bool | *false
    pet_friendly:    bool | *true
    furnished:       bool | *false
    
    // Financial
    market_rent?:    #NonNegativeMoney
    
    // Active lease — computed from relationship traversal
    active_lease_id?: string
    
    // For affordable housing — unit-level income restrictions
    ami_restriction?:  int & >= 0 & <= 150  // % of Area Median Income
    
    // CONSTRAINTS:
    
    // Occupied units MUST have an active lease
    if status == "occupied" {
        active_lease_id: string & !=""
    }
    
    // Residential units should have bedroom/bathroom counts
    if unit_type == "residential" {
        bedrooms:  int
        bathrooms: float
    }
    
    // Parking/storage don't have bedrooms
    if unit_type == "parking" || unit_type == "storage" {
        bedrooms:  0 | *0
        bathrooms: 0 | *0
    }
    
    audit: #AuditMetadata
}
```

### 4.3 Lease Model

```cue
// ontology/lease.cue
package propeller

import "time"

// ─── Lease ───────────────────────────────────────────────────────────────────
// The Lease is the central contractual entity in property management.
// It connects a Unit to one or more Persons (via PersonRole with role_type "tenant"),
// defines the financial terms, and drives the majority of downstream operations.

#Lease: {
    id:          string & !=""
    unit_id:     string & !=""
    property_id: string & !=""  // Denormalized for query efficiency
    
    // Tenant references — via PersonRole, not directly to Person
    tenant_role_ids: [...string] & [_, ...]  // At least one tenant role
    guarantor_role_ids?: [...string]
    
    lease_type: "fixed_term" | "month_to_month" |
                "commercial_nnn" | "commercial_gross" | "commercial_modified_gross" |
                "affordable" | "section_8" | "student"
    
    status: "draft" | "pending_approval" | "pending_signature" | "active" |
            "expired" | "month_to_month_holdover" | "renewed" |
            "terminated" | "eviction"
    
    // Term
    term: #DateRange
    
    // Financial — base rent
    base_rent:        #NonNegativeMoney
    security_deposit: #NonNegativeMoney
    
    // Rent schedule — handles escalations, concessions, free periods
    rent_schedule?: [...#RentScheduleEntry]
    
    // Recurring charges beyond base rent (pet rent, storage, parking, utilities)
    recurring_charges?: [...#RecurringCharge]
    
    // Late fee policy — can override property/portfolio defaults
    late_fee_policy?: #LateFeePolicy
    
    // Commercial-specific
    cam_terms?:           #CAMTerms
    tenant_improvement?:  #TenantImprovement
    renewal_options?:     [...#RenewalOption]
    
    // Affordable housing-specific
    subsidy?: #SubsidyTerms
    
    // Move-in / move-out
    move_in_date?:     time.Time
    move_out_date?:    time.Time
    notice_date?:      time.Time
    notice_required_days: int & >= 0 & *30
    
    // Signing
    signing_method?:    "electronic" | "wet_ink" | "both"
    signed_at?:         time.Time
    document_id?:       string  // Reference to signed lease document
    
    // CONSTRAINTS:
    
    // Fixed-term leases MUST have an end date
    if lease_type == "fixed_term" || lease_type == "student" {
        term: end: time.Time
    }
    
    // Commercial leases MUST have CAM terms
    if lease_type == "commercial_nnn" || lease_type == "commercial_gross" || lease_type == "commercial_modified_gross" {
        cam_terms: #CAMTerms
    }
    
    // NNN leases must include all three nets
    if lease_type == "commercial_nnn" {
        cam_terms: includes_property_tax: true
        cam_terms: includes_insurance:    true
        cam_terms: includes_utilities:    true
    }
    
    // Section 8 leases must have subsidy terms
    if lease_type == "section_8" {
        subsidy: #SubsidyTerms
    }
    
    // Active leases must have a move-in date
    if status == "active" {
        move_in_date: time.Time
    }
    
    // Signed leases must have a signature timestamp
    if status == "active" || status == "expired" || status == "renewed" {
        signed_at: time.Time
    }
    
    audit: #AuditMetadata
}

#RentScheduleEntry: {
    effective_period: #DateRange
    amount:           #NonNegativeMoney
    description:      string & !=""  // "Year 1", "Move-in concession", "Year 2 escalation"
    charge_code:      string & !=""  // Links to chart of accounts
}

#RecurringCharge: {
    id:          string & !=""
    charge_code: string & !=""
    description: string & !=""
    amount:      #NonNegativeMoney
    frequency:   "monthly" | "quarterly" | "annually" | "one_time"
    effective_period: #DateRange
    taxable:     bool | *false
}

#LateFeePolicy: {
    grace_period_days:  int & >= 0 & *5
    fee_type:           "flat" | "percent" | "per_day" | "tiered"
    flat_amount?:       #NonNegativeMoney
    percent?:           float & > 0 & <= 100
    per_day_amount?:    #NonNegativeMoney
    max_fee?:           #NonNegativeMoney
    // Tiered fees
    tiers?: [...{
        days_late_min: int & >= 0
        days_late_max: int
        amount:        #NonNegativeMoney
    }]
}

#CAMTerms: {
    reconciliation_type: "estimated_with_annual_reconciliation" | "fixed" | "actual"
    pro_rata_share_percent: float & > 0 & <= 100
    estimated_monthly_cam:  #NonNegativeMoney
    annual_cap?:            #NonNegativeMoney
    base_year?:             int
    includes_property_tax:  bool
    includes_insurance:     bool
    includes_utilities:     bool
    excluded_categories?:   [...string]
    
    // CONSTRAINT: Fixed CAM has no cap and no reconciliation
    if reconciliation_type == "fixed" {
        annual_cap: _|_  // Explicitly disallowed
    }
}

#TenantImprovement: {
    allowance:       #NonNegativeMoney
    amortized:       bool | *false
    amortization_term_months?: int & > 0
    interest_rate_percent?:    float & >= 0
    completion_deadline?: time.Time
}

#RenewalOption: {
    option_number:     int & >= 1
    term_months:       int & > 0
    rent_adjustment:   "fixed" | "cpi" | "percent_increase" | "market"
    fixed_rent?:       #NonNegativeMoney
    percent_increase?: float & >= 0
    notice_required_days: int & >= 0 & *90
    must_exercise_by?: time.Time
}

#SubsidyTerms: {
    program:           "section_8" | "pbv" | "vash" | "home" | "lihtc"
    housing_authority: string & !=""
    hap_contract_id?:  string
    contract_rent:     #NonNegativeMoney  // Total rent per HAP contract
    tenant_portion:    #NonNegativeMoney  // What tenant pays
    subsidy_portion:   #NonNegativeMoney  // What HA pays
    utility_allowance: #NonNegativeMoney
    annual_recert_date?: time.Time
    income_limit_ami_percent: int & > 0 & <= 150
}

// ─── Application ─────────────────────────────────────────────────────────────
// Lease applications track prospects through the leasing pipeline.

#Application: {
    id:          string & !=""
    property_id: string & !=""
    unit_id?:    string         // May apply to a property without specific unit
    applicant_person_id: string & !=""
    
    status: "submitted" | "screening" | "under_review" | "approved" |
            "conditionally_approved" | "denied" | "withdrawn" | "expired"
    
    desired_move_in:  time.Time
    desired_lease_term_months: int & > 0
    
    // Screening
    screening_request_id?: string
    screening_completed?:  time.Time
    credit_score?:         int & >= 300 & <= 850
    background_clear:      bool | *false
    income_verified:       bool | *false
    income_to_rent_ratio?: float & >= 0
    
    // Decision
    decision_by?:     string   // Person ID of reviewer
    decision_at?:     time.Time
    decision_reason?: string
    conditions?:      [...string]  // For conditional approval
    
    // Financial
    application_fee:  #NonNegativeMoney
    fee_paid:         bool | *false
    
    // CONSTRAINTS:
    
    // Approved applications must have a decision
    if status == "approved" || status == "conditionally_approved" || status == "denied" {
        decision_by: string & !=""
        decision_at: time.Time
    }
    
    // Denied must have a reason (fair housing compliance)
    if status == "denied" {
        decision_reason: string & !=""
    }
    
    audit: #AuditMetadata
}
```

### 4.4 Accounting Model

```cue
// ontology/accounting.cue
package propeller

import "time"

// ─── Chart of Accounts ──────────────────────────────────────────────────────
// The CoA is hierarchical and supports multi-dimensional accounting:
// entity + property + custom dimensions. This addresses the key AppFolio
// deficiency of flat, inflexible account structures.

#Account: {
    id:               string & !=""
    account_number:   string & !=""  // e.g., "1000", "4100.001"
    name:             string & !=""
    description?:     string
    
    account_type: "asset" | "liability" | "equity" | "revenue" | "expense"
    
    account_subtype: "cash" | "accounts_receivable" | "prepaid" | "fixed_asset" |
                     "accumulated_depreciation" | "other_asset" |
                     "accounts_payable" | "accrued_liability" | "unearned_revenue" |
                     "security_deposits_held" | "other_liability" |
                     "owners_equity" | "retained_earnings" | "distributions" |
                     "rental_income" | "other_income" | "cam_recovery" |
                     "operating_expense" | "maintenance_expense" | "utility_expense" |
                     "management_fee_expense" | "depreciation_expense" | "other_expense"
    
    // Hierarchy
    parent_account_id?: string
    depth:              int & >= 0  // 0 = top-level
    
    // Multi-dimensional — enables entity + property + cost center tracking
    // without needing separate accounts per combination
    dimensions?: #AccountDimensions
    
    // Behavior
    normal_balance:   "debit" | "credit"
    is_header:        bool | *false      // Header accounts can't receive postings
    is_system:        bool | *false      // System accounts can't be deleted
    allows_direct_posting: bool | *true  // Some accounts only accept sub-account postings
    
    // Status
    status: "active" | "inactive" | "archived"
    
    // Trust accounting flag — segregation enforcement
    is_trust_account: bool | *false
    trust_type?:      "operating" | "security_deposit" | "escrow"
    
    // Budgeting
    budget_amount?:   #Money           // Annual budget
    
    // Tax
    tax_line?:        string           // Maps to tax form line (1099, K-1)
    
    // CONSTRAINTS:
    
    // Normal balance must match account type
    if account_type == "asset" || account_type == "expense" {
        normal_balance: "debit"
    }
    if account_type == "liability" || account_type == "equity" || account_type == "revenue" {
        normal_balance: "credit"
    }
    
    // Header accounts can't receive postings
    if is_header {
        allows_direct_posting: false
    }
    
    // Trust accounts must specify trust type
    if is_trust_account {
        trust_type: string & !=""
    }
    
    audit: #AuditMetadata
}

#AccountDimensions: {
    // Primary dimensions — always available
    entity_id?:     string   // Legal entity (LLC, fund, etc.)
    property_id?:   string   // Property-level tracking
    
    // Custom dimensions — flexible per-organization
    dimension_1?:   string   // Commonly: department
    dimension_2?:   string   // Commonly: cost center
    dimension_3?:   string   // Commonly: project / job code
}

// ─── Ledger Entry ────────────────────────────────────────────────────────────
// The atomic unit of financial record-keeping. Every financial event in the
// system produces one or more LedgerEntries. Entries are IMMUTABLE — errors
// are corrected with adjustment entries, never by modifying existing entries.

#LedgerEntry: {
    id:         string & !=""
    account_id: string & !=""
    
    entry_type: "charge" | "payment" | "credit" | "adjustment" |
                "refund" | "deposit" | "nsf" | "write_off" |
                "late_fee" | "management_fee" | "owner_draw"
    
    amount: #Money
    
    // Double-entry: every entry belongs to a journal entry
    journal_entry_id: string & !=""
    
    // Temporal — effective vs posted supports accrual accounting
    effective_date: time.Time  // When the economic event occurred
    posted_date:    time.Time  // When it was recorded in the system
    
    description: string & !=""
    charge_code: string & !=""  // Links to CoA for categorization
    memo?:       string         // Additional detail
    
    // Dimensional references — what entities does this entry relate to?
    property_id:   string & !=""  // Always required
    unit_id?:      string
    lease_id?:     string
    person_id?:    string         // Tenant, owner, vendor who this relates to
    
    // Bank / trust accounting
    bank_account_id?: string
    bank_transaction_id?: string  // Reference to bank feed transaction
    
    // Reconciliation
    reconciled:        bool | *false
    reconciliation_id?: string
    reconciled_at?:    time.Time
    
    // For adjustments — what entry is this correcting?
    adjusts_entry_id?: string
    
    // CONSTRAINTS:
    
    // Payments and refunds must reference a person
    if entry_type == "payment" || entry_type == "refund" || entry_type == "nsf" {
        person_id: string & !=""
    }
    
    // Lease-related charges must reference a lease
    if entry_type == "charge" || entry_type == "late_fee" {
        lease_id: string & !=""
    }
    
    // Adjustments must reference the entry being corrected
    if entry_type == "adjustment" {
        adjusts_entry_id: string & !=""
    }
    
    // Reconciled entries must have reconciliation details
    if reconciled {
        reconciliation_id: string & !=""
        reconciled_at:     time.Time
    }
    
    // IMMUTABILITY — enforced at Ent hook level, documented here for ontological completeness
    // LedgerEntries cannot be updated or deleted. Corrections use adjustment entries.
    
    audit: #AuditMetadata
}

// ─── Journal Entry ───────────────────────────────────────────────────────────
// Groups LedgerEntries that must balance (debits = credits).
// This is the enforcement point for double-entry accounting.

#JournalEntry: {
    id:          string & !=""
    
    entry_date:  time.Time
    posted_date: time.Time
    
    description: string & !=""
    
    // Source tracking — where did this journal entry come from?
    source_type: "manual" | "auto_charge" | "payment" | "bank_import" |
                 "cam_reconciliation" | "depreciation" | "accrual" |
                 "intercompany" | "management_fee" | "system"
    source_id?:  string  // ID of the originating transaction/process
    
    // Approval
    status: "draft" | "pending_approval" | "posted" | "voided"
    approved_by?: string
    approved_at?: time.Time
    
    // Batch (for bulk operations like rent charges, management fees)
    batch_id?: string
    
    // Entity / property scope
    entity_id?:   string
    property_id?: string
    
    // Reversal tracking
    reverses_journal_id?: string
    reversed_by_journal_id?: string
    
    // The actual line items — must balance
    // (Validated at Ent hook level: sum of debits = sum of credits)
    lines: [...#JournalLine] & [_, _, ...]  // At least 2 lines
    
    // CONSTRAINTS:
    
    // Posted entries must be approved (unless auto-generated)
    if status == "posted" && source_type == "manual" {
        approved_by: string & !=""
        approved_at: time.Time
    }
    
    // Voided entries must reference a reversal
    if status == "voided" {
        reversed_by_journal_id: string & !=""
    }
    
    audit: #AuditMetadata
}

#JournalLine: {
    account_id:  string & !=""
    debit?:      #NonNegativeMoney
    credit?:     #NonNegativeMoney
    description?: string
    dimensions?: #AccountDimensions
    
    // Must have exactly one of debit or credit (not both, not neither)
    // This is enforced at validation
}

// ─── Bank Account ────────────────────────────────────────────────────────────

#BankAccount: {
    id:               string & !=""
    name:             string & !=""
    
    account_type: "operating" | "trust" | "security_deposit" | "escrow" | "reserve"
    
    // Linked CoA account
    gl_account_id: string & !=""
    
    // Bank details — stored encrypted
    bank_name:         string & !=""
    routing_number?:   string  // Encrypted
    account_number_last_four: =~"^[0-9]{4}$"
    
    // Scope
    portfolio_id?: string  // If portfolio-specific
    property_id?:  string  // If property-specific (rare)
    entity_id?:    string  // Legal entity that owns this account
    
    status: "active" | "inactive" | "frozen" | "closed"
    
    // Current balance (computed from reconciled transactions)
    current_balance?: #Money
    last_reconciled_at?: time.Time
    
    // Trust accounting controls
    is_trust: bool | *false
    trust_state?: =~"^[A-Z]{2}$"  // Trust account regulations vary by state
    commingling_allowed: bool | *false  // Operating + trust in same account
    
    // CONSTRAINTS:
    
    // Trust accounts must specify the trust state
    if is_trust {
        trust_state:         =~"^[A-Z]{2}$"
        commingling_allowed: false  // Trust accounts NEVER allow commingling
    }
    
    audit: #AuditMetadata
}

// ─── Reconciliation ──────────────────────────────────────────────────────────

#Reconciliation: {
    id:              string & !=""
    bank_account_id: string & !=""
    
    period_start:    time.Time
    period_end:      time.Time
    
    statement_balance: #Money
    system_balance:    #Money
    difference:        #Money
    
    status: "in_progress" | "balanced" | "unbalanced" | "approved"
    
    matched_transaction_count:   int & >= 0
    unmatched_transaction_count: int & >= 0
    
    completed_by?: string
    completed_at?: time.Time
    approved_by?:  string
    approved_at?:  time.Time
    
    // CONSTRAINT: Balanced means difference is zero
    if status == "balanced" || status == "approved" {
        difference: amount_cents: 0
    }
    
    audit: #AuditMetadata
}
```

### 4.5 State Machines

All entity state machines are defined in a single file for cross-referencing and consistency.

```cue
// ontology/state_machines.cue
package propeller

// Every status enum in the ontology has a corresponding transition map here.
// These are generated into Ent hooks that reject invalid transitions at the
// persistence layer. No code path can violate these transitions.

#LeaseTransitions: {
    draft:                   ["pending_approval", "pending_signature", "terminated"]
    pending_approval:        ["draft", "pending_signature", "terminated"]
    pending_signature:       ["active", "draft", "terminated"]
    active:                  ["expired", "month_to_month_holdover", "terminated", "eviction"]
    expired:                 ["active", "month_to_month_holdover", "renewed", "terminated"]
    month_to_month_holdover: ["active", "renewed", "terminated", "eviction"]
    renewed:                 []  // Terminal — a new lease is created
    terminated:              []  // Terminal
    eviction:                ["terminated"]
}

#UnitTransitions: {
    vacant:       ["occupied", "make_ready", "down", "model", "reserved"]
    occupied:     ["notice_given"]
    notice_given: ["make_ready", "occupied"]  // Can rescind notice
    make_ready:   ["vacant", "down"]
    down:         ["make_ready", "vacant"]
    model:        ["vacant", "occupied"]
    reserved:     ["vacant", "occupied"]
}

#ApplicationTransitions: {
    submitted:              ["screening", "withdrawn"]
    screening:              ["under_review", "withdrawn"]
    under_review:           ["approved", "conditionally_approved", "denied", "withdrawn"]
    approved:               ["expired"]  // If lease not signed in time
    conditionally_approved: ["approved", "denied", "withdrawn", "expired"]
    denied:                 []  // Terminal
    withdrawn:              []  // Terminal
    expired:                []  // Terminal
}

#JournalEntryTransitions: {
    draft:            ["pending_approval", "posted"]  // Auto-generated can go straight to posted
    pending_approval: ["posted", "draft"]              // Reject sends back to draft
    posted:           ["voided"]
    voided:           []  // Terminal
}

#PortfolioTransitions: {
    onboarding: ["active"]
    active:     ["inactive", "offboarding"]
    inactive:   ["active", "offboarding"]
    offboarding: ["inactive"]  // After all properties migrated
}

#PropertyTransitions: {
    onboarding:       ["active"]
    active:           ["inactive", "under_renovation", "for_sale"]
    inactive:         ["active"]
    under_renovation: ["active", "for_sale"]
    for_sale:         ["active", "inactive"]
}

#PersonRoleTransitions: {
    pending:    ["active", "terminated"]
    active:     ["inactive", "terminated"]
    inactive:   ["active", "terminated"]
    terminated: []  // Terminal
}

#BankAccountTransitions: {
    active:   ["inactive", "frozen", "closed"]
    inactive: ["active", "closed"]
    frozen:   ["active", "closed"]
    closed:   []  // Terminal
}

#ReconciliationTransitions: {
    in_progress: ["balanced", "unbalanced"]
    balanced:    ["approved", "in_progress"]  // Reopen if errors found
    unbalanced:  ["in_progress"]
    approved:    []  // Terminal
}
```

### 4.6 Cross-Model Relationships

```cue
// ontology/relationships.cue
package propeller

// This file defines the EDGES between domain models.
// These relationships drive:
//   - Ent edge generation (foreign keys + graph traversal)
//   - Permission model (access paths through the relationship graph)
//   - Agent reasoning (understanding how entities connect)
//   - Event routing (which subscribers care about this entity?)

#OntologyRelationship: {
    from:         string  // Source entity type
    to:           string  // Target entity type
    edge_name:    string  // Name of the edge (lowercase)
    cardinality:  "O2O" | "O2M" | "M2O" | "M2M"
    required:     bool | *false
    semantic:     string  // Human-readable relationship meaning
    inverse_name: string  // Edge name on the target side
}

relationships: [...#OntologyRelationship]
relationships: [
    // Portfolio relationships
    {from: "Portfolio", to: "Property", edge_name: "properties", cardinality: "O2M",
     semantic: "Portfolio contains Properties", inverse_name: "portfolio"},
    {from: "Portfolio", to: "Organization", edge_name: "owner", cardinality: "M2O", required: true,
     semantic: "Portfolio is owned by Organization", inverse_name: "owned_portfolios"},
    {from: "Portfolio", to: "BankAccount", edge_name: "trust_account", cardinality: "O2O",
     semantic: "Portfolio uses BankAccount for trust funds", inverse_name: "trust_portfolio"},
    
    // Property relationships
    {from: "Property", to: "Unit", edge_name: "units", cardinality: "O2M",
     semantic: "Property contains Units", inverse_name: "property"},
    {from: "Property", to: "BankAccount", edge_name: "bank_account", cardinality: "M2O",
     semantic: "Property uses BankAccount", inverse_name: "properties"},
    
    // Unit relationships
    {from: "Unit", to: "Lease", edge_name: "leases", cardinality: "O2M",
     semantic: "Unit has Leases over time", inverse_name: "unit"},
    {from: "Unit", to: "Lease", edge_name: "active_lease", cardinality: "O2O",
     semantic: "Unit has at most one active Lease", inverse_name: "occupied_unit"},
    
    // Lease relationships
    {from: "Lease", to: "PersonRole", edge_name: "tenant_roles", cardinality: "M2M",
     semantic: "Lease is held by tenant PersonRoles", inverse_name: "leases"},
    {from: "Lease", to: "PersonRole", edge_name: "guarantor_roles", cardinality: "M2M",
     semantic: "Lease is guaranteed by guarantor PersonRoles", inverse_name: "guaranteed_leases"},
    {from: "Lease", to: "LedgerEntry", edge_name: "ledger_entries", cardinality: "O2M",
     semantic: "Lease generates LedgerEntries", inverse_name: "lease"},
    {from: "Lease", to: "Application", edge_name: "application", cardinality: "O2O",
     semantic: "Lease originated from Application", inverse_name: "resulting_lease"},
    
    // Person relationships
    {from: "Person", to: "PersonRole", edge_name: "roles", cardinality: "O2M",
     semantic: "Person has Roles in various contexts", inverse_name: "person"},
    {from: "Person", to: "Organization", edge_name: "organizations", cardinality: "M2M",
     semantic: "Person is affiliated with Organizations", inverse_name: "people"},
    
    // Organization relationships
    {from: "Organization", to: "Organization", edge_name: "subsidiaries", cardinality: "O2M",
     semantic: "Organization has subsidiary Organizations", inverse_name: "parent_org"},
    
    // Accounting relationships
    {from: "Account", to: "Account", edge_name: "children", cardinality: "O2M",
     semantic: "Account has sub-Accounts", inverse_name: "parent"},
    {from: "LedgerEntry", to: "JournalEntry", edge_name: "journal_entry", cardinality: "M2O", required: true,
     semantic: "LedgerEntry belongs to JournalEntry", inverse_name: "lines"},
    {from: "LedgerEntry", to: "Account", edge_name: "account", cardinality: "M2O", required: true,
     semantic: "LedgerEntry posts to Account", inverse_name: "entries"},
    {from: "LedgerEntry", to: "Property", edge_name: "property", cardinality: "M2O", required: true,
     semantic: "LedgerEntry relates to Property", inverse_name: "ledger_entries"},
    {from: "LedgerEntry", to: "Person", edge_name: "person", cardinality: "M2O",
     semantic: "LedgerEntry relates to Person", inverse_name: "ledger_entries"},
    {from: "BankAccount", to: "Account", edge_name: "gl_account", cardinality: "M2O", required: true,
     semantic: "BankAccount is tracked via GL Account", inverse_name: "bank_accounts"},
    {from: "Reconciliation", to: "BankAccount", edge_name: "bank_account", cardinality: "M2O", required: true,
     semantic: "Reconciliation is for BankAccount", inverse_name: "reconciliations"},
    
    // Application relationships
    {from: "Application", to: "Person", edge_name: "applicant", cardinality: "M2O", required: true,
     semantic: "Application was submitted by Person", inverse_name: "applications"},
    {from: "Application", to: "Property", edge_name: "property", cardinality: "M2O", required: true,
     semantic: "Application is for Property", inverse_name: "applications"},
    {from: "Application", to: "Unit", edge_name: "unit", cardinality: "M2O",
     semantic: "Application is for specific Unit", inverse_name: "applications"},
]
```

---

## 5. Schema Projections

The ontology projects into three storage representations, each optimized for different access patterns.

### 5.1 Postgres via Ent

**Primary datastore.** All CRUD operations, transactional consistency, and authoritative state. Ent schemas are generated from the CUE ontology via `cmd/entgen`.

**Generation pipeline:**

```
ontology/*.cue → codegen/entgen.cue → cmd/entgen → ent/schema/*.go → ent generate → Go CRUD code
```

**What `entgen.cue` maps:**

| CUE Concept | Ent Construct | Example |
|---|---|---|
| Entity fields | `field.*` methods | `field.Enum("status").Values(...)` |
| `#EntityRef` edges | `edge.To/From` | `edge.To("units", Unit.Type)` |
| `*Transitions` state machines | Hooks with `EnforceStateMachine` | Reject invalid status changes |
| Cross-field constraints | Hooks with `ValidateOntology` | "Fixed-term lease needs end date" |
| `#AuditMetadata` | `AuditMixin` | Created/updated timestamps, source |
| `sensitive: true` fields | `.Sensitive()` | SSN, bank numbers excluded from logs |
| Relationship cardinality | Edge type + `.Required()` + `.Unique()` | M2O required = FK NOT NULL |

**What Ent generates from our schemas:**

- Go structs for every entity with typed fields
- CRUD operations (Create, Query, Update, Delete) with builder pattern
- Graph traversal: `client.Property.Query().QueryUnits().QueryLeases().Where(lease.StatusEQ("active")).All(ctx)`
- Database migrations: `atlas` format, reviewable, version-controlled
- Privacy policies: per-entity authorization evaluated on every query
- Hooks: lifecycle callbacks for validation, state machines, event emission

**Key architectural decisions:**

1. **Money stored as integer cents.** `base_rent_cents int64` not `base_rent decimal`. Eliminates floating-point accounting errors at the storage layer.

2. **Embedded JSON for structured sub-objects.** `rent_schedule`, `cam_terms`, `recurring_charges` are stored as JSON columns with Go struct typing. This keeps the entity table count manageable while preserving type safety.

3. **Denormalized references for query efficiency.** `Lease.property_id` exists even though the path `Lease → Unit → Property` could derive it. The denormalization is documented in the ontology (`// Denormalized for query efficiency`) so it's explicit, not accidental.

4. **Soft deletes only for financial entities.** LedgerEntries, JournalEntries, and Reconciliations are immutable — no update or delete. Corrections use adjustment entries. The Ent hook `DenyLedgerDelete` enforces this.

5. **UUID primary keys everywhere.** Generated at application layer, not database. Enables distributed ID generation and eliminates sequential ID enumeration attacks.

### 5.2 Graph Projection (Neo4j)

**Secondary datastore.** Optimized for relationship-heavy queries that are expensive in Postgres: "find all entities connected to this person across all their roles," "what is the full permission path from this user to this unit?" "show me the complete financial chain from this payment to the bank account."

**Sync mechanism:** Event-driven. Every domain event (emitted from Ent hooks) is consumed by a graph sync worker that maintains a Neo4j projection of the ontology's relationship graph.

```
Ent Hook → Domain Event → NATS → Graph Sync Worker → Neo4j
```

**What lives in the graph:**

- All entities as nodes (with minimal properties — ID, type, status, name)
- All relationships from `relationships.cue` as typed edges
- PersonRole as the relationship node connecting Person to scoped entities
- Account hierarchy (parent/child) for financial roll-ups

**What does NOT live in the graph:**

- Full entity data (that's in Postgres)
- Financial amounts (that's in the ledger)
- Mutable state that changes frequently (balances, computed fields)

**Key queries the graph serves:**

```cypher
// Permission check: Can user X access lease Y?
MATCH path = (user:Person)-[:HAS_ROLE]->(role:PersonRole)-[:SCOPED_TO]->
  ()-[:CONTAINS*0..3]->(unit:Unit)<-[:FOR_UNIT]-(lease:Lease {id: $leaseId})
WHERE role.status = 'active'
RETURN path

// Impact analysis: What entities are affected if we terminate this lease?
MATCH (lease:Lease {id: $leaseId})-[*1..3]-(affected)
RETURN affected, type(affected)

// Financial chain: Trace this payment to its bank account
MATCH path = (entry:LedgerEntry {id: $entryId})-[:BELONGS_TO]->(journal:JournalEntry),
      (entry)-[:POSTS_TO]->(account:Account),
      (entry)-[:DEPOSITED_TO]->(bank:BankAccount)
RETURN path
```

### 5.3 Search Index (Meilisearch or Typesense)

**Tertiary datastore.** Full-text search across all entity types, powering the unified search experience for both human users and agents.

**Sync mechanism:** Same event-driven pattern as the graph projection.

```
Ent Hook → Domain Event → NATS → Search Sync Worker → Meilisearch
```

**Index structure — one index per entity type:**

```json
{
  "index": "properties",
  "primaryKey": "id",
  "searchableAttributes": ["name", "address_line1", "address_city", "address_state"],
  "filterableAttributes": ["portfolio_id", "property_type", "status", "rent_controlled"],
  "sortableAttributes": ["name", "year_built", "total_units"]
}
```

**Agent search tool:** The agent's `search_entities` tool issues queries against this index with `entity_type` as a filter. This gives the agent sub-50ms search across all entity types with structured filtering — the same tool described in the Propeller agent spec.

---

## 6. API Contracts — Generated from Ontology

### 6.1 Architecture

APIs are defined in `codegen/apigen.cue` and generated into Connect-RPC service definitions. Connect-RPC provides gRPC, gRPC-Web, and REST from a single proto definition, meaning the same service handles browser clients, mobile apps, server-to-server calls, and agent tool invocations.

**Generation pipeline:**

```
ontology/*.cue → codegen/apigen.cue → cmd/apigen →
  ├── gen/proto/*_service.proto → buf generate → Go server stubs + TS client SDK
  ├── gen/connect/*_handler.go (scaffolds with auth + Ent queries + event emission)
  ├── gen/openapi/propeller-api.json (OpenAPI 3.1 spec)
  └── gen/agent/propeller-tools.json (Anthropic function-calling format)
```

### 6.2 Service Organization

One service per top-level domain area, not per entity:

| Service | Entities | Base Path |
|---|---|---|
| `PersonService` | Person, Organization, PersonRole | `/v1/persons` |
| `PropertyService` | Portfolio, Property, Unit | `/v1/properties` |
| `LeaseService` | Lease, Application | `/v1/leases` |
| `AccountingService` | Account, LedgerEntry, JournalEntry, BankAccount, Reconciliation | `/v1/accounting` |

### 6.3 API Design Principles

**State transitions are named operations, not generic updates.**

Each ontological state transition becomes its own RPC method with a specific request type, specific authorization, and specific event:

| Instead of | Use |
|---|---|
| `UpdateLease(status: "active")` | `ActivateLease(lease_id, move_in_date, confirmed_rent)` |
| `UpdateLease(status: "terminated")` | `TerminateLease(lease_id, reason, move_out_date)` |
| `UpdateApplication(status: "approved")` | `ApproveApplication(application_id, conditions)` |
| `UpdateJournalEntry(status: "posted")` | `PostJournalEntry(journal_id, approval_notes)` |

This matters for agents — calling `activate_lease` communicates clear intent. The agent doesn't need to know which status values are valid from which states.

**Side effects are reported, not hidden.**

When an operation causes downstream changes, the response includes them:

```protobuf
message ActivateLeaseResponse {
    Lease lease = 1;
    repeated SideEffect side_effects = 2;
    // Example side_effects:
    // - Unit status changed to "occupied"
    // - Security deposit LedgerEntry created
    // - First month rent charge scheduled
    // - Welcome email queued for tenant
}
```

This is essential for agent reasoning — the agent needs to know the full consequence chain of its actions.

**Pagination uses cursor-based tokens, not offset.**

All list operations return `next_page_token` and accept `page_token`. This provides stable pagination under concurrent writes and works naturally with graph traversal results.

**Includes are ontology-driven.**

The `include` parameter on GET operations only accepts edge names defined in the ontology:

```
GET /v1/leases/{id}?include=unit,tenant_roles,ledger_entries
```

Invalid includes (edges that don't exist on the entity) return 400. This prevents N+1 queries while keeping the API surface honest about entity relationships.

### 6.4 Service Definitions (Lease Example)

```
LeaseService:
  ├── CreateLease          POST   /v1/leases
  ├── GetLease             GET    /v1/leases/{id}
  ├── ListLeases           GET    /v1/leases
  ├── UpdateLease          PATCH  /v1/leases/{id}           (mutable fields only)
  ├── SearchLeases         POST   /v1/leases/search
  │
  │   State Transitions:
  ├── SubmitForApproval    POST   /v1/leases/{id}/submit
  ├── ApproveLease         POST   /v1/leases/{id}/approve
  ├── SendForSignature     POST   /v1/leases/{id}/sign
  ├── ActivateLease        POST   /v1/leases/{id}/activate
  ├── RecordNotice         POST   /v1/leases/{id}/notice
  ├── RenewLease           POST   /v1/leases/{id}/renew
  ├── TerminateLease       POST   /v1/leases/{id}/terminate
  ├── InitiateEviction     POST   /v1/leases/{id}/evict
  │
  │   Sub-resources:
  ├── GetLeaseLedger       GET    /v1/leases/{id}/ledger
  ├── RecordPayment        POST   /v1/leases/{id}/payments
  ├── PostCharge           POST   /v1/leases/{id}/charges
  └── ApplyCredit          POST   /v1/leases/{id}/credits
```

---

## 7. Event Schemas — Ontologically Typed

### 7.1 Event Architecture

Every mutation in the system emits a domain event. Events are ontologically typed — they reference specific entity types and state changes from the CUE ontology. Events are the mechanism that keeps the graph projection, search index, and any other derived datastores in sync.

**Infrastructure:** NATS JetStream with schema registry enforcement.

**Event envelope:**

```cue
#DomainEvent: {
    // Identity
    event_id:    string & !=""       // UUID, globally unique
    event_type:  string & !=""       // e.g., "LeaseActivated"
    
    // Ontological classification
    entity_type: #EntityType         // From the ontology
    entity_id:   string & !=""
    
    // Temporal
    occurred_at: time.Time           // When the business event happened
    recorded_at: time.Time           // When we recorded it (always >= occurred_at)
    
    // Causation chain
    correlation_id: string & !=""    // Groups related events across entities
    causation_id?:  string           // The event that caused this event
    
    // Actor
    actor_id:    string & !=""       // Who/what triggered this
    actor_type:  "user" | "agent" | "system" | "migration"
    agent_goal_id?: string           // If actor_type == "agent"
    
    // Payload — entity-specific, typed per event_type
    payload: {...}
    
    // What changed — for consumers that need diff semantics
    changed_fields?: [...string]
    previous_values?: {...}
}
```

### 7.2 Event Catalog

Events are defined in `codegen/eventgen.cue` and map directly to state transitions and operations in the ontology:

**Person Events:**
- `PersonCreated`, `PersonUpdated`
- `PersonRoleAssigned`, `PersonRoleDeactivated`, `PersonRoleTerminated`
- `OrganizationCreated`, `OrganizationUpdated`

**Property Events:**
- `PortfolioCreated`, `PortfolioActivated`
- `PropertyCreated`, `PropertyUpdated`, `PropertyStatusChanged`
- `UnitCreated`, `UnitStatusChanged`

**Lease Events:**
- `LeaseCreated`, `LeaseSubmittedForApproval`, `LeaseApproved`
- `LeaseSentForSignature`, `LeaseSigned`, `LeaseActivated`
- `TenantNoticeRecorded`, `LeaseRenewed`, `LeaseTerminated`
- `EvictionInitiated`
- `ApplicationSubmitted`, `ApplicationScreeningComplete`
- `ApplicationApproved`, `ApplicationDenied`

**Accounting Events:**
- `ChargePosted`, `PaymentRecorded`, `CreditApplied`
- `LateFeeAssessed`, `NSFRecorded`, `WriteOffPosted`
- `JournalEntryPosted`, `JournalEntryVoided`
- `ReconciliationCompleted`, `ReconciliationApproved`
- `ManagementFeeCalculated`, `OwnerDistributionProcessed`

### 7.3 Event Consumers

| Consumer | Purpose | Events Consumed |
|---|---|---|
| Graph Sync Worker | Maintain Neo4j projection | All entity create/update events |
| Search Sync Worker | Maintain search index | All entity create/update events |
| Notification Service | Tenant/owner communications | Lease, payment, maintenance events |
| Agent Trigger Service | Start agent goals from events | Configurable per portfolio |
| Audit Log Writer | Compliance audit trail | All events |
| Analytics Pipeline | BI and reporting | All events (async) |

---

## 8. Permission Model — Derived from Relationships

### 8.1 Design

Permissions are derived from the ontological relationship graph, not from flat permission tables. The question "can user X do action Y on entity Z?" is answered by traversing the relationship graph from the user to the target entity.

**Implementation:** OPA (Open Policy Agent) with Rego policies, backed by Ent's privacy framework for database-level enforcement.

### 8.2 Access Paths

Every API operation declares an access path — the graph traversal that determines authorization:

```
Person → (has PersonRole) → PersonRole → (scoped to) → Scope Entity → (contains) → Target Entity
```

Concrete examples:

| Operation | Required Role | Access Path |
|---|---|---|
| View a Property | `viewer` at portfolio+ | `Person → Role(viewer) → Portfolio → contains → Property` |
| Create a Lease | `manager` at property+ | `Person → Role(manager) → Property → contains → Unit` |
| Activate a Lease | `manager` at property+ | `Person → Role(manager) → Property → contains → Unit → has → Lease` |
| Post a Journal Entry | `accountant` at portfolio+ | `Person → Role(accountant) → Portfolio → contains → Property` |
| Approve an Expense | `manager` with approval_limit | `Person → Role(manager, approval_limit >= amount) → Property` |
| Record a Payment | `manager` at property+ | `Person → Role(manager) → Property → ... → Lease → has → Tenant` |

### 8.3 Role Hierarchy

Roles inherit downward through the organizational hierarchy:

```
Organization Admin
  └── Portfolio Admin (all properties in portfolio)
       └── Property Manager (all units in property)
            └── Leasing Agent (leasing operations only)
            └── Maintenance Coordinator (work orders only)
            └── Accountant (financial operations only)
       └── Viewer (read-only across scope)
```

A person with `portfolio_admin` at "West Coast Portfolio" implicitly has `property_manager` at every property in that portfolio. This inheritance is computed from the relationship graph, not stored redundantly.

### 8.4 Agent Authorization

AI agents are authorized through the same PersonRole system as human users. An agent has:

- A `Person` record (type: "system")
- One or more `PersonRole` assignments with explicit scopes
- An `approval_limit` on the `ManagerAttributes` that caps financial authority

The agent cannot exceed its role scope. If the agent's role is scoped to "Sunset Apartments" with a $500 approval limit, it cannot approve a $600 expense — the Ent privacy policy rejects the mutation before it reaches the database.

### 8.5 Ent Privacy Integration

Every Ent schema includes privacy policies generated from the ontology:

```go
// Generated from codegen/authzgen.cue
func (Lease) Policy() ent.Policy {
    return privacy.Policy{
        Query: privacy.QueryPolicy{
            rule.AllowIfAccessPath(
                "person -> role(viewer+) -> portfolio -> property -> unit -> lease",
            ),
        },
        Mutation: privacy.MutationPolicy{
            rule.AllowIfAccessPath(
                "person -> role(manager+) -> portfolio -> property -> unit -> lease",
            ),
        },
    }
}
```

This means every query and mutation automatically filters by authorization. An agent querying leases only sees leases within its role scope. There is no way to bypass this — it's enforced at the ORM layer.

---

## 9. Agent World Model — Ontology as Context

### 9.1 Purpose

The agent world model is a projection of the ontology optimized for LLM consumption. It gives the Propeller agent runtime a complete understanding of what entities exist, how they relate, what actions are valid, and what constraints apply — without the agent needing to "learn" the system through trial and error.

### 9.2 Generated Artifacts

**`gen/agent/propeller-tools.json`** — Anthropic function-calling format tool definitions. One tool per API operation, with:

- Input schemas derived from request specs
- Descriptions written for LLM comprehension
- `when_to_use` guidance for tool selection
- `common_mistakes` to avoid known failure patterns

**`gen/agent/ONTOLOGY.md`** — A markdown document injected into the agent's system prompt that describes:

- All entity types and their key fields
- Relationship graph (what connects to what)
- State machines (valid transitions per entity)
- Business rules (ontological constraints in natural language)
- Financial rules (trust accounting, double-entry requirements)

**`gen/agent/STATE_MACHINES.md`** — Visual state machine diagrams in markdown/mermaid format that the agent can reference when reasoning about status transitions.

### 9.3 Agent Tool Design

Tools map 1:1 to API operations. There is no separate "agent API." The same Connect-RPC services serve both the human UI and the agent.

Key design decisions for agent tools:

1. **Enum constraints reduce hallucination.** Every enum field in the ontology becomes an enum constraint in the tool schema. The agent cannot invent a status value or property type that doesn't exist.

2. **Required fields prevent incomplete operations.** The ontology's constraint system (e.g., "payments must reference a person") becomes required fields in the tool schema. The agent cannot record a payment without specifying who paid.

3. **Side effects in responses enable planning.** When the agent calls `activate_lease` and gets back side effects ("unit status changed," "deposit charge created"), it can factor these into its next action without making follow-up queries.

4. **Common mistakes prevent known failures.** Each tool includes a `common_mistakes` section derived from real agent failure analysis. Example: "Don't set priority to 'emergency' for non-habitability issues" on the work order tool.

### 9.4 Agent Context Assembly

When the agent processes a goal, the context assembler builds a prompt that includes:

```
System: SOUL + IDENTITY (who the agent is, personality, guardrails)
Developer:
  ├── ONTOLOGY.md (filtered to relevant domain area)
  ├── STATE_MACHINES.md (for entities in current workflow)
  ├── TOOLS (filtered to current UI/workflow context)
  └── Active role definition (what the agent can do in this scope)
User:
  ├── Current goal + success criteria
  ├── Entity context (current property, unit, tenant details)
  ├── Structured memory (relevant past interactions)
  └── Session history (sliding window with summarization)
```

The ontology is not dumped wholesale into every prompt. The context assembler selects the relevant slice based on the current goal and workflow state. If the agent is handling a maintenance request, it gets the WorkOrder state machine, vendor tools, and property context — not the accounting ontology.

### 9.5 Ontological Grounding

The most important function of the agent world model: **the agent cannot take actions that violate the ontology.** This is enforced at multiple layers:

| Layer | Enforcement | Example |
|---|---|---|
| Tool schema | Invalid inputs rejected before API call | Agent can't pass "super_urgent" as a priority |
| API validation | Request validation against proto/CUE | Agent can't create a lease without a unit_id |
| Ent hooks | State machine + cross-field constraints | Agent can't activate a lease that's still in draft |
| Ent privacy | Authorization policies per entity | Agent can't access properties outside its scope |
| Database | NOT NULL, CHECK, FK constraints | Data integrity even if all other layers fail |

Five layers of defense, all derived from one CUE ontology.

---

## 10. Codegen Pipeline

### 10.1 Pipeline Overview

```bash
# Validate the ontology is internally consistent
make validate
# → cue vet ontology/...

# Generate everything
make generate
# → cmd/entgen    (CUE → Ent schemas)
# → ent generate  (Ent schemas → Go CRUD code + migrations)
# → cmd/apigen    (CUE → Proto services + OpenAPI + agent tools + handler scaffolds)
# → buf generate  (Proto → Go server/client + TS SDK)
# → cmd/eventgen  (CUE → Event schemas for NATS schema registry)
# → cmd/authzgen  (CUE → OPA policy scaffolds)
# → cmd/agentgen  (CUE → ONTOLOGY.md + STATE_MACHINES.md + TOOLS.md)

# CI check: fail if generated code is stale
make ci-check
# → Regenerate everything, fail if git diff shows changes
```

### 10.2 What Is Generated vs Hand-Written

| Artifact | Generated | Hand-Written |
|---|---|---|
| CUE ontology definitions | — | ✅ Source of truth |
| CUE codegen mappings | — | ✅ Maps ontology → implementation |
| Ent schemas | ✅ | — |
| Ent-generated Go code | ✅ | — |
| Database migrations | ✅ | — |
| Proto service definitions | ✅ | — |
| Go gRPC/Connect stubs | ✅ | — |
| TypeScript client SDK | ✅ | — |
| OpenAPI specification | ✅ | — |
| Connect handler scaffolds | ✅ | — |
| Handler business logic | — | ✅ Within generated scaffolds |
| Event payload schemas | ✅ | — |
| OPA policy scaffolds | ✅ | — |
| OPA policy rules | — | ✅ Complex authorization logic |
| Agent tool definitions | ✅ | — |
| Agent ONTOLOGY.md | ✅ | — |
| Agent system prompt | — | ✅ Personality and guardrails |
| State machine hooks | ✅ | — |
| Cross-field validation hooks | ⚠️ Scaffolds generated | ✅ Complex validation logic |

### 10.3 The No-Drift Guarantee

The CI pipeline enforces that generated code matches the ontology:

1. On every PR, `make ci-check` regenerates all artifacts
2. If any generated file differs from what's committed, the build fails
3. The error message says: "Generated code is out of date. Run `make generate` and commit."
4. This prevents ontological drift — no one can change a generated file without changing the ontology

---

## 11. Implementation Sequence

### Phase 1: Foundation (Weeks 1-2)

- [ ] Set up CUE toolchain and project structure
- [ ] Define `common.cue` with shared types
- [ ] Build `cmd/entgen` — CUE to Ent schema generator
- [ ] Validate with 2-3 simple entities (Property, Unit)
- [ ] Verify round-trip: CUE → Ent schema → `ent generate` → working Go code → Postgres

### Phase 2: Core Ontology (Weeks 2-4)

- [ ] Complete all four domain models in CUE (Person, Property, Lease, Accounting)
- [ ] Define all relationships in `relationships.cue`
- [ ] Define all state machines in `state_machines.cue`
- [ ] Run `cue vet` — resolve all internal inconsistencies
- [ ] Generate full Ent schema set, verify migrations run clean

### Phase 3: API Layer (Weeks 4-6)

- [ ] Build `cmd/apigen` — CUE to Connect-RPC generator
- [ ] Define API services in `codegen/apigen.cue`
- [ ] Generate proto services, run `buf generate`
- [ ] Implement handler business logic for PropertyService and LeaseService
- [ ] Verify end-to-end: client → Connect-RPC → handler → Ent → Postgres

### Phase 4: Events + Projections (Weeks 6-8)

- [ ] Build `cmd/eventgen` — CUE to event schema generator
- [ ] Set up NATS JetStream with schema registry
- [ ] Implement Ent hooks for event emission
- [ ] Build graph sync worker (NATS → Neo4j)
- [ ] Build search sync worker (NATS → Meilisearch)
- [ ] Verify: mutation → event → graph updated + search updated

### Phase 5: Permissions (Weeks 8-10)

- [ ] Build `cmd/authzgen` — CUE to OPA policy generator
- [ ] Implement Ent privacy policies from generated scaffolds
- [ ] Implement OPA rules for access path evaluation
- [ ] Integrate with PersonRole system
- [ ] Verify: unauthorized access blocked at every layer

### Phase 6: Agent Integration (Weeks 10-12)

- [ ] Build `cmd/agentgen` — CUE to agent tool definition generator
- [ ] Generate ONTOLOGY.md, STATE_MACHINES.md, TOOLS.md
- [ ] Integrate generated tools with Propeller agent runtime
- [ ] Test agent goal execution against the live system
- [ ] Verify: agent constrained by ontology at every layer

### Phase 7: Remaining Modules (Weeks 12+)

- [ ] Extend ontology for remaining ~86 modules
- [ ] Each module follows the same pattern: CUE → generated → implement handlers
- [ ] Parallel implementation by multiple agents/teams — ontology ensures coherence

---

## Appendix A: Glossary

| Term | Definition |
|---|---|
| **Ontology** | The canonical domain model defined in CUE that describes what entities exist, how they relate, and what constraints apply |
| **Projection** | A representation of the ontology optimized for a specific purpose (Postgres for transactions, Neo4j for relationships, Meilisearch for search) |
| **Generative kernel** | The CUE ontology as the single source from which all other representations are generated |
| **State machine** | The set of valid status transitions for an entity, defined in CUE and enforced by Ent hooks |
| **Access path** | A graph traversal from a user to a target entity that determines authorization |
| **PersonRole** | The relationship entity that connects a Person to a scoped entity (portfolio, property, etc.) in a specific capacity (manager, tenant, etc.) |
| **Side effect** | A downstream change caused by an operation, reported in the API response so agents and humans understand the full consequence |
| **Ontological drift** | When multiple representations of the same concept diverge because they're maintained independently — what this architecture prevents |

## Appendix B: Technology Stack

| Component | Technology | Version |
|---|---|---|
| Ontology definition | CUE | 0.8+ |
| Persistence framework | Ent (entgo.io) | 0.13+ |
| Database | PostgreSQL | 16+ |
| Graph database | Neo4j | 5+ |
| Search engine | Meilisearch | 1.6+ |
| API framework | Connect-RPC | 1.0+ |
| Schema registry | Buf BSR | — |
| Message bus | NATS JetStream | 2.10+ |
| Authorization | OPA (Open Policy Agent) | 0.62+ |
| Language | Go | 1.22+ |
| CI codegen check | Make + Git | — |

## Appendix C: Referenced Documents

- Propeller Agent Runtime Specification
- Propeller RBAC System v2 Specification
- Propeller Accounting Modules (FIN-001 through FIN-017)
- Propeller Leasing Service Specification
- Propeller Rules Engine Specification
- "The Rebuild Window" — Executive Presentation