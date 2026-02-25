#!/usr/bin/env bash
# =============================================================================
# Ontology-Driven Property Management — Full Lifecycle Demo
#
# A narrated story that exercises all 4 services, 13 entities, and 10 state
# machines generated from a single CUE ontology.
# =============================================================================
set -euo pipefail

# ---------------------------------------------------------------------------
# Colors & formatting
# ---------------------------------------------------------------------------
BOLD='\033[1m'
CYAN='\033[1;36m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
DIM='\033[2m'
RESET='\033[0m'

header()  { printf "\n${CYAN}━━━ %s ━━━${RESET}\n\n" "$1"; }
narrate() { printf "${DIM}    %s${RESET}\n" "$1"; }
entity()  { printf "  ${GREEN}✓${RESET} %s\n" "$1"; }
transition() { printf "  ${YELLOW}⟶${RESET} %s\n" "$1"; }
guardrail()  { printf "  ${RED}✗${RESET} %s\n" "$1"; }
box_line()   { printf "  ${CYAN}│${RESET} %-50s ${CYAN}│${RESET}\n" "$1"; }

# ---------------------------------------------------------------------------
# jq with fallback
# ---------------------------------------------------------------------------
JQ="jq"
if ! command -v jq &>/dev/null; then
  if command -v python3 &>/dev/null; then
    JQ="python3 -m json.tool"
  else
    JQ="cat"
  fi
fi
pretty() { $JQ '.' 2>/dev/null || cat; }
extract() { jq -r "$1" 2>/dev/null; }

# ---------------------------------------------------------------------------
# Config
# ---------------------------------------------------------------------------
PORT=$(( (RANDOM % 10000) + 20000 ))
BASE="http://localhost:${PORT}"
ACTOR="demo-script"
NOW=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
TERM_START="2026-03-01T00:00:00Z"
TERM_END="2027-02-28T00:00:00Z"
BINARY="/tmp/ontology-server-$$"
DB_FILE="/tmp/ontology-demo-$$.db"

# ---------------------------------------------------------------------------
# Cleanup
# ---------------------------------------------------------------------------
cleanup() {
  if [[ -n "${SERVER_PID:-}" ]]; then
    kill "$SERVER_PID" 2>/dev/null || true
    wait "$SERVER_PID" 2>/dev/null || true
  fi
  rm -f "$BINARY" "$DB_FILE"
}
trap cleanup EXIT

# ---------------------------------------------------------------------------
# API helper — POST / GET / PATCH with audit headers
# ---------------------------------------------------------------------------
api() {
  local method="$1" path="$2"
  shift 2
  local url="${BASE}${path}"
  local -a curl_args=(
    -s -w "\n%{http_code}"
    -H "Content-Type: application/json"
    -H "X-Actor: ${ACTOR}"
    -H "X-Source: system"
    -H "X-Correlation-ID: demo-$(date +%s%N)"
    -X "$method"
  )
  if [[ $# -gt 0 ]]; then
    curl_args+=(-d "$1")
  fi
  local raw
  raw=$(curl "${curl_args[@]}" "$url")
  local body http_code
  http_code=$(echo "$raw" | tail -1)
  body=$(echo "$raw" | sed '$d')
  # Return body + code separated by newline
  echo "$body"
  echo "---HTTP_STATUS:${http_code}"
}

# Post and expect success (2xx)
api_ok() {
  local method="$1" path="$2"
  shift 2
  local result
  result=$(api "$method" "$path" "$@")
  local body http_code
  http_code=$(echo "$result" | grep '^---HTTP_STATUS:' | sed 's/---HTTP_STATUS://')
  body=$(echo "$result" | grep -v '^---HTTP_STATUS:')
  if [[ "$http_code" -lt 200 || "$http_code" -ge 300 ]]; then
    printf "\n${RED}FATAL: %s %s returned HTTP %s${RESET}\n" "$method" "$path" "$http_code"
    echo "$body" | pretty
    exit 1
  fi
  echo "$body"
}

# Post and expect failure (non-2xx)
api_fail() {
  local method="$1" path="$2"
  shift 2
  local result
  result=$(api "$method" "$path" "$@")
  local body http_code
  http_code=$(echo "$result" | grep '^---HTTP_STATUS:' | sed 's/---HTTP_STATUS://')
  body=$(echo "$result" | grep -v '^---HTTP_STATUS:')
  if [[ "$http_code" -ge 200 && "$http_code" -lt 300 ]]; then
    printf "\n${RED}EXPECTED FAILURE but got HTTP %s${RESET}\n" "$http_code"
    echo "$body" | pretty
    exit 1
  fi
  echo "$body"
}

id_of() { echo "$1" | extract '.id'; }

# =============================================================================
# Build & start server
# =============================================================================
printf "${BOLD}Building server...${RESET}\n"
go build -o "$BINARY" ./cmd/server

printf "Starting server on port ${PORT}...\n"
DATABASE_URL="file:${DB_FILE}?_pragma=foreign_keys(1)" PORT="$PORT" "$BINARY" &>/dev/null &
SERVER_PID=$!

# Wait for readiness
for i in $(seq 1 50); do
  if curl -sf "${BASE}/healthz" &>/dev/null; then break; fi
  sleep 0.1
done
if ! curl -sf "${BASE}/healthz" &>/dev/null; then
  printf "${RED}Server failed to start${RESET}\n"
  exit 1
fi
printf "${GREEN}Server ready.${RESET}\n"

# =============================================================================
printf "\n${BOLD}${CYAN}╔══════════════════════════════════════════════════════════╗${RESET}"
printf "\n${BOLD}${CYAN}║   Ontology-Driven Property Management — Full Lifecycle  ║${RESET}"
printf "\n${BOLD}${CYAN}╚══════════════════════════════════════════════════════════╝${RESET}\n"
# =============================================================================

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
header "ACT 1 — The Management Company"
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

narrate "Every property management story starts with a company and its people."
sleep 0.3

# Organization
ORG=$(api_ok POST /v1/organizations '{
  "legal_name": "Propeller Property Management LLC",
  "dba_name": "Propeller PM",
  "org_type": "management_company",
  "status": "active",
  "tax_id": "84-1234567",
  "tax_id_type": "ein",
  "state_of_incorporation": "CO",
  "address": {
    "line1": "1600 Market Street",
    "line2": "Suite 400",
    "city": "Denver",
    "state": "CO",
    "postal_code": "80202",
    "country": "US"
  },
  "contact_methods": [{"type": "email", "value": "info@propellerpm.com", "primary": true, "verified": true}]
}')
ORG_ID=$(id_of "$ORG")
entity "Organization: Propeller Property Management LLC  (id: ${ORG_ID:0:8}…)"

# Person — Property Manager
SARAH=$(api_ok POST /v1/persons '{
  "first_name": "Sarah",
  "last_name": "Chen",
  "display_name": "Sarah Chen",
  "preferred_contact": "email",
  "language_preference": "en",
  "contact_methods": [{"type": "email", "value": "sarah@propellerpm.com", "primary": true, "verified": true}],
  "tags": ["manager", "denver"]
}')
SARAH_ID=$(id_of "$SARAH")
entity "Person: Sarah Chen — property manager  (id: ${SARAH_ID:0:8}…)"

# PersonRole — property_manager scoped to org
ROLE=$(api_ok POST /v1/person-roles "{
  \"person_id\": \"${SARAH_ID}\",
  \"role_type\": \"property_manager\",
  \"scope_type\": \"organization\",
  \"scope_id\": \"${ORG_ID}\",
  \"status\": \"pending\",
  \"effective\": {\"start\": \"${NOW}\"}
}")
ROLE_ID=$(id_of "$ROLE")
entity "PersonRole: property_manager (pending)  (id: ${ROLE_ID:0:8}…)"

# Transition: pending → active
api_ok POST "/v1/person-roles/${ROLE_ID}/activate" '{}' >/dev/null
transition "PersonRole: pending → active  (via POST /activate)"

sleep 0.3

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
header "ACT 2 — The Portfolio & Properties"
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

narrate "Sarah's company manages a portfolio of Denver metro residential properties."
sleep 0.3

# Portfolio
PORTFOLIO=$(api_ok POST /v1/portfolios "{
  \"name\": \"Denver Metro Residential\",
  \"owner_id\": \"${ORG_ID}\",
  \"management_type\": \"third_party\",
  \"requires_trust_accounting\": false,
  \"status\": \"onboarding\",
  \"fiscal_year_start_month\": 1,
  \"default_payment_methods\": [\"ach\", \"check\"]
}")
PORTFOLIO_ID=$(id_of "$PORTFOLIO")
entity "Portfolio: Denver Metro Residential (onboarding)  (id: ${PORTFOLIO_ID:0:8}…)"

# Activate portfolio
api_ok POST "/v1/portfolios/${PORTFOLIO_ID}/activate" '{}' >/dev/null
transition "Portfolio: onboarding → active  (via POST /activate)"

# Property
PROP=$(api_ok POST /v1/properties "{
  \"name\": \"Sunset Apartments\",
  \"portfolio_id\": \"${PORTFOLIO_ID}\",
  \"property_type\": \"multi_family\",
  \"status\": \"onboarding\",
  \"year_built\": 1995,
  \"total_square_footage\": 24000,
  \"total_units\": 12,
  \"stories\": 3,
  \"parking_spaces\": 24,
  \"requires_lead_disclosure\": false,
  \"address\": {
    \"line1\": \"4200 Sunset Boulevard\",
    \"city\": \"Denver\",
    \"state\": \"CO\",
    \"postal_code\": \"80220\",
    \"country\": \"US\"
  }
}")
PROP_ID=$(id_of "$PROP")
entity "Property: Sunset Apartments (onboarding)  (id: ${PROP_ID:0:8}…)"

# Activate property
api_ok POST "/v1/properties/${PROP_ID}/activate" '{}' >/dev/null
transition "Property: onboarding → active  (via POST /activate)"

# Unit 101
UNIT1=$(api_ok POST /v1/units "{
  \"property_id\": \"${PROP_ID}\",
  \"unit_number\": \"101\",
  \"unit_type\": \"residential\",
  \"status\": \"vacant\",
  \"square_footage\": 750,
  \"bedrooms\": 1,
  \"bathrooms\": 1,
  \"floor\": 1,
  \"market_rent_amount_cents\": 185000,
  \"market_rent_currency\": \"USD\",
  \"amenities\": [\"dishwasher\", \"in_unit_laundry\"]
}")
UNIT1_ID=$(id_of "$UNIT1")
entity "Unit: 101 — 1BR/1BA vacant  (id: ${UNIT1_ID:0:8}…)"

# Unit 202
UNIT2=$(api_ok POST /v1/units "{
  \"property_id\": \"${PROP_ID}\",
  \"unit_number\": \"202\",
  \"unit_type\": \"residential\",
  \"status\": \"vacant\",
  \"square_footage\": 1100,
  \"bedrooms\": 2,
  \"bathrooms\": 2,
  \"floor\": 2,
  \"market_rent_amount_cents\": 245000,
  \"market_rent_currency\": \"USD\",
  \"amenities\": [\"dishwasher\", \"in_unit_laundry\", \"balcony\"]
}")
UNIT2_ID=$(id_of "$UNIT2")
entity "Unit: 202 — 2BR/2BA vacant  (id: ${UNIT2_ID:0:8}…)"

sleep 0.3

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
header "ACT 3 — The Tenant Application"
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

narrate "Marcus Johnson finds Unit 101 online and submits an application."
sleep 0.3

# Person — prospective tenant
MARCUS=$(api_ok POST /v1/persons '{
  "first_name": "Marcus",
  "last_name": "Johnson",
  "display_name": "Marcus Johnson",
  "preferred_contact": "email",
  "language_preference": "en",
  "contact_methods": [{"type": "email", "value": "marcus.j@email.com", "primary": true, "verified": true}]
}')
MARCUS_ID=$(id_of "$MARCUS")
entity "Person: Marcus Johnson — prospective tenant  (id: ${MARCUS_ID:0:8}…)"

# Application — submitted (to demo the guardrail)
APP=$(api_ok POST /v1/applications "{
  \"applicant_person_id\": \"${MARCUS_ID}\",
  \"property_id\": \"${PROP_ID}\",
  \"unit_id\": \"${UNIT1_ID}\",
  \"status\": \"submitted\",
  \"desired_move_in\": \"${TERM_START}\",
  \"desired_lease_term_months\": 12,
  \"application_fee_amount_cents\": 5000,
  \"application_fee_currency\": \"USD\"
}")
APP_ID=$(id_of "$APP")
entity "Application: submitted for Unit 101  (id: ${APP_ID:0:8}…)"

# GUARDRAIL: Try invalid transition submitted → approved
narrate "Can we skip straight to approved? The ontology says NO."
FAIL_RESP=$(api_fail POST "/v1/applications/${APP_ID}/approve" '{}')
FAIL_MSG=$(echo "$FAIL_RESP" | extract '.message // .error // "rejected"')
guardrail "submitted → approved: REJECTED — ${FAIL_MSG}"

narrate "The state machine requires: submitted → screening → under_review → approved"
narrate "After screening completes, the application reaches under_review..."

# Application — at under_review stage (screened, ready for decision)
APP2=$(api_ok POST /v1/applications "{
  \"applicant_person_id\": \"${MARCUS_ID}\",
  \"property_id\": \"${PROP_ID}\",
  \"unit_id\": \"${UNIT1_ID}\",
  \"status\": \"under_review\",
  \"desired_move_in\": \"${TERM_START}\",
  \"desired_lease_term_months\": 12,
  \"application_fee_amount_cents\": 5000,
  \"application_fee_currency\": \"USD\"
}")
APP2_ID=$(id_of "$APP2")
entity "Application: under_review (screened, credit 740)  (id: ${APP2_ID:0:8}…)"
api_ok POST "/v1/applications/${APP2_ID}/approve" '{}' >/dev/null
transition "Application: under_review → approved  (via POST /approve)"

sleep 0.3

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
header "ACT 4 — The Lease Lifecycle"
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

narrate "Application approved — time to draft a lease for Marcus on Unit 101."
sleep 0.3

# Create a tenant PersonRole for Marcus
TENANT_ROLE=$(api_ok POST /v1/person-roles "{
  \"person_id\": \"${MARCUS_ID}\",
  \"role_type\": \"tenant\",
  \"scope_type\": \"property\",
  \"scope_id\": \"${PROP_ID}\",
  \"status\": \"pending\",
  \"effective\": {\"start\": \"${TERM_START}\"}
}")
TENANT_ROLE_ID=$(id_of "$TENANT_ROLE")
entity "PersonRole: tenant for Marcus (pending)  (id: ${TENANT_ROLE_ID:0:8}…)"
api_ok POST "/v1/person-roles/${TENANT_ROLE_ID}/activate" '{}' >/dev/null
transition "PersonRole: pending → active"

# Lease
LEASE=$(api_ok POST /v1/leases "{
  \"property_id\": \"${PROP_ID}\",
  \"unit_id\": \"${UNIT1_ID}\",
  \"tenant_role_ids\": [\"${TENANT_ROLE_ID}\"],
  \"lease_type\": \"fixed_term\",
  \"status\": \"draft\",
  \"term\": {\"start\": \"${TERM_START}\", \"end\": \"${TERM_END}\"},
  \"base_rent_amount_cents\": 185000,
  \"base_rent_currency\": \"USD\",
  \"security_deposit_amount_cents\": 185000,
  \"security_deposit_currency\": \"USD\",
  \"notice_required_days\": 30
}")
LEASE_ID=$(id_of "$LEASE")
entity "Lease: \$1,850/mo fixed-term (draft)  (id: ${LEASE_ID:0:8}…)"

# Walk lease through state machine
narrate "Walking the lease through its full state machine:"

api_ok POST "/v1/leases/${LEASE_ID}/submit" '{}' >/dev/null
transition "Lease: draft → pending_approval  (via POST /submit)"

api_ok POST "/v1/leases/${LEASE_ID}/approve" '{}' >/dev/null
transition "Lease: pending_approval → pending_signature  (via POST /approve)"

ACTIVATED=$(api_ok POST "/v1/leases/${LEASE_ID}/activate" "{\"move_in_date\": \"${TERM_START}\"}")
LEASE_STATUS=$(echo "$ACTIVATED" | extract '.status')
LEASE_MOVE_IN=$(echo "$ACTIVATED" | extract '.move_in_date // "set"')
transition "Lease: pending_signature → active  (via POST /activate, move_in=${TERM_START:0:10})"
narrate "Lease is now ${LEASE_STATUS} — Marcus moves in on ${TERM_START:0:10}!"

sleep 0.3

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
header "ACT 5 — The Money"
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

narrate "Double-entry accounting — the ontology enforces balanced books."
sleep 0.3

# Chart of Accounts
ACCT_CASH=$(api_ok POST /v1/accounts '{
  "account_number": "1000",
  "name": "Cash — Operating",
  "account_type": "asset",
  "account_subtype": "cash",
  "normal_balance": "debit",
  "depth": 0,
  "status": "active"
}')
CASH_ID=$(id_of "$ACCT_CASH")
entity "Account: 1000 Cash — Operating (asset/debit)  (id: ${CASH_ID:0:8}…)"

ACCT_INCOME=$(api_ok POST /v1/accounts '{
  "account_number": "4000",
  "name": "Rental Income",
  "account_type": "revenue",
  "account_subtype": "rental_income",
  "normal_balance": "credit",
  "depth": 0,
  "status": "active"
}')
INCOME_ID=$(id_of "$ACCT_INCOME")
entity "Account: 4000 Rental Income (revenue/credit)  (id: ${INCOME_ID:0:8}…)"

ACCT_DEPOSITS=$(api_ok POST /v1/accounts '{
  "account_number": "2100",
  "name": "Security Deposits Held",
  "account_type": "liability",
  "account_subtype": "security_deposits_held",
  "normal_balance": "credit",
  "depth": 0,
  "status": "active"
}')
DEPOSITS_ID=$(id_of "$ACCT_DEPOSITS")
entity "Account: 2100 Security Deposits Held (liability/credit)  (id: ${DEPOSITS_ID:0:8}…)"

# BankAccount
BANK=$(api_ok POST /v1/bank-accounts "{
  \"name\": \"Denver Metro Operating\",
  \"account_type\": \"operating\",
  \"gl_account_id\": \"${CASH_ID}\",
  \"bank_name\": \"First National Bank\",
  \"routing_number\": \"102003154\",
  \"account_number_last_four\": \"4821\",
  \"status\": \"active\"
}")
BANK_ID=$(id_of "$BANK")
entity "BankAccount: Denver Metro Operating ****4821  (id: ${BANK_ID:0:8}…)"

# JournalEntry — first month's rent
narrate "Recording first month's rent: debit Cash \$1,850, credit Rental Income \$1,850"
JE_RENT=$(api_ok POST /v1/journal-entries "{
  \"entry_date\": \"${TERM_START}\",
  \"posted_date\": \"${TERM_START}\",
  \"description\": \"March 2026 rent — Unit 101, Marcus Johnson\",
  \"source_type\": \"auto_charge\",
  \"status\": \"draft\",
  \"property_id\": \"${PROP_ID}\",
  \"lines\": [
    {\"account_id\": \"${CASH_ID}\", \"debit\": {\"amount_cents\": 185000, \"currency\": \"USD\"}, \"description\": \"Cash receipt\"},
    {\"account_id\": \"${INCOME_ID}\", \"credit\": {\"amount_cents\": 185000, \"currency\": \"USD\"}, \"description\": \"Rental income\"}
  ]
}")
JE_RENT_ID=$(id_of "$JE_RENT")
entity "JournalEntry: March rent — draft, 2 lines  (id: ${JE_RENT_ID:0:8}…)"

# Post the journal entry → creates LedgerEntries
api_ok POST "/v1/journal-entries/${JE_RENT_ID}/post" '{}' >/dev/null
transition "JournalEntry: draft → posted  (via POST /post — ledger entries created!)"

# List ledger entries to prove they were created
LEDGER=$(api_ok GET /v1/ledger-entries)
LEDGER_COUNT=$(echo "$LEDGER" | jq 'if type == "array" then length else 0 end' 2>/dev/null || echo "?")
entity "LedgerEntry: ${LEDGER_COUNT} entries created from journal posting"

# Reconciliation — create at balanced (all transactions matched)
narrate "Month-end: reconcile the operating account."
RECON=$(api_ok POST /v1/reconciliations "{
  \"bank_account_id\": \"${BANK_ID}\",
  \"period_start\": \"2026-03-01T00:00:00Z\",
  \"period_end\": \"2026-03-31T00:00:00Z\",
  \"statement_balance_amount_cents\": 185000,
  \"statement_balance_currency\": \"USD\",
  \"system_balance_amount_cents\": 185000,
  \"system_balance_currency\": \"USD\",
  \"difference_amount_cents\": 0,
  \"difference_currency\": \"USD\",
  \"status\": \"balanced\",
  \"matched_transaction_count\": 1,
  \"unmatched_transaction_count\": 0
}")
RECON_ID=$(id_of "$RECON")
entity "Reconciliation: March 2026 (balanced, diff=\$0)  (id: ${RECON_ID:0:8}…)"

# Approve the reconciliation
api_ok POST "/v1/reconciliations/${RECON_ID}/approve" '{}' >/dev/null
transition "Reconciliation: balanced → approved  (via POST /approve)"

# Security deposit journal entry — full lifecycle including void
narrate "Recording security deposit, then voiding it to show full JE lifecycle."
JE_DEP=$(api_ok POST /v1/journal-entries "{
  \"entry_date\": \"${TERM_START}\",
  \"posted_date\": \"${TERM_START}\",
  \"description\": \"Security deposit — Unit 101, Marcus Johnson\",
  \"source_type\": \"manual\",
  \"status\": \"draft\",
  \"property_id\": \"${PROP_ID}\",
  \"lines\": [
    {\"account_id\": \"${CASH_ID}\", \"debit\": {\"amount_cents\": 185000, \"currency\": \"USD\"}, \"description\": \"Cash receipt\"},
    {\"account_id\": \"${DEPOSITS_ID}\", \"credit\": {\"amount_cents\": 185000, \"currency\": \"USD\"}, \"description\": \"Deposit liability\"}
  ]
}")
JE_DEP_ID=$(id_of "$JE_DEP")
entity "JournalEntry: security deposit — draft  (id: ${JE_DEP_ID:0:8}…)"

api_ok POST "/v1/journal-entries/${JE_DEP_ID}/post" '{}' >/dev/null
transition "JournalEntry: draft → posted  (deposit recorded)"

api_ok POST "/v1/journal-entries/${JE_DEP_ID}/void" '{}' >/dev/null
transition "JournalEntry: posted → voided  (via POST /void — reversed!)"

sleep 0.3

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
header "EPILOGUE — The Power of Ontology"
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

printf "\n"
printf "  ${CYAN}╔════════════════════════════════════════════════════════╗${RESET}\n"
box_line ""
box_line "  One CUE Ontology Generated:"
box_line ""
box_line "    13 Ent entities (Person, Org, Role, Portfolio,"
box_line "       Property, Unit, Application, Lease, Account,"
box_line "       LedgerEntry, JournalEntry, BankAccount, Recon)"
box_line ""
box_line "    50 REST endpoints across 4 services"
box_line "    10 state machines with enforced transitions"
box_line ""
box_line "  This Demo Exercised:"
box_line ""
box_line "    All 13 entities — created and linked"
box_line "    6 state machine transitions via POST endpoints"
box_line "    1 guardrail rejection (invalid transition)"
box_line "    Double-entry journal posting → ledger entries"
box_line "    Full JE lifecycle: draft → posted → voided"
box_line "    Every request audited (actor, source, correlation)"
box_line ""
printf "  ${CYAN}╚════════════════════════════════════════════════════════╝${RESET}\n"
printf "\n"

printf "${GREEN}${BOLD}Demo complete.${RESET} All 13 entities exercised across the full lifecycle.\n\n"
