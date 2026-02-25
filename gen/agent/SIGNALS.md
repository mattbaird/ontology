# Signal Reasoning Guide

This document teaches the agent how to reason about signals from the entity activity stream.
It is auto-generated from the CUE ontology signal definitions.

## Signal Categories

### financial
Payment patterns, balance changes, fee assessments, NSF events, collection actions, credit changes.

### maintenance
Work orders, complaints, inspection results, emergency repairs, recurring issues. Both tenant-initiated and property-initiated.

### communication
Outbound contact attempts, response rates, response times, channel preferences, portal activity. Silence is a signal.

### compliance
Lease violations, policy infractions, regulatory notices, inspection failures, permit issues.

### behavioral
Occupancy pattern changes, amenity usage changes, parking behavior, portal login frequency, maintenance request pattern changes. Proxy indicators of engagement or disengagement.

### market
Comparable rents, vacancy rate changes, local market trends, seasonal patterns.

### relationship
Household composition changes, guarantor changes, emergency contact updates, roommate dynamics, co-signer events.

### lifecycle
Lease milestone events: approaching expiration, renewal window open, notice period started, move-in anniversary, option exercise deadlines.

## Signal Weights

| Weight | Meaning |
|---|---|
| critical | Requires immediate action. |
| strong | Likely to affect outcome if unaddressed. |
| moderate | Worth noting, contributes to pattern. |
| weak | Background signal, meaningful only in aggregate. |
| info | No signal value alone but provides context. |

## Signal Polarity

| Polarity | Meaning |
|---|---|
| positive | Favorable indicator. |
| negative | Unfavorable indicator. |
| neutral | Neither favorable nor unfavorable on its own. |
| contextual | Polarity depends on context. Agent must reason. |

## Signal Registrations

### financial signals

| Signal | Event Type | Weight | Polarity | Description |
|---|---|---|---|---|
| payment_on_time | PaymentRecorded | info | positive | Payment received on time |
| payment_late | PaymentRecorded | moderate | negative | Payment received late |
| payment_nsf | PaymentReturned | strong | negative | Payment returned NSF (insufficient funds) |
| late_fee_assessed | LateFeeAssessed | moderate | negative | Late fee assessed on account |
| balance_increasing | BalanceChanged | moderate | negative | Account balance increasing (growing debt) |
| payment_partial | PaymentRecorded | moderate | negative | Partial payment received |
| write_off_posted | WriteOffPosted | strong | negative | Balance written off as uncollectible |

### maintenance signals

| Signal | Event Type | Weight | Polarity | Description |
|---|---|---|---|---|
| complaint_filed | ComplaintCreated | moderate | negative | Complaint filed by tenant |
| maintenance_request | WorkOrderCreated | info | contextual | Maintenance request submitted |
| emergency_maintenance | WorkOrderCreated | strong | negative | Emergency maintenance request |
| work_order_unresolved | WorkOrderOverdue | moderate | negative | Work order past expected resolution date |
| recurring_issue | WorkOrderCreated | strong | negative | Recurring maintenance issue detected |

### communication signals

| Signal | Event Type | Weight | Polarity | Description |
|---|---|---|---|---|
| outreach_no_response | OutreachAttempted | moderate | negative | Outreach attempt with no response |
| tenant_initiated_contact | TenantContactReceived | info | positive | Tenant initiated contact |
| portal_activity_drop | PortalActivityChanged | weak | negative | Portal login frequency decreased |
| communication_preference_changed | ContactPreferenceUpdated | weak | neutral | Communication preference changed |

### compliance signals

| Signal | Event Type | Weight | Polarity | Description |
|---|---|---|---|---|
| lease_violation | LeaseViolationRecorded | strong | negative | Lease violation recorded |
| violation_cured | LeaseViolationCured | moderate | positive | Lease violation cured by tenant |
| inspection_failed | InspectionCompleted | strong | negative | Space failed inspection |

### behavioral signals

| Signal | Event Type | Weight | Polarity | Description |
|---|---|---|---|---|
| parking_violation | ParkingViolationRecorded | weak | negative | Parking violation recorded |
| amenity_usage_change | AmenityUsageChanged | weak | contextual | Amenity usage pattern changed |
| occupancy_pattern_change | OccupancyPatternChanged | weak | contextual | Occupancy pattern change detected |

### relationship signals

| Signal | Event Type | Weight | Polarity | Description |
|---|---|---|---|---|
| occupant_added | OccupantAdded | moderate | contextual | Occupant added to lease |
| occupant_removed | OccupantRemoved | moderate | negative | Occupant removed from lease |
| guarantor_change | GuarantorChanged | moderate | contextual | Guarantor added, removed, or changed |
| emergency_contact_updated | EmergencyContactUpdated | weak | neutral | Emergency contact information updated |
| roommate_departed | RoommateDeparted | strong | negative | Roommate departed the lease |

### lifecycle signals

| Signal | Event Type | Weight | Polarity | Description |
|---|---|---|---|---|
| lease_expiring_90 | LeaseExpirationApproaching | moderate | neutral | Lease expiring within 90 days |
| lease_expiring_30 | LeaseExpirationApproaching | strong | negative | Lease expiring within 30 days |
| notice_given | NoticeRecorded | critical | negative | Tenant gave notice to vacate |
| renewal_offered | RenewalOfferSent | info | neutral | Renewal offer sent to tenant |
| renewal_signed | RenewalSigned | strong | positive | Lease renewal signed |
| move_in_anniversary | MoveInAnniversary | info | positive | Move-in anniversary milestone |
| option_exercise_deadline | OptionDeadlineApproaching | strong | neutral | Renewal/expansion option exercise deadline approaching |

## Assessment Workflow

When evaluating risk, health, or status of any entity:

1. Start with GetSignalSummary for overall sentiment and category breakdown.
2. Drill into concerning categories with GetEntityActivity.
3. Look for patterns ACROSS categories — single signals are rarely actionable.
4. Check for ABSENCE of expected signals — silence from an occupied unit is itself data.
5. Evaluate trajectory, not just current state — improving vs declining matters more than absolute counts.

## Cross-Category Pattern Recognition

### Non-Renewal Predictors (in combination):
- 2+ maintenance complaints AND payment pattern worsening
- Communication responsiveness declining AND lease expiring within 90 days
- Behavioral changes (parking, amenity) AND no renewal conversation initiated
- Roommate departure AND remaining income below 3x rent

### Retention Opportunities:
- Long tenancy (2+ years) AND good payment AND recent complaint → Fast resolution retains high-value tenant.
- First-year tenant AND perfect payment AND lease expiring → Standard renewal with minor gesture has high ROI.

### Escalation Required:
- Any "critical" signal → Manager attention within 24 hours
- 3+ "strong" signals across different categories in 90 days → Proactive intervention
- Financial "critical" + Communication "strong" → In-person visit

## Category-Specific Reasoning

### Financial
- Day-of-month consistency is stronger than occasional lateness.
- Partial payments: may indicate effort or decline. Check trend direction.
- NSF is stronger than simple lateness — attempted payment with no funds.

### Maintenance
- Complaint frequency matters more than severity.
- Unresolved complaints are exponentially worse than resolved ones.
- Zero maintenance requests from a long-term tenant is unusual — possible disengagement.
- Maintenance requests (not complaints) are POSITIVE — tenant is engaged.

### Communication
- Response time TREND matters more than individual response times.
- Tenant-initiated contact is almost always positive regardless of content.
- Silence is the most dangerous communication signal.

### Behavioral
- Individual behavioral signals are weak. Only meaningful in combination.
- Parking violations: often first visible sign of norm disengagement.
- Portal activity: leading indicator — drops precede other changes.

### Relationship
- Roommate departure: check remaining tenant's income against rent.
- Occupant additions: positive (growing household) or compliance concern.
- Guarantor removal: may signal changed family dynamics.

### Lifecycle
- 90-day pre-expiration: when most renewal decisions are made.
- No renewal response by 30 days out: likely leaving.
- Option exercise deadlines: legally binding, never miss.

## Interpreting Absence

These "non-events" carry signal value:
- Long-term tenant, no maintenance requests in 12+ months: possible disengagement
- Previously active portal user stops logging in: check for other negative signals
- No response to renewal offer within 14 days: escalate
- Tenant who always paid early now pays on time: subtle trend shift, monitor
