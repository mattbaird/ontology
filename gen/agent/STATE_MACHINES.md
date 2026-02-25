# State Machines

Every entity with a `status` field has an explicit state machine.
Invalid transitions are rejected at the persistence layer.

## Application

| Current State | Valid Transitions |
|---|---|
| submitted | screening, withdrawn |
| screening | under_review, withdrawn |
| under_review | approved, conditionally_approved, denied, withdrawn |
| approved | expired |
| conditionally_approved | approved, denied, withdrawn, expired |
| denied | *(terminal)* |
| withdrawn | *(terminal)* |
| expired | *(terminal)* |

## BankAccount

| Current State | Valid Transitions |
|---|---|
| active | inactive, frozen, closed |
| inactive | active, closed |
| frozen | active, closed |
| closed | *(terminal)* |

## Building

| Current State | Valid Transitions |
|---|---|
| active | inactive, under_renovation |
| inactive | active |
| under_renovation | active |

## JournalEntry

| Current State | Valid Transitions |
|---|---|
| draft | pending_approval, posted |
| pending_approval | posted, draft |
| posted | voided |
| voided | *(terminal)* |

## Lease

| Current State | Valid Transitions |
|---|---|
| draft | pending_approval, pending_signature, terminated |
| pending_approval | draft, pending_signature, terminated |
| pending_signature | active, draft, terminated |
| active | expired, month_to_month_holdover, terminated, eviction |
| expired | active, month_to_month_holdover, renewed, terminated |
| month_to_month_holdover | active, renewed, terminated, eviction |
| renewed | *(terminal)* |
| terminated | *(terminal)* |
| eviction | terminated |

## Organization

| Current State | Valid Transitions |
|---|---|
| active | inactive, suspended, dissolved |
| inactive | active, dissolved |
| suspended | active, dissolved |
| dissolved | *(terminal)* |

## PersonRole

| Current State | Valid Transitions |
|---|---|
| pending | active, terminated |
| active | inactive, terminated |
| inactive | active, terminated |
| terminated | *(terminal)* |

## Portfolio

| Current State | Valid Transitions |
|---|---|
| onboarding | active |
| active | inactive, offboarding |
| inactive | active, offboarding |
| offboarding | inactive |

## Property

| Current State | Valid Transitions |
|---|---|
| onboarding | active |
| active | inactive, under_renovation, for_sale |
| inactive | active |
| under_renovation | active, for_sale |
| for_sale | active, inactive |

## Reconciliation

| Current State | Valid Transitions |
|---|---|
| in_progress | balanced, unbalanced |
| balanced | approved, in_progress |
| unbalanced | in_progress |
| approved | *(terminal)* |

## Space

| Current State | Valid Transitions |
|---|---|
| vacant | occupied, make_ready, down, model, reserved |
| occupied | notice_given |
| notice_given | make_ready, occupied |
| make_ready | vacant, down |
| down | make_ready, vacant |
| model | vacant, occupied |
| reserved | vacant, occupied |
| owner_occupied | vacant |

