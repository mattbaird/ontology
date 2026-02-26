# Propeller Intelligence Layer Specification

**Version:** 1.0  
**Date:** February 25, 2026  
**Author:** Matthew Baird, CTO — AppFolio  
**Status:** For Claude Code Implementation  
**Depends on:** propeller-ontology-spec-v2.md, propeller-signal-discovery-spec-v2.md

---

## 1. Governing Principle

LLMs are for language, not for math.

Every time the system calls an LLM to count events, compare dates, evaluate thresholds, detect trends, score risk, or classify text from a known taxonomy — that is waste. Traditional ML, statistical methods, and deterministic rules do all of that faster, cheaper, more reliably, and more explainably.

The LLM engages only when the task requires natural language understanding, natural language generation, or genuinely novel reasoning that cannot be reduced to a model, a rule, or a statistical test.

The intelligence layer sits between the signal system (which discovers and classifies what happened) and the agent (which communicates with humans and handles novel situations). It is the brain. The LLM is the voice.

---

## 2. Processing Tier Architecture

```
Events arrive (~100K/day for 10K units)
    │
    ▼
Tier 0: Deterministic Rules Engine ──────────── 100% of events
    Signal classification, activity indexing,
    escalation rules, materialized summaries,
    state machine enforcement
    Cost: $0 (compute only)
    │
    ▼
Tier 1: Statistical Methods ─────────────────── Runs on schedule
    Anomaly detection, trend analysis,
    behavioral baselines, statistical process control
    Cost: $0 (compute only)
    │
    ▼
Tier 2: Traditional ML Models ───────────────── Runs on schedule + on event
    Renewal prediction, delinquency scoring,
    tenant segmentation, pricing optimization,
    text classification, vendor matching
    Cost: $0 (local model inference on existing compute)
    │
    ▼
Tier 3: LLM — Haiku ────────────────────────── Routine language tasks
    Standard communications, simple summaries,
    template-based generation with personalization
    Cost: ~$80/month for 10K units
    │
    ▼
Tier 4: LLM — Sonnet ───────────────────────── Complex reasoning + conversation
    Manager Q&A, intervention planning,
    multi-entity synthesis, novel situations
    Cost: ~$120/month for 10K units
    │
    ▼
Tier 5: LLM — Opus ─────────────────────────── Critical decisions
    Eviction strategy, portfolio rebalancing,
    unprecedented anomaly investigation
    Cost: ~$90/month for 10K units
```

Total LLM spend: ~$290/month for 10,000 units (~$0.03/unit/month).

The deterministic and ML tiers handle ~98% of all decisions. The LLM tiers handle ~2% — the part that requires language.

---

## 3. Tier 0: Deterministic Rules Engine

Processes every event. No ML, no LLM. Pure logic operating on the signal registry and ontology constraints.

### 3.1 What It Does

```
For every domain event:
  1. Signal classification    — Registry lookup: event type + field values → category/weight/polarity
  2. Activity indexing        — Fan out to referenced entities with one-hop traversal
  3. Escalation evaluation    — Check count thresholds and cross-category rules
  4. Summary materialization  — Increment per-entity signal counters, update sentiment
  5. State machine enforcement — Reject invalid transitions (already in Ent hooks)
  6. Derived field updates    — Recompute Space.status from lease state, balance from ledger
  7. Jurisdiction constraint enforcement — On lease mutations, resolve property's jurisdiction
                                  stack and validate against applicable rules (deposit limits,
                                  notice periods, rent caps, required disclosures). Hard
                                  violations rejected; soft violations flagged for review.
```

### 3.2 Escalation Rule Evaluation

All escalation rules from the signal spec execute here as counter checks:

```
Count-based:
  "3 late payments in 180 days" →
  SELECT COUNT(*) FROM activity
  WHERE indexed_entity_id = $person_id
    AND category = 'financial'
    AND polarity = 'negative'
    AND occurred_at > NOW() - INTERVAL '180 days'
  
  If count >= 3 → write escalation record, update materialized summary

Cross-category:
  "financial negative + communication negative within 90 days" →
  Check if both category counters exceed thresholds in window.
  Pure counter arithmetic.

Absence-based:
  "No maintenance requests in 12 months from tenant with lease > 24 months" →
  Scheduled daily scan. Query for tenants matching condition with zero
  maintenance signals in window. Write absence signal to activity stream.
```

### 3.3 Materialized Signal Summaries

Per-entity pre-computed summary updated on every event. Eliminates O(N²) context accumulation at query time.

```
#MaterializedSignalSummary:
  entity_type (#EntityType)
  entity_id (string)
  
  # Per-category counts and trends (updated on each event)
  categories: map of #SignalCategory → {
    total_count (int)
    count_by_weight: map of #SignalWeight → int
    count_by_polarity: map of #SignalPolarity → int
    trend ("improving" | "stable" | "declining")  — from Tier 1 statistical analysis
    last_signal_at (time)
  }
  
  # Active escalations
  escalations: list of {
    rule_id (string)
    triggered_at (time)
    description (string)
    recommended_action (string)
  }
  
  # Overall sentiment (computed from weighted category scores)
  overall_sentiment ("positive" | "mixed" | "concerning" | "critical")
  overall_sentiment_score (float 0.0-1.0)
  
  # ML model outputs (populated by Tier 2, read by agent)
  renewal_probability (optional float 0.0-1.0)
  renewal_probability_factors (optional list of { feature, importance })
  delinquency_probability (optional float 0.0-1.0)
  tenant_segment (optional string)
  anomaly_flags (optional list of { metric, z_score, description })
  
  # Recommended actions (populated by Tier 2 models + Tier 0 rules)
  recommended_actions: list of {
    priority ("immediate" | "soon" | "routine" | "monitor")
    action_type (string — "send_renewal", "outreach", "escalate", "resolve_maintenance")
    description (string)
    source ("rule" | "model" | "anomaly")
    confidence (optional float)
  }
  
  updated_at (time)

Storage: Postgres table, one row per entity. Updated in-place on each event.
Index: (entity_type, entity_id) primary key
Additional index: (overall_sentiment_score) for portfolio-wide sorting
```

**Update cost:** O(1) per event per referenced entity. Increment counters, re-evaluate escalation rules for affected category only, recompute sentiment score.

**Read cost:** O(1) for single entity. O(T) for portfolio screening (read T rows, already sorted by sentiment score).

### 3.4 Sentiment Score Computation

Deterministic formula, not LLM judgment:

```
sentiment_score = weighted_sum(
  for each category:
    category_score = sum(
      critical_count × 10 +
      strong_count × 5 +
      moderate_count × 2 +
      weak_count × 0.5
    ) × polarity_multiplier(negative: -1, positive: +0.5, neutral: 0)
) / normalization_factor

Thresholds:
  score > 0.3:   "positive"
  score > -0.3:  "mixed"
  score > -1.0:  "concerning"
  score <= -1.0: "critical"
```

The specific weights and thresholds are tunable via `signals_overrides.cue`. The formula itself is deterministic. No LLM involved.

---

## 4. Tier 1: Statistical Methods

Runs on schedule (nightly or hourly depending on method). Populates fields on the materialized signal summary. Zero ML training required — these are classical statistics.

### 4.1 Anomaly Detection: Per-Tenant Behavioral Baselines

Every tenant accumulates a statistical profile from their activity stream.

```
#TenantBaseline:
  tenant_person_id (string)
  
  # Payment behavior
  typical_payment_day_of_month (float — median)
  payment_day_stddev (float)
  typical_payment_amount (int — median, in cents)
  
  # Maintenance behavior
  maintenance_request_rate (float — requests per month, trailing 12mo)
  complaint_rate (float — complaints per month, trailing 12mo)
  
  # Communication behavior
  avg_response_time_hours (float — median time to respond to outreach)
  response_rate (float — % of outbound communications that get a response)
  portal_login_rate (float — logins per week, trailing 3mo)
  
  # Engagement proxy
  tenant_initiated_contact_rate (float — per month)
  
  baseline_window: 12 months (rolling)
  minimum_data_points: 6 (don't compute baseline with fewer observations)

Update schedule: nightly batch
Method: Rolling median and MAD (median absolute deviation) over trailing window
```

**Anomaly flagging on each event:**

```
When a payment arrives:
  z_score = |payment_day - typical_payment_day| / payment_day_stddev
  
  if z_score > 2.0:
    Write anomaly flag to materialized summary:
    { metric: "payment_timing", z_score: 3.2, 
      description: "Paid on day 15, typical is day 3 (±1.5 days)" }

When portal activity is measured:
  current_rate = logins in last 30 days / 4 weeks
  baseline_rate = portal_login_rate from baseline
  
  if current_rate < baseline_rate × 0.3:
    Write anomaly flag:
    { metric: "portal_activity", z_score: -2.8,
      description: "Portal logins dropped 70% from baseline" }
```

Anomaly flags are signals — they flow into the activity stream and materialized summary. The agent sees them pre-computed. No LLM needed to detect that "this tenant's behavior changed."

### 4.2 Trend Analysis: Linear Regression on Time Series

```
For each tenant, for each signal category:
  
  Input: signal events in category over trailing 12 months
  
  Method:
    1. Bucket events by month
    2. Compute monthly weighted score (same formula as sentiment score)
    3. Fit linear regression: score = β₀ + β₁ × month
    4. Classify:
       β₁ < -threshold: "improving" (negative signals decreasing)
       β₁ > +threshold: "declining" (negative signals increasing)
       else: "stable"
    5. Store slope and classification on materialized summary

  Update schedule: nightly batch
  Cost: microseconds per tenant, seconds for entire portfolio
```

### 4.3 Property-Level Statistical Process Control

Detect systemic issues before individual tenant analysis would catch them.

```
For each property, maintain control charts:
  
  Metrics:
    - complaint_rate_per_unit (monthly)
    - vacancy_rate (monthly)
    - avg_days_to_fill (monthly)
    - delinquency_rate (monthly)
    - work_order_volume_per_unit (monthly)
    - avg_work_order_resolution_days (monthly)
  
  Method: Shewhart control chart (mean ± 3σ from 12-month baseline)
  
  When metric exceeds control limit:
    Write property-level anomaly signal:
    { metric: "complaint_rate", value: 0.45, control_limit: 0.28,
      description: "Complaint rate 60% above normal. Investigate systemic cause." }

  Additional: Compare building-to-building within property.
    "Building B complaint rate is 3.1x Building A this month"
    → Likely localized issue (broken elevator, HVAC, problem tenant)

  Update schedule: daily
```

### 4.4 Seasonal Adjustment

Property management has strong seasonal patterns. Fail to adjust and you'll flag normal seasonal variation as anomalies.

```
For key metrics (vacancy, applications, move-outs, maintenance volume):
  
  Method: STL decomposition (Seasonal-Trend-Loess)
    Separate: trend + seasonal component + residual
    Flag anomalies on the RESIDUAL, not the raw metric
  
  Example:
    Raw vacancy in January: 8% (looks high)
    Seasonal adjustment: January typically 7.5% for this property type
    Residual: +0.5% (within normal range)
    → No anomaly
    
    Raw vacancy in June: 6% (looks normal)
    Seasonal adjustment: June typically 3% for this property type  
    Residual: +3% (well above expected)
    → Flag anomaly despite "normal looking" raw number

  Minimum data: 24 months before seasonal decomposition is reliable.
  Fallback: Use property_type cohort seasonal patterns until enough local data.
```

### 4.5 Cohort Analysis

Compare entities against their peers to identify relative outliers.

```
Tenant cohorts (defined by Tier 2 segmentation):
  "How does this tenant compare to others in their segment?"
  
  If a "reliable_long_term" tenant starts showing financial stress,
  that's more significant than the same pattern from an "at_risk" tenant.
  
  Method: percentile rank within segment for each metric
  Output: percentile scores on materialized summary
    { payment_reliability_percentile: 0.92,
      engagement_percentile: 0.45,   ← dropped from 0.80 six months ago
      complaint_percentile: 0.78 }   ← higher = more complaints

Property cohorts (by type, size, market):
  "How does this property perform relative to comparable properties?"
  
  Method: percentile rank within cohort
  Output: property health scorecard
```

---

## 5. Tier 2: Traditional ML Models

Trained on historical outcome data. Run on schedule (nightly/weekly) or on specific events. Inference is local — fine-tuned models running on CPU/GPU in the application cluster. Zero API cost per inference.

### 5.1 Renewal Prediction Model

The most valuable model. Directly answers "which residents are at risk?"

```
Model: Gradient-boosted tree (XGBoost or LightGBM)

Features (all derived from signal system + ontology):
  Tenant features:
    tenure_months                       — from PersonRole.effective.start
    payment_on_time_rate_6m             — from ledger entries
    payment_on_time_rate_12m            — from ledger entries
    payment_trend_slope                 — from Tier 1 trend analysis
    avg_days_late_when_late             — from ledger entries
    current_balance_cents               — from materialized summary
    maintenance_complaint_count_6m      — from activity stream
    maintenance_request_count_6m        — from activity stream
    unresolved_work_order_count         — from activity stream
    avg_work_order_resolution_days      — from activity stream
    communication_response_rate         — from Tier 1 baseline
    portal_login_frequency_trend        — from Tier 1 baseline
    tenant_initiated_contact_rate       — from Tier 1 baseline
    lease_violation_count_12m           — from activity stream
    occupant_changes_12m               — from activity stream
    renewals_completed                  — from historical lease records
    tenant_segment                      — from Tier 2 segmentation (one-hot encoded)
    anomaly_flag_count_90d              — from Tier 1 anomaly detection
    escalation_count_active             — from Tier 0 escalation rules
    
  Lease features:
    rent_to_market_ratio               — lease.base_rent / space.market_rent
    rent_increase_at_last_renewal_pct  — from historical lease records
    months_until_expiration             — computed
    lease_type                          — from lease entity (one-hot)
    liability_type                      — from lease entity
    
  Space features:
    space_type                          — one-hot
    bedrooms                            — from space entity
    square_footage                      — from space entity
    floor                               — from space entity
    ada_accessible                      — from space entity
    
  Property features:
    current_vacancy_rate               — computed
    vacancy_trend                       — from Tier 1
    property_type                       — one-hot
    rent_controlled                     — from property entity
    
  Market features:
    comparable_rent_percentile          — space rent vs comparable group median
    seasonal_move_out_factor            — from Tier 1 seasonal decomposition

Target variable:
  did_renew (binary: 1 if tenant renewed or converted to M2M, 0 if vacated)

Training data:
  Every lease that reached expiration in the portfolio's history.
  Minimum: 500 lease outcomes for a usable model.
  Ideal: 2,000+ for stable feature importance.

Output per tenant:
  renewal_probability: float 0.0-1.0
  feature_importance: top 5 features driving this prediction
    e.g., [("complaint_count_6m", 0.31), ("payment_trend", 0.28), 
           ("rent_to_market_ratio", 0.22), ("response_rate", 0.11),
           ("tenure_months", 0.08)]

Schedule: nightly for all tenants with leases expiring in 120 days
Inference time: <1ms per tenant, <1s for entire portfolio
Retraining: monthly, or when 100+ new outcomes are available
```

**Why this beats LLM reasoning:** The model learns from *actual outcomes in this portfolio*. An LLM reasons from general property management knowledge. If this portfolio's tenants are unusually sensitive to maintenance response time (because it's Class A luxury), the model captures that from data. The LLM would apply average industry intuition.

**Feature importance is the key output.** The agent doesn't need to reason about *why* a tenant is at risk. The model tells it: "Top factor: complaint_count (0.31)." The agent can then focus its language capabilities on crafting the right response to that specific factor.

### 5.2 Delinquency Prediction Model

```
Model: Gradient-boosted tree

Features:
  payment_on_time_rate_3m              — short-term trend
  payment_on_time_rate_12m             — long-term pattern
  payment_trend_slope                  — from Tier 1
  current_balance_cents                — current outstanding
  nsf_count_12m                        — bounced payments
  days_since_last_payment              — recency
  income_to_rent_ratio                 — from application (if available)
  occupant_change_recent               — household disruption
  communication_responsiveness_trend   — from Tier 1
  seasonal_delinquency_factor          — from Tier 1 (January post-holiday spike)
  tenant_segment                       — from segmentation

Target: will_be_delinquent_next_month (binary: balance > 0 on day 10 of next month)

Output:
  delinquency_probability: float 0.0-1.0
  risk_tier: "low" | "moderate" | "high" | "critical"
  primary_risk_factor: string

Schedule: weekly, or triggered on payment anomaly
```

### 5.3 Tenant Segmentation

Unsupervised clustering that creates natural groupings from behavior patterns. The segments become features for other models and context for the agent.

```
Model: K-Means or HDBSCAN clustering

Features (normalized):
  tenure_months
  payment_reliability_score (composite of on-time rate + trend)
  maintenance_engagement_score (request rate, not complaint rate)
  communication_engagement_score (response rate + initiated contact rate)
  renewal_count
  rent_level_percentile (within property)
  complaint_rate

Expected emergent segments (labels assigned post-hoc):
  Cluster 0: "Reliable long-term"
    High tenure, excellent payment, low maintenance, low communication.
    These tenants are self-sufficient. Retention priority: high value, low effort.
    
  Cluster 1: "Engaged and demanding"
    Moderate tenure, good payment, high maintenance requests (not complaints).
    Active communicators. They care about their space. High retention value.
    
  Cluster 2: "At-risk"
    Declining payment, declining communication, increasing complaints.
    Active intervention needed. May be salvageable with proactive outreach.
    
  Cluster 3: "Corporate/professional"
    Perfect autopay, zero communication, zero maintenance requests.
    Low-touch. Just needs smooth renewals at market rate.
    
  Cluster 4: "New and settling in"
    < 6 months tenure. Moderate everything. Profile not yet established.
    Focus on positive onboarding experience.

Output per tenant:
  segment_id: int
  segment_label: string
  segment_confidence: float (distance to cluster center)
  
Schedule: weekly
Retraining: monthly (cluster centers may shift as portfolio evolves)

Note: Segment labels are assigned by Claude Code during model setup,
reviewing cluster characteristics and applying domain knowledge.
This is a one-time LLM cost, not per-inference.
```

### 5.4 Renewal Pricing Optimization

```
Model: Gradient-boosted regression

Features:
  current_rent_cents
  market_rent_cents (from space.market_rent)
  rent_to_market_ratio
  comparable_recent_renewal_increases (median % in comparable group)
  tenant_segment
  tenure_months
  renewal_probability (from renewal prediction model)
  current_vacancy_rate
  seasonal_demand_factor
  estimated_turn_cost_cents (from turn cost model)
  estimated_vacancy_days (from fill time model)

Target: accepted_renewal_increase_percent (from historical renewals that were accepted)

Output:
  recommended_increase_percent: float
  recommended_rent_cents: int
  expected_acceptance_probability: float
  confidence_interval: { low, high }
  context: {
    if_rejected_estimated_cost: int  — turn cost + vacancy loss
    if_accepted_annual_gain: int     — incremental rent × 12
    breakeven_vacancy_days: int      — days vacant that would offset increase
  }

Schedule: on-demand when renewal is being prepared
Also: nightly batch for all leases expiring in 90 days (pre-computed recommendations)
```

**The agent receives the recommendation as structured data.** It doesn't reason about pricing — the model already computed the optimal point. The agent's job is to *communicate* the offer appropriately given the tenant's segment and history.

### 5.5 Turn Cost Estimation

```
Model: Gradient-boosted regression

Features:
  space_type
  square_footage
  bedrooms, bathrooms
  tenant_tenure_months (longer tenure = more wear)
  last_inspection_score (if available)
  building_age
  previous_turn_costs_for_space (historical)
  previous_turn_costs_for_space_type (portfolio average)
  pet_count (from TenantAttributes)
  smoking_violation_history

Target: actual_turn_cost_cents (from historical turns)

Output:
  estimated_cost_cents: int
  confidence_interval: { low, high }
  cost_breakdown_estimate: { cleaning, paint, flooring, appliances, other }

Schedule: on lease termination event, and nightly for expiring leases
Minimum training data: 50 completed turns for usable model
```

### 5.6 Vacancy Duration (Fill Time) Prediction

```
Model: Survival analysis (Cox proportional hazards) or gradient-boosted regression

Features:
  space_type
  bedrooms, bathrooms
  square_footage
  asking_rent_to_market_ratio
  property_vacancy_rate
  seasonal_demand_factor (month of listing)
  property_type
  ada_accessible
  pet_friendly
  furnished
  floor
  comparable_fill_times_90d (recent experience for similar spaces)

Target: days_to_fill (from listing activation to lease signing)

Output:
  expected_days_to_fill: int
  probability_filled_within_30_days: float
  probability_filled_within_60_days: float

Schedule: on space vacancy event, refreshed weekly while vacant
```

### 5.7 Text Classification Models

Local models for understanding unstructured text without LLM calls.

#### Work Order Classification

```
Model: Fine-tuned sentence transformer (e.g., all-MiniLM-L6-v2) + classification head

Input: work_order.description (free text)
Output:
  category: "plumbing" | "electrical" | "hvac" | "appliance" | "structural" |
            "pest" | "locksmith" | "cleaning" | "landscaping" | "noise" |
            "parking" | "safety" | "other"
  urgency: "routine" | "urgent" | "emergency"
  sentiment: "neutral" | "frustrated" | "angry" | "concerned"
  confidence: float

Training data: historical work orders with human-assigned categories
  Minimum: 1,000 labeled work orders
  Ideal: 5,000+
  
  Bootstrap: Use Claude to label the first 1,000 from historical data,
  then human-review a sample for quality. Retrain on corrections.

Inference: <1ms on CPU, zero API cost
Fallback: If confidence < 0.7, route to Haiku for classification (~0.1% of cases)
```

#### Communication Sentiment Analysis

```
Model: Fine-tuned sentence transformer + sentiment head

Input: communication body text
Output:
  sentiment: "positive" | "neutral" | "negative" | "urgent"
  topics: list of detected topics ("maintenance", "payment", "noise", "lease", "moving")
  intent: "complaint" | "question" | "request" | "notification" | "praise" | "unclear"
  confidence: float

Training data: historical tenant communications
Fallback: confidence < 0.7 → Haiku
```

#### Maintenance Issue Deduplication

Detect when multiple work orders describe the same underlying issue.

```
Model: Sentence transformer embeddings + cosine similarity

Input: new work_order.description embedding
Compare against: recent work orders for same space/building (last 90 days)

If cosine_similarity > 0.85:
  Flag as potential duplicate or recurring issue.
  Link to previous work orders.
  
This directly feeds the "recurring_issue" signal without any LLM reasoning.
```

### 5.8 Vendor Matching and Routing

```
Model: Learning-to-rank (LambdaMART)

Features per candidate vendor:
  specialty_match_score          — vendor specialties vs work order category
  historical_rating_for_category — from past work orders of this type
  avg_response_time_hours        — historical
  avg_resolution_time_hours      — historical
  current_open_work_orders       — workload proxy
  distance_to_property           — if location data available
  cost_percentile                — historical cost vs peers for this category
  availability_today             — from vendor schedule if integrated

Target: resolution_satisfaction_score (from historical work order outcomes — was it resolved well, on time, at expected cost?)

Output:
  ranked_vendors: list of { vendor_id, match_score, expected_response_hours, expected_cost_range }
  
Schedule: on work order creation
```

### 5.9 Model Training Infrastructure

```
Training pipeline:
  1. Feature extraction: Nightly batch job queries signal system + ontology
     → produces feature matrices in Parquet format
  2. Training: Scheduled job (weekly for most models, monthly for segmentation)
     → reads feature matrices, trains models, outputs model artifacts
  3. Validation: Holdout evaluation, drift detection against previous model
  4. Deployment: Model artifact pushed to model registry, picked up by inference service
  5. Monitoring: Prediction vs actual tracking, alert on accuracy degradation

Infrastructure:
  - Feature store: Postgres materialized views over signal/activity tables
  - Training compute: Single GPU instance, shared across all models
    (none of these models are large — biggest is the sentence transformer at ~100MB)
  - Model registry: S3 + metadata in Postgres
  - Inference: CPU-only, embedded in application pods (no separate serving infrastructure)
  - Monitoring: Prediction logging, weekly accuracy reports

Retraining triggers:
  - Scheduled (weekly/monthly per model)
  - Accuracy degradation detected (prediction vs actual diverges beyond threshold)
  - Significant portfolio change (new property type onboarded, major market shift)
  
Cold start (new portfolio with no historical data):
  - Use cohort models trained on aggregate data from similar property types
  - Degrade gracefully: models output lower confidence scores
  - Signal system and statistical methods still work (they don't need training data)
  - Switch to portfolio-specific models after 6 months of data accumulation
```

---

## 6. Tier 3: LLM — Haiku (Routine Language Generation)

Haiku handles tasks that need language but not deep reasoning. The input is always pre-assembled by Tiers 0-2 — Haiku never gathers data or makes decisions. It turns structured data into natural language.

### 6.1 Scope

```
Haiku handles:
  - Standard payment reminders with personalization
  - Renewal offer communications
  - Maintenance status updates to tenants
  - Move-in/move-out instruction emails
  - Simple acknowledgment responses
  - Templated notices with variable insertion
  - Activity summaries (converting structured data to prose)

Haiku does NOT handle:
  - Any task requiring multi-step reasoning
  - Novel situations
  - Sensitive communications (eviction, hardship, legal)
  - Manager Q&A (interactive conversation)
  - Cross-entity analysis
```

### 6.2 Invocation Pattern

Haiku receives a **fully assembled context packet** — it never calls tools or gathers data.

```
Context packet structure:
{
  task: "generate_renewal_offer",
  channel: "email",
  
  tenant: {
    name: "Sarah Chen",
    segment: "reliable_long_term",
    tenure_months: 36,
    preferred_contact: "email"
  },
  
  lease: {
    current_rent: "$1,800",
    proposed_rent: "$1,875",        ← from pricing model
    increase_percent: "4.2%",       ← computed
    current_term_end: "2026-04-30",
    proposed_new_term: "12 months"
  },
  
  context_notes: [
    "Noise complaint last month — resolved in 2 days",
    "Perfect payment history — 36 consecutive on-time payments",
    "Has renewed twice before"
  ],                                ← from signal summary
  
  tone_guidance: "Warm, appreciative of long tenancy. Acknowledge resolved complaint."
                                    ← from tenant segment + signal analysis
}
```

Haiku outputs the email text. It doesn't decide the rent amount, choose the tone, or determine what context to include — all of that was decided by the models and rules before Haiku was invoked.

### 6.3 Token Budget

```
Typical Haiku invocation:
  Input:  ~500-800 tokens (context packet)
  Output: ~200-400 tokens (generated communication)
  
  No conversation history — each generation is independent.
  No tool definitions — Haiku doesn't call tools.
  No ontology context — Haiku doesn't need to understand the domain model.
  
Estimated volume for 10K units:
  ~50 communications/day (reminders, updates, acknowledgments)
  ~50 × 1,000 tokens = 50K tokens/day
  Monthly: ~1.5M tokens
  Cost at Haiku pricing: ~$0.75/month input + ~$1.50/month output ≈ $2.25/month
  
  Wait — that's far less than the $80 estimate. The $80 assumed higher volume.
  Realistic range: $2-20/month depending on communication volume and property count.
  At 10K units with active agent automation: closer to $20/month.
```

### 6.4 Quality Control

Haiku outputs are validated before sending:

```
Post-generation checks (deterministic, no LLM):
  1. Factual accuracy: All numbers in output match input context packet.
     Regex extraction of dollar amounts, dates, percentages → compare to source.
  2. Completeness: Required elements present (tenant name, amount, date, call-to-action).
  3. Length: Within channel-appropriate bounds (text: <160 chars, email: <500 words).
  4. Prohibited content: No language from blocklist. No legal claims.
  5. Personalization: Tenant name appears. Segment-appropriate tone markers present.

If any check fails: regenerate with more explicit constraints, or escalate to Sonnet.
```

---

## 7. Tier 4: LLM — Sonnet (Complex Reasoning)

Sonnet handles tasks requiring multi-step reasoning, conversation, and synthesis across entities. It receives pre-computed data from Tiers 0-2 and reasons about what to do.

### 7.1 Scope

```
Sonnet handles:
  - Manager Q&A ("What's going on with building B?")
  - Intervention planning for at-risk tenants
  - Multi-entity synthesis ("Compare my top 5 problem properties")
  - Novel situations that don't fit model predictions
  - Sensitive communications requiring nuance (hardship, disputes)
  - Complex scheduling and coordination reasoning
  - Explaining model predictions in context

Sonnet does NOT handle:
  - Any task where the answer is fully determined by a model or rule
  - Simple data retrieval (use tools directly)
  - Routine communications (Haiku)
  - Strategic portfolio decisions (Opus)
```

### 7.2 Invocation Pattern

Sonnet receives assembled context but CAN call tools for follow-up investigation. This is where the two-phase gather/reason pattern applies.

```
Phase 1: Context Assembly (deterministic, no LLM)
  The orchestrator reads the goal and assembles relevant data:
  
  Goal: "Manager asked: What's going on with Building B?"
  
  Assembly:
    1. Read materialized summaries for all spaces in Building B     → structured data
    2. Read property-level anomaly flags for Building B              → structured data
    3. Get top 5 most concerning tenants (sorted by sentiment score) → structured data
    4. Get active escalations for Building B                         → structured data
    5. Get recent anomalies from Tier 1                              → structured data
  
  All of this is pre-computed. Zero LLM calls. Pure database reads.
  Assembled into a context packet: ~2-4K tokens of dense, structured data.

Phase 2: Reasoning (Sonnet)
  Input:
    Filtered ONTOLOGY.md slice (~2K tokens)
    Filtered SIGNALS.md slice (~1K tokens)
    Assembled context packet (~3K tokens)
    Original question (~50 tokens)
    Total: ~6K input tokens
  
  Sonnet reasons and generates response: ~500-1K output tokens
  
  Total cost per invocation: ~$0.02
```

**Critical optimization:** The context assembly phase eliminates the O(N²) accumulation problem. Sonnet doesn't make 5 tool calls that stack up in conversation history. It receives one dense packet and reasons once.

### 7.3 When Sonnet Needs Follow-Up Tools

Sometimes the assembled context isn't enough. Sonnet can request additional data:

```
Sonnet: "The anomaly flags mention elevated complaints in Building B, but the
summary doesn't show what they're about. I need the actual complaint descriptions."

Tool call: GetEntityActivity
  entity_type: "building"
  entity_id: "building-b"
  categories: ["maintenance"]
  min_weight: "moderate"
  since: 90 days ago

→ Returns 12 complaints. Sonnet scans summaries, identifies pattern:
  8 of 12 mention "elevator."

Sonnet: "Building B has an elevator problem. 8 of 12 complaints in the last
90 days mention elevator issues. There are 2 open work orders related to the
elevator, both over 14 days old. This is driving the complaint rate anomaly
and is likely affecting retention for upper-floor tenants. Recommend: 
escalate elevator repair to emergency priority, and proactively reach out
to upper-floor tenants with expiring leases."
```

This follow-up adds ~4K tokens to the context. Total for the interaction: ~10K input + ~1K output = ~$0.03. Still cheap because the bulk of the work (identifying Building B as problematic, computing sentiment scores, flagging anomalies) was done by Tiers 0-2.

### 7.4 Estimated Volume

```
For 10K units:
  Manager Q&A: ~20 conversations/day × ~15K tokens avg = 300K tokens/day
  Intervention planning: ~5/day × ~10K tokens = 50K tokens/day
  Novel situation handling: ~3/day × ~20K tokens = 60K tokens/day
  Sensitive communications: ~5/day × ~8K tokens = 40K tokens/day
  
  Total: ~450K tokens/day → ~13.5M tokens/month
  Cost: ~$40/month input + ~$80/month output ≈ $120/month
```

---

## 8. Tier 5: LLM — Opus (Critical Decisions)

Opus is reserved for high-stakes reasoning where depth and nuance justify the cost. It handles situations where being wrong has significant financial or legal consequences.

### 8.1 Scope

```
Opus handles:
  - Eviction strategy evaluation (legal and financial implications)
  - Portfolio-wide rebalancing recommendations
  - Unprecedented anomaly investigation (pattern never seen before)
  - Complex multi-party dispute analysis
  - Strategic pricing recommendations for unusual properties
  - Regulatory compliance analysis for new jurisdictions
  - Year-end portfolio performance synthesis

Opus does NOT handle:
  - Anything Sonnet can handle (use Sonnet first)
  - Any task with a model or rule-based answer
  - Routine operations regardless of complexity
```

### 8.2 Invocation Pattern

Same two-phase pattern as Sonnet but with richer context assembly and explicit reasoning instructions.

```
Phase 1: Extended Context Assembly (deterministic)
  Goal: "Evaluate eviction strategy for tenant in unit 507"
  
  Assembly:
    1. Complete tenant signal history (12 months)
    2. Complete lease terms and violation history
    3. Complete payment history with amounts and dates
    4. Communication log summary
    5. Jurisdiction-specific eviction requirements (from config)
    6. Similar historical eviction outcomes in portfolio
    7. Estimated costs: legal fees, lost rent, turn cost, re-leasing time
    8. Alternative scenarios: payment plan, cash-for-keys, voluntary termination
  
  Assembled context: ~5-8K tokens of dense structured data

Phase 2: Reasoning (Opus)
  Input:
    Relevant ONTOLOGY.md slice (~2K)
    Legal/compliance context (~1K)
    Assembled context (~6K)
    Explicit reasoning request (~200)
    Total: ~10K input
  
  Opus output: ~2-3K tokens (detailed analysis with recommendations)
  Total cost per invocation: ~$0.15-0.25
```

### 8.3 Estimated Volume

```
For 10K units:
  Eviction evaluations: ~2/week × ~15K tokens = 30K tokens/week
  Portfolio analyses: ~1/week × ~30K tokens = 30K tokens/week
  Unprecedented situations: ~2/week × ~20K tokens = 40K tokens/week
  
  Total: ~100K tokens/week → ~400K tokens/month
  Cost: ~$30/month input + ~$60/month output ≈ $90/month
```

---

## 9. Invocation Router

The system needs a routing layer that decides which tier handles each task. The router itself is deterministic — no LLM.

### 9.1 Event-Triggered Routing

```
On domain event arrival:

  1. Always: Tier 0 processes (classify, index, escalate, materialize)
  
  2. Check: Does this event trigger an escalation rule?
     No  → done. No further processing.
     Yes → check escalation type:
     
       Escalation is informational (update summary only):
         → done. Agent picks it up on next query.
         
       Escalation requires outreach (e.g., 3rd late payment):
         → Tier 2: generate recommended action
         → Tier 3 (Haiku): draft communication
         → Queue for human approval or auto-send per policy
         
       Escalation is critical (e.g., financial + communication cross-category):
         → Tier 4 (Sonnet): plan intervention
         → Notify manager with plan
         
  3. Check: Does this event require text classification?
     (work order description, communication body)
     Yes → Tier 2: local text classifier
       Confident (>0.7) → use classification, done
       Not confident     → Tier 3 (Haiku): classify, done

  4. Check: Does this event trigger a model refresh?
     (e.g., payment event → refresh delinquency score for this tenant)
     Yes → Tier 2: run relevant model for this entity only
```

### 9.2 Scheduled Task Routing

```
Nightly:
  Tier 1: Refresh all tenant baselines, trend analyses, anomaly detection
  Tier 1: Property-level statistical process control
  Tier 2: Run renewal prediction for leases expiring in 120 days
  Tier 2: Run delinquency prediction for all active tenants
  Tier 2: Run turn cost estimation for leases in notice period
  Tier 0: Generate lifecycle events (expiration approaching, anniversaries)

Weekly:
  Tier 2: Refresh tenant segmentation
  Tier 2: Refresh comparable groupings
  Tier 2: Refresh fill time predictions for vacant spaces
  Tier 1: Refresh seasonal adjustments

Monthly:
  Tier 2: Retrain all models with latest outcome data
  Tier 2: Cluster label review (Claude Code reviews if segments shifted)
```

### 9.3 Human-Requested Task Routing

```
Manager asks a question:

  1. Parse intent (deterministic pattern matching, NOT LLM):
     "What's the vacancy rate?" → direct data query, no LLM
     "Show me delinquent tenants" → read materialized summaries, no LLM
     "What's going on with building B?" → Sonnet (synthesis required)
     "Should we evict tenant 507?" → Opus (high-stakes reasoning)
     "Send a renewal to Sarah Chen" → Tier 2 (pricing model) + Haiku (draft)
     
  2. For LLM-routed tasks:
     Assemble context (deterministic Phase 1)
     Route to appropriate tier
     Return response

  Intent classification uses keyword matching + simple regex patterns:
    Contains "rate", "count", "total", "how many" → data query
    Contains "what's going on", "tell me about", "summarize" → Sonnet
    Contains "should we", "evaluate", "strategy" + "evict" → Opus
    Contains "send", "draft", "write" → Haiku
    
  Ambiguous → default to Sonnet (it can triage down to Haiku if task is simple)
```

### 9.4 Cost Guards

```
Per-property daily LLM budget:
  Haiku:  no limit (cost is negligible)
  Sonnet: 50 invocations/day soft limit, 100 hard limit
  Opus:   5 invocations/day soft limit, 10 hard limit

When soft limit is reached:
  Log warning, continue processing.
  
When hard limit is reached:
  Queue non-urgent tasks for next day.
  Allow only: human-initiated requests and critical escalations.
  Alert engineering.

Per-invocation token limits:
  Haiku:  2K input, 500 output
  Sonnet: 30K input, 3K output
  Opus:   60K input, 5K output
  
  If assembled context exceeds input limit:
    Truncate oldest activity entries.
    Summarize instead of including raw data.
    If still too large: split into multiple focused queries.

Monthly cost tracking:
  Track actual LLM spend per property, per tier, per task type.
  Dashboard: actual vs budget, cost per unit, cost per decision.
  Alert if any property exceeds 2x expected spend.
```

---

## 10. Context Assembly Service

The most important cost optimization component. It sits between the routing layer and the LLM tiers, assembling exactly the right context for each task.

### 10.1 Responsibility

```
Takes:
  - Task type ("renewal_risk_assessment", "manager_qa", "draft_communication", etc.)
  - Target entities (which tenants, properties, spaces)
  - Question/goal (if human-initiated)

Returns:
  - Assembled context packet (structured data, pre-computed scores, relevant history)
  - Recommended LLM tier
  - Filtered ONTOLOGY.md slice (only relevant entities)
  - Filtered SIGNALS.md slice (only relevant categories)
  - Token budget for this task
```

### 10.2 Context Filtering Rules

```
Task: "draft_renewal_offer"
  Include: tenant name, segment, tenure, payment summary, current rent,
           proposed rent (from model), recent complaints (if any)
  Exclude: full payment history, all work order details, communication log,
           property statistics, other tenants
  ONTOLOGY.md: none needed (Haiku doesn't reason about domain model)
  SIGNALS.md: none needed
  Token budget: ~800 input

Task: "manager_qa_about_property"
  Include: property summary, vacancy rate, anomaly flags, top concerning tenants
           (from materialized summaries), active escalations, trend data
  Exclude: individual tenant payment details, work order descriptions,
           communication content, lease terms
  ONTOLOGY.md: property + space + lease entity overview (~1K tokens)
  SIGNALS.md: relevant categories only (~500 tokens)
  Token budget: ~8K input

Task: "eviction_evaluation"
  Include: EVERYTHING relevant to this tenant — full signal history, payment
           timeline, violation details, communication log, lease terms,
           jurisdiction requirements, comparable outcomes, cost analysis
  Exclude: other tenants, property-wide statistics (unless relevant to argument)
  ONTOLOGY.md: lease + person + accounting slice (~2K tokens)
  SIGNALS.md: financial + compliance + communication sections (~1K tokens)
  Token budget: ~15K input
```

### 10.3 Summary Compression

When raw data exceeds the token budget, the context assembler compresses:

```
Strategy 1: Use pre-computed summaries instead of raw data
  Instead of 200 payment records → "36 on-time, 2 late (Jan, Mar), trend: stable"
  Instead of 50 work orders → "12 complaints (8 noise, 3 plumbing, 1 pest), 
    38 routine maintenance. Avg resolution: 3.2 days."

Strategy 2: Recency bias
  Include full detail for last 90 days.
  Include summary only for 90-365 days.
  Omit detail beyond 365 days (unless specifically relevant).

Strategy 3: Relevance filtering
  For renewal assessment: expand financial and maintenance detail, compress communication.
  For delinquency assessment: expand financial detail, compress everything else.
  For maintenance assessment: expand maintenance detail, compress financial.

All compression is deterministic string formatting. No LLM is used to summarize.
```

---

## 11. Model Output Integration with Agent

The agent (Tiers 3-5) never calls Tier 2 models directly. Model outputs are pre-computed and available on the materialized signal summary. The agent reads them as structured data.

### 11.1 What the Agent Sees

When the agent retrieves a tenant's signal summary (via GetSignalSummary), it includes:

```
{
  entity_type: "person",
  entity_id: "person-312",
  
  // From Tier 0: deterministic
  overall_sentiment: "concerning",
  overall_sentiment_score: -0.72,
  escalations: [
    { rule: "maint_complaint_pattern", description: "3 complaints in 5 months" },
    { rule: "cross_maintenance_lifecycle", description: "Open complaints + lease expiring" }
  ],
  
  // From Tier 1: statistical
  anomaly_flags: [
    { metric: "payment_timing", z_score: 2.1, description: "Payment 5 days later than typical" },
    { metric: "portal_activity", z_score: -2.4, description: "Portal logins dropped 65%" }
  ],
  trends: {
    financial: "declining",
    maintenance: "declining",
    communication: "stable"
  },
  
  // From Tier 2: ML models
  renewal_probability: 0.23,
  renewal_probability_factors: [
    { feature: "complaint_count_6m", importance: 0.31 },
    { feature: "payment_trend", importance: 0.28 },
    { feature: "portal_activity_trend", importance: 0.22 }
  ],
  delinquency_probability: 0.15,
  tenant_segment: "at_risk",
  
  // Pre-computed recommended actions
  recommended_actions: [
    { priority: "immediate", action: "resolve_open_complaints",
      description: "2 noise complaints unresolved. Primary renewal risk factor.",
      source: "model" },
    { priority: "soon", action: "personal_outreach",
      description: "Schedule manager conversation before sending renewal.",
      source: "rule" },
    { priority: "routine", action: "prepare_renewal_below_market",
      description: "If retaining, price at $1,825 (1.4% increase). Model suggests 
        market-rate increase would reduce acceptance to 31%.",
      source: "model" }
  ]
}
```

The agent's job with this data is to:
1. Understand the pre-computed analysis (it reads the structured data)
2. Decide on the communication approach (language task)
3. Draft the appropriate outreach (language task)
4. Explain the reasoning to the manager if asked (language task)

It does NOT re-derive the risk score, recalculate the pricing, or re-evaluate the escalation rules. All of that is done before the agent is invoked.

### 11.2 Agent Decision Tree With Model Support

```
Agent goal: "Process renewal for expiring lease"

Step 1: Read materialized summary (zero LLM cost)
  renewal_probability: 0.23 (low — high risk of non-renewal)
  top_factor: complaint_count
  recommended_action: resolve complaints first, then personal outreach

Step 2: Agent decides approach based on pre-computed data (Sonnet — one call)
  Input: summary + recommended actions + tenant segment
  Reasoning: "This is an at-risk tenant with unresolved complaints. The model
    recommends resolving complaints before sending a renewal offer. I should
    alert the property manager to schedule a personal conversation."
  Output: communication plan (not the actual communications)

Step 3: Draft communications (Haiku — one or two calls)
  a. Internal note to manager: "Tenant 312 is at high risk of non-renewal 
     (23% likelihood). Primary concern: 3 noise complaints, 2 unresolved. 
     Recommend: resolve noise issue, then personal outreach before formal renewal."
  b. If manager approves outreach: draft the tenant communication

Total LLM cost for this renewal: ~$0.03-0.05
Without ML pre-computation: ~$0.50-0.70 (agent would need to gather all data, 
  reason about risk, compute pricing, plan intervention — all via LLM)
```

---

## 12. Cold Start Strategy

New portfolio with no historical data for ML models.

### 12.1 What Works Immediately (No Training Data Needed)

```
Tier 0: Fully operational from day one.
  Signal classification: registry lookup (no training data needed)
  Activity indexing: works on first event
  Escalation rules: count-based thresholds work with zero history
  Materialized summaries: accumulate from first event

Tier 1: Partially operational.
  Anomaly detection: needs 6+ data points per tenant for baseline
    → Start producing anomaly flags after ~6 months
  Trend analysis: needs 6+ months of monthly data
    → Start producing trends after ~6 months
  Property SPC: needs 12+ months
    → Use cohort averages (by property type) until local data exists
  Seasonal adjustment: needs 24+ months
    → Use property-type cohort seasonal patterns until local data exists

Tier 2 text classifiers: Bootstrappable.
  Use Claude to label first 1,000 historical work orders from imported data.
  Train initial classifier. Human-review sample. Retrain.
  Operational within first week.

Tiers 3-5: Fully operational from day one.
  LLMs don't need training data — they reason from domain knowledge.
  During cold start, LLMs handle MORE tasks (filling in for missing ML models).
  Cost is higher during cold start: ~3-5x the steady-state estimate.
```

### 12.2 What Needs Data (Graduated Availability)

```
Month 1-3: No ML prediction models available.
  Agent uses signal summaries + escalation rules only.
  Renewal risk: based on escalation flags, not probability score.
  Pricing: based on market_rent from Space entity, not optimization model.
  LLM cost: ~$0.10/unit/month (higher because LLM compensates for missing models)

Month 3-6: Early models with limited accuracy.
  ~200 lease outcomes → train initial renewal model (AUC ~0.65)
  ~500 payment records → train initial delinquency model
  Text classifiers fully operational.
  LLM cost: ~$0.06/unit/month (models handling more, LLM handling less)

Month 6-12: Models improving with more data.
  ~500+ lease outcomes → renewal model improves (AUC ~0.75)
  Anomaly detection baselines established for most tenants.
  Trend analysis producing meaningful results.
  LLM cost: ~$0.04/unit/month

Month 12+: Full steady state.
  ~1,000+ lease outcomes → renewal model at target accuracy (AUC ~0.80+)
  Seasonal adjustments available.
  All models at full capability.
  LLM cost: ~$0.03/unit/month
```

### 12.3 Cohort Models for Accelerated Cold Start

For new portfolios, use models trained on aggregated anonymized data from similar property types.

```
Cohort model library (maintained centrally):
  residential_multifamily_model
  residential_single_family_model
  commercial_office_model
  commercial_retail_model
  student_housing_model
  affordable_housing_model
  mixed_use_model

Each cohort model is pre-trained on aggregate patterns:
  "Across all multifamily portfolios, tenants with 3+ complaints AND
   payment trend declining have 28% renewal rate."

New portfolio starts with cohort model.
As local data accumulates, blend: (1-α) × cohort + α × local, where α 
increases with local data volume.
Eventually: local model fully replaces cohort model.
```

---

## 13. Feedback Loops and Model Improvement

### 13.1 Outcome Tracking

Every prediction and recommendation gets tracked against actual outcomes:

```
Renewal predictions:
  Prediction: renewal_probability = 0.23 for tenant 312
  Actual outcome: tenant did not renew (vacated)
  → Training example: features → 0 (did not renew)
  → Model was correct (probability was low)

  Prediction: renewal_probability = 0.85 for tenant 108
  Actual outcome: tenant did not renew (vacated)
  → Training example: features → 0
  → Model missed this one. Feature investigation:
     What signals were present that the model underweighted?

Intervention effectiveness:
  Recommendation: "Resolve complaints, then personal outreach"
  Intervention taken: Yes, complaints resolved, manager called
  Outcome: tenant renewed
  → Training signal: this intervention pattern works for "at_risk" segment
  
  Recommendation: "Resolve complaints, then personal outreach"  
  Intervention taken: Yes, complaints resolved, manager called
  Outcome: tenant still left
  → Training signal: intervention was insufficient. What else could have been done?

Agent communication effectiveness:
  Renewal offer sent via email, tenant segment: "reliable_long_term"
  Response within 24 hours: signed renewal
  → Signal: email channel effective for this segment, offer was acceptable

  Renewal offer sent via email, tenant segment: "at_risk"
  No response after 7 days
  → Signal: email may be wrong channel for this segment when at-risk
```

### 13.2 Weight Calibration Loop

Signal weights from the registry (set by Claude Code at build time) get validated against outcomes:

```
Quarterly analysis:
  For each signal in the registry:
    1. Compute correlation between signal presence and non-renewal
    2. Compare to assigned weight
    3. If correlation significantly differs from weight expectation:
       → Flag for review
       → Option: auto-adjust weight based on observed correlation
       → Option: human review and manual adjustment in overrides

Example:
  Registry says: parking_violation weight = "weak"
  Data shows: tenants with 3+ parking violations had 4x non-renewal rate
  → Auto-escalation rule was already catching this (3 violations → moderate)
  → But single violation may deserve "moderate" not "weak"
  → Update weight in signals_overrides.cue
```

### 13.3 Model Monitoring Dashboard

```
Metrics tracked per model:
  Accuracy: precision, recall, F1 (classification) or MAE, RMSE (regression)
  Calibration: are predicted probabilities well-calibrated?
  Feature drift: are input feature distributions shifting?
  Prediction drift: is the distribution of predictions shifting?
  Outcome drift: is the base rate of outcomes changing?
  Staleness: when was the model last retrained?

Alerts:
  Accuracy degradation > 5% from baseline → retrain
  Feature drift detected (PSI > 0.2) → investigate
  Prediction distribution shift → investigate
  Base rate change > 10% → retrain with fresh data
  Model age > 60 days without retrain → warning
```

---

## 14. Implementation Sequence

### Phase 1: Tier 0 Foundation (Weeks 1-2)
- Implement materialized signal summary table and update logic
- Build sentiment score computation
- Verify escalation rules run on every event
- Build GetSignalSummary and GetPortfolioSignals reading from materialized summaries
- Validate: events flow through, summaries update, portfolio screening works

### Phase 2: Tier 1 Statistical Methods (Weeks 2-4)
- Implement per-tenant baseline computation (nightly batch)
- Implement anomaly detection (on-event check against baseline)
- Implement trend analysis (nightly linear regression)
- Implement property-level SPC (daily control chart update)
- Populate anomaly_flags and trends on materialized summaries
- Validate: behavioral anomalies detected and flagged without ML or LLM

### Phase 3: Tier 2 Text Classifiers (Weeks 3-5)
- Bootstrap work order classifier: Claude labels 1K historical records
- Train sentence transformer + classification head
- Deploy as in-process model (no separate inference server)
- Wire into event pipeline: work order created → classify → use classification
- Build communication sentiment classifier with same pattern
- Build maintenance issue deduplication via embeddings
- Validate: work orders classified correctly at >90% accuracy

### Phase 4: Tier 2 Prediction Models (Weeks 5-8)
- Build feature extraction pipeline (nightly batch from signal system)
- Build model training pipeline (train, validate, deploy)
- Train initial renewal prediction model (requires historical data)
- Train delinquency prediction model
- Train turn cost estimation model
- Wire model outputs into materialized summaries
- Build model monitoring dashboard
- Validate: model predictions appear on signal summaries

### Phase 5: Context Assembly Service (Weeks 6-8)
- Build context assembler with task-specific filtering rules
- Implement summary compression strategies
- Implement token budgeting per task type
- Validate: context packets are correctly assembled, within token limits

### Phase 6: Invocation Router (Weeks 8-9)
- Build event-triggered routing logic
- Build scheduled task routing
- Build human-request intent classification (deterministic)
- Implement cost guards (per-property daily limits, per-invocation token limits)
- Validate: events route to correct tier, LLM only invoked when necessary

### Phase 7: LLM Integration — Tiers 3-5 (Weeks 9-11)
- Wire Haiku for routine communications (pre-assembled context → generation)
- Wire Sonnet for complex reasoning (context assembly → reasoning → response)
- Wire Opus for critical decisions
- Implement post-generation quality checks for Haiku
- Build outcome tracking for all LLM-generated recommendations
- Validate: end-to-end flow from event → tier routing → LLM → response

### Phase 8: Tier 2 Advanced Models (Weeks 11-14)
- Train tenant segmentation model
- Train renewal pricing optimization model
- Train vacancy duration prediction model
- Train vendor matching model
- Implement cohort models for cold start
- Build feedback loop: outcome tracking → model retraining
- Validate: model recommendations align with actual outcomes

### Phase 9: Feedback and Optimization (Weeks 14-16)
- Implement signal weight calibration loop
- Build model monitoring alerts
- Implement cold-start-to-steady-state model transition
- Performance and cost optimization
- Validate: system self-improves as data accumulates

---

## 15. File Structure for ML Components

```
propeller/
├── ml/
│   ├── features/
│   │   ├── extraction.go           # Feature extraction from signal system
│   │   ├── store.go                # Feature store interface
│   │   └── definitions/            # Feature definitions per model
│   │       ├── renewal.go
│   │       ├── delinquency.go
│   │       └── pricing.go
│   ├── models/
│   │   ├── renewal/
│   │   │   ├── train.py            # Training script (Python, runs in batch job)
│   │   │   ├── inference.go        # Go inference wrapper (loads ONNX model)
│   │   │   └── config.yaml         # Hyperparameters, feature list
│   │   ├── delinquency/
│   │   ├── segmentation/
│   │   ├── pricing/
│   │   ├── turn_cost/
│   │   ├── fill_time/
│   │   └── text_classifiers/
│   │       ├── work_order/
│   │       ├── sentiment/
│   │       └── dedup/
│   ├── registry/
│   │   ├── registry.go             # Model artifact registry
│   │   └── monitoring.go           # Prediction tracking, drift detection
│   ├── training/
│   │   ├── pipeline.go             # Orchestrates training runs
│   │   ├── validation.go           # Holdout evaluation
│   │   └── scheduler.go            # Retraining schedule
│   └── cohort/
│       ├── models/                  # Pre-trained cohort models by property type
│       └── blending.go             # Cohort-to-local model transition
├── internal/
│   ├── intelligence/
│   │   ├── router.go               # Invocation router (Tier selection)
│   │   ├── assembler.go            # Context assembly service
│   │   ├── compression.go          # Summary compression for token budgets
│   │   ├── cost_guard.go           # Per-property LLM budget enforcement
│   │   └── outcome_tracker.go      # Prediction vs actual tracking
│   ├── statistics/
│   │   ├── baseline.go             # Per-tenant behavioral baselines
│   │   ├── anomaly.go              # Anomaly detection (z-score)
│   │   ├── trend.go                # Linear regression trend analysis
│   │   ├── spc.go                  # Statistical process control
│   │   └── seasonal.go             # STL seasonal decomposition
│   └── activity/
│       └── materialized.go         # Materialized signal summary maintenance
```

---

## 16. Design Decisions

**Why gradient-boosted trees instead of neural networks for tabular prediction?**
GBTs consistently outperform neural networks on structured/tabular data with <100K training examples. They train in seconds, infer in microseconds, are fully interpretable (feature importance), and don't require GPU. Property management data is tabular with structured features. Neural networks would add complexity and training cost with no accuracy benefit.

**Why ONNX for Go inference?**
Models are trained in Python (scikit-learn/XGBoost ecosystem is mature), exported to ONNX format, and loaded by Go inference wrappers. This avoids running a separate Python inference service. The Go application directly calls the model — no network hop, sub-millisecond latency, no additional infrastructure.

**Why deterministic intent classification for human requests instead of an LLM?**
Routing a human question to the correct tier is a classification task on short text with a small number of categories. Keyword matching + regex handles 90% of cases. For the 10% that are ambiguous, defaulting to Sonnet is safe (Sonnet can always triage down). Using an LLM to decide whether to use an LLM is circular waste.

**Why materialized summaries instead of compute-on-read?**
GetPortfolioSignals screening 800 tenants with compute-on-read costs O(T × A × R) ≈ 6M operations. With materialized summaries it costs O(T) ≈ 800 row reads. The write-side cost is O(R) per event per entity — roughly 50 operations — amortized across all future reads. For any query pattern where the same entity is read more than once (and in practice it's read many times), materialized summaries win overwhelmingly.

**Why separate feature extraction from model training?**
Features are shared across models. The renewal model and pricing model both use `payment_on_time_rate_12m`. Extracting features once into a feature store and sharing them across models avoids redundant computation and ensures consistency (both models see the same feature values).

**Why track intervention effectiveness, not just prediction accuracy?**
A model can be perfectly accurate and useless if it identifies problems nobody acts on. Conversely, a less accurate model that identifies problems early enough to intervene is more valuable. Tracking whether interventions actually changed outcomes tells us whether the system is creating value, not just making correct predictions.

**Why $0.03/unit/month at steady state?**
Tier 0-2 run on existing compute with zero marginal cost per inference. LLM usage (Tiers 3-5) is ~$290/month for 10K units. The LLM is only invoked for language tasks that cannot be handled by models or rules. This cost scales linearly with units but sub-linearly with complexity — a 20K unit portfolio doesn't double the Sonnet invocations because most of the intelligence is pre-computed by Tiers 0-2.

**Why not fine-tune the LLM for property management?**
Fine-tuning would reduce per-invocation token count (less instruction needed) but adds training cost, deployment complexity, and version management. The context assembly service achieves the same token efficiency by pre-computing the heavy reasoning and feeding the LLM only what it needs. Fine-tuning is a Phase 3 optimization when volume justifies the fixed cost.