# Propeller UI Generation Specification v2

**Version:** 2.0  
**Date:** February 25, 2026  
**Author:** Matthew Baird, CTO — AppFolio  
**Status:** For Claude Code Implementation  
**Depends on:** propeller-ontology-spec-v2.md

---

## 1. Purpose

This spec defines deterministic rules for generating UI components from the CUE ontology. The system has two layers:

**Layer 1: UI Schema Generator (`cmd/uigen`).** Reads ontology CUE files, applies mapping rules, outputs a framework-agnostic UI schema (JSON). This is the hard part — all the ontological logic lives here. It knows nothing about Svelte, React, or any framework.

**Layer 2: Svelte Renderer (`cmd/uirender`).** Reads the UI schema, outputs Svelte components using Skeleton UI and Tailwind CSS. This is the thin part — roughly 500 lines of template logic. Swapping to a different framework means writing a new renderer, not touching the schema generator.

```
Ontology (CUE)
    ↓
cmd/uigen → UI Schema (framework-agnostic JSON)
    ↓
cmd/uirender → Svelte + Skeleton + Tailwind components
```

Both run in `make generate`. One CUE change regenerates the schema and all components.

**What gets generated per entity:**
- UI schema (forms, detail views, list views, status config, actions)
- TypeScript types
- API client functions
- Validation logic
- Svelte form component
- Svelte detail component
- Svelte list component
- Svelte status badge
- Svelte action buttons

**What gets generated as shared infrastructure:**
- Svelte components for composite types (Money, Address, DateRange, etc.)
- Shared stores for data fetching, mutation, pagination, real-time subscriptions
- Real-time event client (WebSocket → NATS bridge)
- Entity-level event subscription with automatic store invalidation
- Validation utilities
- API client base

---

## 2. File Structure

### Input

```
ontology/*.cue                  — Entity definitions, constraints, state machines, relationships
codegen/apigen.cue              — API service definitions
codegen/uigen.cue               — UI mapping rules (field type → component mapping)
```

### Output — Layer 1 (framework-agnostic)

```
gen/ui/schema/
├── lease.schema.json            — Complete UI schema for Lease entity
├── space.schema.json
├── person.schema.json
├── property.schema.json
├── building.schema.json
├── portfolio.schema.json
├── person_role.schema.json
├── application.schema.json
├── account.schema.json
├── ledger_entry.schema.json
├── journal_entry.schema.json
├── bank_account.schema.json
├── reconciliation.schema.json
└── _enums.schema.json           — All enum definitions with labels
```

### Output — Layer 2 (Svelte + Skeleton + Tailwind)

```
gen/ui/
├── types/
│   ├── lease.types.ts
│   ├── space.types.ts
│   ├── ... (one per entity)
│   ├── common.types.ts
│   └── enums.ts
│
├── api/
│   ├── lease.api.ts
│   ├── space.api.ts
│   ├── ... (one per entity)
│   └── client.ts
│
├── validation/
│   ├── lease.validation.ts
│   ├── space.validation.ts
│   └── ... (one per entity)
│
├── components/
│   ├── entities/
│   │   ├── lease/
│   │   │   ├── LeaseForm.svelte
│   │   │   ├── LeaseDetail.svelte
│   │   │   ├── LeaseList.svelte
│   │   │   ├── LeaseStatusBadge.svelte
│   │   │   └── LeaseActions.svelte
│   │   ├── space/
│   │   │   ├── SpaceForm.svelte
│   │   │   ├── SpaceDetail.svelte
│   │   │   ├── SpaceList.svelte
│   │   │   ├── SpaceStatusBadge.svelte
│   │   │   └── SpaceActions.svelte
│   │   └── ... (one directory per entity)
│   │
│   ├── sections/                    — Form sections for embedded JSON types
│   │   ├── CAMTermsSection.svelte
│   │   ├── PercentageRentSection.svelte
│   │   ├── RentScheduleSection.svelte
│   │   ├── TenantImprovementSection.svelte
│   │   ├── SubsidyTermsSection.svelte
│   │   ├── RenewalOptionSection.svelte
│   │   ├── ExpansionRightSection.svelte
│   │   ├── ContractionRightSection.svelte
│   │   ├── UsageBasedChargeSection.svelte
│   │   ├── RecurringChargeSection.svelte
│   │   ├── LateFeePolicySection.svelte
│   │   └── LateFeeSection.svelte
│   │
│   └── shared/
│       ├── MoneyInput.svelte
│       ├── MoneyDisplay.svelte
│       ├── AddressForm.svelte
│       ├── AddressDisplay.svelte
│       ├── DateRangeInput.svelte
│       ├── DateRangeDisplay.svelte
│       ├── ContactMethodInput.svelte
│       ├── ContactMethodDisplay.svelte
│       ├── EntityRefSelect.svelte
│       ├── EnumSelect.svelte
│       ├── EnumBadge.svelte
│       ├── FormField.svelte
│       ├── FormSection.svelte
│       ├── ArrayEditor.svelte
│       ├── StatusBadge.svelte
│       ├── TransitionButton.svelte
│       └── ConfirmDialog.svelte
│
└── stores/
    ├── entity.ts                    — Generic entity fetch/cache store
    ├── entityList.ts                — Generic list with pagination, filtering, sorting
    ├── entityMutation.ts            — Generic create/update with optimistic updates
    ├── stateMachine.ts              — Valid transitions for current state
    ├── related.ts                   — Fetch related entities via relationship edges
    ├── events.ts                    — WebSocket event client + entity subscriptions
    └── eventRegistry.ts             — Generated: event type → affected entity mappings
```

---

## 3. Layer 1: UI Schema Format

The UI schema is a JSON document per entity that describes everything needed to render forms, detail views, list views, status badges, and actions. It contains zero framework-specific information.

### 3.1 Top-Level Schema Structure

```json
{
  "entity": "lease",
  "display_name": "Lease",
  "display_name_plural": "Leases",
  "primary_display_field": "id",
  "primary_display_template": "{space_number} — {tenant_name}",

  "fields": [ ... ],
  "enums": { ... },
  "form": { ... },
  "detail": { ... },
  "list": { ... },
  "status": { ... },
  "state_machine": { ... },
  "relationships": [ ... ],
  "validation": { ... },
  "api": { ... }
}
```

### 3.2 Field Definitions

Every field from the ontology entity, with UI-relevant metadata:

```json
{
  "fields": [
    {
      "name": "lease_type",
      "type": "enum",
      "enum_ref": "LeaseType",
      "required": true,
      "default": null,
      "immutable": false,
      "label": "Lease Type",
      "help_text": null,
      "controls_visibility": true,
      "show_in_create": true,
      "show_in_update": false,
      "show_in_list": true,
      "show_in_detail": true,
      "sortable": true,
      "filterable": true
    },
    {
      "name": "base_rent",
      "type": "money",
      "money_variant": "non_negative",
      "required": true,
      "default": null,
      "immutable": false,
      "label": "Base Rent",
      "help_text": "Monthly base rent amount",
      "show_in_create": true,
      "show_in_update": true,
      "show_in_list": true,
      "show_in_detail": true,
      "sortable": true,
      "filterable": true,
      "filter_type": "money_range"
    },
    {
      "name": "cam_terms",
      "type": "embedded_object",
      "object_ref": "CAMTerms",
      "required": false,
      "conditionally_required": true,
      "label": "CAM Terms",
      "show_in_create": true,
      "show_in_update": true,
      "show_in_list": false,
      "show_in_detail": true,
      "sortable": false,
      "filterable": false
    },
    {
      "name": "property_id",
      "type": "entity_ref",
      "ref_entity": "property",
      "ref_display": "name",
      "required": true,
      "immutable": true,
      "label": "Property",
      "show_in_create": true,
      "show_in_update": false,
      "show_in_list": true,
      "show_in_detail": true,
      "sortable": true,
      "filterable": true,
      "filter_type": "entity_ref"
    },
    {
      "name": "tenant_role_ids",
      "type": "entity_ref_list",
      "ref_entity": "person_role",
      "ref_filter": { "role_type": "tenant" },
      "ref_display": "{person.first_name} {person.last_name}",
      "required": true,
      "min_items": 1,
      "label": "Tenants",
      "show_in_create": true,
      "show_in_update": true,
      "show_in_list": false,
      "show_in_detail": true,
      "sortable": false,
      "filterable": false
    },
    {
      "name": "term",
      "type": "date_range",
      "required": true,
      "end_required": false,
      "end_conditionally_required": true,
      "label": "Lease Term",
      "show_in_create": true,
      "show_in_update": true,
      "show_in_list": true,
      "show_in_detail": true,
      "list_display_field": "end",
      "list_column_label": "Expiration",
      "sortable": true,
      "filterable": true,
      "filter_type": "date_range"
    }
  ]
}
```

### 3.3 Field Type Enumeration

Every possible field type in the schema:

```
"string"             — plain text input
"text"               — multiline textarea
"int"                — integer number input, with optional min/max
"float"              — decimal number input, with optional min/max/step
"bool"               — toggle/checkbox
"date"               — date picker
"datetime"           — datetime picker
"enum"               — single-select from defined options
"money"              — cents-based money input (variants: "any", "non_negative", "positive")
"address"            — structured address input
"date_range"         — start/end date pair with optional end
"contact_method"     — type/value/primary/verified structure
"entity_ref"         — single reference to another entity (searchable select)
"entity_ref_list"    — multiple references to entities (multi-select)
"embedded_object"    — inline JSON sub-form
"embedded_array"     — list of JSON objects with add/remove
"string_list"        — list of strings (tags, amenities, etc.)
```

### 3.4 Enum Definitions

Separate file with all enums and display labels:

```json
{
  "LeaseType": {
    "values": [
      { "value": "fixed_term", "label": "Fixed Term" },
      { "value": "month_to_month", "label": "Month to Month" },
      { "value": "commercial_nnn", "label": "Triple Net (NNN)" },
      { "value": "commercial_nn", "label": "Double Net (NN)" },
      { "value": "commercial_n", "label": "Single Net (N)" },
      { "value": "commercial_gross", "label": "Gross / Full Service" },
      { "value": "commercial_modified_gross", "label": "Modified Gross" },
      { "value": "affordable", "label": "Affordable Housing" },
      { "value": "section_8", "label": "Section 8" },
      { "value": "student", "label": "Student" },
      { "value": "ground_lease", "label": "Ground Lease" },
      { "value": "short_term", "label": "Short Term" },
      { "value": "membership", "label": "Membership" }
    ],
    "groups": [
      {
        "label": "Residential",
        "values": ["fixed_term", "month_to_month", "affordable", "section_8", "student", "short_term"]
      },
      {
        "label": "Commercial",
        "values": ["commercial_nnn", "commercial_nn", "commercial_n", "commercial_gross", "commercial_modified_gross", "ground_lease"]
      },
      {
        "label": "Other",
        "values": ["membership"]
      }
    ]
  },
  "LeaseStatus": {
    "values": [
      { "value": "draft", "label": "Draft" },
      { "value": "pending_approval", "label": "Pending Approval" },
      { "value": "pending_signature", "label": "Pending Signature" },
      { "value": "active", "label": "Active" },
      { "value": "expired", "label": "Expired" },
      { "value": "month_to_month_holdover", "label": "Month-to-Month" },
      { "value": "renewed", "label": "Renewed" },
      { "value": "terminated", "label": "Terminated" },
      { "value": "eviction", "label": "Eviction" }
    ]
  }
}
```

Enum label generation rules (deterministic string transforms):
- Split on underscore, capitalize each word
- Known abbreviations stay uppercase: NNN, NN, N, CAM, ACH, CPI, NSF, HUD, LIHTC, AMI
- Known phrases get proper casing: "Section 8", "Month to Month"

Enum grouping rules (derived from ontology knowledge):
- Lease types: group by residential vs commercial
- Space types: group by residential vs commercial vs utility
- If no natural grouping exists: no groups (flat list)

### 3.5 Form Schema

Describes field grouping, ordering, conditional visibility, and section structure:

```json
{
  "form": {
    "sections": [
      {
        "id": "identity",
        "title": "Lease Details",
        "collapsible": false,
        "fields": ["lease_type", "liability_type", "property_id", "tenant_role_ids", "guarantor_role_ids"]
      },
      {
        "id": "term",
        "title": "Lease Term",
        "collapsible": false,
        "fields": ["term", "lease_commencement_date", "rent_commencement_date"]
      },
      {
        "id": "financial",
        "title": "Financial Terms",
        "collapsible": false,
        "fields": ["base_rent", "security_deposit"]
      },
      {
        "id": "cam",
        "title": "Common Area Maintenance",
        "collapsible": true,
        "initially_collapsed": false,
        "visible_when": {
          "field": "lease_type",
          "operator": "in",
          "values": ["commercial_nnn", "commercial_nn", "commercial_n", "commercial_gross", "commercial_modified_gross"]
        },
        "required_when": {
          "field": "lease_type",
          "operator": "in",
          "values": ["commercial_nnn", "commercial_nn", "commercial_n"]
        },
        "embedded_object": "CAMTerms"
      },
      {
        "id": "percentage_rent",
        "title": "Percentage Rent",
        "collapsible": true,
        "initially_collapsed": true,
        "visible_when": {
          "field": "lease_type",
          "operator": "in",
          "values": ["commercial_nnn", "commercial_nn", "commercial_n", "commercial_gross", "commercial_modified_gross"]
        },
        "embedded_object": "PercentageRent"
      },
      {
        "id": "subsidy",
        "title": "Subsidy Terms",
        "collapsible": true,
        "initially_collapsed": false,
        "visible_when": {
          "field": "lease_type",
          "operator": "in",
          "values": ["section_8", "affordable"]
        },
        "required_when": {
          "field": "lease_type",
          "operator": "eq",
          "value": "section_8"
        },
        "embedded_object": "SubsidyTerms"
      },
      {
        "id": "short_term",
        "title": "Short-Term Rental",
        "collapsible": true,
        "initially_collapsed": false,
        "visible_when": {
          "field": "lease_type",
          "operator": "eq",
          "value": "short_term"
        },
        "fields": ["check_in_time", "check_out_time", "cleaning_fee", "platform_booking_id"]
      },
      {
        "id": "membership",
        "title": "Membership",
        "collapsible": true,
        "initially_collapsed": false,
        "visible_when": {
          "field": "lease_type",
          "operator": "eq",
          "value": "membership"
        },
        "fields": ["membership_tier"]
      },
      {
        "id": "rent_schedule",
        "title": "Rent Schedule",
        "collapsible": true,
        "initially_collapsed": true,
        "embedded_array": "RentScheduleEntry"
      },
      {
        "id": "recurring_charges",
        "title": "Recurring Charges",
        "collapsible": true,
        "initially_collapsed": true,
        "embedded_array": "RecurringCharge"
      },
      {
        "id": "usage_charges",
        "title": "Usage-Based Charges",
        "collapsible": true,
        "initially_collapsed": true,
        "visible_when": {
          "field": "lease_type",
          "operator": "in",
          "values": ["commercial_nnn", "commercial_nn", "commercial_n", "commercial_gross", "commercial_modified_gross"]
        },
        "embedded_array": "UsageBasedCharge"
      },
      {
        "id": "renewal_options",
        "title": "Renewal Options",
        "collapsible": true,
        "initially_collapsed": true,
        "embedded_array": "RenewalOption"
      },
      {
        "id": "expansion",
        "title": "Expansion Rights",
        "collapsible": true,
        "initially_collapsed": true,
        "visible_when": {
          "field": "lease_type",
          "operator": "in",
          "values": ["commercial_nnn", "commercial_nn", "commercial_n", "commercial_gross", "commercial_modified_gross"]
        },
        "embedded_array": "ExpansionRight"
      },
      {
        "id": "contraction",
        "title": "Contraction Rights",
        "collapsible": true,
        "initially_collapsed": true,
        "visible_when": {
          "field": "lease_type",
          "operator": "in",
          "values": ["commercial_nnn", "commercial_nn", "commercial_n", "commercial_gross", "commercial_modified_gross"]
        },
        "embedded_array": "ContractionRight"
      },
      {
        "id": "tenant_improvement",
        "title": "Tenant Improvement Allowance",
        "collapsible": true,
        "initially_collapsed": true,
        "visible_when": {
          "field": "lease_type",
          "operator": "in",
          "values": ["commercial_nnn", "commercial_nn", "commercial_n", "commercial_gross", "commercial_modified_gross"]
        },
        "embedded_object": "TenantImprovement"
      },
      {
        "id": "sublease",
        "title": "Sublease Details",
        "collapsible": true,
        "initially_collapsed": true,
        "visible_when": {
          "field": "is_sublease",
          "operator": "eq",
          "value": true
        },
        "fields": ["parent_lease_id", "sublease_billing"]
      },
      {
        "id": "late_fee",
        "title": "Late Fee Policy",
        "collapsible": true,
        "initially_collapsed": true,
        "embedded_object": "LateFeePolicy"
      },
      {
        "id": "signing",
        "title": "Signing",
        "collapsible": true,
        "initially_collapsed": true,
        "fields": ["signing_method", "document_id"]
      }
    ],
    "field_order_rule": "required_first"
  }
}
```

Section ordering rules (deterministic):
1. Identity and type selectors (they control conditional visibility — must be first)
2. Dates and term
3. Core financial (base rent, deposit)
4. Type-specific conditional sections (ordered by frequency: CAM, subsidy, short-term, membership)
5. Optional arrays (rent schedule, recurring charges, usage charges)
6. Optional commercial structures (renewal, expansion, contraction, TI)
7. Flags and secondary fields (sublease, late fees, signing)

Within each section: required fields before optional fields.

### 3.6 Visibility Rule Format

```json
{
  "visible_when": {
    "field": "lease_type",
    "operator": "in",
    "values": ["commercial_nnn", "commercial_nn"]
  }
}
```

Supported operators:
```
"eq"      — field equals value
"neq"     — field does not equal value
"in"      — field value is in list
"not_in"  — field value is not in list
"truthy"  — field is non-null and non-empty
"falsy"   — field is null or empty
```

Derivation from ontology: every CONSTRAINT block with "If [field] is [value], then [other_field] is required" produces a visibility rule where the other_field's section becomes visible when field matches value.

### 3.7 Detail View Schema

```json
{
  "detail": {
    "header": {
      "title_template": "Lease: {space_number}",
      "status_field": "status",
      "actions": true
    },
    "sections": [
      {
        "id": "overview",
        "title": "Overview",
        "layout": "grid_2col",
        "fields": ["lease_type", "liability_type", "status", "property_id", "term"]
      },
      {
        "id": "financial",
        "title": "Financial",
        "layout": "grid_2col",
        "fields": ["base_rent", "security_deposit"]
      },
      {
        "id": "cam",
        "title": "CAM Terms",
        "visible_when": { "field": "cam_terms", "operator": "truthy" },
        "embedded_object": "CAMTerms",
        "display_mode": "readonly"
      }
    ],
    "related_sections": [
      {
        "title": "Spaces",
        "relationship": "lease_spaces",
        "entity": "lease_space",
        "display": "list",
        "include_fields": ["space.space_number", "space.space_type", "relationship", "effective"]
      },
      {
        "title": "Tenants",
        "relationship": "tenant_roles",
        "entity": "person_role",
        "display": "list",
        "include_fields": ["person.first_name", "person.last_name", "attributes.standing", "attributes.current_balance"]
      },
      {
        "title": "Ledger",
        "relationship": "ledger_entries",
        "entity": "ledger_entry",
        "display": "table",
        "include_fields": ["effective_date", "entry_type", "description", "amount", "reconciled"]
      }
    ]
  }
}
```

### 3.8 List View Schema

```json
{
  "list": {
    "default_columns": [
      { "field": "status", "width": "100px" },
      { "field": "lease_type", "width": "140px" },
      { "field": "property_id", "display_as": "property.name", "width": "180px" },
      { "field": "base_rent", "width": "120px", "align": "right" },
      { "field": "term.end", "label": "Expiration", "width": "120px" },
      { "field": "updated_at", "label": "Last Updated", "width": "140px" }
    ],
    "max_default_columns": 7,
    "filters": [
      { "field": "status", "type": "multi_enum", "enum_ref": "LeaseStatus" },
      { "field": "lease_type", "type": "multi_enum", "enum_ref": "LeaseType" },
      { "field": "property_id", "type": "entity_ref", "ref_entity": "property" },
      { "field": "term.end", "type": "date_range", "label": "Expiration Date" },
      { "field": "base_rent", "type": "money_range", "label": "Base Rent" }
    ],
    "default_sort": { "field": "updated_at", "direction": "desc" },
    "row_click_action": "navigate_to_detail",
    "bulk_actions": false
  }
}
```

Column selection rules (deterministic):
- Priority 1 (always visible): primary display field, status, most important reference
- Priority 2 (visible by default): money fields, key date fields, type enums
- Priority 3 (available via column picker): everything else
- Max 7 default columns
- Audit fields: hidden by default, available in column picker
- JSON embedded objects: never shown as columns

### 3.9 Status Configuration

```json
{
  "status": {
    "field": "status",
    "color_mapping": {
      "draft": "surface",
      "pending_approval": "secondary",
      "pending_signature": "secondary",
      "active": "success",
      "expired": "warning",
      "month_to_month_holdover": "warning",
      "renewed": "surface",
      "terminated": "surface",
      "eviction": "error"
    }
  }
}
```

Color derivation rules (from state machine structure):
```
initial state           → "surface"    (gray/neutral)
forward progression     → "secondary"  (blue/info)
active/operational      → "success"    (green)
warning/attention       → "warning"    (yellow/amber)
negative/problem        → "error"      (red)
terminal (completed)    → "surface"    (gray/neutral)
```

Classification:
- Initial: first state in state machine (has `[*] -->`)
- Terminal: states with no outgoing transitions
- Active: states named "active", "occupied", "posted", "approved", "balanced"
- Negative: states named "eviction", "denied", "down", "frozen", "voided"
- Warning: states indicating action needed but not negative ("expired", "notice_given", "make_ready", "unbalanced")
- Forward: states between initial and active

### 3.10 State Machine Actions

```json
{
  "state_machine": {
    "transitions": {
      "draft": [
        {
          "target": "pending_approval",
          "label": "Submit for Approval",
          "variant": "primary",
          "confirm": false,
          "api_endpoint": "POST /v1/leases/{id}/submit"
        },
        {
          "target": "pending_signature",
          "label": "Send for Signature",
          "variant": "primary",
          "confirm": false,
          "api_endpoint": "POST /v1/leases/{id}/sign"
        },
        {
          "target": "terminated",
          "label": "Cancel",
          "variant": "danger",
          "confirm": true,
          "confirm_message": "Are you sure you want to cancel this lease? This cannot be undone.",
          "api_endpoint": "POST /v1/leases/{id}/terminate"
        }
      ],
      "pending_approval": [
        {
          "target": "pending_signature",
          "label": "Approve",
          "variant": "primary",
          "confirm": false,
          "api_endpoint": "POST /v1/leases/{id}/approve"
        },
        {
          "target": "draft",
          "label": "Return to Draft",
          "variant": "secondary",
          "confirm": false,
          "api_endpoint": "POST /v1/leases/{id}/reject"
        },
        {
          "target": "terminated",
          "label": "Reject",
          "variant": "danger",
          "confirm": true,
          "confirm_message": "Are you sure you want to reject this lease?",
          "api_endpoint": "POST /v1/leases/{id}/terminate"
        }
      ],
      "pending_signature": [
        {
          "target": "active",
          "label": "Activate",
          "variant": "primary",
          "confirm": false,
          "requires_fields": ["move_in_date", "signed_at"],
          "api_endpoint": "POST /v1/leases/{id}/activate"
        },
        {
          "target": "draft",
          "label": "Return to Draft",
          "variant": "secondary",
          "confirm": false,
          "api_endpoint": "POST /v1/leases/{id}/reject"
        },
        {
          "target": "terminated",
          "label": "Cancel",
          "variant": "danger",
          "confirm": true,
          "confirm_message": "Are you sure you want to cancel this lease?",
          "api_endpoint": "POST /v1/leases/{id}/terminate"
        }
      ],
      "active": [
        {
          "target": "expired",
          "label": "Mark Expired",
          "variant": "secondary",
          "confirm": true,
          "api_endpoint": "POST /v1/leases/{id}/expire"
        },
        {
          "target": "month_to_month_holdover",
          "label": "Convert to Month-to-Month",
          "variant": "secondary",
          "confirm": true,
          "api_endpoint": "POST /v1/leases/{id}/holdover"
        },
        {
          "target": "terminated",
          "label": "Terminate",
          "variant": "danger",
          "confirm": true,
          "confirm_message": "Are you sure you want to terminate this lease? This will begin the move-out process.",
          "api_endpoint": "POST /v1/leases/{id}/terminate"
        },
        {
          "target": "eviction",
          "label": "Initiate Eviction",
          "variant": "danger",
          "confirm": true,
          "confirm_message": "Are you sure you want to initiate eviction proceedings?",
          "api_endpoint": "POST /v1/leases/{id}/evict"
        }
      ],
      "terminated": [],
      "renewed": []
    }
  }
}
```

Action button derivation rules:
- `variant: "primary"`: forward-progress transitions (toward active/completed)
- `variant: "secondary"`: backward or lateral transitions (return to draft, convert)
- `variant: "danger"`: transitions to terminal or negative states
- `confirm: true`: all danger variants, all irreversible transitions
- `confirm_message`: generated from target state and entity name
- `requires_fields`: derived from ontology constraints on target state ("active requires move_in_date and signed_at")
- `api_endpoint`: derived from apigen.cue service definitions

### 3.11 Relationship Definitions

```json
{
  "relationships": [
    {
      "name": "tenant_roles",
      "target_entity": "person_role",
      "cardinality": "many_to_many",
      "api_endpoint": "GET /v1/leases/{id}/tenant-roles",
      "display_in_detail": true,
      "display_mode": "list"
    },
    {
      "name": "spaces",
      "target_entity": "lease_space",
      "cardinality": "one_to_many",
      "api_endpoint": "GET /v1/leases/{id}/spaces",
      "display_in_detail": true,
      "display_mode": "list"
    },
    {
      "name": "ledger_entries",
      "target_entity": "ledger_entry",
      "cardinality": "one_to_many",
      "api_endpoint": "GET /v1/leases/{id}/ledger",
      "display_in_detail": true,
      "display_mode": "table"
    }
  ]
}
```

### 3.12 Validation Rules

```json
{
  "validation": {
    "field_rules": [
      { "field": "property_id", "rule": "required" },
      { "field": "tenant_role_ids", "rule": "min_length", "value": 1 },
      { "field": "lease_type", "rule": "required" },
      { "field": "base_rent.amount_cents", "rule": "min", "value": 0 },
      { "field": "security_deposit.amount_cents", "rule": "min", "value": 0 }
    ],
    "cross_field_rules": [
      {
        "id": "fixed_term_requires_end",
        "description": "Fixed-term and student leases require an end date",
        "condition": { "field": "lease_type", "operator": "in", "values": ["fixed_term", "student"] },
        "then": { "field": "term.end", "rule": "required" },
        "message": "End date is required for fixed-term and student leases"
      },
      {
        "id": "nnn_requires_cam",
        "description": "NNN leases require CAM terms",
        "condition": { "field": "lease_type", "operator": "eq", "value": "commercial_nnn" },
        "then": { "field": "cam_terms", "rule": "required" },
        "message": "CAM terms are required for NNN leases"
      },
      {
        "id": "nnn_cam_requires_tax",
        "description": "NNN CAM must include property tax",
        "condition": { "field": "lease_type", "operator": "eq", "value": "commercial_nnn" },
        "then": { "field": "cam_terms.includes_property_tax", "rule": "eq", "value": true },
        "message": "NNN leases must include property tax in CAM"
      },
      {
        "id": "nnn_cam_requires_insurance",
        "description": "NNN CAM must include insurance",
        "condition": { "field": "lease_type", "operator": "eq", "value": "commercial_nnn" },
        "then": { "field": "cam_terms.includes_insurance", "rule": "eq", "value": true },
        "message": "NNN leases must include insurance in CAM"
      },
      {
        "id": "nnn_cam_requires_utilities",
        "description": "NNN CAM must include utilities",
        "condition": { "field": "lease_type", "operator": "eq", "value": "commercial_nnn" },
        "then": { "field": "cam_terms.includes_utilities", "rule": "eq", "value": true },
        "message": "NNN leases must include utilities in CAM"
      },
      {
        "id": "section8_requires_subsidy",
        "description": "Section 8 leases require subsidy terms",
        "condition": { "field": "lease_type", "operator": "eq", "value": "section_8" },
        "then": { "field": "subsidy", "rule": "required" },
        "message": "Subsidy terms are required for Section 8 leases"
      },
      {
        "id": "sublease_requires_parent",
        "description": "Subleases require parent lease",
        "condition": { "field": "is_sublease", "operator": "eq", "value": true },
        "then": { "field": "parent_lease_id", "rule": "required" },
        "message": "Parent lease is required for subleases"
      },
      {
        "id": "rent_commencement_after_lease",
        "description": "Rent commencement must be on or after lease commencement",
        "condition": { "field": "rent_commencement_date", "operator": "truthy" },
        "then": { "field": "rent_commencement_date", "rule": "gte_field", "value": "lease_commencement_date" },
        "message": "Rent commencement cannot be before lease commencement"
      }
    ]
  }
}
```

### 3.13 API Endpoint Mapping

```json
{
  "api": {
    "base_path": "/v1/leases",
    "operations": {
      "create": { "method": "POST", "path": "/v1/leases" },
      "get": { "method": "GET", "path": "/v1/leases/{id}" },
      "list": { "method": "GET", "path": "/v1/leases" },
      "update": { "method": "PATCH", "path": "/v1/leases/{id}" },
      "search": { "method": "POST", "path": "/v1/leases/search" }
    },
    "transitions": {
      "submit": { "method": "POST", "path": "/v1/leases/{id}/submit" },
      "approve": { "method": "POST", "path": "/v1/leases/{id}/approve" },
      "sign": { "method": "POST", "path": "/v1/leases/{id}/sign" },
      "activate": { "method": "POST", "path": "/v1/leases/{id}/activate" },
      "notice": { "method": "POST", "path": "/v1/leases/{id}/notice" },
      "renew": { "method": "POST", "path": "/v1/leases/{id}/renew" },
      "terminate": { "method": "POST", "path": "/v1/leases/{id}/terminate" },
      "evict": { "method": "POST", "path": "/v1/leases/{id}/evict" }
    },
    "related": {
      "tenant_roles": { "method": "GET", "path": "/v1/leases/{id}/tenant-roles" },
      "guarantor_roles": { "method": "GET", "path": "/v1/leases/{id}/guarantor-roles" },
      "spaces": { "method": "GET", "path": "/v1/leases/{id}/spaces" },
      "ledger": { "method": "GET", "path": "/v1/leases/{id}/ledger" }
    }
  }
}
```

---

## 4. Layer 1: Schema Generation Rules

These are the deterministic rules that `cmd/uigen` applies when reading ontology CUE files and producing UI schemas.

### 4.1 Field Type Mapping

```
CUE type                    → Schema field type
──────────────────────────────────────────────────
string (short)              → "string"
string (long/description)   → "text"
int                         → "int"
float                       → "float"
bool                        → "bool"
time                        → "datetime"
time (date-only context)    → "date"
enum ("|" separated)        → "enum"
#Money                      → "money" with money_variant: "any"
#NonNegativeMoney           → "money" with money_variant: "non_negative"
#PositiveMoney              → "money" with money_variant: "positive"
#Address                    → "address"
#DateRange                  → "date_range"
#ContactMethod              → "contact_method"
list of strings             → "string_list"
list of #X                  → "embedded_array" with object_ref
single #X                   → "embedded_object" with object_ref
entity reference (string)   → "entity_ref" with ref_entity
list of entity refs         → "entity_ref_list" with ref_entity
```

How to distinguish "string" from "text": if field name contains "description", "memo", "notes", "reason", or "guidance" → "text". Everything else → "string".

How to distinguish entity_ref from plain string: if field name ends in "_id" and a relationship exists in relationships.cue referencing that entity → "entity_ref". If field name ends in "_ids" → "entity_ref_list".

### 4.2 Required / Optional Derivation

```
CUE "required string"      → required: true
CUE "optional string"      → required: false
CUE field with default      → required: false (has default)
CUE field in CONSTRAINT     → conditionally_required: true + visible_when rule
```

### 4.3 Show/Hide in Views

```
Field category          → create  update  list   detail
──────────────────────────────────────────────────────────
id                      → no      no      no     yes
audit fields            → no      no      no     yes (collapsible)
status                  → no      no      yes    yes (as badge)
immutable refs          → yes     no      yes    yes
mutable fields          → yes     yes     maybe  yes
JSON embedded objects   → yes     yes     no     yes (as section)
JSON embedded arrays    → yes     yes     no     yes (as sub-list)
computed fields         → no      no      maybe  yes
```

List view column selection (deterministic priority):
1. Status field (if exists)
2. Primary type enum (lease_type, space_type)
3. Most important entity_ref (property, person)
4. Money fields
5. Key date fields
6. Max 7 columns default; remaining available via column picker

### 4.4 Constraint → Visibility Rule Derivation

For every CONSTRAINT in the ontology of the form:
```
"If [field_A] is [value], then [field_B] is required"
```

Generate:
```json
{
  "visible_when": { "field": "field_A", "operator": "eq/in", "value/values": "..." },
  "required_when": { "field": "field_A", "operator": "eq/in", "value/values": "..." }
}
```

Applied to the form section containing field_B. If multiple fields share the same visibility condition, they're grouped into one conditional section.

### 4.5 State Machine → Color + Action Derivation

Read state_machines.cue. For each state machine:

1. Build directed graph of states and transitions
2. Classify each state by position (initial, forward, active, warning, negative, terminal)
3. Map classification to Skeleton color token
4. For each state, list valid transitions as action buttons
5. Classify each transition button variant:
   - Target is "closer to active/completed" → "primary"
   - Target is "backward" → "secondary"
   - Target is "terminal or negative" → "danger"
6. Set `confirm: true` for all danger variants
7. Cross-reference ontology constraints to find `requires_fields` for each transition target state

### 4.6 Relationship → Detail Section Derivation

Read relationships.cue. For each relationship where this entity is the "from" side:
- Generate a related_section in the detail schema
- Set display_mode: "table" for O2M with many expected items (ledger_entries, work_orders)
- Set display_mode: "list" for O2M/M2M with fewer expected items (tenant_roles, spaces)
- Include fields: select 3-5 most relevant fields from the target entity (prioritize status, type, amount, date fields)

### 4.7 Embedded Object → Form Section Derivation

For every JSON embedded type (CAMTerms, PercentageRent, SubsidyTerms, etc.):
1. Read the embedded type definition from the ontology
2. Generate a form section schema with all the embedded type's fields
3. Apply the same field type mapping rules recursively
4. Generate a `sections/` Svelte component from the section schema

---

## 5. Layer 2: Svelte + Skeleton + Tailwind Renderer

### 5.1 Component Mapping

The renderer reads the UI schema and maps component types to Svelte + Skeleton implementations:

```
Schema field type       → Svelte component
──────────────────────────────────────────────────
"string"                → <input class="input" type="text" />
"text"                  → <textarea class="textarea" />
"int"                   → <input class="input" type="number" step="1" />
"float"                 → <input class="input" type="number" step="any" />
"bool"                  → <SlideToggle /> from Skeleton
"date"                  → <input class="input" type="date" />
"datetime"              → <input class="input" type="datetime-local" />
"enum"                  → <select class="select"> with options from enum_ref
"money"                 → <MoneyInput /> (custom shared component)
"address"               → <AddressForm /> (custom shared component)
"date_range"            → <DateRangeInput /> (custom shared component)
"contact_method"        → <ContactMethodInput /> (custom shared component)
"entity_ref"            → <Autocomplete /> from Skeleton, hitting List API
"entity_ref_list"       → <Autocomplete /> from Skeleton, multiple mode
"embedded_object"       → generated <{Type}Section /> from sections/
"embedded_array"        → <ArrayEditor /> with generated item component
"string_list"           → <InputChip /> from Skeleton
```

### 5.2 Form Component Template

```svelte
<!-- gen/ui/components/entities/lease/LeaseForm.svelte -->
<!-- GENERATED FROM ONTOLOGY. DO NOT HAND-EDIT. -->

<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import { SlideToggle } from '@skeletonlabs/skeleton';
  import FormField from '../../shared/FormField.svelte';
  import FormSection from '../../shared/FormSection.svelte';
  import MoneyInput from '../../shared/MoneyInput.svelte';
  import DateRangeInput from '../../shared/DateRangeInput.svelte';
  import EnumSelect from '../../shared/EnumSelect.svelte';
  import EntityRefSelect from '../../shared/EntityRefSelect.svelte';
  import CAMTermsSection from '../../sections/CAMTermsSection.svelte';
  import SubsidyTermsSection from '../../sections/SubsidyTermsSection.svelte';
  import { validateLease } from '../../../validation/lease.validation';
  import { leaseApi } from '../../../api/lease.api';
  import { LEASE_TYPE_OPTIONS, LIABILITY_TYPE_OPTIONS } from '../../../types/enums';
  import type { LeaseCreateInput } from '../../../types/lease.types';

  // Generated from schema: form.sections[*].visible_when
  import { LEASE_FORM_VISIBILITY } from './lease.config';

  export let initialValues: Partial<LeaseCreateInput> = {};
  export let mode: 'create' | 'edit' = 'create';

  const dispatch = createEventDispatcher();

  let values: Partial<LeaseCreateInput> = { ...initialValues };
  let errors: Record<string, string> = {};

  function handleChange(field: string, value: any) {
    values = { ...values, [field]: value };
    if (errors[field]) {
      const { [field]: _, ...rest } = errors;
      errors = rest;
    }
  }

  function isVisible(sectionId: string): boolean {
    const rule = LEASE_FORM_VISIBILITY[sectionId];
    if (!rule) return true;
    return rule(values);
  }

  async function handleSubmit() {
    const validationErrors = validateLease(values as LeaseCreateInput);
    if (Object.keys(validationErrors).length > 0) {
      errors = validationErrors;
      return;
    }
    const result = mode === 'create'
      ? await leaseApi.create(values as LeaseCreateInput)
      : await leaseApi.update(values as LeaseUpdateInput);
    dispatch('submit', result);
  }
</script>

<form on:submit|preventDefault={handleSubmit} class="space-y-6">
  
  <!-- Section: Lease Details (always visible) -->
  <FormSection title="Lease Details">
    <FormField label="Lease Type" required error={errors['lease_type']}>
      <EnumSelect
        options={LEASE_TYPE_OPTIONS}
        value={values.lease_type}
        on:change={(e) => handleChange('lease_type', e.detail)}
      />
    </FormField>

    <FormField label="Liability Type" error={errors['liability_type']}>
      <EnumSelect
        options={LIABILITY_TYPE_OPTIONS}
        value={values.liability_type ?? 'joint_and_several'}
        on:change={(e) => handleChange('liability_type', e.detail)}
      />
    </FormField>

    <FormField label="Property" required error={errors['property_id']}>
      <EntityRefSelect
        entityType="property"
        displayField="name"
        value={values.property_id}
        on:change={(e) => handleChange('property_id', e.detail)}
      />
    </FormField>

    <FormField label="Tenants" required error={errors['tenant_role_ids']}>
      <EntityRefSelect
        entityType="person_role"
        filter={{ role_type: 'tenant' }}
        displayTemplate="{person.first_name} {person.last_name}"
        multiple
        value={values.tenant_role_ids}
        on:change={(e) => handleChange('tenant_role_ids', e.detail)}
      />
    </FormField>
  </FormSection>

  <!-- Section: Lease Term -->
  <FormSection title="Lease Term">
    <FormField label="Term" required error={errors['term']}>
      <DateRangeInput
        value={values.term}
        requireEnd={['fixed_term', 'student'].includes(values.lease_type ?? '')}
        on:change={(e) => handleChange('term', e.detail)}
      />
    </FormField>
  </FormSection>

  <!-- Section: Financial Terms -->
  <FormSection title="Financial Terms">
    <FormField label="Base Rent" required error={errors['base_rent']}>
      <MoneyInput
        value={values.base_rent}
        min={0}
        on:change={(e) => handleChange('base_rent', e.detail)}
      />
    </FormField>

    <FormField label="Security Deposit" required error={errors['security_deposit']}>
      <MoneyInput
        value={values.security_deposit}
        min={0}
        on:change={(e) => handleChange('security_deposit', e.detail)}
      />
    </FormField>
  </FormSection>

  <!-- Section: CAM Terms (conditional) -->
  {#if isVisible('cam')}
    <FormSection
      title="Common Area Maintenance"
      collapsible
      required={['commercial_nnn', 'commercial_nn', 'commercial_n'].includes(values.lease_type ?? '')}
    >
      <CAMTermsSection
        value={values.cam_terms}
        leaseType={values.lease_type}
        errors={errors}
        on:change={(e) => handleChange('cam_terms', e.detail)}
      />
    </FormSection>
  {/if}

  <!-- Section: Subsidy Terms (conditional) -->
  {#if isVisible('subsidy')}
    <FormSection title="Subsidy Terms" collapsible required>
      <SubsidyTermsSection
        value={values.subsidy}
        errors={errors}
        on:change={(e) => handleChange('subsidy', e.detail)}
      />
    </FormSection>
  {/if}

  <!-- ... remaining conditional sections follow same pattern -->

  <div class="flex justify-end gap-2 pt-4">
    <button type="button" class="btn variant-soft" on:click={() => dispatch('cancel')}>
      Cancel
    </button>
    <button type="submit" class="btn variant-filled-primary">
      {mode === 'create' ? 'Create Lease' : 'Save Changes'}
    </button>
  </div>
</form>
```

### 5.3 Status Badge Component Template

```svelte
<!-- gen/ui/components/entities/lease/LeaseStatusBadge.svelte -->
<!-- GENERATED FROM ONTOLOGY. DO NOT HAND-EDIT. -->

<script lang="ts">
  import type { LeaseStatus } from '../../../types/lease.types';
  import { LEASE_STATUS_LABELS } from '../../../types/enums';

  export let status: LeaseStatus;

  // Generated from state machine position analysis
  const colorMap: Record<LeaseStatus, string> = {
    draft: 'variant-soft',
    pending_approval: 'variant-soft-secondary',
    pending_signature: 'variant-soft-secondary',
    active: 'variant-soft-success',
    expired: 'variant-soft-warning',
    month_to_month_holdover: 'variant-soft-warning',
    renewed: 'variant-soft',
    terminated: 'variant-soft',
    eviction: 'variant-soft-error',
  };
</script>

<span class="badge {colorMap[status]}">
  {LEASE_STATUS_LABELS[status]}
</span>
```

### 5.4 Actions Component Template

```svelte
<!-- gen/ui/components/entities/lease/LeaseActions.svelte -->
<!-- GENERATED FROM ONTOLOGY. DO NOT HAND-EDIT. -->

<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import TransitionButton from '../../shared/TransitionButton.svelte';
  import type { LeaseStatus } from '../../../types/lease.types';

  export let entityId: string;
  export let currentStatus: LeaseStatus;

  const dispatch = createEventDispatcher();

  // Generated from state_machines.cue: LeaseTransitions
  const transitions: Record<LeaseStatus, Array<{
    target: LeaseStatus;
    label: string;
    variant: 'primary' | 'secondary' | 'danger';
    confirm: boolean;
    confirmMessage?: string;
    endpoint: string;
    requiresFields?: string[];
  }>> = {
    draft: [
      { target: 'pending_approval', label: 'Submit for Approval', variant: 'primary', confirm: false, endpoint: `/v1/leases/${entityId}/submit` },
      { target: 'pending_signature', label: 'Send for Signature', variant: 'primary', confirm: false, endpoint: `/v1/leases/${entityId}/sign` },
      { target: 'terminated', label: 'Cancel', variant: 'danger', confirm: true, confirmMessage: 'Are you sure you want to cancel this lease? This cannot be undone.', endpoint: `/v1/leases/${entityId}/terminate` },
    ],
    pending_approval: [
      { target: 'pending_signature', label: 'Approve', variant: 'primary', confirm: false, endpoint: `/v1/leases/${entityId}/approve` },
      { target: 'draft', label: 'Return to Draft', variant: 'secondary', confirm: false, endpoint: `/v1/leases/${entityId}/reject` },
      { target: 'terminated', label: 'Reject', variant: 'danger', confirm: true, confirmMessage: 'Are you sure you want to reject this lease?', endpoint: `/v1/leases/${entityId}/terminate` },
    ],
    pending_signature: [
      { target: 'active', label: 'Activate', variant: 'primary', confirm: false, endpoint: `/v1/leases/${entityId}/activate`, requiresFields: ['move_in_date', 'signed_at'] },
      { target: 'draft', label: 'Return to Draft', variant: 'secondary', confirm: false, endpoint: `/v1/leases/${entityId}/reject` },
      { target: 'terminated', label: 'Cancel', variant: 'danger', confirm: true, confirmMessage: 'Are you sure you want to cancel this lease?', endpoint: `/v1/leases/${entityId}/terminate` },
    ],
    active: [
      { target: 'month_to_month_holdover', label: 'Convert to Month-to-Month', variant: 'secondary', confirm: true, endpoint: `/v1/leases/${entityId}/holdover` },
      { target: 'terminated', label: 'Terminate', variant: 'danger', confirm: true, confirmMessage: 'Are you sure you want to terminate this lease? This will begin the move-out process.', endpoint: `/v1/leases/${entityId}/terminate` },
      { target: 'eviction', label: 'Initiate Eviction', variant: 'danger', confirm: true, confirmMessage: 'Are you sure you want to initiate eviction proceedings?', endpoint: `/v1/leases/${entityId}/evict` },
    ],
    expired: [
      { target: 'month_to_month_holdover', label: 'Convert to Month-to-Month', variant: 'secondary', confirm: true, endpoint: `/v1/leases/${entityId}/holdover` },
      { target: 'renewed', label: 'Renew', variant: 'primary', confirm: false, endpoint: `/v1/leases/${entityId}/renew` },
      { target: 'terminated', label: 'Terminate', variant: 'danger', confirm: true, confirmMessage: 'Terminate expired lease?', endpoint: `/v1/leases/${entityId}/terminate` },
    ],
    month_to_month_holdover: [
      { target: 'renewed', label: 'Renew', variant: 'primary', confirm: false, endpoint: `/v1/leases/${entityId}/renew` },
      { target: 'terminated', label: 'Terminate', variant: 'danger', confirm: true, confirmMessage: 'Terminate month-to-month lease?', endpoint: `/v1/leases/${entityId}/terminate` },
      { target: 'eviction', label: 'Initiate Eviction', variant: 'danger', confirm: true, confirmMessage: 'Initiate eviction on month-to-month tenant?', endpoint: `/v1/leases/${entityId}/evict` },
    ],
    terminated: [],
    renewed: [],
    eviction: [
      { target: 'terminated', label: 'Complete Eviction', variant: 'danger', confirm: true, confirmMessage: 'Mark eviction as complete?', endpoint: `/v1/leases/${entityId}/terminate` },
    ],
  };

  $: availableTransitions = transitions[currentStatus] ?? [];
</script>

{#if availableTransitions.length > 0}
  <div class="flex gap-2">
    {#each availableTransitions as transition}
      <TransitionButton
        label={transition.label}
        variant={transition.variant}
        confirm={transition.confirm}
        confirmMessage={transition.confirmMessage}
        endpoint={transition.endpoint}
        requiresFields={transition.requiresFields}
        on:complete={(e) => dispatch('transition', e.detail)}
      />
    {/each}
  </div>
{/if}
```

### 5.5 List Component Template

```svelte
<!-- gen/ui/components/entities/lease/LeaseList.svelte -->
<!-- GENERATED FROM ONTOLOGY. DO NOT HAND-EDIT. -->

<script lang="ts">
  import { Paginator, Table } from '@skeletonlabs/skeleton';
  import LeaseStatusBadge from './LeaseStatusBadge.svelte';
  import MoneyDisplay from '../../shared/MoneyDisplay.svelte';
  import EnumBadge from '../../shared/EnumBadge.svelte';
  import { entityListStore } from '../../../stores/entityList';
  import { LEASE_TYPE_LABELS, LEASE_STATUS_LABELS } from '../../../types/enums';
  import type { Lease } from '../../../types/lease.types';

  const store = entityListStore<Lease>('lease', {
    defaultSort: { field: 'updated_at', direction: 'desc' },
  });

  // Generated from list schema
  const columns = [
    { field: 'status', label: 'Status', width: '100px', component: 'status_badge' },
    { field: 'lease_type', label: 'Type', width: '140px', component: 'enum_badge' },
    { field: 'property_id', label: 'Property', width: '180px', display: 'property.name' },
    { field: 'base_rent', label: 'Base Rent', width: '120px', align: 'right', component: 'money' },
    { field: 'term.end', label: 'Expiration', width: '120px', component: 'date' },
    { field: 'updated_at', label: 'Last Updated', width: '140px', component: 'datetime' },
  ];

  // Generated from list schema filters
  const filterConfig = [
    { field: 'status', type: 'multi_enum', options: LEASE_STATUS_LABELS, label: 'Status' },
    { field: 'lease_type', type: 'multi_enum', options: LEASE_TYPE_LABELS, label: 'Type' },
    { field: 'property_id', type: 'entity_ref', entityType: 'property', label: 'Property' },
    { field: 'term.end', type: 'date_range', label: 'Expiration' },
    { field: 'base_rent', type: 'money_range', label: 'Base Rent' },
  ];
</script>

<!-- Filter bar -->
<div class="flex gap-2 mb-4">
  {#each filterConfig as filter}
    <!-- Skeleton filter components based on filter.type -->
  {/each}
</div>

<!-- Table using Skeleton Table component -->
<div class="table-container">
  <!-- Table rendering with sortable columns -->
  <!-- Row click navigates to detail view -->
</div>

<!-- Pagination -->
<Paginator
  bind:settings={$store.pagination}
  on:page={(e) => store.setPage(e.detail)}
/>
```

### 5.6 Detail Component Template

```svelte
<!-- gen/ui/components/entities/lease/LeaseDetail.svelte -->
<!-- GENERATED FROM ONTOLOGY. DO NOT HAND-EDIT. -->

<script lang="ts">
  import LeaseStatusBadge from './LeaseStatusBadge.svelte';
  import LeaseActions from './LeaseActions.svelte';
  import MoneyDisplay from '../../shared/MoneyDisplay.svelte';
  import DateRangeDisplay from '../../shared/DateRangeDisplay.svelte';
  import AddressDisplay from '../../shared/AddressDisplay.svelte';
  import EnumBadge from '../../shared/EnumBadge.svelte';
  import FormSection from '../../shared/FormSection.svelte';
  import { entityStore } from '../../../stores/entity';
  import { relatedStore } from '../../../stores/related';
  import type { Lease } from '../../../types/lease.types';

  export let id: string;

  const lease = entityStore<Lease>('lease', id, ['spaces', 'tenant_roles']);
  const ledger = relatedStore('lease', id, 'ledger_entries');
</script>

{#if $lease.data}
  {@const l = $lease.data}

  <!-- Header with status and actions -->
  <div class="flex items-center justify-between mb-6">
    <div class="flex items-center gap-3">
      <h1 class="h2">Lease</h1>
      <LeaseStatusBadge status={l.status} />
    </div>
    <LeaseActions
      entityId={l.id}
      currentStatus={l.status}
      on:transition={() => lease.refetch()}
    />
  </div>

  <!-- Overview section -->
  <FormSection title="Overview">
    <div class="grid grid-cols-2 gap-4">
      <div>
        <dt class="text-sm text-surface-500">Lease Type</dt>
        <dd><EnumBadge value={l.lease_type} labels={LEASE_TYPE_LABELS} /></dd>
      </div>
      <div>
        <dt class="text-sm text-surface-500">Liability</dt>
        <dd><EnumBadge value={l.liability_type} labels={LIABILITY_TYPE_LABELS} /></dd>
      </div>
      <div>
        <dt class="text-sm text-surface-500">Term</dt>
        <dd><DateRangeDisplay value={l.term} /></dd>
      </div>
      <div>
        <dt class="text-sm text-surface-500">Base Rent</dt>
        <dd><MoneyDisplay value={l.base_rent} /></dd>
      </div>
    </div>
  </FormSection>

  <!-- CAM Terms (conditional — only show if present) -->
  {#if l.cam_terms}
    <FormSection title="CAM Terms" collapsible>
      <!-- CAM terms display fields -->
    </FormSection>
  {/if}

  <!-- Related: Tenants -->
  <FormSection title="Tenants">
    <!-- Tenant list from relationship -->
  </FormSection>

  <!-- Related: Spaces -->
  <FormSection title="Spaces">
    <!-- Space list from relationship -->
  </FormSection>

  <!-- Related: Ledger -->
  <FormSection title="Ledger" collapsible>
    <!-- Ledger entry table from relationship -->
  </FormSection>
{/if}
```

### 5.7 Shared Component: MoneyInput

```svelte
<!-- gen/ui/components/shared/MoneyInput.svelte -->

<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import type { Money } from '../../types/common.types';

  export let value: Money | null = null;
  export let min: number | null = null;
  export let currencies: string[] = ['USD'];
  export let disabled = false;

  const dispatch = createEventDispatcher();

  let displayValue = value ? (value.amount_cents / 100).toFixed(2) : '';
  let currency = value?.currency ?? 'USD';

  function handleBlur() {
    const parsed = parseFloat(displayValue);
    if (isNaN(parsed)) return;
    if (min !== null && parsed * 100 < min) return;

    const cents = Math.round(parsed * 100);
    dispatch('change', { amount_cents: cents, currency });
  }

  // Format on blur
  function formatDisplay() {
    const parsed = parseFloat(displayValue);
    if (!isNaN(parsed)) {
      displayValue = parsed.toLocaleString('en-US', {
        minimumFractionDigits: 2,
        maximumFractionDigits: 2,
      });
    }
  }
</script>

<div class="input-group input-group-divider grid-cols-[auto_1fr_auto]">
  <div class="input-group-shim">$</div>
  <input
    type="text"
    inputmode="decimal"
    bind:value={displayValue}
    on:blur={() => { handleBlur(); formatDisplay(); }}
    {disabled}
    class="input"
    placeholder="0.00"
  />
  {#if currencies.length > 1}
    <select bind:value={currency} class="select" on:change={handleBlur}>
      {#each currencies as c}
        <option value={c}>{c}</option>
      {/each}
    </select>
  {:else}
    <div class="input-group-shim">{currency}</div>
  {/if}
</div>
```

### 5.8 Shared Component: EntityRefSelect

```svelte
<!-- gen/ui/components/shared/EntityRefSelect.svelte -->

<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import { Autocomplete } from '@skeletonlabs/skeleton';
  import { apiClient } from '../../api/client';

  export let entityType: string;
  export let displayField: string = 'name';
  export let displayTemplate: string | null = null;
  export let filter: Record<string, any> = {};
  export let multiple = false;
  export let value: string | string[] | null = null;

  const dispatch = createEventDispatcher();

  let options: Array<{ label: string; value: string }> = [];
  let searchTimeout: ReturnType<typeof setTimeout>;

  async function search(query: string) {
    const params = { ...filter, search: query, limit: 20 };
    const response = await apiClient.get(`/v1/${entityType}s`, params);

    options = response.data.map((item: any) => ({
      label: displayTemplate
        ? displayTemplate.replace(/\{(\w+(?:\.\w+)*)\}/g, (_, path) =>
            path.split('.').reduce((obj: any, key: string) => obj?.[key], item) ?? '')
        : item[displayField] ?? item.id,
      value: item.id,
    }));
  }

  function handleInput(event: Event) {
    const query = (event.target as HTMLInputElement).value;
    clearTimeout(searchTimeout);
    searchTimeout = setTimeout(() => search(query), 300);
  }

  function handleSelect(event: CustomEvent) {
    dispatch('change', multiple
      ? [...(Array.isArray(value) ? value : []), event.detail.value]
      : event.detail.value
    );
  }
</script>

<Autocomplete
  {options}
  on:input={handleInput}
  on:selection={handleSelect}
  placeholder="Search..."
/>
```

---

## 6. Svelte Stores

### 6.1 Entity Store

```typescript
// gen/ui/stores/entity.ts
import { writable, derived } from 'svelte/store';
import { apiClient } from '../api/client';

export function entityStore<T>(entityType: string, id: string, include?: string[]) {
  const { subscribe, set, update } = writable<{
    data: T | null;
    loading: boolean;
    error: Error | null;
  }>({ data: null, loading: true, error: null });

  async function fetch() {
    update(s => ({ ...s, loading: true, error: null }));
    try {
      const params = include ? { include: include.join(',') } : {};
      const data = await apiClient.get<T>(`/v1/${entityType}s/${id}`, params);
      set({ data, loading: false, error: null });
    } catch (error) {
      set({ data: null, loading: false, error: error as Error });
    }
  }

  fetch(); // initial load

  return { subscribe, refetch: fetch };
}
```

### 6.2 State Machine Store

```typescript
// gen/ui/stores/stateMachine.ts

export function stateMachineStore(entityType: string, currentStatus: string) {
  // Import the generated transition config for this entity type
  // (from the UI schema state_machine section)
  // Return valid transitions for current status
  // Provide transition function that calls the appropriate API endpoint
}
```

---

## 7. Generator Implementation

### 7.1 cmd/uigen — Schema Generator

```
Pipeline:
  1. Load ontology CUE files → parsed entity/field/constraint/state_machine/relationship structures
  2. Load codegen/uigen.cue → field type mapping rules, section grouping rules
  3. For each entity:
     a. Map fields to schema field types (Section 4.1)
     b. Determine required/optional/conditional (Section 4.2)
     c. Determine show/hide per view (Section 4.3)
     d. Extract visibility rules from constraints (Section 4.4)
     e. Process state machine for colors and actions (Section 4.5)
     f. Process relationships for detail sections (Section 4.6)
     g. Process embedded types for form sections (Section 4.7)
     h. Generate validation rules from constraints
     i. Generate API endpoint mapping from apigen.cue
     j. Write {entity}.schema.json
  4. Generate _enums.schema.json from all enum types
```

### 7.2 cmd/uirender — Svelte Renderer

```
Pipeline:
  1. Load gen/ui/schema/*.schema.json
  2. For each entity schema:
     a. Generate TypeScript types (from schema fields)
     b. Generate API client (from schema api section)
     c. Generate validation (from schema validation section)
     d. Generate form config (from schema form.sections + visibility rules)
     e. Apply Svelte form template → {Entity}Form.svelte
     f. Apply Svelte detail template → {Entity}Detail.svelte
     g. Apply Svelte list template → {Entity}List.svelte
     h. If status exists: apply status badge template → {Entity}StatusBadge.svelte
     i. If state machine exists: apply actions template → {Entity}Actions.svelte
  3. For each embedded type: generate section component → sections/{Type}Section.svelte
  4. Generate shared components (one-time, from common.cue)
  5. Generate stores (one-time, generic)
  6. Generate enum file (from _enums.schema.json)
  7. Generate barrel exports
```

Templates live in `cmd/uirender/templates/`:
```
cmd/uirender/templates/
├── form.svelte.tmpl
├── detail.svelte.tmpl
├── list.svelte.tmpl
├── status_badge.svelte.tmpl
├── actions.svelte.tmpl
├── section.svelte.tmpl
├── types.ts.tmpl
├── api.ts.tmpl
├── validation.ts.tmpl
├── config.ts.tmpl
└── enums.ts.tmpl
```

### 7.3 Makefile Integration

```makefile
generate: generate-ent generate-api generate-events generate-authz generate-agent generate-ui

generate-ui: generate-ui-schema generate-ui-svelte

generate-ui-schema:
	go run ./cmd/uigen

generate-ui-svelte:
	go run ./cmd/uirender
```

---

## 8. What the Generator Does NOT Produce

- Page layouts (how forms/lists/details compose into pages and routes)
- Navigation (sidebar, breadcrumbs, routing)
- Authentication UI (login, session)
- Custom visualizations (charts, dashboards, graphs)
- Drag-and-drop interactions
- Mobile-specific layouts
- Dark mode / theme configuration (handled by Skeleton theme layer)
- Error boundaries and global error handling
- Loading skeleton screens
- Animations and transitions beyond Skeleton defaults
- Search pages (global search across entity types)

These require design and UX decisions. The generator produces correct, functional, typed, validated, API-wired components. A designer and frontend engineer compose them into a complete application.

**Boundary: the generator handles data correctness. Humans handle user experience.**

---

## 9. Regeneration Behavior

When any ontology .cue file changes:

```
1. cmd/uigen re-reads CUE files, regenerates all .schema.json files
2. cmd/uirender re-reads schemas, regenerates all Svelte components
3. Report: "X schemas updated, Y components regenerated, Z new components, W removed"
```

All generated files carry a header:

```
<!-- GENERATED FROM PROPELLER ONTOLOGY. DO NOT HAND-EDIT. -->
<!-- Source: ontology/lease.cue -->
<!-- Generated: 2026-02-25T10:30:00Z -->
```

Custom behavior uses wrapper components that import and extend generated components. Generated components are never hand-edited.

---

## 10. Real-Time Event Wiring

### 10.1 The Free Lunch

The ontology already emits events tagged with every entity they reference (via the activity indexing in the signal system). The generated stores already know which entity they're watching. Wiring them together means every generated component gets real-time updates with zero per-entity code.

```
Backend:
  Ent Hook → NATS JetStream event → Activity Indexer (indexes by entity ID)
                                   → WebSocket Gateway (fans out to subscribers)

Frontend:
  WebSocket Client → Event Router → Entity Store → Component re-renders
```

A user looking at Lease #4872's detail view sees it update live when:
- A payment posts against that lease
- A tenant submits a maintenance request referencing that lease's space
- A co-worker changes the lease status from another browser tab
- The nightly ML model updates the renewal probability

No polling. No manual refresh. No per-component subscription code. The ontology already defines which events affect which entities — the frontend just listens.

### 10.2 Architecture

**WebSocket Gateway** (Go, backend service):
- Accepts WebSocket connections from authenticated browser sessions
- Client subscribes to entity channels: `entity:{entity_type}:{entity_id}`
- Gateway subscribes to corresponding NATS JetStream subjects
- Fans out matching events to connected WebSocket clients
- Handles connection lifecycle, heartbeat, reconnection tokens

**Why not NATS directly in the browser:**
NATS has WebSocket support, but exposing NATS subjects directly to the browser leaks internal topology, bypasses authorization, and makes permission changes require frontend changes. The gateway is a thin authorization-aware bridge.

**Event flow:**

```
1. User A updates Lease #4872 status → active
2. Ent hook emits: lease.status_changed { lease_id: "4872", space_ids: ["91", "92"], person_ids: ["301", "302"] }
3. Activity indexer creates index entries for lease:4872, space:91, space:92, person:301, person:302
4. WebSocket gateway receives event from NATS
5. Gateway checks subscriber map:
   - User B has detail view of lease:4872 open → send event
   - User C has list view of spaces (includes space:91) → send event
   - User D has tenant profile of person:301 open → send event
6. Each client's event router dispatches to the appropriate store
7. Stores invalidate/refetch. Components re-render.
```

### 10.3 WebSocket Gateway

```go
// internal/ws/gateway.go

// Client subscription: subscribe to events affecting specific entities
// Message format (client → server):
{
  "action": "subscribe",
  "channels": [
    "entity:lease:4872",
    "entity:space:91",
    "entity:person:301"
  ]
}

{
  "action": "unsubscribe",
  "channels": ["entity:lease:4872"]
}

// Message format (server → client):
{
  "channel": "entity:lease:4872",
  "event_type": "lease.status_changed",
  "entity_type": "lease",
  "entity_id": "4872",
  "timestamp": "2026-02-25T10:30:00Z",
  "actor_id": "user_123",
  "payload": {
    "field": "status",
    "old_value": "pending_signature",
    "new_value": "active"
  },
  "affected_entities": [
    { "type": "lease", "id": "4872" },
    { "type": "space", "id": "91" },
    { "type": "space", "id": "92" },
    { "type": "person", "id": "301" },
    { "type": "person", "id": "302" }
  ]
}
```

**Authorization in the gateway:**
- On connect: validate session token, extract user permissions
- On subscribe: check user has read access to the requested entity (via OPA policy, same as API)
- On event fan-out: re-check permission (entity visibility may have changed)
- Events never contain data the user shouldn't see — they contain just enough to trigger a refetch through the authorized API

**Gateway is NOT a data transport.** It carries notifications, not payloads. The `payload` field contains minimal change metadata (which field changed, old/new status). The store uses this to decide whether to refetch or apply an optimistic update. Actual data always comes through the authorized API.

### 10.4 Generated Event Registry

The ontology already defines events and which entities they reference. `cmd/uigen` extracts this into a client-side registry:

```typescript
// gen/ui/stores/eventRegistry.ts
// GENERATED FROM ONTOLOGY. DO NOT HAND-EDIT.

export type EntityType = 'lease' | 'space' | 'person' | 'property' | 'building'
  | 'portfolio' | 'person_role' | 'application' | 'work_order' | 'account'
  | 'ledger_entry' | 'journal_entry' | 'bank_account' | 'reconciliation'
  | 'jurisdiction' | 'jurisdiction_rule';

export type EventAction = 'created' | 'updated' | 'deleted' | 'status_changed'
  | 'field_changed' | 'relationship_added' | 'relationship_removed';

// For each event type, which entity types might be affected?
// Used by list views to know when to refetch.
export const EVENT_ENTITY_IMPACT: Record<string, EntityType[]> = {
  'lease.created':           ['lease', 'space', 'person', 'property'],
  'lease.status_changed':    ['lease', 'space', 'person'],
  'lease.updated':           ['lease'],
  'lease.terminated':        ['lease', 'space', 'person'],
  'payment.received':        ['ledger_entry', 'journal_entry', 'lease', 'person'],
  'payment.reversed':        ['ledger_entry', 'journal_entry', 'lease', 'person'],
  'work_order.created':      ['work_order', 'space', 'property'],
  'work_order.status_changed': ['work_order', 'space'],
  'work_order.assigned':     ['work_order'],
  'application.submitted':   ['application', 'space', 'property'],
  'application.approved':    ['application', 'space', 'lease'],
  'space.status_changed':    ['space', 'property', 'building'],
  'person.updated':          ['person', 'person_role'],
  'journal_entry.posted':    ['journal_entry', 'ledger_entry', 'account'],
  'journal_entry.voided':    ['journal_entry', 'ledger_entry', 'account'],
  'reconciliation.completed': ['reconciliation', 'bank_account', 'ledger_entry'],
  
  // Jurisdiction events
  'jurisdiction_rule.activated':  ['jurisdiction_rule', 'jurisdiction', 'property'],
  'jurisdiction_rule.superseded': ['jurisdiction_rule', 'jurisdiction', 'property'],
  'jurisdiction_rule.expired':    ['jurisdiction_rule', 'jurisdiction', 'property'],
  'jurisdiction_rule.repealed':   ['jurisdiction_rule', 'jurisdiction', 'property'],
  'property_jurisdiction.added':  ['property', 'jurisdiction'],
  'property_jurisdiction.removed': ['property', 'jurisdiction'],
};

// For each entity type, which related entity types should refetch
// when this entity changes? Derived from relationships.cue.
export const ENTITY_RELATIONSHIPS: Record<EntityType, EntityType[]> = {
  lease:           ['space', 'person', 'person_role', 'ledger_entry'],
  space:           ['lease', 'property', 'building', 'work_order'],
  person:          ['person_role', 'lease'],
  property:        ['space', 'building', 'portfolio', 'jurisdiction'],
  building:        ['space', 'property'],
  portfolio:       ['property'],
  person_role:     ['person', 'lease'],
  application:     ['person', 'space', 'property'],
  work_order:      ['space', 'property'],
  account:         ['ledger_entry'],
  ledger_entry:    ['journal_entry', 'lease', 'account'],
  journal_entry:   ['ledger_entry'],
  bank_account:    ['reconciliation'],
  reconciliation:  ['bank_account', 'ledger_entry'],
  jurisdiction:    ['property', 'jurisdiction_rule'],
  jurisdiction_rule: ['jurisdiction', 'property'],
};
```

This file is regenerated from the ontology's event definitions and relationship graph. When a new entity or event type is added to the ontology, the registry updates automatically.

### 10.5 Client-Side Event System

```typescript
// gen/ui/stores/events.ts

import { writable, get } from 'svelte/store';

// --- WebSocket Connection ---

type ConnectionState = 'connecting' | 'connected' | 'disconnected' | 'reconnecting';

interface EventMessage {
  channel: string;
  event_type: string;
  entity_type: string;
  entity_id: string;
  timestamp: string;
  actor_id: string;
  payload: Record<string, any>;
  affected_entities: Array<{ type: string; id: string }>;
}

type EventCallback = (event: EventMessage) => void;

class EventClient {
  private ws: WebSocket | null = null;
  private subscriptions = new Map<string, Set<EventCallback>>();
  private pendingSubscriptions = new Set<string>();
  private reconnectAttempts = 0;
  private maxReconnectDelay = 30_000; // 30 seconds
  private heartbeatInterval: ReturnType<typeof setInterval> | null = null;

  public state = writable<ConnectionState>('disconnected');

  connect(wsUrl: string, sessionToken: string) {
    this.state.set('connecting');
    this.ws = new WebSocket(`${wsUrl}?token=${sessionToken}`);

    this.ws.onopen = () => {
      this.state.set('connected');
      this.reconnectAttempts = 0;
      this.resubscribeAll();
      this.startHeartbeat();
    };

    this.ws.onmessage = (msg) => {
      const event: EventMessage = JSON.parse(msg.data);
      this.dispatch(event);
    };

    this.ws.onclose = () => {
      this.state.set('reconnecting');
      this.stopHeartbeat();
      this.reconnect(wsUrl, sessionToken);
    };

    this.ws.onerror = () => {
      this.ws?.close();
    };
  }

  subscribe(channel: string, callback: EventCallback): () => void {
    if (!this.subscriptions.has(channel)) {
      this.subscriptions.set(channel, new Set());
      this.sendSubscribe([channel]);
    }
    this.subscriptions.get(channel)!.add(callback);

    // Return unsubscribe function
    return () => {
      const callbacks = this.subscriptions.get(channel);
      if (callbacks) {
        callbacks.delete(callback);
        if (callbacks.size === 0) {
          this.subscriptions.delete(channel);
          this.sendUnsubscribe([channel]);
        }
      }
    };
  }

  // Subscribe to all events affecting a specific entity
  subscribeEntity(entityType: string, entityId: string, callback: EventCallback): () => void {
    return this.subscribe(`entity:${entityType}:${entityId}`, callback);
  }

  // Subscribe to all events of a given entity type (for list views)
  subscribeEntityType(entityType: string, callback: EventCallback): () => void {
    return this.subscribe(`entity_type:${entityType}`, callback);
  }

  private dispatch(event: EventMessage) {
    // Dispatch to specific entity channel
    const entityChannel = `entity:${event.entity_type}:${event.entity_id}`;
    this.subscriptions.get(entityChannel)?.forEach(cb => cb(event));

    // Dispatch to all affected entities
    for (const affected of event.affected_entities) {
      const affectedChannel = `entity:${affected.type}:${affected.id}`;
      if (affectedChannel !== entityChannel) {
        this.subscriptions.get(affectedChannel)?.forEach(cb => cb(event));
      }
    }

    // Dispatch to entity type channels (for list views)
    const typeChannel = `entity_type:${event.entity_type}`;
    this.subscriptions.get(typeChannel)?.forEach(cb => cb(event));

    // Also notify related entity type channels
    for (const affected of event.affected_entities) {
      const affectedTypeChannel = `entity_type:${affected.type}`;
      if (affectedTypeChannel !== typeChannel) {
        this.subscriptions.get(affectedTypeChannel)?.forEach(cb => cb(event));
      }
    }
  }

  private sendSubscribe(channels: string[]) {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify({ action: 'subscribe', channels }));
    } else {
      channels.forEach(c => this.pendingSubscriptions.add(c));
    }
  }

  private sendUnsubscribe(channels: string[]) {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify({ action: 'unsubscribe', channels }));
    }
    channels.forEach(c => this.pendingSubscriptions.delete(c));
  }

  private resubscribeAll() {
    const allChannels = [
      ...this.subscriptions.keys(),
      ...this.pendingSubscriptions,
    ];
    if (allChannels.length > 0) {
      this.sendSubscribe(allChannels);
      this.pendingSubscriptions.clear();
    }
  }

  private reconnect(wsUrl: string, sessionToken: string) {
    const delay = Math.min(
      1000 * Math.pow(2, this.reconnectAttempts),
      this.maxReconnectDelay
    );
    this.reconnectAttempts++;
    setTimeout(() => this.connect(wsUrl, sessionToken), delay);
  }

  private startHeartbeat() {
    this.heartbeatInterval = setInterval(() => {
      if (this.ws?.readyState === WebSocket.OPEN) {
        this.ws.send(JSON.stringify({ action: 'ping' }));
      }
    }, 30_000);
  }

  private stopHeartbeat() {
    if (this.heartbeatInterval) {
      clearInterval(this.heartbeatInterval);
      this.heartbeatInterval = null;
    }
  }
}

// Singleton — one WebSocket connection per browser tab
export const eventClient = new EventClient();
```

### 10.6 Store Integration

The entity stores from Section 6 gain automatic event subscription. When a store mounts, it subscribes to events for its entity. When an event arrives, it decides whether to refetch or apply an optimistic patch.

```typescript
// gen/ui/stores/entity.ts (updated with real-time)
import { writable, onDestroy } from 'svelte/store';
import { apiClient } from '../api/client';
import { eventClient } from './events';

export function entityStore<T>(entityType: string, id: string, include?: string[]) {
  const { subscribe, set, update } = writable<{
    data: T | null;
    loading: boolean;
    error: Error | null;
    lastEvent: string | null;
  }>({ data: null, loading: true, error: null, lastEvent: null });

  async function fetch() {
    update(s => ({ ...s, loading: true, error: null }));
    try {
      const params = include ? { include: include.join(',') } : {};
      const data = await apiClient.get<T>(`/v1/${entityType}s/${id}`, params);
      set({ data, loading: false, error: null, lastEvent: null });
    } catch (error) {
      set({ data: null, loading: false, error: error as Error, lastEvent: null });
    }
  }

  // Subscribe to real-time events for this entity
  const unsubscribe = eventClient.subscribeEntity(entityType, id, (event) => {
    // Skip events caused by the current user's own actions
    // (those are handled by optimistic updates in entityMutation)
    if (event.actor_id === getCurrentUserId()) return;

    if (event.payload?.field === 'status') {
      // Status change: can apply optimistically without refetch
      update(s => {
        if (s.data && 'status' in (s.data as any)) {
          return {
            ...s,
            data: { ...s.data, status: event.payload.new_value } as T,
            lastEvent: event.event_type,
          };
        }
        return s;
      });
    } else {
      // General change: refetch to get fresh data
      update(s => ({ ...s, lastEvent: event.event_type }));
      fetch();
    }
  });

  fetch(); // initial load

  return {
    subscribe,
    refetch: fetch,
    destroy: unsubscribe,  // call on component unmount
  };
}
```

```typescript
// gen/ui/stores/entityList.ts (updated with real-time)
import { writable } from 'svelte/store';
import { apiClient } from '../api/client';
import { eventClient } from './events';
import { EVENT_ENTITY_IMPACT } from './eventRegistry';

export function entityListStore<T>(entityType: string, options: {
  defaultSort?: { field: string; direction: 'asc' | 'desc' };
  defaultFilters?: Record<string, any>;
  pageSize?: number;
}) {
  const { subscribe, set, update } = writable<{
    data: T[];
    total: number;
    loading: boolean;
    error: Error | null;
    stale: boolean;  // true when we know data has changed but haven't refetched
    pagination: { page: number; size: number };
    sort: { field: string; direction: string };
    filters: Record<string, any>;
  }>({
    data: [],
    total: 0,
    loading: true,
    error: null,
    stale: false,
    pagination: { page: 0, size: options.pageSize ?? 25 },
    sort: options.defaultSort ?? { field: 'updated_at', direction: 'desc' },
    filters: options.defaultFilters ?? {},
  });

  async function fetch() {
    // ... standard list fetch logic ...
  }

  // Subscribe to entity type channel — any create/update/delete of this type
  const unsubscribe = eventClient.subscribeEntityType(entityType, (event) => {
    if (event.actor_id === getCurrentUserId()) return;

    if (event.event_type.endsWith('.created') || event.event_type.endsWith('.deleted')) {
      // Item added or removed — refetch immediately (count changed)
      fetch();
    } else {
      // Item updated — mark stale, show indicator, debounce refetch
      update(s => ({ ...s, stale: true }));
      debouncedFetch();
    }
  });

  // Debounce rapid updates (e.g., bulk operations)
  let fetchTimeout: ReturnType<typeof setTimeout>;
  function debouncedFetch() {
    clearTimeout(fetchTimeout);
    fetchTimeout = setTimeout(fetch, 500);
  }

  fetch();

  return {
    subscribe,
    refetch: fetch,
    setPage: (page: number) => { /* ... */ fetch(); },
    setSort: (field: string) => { /* ... */ fetch(); },
    setFilter: (field: string, value: any) => { /* ... */ fetch(); },
    destroy: unsubscribe,
  };
}
```

### 10.7 Component Wiring

The generated components automatically use the event-aware stores. No additional code needed per component. The only requirement is calling `destroy()` on unmount:

```svelte
<!-- Detail view: automatically subscribes to entity:lease:4872 -->
<script lang="ts">
  import { onDestroy } from 'svelte';
  import { entityStore } from '../../../stores/entity';

  export let id: string;
  const store = entityStore<Lease>('lease', id, ['spaces', 'tenant_roles']);

  onDestroy(() => store.destroy());
</script>

{#if $store.data}
  <!-- Renders. Re-renders automatically when events arrive. -->
  <!-- Status badge updates live. Related sections refetch. -->
{/if}
```

```svelte
<!-- List view: automatically subscribes to entity_type:lease -->
<script lang="ts">
  import { onDestroy } from 'svelte';
  import { entityListStore } from '../../../stores/entityList';

  const store = entityListStore<Lease>('lease', {
    defaultSort: { field: 'updated_at', direction: 'desc' },
  });

  onDestroy(() => store.destroy());
</script>

<!-- Stale indicator when data has changed -->
{#if $store.stale}
  <div class="alert variant-soft-warning">
    <span>Data has been updated.</span>
    <button class="btn btn-sm variant-filled" on:click={() => store.refetch()}>
      Refresh
    </button>
  </div>
{/if}
```

### 10.8 Subscription Lifecycle

Subscriptions follow component lifecycle automatically:

```
Component mounts
  → Store created
    → eventClient.subscribeEntity(type, id) called
      → WebSocket subscribe message sent to gateway
        → Gateway subscribes to NATS subject

Component unmounts
  → onDestroy calls store.destroy()
    → eventClient unsubscribe callback fires
      → If no more listeners for that channel, WebSocket unsubscribe sent
        → Gateway drops NATS subscription
```

**Page navigation:** SvelteKit's page transitions destroy old components, unsubscribing their channels, and mount new ones, subscribing to new channels. No leaked subscriptions.

**Tab visibility:** When browser tab becomes hidden (Page Visibility API), the WebSocket connection stays alive but the client stops refetching on events. Events queue. When the tab becomes visible again, one refetch per active store catches up. This prevents unnecessary API calls for background tabs.

```typescript
// In events.ts
document.addEventListener('visibilitychange', () => {
  if (document.hidden) {
    eventClient.pause();  // queue events, don't dispatch
  } else {
    eventClient.resume(); // dispatch queued events (triggers refetches)
  }
});
```

### 10.9 Optimistic Updates + Event Reconciliation

When the current user performs an action (status transition, field update), the mutation store applies the change optimistically before the API responds. When the server event arrives confirming the change, the store skips the redundant refetch:

```typescript
// gen/ui/stores/entityMutation.ts
export async function transitionEntity(
  entityType: string,
  entityId: string,
  endpoint: string,
  optimisticUpdate: Partial<any>,
) {
  // 1. Apply optimistic update to the entity store immediately
  entityCache.patch(entityType, entityId, optimisticUpdate);

  // 2. Record the pending mutation (so event handler knows to skip)
  pendingMutations.add(`${entityType}:${entityId}:${Date.now()}`);

  try {
    // 3. Call the API
    const result = await apiClient.post(endpoint);

    // 4. On success: reconcile with server response (server is source of truth)
    entityCache.set(entityType, entityId, result);
  } catch (error) {
    // 5. On failure: rollback optimistic update
    entityCache.rollback(entityType, entityId);
    throw error;
  } finally {
    // 6. Clear pending mutation after short delay
    // (allows the echo event from server to be ignored)
    setTimeout(() => {
      pendingMutations.delete(`${entityType}:${entityId}:${Date.now()}`);
    }, 5000);
  }
}
```

User experience: click "Activate" on a lease → status badge immediately turns green → API call completes → server event arrives → skipped (already up to date). If API fails → badge rolls back to previous color, error toast shown. Zero perceived latency.

### 10.10 Multi-Tab Coordination

Multiple browser tabs for the same user share the same WebSocket connection via `BroadcastChannel`:

```typescript
// In events.ts
const broadcast = new BroadcastChannel('propeller-events');

// Leader tab: has the WebSocket connection, broadcasts to followers
// Follower tabs: receive events from leader via BroadcastChannel

// Leader election: first tab to connect becomes leader
// If leader closes: next tab promotes itself and opens WebSocket
```

This prevents N WebSocket connections for N open tabs. One connection, N listeners.

### 10.11 Gateway Implementation Notes

**Backend WebSocket Gateway** (`internal/ws/`):

```go
// Subscription map: channel → set of connection IDs
// NATS consumer: one JetStream consumer per gateway instance
// Fan-out: event arrives from NATS → look up channel subscribers → send to each

// Key design decisions:
// 1. Gateway does NOT filter by permission at subscribe time for most entities
//    (user can only see entities they already have in the UI, which means
//     they already passed the API-level permission check)
// 2. Gateway DOES check permission for entity_type subscriptions
//    (list views need row-level filtering, which the API handles on refetch)
// 3. Event payload is minimal (entity type, id, event type, changed field)
//    No sensitive data in the event — actual data comes via authorized API refetch
// 4. One NATS consumer group per gateway pod (shared nothing between pods)
//    Each pod manages its own WebSocket connections

// Scale: 10K concurrent WebSocket connections per gateway pod
// At 10K managed units with ~50 events/minute across portfolio:
//   ~50 NATS messages/minute → fan-out to ~10-20 relevant connections each
//   ~500-1000 WebSocket messages/minute across all connections
//   Trivial load
```

### 10.12 What This Enables

With real-time wiring, the generated UI gets several capabilities for free:

**Collaborative editing awareness:** Two property managers looking at the same lease see each other's changes live. No save conflicts. No stale data.

**Live dashboards:** A portfolio overview showing vacancy counts, pending applications, overdue balances updates in real-time as events flow through the system. No polling interval. No refresh button.

**Agent visibility:** When the AI agent processes a maintenance request (routes to vendor, sends communication), the property manager watching the work order detail view sees each step happen live. The agent's actions flow through the same event system as human actions.

**Operational awareness:** Site manager has the lease list open. A new application comes in → list updates with new row. A payment posts → lease's balance indicator changes. A work order escalates → the urgency badge on the related space updates. All without the manager doing anything.

**Audit trail transparency:** Because events carry `actor_id`, the UI can show "Updated by Sarah Chen 2 minutes ago" or "Updated by AI Agent 30 seconds ago" on detail views. Users see who (human or agent) made every change.

### 10.13 What This Does NOT Do

- **Conflict resolution.** If two users edit the same field simultaneously, last write wins at the API level. The event system shows the result, not the conflict. For true collaborative editing (Google Docs style), you need CRDTs or OT — that's a different problem and not in scope.
- **Offline support.** If the WebSocket disconnects, events are missed. On reconnect, stores refetch current state. No event replay to the client. (NATS JetStream handles replay on the backend for the signal system, but the frontend doesn't need it — it just needs current state.)
- **Cross-entity computed aggregations.** "Total portfolio vacancy rate" doesn't update from a single space event. Aggregations are computed server-side and exposed as their own endpoints. The real-time system can notify when an aggregation changes, but doesn't compute it client-side.

### 10.14 Generation Rules

`cmd/uigen` adds to each entity schema:

```json
{
  "realtime": {
    "subscribable": true,
    "channels": {
      "entity": "entity:{entity_type}:{id}",
      "entity_type": "entity_type:{entity_type}"
    },
    "events": [
      "lease.created",
      "lease.updated",
      "lease.status_changed",
      "lease.terminated"
    ],
    "optimistic_fields": ["status"],
    "refetch_on": ["*"]
  }
}
```

Derivation rules:
- `subscribable: true` for all entities with events defined in the ontology
- `events`: collected from events.cue for this entity type
- `optimistic_fields`: status field (if entity has state machine) + any enum field that's displayed as a badge
- `refetch_on: ["*"]`: default. Refetch on any non-optimistic event. Can be narrowed per entity if needed.

`cmd/uirender` reads the realtime schema section and:
- Wires entity stores with `eventClient.subscribeEntity` calls
- Wires list stores with `eventClient.subscribeEntityType` calls
- Generates `onDestroy` cleanup in every component
- Adds stale indicator to list components
- Adds `lastEvent` display to detail components (optional, for debugging)