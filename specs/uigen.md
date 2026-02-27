# Propeller UI Generation Specification v3

**Version:** 3.0  
**Date:** February 26, 2026  
**Author:** Matthew Baird, CTO — AppFolio  
**Status:** For Claude Code Implementation  
**Depends on:** propeller-ontology-spec-v2.md

---

## 1. Purpose

This spec defines how UI components are generated from two CUE inputs:

**Ontology CUE** (`ontology/*.cue`) — domain truth. Field types, constraints, state machines, relationships. Knows what a Lease IS. Says nothing about how to display one.

**View Definitions CUE** (`codegen/uigen.cue`) — UI decisions. Which fields appear in which views, column ordering, section grouping, filter configuration, display labels. Explicit declarations, not inferred heuristics.

The generator merges them: view definitions say WHAT to show, the ontology says HOW it behaves (types, validation, transitions, relationships). Neither is complete without the other.

```
Ontology (CUE) ──────────┐
                          ├──→ cmd/uigen → UI Schema (JSON) → cmd/uirender → Svelte + Skeleton + Tailwind
View Definitions (CUE) ───┘
```

Three layers:

**Layer 0: View Definitions** (`codegen/uigen.cue`). CUE file where humans (or Claude Code) declare what each entity's list, form, and detail views contain. CUE validates that every field reference actually exists in the ontology. This is where all UI decisions live.

**Layer 1: UI Schema Generator** (`cmd/uigen`). Reads both inputs. Resolves field references against ontology for type information, constraints, state machines, and relationships. Outputs framework-agnostic UI schema JSON. Does NOT guess or infer which fields to show — that's declared in Layer 0.

**Layer 2: Svelte Renderer** (`cmd/uirender`). Reads UI schema JSON, outputs Svelte components using Skeleton UI and Tailwind CSS. Thin template layer (~500 lines). Swap to another framework by writing a new renderer.

All three run in `make generate`. Change a CUE file, everything regenerates.

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
codegen/uigen.cue               — View definitions per entity (list, form, detail, shared components)
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
├── jurisdiction.schema.json
├── jurisdiction_rule.schema.json
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

## 3. Layer 0: View Definitions — `codegen/uigen.cue`

This is the CUE file where all UI decisions are declared. It imports ontology types for field reference validation. CUE guarantees every field name referenced here actually exists in the ontology — a typo is a compile error, not a silent omission.

### 3.1 View Definition Schema

```cue
package uigen

import "propeller.io/ontology"

// ===================================================================
// Schema: the shapes that view definitions must conform to
// ===================================================================

#FieldType: "string" | "text" | "int" | "float" | "bool" | "date" |
            "datetime" | "enum" | "money" | "address" | "date_range" |
            "contact_method" | "entity_ref" | "entity_ref_list" |
            "embedded_object" | "embedded_array" | "string_list"

#ColumnAlign: "left" | "right" | "center"
#SortDirection: "asc" | "desc"
#LayoutMode: "grid_2col" | "grid_3col" | "stacked"
#DisplayMode: "list" | "table" | "cards"
#FilterType: "multi_enum" | "entity_ref" | "date_range" | "money_range" | "bool" | "text_search"

#VisibilityRule: {
    field:     string
    operator:  "eq" | "neq" | "in" | "not_in" | "truthy" | "falsy"
    value?:    _
    values?:   [...]
}

// --- List View ---

#ListColumn: {
    field:          string                    // ontology field path (e.g., "base_rent", "term.end")
    label?:         string                    // override default label
    width?:         string                    // CSS width
    align?:         #ColumnAlign              // default: left
    sortable?:      bool                      // default: false
    component?:     string                    // override component: "status_badge", "money", "date", "enum_badge"
    display_as?:    string                    // for entity refs: "property.name"
}

#ListFilter: {
    field:      string
    type:       #FilterType
    label?:     string                        // override default label
    enum_ref?:  string                        // for multi_enum: which enum
    ref_entity?: string                       // for entity_ref: which entity type
}

#ListView: {
    columns:        [...#ListColumn]
    filters?:       [...#ListFilter]
    default_sort?:  {field: string, direction: #SortDirection}
    row_click?:     "navigate_to_detail" | "expand_inline" | "none"
    bulk_actions?:  bool
}

// --- Form View ---

#FormField: string                           // simple: just the field name

#FormSection: {
    id:                 string
    title:              string
    collapsible?:       bool
    initially_collapsed?: bool
    fields?:            [...#FormField]
    embedded_object?:   string               // render a sub-form for this type
    embedded_array?:    string               // render an array editor for this type
    visible_when?:      #VisibilityRule
    required_when?:     #VisibilityRule
}

#FormView: {
    sections:   [...#FormSection]
}

// --- Detail View ---

#DetailField: string | {
    field:      string
    label?:     string
    component?: string
}

#DetailSection: {
    id:             string
    title:          string
    layout?:        #LayoutMode              // default: grid_2col
    collapsible?:   bool
    visible_when?:  #VisibilityRule
    fields?:        [...#DetailField]
    embedded_object?: string
}

#RelatedSection: {
    title:          string
    relationship:   string                   // from relationships.cue
    entity:         string                   // target entity type
    display:        #DisplayMode
    include:        [...string]              // fields to show from related entity
}

#DetailView: {
    header: {
        title_template:  string              // e.g., "Lease: {space_number}"
        status_field?:   string
        show_actions?:   bool
    }
    sections:           [...#DetailSection]
    related?:           [...#RelatedSection]
}

// --- Status Badge ---

#StatusColor: "surface" | "primary" | "secondary" | "tertiary" |
              "success" | "warning" | "error"

#StatusConfig: {
    field:      string
    colors:     {[string]: #StatusColor}
}

// --- Entity View Bundle ---

#EntityViews: {
    entity_ref:     _                        // reference to ontology type for CUE validation
    display_name:   string
    display_name_plural: string
    
    list:           #ListView
    form:           #FormView
    detail:         #DetailView
    status?:        #StatusConfig
}
```

### 3.2 Lease View Definition (Complete Example)

```cue
lease: #EntityViews & {
    entity_ref:         ontology.#Lease
    display_name:       "Lease"
    display_name_plural: "Leases"
    
    list: {
        columns: [
            {field: "status",      width: "100px",  component: "status_badge", sortable: true},
            {field: "lease_type",  width: "140px",  sortable: true},
            {field: "property_id", width: "180px",  display_as: "property.name", sortable: true},
            {field: "base_rent",   width: "120px",  align: "right", sortable: true},
            {field: "term.end",    width: "120px",  label: "Expiration", sortable: true},
            {field: "updated_at",  width: "140px",  label: "Last Updated", sortable: true},
        ]
        filters: [
            {field: "status",      type: "multi_enum", enum_ref: "LeaseStatus"},
            {field: "lease_type",  type: "multi_enum", enum_ref: "LeaseType"},
            {field: "property_id", type: "entity_ref", ref_entity: "property"},
            {field: "term.end",    type: "date_range", label: "Expiration Date"},
            {field: "base_rent",   type: "money_range"},
        ]
        default_sort: {field: "updated_at", direction: "desc"}
        row_click: "navigate_to_detail"
    }
    
    form: {
        sections: [
            {
                id: "identity"
                title: "Lease Details"
                fields: ["lease_type", "liability_type", "property_id", "tenant_role_ids", "guarantor_role_ids"]
            },
            {
                id: "term"
                title: "Lease Term"
                fields: ["term", "lease_commencement_date", "rent_commencement_date"]
            },
            {
                id: "financial"
                title: "Financial Terms"
                fields: ["base_rent", "security_deposit"]
            },
            {
                id: "cam"
                title: "Common Area Maintenance"
                collapsible: true
                embedded_object: "CAMTerms"
                visible_when:  {field: "lease_type", operator: "in", values: ["commercial_nnn", "commercial_nn", "commercial_n", "commercial_gross", "commercial_modified_gross"]}
                required_when: {field: "lease_type", operator: "in", values: ["commercial_nnn", "commercial_nn", "commercial_n"]}
            },
            {
                id: "percentage_rent"
                title: "Percentage Rent"
                collapsible: true
                initially_collapsed: true
                embedded_object: "PercentageRent"
                visible_when: {field: "lease_type", operator: "in", values: ["commercial_nnn", "commercial_nn", "commercial_n", "commercial_gross", "commercial_modified_gross"]}
            },
            {
                id: "subsidy"
                title: "Subsidy Terms"
                collapsible: true
                embedded_object: "SubsidyTerms"
                visible_when:  {field: "lease_type", operator: "in", values: ["section_8", "affordable"]}
                required_when: {field: "lease_type", operator: "eq", value: "section_8"}
            },
            {
                id: "short_term"
                title: "Short-Term Rental"
                collapsible: true
                fields: ["check_in_time", "check_out_time", "cleaning_fee", "platform_booking_id"]
                visible_when: {field: "lease_type", operator: "eq", value: "short_term"}
            },
            {
                id: "membership"
                title: "Membership"
                collapsible: true
                fields: ["membership_tier"]
                visible_when: {field: "lease_type", operator: "eq", value: "membership"}
            },
            {
                id: "rent_schedule"
                title: "Rent Schedule"
                collapsible: true
                initially_collapsed: true
                embedded_array: "RentScheduleEntry"
            },
            {
                id: "recurring_charges"
                title: "Recurring Charges"
                collapsible: true
                initially_collapsed: true
                embedded_array: "RecurringCharge"
            },
            {
                id: "usage_charges"
                title: "Usage-Based Charges"
                collapsible: true
                initially_collapsed: true
                embedded_array: "UsageBasedCharge"
                visible_when: {field: "lease_type", operator: "in", values: ["commercial_nnn", "commercial_nn", "commercial_n", "commercial_gross", "commercial_modified_gross"]}
            },
            {
                id: "renewal_options"
                title: "Renewal Options"
                collapsible: true
                initially_collapsed: true
                embedded_array: "RenewalOption"
            },
            {
                id: "expansion"
                title: "Expansion Rights"
                collapsible: true
                initially_collapsed: true
                embedded_array: "ExpansionRight"
                visible_when: {field: "lease_type", operator: "in", values: ["commercial_nnn", "commercial_nn", "commercial_n", "commercial_gross", "commercial_modified_gross"]}
            },
            {
                id: "contraction"
                title: "Contraction Rights"
                collapsible: true
                initially_collapsed: true
                embedded_array: "ContractionRight"
                visible_when: {field: "lease_type", operator: "in", values: ["commercial_nnn", "commercial_nn", "commercial_n", "commercial_gross", "commercial_modified_gross"]}
            },
            {
                id: "tenant_improvement"
                title: "Tenant Improvement Allowance"
                collapsible: true
                initially_collapsed: true
                embedded_object: "TenantImprovement"
                visible_when: {field: "lease_type", operator: "in", values: ["commercial_nnn", "commercial_nn", "commercial_n", "commercial_gross", "commercial_modified_gross"]}
            },
            {
                id: "sublease"
                title: "Sublease Details"
                collapsible: true
                initially_collapsed: true
                fields: ["parent_lease_id", "sublease_billing"]
                visible_when: {field: "is_sublease", operator: "eq", value: true}
            },
            {
                id: "late_fee"
                title: "Late Fee Policy"
                collapsible: true
                initially_collapsed: true
                embedded_object: "LateFeePolicy"
            },
            {
                id: "signing"
                title: "Signing"
                collapsible: true
                initially_collapsed: true
                fields: ["signing_method", "document_id"]
            },
        ]
    }
    
    detail: {
        header: {
            title_template: "Lease: {space_number}"
            status_field: "status"
            show_actions: true
        }
        sections: [
            {
                id: "overview"
                title: "Overview"
                layout: "grid_2col"
                fields: ["lease_type", "liability_type", "status", "property_id", "term"]
            },
            {
                id: "financial"
                title: "Financial"
                layout: "grid_2col"
                fields: ["base_rent", "security_deposit"]
            },
            {
                id: "cam"
                title: "CAM Terms"
                collapsible: true
                embedded_object: "CAMTerms"
                visible_when: {field: "cam_terms", operator: "truthy"}
            },
            {
                id: "subsidy"
                title: "Subsidy Terms"
                collapsible: true
                embedded_object: "SubsidyTerms"
                visible_when: {field: "subsidy", operator: "truthy"}
            },
            {
                id: "jurisdiction"
                title: "Jurisdiction Constraints"
                collapsible: true
                fields: [
                    {field: "resolved_deposit_limit", label: "Max Security Deposit"},
                    {field: "resolved_rent_cap", label: "Rent Increase Cap"},
                    {field: "resolved_notice_period", label: "Required Notice Period"},
                ]
            },
        ]
        related: [
            {
                title: "Spaces"
                relationship: "lease_spaces"
                entity: "lease_space"
                display: "list"
                include: ["space.space_number", "space.space_type", "relationship", "effective"]
            },
            {
                title: "Tenants"
                relationship: "tenant_roles"
                entity: "person_role"
                display: "list"
                include: ["person.first_name", "person.last_name", "attributes.standing", "attributes.current_balance"]
            },
            {
                title: "Ledger"
                relationship: "ledger_entries"
                entity: "ledger_entry"
                display: "table"
                include: ["effective_date", "entry_type", "description", "amount", "reconciled"]
            },
        ]
    }
    
    status: {
        field: "status"
        colors: {
            draft:                    "surface"
            pending_approval:         "secondary"
            pending_signature:        "secondary"
            active:                   "success"
            expired:                  "warning"
            month_to_month_holdover:  "warning"
            renewed:                  "surface"
            terminated:               "surface"
            eviction:                 "error"
        }
    }
}
```

### 3.3 Property View Definition

```cue
property: #EntityViews & {
    entity_ref:         ontology.#Property
    display_name:       "Property"
    display_name_plural: "Properties"
    
    list: {
        columns: [
            {field: "status",         width: "100px",  component: "status_badge", sortable: true},
            {field: "name",           width: "200px",  sortable: true},
            {field: "property_type",  width: "140px",  sortable: true},
            {field: "address",        width: "250px",  component: "address_short"},
            {field: "total_spaces",   width: "80px",   label: "Spaces", align: "right", sortable: true},
            {field: "portfolio_id",   width: "160px",  display_as: "portfolio.name", sortable: true},
        ]
        filters: [
            {field: "status",        type: "multi_enum", enum_ref: "PropertyStatus"},
            {field: "property_type", type: "multi_enum", enum_ref: "PropertyType"},
            {field: "portfolio_id",  type: "entity_ref", ref_entity: "portfolio"},
        ]
        default_sort: {field: "name", direction: "asc"}
        row_click: "navigate_to_detail"
    }
    
    form: {
        sections: [
            {
                id: "identity"
                title: "Property Details"
                fields: ["name", "property_type", "portfolio_id", "status"]
            },
            {
                id: "address"
                title: "Address"
                embedded_object: "Address"
            },
            {
                id: "physical"
                title: "Physical Characteristics"
                fields: ["year_built", "total_square_footage", "total_spaces", "lot_size_sqft", "stories", "parking_spaces"]
            },
            {
                id: "compliance"
                title: "Compliance"
                collapsible: true
                fields: ["compliance_programs"]
                visible_when: {field: "property_type", operator: "in", values: ["affordable_housing", "senior_living"]}
            },
            {
                id: "financial"
                title: "Financial"
                collapsible: true
                fields: ["chart_of_accounts_id", "bank_account_id"]
            },
            {
                id: "insurance"
                title: "Insurance"
                collapsible: true
                initially_collapsed: true
                fields: ["insurance_policy_number", "insurance_expiry"]
            },
        ]
    }
    
    detail: {
        header: {
            title_template: "{name}"
            status_field: "status"
            show_actions: true
        }
        sections: [
            {
                id: "overview"
                title: "Overview"
                layout: "grid_2col"
                fields: ["property_type", "status", "portfolio_id", "year_built", "total_square_footage", "total_spaces"]
            },
            {
                id: "address"
                title: "Address"
                embedded_object: "Address"
            },
            {
                id: "jurisdictions"
                title: "Jurisdictions"
                collapsible: true
                // Special: rendered from PropertyJurisdiction relationship, not a field
                fields: [{field: "_jurisdictions", component: "jurisdiction_stack"}]
            },
        ]
        related: [
            {title: "Buildings", relationship: "buildings", entity: "building", display: "list", include: ["name", "building_type", "status", "floors"]},
            {title: "Spaces",    relationship: "spaces",    entity: "space",    display: "table", include: ["space_number", "space_type", "status", "square_footage"]},
            {title: "Leases",    relationship: "leases",    entity: "lease",    display: "table", include: ["status", "lease_type", "base_rent", "term.end"]},
        ]
    }
    
    status: {
        field: "status"
        colors: {
            onboarding:        "secondary"
            active:            "success"
            inactive:          "surface"
            under_renovation:  "warning"
            for_sale:          "warning"
        }
    }
}
```

### 3.4 Space View Definition

```cue
space: #EntityViews & {
    entity_ref:         ontology.#Space
    display_name:       "Space"
    display_name_plural: "Spaces"
    
    list: {
        columns: [
            {field: "status",          width: "100px",  component: "status_badge", sortable: true},
            {field: "space_number",    width: "100px",  sortable: true},
            {field: "space_type",      width: "120px",  sortable: true},
            {field: "property_id",     width: "180px",  display_as: "property.name", sortable: true},
            {field: "square_footage",  width: "100px",  align: "right", sortable: true},
            {field: "market_rent",     width: "120px",  align: "right", sortable: true},
        ]
        filters: [
            {field: "status",      type: "multi_enum", enum_ref: "SpaceStatus"},
            {field: "space_type",  type: "multi_enum", enum_ref: "SpaceType"},
            {field: "property_id", type: "entity_ref", ref_entity: "property"},
            {field: "leasable",    type: "bool", label: "Leasable Only"},
            {field: "market_rent", type: "money_range"},
        ]
        default_sort: {field: "space_number", direction: "asc"}
        row_click: "navigate_to_detail"
    }
    
    form: {
        sections: [
            {
                id: "identity"
                title: "Space Details"
                fields: ["space_number", "space_type", "property_id", "building_id", "parent_space_id"]
            },
            {
                id: "physical"
                title: "Physical"
                fields: ["floor", "square_footage", "bedrooms", "bathrooms", "leasable"]
            },
            {
                id: "pricing"
                title: "Pricing"
                fields: ["market_rent"]
            },
            {
                id: "amenities"
                title: "Amenities & Features"
                collapsible: true
                fields: ["amenities", "specialized_infrastructure"]
            },
        ]
    }
    
    detail: {
        header: {
            title_template: "Space {space_number}"
            status_field: "status"
            show_actions: true
        }
        sections: [
            {
                id: "overview"
                title: "Overview"
                layout: "grid_2col"
                fields: ["space_type", "status", "property_id", "building_id", "floor", "square_footage", "leasable"]
            },
            {
                id: "pricing"
                title: "Pricing"
                fields: ["market_rent"]
            },
        ]
        related: [
            {title: "Current Lease", relationship: "leases", entity: "lease", display: "list", include: ["status", "lease_type", "base_rent", "term.end"]},
            {title: "Sub-Spaces",    relationship: "children", entity: "space", display: "list", include: ["space_number", "space_type", "status"]},
            {title: "Work Orders",   relationship: "work_orders", entity: "work_order", display: "table", include: ["status", "priority", "description", "created_at"]},
        ]
    }
    
    status: {
        field: "status"
        colors: {
            vacant:         "success"
            occupied:       "surface"
            notice_given:   "warning"
            make_ready:     "secondary"
            down:           "error"
            model:          "tertiary"
            reserved:       "secondary"
            owner_occupied: "surface"
        }
    }
}
```

### 3.5 Person View Definition

```cue
person: #EntityViews & {
    entity_ref:         ontology.#Person
    display_name:       "Person"
    display_name_plural: "People"
    
    list: {
        columns: [
            {field: "first_name",    width: "140px",  sortable: true},
            {field: "last_name",     width: "140px",  sortable: true},
            {field: "email",         width: "220px"},
            {field: "phone",         width: "140px"},
            {field: "source",        width: "100px",  sortable: true},
        ]
        filters: [
            {field: "source",  type: "multi_enum", enum_ref: "PersonSource"},
        ]
        default_sort: {field: "last_name", direction: "asc"}
        row_click: "navigate_to_detail"
    }
    
    form: {
        sections: [
            {
                id: "identity"
                title: "Personal Information"
                fields: ["first_name", "middle_name", "last_name", "date_of_birth"]
            },
            {
                id: "contact"
                title: "Contact"
                embedded_array: "ContactMethod"
            },
            {
                id: "address"
                title: "Address"
                embedded_object: "Address"
            },
        ]
    }
    
    detail: {
        header: {
            title_template: "{first_name} {last_name}"
            show_actions: false
        }
        sections: [
            {
                id: "overview"
                title: "Overview"
                layout: "grid_2col"
                fields: ["first_name", "last_name", "date_of_birth", "source"]
            },
            {
                id: "contact"
                title: "Contact Methods"
                embedded_object: "ContactMethod"
            },
        ]
        related: [
            {title: "Roles",         relationship: "roles",        entity: "person_role",  display: "list",  include: ["role_type", "status", "scope_description"]},
            {title: "Applications",  relationship: "applications", entity: "application",  display: "table", include: ["status", "property_id", "space_id", "created_at"]},
        ]
    }
}
```

### 3.6 Accounting View Definitions

```cue
account: #EntityViews & {
    entity_ref:         ontology.#Account
    display_name:       "Account"
    display_name_plural: "Chart of Accounts"
    
    list: {
        columns: [
            {field: "account_number",  width: "120px", sortable: true},
            {field: "name",            width: "250px", sortable: true},
            {field: "account_type",    width: "140px", sortable: true},
            {field: "normal_balance",  width: "100px"},
            {field: "parent_id",       width: "180px", display_as: "parent.name"},
        ]
        filters: [
            {field: "account_type",   type: "multi_enum", enum_ref: "AccountType"},
            {field: "normal_balance", type: "multi_enum", enum_ref: "NormalBalance"},
        ]
        default_sort: {field: "account_number", direction: "asc"}
        row_click: "navigate_to_detail"
    }
    
    form: {
        sections: [
            {
                id: "identity"
                title: "Account"
                fields: ["account_number", "name", "account_type", "normal_balance", "parent_id"]
            },
            {
                id: "dimensions"
                title: "Dimensions"
                collapsible: true
                fields: ["dimensions"]
            },
        ]
    }
    
    detail: {
        header: {
            title_template: "{account_number} — {name}"
            show_actions: false
        }
        sections: [
            {
                id: "overview"
                title: "Overview"
                layout: "grid_2col"
                fields: ["account_number", "account_type", "normal_balance", "parent_id"]
            },
        ]
        related: [
            {title: "Sub-Accounts", relationship: "children", entity: "account", display: "list", include: ["account_number", "name", "account_type"]},
            {title: "Entries",      relationship: "entries",  entity: "ledger_entry", display: "table", include: ["effective_date", "description", "amount", "reconciled"]},
        ]
    }
}

journal_entry: #EntityViews & {
    entity_ref:         ontology.#JournalEntry
    display_name:       "Journal Entry"
    display_name_plural: "Journal Entries"
    
    list: {
        columns: [
            {field: "status",          width: "100px",  component: "status_badge", sortable: true},
            {field: "entry_date",      width: "120px",  sortable: true},
            {field: "source",          width: "120px",  sortable: true},
            {field: "memo",            width: "300px"},
            {field: "total_debits",    width: "120px",  align: "right", label: "Total"},
        ]
        filters: [
            {field: "status", type: "multi_enum", enum_ref: "JournalEntryStatus"},
            {field: "source", type: "multi_enum", enum_ref: "JournalEntrySource"},
            {field: "entry_date", type: "date_range"},
        ]
        default_sort: {field: "entry_date", direction: "desc"}
        row_click: "navigate_to_detail"
    }
    
    form: {
        sections: [
            {
                id: "header"
                title: "Journal Entry"
                fields: ["entry_date", "memo", "source"]
            },
            {
                id: "lines"
                title: "Lines"
                embedded_array: "LedgerEntry"
            },
        ]
    }
    
    detail: {
        header: {
            title_template: "Journal Entry — {entry_date}"
            status_field: "status"
            show_actions: true
        }
        sections: [
            {id: "overview", title: "Overview", layout: "grid_2col", fields: ["entry_date", "status", "source", "memo"]},
        ]
        related: [
            {title: "Lines", relationship: "lines", entity: "ledger_entry", display: "table", include: ["account_id", "description", "debit", "credit", "property_id"]},
        ]
    }
    
    status: {
        field: "status"
        colors: {
            draft:            "surface"
            pending_approval: "secondary"
            posted:           "success"
            voided:           "error"
        }
    }
}
```

### 3.7 Jurisdiction View Definitions

```cue
jurisdiction: #EntityViews & {
    entity_ref:         ontology.#Jurisdiction
    display_name:       "Jurisdiction"
    display_name_plural: "Jurisdictions"
    
    list: {
        columns: [
            {field: "status",            width: "100px",  component: "status_badge"},
            {field: "name",              width: "250px",  sortable: true},
            {field: "jurisdiction_type", width: "140px",  sortable: true},
            {field: "state_code",        width: "80px",   sortable: true},
            {field: "parent_jurisdiction_id", width: "200px", display_as: "parent_jurisdiction.name"},
        ]
        filters: [
            {field: "status",            type: "multi_enum", enum_ref: "JurisdictionStatus"},
            {field: "jurisdiction_type", type: "multi_enum", enum_ref: "JurisdictionType"},
            {field: "state_code",        type: "text_search"},
        ]
        default_sort: {field: "name", direction: "asc"}
        row_click: "navigate_to_detail"
    }
    
    form: {
        sections: [
            {
                id: "identity"
                title: "Jurisdiction"
                fields: ["name", "jurisdiction_type", "parent_jurisdiction_id", "country_code", "state_code", "fips_code"]
            },
            {
                id: "admin"
                title: "Administrative"
                collapsible: true
                fields: ["governing_body", "regulatory_url"]
            },
            {
                id: "dissolution"
                title: "Dissolution"
                collapsible: true
                initially_collapsed: true
                fields: ["successor_jurisdiction_id", "dissolution_date"]
                visible_when: {field: "status", operator: "in", values: ["dissolved", "merged"]}
            },
        ]
    }
    
    detail: {
        header: {
            title_template: "{name}"
            status_field: "status"
            show_actions: true
        }
        sections: [
            {id: "overview", title: "Overview", layout: "grid_2col", fields: ["jurisdiction_type", "status", "parent_jurisdiction_id", "state_code", "fips_code", "country_code"]},
            {id: "admin", title: "Administrative", fields: ["governing_body", "regulatory_url"]},
        ]
        related: [
            {title: "Rules",           relationship: "rules",    entity: "jurisdiction_rule", display: "table", include: ["rule_type", "status", "effective_date", "expiration_date", "statute_reference"]},
            {title: "Sub-Jurisdictions", relationship: "children", entity: "jurisdiction", display: "list", include: ["name", "jurisdiction_type", "status"]},
            {title: "Properties",      relationship: "properties", entity: "property", display: "table", include: ["name", "property_type", "address"]},
        ]
    }
    
    status: {
        field: "status"
        colors: {
            active:   "success"
            dissolved: "surface"
            merged:   "surface"
            pending:  "secondary"
        }
    }
}

jurisdiction_rule: #EntityViews & {
    entity_ref:         ontology.#JurisdictionRule
    display_name:       "Jurisdiction Rule"
    display_name_plural: "Jurisdiction Rules"
    
    list: {
        columns: [
            {field: "status",            width: "100px",  component: "status_badge", sortable: true},
            {field: "rule_type",         width: "180px",  sortable: true},
            {field: "jurisdiction_id",   width: "200px",  display_as: "jurisdiction.name", sortable: true},
            {field: "effective_date",    width: "120px",  sortable: true},
            {field: "expiration_date",   width: "120px",  sortable: true},
            {field: "statute_reference", width: "180px"},
        ]
        filters: [
            {field: "status",          type: "multi_enum", enum_ref: "JurisdictionRuleStatus"},
            {field: "rule_type",       type: "multi_enum", enum_ref: "JurisdictionRuleType"},
            {field: "jurisdiction_id", type: "entity_ref", ref_entity: "jurisdiction"},
            {field: "effective_date",  type: "date_range"},
        ]
        default_sort: {field: "effective_date", direction: "desc"}
        row_click: "navigate_to_detail"
    }
    
    form: {
        sections: [
            {
                id: "identity"
                title: "Rule"
                fields: ["rule_type", "jurisdiction_id", "status"]
            },
            {
                id: "applicability"
                title: "Applies To"
                fields: ["applies_to_lease_types", "applies_to_property_types", "applies_to_space_types"]
            },
            {
                id: "exemptions"
                title: "Exemptions"
                collapsible: true
                embedded_object: "RuleExemptions"
            },
            {
                id: "definition"
                title: "Rule Definition"
                // Component determined dynamically by rule_type
                embedded_object: "_dynamic_rule_definition"
            },
            {
                id: "legal"
                title: "Legal Reference"
                collapsible: true
                fields: ["statute_reference", "ordinance_number", "statute_url"]
            },
            {
                id: "dates"
                title: "Effective Dates"
                fields: ["effective_date", "expiration_date"]
            },
            {
                id: "verification"
                title: "Verification"
                collapsible: true
                initially_collapsed: true
                fields: ["last_verified", "verified_by", "verification_source"]
            },
            {
                id: "supersession"
                title: "Supersession"
                collapsible: true
                initially_collapsed: true
                fields: ["superseded_by_id"]
                visible_when: {field: "status", operator: "eq", value: "superseded"}
            },
        ]
    }
    
    detail: {
        header: {
            title_template: "{rule_type} — {jurisdiction.name}"
            status_field: "status"
            show_actions: true
        }
        sections: [
            {id: "overview", title: "Overview", layout: "grid_2col", fields: ["rule_type", "status", "jurisdiction_id", "effective_date", "expiration_date"]},
            {id: "applicability", title: "Applies To", layout: "grid_3col", fields: ["applies_to_lease_types", "applies_to_property_types", "applies_to_space_types"]},
            {id: "definition", title: "Rule Definition", embedded_object: "_dynamic_rule_definition"},
            {id: "legal", title: "Legal Reference", fields: ["statute_reference", "ordinance_number", "statute_url"]},
            {id: "verification", title: "Verification", collapsible: true, fields: ["last_verified", "verified_by", "verification_source"]},
        ]
        related: [
            {title: "Affected Properties", relationship: "affected_properties", entity: "property", display: "table", include: ["name", "property_type", "address"]},
            {title: "Supersedes",          relationship: "supersedes",          entity: "jurisdiction_rule", display: "list", include: ["rule_type", "status", "effective_date"]},
        ]
    }
    
    status: {
        field: "status"
        colors: {
            draft:      "surface"
            active:     "success"
            superseded: "surface"
            expired:    "surface"
            repealed:   "error"
        }
    }
}
```

### 3.8 Remaining Entity View Definitions

The following entities follow the same pattern. For brevity, showing compact form:

```cue
// Portfolio, Building, PersonRole, Application, LedgerEntry, BankAccount, Reconciliation
// all follow #EntityViews with explicit list/form/detail/status declarations.
// Complete definitions in the actual codegen/uigen.cue file.

portfolio: #EntityViews & {
    entity_ref: ontology.#Portfolio
    display_name: "Portfolio"
    display_name_plural: "Portfolios"
    list: {columns: [{field: "status", ...}, {field: "name", ...}, {field: "owner", ...}], ...}
    form: {sections: [...]}
    detail: {header: {title_template: "{name}", ...}, sections: [...], related: [...]}
    status: {field: "status", colors: {onboarding: "secondary", active: "success", ...}}
}

building: #EntityViews & {
    entity_ref: ontology.#Building
    display_name: "Building"
    display_name_plural: "Buildings"
    // ...
}

application: #EntityViews & {
    entity_ref: ontology.#Application
    display_name: "Application"
    display_name_plural: "Applications"
    // ...
}

// etc. for all entities with UI views
```

### 3.9 Claude Code Authoring View Definitions

View definitions are an excellent candidate for Claude Code generation with human review:

1. Claude Code reads the ontology entity definition
2. Applies property management domain knowledge to decide:
   - Which fields a PM would want in a list view (the 5-7 most important)
   - How form sections should group logically (identity → dates → financial → conditional → optional)
   - Which fields belong in the detail overview vs collapsible sections
   - Which relationships are most useful to show
3. Generates the `#EntityViews` CUE block
4. Human reviews, adjusts column selection, reorders sections, tweaks labels
5. `cue vet` validates all field references exist

This mirrors the signal annotation pattern exactly: Claude Code as domain expert at build time, humans as editors.

---

## 4. Layer 1: What the Generator Still Derives from the Ontology

View definitions declare WHAT to show. The ontology provides HOW it behaves. The generator merges both. Here's what comes from the ontology, not from view definitions:

### 4.1 Field Type → Component Mapping

The generator reads each referenced field's ontology type and maps it to a UI component:

```
CUE type                    → UI component type
──────────────────────────────────────────────────
string (short)              → "string"
string (long/description)   → "text"
int                         → "int"
float                       → "float"
bool                        → "bool"
time                        → "datetime"
time (date-only context)    → "date"
enum ("|" separated)        → "enum"
#Money                      → "money"
#NonNegativeMoney           → "money" (variant: non_negative)
#PositiveMoney              → "money" (variant: positive)
#Address                    → "address"
#DateRange                  → "date_range"
#ContactMethod              → "contact_method"
list of strings             → "string_list"
list of #X                  → "embedded_array"
single #X                   → "embedded_object"
entity reference (*_id)     → "entity_ref"
list of entity refs (*_ids) → "entity_ref_list"
```

How to distinguish "string" from "text": if field name contains "description", "memo", "notes", "reason", or "guidance" → "text". Everything else → "string". (This is the one heuristic that survives — it's about the field's nature, not about where to display it.)

### 4.2 Constraints → Validation Rules

For every field referenced in a view definition, the generator extracts its constraints from the ontology:

```
CUE constraint              → Validation rule
──────────────────────────────────────────────────
"required string"           → required: true
"optional string"           → required: false
field with default value    → required: false, default: <value>
int with min/max            → min/max validation
float > 0                   → positive validation
money variant               → non-negative/positive validation
```

### 4.3 Cross-Field Constraints → Validation Rules

For every CONSTRAINT block in the ontology:
```
"If [field_A] is [value], then [field_B] is required"
```

The generator produces a cross-field validation rule:

```json
{
    "id": "nnn_requires_cam",
    "condition": {"field": "lease_type", "operator": "eq", "value": "commercial_nnn"},
    "then": {"field": "cam_terms", "rule": "required"},
    "message": "CAM terms are required for NNN leases"
}
```

These exist independent of view definitions — they're domain constraints that apply regardless of where fields are displayed.

### 4.4 State Machines → Actions and Transitions

The generator reads state_machines.cue and for each entity that has a state machine:

1. Extracts valid transitions from each state
2. Generates action button metadata:
   - Forward-progress transitions → variant: "primary"
   - Backward/lateral transitions → variant: "secondary"
   - Terminal/negative transitions → variant: "danger"
3. All danger variants get `confirm: true` with generated confirmation messages
4. Cross-references ontology constraints for `requires_fields` on each transition target
5. Maps to API endpoints from apigen.cue

This is entirely ontology-driven. View definitions only control whether actions are shown at all (`show_actions: true` in the detail header).

### 4.5 Relationships → Related Entity Sections

View definitions declare which relationships to show and which fields to include:

```cue
related: [
    {title: "Tenants", relationship: "tenant_roles", entity: "person_role", display: "list", include: ["person.first_name", ...]}
]
```

The generator resolves the relationship from relationships.cue to get:
- Cardinality (O2M, M2M)
- API endpoint for fetching related entities
- Inverse relationship name

### 4.6 Enum Labels

Generated deterministically from enum values:
- Split on underscore, capitalize each word
- Known abbreviations stay uppercase: NNN, NN, N, CAM, ACH, CPI, NSF, HUD, LIHTC, AMI
- Known phrases get proper casing: "Section 8", "Month to Month"

Enum grouping (for dropdowns) declared in uigen.cue if needed:

```cue
#EnumGroups: {
    LeaseType: [
        {label: "Residential", values: ["fixed_term", "month_to_month", "affordable", "section_8", "student", "short_term"]},
        {label: "Commercial",  values: ["commercial_nnn", "commercial_nn", "commercial_n", "commercial_gross", "commercial_modified_gross", "ground_lease"]},
        {label: "Other",       values: ["membership"]},
    ]
}
```

### 4.7 API Endpoint Mapping

Derived from apigen.cue, not from view definitions:

```json
{
    "api": {
        "operations": {
            "create": {"method": "POST", "path": "/v1/leases"},
            "get":    {"method": "GET",  "path": "/v1/leases/{id}"},
            "list":   {"method": "GET",  "path": "/v1/leases"},
            "update": {"method": "PATCH","path": "/v1/leases/{id}"},
            "search": {"method": "POST", "path": "/v1/leases/search"}
        },
        "transitions": {
            "submit":    {"method": "POST", "path": "/v1/leases/{id}/submit"},
            "approve":   {"method": "POST", "path": "/v1/leases/{id}/approve"},
            // ... from apigen.cue
        }
    }
}
```

### 4.8 Immutability

Fields marked `@immutable` in the ontology:
- Shown in create form
- Hidden or disabled in edit form
- Shown in detail view

The view definition doesn't need to know about this — the generator handles it.

### 4.9 Audit Fields

Fields from `#AuditMetadata` (created_at, updated_at, created_by, updated_by):
- Never shown in forms
- Available in detail views (collapsed by default)
- `updated_at` available as list column if declared in view definition

### 4.10 Summary: What Comes From Where

```
                        View Definition (uigen.cue)    Ontology (*.cue)
                        ───────────────────────────    ─────────────────
Which fields in list    ✓ explicit columns             
Column width/align      ✓                              
Column labels           ✓ (override) or                ✓ (default from field name)
Column sortable         ✓                              
Filter config           ✓                              
List default sort       ✓                              
Form section grouping   ✓                              
Form section ordering   ✓                              
Section titles          ✓                              
Section visibility      ✓ visible_when                 
Section required_when   ✓                              ← (references ontology constraint)
Collapsible/collapsed   ✓                              
Detail layout           ✓                              
Detail sections         ✓                              
Related sections        ✓                              
Related display mode    ✓                              
Which related fields    ✓                              
Status badge colors     ✓                              
Display name            ✓                              
Title template          ✓                              
                                                       
Field type              from field name reference  →   ✓ CUE type
Component mapping                                      ✓ type → component
Required/optional                                      ✓ CUE required/optional
Validation rules                                       ✓ constraints
Cross-field validation                                 ✓ CONSTRAINT blocks
Enum options                                           ✓ union types
State machine actions                                  ✓ state_machines.cue
Transition buttons                                     ✓ state_machines.cue
Confirmation messages                                  ✓ generated from transition
API endpoints                                          ✓ apigen.cue
Relationship details                                   ✓ relationships.cue
Immutability                                           ✓ @immutable
Audit field handling                                   ✓ #AuditMetadata
```

---

## 5. Layer 1: UI Schema Output Format

The generator outputs one JSON file per entity. This is the contract between Layer 1 and Layer 2 (the Svelte renderer). The format is identical to v2 spec Section 3 — it hasn't changed, only the generation method has.

### 5.1 Top-Level Structure

```json
{
  "entity": "lease",
  "display_name": "Lease",
  "display_name_plural": "Leases",

  "fields": [ ... ],
  "enums": { ... },
  "form": { ... },
  "detail": { ... },
  "list": { ... },
  "status": { ... },
  "state_machine": { ... },
  "relationships": [ ... ],
  "validation": { ... },
  "api": { ... },
  "realtime": { ... }
}
```

Each section is populated by merging the view definition (layout, field selection) with ontology-derived data (types, constraints, transitions, endpoints). The JSON schema format remains exactly as specified in v2 Sections 3.2–3.13. It is framework-agnostic.

### 5.2 Field Definitions in Schema

For every field referenced in any view definition for an entity, the generator emits:

```json
{
  "name": "base_rent",
  "type": "money",
  "money_variant": "non_negative",
  "required": true,
  "default": null,
  "immutable": false,
  "label": "Base Rent",
  "help_text": "Monthly base rent amount"
}
```

Where:
- `name` — from view definition field reference
- `type`, `money_variant` — from ontology type mapping (Section 4.1)
- `required`, `default`, `immutable` — from ontology
- `label` — from view definition (if provided) or generated from field name
- `help_text` — from ontology (if docstring exists)

---

## 6. Layer 2: Svelte + Skeleton + Tailwind Renderer

The renderer reads UI schema JSON and applies Svelte templates. This section is unchanged from v2.

### 6.1 Component Mapping

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

### 6.2 Form Component Template

```svelte
<!-- gen/ui/components/entities/lease/LeaseForm.svelte -->
<!-- GENERATED FROM ONTOLOGY + VIEW DEFINITIONS. DO NOT HAND-EDIT. -->

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

  // Generated from uigen.cue form.sections[*].visible_when
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
    <!-- ... remaining fields from section declaration -->
  </FormSection>

  <!-- Conditional sections driven by visible_when from uigen.cue -->
  {#if isVisible('cam')}
    <FormSection title="Common Area Maintenance" collapsible required={...}>
      <CAMTermsSection ... />
    </FormSection>
  {/if}

  <!-- ... remaining sections exactly as declared in uigen.cue -->

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

### 6.3 Status Badge Component Template

```svelte
<!-- gen/ui/components/entities/lease/LeaseStatusBadge.svelte -->
<!-- GENERATED FROM ONTOLOGY + VIEW DEFINITIONS. DO NOT HAND-EDIT. -->

<script lang="ts">
  import type { LeaseStatus } from '../../../types/lease.types';
  import { LEASE_STATUS_LABELS } from '../../../types/enums';

  export let status: LeaseStatus;

  // Generated from uigen.cue status.colors
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

### 6.4 Actions Component Template

Generated from state_machines.cue (ontology-driven, not view-definition-driven). Same as v2 Section 5.4.

### 6.5 List Component Template

Same as v2 Section 5.5 but columns and filters come from uigen.cue declarations.

### 6.6 Detail Component Template

Same as v2 Section 5.6 but sections and related sections come from uigen.cue declarations.

### 6.7 Shared Components

MoneyInput, EntityRefSelect, AddressForm, DateRangeInput, etc. — unchanged from v2 Sections 5.7–5.8.

---

## 7. Svelte Stores

Unchanged from v2 Section 6. Entity stores, list stores, state machine stores, event-aware stores.

---

## 8. Generator Implementation

### 8.1 cmd/uigen — Schema Generator

```
Pipeline:
  1. Load ontology CUE files → parsed entity, field, constraint, state machine, relationship structures
  2. Load codegen/uigen.cue → parsed view definitions per entity
  3. Load codegen/apigen.cue → API endpoint mappings
  4. Validate: every field referenced in uigen.cue exists in ontology (CUE does this, but double-check)
  5. For each entity with a view definition:
     a. Resolve each referenced field to its ontology type → component type (Section 4.1)
     b. Extract validation rules from ontology constraints (Section 4.2, 4.3)
     c. Map state machine to action buttons (Section 4.4)
     d. Resolve relationships for related sections (Section 4.5)
     e. Generate enum labels (Section 4.6)
     f. Map API endpoints (Section 4.7)
     g. Merge view definition layout with ontology-derived metadata
     h. Add realtime subscription config (from event catalog)
     i. Write {entity}.schema.json
  6. Generate _enums.schema.json from all enum types
```

**What changed from v2:** Steps 5a–5i used to be heuristic inference ("guess which fields go in the list, which in forms, which in detail"). Now the view definition tells the generator what goes where. The generator only derives TYPE information, CONSTRAINTS, TRANSITIONS, and ENDPOINTS — things that are ontological truth, not UI opinion.

### 8.2 cmd/uirender — Svelte Renderer

```
Pipeline:
  1. Load gen/ui/schema/*.schema.json
  2. For each entity schema:
     a. Generate TypeScript types (from schema fields)
     b. Generate API client (from schema api section)
     c. Generate validation (from schema validation section)
     d. Generate form visibility config (from schema form.sections.visible_when)
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

### 8.3 Makefile Integration

```makefile
generate: generate-ent generate-api generate-events generate-authz generate-agent generate-ui

generate-ui: generate-ui-schema generate-ui-svelte

generate-ui-schema:
	go run ./cmd/uigen

generate-ui-svelte:
	go run ./cmd/uirender
```

---

## 9. What the Generator Does NOT Produce

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
- Role-based view variants (future: multiple #EntityViews per entity)

These require design and UX decisions. The generator produces correct, functional, typed, validated, API-wired components. A designer and frontend engineer compose them into a complete application.

**Boundary: the ontology handles domain correctness. View definitions handle field selection and layout. The generator handles type resolution and wiring. Humans handle user experience.**

---

## 10. Regeneration Behavior

When any ontology .cue file OR codegen/uigen.cue changes:

```
1. cmd/uigen re-reads both inputs, regenerates all .schema.json files
2. cmd/uirender re-reads schemas, regenerates all Svelte components
3. Report: "X schemas updated, Y components regenerated, Z new components, W removed"
```

All generated files carry a header:

```
<!-- GENERATED FROM PROPELLER ONTOLOGY + VIEW DEFINITIONS. DO NOT HAND-EDIT. -->
<!-- Sources: ontology/lease.cue, codegen/uigen.cue -->
<!-- Generated: 2026-02-26T10:30:00Z -->
```

Custom behavior uses wrapper components that import and extend generated components. Generated components are never hand-edited.

---

## 11. Real-Time Event Wiring

Unchanged from v2 Section 10. The real-time system is ontology-driven (events, entity references) and independent of view definitions. View definitions don't affect what events flow — they only affect what the user sees when events arrive.

### 11.1 The Free Lunch

The ontology already emits events tagged with every entity they reference (via the activity indexing in the signal system). The generated stores already know which entity they're watching. Wiring them together means every generated component gets real-time updates with zero per-entity code.

```
Backend:
  Ent Hook → NATS JetStream event → Activity Indexer (indexes by entity ID)
                                   → WebSocket Gateway (fans out to subscribers)

Frontend:
  WebSocket Client → Event Router → Entity Store → Component re-renders
```

### 11.2 Architecture

**WebSocket Gateway** (Go, backend service):
- Accepts WebSocket connections from authenticated browser sessions
- Client subscribes to entity channels: `entity:{entity_type}:{entity_id}`
- Gateway subscribes to corresponding NATS JetStream subjects
- Fans out matching events to connected WebSocket clients
- Handles connection lifecycle, heartbeat, reconnection tokens

**Gateway is NOT a data transport.** It carries notifications, not payloads. The store uses the notification to decide whether to refetch or apply an optimistic update. Actual data always comes through the authorized API.

### 11.3 Generated Event Registry

```typescript
// gen/ui/stores/eventRegistry.ts
// GENERATED FROM ONTOLOGY. DO NOT HAND-EDIT.

export type EntityType = 'lease' | 'space' | 'person' | 'property' | 'building'
  | 'portfolio' | 'person_role' | 'application' | 'work_order' | 'account'
  | 'ledger_entry' | 'journal_entry' | 'bank_account' | 'reconciliation'
  | 'jurisdiction' | 'jurisdiction_rule';

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
  'jurisdiction_rule.activated':  ['jurisdiction_rule', 'jurisdiction', 'property'],
  'jurisdiction_rule.superseded': ['jurisdiction_rule', 'jurisdiction', 'property'],
  'jurisdiction_rule.expired':    ['jurisdiction_rule', 'jurisdiction', 'property'],
  'jurisdiction_rule.repealed':   ['jurisdiction_rule', 'jurisdiction', 'property'],
  'property_jurisdiction.added':  ['property', 'jurisdiction'],
  'property_jurisdiction.removed': ['property', 'jurisdiction'],
};

export const ENTITY_RELATIONSHIPS: Record<EntityType, EntityType[]> = {
  lease:              ['space', 'person', 'person_role', 'ledger_entry'],
  space:              ['lease', 'property', 'building', 'work_order'],
  person:             ['person_role', 'lease'],
  property:           ['space', 'building', 'portfolio', 'jurisdiction'],
  building:           ['space', 'property'],
  portfolio:          ['property'],
  person_role:        ['person', 'lease'],
  application:        ['person', 'space', 'property'],
  work_order:         ['space', 'property'],
  account:            ['ledger_entry'],
  ledger_entry:       ['journal_entry', 'lease', 'account'],
  journal_entry:      ['ledger_entry'],
  bank_account:       ['reconciliation'],
  reconciliation:     ['bank_account', 'ledger_entry'],
  jurisdiction:       ['property', 'jurisdiction_rule'],
  jurisdiction_rule:  ['jurisdiction', 'property'],
};
```

### 11.4 Client-Side Event System, Store Integration, Optimistic Updates, Multi-Tab, Gateway

All unchanged from v2 Sections 10.5–10.14. See those sections for complete implementation details including:
- EventClient WebSocket class with reconnection and heartbeat
- Store integration with automatic event subscription/unsubscription
- Optimistic updates with event reconciliation
- BroadcastChannel for multi-tab coordination
- Tab visibility pausing
- Gateway implementation notes