# Propeller Ontology Specification v3.1

**Version:** 3.1  
**Date:** February 27, 2026  
**Author:** Matthew Baird, CTO — AppFolio  
**Status:** For Claude Code Implementation  
**Revision note:** Architectural reframe per chief architect review. Ontology as shared vocabulary and constraint system, not contract generator. Added CQRS commands, domain events, external API anti-corruption layer, business-defined permissions. Comprehensions reframed as scaffolding + drift detection.

---

## 1. Purpose

This document defines the domain model for Propeller as a CUE ontology. The ontology is a **shared vocabulary and constraint system** — it defines what things ARE, not what you can DO to them, what you TELL the outside world happened, or who is ALLOWED to do what.

Commands, external APIs, domain events, and permission policies are **separately defined** concerns that **reference** the ontology for field validation, type reuse, and drift detection. They are not mechanically derived from it. Each boundary has its own shape, its own evolution pace, and its own authors.

**This version uses real CUE syntax.** Every entity definition is valid CUE. Pseudocode is gone. `cue vet` validates the entire domain model and cross-references every command, event, and API contract that imports ontology types.

### 1.1 CUE Features Used

- **Closed structs** (`close()`) — prevent unexpected fields on every entity
- **Embedding** — `#BaseEntity`, `#StatefulEntity` compose shared behavior
- **Conditional constraints** (`if`) — business rules enforced at compile time
- **Defaults** (`*`) — self-documenting, validated against constraints
- **Pattern constraints** (`=~`) — format validation on strings
- **Hidden fields** (`_`) — generator metadata that doesn't leak into data schema
- **Comprehensions** — scaffold starter templates, generate test matrices, detect drift
- **Unification** (`&`) — compose jurisdiction constraints (most restrictive wins)
- **Cross-package imports** — commands, events, and API contracts import ontology types for validation without tight coupling

### 1.2 Architecture

The ontology sits at the center but does not generate everything. Different concerns have different coupling tightness:

```
TIGHT COUPLING (1:1 derivation):
  Ontology → Data store schemas (Ent → Postgres)
  Ontology → State machine enforcement (Ent hooks)
  Ontology → Agent world model vocabulary (ONTOLOGY.md)
  Ontology → Test matrices (from constraints + state machines)

SHARED VOCABULARY (references ontology types, defines own shapes):
  Commands (CQRS)     — import ontology types for field validation
  Domain Events       — import ontology types, carry projections not full entities
  External API (v1)   — import ontology enums, define own response/request shapes
  Permission Policies — reference domain entities for scope, define own groups
  View Definitions    — reference ontology fields, define own layout

LOOSE COUPLING (informed by ontology, not derived):
  Read models / projections — optimized for query patterns
  Search indices            — subset fields, denormalized
  Graph projections         — relationship-focused
```

**Golden rule**: The ontology defines what a Lease IS. A command defines what "Move In Tenant" DOES (touching Lease, Space, PersonRole, and LedgerEntry simultaneously). An API response defines what a consumer SEES. An event defines what the rest of the system HEARS. These are different concerns with different authors, different evolution speeds, and different shapes. They share vocabulary. They do not share structure.

**Anti-Zuora rule**: Internal domain model changes must not automatically break external API consumers. The external API is a versioned contract with an anti-corruption layer. Internal field renames, type restructurings, and model evolution happen behind that boundary.

---

## 2. File Structure

```
propeller/
├── ontology/                        # DOMAIN TRUTH: What things ARE
│   ├── common.cue                   #   Shared types: Money, Address, DateRange, AuditMetadata
│   ├── base.cue                     #   Base entity types: #BaseEntity, #StatefulEntity
│   ├── person.cue                   #   Person, Organization, PersonRole
│   ├── property.cue                 #   Portfolio, Property, Building, Space
│   ├── jurisdiction.cue             #   Jurisdiction, PropertyJurisdiction, JurisdictionRule
│   ├── lease.cue                    #   Lease, LeaseSpace, Application, commercial structures
│   ├── accounting.cue               #   Account, LedgerEntry, JournalEntry, BankAccount
│   ├── config_schema.cue            #   Configuration schema — tunable values and ranges
│   ├── relationships.cue            #   All cross-model edges with cardinality and semantics
│   └── state_machines.cue           #   All entity state machines
│
├── commands/                        # ACTIONS: What you can DO (CQRS write side)
│   ├── lease_commands.cue           #   MoveInTenant, RecordPayment, RenewLease, etc.
│   ├── property_commands.cue        #   OnboardProperty, TransferProperty, etc.
│   ├── accounting_commands.cue      #   PostJournalEntry, RecordPayment, Reconcile, etc.
│   └── application_commands.cue     #   SubmitApplication, ApproveApplication, etc.
│
├── events/                          # NOTIFICATIONS: What HAPPENED (domain events)
│   ├── lease_events.cue             #   TenantMovedIn, RentIncreaseApplied, LeaseRenewed, etc.
│   ├── property_events.cue          #   PropertyOnboarded, SpaceStatusChanged, etc.
│   ├── accounting_events.cue        #   PaymentReceived, JournalEntryPosted, etc.
│   └── jurisdiction_events.cue      #   JurisdictionRuleActivated, ComplianceAlertTriggered, etc.
│
├── api/                             # EXTERNAL CONTRACTS: What the outside world SEES
│   └── v1/
│       ├── lease_api.cue            #   Versioned request/response shapes
│       ├── property_api.cue
│       ├── person_api.cue
│       ├── accounting_api.cue
│       └── common_api.cue           #   Shared API types (pagination, error envelope)
│
├── policies/                        # AUTHORIZATION: Who can do WHAT
│   ├── permission_groups.cue        #   Business-defined role groups
│   ├── command_permissions.cue      #   Which groups can execute which commands
│   └── field_policies.cue           #   Per-attribute visibility rules
│
├── codegen/                         # GENERATION + VALIDATION
│   ├── entgen.cue                   #   Ontology → Ent schema mapping (tight coupling, 1:1)
│   ├── testgen.cue                  #   Constraints + state machines → test matrices
│   ├── agentgen.cue                 #   Ontology → ONTOLOGY.md + TOOLS.md for agent context
│   ├── uigen.cue                    #   View definitions per entity (references ontology fields)
│   └── drift.cue                    #   Cross-reference validation (see Section 12)
│
├── cmd/
│   ├── entgen/main.go               #   CUE → Ent schema generator
│   ├── testgen/main.go              #   CUE → test case generator
│   ├── agentgen/main.go             #   CUE → ONTOLOGY.md + STATE_MACHINES.md + TOOLS.md
│   ├── uigen/main.go                #   CUE → UI schema generator
│   ├── uirender/main.go             #   UI schema → Svelte components
│   └── driftcheck/main.go           #   Validate cross-boundary references
└── Makefile
```

### Boundary Responsibilities

| Directory | Author | Evolution Speed | Coupling to Ontology |
|---|---|---|---|
| `ontology/` | Domain modelers | Slow (schema changes) | IS the ontology |
| `commands/` | Domain experts + engineers | Medium (new actions, payload changes) | Imports types for field validation |
| `events/` | Domain experts + engineers | Medium (new events, payload changes) | Imports types for projections |
| `api/v1/` | API designers + product | Slow (versioned, customer-facing) | Imports enums, defines own shapes |
| `policies/` | Business stakeholders + security | Medium (new roles, policy changes) | References entities for scope |
| `codegen/` | Platform engineers | Follows ontology | Tightly coupled by design |

---

## 3. Base Entity Types — `ontology/base.cue`

Every entity in the system embeds one of these base types. Cross-cutting concerns live here once, not repeated per entity.

```cue
package ontology

import "time"

// #BaseEntity provides id and audit metadata to every entity.
// Embed this in every entity definition.
#BaseEntity: {
    id: string & =~"^[a-zA-Z0-9_-]{20,36}$"
    audit: #AuditMetadata

    // Domain-level attributes readable by all generators
    @immutable(id)           // id cannot change after creation
    @computed(audit)         // audit fields are system-managed
}

// #StatefulEntity adds a status field with state machine enforcement.
// Embed this in entities that have lifecycle states.
#StatefulEntity: {
    #BaseEntity
    status: string           // refined to specific enum by each entity

    _has_state_machine: true // generators check this to wire up transitions
}

// #ImmutableEntity cannot be updated or deleted after creation.
// Used for ledger entries and audit logs.
#ImmutableEntity: {
    #BaseEntity

    _immutable: true         // Ent hook: reject all Update/Delete mutations
}
```

---

## 4. Common Types — `ontology/common.cue`

Shared types that establish the foundational vocabulary. All use `close()` to prevent unexpected fields.

```cue
package ontology

import "time"

// === Monetary ===
// CRITICAL: All financials use integer cents. No floating point anywhere in the financial chain.

#Money: close({
    amount_cents: int
    currency:     string & =~"^[A-Z]{3}$" | *"USD"  // ISO 4217
})

#NonNegativeMoney: #Money & close({
    amount_cents: >=0
})

#PositiveMoney: #Money & close({
    amount_cents: >0
})


// === Temporal ===

#DateRange: close({
    start: time.Time
    end?:  time.Time

    // CUE enforces: if end is set, it must be after start
    if end != _|_ {
        end: >start
    }
})


// === Geographic ===

#Address: close({
    line1:       string & strings.MinRunes(1)
    line2?:      string
    city:        string & strings.MinRunes(1)
    state:       string & =~"^[A-Z]{2}$"
    postal_code: string & =~"^[0-9]{5}(-[0-9]{4})?$"
    country:     string & =~"^[A-Z]{2}$" | *"US"     // ISO 3166-1 alpha-2
    latitude?:   float & >=-90 & <=90
    longitude?:  float & >=-180 & <=180
    county?:     string                                // important for tax jurisdictions
})


// === Identity and References ===

#EntityType: "person" | "organization" | "portfolio" | "property" | "building" |
             "space" | "lease" | "lease_space" | "work_order" | "vendor" |
             "ledger_entry" | "journal_entry" | "account" | "bank_account" |
             "application" | "jurisdiction" | "jurisdiction_rule" | "document"

#RelationshipType: "belongs_to" | "contains" | "managed_by" | "owned_by" |
                   "leased_to" | "occupied_by" | "reported_by" | "assigned_to" |
                   "billed_to" | "paid_by" | "performed_by" | "approved_by" |
                   "guarantor_for" | "emergency_contact_for" | "employed_by" |
                   "related_to" | "parent_of" | "child_of" | "sublease_of"

#EntityRef: close({
    entity_type:  #EntityType
    entity_id:    string
    relationship: #RelationshipType
})


// === Audit ===

#AuditSource: "user" | "agent" | "import" | "system" | "migration"

#AuditMetadata: close({
    created_by:     string                   // user ID, agent ID, or "system"
    updated_by:     string
    created_at:     time.Time
    updated_at:     time.Time
    source:         #AuditSource | *"user"
    correlation_id?: string                  // links related changes across entities
    agent_goal_id?:  string                  // if source == "agent", which goal triggered this

    if source != "agent" {
        agent_goal_id: null
    }
})


// === Contact ===

#ContactType: "email" | "phone" | "sms" | "mail" | "portal"

#ContactMethod: close({
    type:     #ContactType
    value:    string & strings.MinRunes(1)
    primary:  *false | bool
    verified: *false | bool
    opt_out:  *false | bool                  // communication preference
    label?:   string                         // "work", "home", "mobile"

    // Email format validation
    if type == "email" {
        value: =~"^[a-zA-Z0-9._%+\\-]+@[a-zA-Z0-9.\\-]+\\.[a-zA-Z]{2,}$"
    }

    // Phone: digits, spaces, dashes, parens, plus sign
    if type == "phone" || type == "sms" {
        value: =~"^[+]?[0-9() \\-]{7,20}$"
    }
})
```

---

## 5. Person Model — `ontology/person.cue`

A single person can be a tenant, owner, vendor contact, and emergency contact simultaneously. Roles are relationships, not types.

### 5.1 Person

```cue
#Person: close({
    #BaseEntity

    first_name:   string & strings.MinRunes(1)
    last_name:    string & strings.MinRunes(1)
    middle_name?: string
    display_name: string | *"\(first_name) \(last_name)"

    date_of_birth?:   time.Time                        // required for tenant screening
    ssn_last_four?:   string & =~"^[0-9]{4}$"         // stored encrypted @sensitive

    contact_methods:    [...#ContactMethod] & list.MinItems(1)
    preferred_contact:  #ContactType | *"email"

    language_preference: string & =~"^[a-z]{2}$" | *"en"   // ISO 639-1
    timezone?:           string                              // IANA timezone
    do_not_contact:      *false | bool                       // legal hold, agent must respect

    identity_verified:     *false | bool
    verification_method?:  "manual" | "id_check" | "credit_check" | "ssn_verify"
    verified_at?:          time.Time

    tags?: [...string]

    source: "user" | "applicant" | "import" | "system" | *"user"

    // If preferred contact is phone/sms, must have a phone/sms contact method
    if preferred_contact == "sms" || preferred_contact == "phone" {
        _has_phone_contact: true  // validated at Ent hook: >=1 contact_method with type phone|sms
    }

    // Hidden: generator metadata
    _display_template: "{first_name} {last_name}"
})
```

### 5.2 Organization

```cue
#OrgType: "management_company" | "ownership_entity" | "vendor" |
          "corporate_tenant" | "government_agency" | "hoa" |
          "investment_fund" | "other"

#Organization: close({
    #StatefulEntity

    legal_name: string & strings.MinRunes(1)
    dba_name?:  string                                  // "Doing Business As"

    org_type: #OrgType

    tax_id?:      string                                // EIN/Tax ID @sensitive
    tax_id_type?: "ein" | "ssn" | "itin" | "foreign"

    status: "active" | "inactive" | "suspended" | "dissolved"

    address?:          #Address
    contact_methods?:  [...#ContactMethod]

    state_of_incorporation?: string & =~"^[A-Z]{2}$"
    formation_date?:         time.Time

    management_license?: string                         // for management companies
    license_state?:      string & =~"^[A-Z]{2}$"
    license_expiry?:     time.Time

    _display_template: "{legal_name}"
})
```

### 5.3 PersonRole

Roles are relationships between a Person and other entities. A PersonRole captures context-specific attributes that apply when a Person acts in a particular capacity.

```cue
#RoleType: "tenant" | "owner" | "property_manager" | "maintenance_tech" |
           "leasing_agent" | "accountant" | "vendor_contact" |
           "guarantor" | "emergency_contact" | "authorized_occupant" |
           "co_signer"

#ScopeType: "organization" | "portfolio" | "property" | "building" | "space" | "lease"

#PersonRole: close({
    #StatefulEntity

    person_id: string
    role_type: #RoleType
    scope_type: #ScopeType
    scope_id:   string

    status: "pending" | "active" | "inactive" | "terminated"

    effective: #DateRange

    // Attributes are role-specific, determined by role_type
    attributes?: #TenantAttributes | #OwnerAttributes |
                 #ManagerAttributes | #GuarantorAttributes

    // Type-safe: tenant roles must have tenant attributes
    if role_type == "tenant" && attributes != _|_ {
        attributes: #TenantAttributes
    }
    if role_type == "owner" && attributes != _|_ {
        attributes: #OwnerAttributes
    }
    if role_type == "property_manager" && attributes != _|_ {
        attributes: #ManagerAttributes
    }
    if role_type == "guarantor" && attributes != _|_ {
        attributes: #GuarantorAttributes
    }
})


// === Role-Specific Attributes ===

#TenantAttributes: close({
    _type:             "tenant"
    standing:          *"good" | "late" | "collections" | "eviction"
    occupancy_status:  *"occupying" | "vacated" | "never_occupied"
    liability_status:  *"active" | "released" | "guarantor_only"
    screening_status:  "not_started" | "in_progress" | "approved" | "denied" | "conditional"
    screening_date?:   time.Time
    current_balance?:  #Money                          // computed from ledger @computed
    move_in_date?:     time.Time
    move_out_date?:    time.Time
    pet_count?:        int & >=0
    vehicle_count?:    int & >=0
})

#OwnerAttributes: close({
    _type:                  "owner"
    ownership_percent:      float & >0 & <=100
    distribution_method:    *"ach" | "check" | "hold"
    management_fee_percent?: float & >=0 & <=100
    tax_reporting:          *"1099" | "k1" | "none"
    reserve_amount?:        #NonNegativeMoney
})

#ManagerAttributes: close({
    _type:               "manager"
    license_number?:     string
    license_state?:      string & =~"^[A-Z]{2}$"
    approval_limit?:     #NonNegativeMoney             // max they can approve without escalation
    can_sign_leases:     *false | bool
    can_approve_expenses: *true | bool
})

#GuarantorAttributes: close({
    _type:            "guarantor"
    guarantee_type:   "full" | "partial" | "conditional"
    guarantee_amount?: #PositiveMoney                  // for partial guarantees
    guarantee_term?:  #DateRange
    credit_score?:    int & >=300 & <=850
})
```

IMPORTANT: The `occupancy_status` and `liability_status` on TenantAttributes handle the roommate-departure scenario. A tenant who moves out but remains legally liable has `occupancy_status: "vacated"` and `liability_status: "active"`.

---

## 6. Property Model — `ontology/property.cue`

### CRITICAL DESIGN DECISION: Building is Optional, Space Replaces Unit

```
Property Hierarchy:
  Portfolio → Property → Building (OPTIONAL) → Space (self-referential) → Lease (M2M via LeaseSpace)
```

- Building is an optional grouping, not a mandatory layer. Parking lots, mobile home parks, single-family homes have no buildings.
- Space replaces "Unit" as the universal term. Apartments, offices, parking spots, storage, bed-spaces, lots, desks — all are Spaces.
- Space is self-referential (`parent_space_id`) for apartments→bedrooms, food courts→stalls.
- Space has `leasable` flag. In by-the-bed configs, parent Space is non-leasable.
- Floor is an attribute, not a hierarchy level.

### 6.1 Portfolio

```cue
#PortfolioStatus: "onboarding" | "active" | "inactive" | "offboarding"

#Portfolio: close({
    #StatefulEntity

    name:      string & strings.MinRunes(1)
    owner_id:  string                                  // Organization ID

    status: #PortfolioStatus

    description?: string

    // Financial defaults (can be overridden at property level)
    default_chart_of_accounts_id?: string
    default_bank_account_id?:      string

    _display_template: "{name}"
})
```

### 6.2 Property

```cue
#PropertyType: "single_family" | "multi_family" | "commercial_office" |
               "commercial_retail" | "mixed_use" | "industrial" |
               "affordable_housing" | "student_housing" | "senior_living" |
               "vacation_rental" | "mobile_home_park" | "self_storage" |
               "coworking" | "data_center" | "medical_office"

#PropertyStatus: "onboarding" | "active" | "inactive" | "under_renovation" | "for_sale"

#Property: close({
    #StatefulEntity

    name:         string & strings.MinRunes(1)
    portfolio_id: string
    address:      #Address

    property_type: #PropertyType
    status:        #PropertyStatus

    year_built:            int & >=1800 & <=2030
    total_square_footage:  float & >0
    total_spaces:          int & >=1                    // denormalized count
    lot_size_sqft?:        float & >0
    stories?:              int & >=1
    parking_spaces?:       int & >=0

    // Jurisdiction: M2M via PropertyJurisdiction (see Section 8)
    // rent_controlled and requires_lead_disclosure are DERIVED from jurisdiction rules

    compliance_programs?: [...("LIHTC" | "Section8" | "HUD" | "HOME" | "RAD" | "VASH" | "PBV")]

    chart_of_accounts_id?: string                      // override portfolio default
    bank_account_id?:      string

    insurance_policy_number?: string
    insurance_expiry?:        time.Time

    // --- Conditional constraints ---

    if property_type == "single_family" {
        total_spaces: 1
    }

    if property_type == "affordable_housing" {
        compliance_programs: list.MinItems(1)
    }

    // Lead paint: federal rule, derived from jurisdiction, but also enforceable here
    if year_built < 1978 {
        _requires_lead_disclosure: true
    }

    _display_template: "{name}"
})
```

### 6.3 Building

Building is an OPTIONAL grouping entity. Not all spaces belong to a building.

```cue
#BuildingType: "residential" | "commercial" | "mixed_use" |
               "parking_structure" | "industrial" | "storage" | "auxiliary"

#BuildingStatus: "active" | "inactive" | "under_renovation"

#Building: close({
    #StatefulEntity

    property_id: string
    name:        string & strings.MinRunes(1)           // "Building A", "Main Tower", etc.

    building_type: #BuildingType
    status:        #BuildingStatus

    address?: #Address                                  // may differ from property

    floors?:                        int & >=1
    year_built?:                    int & >=1800 & <=2030
    total_square_footage?:          float & >0
    total_rentable_square_footage?: float & >0          // for CAM calculations

    _display_template: "{name}"
})
```

### 6.4 Space

Space is the universal entity for any rentable or non-rentable location within a property.

```cue
#SpaceType: "apartment" | "suite" | "office" | "retail" | "industrial" |
            "warehouse" | "parking_spot" | "parking_garage" | "storage_unit" |
            "bed_space" | "desk_space" | "private_office" | "common_area" |
            "amenity" | "lot_pad" | "rack" | "cage" | "server_room" | "other"

#SpaceStatus: "vacant" | "occupied" | "notice_given" | "make_ready" |
              "down" | "model" | "reserved" | "owner_occupied"

#Property: close({  // NOTE: This block extends the Property above via CUE unification
})

#Space: close({
    #StatefulEntity

    property_id:     string                             // always set
    building_id?:    string                             // only if in a building
    parent_space_id?: string                            // self-referential: apt→bedrooms

    space_number:  string & strings.MinRunes(1)         // "101", "A-204", "P-15"
    space_type:    #SpaceType
    status:        #SpaceStatus

    floor?:         int                                 // attribute, not hierarchy
    square_footage?: float & >0
    bedrooms?:      int & >=0
    bathrooms?:     float & >=0 & <=20                  // 1.5, 2.0, etc.

    leasable: *true | bool                              // false for common areas, parent in by-the-bed

    market_rent?: #NonNegativeMoney                     // current asking rent
    amenities?:   [...string]                           // "washer_dryer", "balcony", "parking"
    specialized_infrastructure?: [...string]            // "medical_gas", "3_phase_power", "grease_trap"

    _display_template: "Space {space_number}"
})
```

---

## 7. Lease Model — `ontology/lease.cue`

The Lease is the central contractual entity. It connects one or more Spaces to one or more Persons via LeaseSpace (M2M join entity).

### CRITICAL DESIGN DECISIONS

1. **Lease ↔ Space is M2M via LeaseSpace.** A commercial tenant can lease multiple suites. A residential lease can include a parking spot.
2. **LeaseSpace is a first-class entity**, not a join table. It carries effective dates, relationship types, and optional per-space data.
3. **Lease has NO space_id or tenant_id.** All connections go through LeaseSpace (for spaces) and PersonRole (for tenants).

### 7.1 Lease

This is where CUE conditional constraints shine. Every business rule is a `if` block. `cue vet` validates them. `cmd/entgen` generates Ent hooks from them. `cmd/testgen` generates both positive and negative tests from them.

```cue
#LeaseType: "fixed_term" | "month_to_month" |
            "commercial_nnn" | "commercial_nn" | "commercial_n" |
            "commercial_gross" | "commercial_modified_gross" |
            "affordable" | "section_8" | "student" |
            "ground_lease" | "short_term" | "membership"

#LeaseStatus: "draft" | "pending_approval" | "pending_signature" | "active" |
              "expired" | "month_to_month_holdover" | "renewed" |
              "terminated" | "eviction"

#LiabilityType: "joint_and_several" | "several" | "individual"

// Groups of lease types for conditional logic
_commercial_types: ["commercial_nnn", "commercial_nn", "commercial_n",
                    "commercial_gross", "commercial_modified_gross"]

_net_lease_types: ["commercial_nnn", "commercial_nn", "commercial_n"]

#Lease: close({
    #StatefulEntity

    lease_type:     #LeaseType
    liability_type: #LiabilityType | *"joint_and_several"
    status:         #LeaseStatus

    property_id: string                                @immutable

    // Financial
    base_rent:        #NonNegativeMoney
    security_deposit: #NonNegativeMoney

    // Term
    term:                    #DateRange
    lease_commencement_date: time.Time
    rent_commencement_date?: time.Time                 // may differ for free rent periods
    notice_required_days:    int & >=0 & <=365 | *30
    move_in_date?:           time.Time
    move_out_date?:          time.Time

    // Flags
    is_sublease:     *false | bool
    parent_lease_id?: string                           // if is_sublease
    sublease_billing?: "direct" | "pass_through"

    // Commercial structures (all optional, conditionally required)
    cam_terms?:              #CAMTerms
    percentage_rent?:        #PercentageRent
    tenant_improvement?:     #TenantImprovement
    rent_schedule?:          [...#RentScheduleEntry]
    recurring_charges?:      [...#RecurringCharge]
    usage_charges?:          [...#UsageBasedCharge]
    renewal_options?:        [...#RenewalOption]
    expansion_rights?:       [...#ExpansionRight]
    contraction_rights?:     [...#ContractionRight]
    late_fee_policy?:        #LateFeePolicy

    // Affordable housing
    subsidy?: #SubsidyTerms

    // Short-term rental
    check_in_time?:        string                      // "15:00"
    check_out_time?:       string                      // "11:00"
    cleaning_fee?:         #NonNegativeMoney
    platform_booking_id?:  string

    // Membership (coworking)
    membership_tier?: string

    // Signing
    signing_method?: "wet_ink" | "e_sign" | "docusign" | "hellosign"
    signed_at?:      time.Time
    document_id?:    string

    // ===================================================================
    // CONDITIONAL CONSTRAINTS
    // Every business rule expressed as a CUE conditional.
    // These generate: Ent hooks, API validation, test cases, UI visibility.
    // ===================================================================

    // Fixed-term and student leases MUST have an end date
    if lease_type == "fixed_term" || lease_type == "student" {
        term: close({
            start: time.Time
            end:   time.Time       // required (not optional) for these types
            end:   >start
        })
    }

    // All commercial leases require CAM terms
    if list.Contains(_commercial_types, lease_type) {
        cam_terms: #CAMTerms
    }

    // NNN: landlord passes through ALL three operating costs
    if lease_type == "commercial_nnn" {
        cam_terms: includes_property_tax: true
        cam_terms: includes_insurance:    true
        cam_terms: includes_utilities:    true
    }

    // NN: property tax + insurance (tenant pays), landlord covers utilities
    if lease_type == "commercial_nn" {
        cam_terms: includes_property_tax: true
        cam_terms: includes_insurance:    true
    }

    // N: tenant pays property tax only
    if lease_type == "commercial_n" {
        cam_terms: includes_property_tax: true
    }

    // Section 8 requires subsidy terms
    if lease_type == "section_8" {
        subsidy: #SubsidyTerms
    }

    // Active leases must have move-in date
    if status == "active" {
        move_in_date: time.Time
    }

    // Post-signing statuses require signed_at
    if status == "active" || status == "expired" || status == "renewed" {
        signed_at: time.Time
    }

    // Subleases require parent lease
    if is_sublease == true {
        parent_lease_id: string
    }

    // Rent commencement cannot be before lease commencement
    if rent_commencement_date != _|_ {
        rent_commencement_date: >=lease_commencement_date
    }

    // Hidden: generator metadata
    _display_template: "{property.name} — {lease_type}"

    // Hidden: test generation hints
    _test_scenarios: {
        cross_field: [
            "NNN requires all three CAM flags",
            "fixed_term requires end date",
            "section_8 requires subsidy",
            "sublease requires parent_lease_id",
            "active requires move_in_date",
        ]
    }
})
```

### 7.2 LeaseSpace (First-Class Join Entity)

```cue
#LeaseSpaceRelationship: "primary" | "expansion" | "sublease" | "shared_access" |
                         "parking" | "storage" | "loading_dock" | "rooftop" |
                         "patio" | "signage" | "included" | "membership"

#LeaseSpace: close({
    #BaseEntity

    lease_id: string                                   @immutable
    space_id: string                                   @immutable

    is_primary:   *true | bool
    relationship: #LeaseSpaceRelationship

    effective: #DateRange                              // when this space was part of this lease

    square_footage_leased?: float & >0                 // partial-floor commercial
})
```

### 7.3 Commercial Lease Structures

```cue
// --- Rent Schedule and Adjustments ---

#RentAdjustmentMethod: "cpi" | "fixed_percent" | "fixed_amount_increase" | "market_review"

#RentScheduleEntry: close({
    effective_period: #DateRange
    description:      string                           // "Year 1", "Move-in concession"
    charge_code:      string                           // links to chart of accounts

    // Exactly one of these two:
    fixed_amount?: #NonNegativeMoney
    adjustment?:   #RentAdjustment
})

#RentAdjustment: close({
    method:      #RentAdjustmentMethod
    base_amount: #NonNegativeMoney

    // For CPI
    cpi_index?:   "CPI-U" | "CPI-W" | "regional"
    cpi_floor?:   float & >=0                          // minimum annual increase
    cpi_ceiling?: float & >0                           // maximum annual increase

    // For fixed percent
    percent_increase?: float & >0

    // For fixed amount
    amount_increase?: #PositiveMoney

    // For market review
    market_review_mechanism?: string

    // Conditional: CPI fields required for CPI method
    if method == "cpi" {
        cpi_index: "CPI-U" | "CPI-W" | "regional"
    }
    if method == "fixed_percent" {
        percent_increase: float & >0
    }
    if method == "fixed_amount_increase" {
        amount_increase: #PositiveMoney
    }
})


// --- Recurring Charges ---

#ChargeFrequency: "monthly" | "quarterly" | "annually" | "one_time"

#RecurringCharge: close({
    #BaseEntity
    charge_code: string
    description: string
    amount:      #NonNegativeMoney
    frequency:   #ChargeFrequency
    effective_period: #DateRange
    taxable:     *false | bool
    space_id?:   string                                // per-space rates in multi-space leases
})


// --- Usage-Based Charges ---

#UnitOfMeasure: "kwh" | "gallon" | "cubic_foot" | "therm" | "hour" | "gb"

#UsageBasedCharge: close({
    #BaseEntity
    charge_code:      string
    description:      string
    unit_of_measure:  #UnitOfMeasure
    rate_per_unit:    #PositiveMoney
    meter_id?:        string
    billing_frequency: "monthly" | "quarterly"
    cap?:             #NonNegativeMoney                // max charge per period
    space_id?:        string
})


// --- Late Fee Policy ---

#LateFeeType: "flat" | "percent" | "per_day" | "tiered"

#LateFeePolicy: close({
    grace_period_days: int & >=0 & <=30 | *5
    fee_type:          #LateFeeType
    flat_amount?:      #NonNegativeMoney
    percent?:          float & >0 & <=100
    per_day_amount?:   #NonNegativeMoney
    max_fee?:          #NonNegativeMoney
    tiers?:            [...close({
        days_late_min: int & >=0
        days_late_max: int
        amount:        #NonNegativeMoney
    })]

    // Conditional: fee details match fee type
    if fee_type == "flat"    { flat_amount:    #NonNegativeMoney }
    if fee_type == "percent" { percent:        float & >0 & <=100 }
    if fee_type == "per_day" { per_day_amount: #NonNegativeMoney }
    if fee_type == "tiered"  { tiers:          list.MinItems(1) }
})


// --- Percentage Rent (Retail) ---

#PercentageRent: close({
    rate:                float & >0 & <=100            // typically 3-8%
    breakpoint_type:     "natural" | "artificial"
    natural_breakpoint?: #NonNegativeMoney
    artificial_breakpoint?: #NonNegativeMoney
    reporting_frequency: "monthly" | "quarterly" | "annually"
    audit_rights:        *true | bool
})


// --- CAM Terms ---

#CAMReconciliationType: "estimated_with_annual_reconciliation" | "fixed" | "actual"

#CAMTerms: close({
    reconciliation_type:     #CAMReconciliationType
    pro_rata_share_percent:  float & >0 & <=100
    estimated_monthly_cam:   #NonNegativeMoney
    annual_cap?:             #NonNegativeMoney
    includes_property_tax:   bool
    includes_insurance:      bool
    includes_utilities:      bool
    excluded_categories?:    [...string]

    // Gross lease expense stops
    base_year?:          int                           // year against which overages are measured
    base_year_expenses?: #NonNegativeMoney
    expense_stop?:       #NonNegativeMoney             // fixed dollar cap instead of base year

    // Per-category controls for modified gross
    category_terms?: [...#CAMCategoryTerms]

    // Fixed reconciliation = no cap (no reconciliation means no cap)
    if reconciliation_type == "fixed" {
        annual_cap: null
    }

    // base_year and expense_stop are mutually exclusive
    if base_year != _|_ {
        expense_stop: null
    }
    if expense_stop != _|_ {
        base_year: null
    }
})

#CAMCategory: "property_tax" | "insurance" | "utilities" | "janitorial" |
              "landscaping" | "security" | "management_fee" | "repairs" |
              "snow_removal" | "elevator" | "hvac_maintenance" | "other"

#CAMCategoryTerms: close({
    category:     #CAMCategory
    tenant_pays:  bool
    landlord_cap?: #NonNegativeMoney
    tenant_cap?:  #NonNegativeMoney
    escalation?:  float                                // annual increase cap
})


// --- Tenant Improvement ---

#TenantImprovement: close({
    allowance:                #NonNegativeMoney
    amortized:                *false | bool
    amortization_term_months?: int & >0
    interest_rate_percent?:   float & >=0
    completion_deadline?:     time.Time

    // If amortized, must have term
    if amortized == true {
        amortization_term_months: int & >0
    }
})


// --- Renewal Options ---

#RenewalOption: close({
    option_number:       int & >=1
    term_months:         int & >0
    rent_adjustment:     "fixed" | "cpi" | "percent_increase" | "market"
    fixed_rent?:         #NonNegativeMoney
    percent_increase?:   float & >=0
    cpi_floor?:          float & >=0
    cpi_ceiling?:        float & >0
    notice_required_days: int & >=0 | *90
    must_exercise_by?:   time.Time

    if rent_adjustment == "fixed"           { fixed_rent:       #NonNegativeMoney }
    if rent_adjustment == "percent_increase" { percent_increase: float & >=0 }
})


// --- Expansion and Contraction Rights ---

#ExpansionType: "first_right_of_refusal" | "first_right_to_negotiate" | "must_take" | "option"

#ExpansionRight: close({
    type:                 #ExpansionType
    target_space_ids:     [...string]
    exercise_deadline?:   time.Time
    terms?:               string                       // description of economic terms
    notice_required_days: int & >=0
})

#ContractionRight: close({
    minimum_retained_sqft: float & >0
    earliest_exercise_date: time.Time
    penalty?:              #NonNegativeMoney
    notice_required_days:  int & >=0
})


// --- Subsidy Terms (Affordable Housing) ---

#SubsidyProgram: "section_8" | "pbv" | "vash" | "home" | "lihtc"

#SubsidyTerms: close({
    program:            #SubsidyProgram
    housing_authority:  string
    hap_contract_id?:   string
    contract_rent:      #NonNegativeMoney
    tenant_portion:     #NonNegativeMoney
    subsidy_portion:    #NonNegativeMoney
    utility_allowance:  #NonNegativeMoney
    annual_recert_date?: time.Time
    income_limit_ami_percent: int & >0 & <=150
})
```

### 7.4 Application

```cue
#ApplicationStatus: "submitted" | "screening" | "under_review" | "approved" |
                    "conditionally_approved" | "denied" | "withdrawn" | "expired"

#Application: close({
    #StatefulEntity

    property_id:         string
    space_id?:           string                        // may apply without specific space
    applicant_person_id: string

    status: #ApplicationStatus

    desired_move_in:          time.Time
    desired_lease_term_months: int & >0

    screening_request_id?: string
    screening_completed?:  time.Time
    credit_score?:         int & >=300 & <=850
    background_clear:      *false | bool
    income_verified:       *false | bool
    income_to_rent_ratio?: float & >=0

    decision_by?:     string                           // Person ID
    decision_at?:     time.Time
    decision_reason?: string
    conditions?:      [...string]                      // for conditional approval

    application_fee: #NonNegativeMoney
    fee_paid:        *false | bool

    // Decisions require decision metadata
    if status == "approved" || status == "conditionally_approved" || status == "denied" {
        decision_by: string
        decision_at: time.Time
    }

    // Denial requires reason (fair housing compliance)
    if status == "denied" {
        decision_reason: string
    }

    // Conditional approval must have conditions listed
    if status == "conditionally_approved" {
        conditions: list.MinItems(1)
    }
})
```

---

## 8. Jurisdiction Model — `ontology/jurisdiction.cue`

Jurisdiction demonstrates CUE's **unification** for constraint composition. Multiple jurisdiction rules compose automatically — most restrictive wins for limits, accumulate for requirements.

### 8.1 Jurisdiction

```cue
#JurisdictionType: "federal" | "state" | "county" | "city" |
                   "special_district" | "unincorporated_area"

#JurisdictionStatus: "active" | "dissolved" | "merged" | "pending"

#Jurisdiction: close({
    #StatefulEntity

    name:              string & strings.MinRunes(1)
    jurisdiction_type: #JurisdictionType

    parent_jurisdiction_id?: string                    // self-referential hierarchy

    fips_code?:    string & =~"^[0-9]{5,10}$"
    state_code?:   string & =~"^[A-Z]{2}$"
    country_code:  string & =~"^[A-Z]{2}$" | *"US"

    status: #JurisdictionStatus

    successor_jurisdiction_id?: string
    effective_date?:            time.Time
    dissolution_date?:          time.Time

    governing_body?: string
    regulatory_url?: string

    // Dissolved/merged must have successor and date
    if status == "dissolved" || status == "merged" {
        successor_jurisdiction_id: string
        dissolution_date:          time.Time
    }

    _display_template: "{name}"
})
```

### 8.2 PropertyJurisdiction (Join Entity)

```cue
#PropertyJurisdiction: close({
    #BaseEntity

    property_id:     string                            @immutable
    jurisdiction_id: string                            @immutable

    effective_date: time.Time
    end_date?:      time.Time                          // null = currently active

    source:   "address_geocode" | "manual" | "api_lookup" | "imported"
    verified: *false | bool
    verified_at?: time.Time
    verified_by?: string

    if end_date != _|_ {
        end_date: >effective_date
    }
})
```

### 8.3 JurisdictionRule

```cue
#JurisdictionRuleType: "security_deposit_limit" | "notice_period" | "rent_increase_cap" |
                       "required_disclosure" | "eviction_procedure" | "late_fee_cap" |
                       "rent_control" | "habitability_standard" | "tenant_screening_restriction" |
                       "lease_term_restriction" | "fee_restriction" | "relocation_assistance" |
                       "right_to_counsel" | "just_cause_eviction" | "source_of_income_protection" |
                       "lead_paint_disclosure" | "mold_disclosure" | "bed_bug_disclosure" |
                       "flood_zone_disclosure" | "utility_billing_restriction" |
                       "short_term_rental_restriction"

#JurisdictionRuleStatus: "draft" | "active" | "superseded" | "expired" | "repealed"

#JurisdictionRule: close({
    #StatefulEntity

    jurisdiction_id: string
    rule_type:       #JurisdictionRuleType
    status:          #JurisdictionRuleStatus

    // Applicability filters
    applies_to_lease_types?:    [...#LeaseType]
    applies_to_property_types?: [...#PropertyType]
    applies_to_space_types?:    [...#SpaceType]

    exemptions?: #RuleExemptions

    // Typed rule definition — schema varies by rule_type
    rule_definition: _                                 // validated by _rule_schema below

    // Legal reference
    statute_reference?: string
    ordinance_number?:  string
    statute_url?:       string

    // Temporal validity
    effective_date:     time.Time
    expiration_date?:   time.Time
    superseded_by_id?:  string

    // Verification
    last_verified?:       time.Time
    verified_by?:         string
    verification_source?: string

    // Conditional constraints
    if status == "superseded" { superseded_by_id: string }
    if expiration_date != _|_ { expiration_date: >effective_date }

    // Rule definition schema dispatch based on rule_type
    _rule_schema: {
        if rule_type == "security_deposit_limit" { rule_definition: #SecurityDepositLimitRule }
        if rule_type == "notice_period"           { rule_definition: #NoticePeriodRule }
        if rule_type == "rent_increase_cap"       { rule_definition: #RentIncreaseCapRule }
        if rule_type == "late_fee_cap"            { rule_definition: #LateFeeCapRule }
        if rule_type == "eviction_procedure"      { rule_definition: #EvictionProcedureRule }
        if rule_type == "required_disclosure"     { rule_definition: #RequiredDisclosureRule }
        if rule_type == "short_term_rental_restriction" { rule_definition: #ShortTermRentalRestrictionRule }
    }
})
```

### 8.4 Rule Exemptions

```cue
#RuleExemptions: close({
    owner_occupied?:           bool
    owner_occupied_max_units?: int & >=1
    units_built_after?:        time.Time
    units_built_within_years?: int & >0
    single_family_exempt?:     bool
    small_property_max_units?: int & >=1
    subsidized_exempt?:        bool
    corporate_owner_only?:     bool
    custom_exemptions?:        [...string]
})
```

### 8.5 Rule Definition Schemas

Each `rule_type` has a typed definition. CUE validates the definition matches the rule type.

```cue
#SecurityDepositLimitRule: close({
    max_months:                    float & >0
    furnished_max_months?:         float & >0
    additional_pet_deposit_allowed?: bool
    max_pet_deposit?:              #NonNegativeMoney
    refund_deadline_days:          int & >0
    itemization_required:          bool
    interest_required?:            bool
    interest_rate?:                float
    notes?:                        string
})

#NoticePeriodRule: close({
    tenancy_under_1_year_days:      int & >=0
    tenancy_over_1_year_days:       int & >=0
    increase_over_threshold_days?:  int & >=0
    increase_threshold_percent?:    float
    month_to_month_termination_days: int & >=0
    fixed_term_non_renewal_days?:   int & >=0
    notes?:                         string
})

#RentIncreaseCapRule: close({
    cap_type:       "fixed_percent" | "cpi_plus_fixed" | "cpi_only" | "board_determined" | "none"
    fixed_percent?: float
    max_percent?:   float
    cpi_index?:     string
    frequency:      "annual" | "biannual" | "per_tenancy"
    applies_to:     "existing_tenants_only" | "all_tenants" | "rent_controlled_units"
    vacancy_decontrol?: bool
    notes?:         string
})

#LateFeeCapRule: close({
    max_flat?:                #NonNegativeMoney
    max_percent?:             float
    grace_period_min_days:    int & >=0
    compound_prohibited?:     bool
    notes?:                   string
})

#EvictionProcedureRule: close({
    just_cause_required:             bool
    just_causes?:                    [...string]
    cure_period_days:                int & >=0
    notice_type:                     string
    mandatory_mediation?:            bool
    relocation_assistance_required?: bool
    relocation_amount?:              #NonNegativeMoney
    right_to_counsel?:               bool
    winter_eviction_moratorium?:     bool
    moratorium_months?:              [...int & >=1 & <=12]
    notes?:                          string
})

#RequiredDisclosureRule: close({
    disclosure_type: string
    timing:         "before_signing" | "at_signing" | "within_days_of_signing" | "annually"
    timing_days?:   int
    form_required?: bool
    form_reference?: string
    notes?:         string
})

#ShortTermRentalRestrictionRule: close({
    permitted:                      bool
    license_required?:              bool
    max_days_per_year?:             int
    hosted_only?:                   bool
    primary_residence_only?:        bool
    transient_occupancy_tax_rate?:  float
    platform_registration_required?: bool
    notes?:                         string
})
```

### 8.6 Jurisdiction Composition via CUE Unification

This is where CUE shines. Jurisdiction constraints compose through unification. Define constraints at each level and CUE computes the combined result:

```cue
// Base residential lease constraints (universal)
#ResidentialLeaseConstraints: {
    security_deposit_max_months: number      // no universal limit
    notice_period_min_days:      int & >=0
    rent_increase_max_percent?:  number
    disclosures_required:        [...string]
}

// California constraints — compose with base
#CaliforniaResidential: #ResidentialLeaseConstraints & {
    security_deposit_max_months: <=1         // AB 12 (effective 2025)
    notice_period_min_days:      >=30        // < 1 year tenancy
    rent_increase_max_percent:   <=10        // AB 1482 ceiling
    disclosures_required: [
        "lead_paint_pre1978",
        "mold",
        "bed_bugs",
        "flood_zone",
        "sex_offender_database",
    ]
}

// Santa Monica constraints — compose with California
#SantaMonicaResidential: #CaliforniaResidential & {
    notice_period_min_days: >=90             // city requires more
    disclosures_required: [
        "lead_paint_pre1978",
        "mold",
        "bed_bugs",
        "flood_zone",
        "sex_offender_database",
        "rent_control_rights",               // city-specific disclosure
    ]
    _just_cause_required:         true
    _mandatory_mediation:         true
    _relocation_assistance:       true
    _rent_control_board_approval: true
}

// CUE computes the result automatically:
// - security_deposit_max_months: <=1 (California, Santa Monica doesn't override)
// - notice_period_min_days: >=90 (Santa Monica overrides California's >=30 because >=90 ⊂ >=30)
// - rent_increase_max_percent: <=10 (California ceiling)
// - disclosures: Santa Monica list (superset of California)
//
// If Santa Monica tried notice_period_min_days: >=15 (less restrictive than state):
// CUE ERROR: >=30 & >=15 would be valid BUT if a specific value like 20 is tested,
// it satisfies >=15 but fails >=30 → the state constraint catches it.
//
// If constraints truly conflict (state says >100, city says <50):
// CUE unification FAILS at `cue vet` time. You can't deploy contradictory rules.
```

This is demonstrative — the runtime jurisdiction resolver (Section 5.10 of the ontology spec v2) handles the actual per-property resolution. But CUE lets you **validate the rule definitions themselves** for consistency at build time, before they reach the database.

---

## 9. Accounting Model — `ontology/accounting.cue`

### 9.1 Account (Chart of Accounts)

```cue
#AccountType: "asset" | "liability" | "equity" | "revenue" | "expense"

#AccountSubtype: "cash" | "accounts_receivable" | "prepaid" | "fixed_asset" |
                 "accumulated_depreciation" | "other_asset" |
                 "accounts_payable" | "accrued_liability" | "unearned_revenue" |
                 "security_deposits_held" | "other_liability" |
                 "owners_equity" | "retained_earnings" | "distributions" |
                 "rental_income" | "other_income" | "cam_recovery" | "percentage_rent_income" |
                 "operating_expense" | "maintenance_expense" | "utility_expense" |
                 "management_fee_expense" | "depreciation_expense" | "other_expense"

#NormalBalance: "debit" | "credit"

#Account: close({
    #BaseEntity

    account_number: string & =~"^[0-9]{4}(\\.[0-9]{1,3})?$"   // "1000", "4100.001"
    name:           string
    description?:   string

    account_type:    #AccountType
    account_subtype: #AccountSubtype
    normal_balance:  #NormalBalance

    parent_account_id?: string
    depth:              int & >=0 | *0

    dimensions?: #AccountDimensions

    is_header:            *false | bool
    is_system:            *false | bool                // system accounts cannot be deleted
    allows_direct_posting: *true | bool

    status: "active" | "inactive" | "archived"

    is_trust_account: *false | bool
    trust_type?:      "operating" | "security_deposit" | "escrow"

    budget_amount?: #Money
    tax_line?:      string                             // maps to tax form line

    // Normal balance is determined by account type
    if account_type == "asset" || account_type == "expense" {
        normal_balance: "debit"
    }
    if account_type == "liability" || account_type == "equity" || account_type == "revenue" {
        normal_balance: "credit"
    }

    // Header accounts cannot have direct postings
    if is_header == true {
        allows_direct_posting: false
    }

    // Trust accounts must specify trust type
    if is_trust_account == true {
        trust_type: "operating" | "security_deposit" | "escrow"
    }

    _display_template: "{account_number} — {name}"
})

#AccountDimensions: close({
    entity_id?:   string                               // legal entity
    property_id?: string
    dimension_1?: string                               // department
    dimension_2?: string                               // cost center
    dimension_3?: string                               // project/job code
})
```

### 9.2 Ledger Entry

Ledger entries are **IMMUTABLE**. Errors are corrected with adjustment entries, never by modifying existing entries.

```cue
#EntryType: "charge" | "payment" | "credit" | "adjustment" |
            "refund" | "deposit" | "nsf" | "write_off" |
            "late_fee" | "management_fee" | "owner_draw"

#LedgerEntry: close({
    #ImmutableEntity                                   // cannot be updated or deleted

    account_id: string
    entry_type: #EntryType
    amount:     #Money

    journal_entry_id: string                           // double-entry: every entry belongs to a JE

    effective_date: time.Time                          // when the economic event occurred
    posted_date:    time.Time                          // when it was recorded

    description: string
    charge_code: string
    memo?:       string

    // Dimensional references
    property_id:       string                          // always required
    space_id?:         string
    lease_id?:         string
    person_id?:        string                          // tenant, owner, vendor @sensitive

    // Bank/trust
    bank_account_id?:     string
    bank_transaction_id?: string

    // Reconciliation
    reconciled:        *false | bool
    reconciliation_id?: string
    reconciled_at?:    time.Time

    // For adjustments
    adjusts_entry_id?: string

    // Payments and refunds require a person
    if entry_type == "payment" || entry_type == "refund" || entry_type == "nsf" {
        person_id: string
    }

    // Charges and late fees require a lease
    if entry_type == "charge" || entry_type == "late_fee" {
        lease_id: string
    }

    // Adjustments reference what they're correcting
    if entry_type == "adjustment" {
        adjusts_entry_id: string
    }

    // Reconciled entries must have reconciliation metadata
    if reconciled == true {
        reconciliation_id: string
        reconciled_at:     time.Time
    }
})
```

### 9.3 Journal Entry

Groups LedgerEntries that must balance (debits = credits).

```cue
#JournalEntrySource: "manual" | "auto_charge" | "payment" | "bank_import" |
                     "cam_reconciliation" | "depreciation" | "accrual" |
                     "intercompany" | "management_fee" | "system"

#JournalEntryStatus: "draft" | "pending_approval" | "posted" | "voided"

#JournalEntry: close({
    #StatefulEntity

    entry_date:  time.Time
    posted_date: time.Time
    description: string

    source_type: #JournalEntrySource
    source_id?:  string

    status: #JournalEntryStatus

    approved_by?: string
    approved_at?: time.Time

    batch_id?:    string
    entity_id?:   string                               // legal entity
    property_id?: string

    reverses_journal_id?:    string
    reversed_by_journal_id?: string

    lines: [...#JournalLine] & list.MinItems(2)        // double-entry: at least 2 lines

    // Manual postings require approval
    if status == "posted" && source_type == "manual" {
        approved_by: string
        approved_at: time.Time
    }

    // Voided entries must have reversal
    if status == "voided" {
        reversed_by_journal_id: string
    }

    // Hidden: runtime validation (enforced at Ent hook, not CUE)
    _invariant: "sum(lines[*].debit) == sum(lines[*].credit)"
})

#JournalLine: close({
    account_id:  string
    debit?:      #NonNegativeMoney
    credit?:     #NonNegativeMoney
    description?: string
    dimensions?: #AccountDimensions

    // Must have exactly one of debit or credit
    // CUE cannot express XOR directly; enforced at Ent hook
    _invariant: "exactly_one_of(debit, credit)"
})
```

### 9.4 Bank Account

```cue
#BankAccountType: "operating" | "trust" | "security_deposit" | "escrow" | "reserve"
#BankAccountStatus: "active" | "inactive" | "frozen" | "closed"

#BankAccount: close({
    #StatefulEntity

    name:         string
    account_type: #BankAccountType
    status:       #BankAccountStatus

    gl_account_id: string                              // linked CoA account

    institution_name:  string
    routing_number:    string & =~"^[0-9]{9}$"         @sensitive
    account_mask:      string & =~"^\\*{4}[0-9]{4}$"  // "****1234" — last 4 only @sensitive
    account_number_encrypted?: string                  @sensitive

    plaid_account_id?: string
    plaid_access_token?: string                        @sensitive

    current_balance?:    #Money                        @computed
    last_statement_date?: time.Time

    is_default:          *false | bool
    accepts_deposits:    *true | bool
    accepts_payments:    *true | bool
})
```

### 9.5 Reconciliation

```cue
#ReconciliationStatus: "in_progress" | "balanced" | "unbalanced" | "approved"

#Reconciliation: close({
    #StatefulEntity

    bank_account_id:  string
    status:           #ReconciliationStatus

    period_start:     time.Time
    period_end:       time.Time
    statement_balance: #Money
    gl_balance?:      #Money                           @computed
    difference?:      #Money                           @computed

    statement_date:   time.Time

    reconciled_by?:   string
    reconciled_at?:   time.Time
    approved_by?:     string
    approved_at?:     time.Time

    unreconciled_items?: int & >=0                     @computed

    if period_end != _|_ {
        period_end: >period_start
    }
})
```

---

## 10. State Machines — `ontology/state_machines.cue`

Every entity with a status field has an explicit state machine. These generate into Ent hooks that reject invalid transitions at the persistence layer.

```cue
package ontology

// State machines defined as maps from state → valid target states.
// Generators read these to produce:
//   - Ent hooks (reject invalid transitions)
//   - API transition endpoints (one per valid transition)
//   - UI action buttons (only valid transitions shown)
//   - Test matrices (every valid + every invalid transition)

#StateMachine: {
    [string]: [...string]                              // from_state → [valid_target_states]
}

_state_machines: {
    lease: #StateMachine & {
        draft:                    ["pending_approval", "pending_signature", "terminated"]
        pending_approval:         ["draft", "pending_signature", "terminated"]
        pending_signature:        ["active", "draft", "terminated"]
        active:                   ["expired", "month_to_month_holdover", "terminated", "eviction"]
        expired:                  ["active", "month_to_month_holdover", "renewed", "terminated"]
        month_to_month_holdover:  ["active", "renewed", "terminated", "eviction"]
        renewed:                  []
        terminated:               []
        eviction:                 ["terminated"]
    }

    space: #StateMachine & {
        vacant:         ["occupied", "make_ready", "down", "model", "reserved"]
        occupied:       ["notice_given"]
        notice_given:   ["make_ready", "occupied"]
        make_ready:     ["vacant", "down"]
        down:           ["make_ready", "vacant"]
        model:          ["vacant", "occupied"]
        reserved:       ["vacant", "occupied"]
        owner_occupied: ["vacant"]
    }

    application: #StateMachine & {
        submitted:              ["screening", "withdrawn"]
        screening:              ["under_review", "withdrawn"]
        under_review:           ["approved", "conditionally_approved", "denied", "withdrawn"]
        approved:               ["expired"]
        conditionally_approved: ["approved", "denied", "withdrawn", "expired"]
        denied:                 []
        withdrawn:              []
        expired:                []
    }

    journal_entry: #StateMachine & {
        draft:            ["pending_approval", "posted"]
        pending_approval: ["posted", "draft"]
        posted:           ["voided"]
        voided:           []
    }

    portfolio: #StateMachine & {
        onboarding: ["active"]
        active:     ["inactive", "offboarding"]
        inactive:   ["active", "offboarding"]
        offboarding: ["inactive"]
    }

    property: #StateMachine & {
        onboarding:       ["active"]
        active:           ["inactive", "under_renovation", "for_sale"]
        inactive:         ["active"]
        under_renovation: ["active", "for_sale"]
        for_sale:         ["active", "inactive"]
    }

    building: #StateMachine & {
        active:           ["inactive", "under_renovation"]
        inactive:         ["active"]
        under_renovation: ["active"]
    }

    person_role: #StateMachine & {
        pending:    ["active", "terminated"]
        active:     ["inactive", "terminated"]
        inactive:   ["active", "terminated"]
        terminated: []
    }

    bank_account: #StateMachine & {
        active:   ["inactive", "frozen", "closed"]
        inactive: ["active", "closed"]
        frozen:   ["active", "closed"]
        closed:   []
    }

    reconciliation: #StateMachine & {
        in_progress: ["balanced", "unbalanced"]
        balanced:    ["approved", "in_progress"]
        unbalanced:  ["in_progress"]
        approved:    []
    }

    jurisdiction: #StateMachine & {
        pending:   ["active"]
        active:    ["dissolved", "merged"]
        dissolved: []
        merged:    []
    }

    jurisdiction_rule: #StateMachine & {
        draft:      ["active"]
        active:     ["superseded", "expired", "repealed"]
        superseded: []
        expired:    []
        repealed:   []
    }
}
```

---

## 11. Relationships — `ontology/relationships.cue`

These define all edges between entities. Each relationship drives Ent edge generation, permission path evaluation, agent reasoning, and event routing.

```cue
package ontology

#Cardinality: "O2O" | "O2M" | "M2O" | "M2M"

#Relationship: close({
    from:        string
    to:          string
    edge_name:   string
    cardinality: #Cardinality
    required:    *false | bool
    semantic:    string
    inverse:     string
    via?:        string                                // for M2M: join entity
})

_relationships: [...#Relationship] & [
    // --- Portfolio ---
    {from: "Portfolio", to: "Property",    edge_name: "properties",      cardinality: "O2M", semantic: "Portfolio contains Properties",          inverse: "portfolio"},
    {from: "Portfolio", to: "Organization", edge_name: "owner",          cardinality: "M2O", required: true, semantic: "Portfolio owned by Organization", inverse: "owned_portfolios"},
    {from: "Portfolio", to: "BankAccount", edge_name: "trust_account",   cardinality: "O2O", semantic: "Portfolio uses BankAccount",              inverse: "trust_portfolio"},

    // --- Property ---
    {from: "Property",  to: "Building",    edge_name: "buildings",       cardinality: "O2M", semantic: "Property has Buildings",                  inverse: "property"},
    {from: "Property",  to: "Space",       edge_name: "spaces",          cardinality: "O2M", semantic: "Property contains Spaces",                inverse: "property"},
    {from: "Property",  to: "BankAccount", edge_name: "bank_account",    cardinality: "M2O", semantic: "Property uses BankAccount",               inverse: "properties"},
    {from: "Property",  to: "Application", edge_name: "applications",    cardinality: "O2M", semantic: "Property receives Applications",          inverse: "property"},
    {from: "Property",  to: "Jurisdiction", edge_name: "jurisdictions",  cardinality: "M2M", via: "PropertyJurisdiction", semantic: "Property subject to Jurisdictions", inverse: "properties"},

    // --- Jurisdiction ---
    {from: "Jurisdiction", to: "Jurisdiction",     edge_name: "children",  cardinality: "O2M", semantic: "Jurisdiction contains sub-Jurisdictions", inverse: "parent_jurisdiction"},
    {from: "Jurisdiction", to: "JurisdictionRule", edge_name: "rules",    cardinality: "O2M", semantic: "Jurisdiction has Rules",                  inverse: "jurisdiction"},
    {from: "JurisdictionRule", to: "JurisdictionRule", edge_name: "superseded_by", cardinality: "O2O", semantic: "Rule superseded by newer Rule", inverse: "supersedes"},

    // --- Building ---
    {from: "Building", to: "Space", edge_name: "spaces", cardinality: "O2M", semantic: "Building contains Spaces", inverse: "building"},

    // --- Space ---
    {from: "Space", to: "Space",       edge_name: "children",     cardinality: "O2M", semantic: "Space contains child Spaces",  inverse: "parent_space"},
    {from: "Space", to: "Lease",       edge_name: "leases",       cardinality: "M2M", via: "LeaseSpace", semantic: "Space has Leases", inverse: "spaces"},
    {from: "Space", to: "Application", edge_name: "applications", cardinality: "O2M", semantic: "Space receives Applications",  inverse: "space"},

    // --- Lease ---
    {from: "Lease", to: "PersonRole",  edge_name: "tenant_roles",    cardinality: "M2M", semantic: "Lease held by tenant PersonRoles",        inverse: "leases"},
    {from: "Lease", to: "PersonRole",  edge_name: "guarantor_roles", cardinality: "M2M", semantic: "Lease guaranteed by guarantor PersonRoles", inverse: "guaranteed_leases"},
    {from: "Lease", to: "LedgerEntry", edge_name: "ledger_entries",  cardinality: "O2M", semantic: "Lease generates LedgerEntries",            inverse: "lease"},
    {from: "Lease", to: "Application", edge_name: "application",     cardinality: "O2O", semantic: "Lease originated from Application",        inverse: "resulting_lease"},
    {from: "Lease", to: "Lease",       edge_name: "subleases",       cardinality: "O2M", semantic: "Master lease has subleases",               inverse: "parent_lease"},

    // --- LeaseSpace ---
    {from: "LeaseSpace", to: "Lease", edge_name: "lease", cardinality: "M2O", required: true, semantic: "LeaseSpace belongs to Lease", inverse: "lease_spaces"},
    {from: "LeaseSpace", to: "Space", edge_name: "space", cardinality: "M2O", required: true, semantic: "LeaseSpace references Space", inverse: "lease_spaces"},

    // --- Person ---
    {from: "Person", to: "PersonRole",   edge_name: "roles",         cardinality: "O2M", semantic: "Person has Roles",                  inverse: "person"},
    {from: "Person", to: "Organization", edge_name: "organizations", cardinality: "M2M", semantic: "Person affiliated with Organizations", inverse: "people"},
    {from: "Person", to: "Application",  edge_name: "applications",  cardinality: "O2M", semantic: "Person submits Applications",       inverse: "applicant"},
    {from: "Person", to: "LedgerEntry",  edge_name: "ledger_entries", cardinality: "O2M", semantic: "Person has financial entries",      inverse: "person"},

    // --- Organization ---
    {from: "Organization", to: "Organization", edge_name: "subsidiaries",     cardinality: "O2M", semantic: "Organization has subsidiaries",    inverse: "parent_org"},
    {from: "Organization", to: "Portfolio",    edge_name: "owned_portfolios", cardinality: "O2M", semantic: "Organization owns Portfolios",     inverse: "owner"},

    // --- Accounting ---
    {from: "Account",        to: "Account",        edge_name: "children",       cardinality: "O2M", semantic: "Account has sub-Accounts",        inverse: "parent"},
    {from: "LedgerEntry",    to: "JournalEntry",   edge_name: "journal_entry",  cardinality: "M2O", required: true, semantic: "Entry belongs to JournalEntry", inverse: "lines"},
    {from: "LedgerEntry",    to: "Account",        edge_name: "account",        cardinality: "M2O", required: true, semantic: "Entry posts to Account",  inverse: "entries"},
    {from: "LedgerEntry",    to: "Property",       edge_name: "property",       cardinality: "M2O", required: true, semantic: "Entry relates to Property", inverse: "ledger_entries"},
    {from: "LedgerEntry",    to: "Space",          edge_name: "space",          cardinality: "M2O", semantic: "Entry relates to Space",          inverse: "ledger_entries"},
    {from: "LedgerEntry",    to: "Lease",          edge_name: "lease",          cardinality: "M2O", semantic: "Entry relates to Lease",          inverse: "ledger_entries"},
    {from: "LedgerEntry",    to: "Person",         edge_name: "person",         cardinality: "M2O", semantic: "Entry relates to Person",         inverse: "ledger_entries"},
    {from: "BankAccount",    to: "Account",        edge_name: "gl_account",     cardinality: "M2O", required: true, semantic: "BankAccount tracked via GL Account", inverse: "bank_accounts"},
    {from: "Reconciliation", to: "BankAccount",    edge_name: "bank_account",   cardinality: "M2O", required: true, semantic: "Reconciliation for BankAccount",  inverse: "reconciliations"},
]
```

---

## 12. Commands — `commands/*.cue`

In a CQRS architecture, mutations are **actions** that modify the state of one or more domain objects. "Move In Tenant" does not create a "move in" entity — it transitions a Lease to active, transitions a Space to occupied, updates PersonRole attributes, and creates LedgerEntries for deposit and first month rent. The command payload is loosely related to any single domain object.

Commands are **separately authored** by domain experts. They import ontology types for field validation but define their own shapes. The ontology tells you what a Lease IS. A command tells you what "Move In Tenant" NEEDS.

### 12.1 Command Schema

```cue
package commands

import "propeller.io/ontology"

// Every command follows this structure:
#Command: {
    // Payload fields defined per command (NOT derived from entity)
    ...
    
    // Metadata: what this command touches
    _affects:              [...ontology.#EntityType]   // which entity types are mutated
    _requires_permission:  string                      // permission key
    _jurisdiction_checks?: [...string]                 // jurisdiction rules to evaluate
    _idempotency_key?:     string                      // for safe retry
}
```

### 12.2 Lease Commands

```cue
#MoveInTenant: close({
    lease_id:             string
    actual_move_in_date:  time.Time
    key_handoff_notes?:   string
    inspection_completed: bool
    initial_meter_readings?: [...close({
        meter_id:  string
        reading:   float & >=0
        photo_id?: string
    })]
    
    // Execution plan (documentation for implementers + agent reasoning):
    // 1. Validate lease is in status "pending_signature" or "active" (for late move-in recording)
    // 2. Lease: set move_in_date = actual_move_in_date
    // 3. Lease: if status is "pending_signature", transition → active
    // 4. Space(s): transition primary space(s) → occupied
    // 5. PersonRole(tenant): set move_in_date on TenantAttributes
    // 6. Create LedgerEntries: security deposit charge, prorated first month rent
    // 7. If initial_meter_readings provided, record baseline readings
    // 8. Emit: TenantMovedIn event
    
    _affects:             ["lease", "space", "person_role", "ledger_entry"]
    _requires_permission: "lease:move_in"
})

#RecordPayment: close({
    lease_id:           string
    amount:             ontology.#PositiveMoney          // reuses ontology type
    payment_method:     "ach" | "check" | "cash" | "money_order" | "credit_card"
    reference_number?:  string
    received_date:      time.Time
    memo?:              string
    bank_account_id?:   string
    allocations?: [...close({
        charge_id: string
        amount:    ontology.#PositiveMoney
    })]
    
    // Execution plan:
    // 1. Create JournalEntry with lines: debit cash, credit receivable
    // 2. Create LedgerEntry(payment) linked to lease and person
    // 3. If allocations provided, apply to specific charges; else auto-allocate oldest first
    // 4. Update TenantAttributes.current_balance
    // 5. If balance reaches $0, update TenantAttributes.standing → "good"
    // 6. Emit: PaymentReceived event
    
    _affects:             ["ledger_entry", "journal_entry", "person_role"]
    _requires_permission: "payment:record"
})

#RenewLease: close({
    lease_id:             string
    new_term:             ontology.#DateRange
    new_base_rent:        ontology.#NonNegativeMoney
    rent_change_reason?:  string
    retain_existing_charges: *true | bool
    updated_charges?: [...close({
        charge_id?: string                               // existing charge to modify, or omit for new
        charge_code: string
        description: string
        amount:     ontology.#NonNegativeMoney
        frequency:  ontology.#ChargeFrequency
    })]
    updated_cam_terms?:   ontology.#CAMTerms
    renewal_option_exercised?: int                       // which renewal option number, if applicable
    
    // Execution plan:
    // 1. Validate jurisdiction constraints (rent increase cap, notice period)
    // 2. Create new Lease entity (renewal = new lease, not mutation of old)
    // 3. Copy LeaseSpace records to new lease
    // 4. Transition old lease status → renewed
    // 5. Transition new lease status → active (or pending_signature if signing required)
    // 6. If renewal_option_exercised, mark option as used
    // 7. Emit: LeaseRenewed event
    
    _affects:             ["lease", "lease_space"]
    _requires_permission: "lease:renew"
    _jurisdiction_checks: ["rent_increase_cap", "notice_period", "required_disclosure"]
})

#InitiateEviction: close({
    lease_id:           string
    reason:             "nonpayment" | "lease_violation" | "nuisance" | 
                        "illegal_activity" | "owner_move_in" | "renovation" | "no_cause"
    violation_details?: string
    balance_owed?:      ontology.#NonNegativeMoney
    cure_offered:       bool
    cure_deadline?:     time.Time
    
    // Execution plan:
    // 1. Validate jurisdiction: is this a valid eviction reason? (just cause check)
    // 2. Validate jurisdiction: cure period requirements met?
    // 3. Validate jurisdiction: is there a winter moratorium?
    // 4. Transition lease status → eviction
    // 5. Update TenantAttributes.standing → "eviction"
    // 6. If relocation_assistance_required by jurisdiction, calculate amount
    // 7. If right_to_counsel jurisdiction, note in communications
    // 8. Emit: EvictionInitiated event
    
    _affects:             ["lease", "person_role"]
    _requires_permission: "lease:eviction"
    _jurisdiction_checks: ["just_cause_eviction", "eviction_procedure", "relocation_assistance",
                           "right_to_counsel"]
})

#OnboardProperty: close({
    name:           string
    portfolio_id:   string
    address:        ontology.#Address                    // reuses ontology type
    property_type:  ontology.#PropertyType               // reuses ontology enum
    year_built:     int & >=1800 & <=2030
    total_spaces:   int & >=1
    
    // Optional: bulk space creation during onboarding
    spaces?: [...close({
        space_number: string
        space_type:   ontology.#SpaceType
        floor?:       int
        square_footage?: float & >0
        bedrooms?:    int & >=0
        bathrooms?:   float & >=0
        market_rent?: ontology.#NonNegativeMoney
    })]
    
    // Execution plan:
    // 1. Create Property entity (status: onboarding)
    // 2. Geocode address → derive jurisdiction stack
    // 3. Create PropertyJurisdiction records for each jurisdiction
    // 4. Resolve jurisdiction rules for this property
    // 5. If spaces provided, create Space entities
    // 6. Create default chart of accounts if portfolio has one
    // 7. Emit: PropertyOnboarded event
    
    _affects:             ["property", "property_jurisdiction", "space"]
    _requires_permission: "property:create"
})
```

### 12.3 What Commands Import vs. Define

```
FROM ONTOLOGY (imported):              DEFINED BY COMMAND (authored):
─────────────────────────              ──────────────────────────────
#PositiveMoney (type validation)       Payload shape and fields
#Address (composite type reuse)        Execution plan (which entities, what order)
#LeaseType, #SpaceType (enum reuse)    Business-specific fields (key_handoff_notes,
#DateRange (constraint enforcement)      inspection_completed, cure_offered)
#CAMTerms (embedded structure reuse)   Permission requirements
#EntityType (metadata)                 Jurisdiction checks needed
                                       Idempotency semantics
```

The ontology provides type safety. The command provides domain semantics. They are separate concerns.

---

## 13. Domain Events — `events/*.cue`

Domain events describe what HAPPENED. They carry enough context for consumers to act without refetching the full entity, but they are NOT entity dumps. Each event is a projection — a specific view of what changed, authored for its consumers.

Events reference ontology types for field validation but define their own shapes. A `TenantMovedIn` event does not carry the full Lease, Person, Space, and LedgerEntry — it carries the facts a subscriber needs.

### 13.1 Lease Events

```cue
package events

import "propeller.io/ontology"

#TenantMovedIn: close({
    // Identifiers — enough for consumers to query their own read models
    lease_id:     string
    property_id:  string
    space_ids:    [...string]
    person_id:    string
    
    // Facts about what happened
    move_in_date: time.Time
    
    // Contextual data consumers commonly need (avoids N+1 fetches)
    lease_type:   ontology.#LeaseType
    base_rent:    ontology.#NonNegativeMoney
    space_number: string                               // denormalized for convenience
    
    // NOT included: full lease, full person, security deposit amount,
    // all tenant attributes, all audit metadata. Consumers query if needed.
})

#PaymentReceived: close({
    lease_id:        string
    property_id:     string
    person_id:       string
    
    amount:          ontology.#PositiveMoney
    payment_method:  string
    received_date:   time.Time
    reference_number?: string
    
    // Post-payment state (consumers need this without refetching)
    new_balance:     ontology.#Money
    standing:        ontology.#TenantAttributes.standing
    
    journal_entry_id: string                           // for audit trail
})

#LeaseRenewed: close({
    old_lease_id:     string
    new_lease_id:     string
    property_id:      string
    
    previous_rent:    ontology.#NonNegativeMoney
    new_rent:         ontology.#NonNegativeMoney
    new_term:         ontology.#DateRange
    
    rent_change_percent: float                         // computed, not stored on entity
    
    // Jurisdiction context for compliance audit
    jurisdiction_rule_ids?: [...string]
    within_cap:             bool
})

#EvictionInitiated: close({
    lease_id:      string
    property_id:   string
    person_id:     string
    
    reason:        string
    balance_owed?: ontology.#NonNegativeMoney
    
    // Jurisdiction context
    just_cause_jurisdiction: bool
    cure_period_days:        int
    relocation_required:     bool
    right_to_counsel:        bool
})
```

### 13.2 Property Events

```cue
#PropertyOnboarded: close({
    property_id:   string
    portfolio_id:  string
    property_type: ontology.#PropertyType
    address:       ontology.#Address
    
    jurisdiction_ids: [...string]                      // resolved from address
    space_count:      int
})

#JurisdictionRuleActivated: close({
    jurisdiction_rule_id: string
    jurisdiction_id:      string
    rule_type:            ontology.#JurisdictionRuleType
    effective_date:       time.Time
    
    // Which properties are affected?
    affected_property_ids: [...string]
    affected_lease_count:  int
    
    statute_reference?: string
})
```

### 13.3 Events vs. Entity State

Key principle: events carry **what changed** and **why it matters**. They do not carry **current entity state**. The distinction:

```
WRONG (entity dump as event):
  LeaseUpdated: { lease: <entire Lease object> }
  
  Problems:
  - Consumers must diff to figure out what changed
  - Couples consumers to internal Lease shape
  - Internal field renames break all consumers
  - Payload is enormous, mostly irrelevant

RIGHT (meaningful domain event):
  RentIncreaseApplied: {
    lease_id, previous_rent, new_rent, effective_date,
    rent_change_percent, within_cap, jurisdiction_rule_ids
  }
  
  Benefits:
  - Self-describing: consumers know exactly what happened
  - Decoupled: internal Lease shape can change freely
  - Compact: only relevant facts
  - Audit-ready: jurisdiction compliance data included
```

For simple CRUD (entity created, entity field updated), the event carries the entity ID + changed fields + previous values — enough to invalidate caches and trigger reindexing. These DO use comprehensions for scaffolding (see Section 16).

---

## 14. External API — `api/v1/*.cue`

The external API is a **versioned contract** with its own shapes. It imports ontology enums and constraint types for validation, but defines its own request/response structures. This is the anti-corruption layer that insulates consumers from internal model evolution.

### 14.1 Design Principles

1. **Internal renames don't break consumers.** Renaming `base_rent` to `monthly_rent` in the ontology updates the database and internal services. The API v1 contract still returns `base_rent` until v2 is cut.

2. **Responses are projections, not entity dumps.** A lease API response includes the property name and primary space number (denormalized for convenience) even though internally those live on different entities.

3. **Requests map to commands, not entity mutations.** `POST /v1/leases/{id}/move-in` maps to the `#MoveInTenant` command, not to `PATCH /v1/leases/{id}` with `{status: "active", move_in_date: ...}`.

4. **Enums are shared, shapes are not.** Both internal and external use `#LeaseType: "fixed_term" | "month_to_month" | ...` — redefining enums is pure drift risk. But the response structure (which fields, what nesting, what names) belongs to the API contract.

### 14.2 API Contract Example

```cue
package api_v1

import "propeller.io/ontology"

// === Shared API types ===

#PaginationRequest: close({
    page?:      int & >=1 | *1
    page_size?: int & >=1 & <=100 | *25
    sort_by?:   string
    sort_dir?:  "asc" | "desc" | *"desc"
})

#PaginationResponse: close({
    page:       int
    page_size:  int
    total:      int
    has_more:   bool
})

#ErrorResponse: close({
    code:    string
    message: string
    details?: [...close({
        field?:  string
        reason:  string
    })]
})

#MoneyResponse: close({
    amount:   number                                   // dollars, not cents (API consumers expect this)
    currency: string
})

// === Lease API ===

#LeaseListResponse: close({
    leases: [...#LeaseSummary]
    pagination: #PaginationResponse
})

#LeaseSummary: close({
    id:             string
    lease_type:     ontology.#LeaseType                 // shared enum
    status:         ontology.#LeaseStatus               // shared enum
    base_rent:      #MoneyResponse
    term_start:     string                             // ISO date string
    term_end?:      string
    
    // Denormalized (not on Lease entity internally)
    property_name:  string
    primary_space:  string
    tenant_name:    string
    
    // Computed
    days_remaining?: int
})

#LeaseDetailResponse: close({
    #LeaseSummary
    
    security_deposit:  #MoneyResponse
    liability_type:    ontology.#LiabilityType
    move_in_date?:     string
    
    // Flattened from relationships
    spaces: [...close({
        space_id:     string
        space_number: string
        space_type:   ontology.#SpaceType
        relationship: ontology.#LeaseSpaceRelationship
        square_footage?: float
    })]
    
    tenants: [...close({
        person_id:  string
        name:       string
        standing:   string
        balance:    #MoneyResponse
    })]
    
    // Commercial structures (conditionally present, flattened)
    cam_terms?:        _                               // API-specific subset
    percentage_rent?:  _
    
    // Jurisdiction constraints currently in effect
    jurisdiction_constraints?: close({
        deposit_limit?:       #MoneyResponse
        rent_increase_cap?:   string                   // "5% + CPI, max 10%"
        notice_period_days?:  int
        just_cause_required?: bool
        rent_controlled?:     bool
    })
    
    // Explicitly EXCLUDED from API:
    // - ssn_last_four, tax_id (PII)
    // - agent_goal_id, correlation_id (internal)
    // - audit metadata (internal)
    // - journal_entry_ids (internal plumbing)
})

// === Command-mapped endpoints ===

// POST /v1/leases/{id}/move-in → MoveInTenant command
#MoveInRequest: close({
    actual_move_in_date:  string                       // ISO date
    key_handoff_notes?:   string
    inspection_completed: bool
    initial_meter_readings?: [...close({
        meter_id: string
        reading:  number
    })]
})

// POST /v1/leases/{id}/renew → RenewLease command
#RenewLeaseRequest: close({
    new_term_start:  string                            // ISO date
    new_term_end:    string                            // ISO date
    new_base_rent:   #MoneyResponse
    rent_change_reason?: string
})
```

### 14.3 API vs. Internal Shape Differences

| Concern | Internal (Ontology) | External (API v1) |
|---|---|---|
| Money | `{amount_cents: int, currency: string}` | `{amount: number, currency: string}` |
| Dates | `time.Time` | ISO 8601 string |
| Entity references | `property_id: string` | `property_name: string` (denormalized) |
| Nested entities | Separate entities via relationships | Flattened into response |
| PII fields | Present with `@sensitive` | Excluded entirely |
| Audit fields | Present on every entity | Not exposed |
| Mutations | Command payloads (CQRS) | Named endpoints mapping to commands |
| Enums | Shared via import | Shared via import |
| Constraints | CUE conditionals | Validated server-side, errors returned |

The transformation between internal and external happens in the **API handler layer** — not in generated code. API handlers are authored code that maps between command payloads and API requests, and between internal entities and API responses. This layer is where field renames, format conversions (cents → dollars), denormalization, and PII filtering happen.

---

## 15. Permission Policies — `policies/*.cue`

Permissions are **business decisions** that reference the domain model but are not derived from it. The ontology tells you Lease has a `security_deposit` field. Business policy decides that leasing agents can see it but maintenance coordinators cannot.

### 15.1 Permission Groups

```cue
package policies

// Permission groups are defined by business stakeholders.
// They are NOT generated from entity schemas.

#PermissionGroup: close({
    name:        string
    description: string
    inherits?:   [...string]                           // inherit from other groups
    commands:    [...string]                            // which commands this group can execute
    queries:     [...string]                            // which query endpoints this group can access
})

_permission_groups: {
    organization_admin: #PermissionGroup & {
        name:        "Organization Admin"
        description: "Full access to all operations within the organization"
        commands:    ["*"]
        queries:     ["*"]
    }
    
    portfolio_admin: #PermissionGroup & {
        name:        "Portfolio Admin"
        description: "Full access within assigned portfolios"
        inherits:    ["property_manager"]
        commands: [
            "property:create", "property:transfer",
            "account:create", "account:modify",
            "bank_account:create",
            "journal_entry:approve",
            "reconciliation:approve",
        ]
        queries: ["portfolio:*", "property:*", "accounting:*"]
    }
    
    property_manager: #PermissionGroup & {
        name:        "Property Manager"
        description: "Day-to-day property operations"
        inherits:    ["leasing_agent", "maintenance_coordinator"]
        commands: [
            "lease:move_in", "lease:move_out", "lease:renew",
            "lease:eviction",
            "payment:record", "payment:reverse",
            "charge:create", "charge:waive",
            "journal_entry:create",
        ]
        queries: ["property:detail", "lease:*", "person:*", "accounting:property_level"]
    }
    
    leasing_agent: #PermissionGroup & {
        name:        "Leasing Agent"
        description: "Leasing operations only"
        commands: [
            "application:process", "application:approve", "application:deny",
            "lease:create", "lease:submit_for_approval",
            "lease:send_for_signature",
        ]
        queries: ["property:list", "space:list", "application:*", "lease:read"]
    }
    
    maintenance_coordinator: #PermissionGroup & {
        name:        "Maintenance Coordinator"
        description: "Work order and maintenance operations"
        commands: [
            "work_order:create", "work_order:assign", "work_order:complete",
            "inspection:schedule", "inspection:record",
        ]
        queries: ["property:detail", "space:detail", "work_order:*"]
    }
    
    accountant: #PermissionGroup & {
        name:        "Accountant"
        description: "Financial operations"
        commands: [
            "payment:record", "charge:create",
            "journal_entry:create", "journal_entry:post",
            "reconciliation:start", "reconciliation:complete",
            "owner_distribution:calculate", "owner_distribution:process",
        ]
        queries: ["accounting:*", "lease:financial", "property:financial"]
    }
    
    viewer: #PermissionGroup & {
        name:        "Viewer"
        description: "Read-only access"
        commands:    []
        queries:     ["property:list", "lease:list", "person:list"]
    }
}
```

### 15.2 Field-Level Policies

Per-attribute visibility is a separate concern from command permissions. Some roles can see a lease but not the security deposit amount. This is defined here, not on the entity.

```cue
// Field policies: which attributes are visible/hidden per group
_field_policies: {
    person: {
        ssn_last_four: {
            visible_to:  ["organization_admin", "portfolio_admin", "accountant"]
            hidden_from: ["*"]                         // default deny
        }
        date_of_birth: {
            visible_to:  ["organization_admin", "portfolio_admin", "leasing_agent"]
            hidden_from: ["maintenance_coordinator", "viewer"]
        }
    }
    
    lease: {
        security_deposit: {
            visible_to:  ["organization_admin", "portfolio_admin", "property_manager", "accountant"]
            hidden_from: ["maintenance_coordinator"]
        }
    }
    
    bank_account: {
        routing_number: {
            visible_to:  ["organization_admin", "accountant"]
            hidden_from: ["*"]
        }
        account_number_encrypted: {
            visible_to:  []                            // nobody sees this in UI; system-only
            hidden_from: ["*"]
        }
    }
}
```

### 15.3 Authorization Mechanism

Permission evaluation uses the PersonRole scope chain from the ontology:

```
Person → PersonRole (scoped to) → Scope Entity → (contains) → Target Entity
```

But the **what they can do** at the end of that chain is defined here in policies, not derived from entity schemas. The ontology provides the scope traversal mechanism. Business policy provides the access decisions.

---

## 16. Drift Detection and Scaffolding — `codegen/drift.cue`

Comprehensions still play a role — but for **validation and scaffolding**, not for whole-cloth generation of contracts. The ontology validates that separately-defined commands, events, and API contracts stay consistent with the domain model.

### 16.1 What CUE Comprehensions Still Generate (Tight Coupling)

These ARE mechanically derived because they have a 1:1 relationship with the domain model:

```cue
// Data store schemas — Ent always mirrors ontology exactly
// Generated by cmd/entgen, no human override needed

// State machine enforcement — Ent hooks from state_machines.cue
// Every valid transition generates a hook. No business judgment involved.

// Test matrices — every constraint and transition generates test cases
_state_machine_tests: {
    for entity_name, machine in ontology._state_machines {
        _valid_transitions: {
            for from_state, targets in machine {
                for _, target in targets {
                    "Test_\(entity_name)_\(from_state)_to_\(target)_succeeds": {
                        entity: entity_name, from: from_state, to: target, expected: "success"
                    }
                }
            }
        }
        _all_states: [ for state, _ in machine { state } ]
        _invalid_transitions: {
            for from_state, valid_targets in machine {
                for _, candidate in _all_states {
                    if !list.Contains(valid_targets, candidate) && candidate != from_state {
                        "Test_\(entity_name)_\(from_state)_to_\(candidate)_rejected": {
                            entity: entity_name, from: from_state, to: candidate, expected: "error"
                        }
                    }
                }
            }
        }
    }
}

// Agent world model — ONTOLOGY.md, STATE_MACHINES.md generated from ontology
// Agent needs to know what things ARE; that's exactly what the ontology defines.
```

### 16.2 What CUE Comprehensions Scaffold (Starting Point, Human Override)

For commands, events, and API contracts, comprehensions generate **starter templates** that domain experts edit:

```cue
// Scaffold: generate a CRUD event pair for every entity
// Domain experts then replace generic "LeaseUpdated" with meaningful events
// like "RentIncreaseApplied", "TenantMovedIn", etc.

_scaffolded_events: {
    for name, _ in ontology._entities {
        "\(name).created": {
            _scaffold: true                            // flag: human should review/replace
            entity_type: name
            description: "\(name) was created"
        }
        "\(name).updated": {
            _scaffold: true
            entity_type: name
            description: "\(name) was updated"
            changed_fields: [...string]
            previous_values: _
        }
        "\(name).deleted": {
            _scaffold: true
            entity_type: name
            description: "\(name) was deleted"
        }
    }
}

// The scaffold gives you 45+ events for 15 entities in one expression.
// Domain experts then:
// 1. Keep generic CRUD events for simple entities (Account, Building)
// 2. Replace generic events with domain-specific ones for rich entities (Lease, Payment)
// 3. Add domain events that don't correspond to any single entity change (ComplianceAlertTriggered)
```

### 16.3 What CUE Validates Across Boundaries (Drift Detection)

This is the key role for CUE in a loosely-coupled architecture. Commands, events, and API contracts are separately authored, but CUE validates that they stay consistent with the ontology:

```cue
package drift

import (
    "propeller.io/ontology"
    "propeller.io/commands"
    "propeller.io/events"
    "propeller.io/api/v1"
)

// DRIFT CHECK 1: Every command's _affects list references valid entity types
_command_entity_check: {
    for cmd_name, cmd in commands {
        for _, entity_type in cmd._affects {
            _valid: ontology.#EntityType & entity_type  // fails if entity_type not in enum
        }
    }
}

// DRIFT CHECK 2: Events that reference ontology enums stay in sync
// If we add a new LeaseStatus to the ontology, any event that
// uses ontology.#LeaseStatus automatically accepts it.
// If we REMOVE a status, any event referencing it fails at cue vet.

// DRIFT CHECK 3: API responses that import ontology enums stay in sync
// api_v1.#LeaseSummary.lease_type is ontology.#LeaseType.
// Add a new lease type → API accepts it. Remove one → cue vet fails.

// DRIFT CHECK 4: Commands that reference ontology types validate field constraints
// commands.#RecordPayment.amount is ontology.#PositiveMoney.
// If we change PositiveMoney to require currency != "USD" (hypothetically),
// any test fixture passing "USD" to that command would fail.

// DRIFT CHECK 5: Permission commands reference actual commands
_permission_command_check: {
    for group_name, group in policies._permission_groups {
        for _, perm in group.commands {
            if perm != "*" {
                // Validate permission key exists as a command
                // (implementation: check against command registry)
            }
        }
    }
}

// Run: cue vet ./ontology/... ./commands/... ./events/... ./api/... ./policies/...
// This validates ALL cross-boundary references in one command.
// Result: shared vocabulary stays in sync. Shapes evolve independently.
```

### 16.4 The Coupling Spectrum

```
          TIGHT (generated)              LOOSE (authored, validated)
          ─────────────────              ──────────────────────────
          
Data store ◄──────── Ontology ────────► Commands
(Ent/Postgres)         │                   (CQRS payloads)
                       │
Test matrices ◄────────┤────────► Events  
(pos + neg tests)      │           (domain projections)
                       │
State machine ◄────────┤────────► External API
(Ent hooks)            │           (versioned contracts)
                       │
Agent world model ◄────┤────────► Permissions
(ONTOLOGY.md)          │           (business-defined groups)
                       │
                       ├────────► View Definitions
                       │           (UI layout decisions)
                       │
                       └────────► Read Models
                                   (query-optimized projections)
```

Left side: generated, 1:1, changes automatically. Right side: authored, loosely coupled, validated for consistency. Both reference the same vocabulary. Neither is subordinate to the other.

---

## 17. Domain-Level Attributes

CUE attributes mark metadata that is true about the field as a domain fact. Each consumer interprets these attributes according to its own concerns — the ontology doesn't prescribe how, it just declares the truth.

```cue
// Attributes used in this ontology:
//
// @immutable        — field cannot change after entity creation
//                     Domain truth: this field is write-once.
//                     Ent interprets:  reject updates to this field
//                     Commands interpret: exclude from update payloads
//                     API interprets:    reject in PATCH requests
//                     UI interprets:     disable in edit forms
//                     Agent interprets:  exclude from update tool parameters
//
// @sensitive         — field contains PII or sensitive data
//                     Domain truth: this field needs protection.
//                     Ent interprets:  encrypt at rest
//                     Events interpret: exclude from event payloads (consumers must query)
//                     API interprets:   exclude from responses unless field policy allows
//                     Logs interpret:   mask value
//                     Agent interprets: omit from context window by default
//                     Policies interpret: field-level visibility rules apply (Section 15.2)
//
// @computed          — field is calculated, not user-supplied
//                     Domain truth: system owns this value.
//                     Commands interpret: exclude from payloads
//                     API interprets:    exclude from request schemas, include in responses
//                     UI interprets:     display-only, never editable
//
// @deprecated(reason, since)  — field is being phased out
//                     Domain truth: this field should not be used in new code.
//                     All consumers interpret: warn at build time if referenced

// Key insight: the ontology declares the attribute.
// Each consumer decides what to do with it.
// The ontology does NOT prescribe "@immutable means disable in edit forms" —
// that's a UI decision made in codegen/uigen.cue.
// The ontology says "this field is write-once" and each consumer acts accordingly.
```

---

## 18. Configuration Schema — `ontology/config_schema.cue`

What's TUNABLE at runtime and the valid ranges. Business preferences resolved in two phases: preference cascade then jurisdiction constraint application.

```cue
package ontology

// Phase 1: Business preference resolution (last wins)
// platform → organization → portfolio → property → lease_type

#LeaseConfiguration: close({
    notice_required_days:        int & >=0 & <=365 | *30
    late_fee_grace_period_days:  int & >=0 & <=30 | *5
    late_fee_type:               #LateFeeType | *"flat"
    late_fee_flat_amount?:       #NonNegativeMoney
    late_fee_percent?:           float & >0 & <=25
    security_deposit_max_months: float & >0 & <=6 | *2
    auto_renewal_enabled:        *false | bool
    auto_renewal_notice_days:    int & >=30 & <=180 | *60
    allow_partial_payments:      *true | bool
    minimum_payment_percent?:    float & >0 & <=100
})

#PropertyConfiguration: close({
    maintenance_auto_approve_limit: #NonNegativeMoney & {amount_cents: <=1000000}  // max $10K
    screening_required:             *true | bool
    screening_income_ratio:         float & >=1 & <=10 | *3
    pet_policy:                     *"allowed" | "restricted" | "prohibited"
    max_pets?:                      int & >=0 & <=10
    pet_deposit?:                   #NonNegativeMoney
    pet_rent?:                      #NonNegativeMoney
})

#ConfigurationScope: "platform" | "organization" | "portfolio" | "property" | "lease_type"

// Phase 2: Jurisdiction constraint application (most restrictive wins)
// See Section 8.6 for how jurisdiction rules compose via CUE unification.
// The resolver compares Phase 1 values against applicable jurisdiction rules.
// Business can be stricter than law, but not more lenient.
```

---

## 19. Implementation Sequence

### Phase 1: Foundation (Weeks 1-2)
- Set up CUE toolchain, project structure with ontology/, commands/, events/, api/v1/, policies/ directories
- Implement base.cue, common.cue with all shared types
- Build cmd/entgen (CUE → Ent schema generator) — tightly coupled, 1:1 derivation
- Validate: Property + Space + simple Lease round-trip through Ent to Postgres
- Demonstrate: `cue vet` catches invalid data against constraints

### Phase 2: Core Ontology (Weeks 2-4)
- Complete all domain model CUE files (person, property, jurisdiction, lease, accounting)
- Complete relationships.cue and state_machines.cue
- Run `cue vet`, resolve inconsistencies
- Generate full Ent schema set, verify migrations
- Seed jurisdiction reference data (US federal + 50 states + major cities)
- Demonstrate: state machine comprehensions generating test matrices

### Phase 3: Commands + Events (Weeks 4-6)
- Define lease commands (MoveInTenant, RecordPayment, RenewLease, InitiateEviction)
- Define property commands (OnboardProperty, TransferProperty)
- Define accounting commands (PostJournalEntry, RecordPayment, Reconcile)
- Define corresponding domain events for each command
- Build command execution infrastructure (command bus → handlers → Ent mutations → event emission)
- Implement MoveInTenant as reference: single command touching 4 entity types
- Demonstrate: command payload validates against ontology types via CUE imports

### Phase 4: External API (Weeks 6-8)
- Define api/v1/ contracts — separate shapes from internal model
- Build API handler layer (request → command → response transformation)
- Implement anti-corruption layer: cents↔dollars, time.Time↔ISO strings, denormalization
- Set up NATS JetStream for domain event distribution
- Build event consumers: graph sync (Neo4j), search sync (Meilisearch), cache invalidation
- Demonstrate: internal field rename does NOT break API contract

### Phase 5: Permissions (Weeks 8-10)
- Define permission groups in policies/ (business-defined, not derived)
- Define field-level policies for PII and sensitive data
- Build authorization middleware: PersonRole scope chain + permission group evaluation
- Integrate with command execution (check _requires_permission before executing)
- Demonstrate: same entity, different field visibility per role

### Phase 6: Drift Detection + Agent Integration (Weeks 10-12)
- Build codegen/drift.cue — cross-boundary validation
- Run `cue vet` across ontology + commands + events + api + policies
- Build cmd/agentgen (ONTOLOGY.md, STATE_MACHINES.md, COMMANDS.md for agent context)
- Agent tools map to commands (not entity CRUD) — agent executes MoveInTenant, not PATCH Lease
- Test agent goal execution against live system
- Demonstrate: add new ontology enum → drift check catches stale API contract

---

## Appendix A: Stress Test Results

This ontology was validated against the following property types and lease structures:

| Scenario | Result |
|---|---|
| Single-family rental (with ADU) | ✅ Works |
| Duplex/fourplex (owner-occupied unit) | ✅ Works (owner_occupied status) |
| Garden-style apartment complex | ✅ Works |
| Mid/high-rise apartment | ✅ Works |
| Mixed-use building (retail + residential) | ✅ Works |
| Student housing (by-the-bed) | ✅ Works (parent/child Space, leasable flag) |
| Student housing (mixed whole-unit and by-the-bed) | ✅ Works |
| Senior living / assisted living | ✅ Works |
| Affordable housing (LIHTC, Section 8) | ✅ Works |
| Vacation / short-term rental | ✅ Works (short_term lease type) |
| Manufactured housing / mobile home park | ✅ Works (lot_pad space type, no building) |
| Commercial office (multi-tenant) | ✅ Works |
| Commercial office (full-floor tenant) | ✅ Works (M2M LeaseSpace) |
| Commercial office (multi-floor expansion) | ✅ Works |
| Retail strip mall with pad sites | ✅ Works |
| Enclosed mall with food court stalls | ✅ Works (parent/child Space) |
| Industrial / warehouse | ✅ Works |
| Flex space (office + warehouse, different rates) | ✅ Works (RecurringCharge.space_id) |
| Medical office building | ✅ Works (specialized_infrastructure) |
| Data center (rack/cage/suite) | ✅ Works (UsageBasedCharge) |
| Self-storage facility | ✅ Works |
| Coworking (hot desk, dedicated, private office) | ⚠️ Partial (membership lease; hourly bookings out of scope) |
| Triple Net (NNN) lease | ✅ Works |
| Double Net (NN) lease | ✅ Works |
| Single Net (N) lease | ✅ Works |
| Gross / Full Service lease | ✅ Works (base_year, expense_stop) |
| Modified Gross lease | ✅ Works (CAMCategoryTerms) |
| Percentage rent (retail) | ✅ Works (PercentageRent) |
| Graduated / step-up lease | ✅ Works (RentScheduleEntry) |
| CPI-indexed lease | ✅ Works (RentAdjustment) |
| Ground lease | ✅ Works (ground_lease type) |
| Sublease (standard) | ✅ Works (parent_lease_id) |
| Sublease (direct-to-landlord billing) | ✅ Works (sublease_billing) |
| Commercial expansion (mid-term) | ✅ Works (LeaseSpace with effective dates) |
| Commercial contraction | ✅ Works (ContractionRight) |
| Joint and several liability | ✅ Works (liability_type on Lease) |
| Roommate departure mid-lease | ✅ Works (occupancy_status + liability_status) |
| Build-to-suit | ✅ Works (lease_commencement vs rent_commencement dates) |
| Flex/hybrid post-COVID | ✅ Works (ExpansionRight) |

**Jurisdiction stress test:**

| Scenario | Result |
|---|---|
| California residential (AB 1482 rent cap, AB 12 deposit limit) | ✅ Works — state rules resolve as most restrictive |
| Santa Monica rent control (city + county + state + federal stack) | ✅ Works — 4-layer hierarchy, city-level rules override |
| New York City rent stabilization (DHCR, RGB) | ✅ Works — special_district jurisdiction type |
| Oregon statewide rent control (SB 608) | ✅ Works — state-level with new construction exemption |
| Commercial lease in unincorporated county area | ✅ Works — county jurisdiction, no city layer |
| Property annexed into new city mid-lease | ✅ Works — PropertyJurisdiction.end_date + new entry |
| New ordinance passed during active lease term | ✅ Works — JurisdictionRule.effective_date, grandfathering via exemptions |
| Sunset clause on rent control ordinance | ✅ Works — JurisdictionRule.expiration_date |
| Ordinance superseded by stricter version | ✅ Works — superseded_by_id chain |
| Multi-property portfolio spanning 3 states | ✅ Works — each property has own jurisdiction stack |
| Section 8 property with federal + state + city rules | ✅ Works — all three layers accumulate |
| Short-term rental restricted by city, allowed by state | ✅ Works — most specific jurisdiction wins |
| Eviction moratorium (temporary, with expiration) | ✅ Works — JurisdictionRule with effective + expiration dates |
| Just cause eviction + right to counsel (dual requirement) | ✅ Works — requirements accumulate, don't override |
| Lead paint disclosure (federal, applies via year_built) | ✅ Works — federal rule with property-level exemption criteria |

**Command stress test:**

| Scenario | Result |
|---|---|
| MoveInTenant touching 4 entity types atomically | ✅ Works — single command, multi-entity mutation |
| RecordPayment with auto-allocation to oldest charges | ✅ Works — command handler logic, not entity CRUD |
| RenewLease with jurisdiction rent cap validation | ✅ Works — command checks _jurisdiction_checks |
| InitiateEviction with just cause + right to counsel | ✅ Works — jurisdiction rules inform command execution |
| OnboardProperty with address geocoding + jurisdiction resolution | ✅ Works — command creates Property + PropertyJurisdictions |
| API v1 response after internal field rename | ✅ Works — API contract decoupled from ontology |
| Permission check: leasing agent tries to post journal entry | ✅ Blocked — command_permissions check |
| Permission check: accountant sees deposit, maintenance tech doesn't | ✅ Works — field_policies per group |

## Appendix B: Technology Stack

| Component | Technology |
|---|---|
| Ontology definition | CUE 0.8+ |
| Persistence | Ent (entgo.io) 0.13+ |
| Database | PostgreSQL 16+ |
| Graph database | Neo4j 5+ |
| Search engine | Meilisearch 1.6+ |
| API framework | Connect-RPC 1.0+ |
| Schema registry | Buf BSR |
| Message bus | NATS JetStream 2.10+ |
| Authorization | OPA 0.62+ |
| Language | Go 1.22+ |
| Frontend | Svelte + Skeleton UI + Tailwind CSS |

## Appendix C: CUE Feature Demonstration Checklist

For the POC, demonstrate these CUE features:

| Feature | Where Demonstrated |
|---|---|
| `close()` — closed structs | Every entity + command + event + API response (extra fields rejected) |
| Embedding (`#BaseEntity`) | Every entity inherits id + audit |
| `#StatefulEntity` | Entities with status get state machine wiring |
| `#ImmutableEntity` | LedgerEntry cannot be updated/deleted |
| `if` conditionals | Lease: NNN requires CAM, fixed_term requires end date |
| `*` defaults | `liability_type: *"joint_and_several"`, `leasable: *true` |
| `=~` patterns | Address postal_code, routing_number, email, state codes |
| `_` hidden fields | `_display_template`, `_affects`, `_requires_permission`, `_jurisdiction_checks` |
| Comprehensions | Test matrices from state machines, scaffolded events, drift detection |
| Unification (`&`) | Jurisdiction constraint composition (CA + Santa Monica) |
| `list.MinItems()` | contact_methods requires ≥1, journal lines requires ≥2 |
| Type unions | Enum types as `"a" \| "b" \| "c"` with full exhaustiveness |
| Nested conditionals | CAMTerms: base_year and expense_stop mutually exclusive |
| Conditional type refinement | PersonRole attributes match role_type |
| Cross-package import | Commands import `ontology.#PositiveMoney` for field validation |
| Drift detection | `cue vet` across ontology + commands + events + api catches stale references |

## Appendix D: Coupling Decisions Rationale

This table documents why each boundary has the coupling level it does.

| Boundary | Coupling | Why This Level | Risk of Tighter | Risk of Looser |
|---|---|---|---|---|
| Data store (Ent) | Tight/1:1 | Store IS the model. No valid reason for divergence. | None (this is correct) | Data loss, inconsistency |
| State machines (Ent hooks) | Tight/1:1 | Enforcement must match definition. | None | Invalid transitions reach DB |
| Test matrices | Tight/generated | Tests SHOULD break when model changes. | None | Stale tests, false confidence |
| Agent world model | Tight/generated | Agent must know current model truth. | None | Agent hallucinates fields |
| Commands (CQRS) | Shared vocabulary | Actions span entities. Payload ≠ any single entity. | Zuora problem: internal refactoring breaks everything | Type drift, validation gaps |
| Domain Events | Shared vocabulary | Events are projections, not dumps. Consumers need stability. | Consumers coupled to internal schema | Wrong types in event payloads |
| External API (v1) | Shared vocabulary | Versioned contract. Customer consistency. | Zuora problem at its worst | Enum drift, inconsistent types |
| Permissions | References only | Business logic, not schema logic. | Over-coupling: schema change = policy change | Stale permission checks |
| Read models | Loose | Optimized for queries, not normalized truth. | Performance bottleneck | Eventual consistency surprises |
| Search indices | Loose | Subset, denormalized for speed. | Indexing overhead | Missing search fields |