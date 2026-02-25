# Propeller Ontological Architecture Specification v2

**Version:** 2.0  
**Date:** February 24, 2026  
**Author:** Matthew Baird, CTO — AppFolio  
**Status:** For Claude Code Implementation

---

## 1. Purpose

This spec defines a canonical domain ontology for property management. An agentic coding system (Claude Code) should use this spec to generate:

1. CUE ontology files (one per domain area + common types + relationships + state machines)
2. CUE codegen mapping files (entgen.cue, apigen.cue, eventgen.cue, authzgen.cue)
3. Go codegen tools (cmd/entgen, cmd/apigen, cmd/eventgen, cmd/authzgen, cmd/agentgen)
4. Makefile and CI pipeline

The CUE ontology is the generative kernel — the single source of truth from which all downstream code is derived:

```
CUE Ontology (source of truth)
  ├── Schema projections (Postgres via Ent, graph DB via Neo4j, search index via Meilisearch)
  ├── API contracts (Connect-RPC services, OpenAPI, TypeScript SDK)
  ├── Event schemas (ontologically typed, NATS JetStream with schema registry)
  ├── Permission model (OPA/Rego, derived from relationship graph)
  └── Agent world model (tool definitions, ONTOLOGY.md, STATE_MACHINES.md)
```

Rule: no hand-written type that represents a domain entity. Constraints belong in the ontology, not scattered in code. State machines are first-class. Relationships carry semantics.

---

## 2. File Structure

```
propeller/
├── ontology/
│   ├── common.cue              # Shared types: Money, Address, DateRange, AuditMetadata, ContactMethod
│   ├── person.cue              # Person, Organization, PersonRole with role-specific attributes
│   ├── property.cue            # Portfolio, Property, Building, Space (replaces Unit)
│   ├── lease.cue               # Lease, LeaseSpace, Application, and all commercial lease structures
│   ├── accounting.cue          # Account, LedgerEntry, JournalEntry, BankAccount, Reconciliation
│   ├── config_schema.cue       # Configuration schema — what's tunable and valid ranges
│   ├── relationships.cue       # All cross-model edges with cardinality and semantics
│   └── state_machines.cue      # All entity state machines
├── codegen/
│   ├── entgen.cue              # Ontology → Ent schema mapping (including type projections)
│   ├── apigen.cue              # Ontology → API service + agent tool mapping
│   ├── eventgen.cue            # Ontology → Event schema mapping
│   └── authzgen.cue            # Ontology → OPA policy scaffold mapping
├── cmd/
│   ├── entgen/main.go          # CUE → Ent schema generator
│   ├── apigen/main.go          # CUE → Proto services + OpenAPI + agent tools + handler scaffolds
│   ├── eventgen/main.go        # CUE → Event schema generator
│   ├── authzgen/main.go        # CUE → OPA policy generator
│   └── agentgen/main.go        # CUE → ONTOLOGY.md + STATE_MACHINES.md + TOOLS.md
└── Makefile
```

---

## 3. Common Types — `ontology/common.cue`

These types are shared across all domain models. They establish the foundational vocabulary.

### 3.1 Monetary

```
#Money: amount_cents (int), currency (ISO 4217, 3 uppercase letters, default "USD")
#NonNegativeMoney: #Money where amount_cents >= 0
#PositiveMoney: #Money where amount_cents > 0
```

CRITICAL: All financials use integer cents. No floating point anywhere in the financial chain.

### 3.2 Temporal

```
#DateRange: start (time), end (optional time)
  CONSTRAINT: if end is set, end must be after start
```

### 3.3 Geographic

```
#Address:
  line1 (required string)
  line2 (optional string)
  city (required string)
  state (2 uppercase letters)
  postal_code (5 digits, optionally with -4 extension)
  country (2 uppercase letters, default "US", ISO 3166-1 alpha-2)
  latitude (optional float, -90 to 90)
  longitude (optional float, -180 to 180)
  county (optional string — important for tax jurisdictions)
```

### 3.4 Identity and References

```
#EntityRef:
  entity_type (#EntityType)
  entity_id (required string)
  relationship (#RelationshipType)

#EntityType enum:
  "person" | "organization" | "portfolio" | "property" | "building" |
  "space" | "lease" | "lease_space" | "work_order" | "vendor" |
  "ledger_entry" | "journal_entry" | "account" | "bank_account" |
  "application" | "inspection" | "document"

#RelationshipType enum:
  "belongs_to" | "contains" | "managed_by" | "owned_by" |
  "leased_to" | "occupied_by" | "reported_by" | "assigned_to" |
  "billed_to" | "paid_by" | "performed_by" | "approved_by" |
  "guarantor_for" | "emergency_contact_for" | "employed_by" |
  "related_to" | "parent_of" | "child_of" | "sublease_of"
```

### 3.5 Audit

```
#AuditMetadata:
  created_by (required string — user ID, agent ID, or "system")
  updated_by (required string)
  created_at (time)
  updated_at (time)
  source ("user" | "agent" | "import" | "system" | "migration")
  correlation_id (optional string — links related changes across entities)
  agent_goal_id (optional string — if source == "agent", which goal triggered this)
```

### 3.6 Contact

```
#ContactMethod:
  type ("email" | "phone" | "sms" | "mail" | "portal")
  value (required string)
  primary (bool, default false)
  verified (bool, default false)
  opt_out (bool, default false — communication preference)
  label (optional string — "work", "home", "mobile", etc.)
```

---

## 4. Person Model — `ontology/person.cue`

The Person model represents all human and organizational actors. A single person can be a tenant, owner, vendor contact, and emergency contact simultaneously. Roles are relationships, not types.

### 4.1 Person

```
#Person:
  id (required string)
  first_name (required string)
  last_name (required string)
  display_name (string, default "{first_name} {last_name}")
  
  date_of_birth (optional time — required for tenant screening)
  ssn_last_four (optional string, exactly 4 digits — stored encrypted, only last 4 in domain model)
  
  contact_methods (list of #ContactMethod, minimum 1)
  preferred_contact ("email" | "sms" | "phone" | "mail" | "portal", default "email")
  
  language_preference (2 lowercase letters ISO 639-1, default "en")
  timezone (optional string — IANA timezone)
  do_not_contact (bool, default false — legal hold, agent must respect)
  
  identity_verified (bool, default false)
  verification_method (optional: "manual" | "id_check" | "credit_check" | "ssn_verify")
  verified_at (optional time)
  
  tags (optional list of strings)
  
  CONSTRAINTS:
    - If preferred_contact is "sms" or "phone", contact_methods must include at least one phone/sms entry
  
  audit: #AuditMetadata
```

### 4.2 Organization

```
#Organization:
  id (required string)
  legal_name (required string)
  dba_name (optional string — "Doing Business As")
  
  org_type ("management_company" | "ownership_entity" | "vendor" |
            "corporate_tenant" | "government_agency" | "hoa" |
            "investment_fund" | "other")
  
  tax_id (optional string — EIN/Tax ID, stored encrypted)
  tax_id_type (optional: "ein" | "ssn" | "itin" | "foreign")
  
  status ("active" | "inactive" | "suspended" | "dissolved")
  
  address (optional #Address)
  contact_methods (optional list of #ContactMethod)
  
  state_of_incorporation (optional, 2 uppercase letters)
  formation_date (optional time)
  
  management_license (optional string — for management companies)
  license_state (optional, 2 uppercase letters)
  license_expiry (optional time)
  
  audit: #AuditMetadata
```

### 4.3 PersonRole

Roles are relationships between a Person and other entities, not properties of the Person. A PersonRole captures context-specific attributes that apply when a Person acts in a particular capacity.

```
#PersonRole:
  id (required string)
  person_id (required string)
  role_type ("tenant" | "owner" | "property_manager" | "maintenance_tech" |
             "leasing_agent" | "accountant" | "vendor_contact" |
             "guarantor" | "emergency_contact" | "authorized_occupant" |
             "co_signer")
  
  scope_type ("organization" | "portfolio" | "property" | "building" | "space" | "lease")
  scope_id (required string)
  
  status ("active" | "inactive" | "pending" | "terminated")
  
  effective (#DateRange)
  
  attributes (optional — one of the role-specific attribute types below, determined by role_type)
  
  audit: #AuditMetadata
```

#### Role-Specific Attributes

```
#TenantAttributes:
  _type: "tenant"
  standing ("good" | "late" | "collections" | "eviction", default "good")
  occupancy_status ("occupying" | "vacated" | "never_occupied", default "occupying")
  liability_status ("active" | "released" | "guarantor_only", default "active")
  screening_status ("not_started" | "in_progress" | "approved" | "denied" | "conditional")
  screening_date (optional time)
  current_balance (optional #Money — computed from ledger)
  move_in_date (optional time)
  move_out_date (optional time)
  pet_count (optional int >= 0)
  vehicle_count (optional int >= 0)

#OwnerAttributes:
  _type: "owner"
  ownership_percent (float > 0 and <= 100)
  distribution_method ("ach" | "check" | "hold", default "ach")
  management_fee_percent (optional float >= 0 and <= 100)
  tax_reporting ("1099" | "k1" | "none", default "1099")
  reserve_amount (optional #NonNegativeMoney)

#ManagerAttributes:
  _type: "manager"
  license_number (optional string)
  license_state (optional 2 uppercase letters)
  approval_limit (optional #NonNegativeMoney — max they can approve without escalation)
  can_sign_leases (bool, default false)
  can_approve_expenses (bool, default true)

#GuarantorAttributes:
  _type: "guarantor"
  guarantee_type ("full" | "partial" | "conditional")
  guarantee_amount (optional #PositiveMoney — for partial guarantees)
  guarantee_term (optional #DateRange)
  credit_score (optional int 300-850)
```

IMPORTANT: The `occupancy_status` and `liability_status` on TenantAttributes handle the roommate-departure scenario. A tenant who moves out but remains legally liable has `occupancy_status: "vacated"` and `liability_status: "active"`. This avoids needing to model the departure on the Lease entity itself.

---

## 5. Property Model — `ontology/property.cue`

### CRITICAL DESIGN DECISION: Building is Optional, Space Replaces Unit

The hierarchy is:

```
Portfolio
  └── Property
       ├── Building (OPTIONAL grouping — some spaces have no building)
       │    └── Space
       └── Space (can exist directly under property, no building)
            └── Space (self-referential parent/child for nested spaces)
```

Building is NOT a required hierarchy level. It's a grouping. Rationale:
- Parking lots have rentable spaces but no building
- Mobile home parks have no buildings (tenants own the structures)
- Single-family homes don't benefit from a building layer
- But apartment complexes, commercial properties, and campuses need buildings for maintenance routing, CAM calculations, and permission scoping

Space replaces the concept of "Unit" as a universal term for any rentable (or non-rentable) location within a property. A residential apartment, commercial suite, parking spot, storage unit, bed-space, desk-space, mobile home lot, and common area are all Spaces with different `space_type` values.

### 5.1 Portfolio

```
#Portfolio:
  id (required string)
  name (required string)
  owner_id (required string — Organization ID of ownership entity)
  
  management_type ("self_managed" | "third_party" | "hybrid")
  
  requires_trust_accounting (bool)
  trust_bank_account_id (optional string)
  
  status ("active" | "inactive" | "onboarding" | "offboarding")
  
  default_payment_methods (optional list of: "ach" | "credit_card" | "check" | "cash" | "money_order")
  fiscal_year_start_month (int 1-12, default 1)
  
  CONSTRAINTS:
    - If requires_trust_accounting is true, trust_bank_account_id is required
  
  audit: #AuditMetadata
```

### 5.2 Property

```
#Property:
  id (required string)
  portfolio_id (required string)
  name (required string)
  address (#Address, required)
  
  property_type ("single_family" | "multi_family" | "commercial_office" |
                 "commercial_retail" | "mixed_use" | "industrial" |
                 "affordable_housing" | "student_housing" | "senior_living" |
                 "vacation_rental" | "mobile_home_park" | "self_storage" |
                 "coworking" | "data_center" | "medical_office")
  
  status ("active" | "inactive" | "under_renovation" | "for_sale" | "onboarding")
  
  year_built (int 1800-2030)
  total_square_footage (float > 0)
  total_spaces (int >= 1 — denormalized count, replaces total_units)
  lot_size_sqft (optional float > 0)
  stories (optional int >= 1)
  parking_spaces (optional int >= 0)
  
  jurisdiction_id (optional string — links to local ordinance rules)
  rent_controlled (bool, default false)
  compliance_programs (optional list of: "LIHTC" | "Section8" | "HUD" | "HOME" | "RAD" | "VASH" | "PBV")
  requires_lead_disclosure (bool, default false)
  
  chart_of_accounts_id (optional string — if different from portfolio default)
  bank_account_id (optional string — if different from portfolio trust account)
  
  insurance_policy_number (optional string)
  insurance_expiry (optional time)
  
  CONSTRAINTS:
    - If property_type is "single_family", total_spaces must be 1
    - If property_type is "affordable_housing", compliance_programs must have at least one entry
    - If rent_controlled is true, jurisdiction_id is required
    - If year_built < 1978, requires_lead_disclosure must be true
  
  audit: #AuditMetadata
```

### 5.3 Building

Building is an OPTIONAL grouping entity. Not all spaces belong to a building.

```
#Building:
  id (required string)
  property_id (required string)
  name (required string — "Building A", "Main Tower", "Parking Garage", "North Wing")
  
  building_type ("residential" | "commercial" | "mixed_use" |
                 "parking_structure" | "industrial" | "storage" | "auxiliary")
  
  address (optional #Address — may differ from property for multi-address campuses)
  
  status ("active" | "inactive" | "under_renovation")
  
  floors (optional int >= 1)
  year_built (optional int — could differ from property for additions)
  total_square_footage (optional float > 0)
  total_rentable_square_footage (optional float > 0 — for CAM calculations)
  
  audit: #AuditMetadata
```

### 5.4 Space

Space is the universal entity for any location within a property — rentable or not. It replaces the concept of "Unit" and covers residential apartments, commercial suites, parking spots, storage units, bed-spaces, desk-spaces, mobile home lots, and common areas.

```
#Space:
  id (required string)
  property_id (required string — ALWAYS required, every space belongs to a property)
  building_id (optional string — spaces may or may not be in a building)
  parent_space_id (optional string — self-referential for nested spaces:
                   apartment → bedrooms, food court → stalls, parking lot → spots)
  
  space_number (required string — "101", "Suite 200", "P-42", "Lot 7", "102-A")
  
  space_type ("residential_unit" | "commercial_office" | "commercial_retail" |
              "industrial" | "storage" | "parking" | "lot_pad" |
              "common_area" | "bed_space" | "desk_space")
  
  status ("vacant" | "occupied" | "notice_given" | "make_ready" |
          "down" | "model" | "reserved" | "owner_occupied")
  
  leasable (bool, default true — can this space directly attach to a lease?
            Parent spaces in by-the-bed configs are leasable: false because
            only their child bed-spaces are leased. Common areas are leasable: false.)
  
  # Physical
  square_footage (float > 0)
  bedrooms (optional int >= 0)
  bathrooms (optional float >= 0 — 1.5 = one full, one half bath)
  floor (optional int — attribute, not a hierarchy level)
  
  # Features
  amenities (optional list of strings)
  floor_plan (optional string — reference to floor plan template)
  ada_accessible (bool, default false)
  pet_friendly (bool, default true)
  furnished (bool, default false)
  
  # Specialized infrastructure — for medical, data center, restaurant, etc.
  specialized_infrastructure (optional list of:
    "medical_plumbing" | "clean_room" | "high_voltage" | "loading_dock" |
    "commercial_kitchen" | "server_room" | "cold_storage" | "hazmat_ventilation" |
    "grease_trap" | "exhaust_hood")
  
  # Financial
  market_rent (optional #NonNegativeMoney)
  
  # For affordable housing — space-level income restrictions
  ami_restriction (optional int 0-150 — % of Area Median Income)
  
  # Active lease (computed from LeaseSpace relationship traversal)
  active_lease_id (optional string)
  
  # Shared access flag for child spaces in by-the-bed/coworking
  shared_with_parent (bool, default false — tenant of this space can access parent space)
  
  CONSTRAINTS:
    - If status is "occupied", active_lease_id is required
    - If space_type is "residential_unit", bedrooms and bathrooms should be present
    - If space_type is "parking" or "storage" or "lot_pad", bedrooms defaults to 0, bathrooms defaults to 0
    - If space_type is "common_area", leasable defaults to false
    - If parent_space_id is set, building_id should match parent's building_id (or both null)
  
  audit: #AuditMetadata
```

---

## 6. Lease Model — `ontology/lease.cue`

The Lease is the central contractual entity. It connects one or more Spaces to one or more Persons via a many-to-many join entity (LeaseSpace) with effective dates and relationship semantics.

### CRITICAL DESIGN DECISIONS

1. **Lease ↔ Space is M2M via LeaseSpace.** A commercial tenant can lease multiple suites. A residential lease can include a parking spot. A tenant can move from Space A to Space B mid-lease.

2. **LeaseSpace is a first-class entity**, not just a join table. It carries effective dates, relationship types, and optional per-space financial data.

3. **State transitions are explicit named API operations**, not generic status updates. "ActivateLease" not "UpdateLease(status=active)".

4. **`lease_commencement_date` and `rent_commencement_date` are separate.** The legal lease may commence before rent starts (free rent periods, build-to-suit).

### 6.1 Lease

```
#Lease:
  id (required string)
  property_id (required string — denormalized for query efficiency)
  
  # Tenant references via PersonRole
  tenant_role_ids (list of strings, minimum 1)
  guarantor_role_ids (optional list of strings)
  
  lease_type ("fixed_term" | "month_to_month" |
              "commercial_nnn" | "commercial_nn" | "commercial_n" |
              "commercial_gross" | "commercial_modified_gross" |
              "affordable" | "section_8" | "student" |
              "ground_lease" | "short_term" | "membership")
  
  liability_type ("joint_and_several" | "individual" | "by_the_bed" | "proportional",
                  default "joint_and_several")
  
  status ("draft" | "pending_approval" | "pending_signature" | "active" |
          "expired" | "month_to_month_holdover" | "renewed" |
          "terminated" | "eviction")
  
  # Term
  term (#DateRange)
  
  # Commencement — these can differ from term dates
  lease_commencement_date (optional time — when legal obligations begin)
  rent_commencement_date (optional time — when rent payments begin, may be after commencement for free rent)
  
  # Financial
  base_rent (#NonNegativeMoney)
  security_deposit (#NonNegativeMoney)
  
  # Rent schedule — handles escalations, concessions, free periods, CPI adjustments
  rent_schedule (optional list of #RentScheduleEntry)
  
  # Recurring charges beyond base rent
  recurring_charges (optional list of #RecurringCharge)
  
  # Usage-based charges (metered utilities, data center power, RUBS)
  usage_charges (optional list of #UsageBasedCharge)
  
  # Late fee policy (can override property/portfolio defaults)
  late_fee_policy (optional #LateFeePolicy)
  
  # Percentage rent (retail)
  percentage_rent (optional #PercentageRent)
  
  # Commercial-specific
  cam_terms (optional #CAMTerms)
  tenant_improvement (optional #TenantImprovement)
  renewal_options (optional list of #RenewalOption)
  expansion_rights (optional list of #ExpansionRight)
  contraction_rights (optional list of #ContractionRight)
  
  # Affordable housing-specific
  subsidy (optional #SubsidyTerms)
  
  # Short-term rental-specific
  check_in_time (optional string — "3:00 PM")
  check_out_time (optional string — "11:00 AM")
  cleaning_fee (optional #NonNegativeMoney)
  platform_booking_id (optional string — Airbnb/VRBO reference)
  
  # Membership-specific (coworking)
  membership_tier (optional: "hot_desk" | "dedicated_desk" | "office" | "suite" | "virtual")
  
  # Sublease
  parent_lease_id (optional string — references master lease for subleases)
  is_sublease (bool, default false)
  sublease_billing ("direct_to_landlord" | "through_master_tenant", default "through_master_tenant")
  
  # Move-in / move-out
  move_in_date (optional time)
  move_out_date (optional time)
  notice_date (optional time)
  notice_required_days (int >= 0, default 30 — NOTE: this default is overridable via configuration)
  
  # Signing
  signing_method (optional: "electronic" | "wet_ink" | "both")
  signed_at (optional time)
  document_id (optional string)
  
  CONSTRAINTS:
    - If lease_type is "fixed_term" or "student", term must have an end date
    - If lease_type is "commercial_nnn" or "commercial_nn" or "commercial_n" or
      "commercial_gross" or "commercial_modified_gross", cam_terms is required
    - If lease_type is "commercial_nnn":
        cam_terms.includes_property_tax must be true
        cam_terms.includes_insurance must be true
        cam_terms.includes_utilities must be true
    - If lease_type is "commercial_nn":
        cam_terms.includes_property_tax must be true
        cam_terms.includes_insurance must be true
    - If lease_type is "commercial_n":
        cam_terms.includes_property_tax must be true
    - If lease_type is "section_8", subsidy is required
    - If status is "active", move_in_date is required
    - If status is "active" or "expired" or "renewed", signed_at is required
    - If is_sublease is true, parent_lease_id is required
    - If lease_type is "membership", LeaseSpace may have zero entries (e.g., hot desk with no assigned space)
    - If rent_commencement_date is set and lease_commencement_date is set,
      rent_commencement_date must be on or after lease_commencement_date
  
  audit: #AuditMetadata
```

### 6.2 LeaseSpace (First-Class Join Entity)

LeaseSpace connects Leases to Spaces with temporal and semantic context. It is NOT just a join table.

```
#LeaseSpace:
  id (required string)
  lease_id (required string)
  space_id (required string)
  
  is_primary (bool, default true — primary space vs ancillary)
  
  relationship ("primary" | "expansion" | "sublease" | "shared_access" |
                "parking" | "storage" | "loading_dock" | "rooftop" |
                "patio" | "signage" | "included" | "membership")
  
  effective (#DateRange — when this space was part of this lease)
  
  square_footage_leased (optional float — for partial-floor commercial leases)
  
  audit: #AuditMetadata
```

### 6.3 Rent Schedule and Adjustments

```
#RentScheduleEntry:
  effective_period (#DateRange)
  description (required string — "Year 1", "Move-in concession", "Year 2 CPI adjustment")
  charge_code (required string — links to chart of accounts)
  
  # Exactly one of these two:
  fixed_amount (optional #NonNegativeMoney — for known amounts)
  adjustment (optional #RentAdjustment — for formula-based amounts)

#RentAdjustment:
  method ("cpi" | "fixed_percent" | "fixed_amount_increase" | "market_review")
  base_amount (#NonNegativeMoney — starting point for the calculation)
  
  # For CPI
  cpi_index (optional: "CPI-U" | "CPI-W" | "regional")
  cpi_floor (optional float >= 0 — minimum annual increase, e.g. 2%)
  cpi_ceiling (optional float > 0 — maximum annual increase, e.g. 5%)
  
  # For fixed percent
  percent_increase (optional float > 0)
  
  # For fixed amount
  amount_increase (optional #PositiveMoney)
  
  # For market review
  market_review_mechanism (optional string — description of appraisal process)
```

### 6.4 Recurring Charges

```
#RecurringCharge:
  id (required string)
  charge_code (required string — links to chart of accounts)
  description (required string)
  amount (#NonNegativeMoney)
  frequency ("monthly" | "quarterly" | "annually" | "one_time")
  effective_period (#DateRange)
  taxable (bool, default false)
  space_id (optional string — which space this charge relates to, for per-space rates
            in multi-space leases like flex industrial with different office vs warehouse rates)
```

### 6.5 Usage-Based Charges

For metered utilities, data center power, RUBS, and any consumption-based billing.

```
#UsageBasedCharge:
  id (required string)
  charge_code (required string)
  description (required string)
  unit_of_measure ("kwh" | "gallon" | "cubic_foot" | "therm" | "hour" | "gb")
  rate_per_unit (#PositiveMoney)
  meter_id (optional string)
  billing_frequency ("monthly" | "quarterly")
  cap (optional #NonNegativeMoney — maximum charge per period)
  space_id (optional string — which space this meter/charge relates to)
```

### 6.6 Late Fee Policy

```
#LateFeePolicy:
  grace_period_days (int >= 0, default 5)
  fee_type ("flat" | "percent" | "per_day" | "tiered")
  flat_amount (optional #NonNegativeMoney)
  percent (optional float > 0 and <= 100)
  per_day_amount (optional #NonNegativeMoney)
  max_fee (optional #NonNegativeMoney)
  tiers (optional list of:
    { days_late_min (int >= 0), days_late_max (int), amount (#NonNegativeMoney) }
  )
```

### 6.7 Percentage Rent (Retail)

```
#PercentageRent:
  rate (float > 0 and <= 100 — typically 3-8%)
  breakpoint_type ("natural" | "artificial")
  natural_breakpoint (optional #NonNegativeMoney — equals base_rent / percentage_rate)
  artificial_breakpoint (optional #NonNegativeMoney)
  reporting_frequency ("monthly" | "quarterly" | "annually")
  audit_rights (bool, default true — landlord can audit tenant's books)
```

### 6.8 CAM Terms

```
#CAMTerms:
  reconciliation_type ("estimated_with_annual_reconciliation" | "fixed" | "actual")
  pro_rata_share_percent (float > 0 and <= 100)
  estimated_monthly_cam (#NonNegativeMoney)
  annual_cap (optional #NonNegativeMoney)
  includes_property_tax (bool)
  includes_insurance (bool)
  includes_utilities (bool)
  excluded_categories (optional list of strings)
  
  # Gross lease expense stops
  base_year (optional int — for gross leases, the year against which overages are measured)
  base_year_expenses (optional #NonNegativeMoney — total operating expenses in base year)
  expense_stop (optional #NonNegativeMoney — fixed dollar cap instead of base year)
  
  # Per-category controls for modified gross
  category_terms (optional list of #CAMCategoryTerms)
  
  CONSTRAINTS:
    - If reconciliation_type is "fixed", annual_cap is not allowed (no reconciliation = no cap)
    - base_year and expense_stop are mutually exclusive (use one or the other)
```

```
#CAMCategoryTerms:
  category ("property_tax" | "insurance" | "utilities" | "janitorial" |
            "landscaping" | "security" | "management_fee" | "repairs" |
            "snow_removal" | "elevator" | "hvac_maintenance" | "other")
  tenant_pays (bool)
  landlord_cap (optional #NonNegativeMoney — landlord pays up to this, tenant pays overage)
  tenant_cap (optional #NonNegativeMoney — tenant's maximum contribution)
  escalation (optional float — annual increase cap on this category)
```

### 6.9 Tenant Improvement

```
#TenantImprovement:
  allowance (#NonNegativeMoney)
  amortized (bool, default false)
  amortization_term_months (optional int > 0)
  interest_rate_percent (optional float >= 0)
  completion_deadline (optional time)
```

### 6.10 Renewal Options

```
#RenewalOption:
  option_number (int >= 1)
  term_months (int > 0)
  rent_adjustment ("fixed" | "cpi" | "percent_increase" | "market")
  fixed_rent (optional #NonNegativeMoney)
  percent_increase (optional float >= 0)
  cpi_floor (optional float >= 0)
  cpi_ceiling (optional float > 0)
  notice_required_days (int >= 0, default 90)
  must_exercise_by (optional time)
```

### 6.11 Expansion and Contraction Rights

```
#ExpansionRight:
  type ("first_right_of_refusal" | "first_right_to_negotiate" | "must_take" | "option")
  target_space_ids (list of strings — Space IDs the tenant has rights to)
  exercise_deadline (optional time)
  terms (optional string — description of economic terms)
  notice_required_days (int >= 0)

#ContractionRight:
  minimum_retained_sqft (float > 0 — floor on how much space tenant must keep)
  earliest_exercise_date (time)
  penalty (optional #NonNegativeMoney — early contraction penalty)
  notice_required_days (int >= 0)
```

### 6.12 Subsidy Terms (Affordable Housing)

```
#SubsidyTerms:
  program ("section_8" | "pbv" | "vash" | "home" | "lihtc")
  housing_authority (required string)
  hap_contract_id (optional string)
  contract_rent (#NonNegativeMoney — total per HAP contract)
  tenant_portion (#NonNegativeMoney)
  subsidy_portion (#NonNegativeMoney)
  utility_allowance (#NonNegativeMoney)
  annual_recert_date (optional time)
  income_limit_ami_percent (int > 0 and <= 150)
```

### 6.13 Application

```
#Application:
  id (required string)
  property_id (required string)
  space_id (optional string — may apply to property without specific space)
  applicant_person_id (required string)
  
  status ("submitted" | "screening" | "under_review" | "approved" |
          "conditionally_approved" | "denied" | "withdrawn" | "expired")
  
  desired_move_in (time)
  desired_lease_term_months (int > 0)
  
  screening_request_id (optional string)
  screening_completed (optional time)
  credit_score (optional int 300-850)
  background_clear (bool, default false)
  income_verified (bool, default false)
  income_to_rent_ratio (optional float >= 0)
  
  decision_by (optional string — Person ID)
  decision_at (optional time)
  decision_reason (optional string)
  conditions (optional list of strings — for conditional approval)
  
  application_fee (#NonNegativeMoney)
  fee_paid (bool, default false)
  
  CONSTRAINTS:
    - If status is "approved" or "conditionally_approved" or "denied":
      decision_by and decision_at are required
    - If status is "denied": decision_reason is required (fair housing compliance)
  
  audit: #AuditMetadata
```

---

## 7. Accounting Model — `ontology/accounting.cue`

### 7.1 Account (Chart of Accounts)

```
#Account:
  id (required string)
  account_number (required string — "1000", "4100.001")
  name (required string)
  description (optional string)
  
  account_type ("asset" | "liability" | "equity" | "revenue" | "expense")
  
  account_subtype ("cash" | "accounts_receivable" | "prepaid" | "fixed_asset" |
                   "accumulated_depreciation" | "other_asset" |
                   "accounts_payable" | "accrued_liability" | "unearned_revenue" |
                   "security_deposits_held" | "other_liability" |
                   "owners_equity" | "retained_earnings" | "distributions" |
                   "rental_income" | "other_income" | "cam_recovery" | "percentage_rent_income" |
                   "operating_expense" | "maintenance_expense" | "utility_expense" |
                   "management_fee_expense" | "depreciation_expense" | "other_expense")
  
  parent_account_id (optional string — for hierarchy)
  depth (int >= 0 — 0 = top-level)
  
  dimensions (optional #AccountDimensions)
  
  normal_balance ("debit" | "credit")
  is_header (bool, default false)
  is_system (bool, default false — system accounts cannot be deleted)
  allows_direct_posting (bool, default true)
  
  status ("active" | "inactive" | "archived")
  
  is_trust_account (bool, default false)
  trust_type (optional: "operating" | "security_deposit" | "escrow")
  
  budget_amount (optional #Money)
  tax_line (optional string — maps to tax form line)
  
  CONSTRAINTS:
    - If account_type is "asset" or "expense": normal_balance must be "debit"
    - If account_type is "liability" or "equity" or "revenue": normal_balance must be "credit"
    - If is_header is true: allows_direct_posting must be false
    - If is_trust_account is true: trust_type is required
  
  audit: #AuditMetadata

#AccountDimensions:
  entity_id (optional string — legal entity: LLC, fund)
  property_id (optional string — property-level tracking)
  dimension_1 (optional string — commonly department)
  dimension_2 (optional string — commonly cost center)
  dimension_3 (optional string — commonly project/job code)
```

### 7.2 Ledger Entry

Ledger entries are IMMUTABLE. Errors are corrected with adjustment entries, never by modifying existing entries.

```
#LedgerEntry:
  id (required string)
  account_id (required string)
  
  entry_type ("charge" | "payment" | "credit" | "adjustment" |
              "refund" | "deposit" | "nsf" | "write_off" |
              "late_fee" | "management_fee" | "owner_draw")
  
  amount (#Money)
  
  journal_entry_id (required string — double-entry: every entry belongs to a journal entry)
  
  effective_date (time — when the economic event occurred)
  posted_date (time — when it was recorded)
  
  description (required string)
  charge_code (required string — links to CoA)
  memo (optional string)
  
  # Dimensional references
  property_id (required string — always required)
  space_id (optional string)
  lease_id (optional string)
  person_id (optional string — tenant, owner, vendor)
  
  # Bank/trust
  bank_account_id (optional string)
  bank_transaction_id (optional string — reference to bank feed)
  
  # Reconciliation
  reconciled (bool, default false)
  reconciliation_id (optional string)
  reconciled_at (optional time)
  
  # For adjustments
  adjusts_entry_id (optional string — what entry is this correcting)
  
  CONSTRAINTS:
    - If entry_type is "payment" or "refund" or "nsf": person_id is required
    - If entry_type is "charge" or "late_fee": lease_id is required
    - If entry_type is "adjustment": adjusts_entry_id is required
    - If reconciled is true: reconciliation_id and reconciled_at are required
    - IMMUTABILITY: LedgerEntries cannot be updated or deleted (enforced at Ent hook level)
  
  audit: #AuditMetadata
```

### 7.3 Journal Entry

Groups LedgerEntries that must balance (debits = credits).

```
#JournalEntry:
  id (required string)
  
  entry_date (time)
  posted_date (time)
  
  description (required string)
  
  source_type ("manual" | "auto_charge" | "payment" | "bank_import" |
               "cam_reconciliation" | "depreciation" | "accrual" |
               "intercompany" | "management_fee" | "system")
  source_id (optional string)
  
  status ("draft" | "pending_approval" | "posted" | "voided")
  approved_by (optional string)
  approved_at (optional time)
  
  batch_id (optional string)
  entity_id (optional string — legal entity)
  property_id (optional string)
  
  reverses_journal_id (optional string)
  reversed_by_journal_id (optional string)
  
  lines (list of #JournalLine, minimum 2)
  
  CONSTRAINTS:
    - If status is "posted" and source_type is "manual": approved_by and approved_at are required
    - If status is "voided": reversed_by_journal_id is required
    - Sum of all line debits must equal sum of all line credits (enforced at Ent hook)
  
  audit: #AuditMetadata

#JournalLine:
  account_id (required string)
  debit (optional #NonNegativeMoney)
  credit (optional #NonNegativeMoney)
  description (optional string)
  dimensions (optional #AccountDimensions)
  # Must have exactly one of debit or credit (not both, not neither)
```

### 7.4 Bank Account

```
#BankAccount:
  id (required string)
  name (required string)
  
  account_type ("operating" | "trust" | "security_deposit" | "escrow" | "reserve")
  
  gl_account_id (required string — linked CoA account)
  
  bank_name (required string)
  routing_number (optional string — stored encrypted)
  account_number_last_four (4 digits)
  
  portfolio_id (optional string)
  property_id (optional string)
  entity_id (optional string — legal entity)
  
  status ("active" | "inactive" | "frozen" | "closed")
  
  current_balance (optional #Money — computed from reconciled transactions)
  last_reconciled_at (optional time)
  
  is_trust (bool, default false)
  trust_state (optional 2 uppercase letters — regulations vary by state)
  commingling_allowed (bool, default false)
  
  CONSTRAINTS:
    - If is_trust is true: trust_state is required and commingling_allowed must be false
  
  audit: #AuditMetadata
```

### 7.5 Reconciliation

```
#Reconciliation:
  id (required string)
  bank_account_id (required string)
  
  period_start (time)
  period_end (time)
  
  statement_balance (#Money)
  system_balance (#Money)
  difference (#Money)
  
  status ("in_progress" | "balanced" | "unbalanced" | "approved")
  
  matched_transaction_count (int >= 0)
  unmatched_transaction_count (int >= 0)
  
  completed_by (optional string)
  completed_at (optional time)
  approved_by (optional string)
  approved_at (optional time)
  
  CONSTRAINTS:
    - If status is "balanced" or "approved": difference.amount_cents must be 0
  
  audit: #AuditMetadata
```

---

## 8. Configuration Schema — `ontology/config_schema.cue`

This defines what's TUNABLE at runtime and the valid ranges. The ontology defines the envelope of valid configuration; runtime values live in the database with a cascade resolution: platform → organization → portfolio → property → lease_type → jurisdiction (last wins).

```
#LeaseConfiguration:
  notice_required_days (int 0-365, default 30)
  late_fee_grace_period_days (int 0-30, default 5)
  late_fee_type ("flat" | "percent" | "per_day", default "flat")
  late_fee_flat_amount (optional #NonNegativeMoney)
  late_fee_percent (optional float > 0 and <= 25)
  security_deposit_max_months (float > 0 and <= 6, default 2)
  auto_renewal_enabled (bool, default false)
  auto_renewal_notice_days (int 30-180, default 60)
  allow_partial_payments (bool, default true)
  minimum_payment_percent (optional float > 0 and <= 100)

#PropertyConfiguration:
  maintenance_auto_approve_limit (#NonNegativeMoney, max $10,000)
  screening_required (bool, default true)
  screening_income_ratio (float 1-10, default 3)
  pet_policy ("allowed" | "restricted" | "prohibited", default "allowed")
  max_pets (optional int 0-10)
  pet_deposit (optional #NonNegativeMoney)
  pet_rent (optional #NonNegativeMoney)

#ConfigurationScope:
  level ("platform" | "organization" | "portfolio" | "property" | "lease_type" | "jurisdiction")
  # Lower levels override higher. Jurisdiction overrides everything (legal requirements trump preferences).
```

IMPORTANT: Do NOT bake specific values like "30 days notice" into the entity definitions as if they're structural truths. The ontology defines the shape and valid ranges. Configuration defines the actual values per company/property/jurisdiction. The codegen should generate a typed configuration resolver that validates against the CUE schema.

---

## 9. State Machines — `ontology/state_machines.cue`

Every entity with a status field has an explicit state machine. These are generated into Ent hooks that reject invalid transitions at the persistence layer.

```
#LeaseTransitions:
  draft → [pending_approval, pending_signature, terminated]
  pending_approval → [draft, pending_signature, terminated]
  pending_signature → [active, draft, terminated]
  active → [expired, month_to_month_holdover, terminated, eviction]
  expired → [active, month_to_month_holdover, renewed, terminated]
  month_to_month_holdover → [active, renewed, terminated, eviction]
  renewed → [] (terminal — a new lease is created)
  terminated → [] (terminal)
  eviction → [terminated]

#SpaceTransitions:
  vacant → [occupied, make_ready, down, model, reserved]
  occupied → [notice_given]
  notice_given → [make_ready, occupied] (can rescind notice)
  make_ready → [vacant, down]
  down → [make_ready, vacant]
  model → [vacant, occupied]
  reserved → [vacant, occupied]
  owner_occupied → [vacant] (owner moves out)

#ApplicationTransitions:
  submitted → [screening, withdrawn]
  screening → [under_review, withdrawn]
  under_review → [approved, conditionally_approved, denied, withdrawn]
  approved → [expired]
  conditionally_approved → [approved, denied, withdrawn, expired]
  denied → [] (terminal)
  withdrawn → [] (terminal)
  expired → [] (terminal)

#JournalEntryTransitions:
  draft → [pending_approval, posted] (auto-generated can skip approval)
  pending_approval → [posted, draft]
  posted → [voided]
  voided → [] (terminal)

#PortfolioTransitions:
  onboarding → [active]
  active → [inactive, offboarding]
  inactive → [active, offboarding]
  offboarding → [inactive]

#PropertyTransitions:
  onboarding → [active]
  active → [inactive, under_renovation, for_sale]
  inactive → [active]
  under_renovation → [active, for_sale]
  for_sale → [active, inactive]

#BuildingTransitions:
  active → [inactive, under_renovation]
  inactive → [active]
  under_renovation → [active]

#PersonRoleTransitions:
  pending → [active, terminated]
  active → [inactive, terminated]
  inactive → [active, terminated]
  terminated → [] (terminal)

#BankAccountTransitions:
  active → [inactive, frozen, closed]
  inactive → [active, closed]
  frozen → [active, closed]
  closed → [] (terminal)

#ReconciliationTransitions:
  in_progress → [balanced, unbalanced]
  balanced → [approved, in_progress] (reopen if errors found)
  unbalanced → [in_progress]
  approved → [] (terminal)
```

---

## 10. Relationships — `ontology/relationships.cue`

These define all edges between entities. Each relationship drives Ent edge generation, permission path evaluation, agent reasoning, and event routing.

Format: `from → to (edge_name, cardinality, required?, semantic, inverse_name)`

### Portfolio relationships

```
Portfolio → Property ("properties", O2M, "Portfolio contains Properties", inverse: "portfolio")
Portfolio → Organization ("owner", M2O, required, "Portfolio is owned by Organization", inverse: "owned_portfolios")
Portfolio → BankAccount ("trust_account", O2O, "Portfolio uses BankAccount for trust funds", inverse: "trust_portfolio")
```

### Property relationships

```
Property → Building ("buildings", O2M, "Property has Buildings", inverse: "property")
Property → Space ("spaces", O2M, "Property contains Spaces", inverse: "property")
Property → BankAccount ("bank_account", M2O, "Property uses BankAccount", inverse: "properties")
Property → Application ("applications", O2M, "Property receives Applications", inverse: "property")
```

### Building relationships

```
Building → Space ("spaces", O2M, "Building contains Spaces", inverse: "building")
```

### Space relationships

```
Space → Space ("children", O2M, "Space contains child Spaces (apt→bedrooms)", inverse: "parent_space")
Space → Lease ("leases", M2M via LeaseSpace, "Space has Leases over time", inverse: "spaces")
Space → Application ("applications", O2M, "Space receives Applications", inverse: "space")
```

NOTE: There are TWO paths from Property to Space:
- Property → Space (direct, for spaces not in a building)
- Property → Building → Space (via building)
Both are valid. The property_id on Space is always set regardless of whether building_id is set.

### Lease relationships

```
Lease → PersonRole ("tenant_roles", M2M, "Lease is held by tenant PersonRoles", inverse: "leases")
Lease → PersonRole ("guarantor_roles", M2M, "Lease is guaranteed by guarantor PersonRoles", inverse: "guaranteed_leases")
Lease → LedgerEntry ("ledger_entries", O2M, "Lease generates LedgerEntries", inverse: "lease")
Lease → Application ("application", O2O, "Lease originated from Application", inverse: "resulting_lease")
Lease → Lease ("subleases", O2M, "Master lease has subleases", inverse: "parent_lease")
```

### LeaseSpace relationships

```
LeaseSpace → Lease ("lease", M2O, required, "LeaseSpace belongs to Lease", inverse: "lease_spaces")
LeaseSpace → Space ("space", M2O, required, "LeaseSpace references Space", inverse: "lease_spaces")
```

### Person relationships

```
Person → PersonRole ("roles", O2M, "Person has Roles", inverse: "person")
Person → Organization ("organizations", M2M, "Person is affiliated with Organizations", inverse: "people")
Person → Application ("applications", O2M, "Person submits Applications", inverse: "applicant")
Person → LedgerEntry ("ledger_entries", O2M, "Person has financial entries", inverse: "person")
```

### Organization relationships

```
Organization → Organization ("subsidiaries", O2M, "Organization has subsidiaries", inverse: "parent_org")
Organization → Portfolio ("owned_portfolios", O2M, "Organization owns Portfolios", inverse: "owner")
```

### Accounting relationships

```
Account → Account ("children", O2M, "Account has sub-Accounts", inverse: "parent")
LedgerEntry → JournalEntry ("journal_entry", M2O, required, "Entry belongs to JournalEntry", inverse: "lines")
LedgerEntry → Account ("account", M2O, required, "Entry posts to Account", inverse: "entries")
LedgerEntry → Property ("property", M2O, required, "Entry relates to Property", inverse: "ledger_entries")
LedgerEntry → Space ("space", M2O, "Entry relates to Space", inverse: "ledger_entries")
LedgerEntry → Lease ("lease", M2O, "Entry relates to Lease", inverse: "ledger_entries")
LedgerEntry → Person ("person", M2O, "Entry relates to Person", inverse: "ledger_entries")
BankAccount → Account ("gl_account", M2O, required, "BankAccount tracked via GL Account", inverse: "bank_accounts")
Reconciliation → BankAccount ("bank_account", M2O, required, "Reconciliation for BankAccount", inverse: "reconciliations")
```

---

## 11. Type Projection Rules — `codegen/entgen.cue`

The codegen mapping layer defines how ontological types project into Ent field types. The ontology stays clean; the projection layer handles storage decisions.

### 11.1 Composite Type Projections

```
#Money → flatten into two columns:
  {field_name}_cents (int64)
  {field_name}_currency (string, default "USD")

#NonNegativeMoney → same as #Money with validation: cents >= 0
#PositiveMoney → same as #Money with validation: cents > 0

#Address → flatten with prefix "address_":
  address_line1 (string)
  address_line2 (optional string)
  address_city (string)
  address_state (string)
  address_postal_code (string)
  address_country (string, default "US")
  address_latitude (optional float)
  address_longitude (optional float)
  address_county (optional string)

#DateRange → flatten:
  {field_name}_start (time)
  {field_name}_end (optional time)

#AuditMetadata → Ent mixin (AuditMixin):
  id (UUID, auto-generated, immutable)
  created_by (string)
  updated_by (string)
  created_at (time, auto-set)
  updated_at (time, auto-update)
  source (enum)
  correlation_id (optional string)
  agent_goal_id (optional string)
```

### 11.2 Complex Embedded Types

These types are stored as JSON columns with Go struct typing:

```
#RentScheduleEntry → JSON column "rent_schedule" (array)
#RecurringCharge → JSON column "recurring_charges" (array)
#UsageBasedCharge → JSON column "usage_charges" (array)
#LateFeePolicy → JSON column "late_fee_policy" (object)
#PercentageRent → JSON column "percentage_rent" (object)
#CAMTerms → JSON column "cam_terms" (object)
#CAMCategoryTerms → embedded within CAMTerms JSON
#TenantImprovement → JSON column "tenant_improvement" (object)
#RenewalOption → JSON column "renewal_options" (array)
#ExpansionRight → JSON column "expansion_rights" (array)
#ContractionRight → JSON column "contraction_rights" (array)
#SubsidyTerms → JSON column "subsidy" (object)
#ContactMethod → JSON column "contact_methods" (array)
Role-specific attributes → JSON column "attributes" (object, discriminated by _type)
```

### 11.3 Reconstitution at API Boundary

The codegen produces `toEnt` and `toProto` mapping functions. The API speaks the ontology's language (single `#Money` object), while the database stores the flattened projection (two columns). The mapping is generated, not hand-written.

---

## 12. API Design Principles — `codegen/apigen.cue`

### 12.1 Service Organization

One service per domain area:

```
PersonService:       Person, Organization, PersonRole      /v1/persons
PropertyService:     Portfolio, Property, Building, Space  /v1/properties
LeaseService:        Lease, LeaseSpace, Application        /v1/leases
AccountingService:   Account, LedgerEntry, JournalEntry,   /v1/accounting
                     BankAccount, Reconciliation
```

### 12.2 State Transitions as Named Operations

Every state transition is its own RPC with specific request type, auth, and event:

```
LeaseService:
  CreateLease         POST   /v1/leases
  GetLease            GET    /v1/leases/{id}
  ListLeases          GET    /v1/leases
  UpdateLease         PATCH  /v1/leases/{id}
  SearchLeases        POST   /v1/leases/search
  SubmitForApproval   POST   /v1/leases/{id}/submit
  ApproveLease        POST   /v1/leases/{id}/approve
  SendForSignature    POST   /v1/leases/{id}/sign
  ActivateLease       POST   /v1/leases/{id}/activate
  RecordNotice        POST   /v1/leases/{id}/notice
  RenewLease          POST   /v1/leases/{id}/renew
  TerminateLease      POST   /v1/leases/{id}/terminate
  InitiateEviction    POST   /v1/leases/{id}/evict
  GetLeaseLedger      GET    /v1/leases/{id}/ledger
  RecordPayment       POST   /v1/leases/{id}/payments
  PostCharge          POST   /v1/leases/{id}/charges
  ApplyCredit         POST   /v1/leases/{id}/credits
```

### 12.3 Side Effects in Responses

Operations that cause downstream changes report them:

```
ActivateLeaseResponse:
  lease: Lease
  side_effects: [
    {entity_type: "Space", entity_id: "...", description: "Space status changed to occupied"},
    {entity_type: "LedgerEntry", description: "Security deposit charge created"},
    {entity_type: "LedgerEntry", description: "First month rent charge scheduled"},
  ]
```

### 12.4 Agent Tool Generation

Every API operation generates a corresponding agent tool definition with:
- Input schema derived from request spec (with enum constraints)
- Description written for LLM comprehension
- `when_to_use` guidance
- `common_mistakes` to avoid

---

## 13. Event Catalog — `codegen/eventgen.cue`

### Person Events
PersonCreated, PersonUpdated, PersonRoleAssigned, PersonRoleDeactivated, PersonRoleTerminated, OrganizationCreated, OrganizationUpdated

### Property Events
PortfolioCreated, PortfolioActivated, PropertyCreated, PropertyUpdated, PropertyStatusChanged, BuildingCreated, BuildingUpdated, SpaceCreated, SpaceUpdated, SpaceStatusChanged

### Lease Events
LeaseCreated, LeaseSubmittedForApproval, LeaseApproved, LeaseSentForSignature, LeaseSigned, LeaseActivated, TenantNoticeRecorded, LeaseRenewed, LeaseTerminated, EvictionInitiated, LeaseSpaceAdded, LeaseSpaceRemoved, ApplicationSubmitted, ApplicationScreeningComplete, ApplicationApproved, ApplicationDenied

### Accounting Events
ChargePosted, PaymentRecorded, CreditApplied, LateFeeAssessed, NSFRecorded, WriteOffPosted, JournalEntryPosted, JournalEntryVoided, ReconciliationCompleted, ReconciliationApproved, ManagementFeeCalculated, OwnerDistributionProcessed

### Event Envelope

Every event carries: event_id, event_type, entity_type, entity_id, occurred_at, recorded_at, correlation_id, causation_id, actor_id, actor_type, agent_goal_id, payload, changed_fields, previous_values.

---

## 14. Permission Model — `codegen/authzgen.cue`

### 14.1 Access Paths

Authorization is determined by traversing the relationship graph from the user to the target entity:

```
Person → PersonRole (scoped to) → Scope Entity → (contains) → Target Entity
```

Examples:

```
View a Property:      Person → Role(viewer+) → Portfolio → contains → Property
Create a Lease:       Person → Role(manager+) → Property → contains → Space
Activate a Lease:     Person → Role(manager+) → Property → contains → Space → has → Lease
Post a Journal Entry: Person → Role(accountant+) → Portfolio → contains → Property
Record a Payment:     Person → Role(manager+) → Property → ... → Lease
```

### 14.2 Role Hierarchy

```
Organization Admin
  └── Portfolio Admin
       └── Property Manager
            ├── Leasing Agent (leasing operations only)
            ├── Maintenance Coordinator (work orders only)
            └── Accountant (financial operations only)
       └── Viewer (read-only)
```

Higher roles inherit permissions of lower roles within their scope.

### 14.3 Agent Authorization

AI agents use the same PersonRole system. An agent has a Person record (source: "system"), PersonRole assignments with explicit scopes, and an approval_limit on ManagerAttributes.

---

## 15. Implementation Sequence

### Phase 1: Foundation (Weeks 1-2)
- Set up CUE toolchain, project structure
- Implement common.cue with all shared types
- Build cmd/entgen (CUE → Ent schema generator) with type projection rules
- Validate: Property + Space + simple Lease round-trip through Ent to Postgres

### Phase 2: Core Ontology (Weeks 2-4)
- Complete all domain model CUE files (person, property, lease, accounting)
- Complete relationships.cue and state_machines.cue
- Run cue vet, resolve inconsistencies
- Generate full Ent schema set, verify migrations

### Phase 3: API Layer (Weeks 4-6)
- Build cmd/apigen (CUE → Connect-RPC + OpenAPI + agent tools)
- Define services in apigen.cue
- Generate protos, handler scaffolds
- Implement LeaseService handlers as reference implementation

### Phase 4: Events + Projections (Weeks 6-8)
- Build cmd/eventgen, set up NATS JetStream
- Implement Ent hooks for event emission
- Build graph sync (Neo4j) and search sync (Meilisearch) workers

### Phase 5: Permissions (Weeks 8-10)
- Build cmd/authzgen, implement OPA rules
- Implement Ent privacy policies
- Integrate with PersonRole system

### Phase 6: Agent Integration (Weeks 10-12)
- Build cmd/agentgen (ONTOLOGY.md, STATE_MACHINES.md, TOOLS.md)
- Integrate tools with Propeller agent runtime
- Test agent goal execution against live system

---

## Appendix A: Stress Test Results

This ontology was validated against the following property types and lease structures:

| Scenario | Result |
|---|---|
| Single-family rental (with ADU) | ✅ Works |
| Duplex/fourplex (owner-occupied unit) | ✅ Works (added owner_occupied status) |
| Garden-style apartment complex | ✅ Works |
| Mid/high-rise apartment | ✅ Works |
| Mixed-use building (retail + residential) | ✅ Works |
| Student housing (by-the-bed) | ✅ Works (parent/child Space, leasable flag) |
| Student housing (mixed whole-unit and by-the-bed) | ✅ Works |
| Senior living / assisted living | ✅ Works |
| Affordable housing (LIHTC, Section 8) | ✅ Works |
| Vacation / short-term rental | ✅ Works (added short_term lease type) |
| Manufactured housing / mobile home park | ✅ Works (lot_pad space type, no building) |
| Commercial office (multi-tenant) | ✅ Works |
| Commercial office (full-floor tenant) | ✅ Works (M2M LeaseSpace) |
| Commercial office (multi-floor expansion) | ✅ Works |
| Retail strip mall with pad sites | ✅ Works |
| Enclosed mall with food court stalls | ✅ Works (parent/child Space) |
| Industrial / warehouse | ✅ Works |
| Flex space (office + warehouse, different rates) | ✅ Works (RecurringCharge.space_id) |
| Medical office building | ✅ Works (specialized_infrastructure) |
| Data center (rack/cage/suite) | ✅ Works (added UsageBasedCharge) |
| Self-storage facility | ✅ Works |
| Coworking (hot desk, dedicated, private office) | ⚠️ Partial (membership lease; hourly bookings out of scope) |
| Triple Net (NNN) lease | ✅ Works |
| Double Net (NN) lease | ✅ Works |
| Single Net (N) lease | ✅ Works |
| Gross / Full Service lease | ✅ Works (added base_year, expense_stop) |
| Modified Gross lease | ✅ Works (added CAMCategoryTerms) |
| Percentage rent (retail) | ✅ Works (added PercentageRent) |
| Graduated / step-up lease | ✅ Works (RentScheduleEntry) |
| CPI-indexed lease | ✅ Works (added RentAdjustment) |
| Ground lease | ✅ Works (added ground_lease type) |
| Sublease (standard) | ✅ Works (parent_lease_id) |
| Sublease (direct-to-landlord billing) | ✅ Works (added sublease_billing) |
| Commercial expansion (mid-term) | ✅ Works (LeaseSpace with effective dates) |
| Commercial contraction | ✅ Works (added ContractionRight) |
| Joint and several liability | ✅ Works (liability_type on Lease) |
| Roommate departure mid-lease | ✅ Works (occupancy_status + liability_status on TenantAttributes) |
| Build-to-suit | ✅ Works (lease_commencement vs rent_commencement dates) |
| Flex/hybrid post-COVID | ✅ Works (added ExpansionRight) |

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