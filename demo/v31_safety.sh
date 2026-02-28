#!/usr/bin/env bash
# =============================================================================
# V3.1 Ontological Safety Net — Demo
#
# A narrated walkthrough showing how the CUE ontology acts as a shared
# vocabulary and constraint system. The ontology catches drift at BUILD TIME
# across commands, events, API contracts, and permission policies.
#
# No server needed. This demo runs entirely at the CUE/Go toolchain level.
#
# Prerequisites: cue, go, and jq must be on PATH.
# =============================================================================
set -euo pipefail

# ---------------------------------------------------------------------------
# Find project root
# ---------------------------------------------------------------------------
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$PROJECT_ROOT"

# ---------------------------------------------------------------------------
# Colors & formatting (same style as signals.sh)
# ---------------------------------------------------------------------------
BOLD=$'\033[1m'
CYAN=$'\033[1;36m'
GREEN=$'\033[0;32m'
YELLOW=$'\033[1;33m'
RED=$'\033[0;31m'
DIM=$'\033[2m'
MAGENTA=$'\033[1;35m'
WHITE=$'\033[1;37m'
RESET=$'\033[0m'

header()   { printf "\n${CYAN}━━━ %s ━━━${RESET}\n\n" "$1"; }
narrate()  { printf "${DIM}    %s${RESET}\n" "$1"; }
positive() { printf "  ${GREEN}●${RESET} %s\n" "$1"; }
negative() { printf "  ${RED}●${RESET} %s\n" "$1"; }
neutral()  { printf "  ${YELLOW}●${RESET} %s\n" "$1"; }
info()     { printf "  ${DIM}○${RESET} %s\n" "$1"; }
label()    { printf "  ${WHITE}%-24s${RESET} %s\n" "$1" "$2"; }
cmd()      { printf "  ${MAGENTA}\$ %s${RESET}\n" "$1"; }
file_show(){ printf "  ${DIM}%s${RESET}\n" "$1"; }

pause() {
  printf "\n${DIM}    ─── press enter to continue ───${RESET}"
  read -r
  printf "\n"
}

# ---------------------------------------------------------------------------
# Cleanup trap — remove any temp files if script is interrupted
# ---------------------------------------------------------------------------
DEMO_PID=""
DEMO_DB=""
DEMO_LOG=""
cleanup() {
  rm -f "${PROJECT_ROOT}/.demo_tmp_close.cue" "${PROJECT_ROOT}/.demo_tmp_enum.cue"
  if [ -n "$DEMO_PID" ] && kill -0 "$DEMO_PID" 2>/dev/null; then
    kill "$DEMO_PID" 2>/dev/null
    wait "$DEMO_PID" 2>/dev/null || true
  fi
  [ -n "$DEMO_DB" ] && rm -f "${DEMO_DB}"*
  [ -n "$DEMO_LOG" ] && rm -f "${DEMO_LOG}"
  [ -f "${PROJECT_ROOT}/tmp_demo_server" ] && rm -f "${PROJECT_ROOT}/tmp_demo_server"
}
trap cleanup EXIT

# ---------------------------------------------------------------------------
# Preflight
# ---------------------------------------------------------------------------
for tool in cue go jq; do
  if ! command -v "$tool" &>/dev/null; then
    printf "${RED}ERROR: %s not found on PATH${RESET}\n" "$tool"
    exit 1
  fi
done

# =============================================================================
printf "\n${BOLD}${CYAN}╔══════════════════════════════════════════════════════════════╗${RESET}"
printf "\n${BOLD}${CYAN}║       V3.1 Ontological Safety Net — Build-Time Guarantees    ║${RESET}"
printf "\n${BOLD}${CYAN}║                                                              ║${RESET}"
printf "\n${BOLD}${CYAN}║  How one CUE ontology prevents drift across 6 packages.      ║${RESET}"
printf "\n${BOLD}${CYAN}╚══════════════════════════════════════════════════════════════╝${RESET}\n"
printf "\n${DIM}    No server needed — this demo runs at the CUE/Go toolchain level.${RESET}\n"
# =============================================================================

pause

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
header "THE ARCHITECTURE — Shared Vocabulary, Not Contract Generation"
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

narrate "V3.1 reframes the ontology as a shared vocabulary and constraint system."
narrate "Six CUE packages all reference ontology types via cross-package imports."
echo ""

printf "  ${WHITE}ontology/${RESET}            ${DIM}18 entities, 13 state machines, enums, constraints${RESET}\n"
printf "  ${DIM}    │${RESET}\n"
printf "  ${DIM}    ├──${RESET} ${WHITE}commands/${RESET}       ${DIM}10 CQRS commands (lease, property, accounting, application)${RESET}\n"
printf "  ${DIM}    ├──${RESET} ${WHITE}events/${RESET}         ${DIM}10 domain events (lease, property, accounting, jurisdiction)${RESET}\n"
printf "  ${DIM}    ├──${RESET} ${WHITE}api/v1/${RESET}         ${DIM}API response contracts (anti-corruption layer)${RESET}\n"
printf "  ${DIM}    ├──${RESET} ${WHITE}policies/${RESET}       ${DIM}7 permission groups, field-level visibility${RESET}\n"
printf "  ${DIM}    └──${RESET} ${WHITE}codegen/${RESET}        ${DIM}generator configs, drift check rules${RESET}\n"
echo ""

narrate "Each package imports ontology types:"
echo ""
file_show "    import \"github.com/matthewbaird/ontology/ontology:propeller\""
echo ""
file_show "    #MoveInTenant: {"
file_show "        lease_id:         string"
file_show "        security_deposit: propeller.#PositiveMoney   // ← ontology type"
file_show "        lease_type:       propeller.#LeaseType       // ← ontology enum"
file_show "        ..."
file_show "    }"
echo ""
narrate "Change the ontology → every downstream package must agree or break."

pause

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
header "DEMO 1 — close() Prevents Unknown Fields"
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

narrate "Every entity type is wrapped in close() — no surprise fields allowed."
narrate "Let's try adding a field that doesn't belong."
echo ""

narrate "Current #Person definition (ontology/person.cue):"
echo ""
file_show "    #Person: close({"
file_show "        #StatefulEntity"
file_show "        first_name:  string & strings.MinRunes(1)"
file_show "        last_name:   string & strings.MinRunes(1)"
file_show "        ...28 more fields..."
file_show "    })"
echo ""

narrate "Imagine a developer creates a Person record with an extra field:"
echo ""
file_show "    bad_person: propeller.#Person & {"
file_show "        first_name:        \"Test\""
file_show "        last_name:         \"User\""
file_show "        display_name:      \"Test User\""
file_show "        ...required fields..."
file_show "        ${RED}favorite_color:    \"blue\"   // <-- doesn't belong!${RESET}"
file_show "    }"

# Create a temporary standalone CUE file that validates against the ontology
DEMO_TMP="${PROJECT_ROOT}/.demo_tmp_close.cue"
cat > "$DEMO_TMP" << 'DEMOCUE'
package demo

import "github.com/matthewbaird/ontology/ontology:propeller"

bad_person: propeller.#Person & {
	first_name:          "Test"
	last_name:           "User"
	display_name:        "Test User"
	preferred_contact:   "email"
	language_preference: "en"
	favorite_color:      "blue"
	contact_methods: [{type: "email", value: "t@t.com", primary: true, verified: true}]
}
DEMOCUE

echo ""
cmd "cue vet close_test.cue"
echo ""

VET_OUTPUT=$(cue vet "$DEMO_TMP" 2>&1 || true)
rm -f "$DEMO_TMP"

if echo "$VET_OUTPUT" | grep -q "not allowed"; then
  negative "REJECTED: $(echo "$VET_OUTPUT" | grep 'not allowed' | head -1 | sed 's|.*demo_tmp_||')"
  echo ""
  narrate "${GREEN}close() caught the unknown field at build time.${RESET}"
  narrate "No runtime surprise — the CUE compiler says no."
else
  negative "Expected rejection (check close() wrapping)"
fi

pause

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
header "DEMO 2 — Cross-Package Type Safety"
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

narrate "Commands import ontology types. What happens when they disagree?"
echo ""

narrate "The ontology defines 13 lease types:"
echo ""
file_show "    #LeaseType: \"fixed_term\" | \"month_to_month\" |"
file_show "        \"commercial_nnn\" | \"commercial_nn\" | \"commercial_n\" |"
file_show "        \"commercial_gross\" | \"commercial_modified_gross\" |"
file_show "        \"affordable\" | \"section_8\" | \"student\" |"
file_show "        \"ground_lease\" | \"short_term\" | \"membership\""
echo ""

narrate "A command uses this type to validate its lease_type field."
narrate "Let's say a developer tries to use a lease type that doesn't exist..."
echo ""

# Create a standalone CUE file that tries to use an invalid lease type
DEMO_TMP="${PROJECT_ROOT}/.demo_tmp_enum.cue"
cat > "$DEMO_TMP" << 'DEMOCUE'
package demo

import "github.com/matthewbaird/ontology/ontology:propeller"

bad_command: {
	lease_id:       string
	bad_lease_type: propeller.#LeaseType & "triple_super_net"
}
DEMOCUE

narrate "Adding a command with:  ${RED}bad_lease_type: propeller.#LeaseType & \"triple_super_net\"${RESET}"
echo ""
cmd "cue vet bad_command.cue"
echo ""

VET_OUTPUT=$(cue vet "$DEMO_TMP" 2>&1 || true)
rm -f "$DEMO_TMP"

if echo "$VET_OUTPUT" | grep -q "conflicting values\|empty disjunction"; then
  negative "REJECTED by CUE compiler"
  echo ""
  # Show the summary and first two conflicting values for clarity.
  echo "$VET_OUTPUT" | head -1 | while IFS= read -r line; do
    file_show "    $line"
  done
  echo "$VET_OUTPUT" | grep "conflicting values" | head -2 | while IFS= read -r line; do
    # Strip file path noise, show just the conflict.
    CLEAN=$(echo "$line" | sed 's|:$||')
    file_show "    $CLEAN"
  done
  file_show "    ... (13 enum values tried, none matched)"
  echo ""
  narrate "${GREEN}CUE tried \"triple_super_net\" against all 13 values in #LeaseType.${RESET}"
  narrate "${GREEN}None matched — the compiler rejected it at build time.${RESET}"
else
  negative "Expected validation error"
fi

pause

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
header "DEMO 3 — Unified Drift Check (All 6 Packages)"
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

narrate "The driftcheck tool validates ALL packages in one pass."
narrate "Ontology, commands, events, API contracts, policies, codegen configs."
echo ""

# Count files per package
ONTOLOGY_COUNT=$(find ontology -name '*.cue' | wc -l | tr -d ' ')
COMMANDS_COUNT=$(find commands -name '*.cue' | wc -l | tr -d ' ')
EVENTS_COUNT=$(find events -name '*.cue' | wc -l | tr -d ' ')
API_COUNT=$(find api -name '*.cue' | wc -l | tr -d ' ')
POLICIES_COUNT=$(find policies -name '*.cue' | wc -l | tr -d ' ')
CODEGEN_COUNT=$(find codegen -name '*.cue' | wc -l | tr -d ' ')
TOTAL_CUE=$((ONTOLOGY_COUNT + COMMANDS_COUNT + EVENTS_COUNT + API_COUNT + POLICIES_COUNT + CODEGEN_COUNT))

label "  ontology/"     "${ONTOLOGY_COUNT} CUE files — entities, enums, state machines, constraints"
label "  commands/"     "${COMMANDS_COUNT} CUE files — CQRS commands importing ontology types"
label "  events/"       "${EVENTS_COUNT} CUE files — domain events with typed payloads"
label "  api/v1/"       "${API_COUNT} CUE files — API response contracts"
label "  policies/"     "${POLICIES_COUNT} CUE files — permission groups, field visibility"
label "  codegen/"      "${CODEGEN_COUNT} CUE files — generator configurations"
echo ""

cmd "make driftcheck"
echo ""

make driftcheck 2>&1 | while IFS= read -r line; do
  if echo "$line" | grep -q "All packages validate"; then
    positive "$line"
  elif echo "$line" | grep -q "OK"; then
    positive "$line"
  elif echo "$line" | grep -q "WARNING"; then
    neutral "$line"
  else
    info "$line"
  fi
done

echo ""
narrate "${TOTAL_CUE} CUE files across 6 packages, all referencing the same ontology."
narrate "One command validates the entire dependency graph."

pause

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
header "DEMO 4 — State Machine Test Generation"
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

narrate "The ontology defines 13 state machines. The test generator reads them"
narrate "and produces every valid AND invalid transition as test cases."
echo ""

cmd "go run ./cmd/testgen"
echo ""
TESTGEN_OUTPUT=$(go run ./cmd/testgen 2>&1)
positive "$TESTGEN_OUTPUT"

echo ""
narrate "Let's look at what it generated:"
echo ""

# Count by entity
TOTAL=$(grep -c 'Entity:' gen/tests/state_machine_tests.go || echo 0)
POSITIVE=$(grep -c '"success"' gen/tests/state_machine_tests.go || echo 0)
NEGATIVE=$(grep -c '"error"' gen/tests/state_machine_tests.go || echo 0)
ENTITIES=$(grep -o 'Entity: "[^"]*"' gen/tests/state_machine_tests.go | sort -u | sed 's/Entity: "//;s/"//' || true)

printf "  ${WHITE}%-22s %8s %8s %8s${RESET}\n" "Entity" "Valid" "Invalid" "Total"
printf "  ${DIM}%-22s %8s %8s %8s${RESET}\n" "──────────────────────" "────────" "────────" "────────"

for entity in $ENTITIES; do
  ent_total=$(grep "Entity: \"${entity}\"" gen/tests/state_machine_tests.go | wc -l | tr -d ' ')
  ent_pos=$(grep "Entity: \"${entity}\"" gen/tests/state_machine_tests.go | grep '"success"' | wc -l | tr -d ' ')
  ent_neg=$(grep "Entity: \"${entity}\"" gen/tests/state_machine_tests.go | grep '"error"' | wc -l | tr -d ' ')
  printf "  %-22s %8s %8s %8s\n" "$entity" "$ent_pos" "$ent_neg" "$ent_total"
done

printf "  ${DIM}%-22s %8s %8s %8s${RESET}\n" "──────────────────────" "────────" "────────" "────────"
printf "  ${BOLD}%-22s %8s %8s %8s${RESET}\n" "TOTAL" "$POSITIVE" "$NEGATIVE" "$TOTAL"

echo ""
narrate "Every row is auto-derived from ontology/state_machines.cue."
narrate "Add a state → new positive AND negative tests appear automatically."

echo ""
narrate "Sample positive test (valid transition):"
file_show "    {Entity: \"Lease\", From: \"draft\", To: \"pending_approval\", Expected: \"success\"}"
echo ""
narrate "Sample negative test (invalid transition):"
file_show "    {Entity: \"Lease\", From: \"draft\", To: \"active\", Expected: \"error\"}"
narrate "    (Can't skip from draft to active — must go through approval first)"

pause

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
header "DEMO 5 — Full Generation Pipeline"
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

narrate "One command regenerates EVERYTHING from the ontology."
narrate "11 generators. One source of truth."
echo ""

cmd "make generate"
echo ""
GENERATE_OUTPUT=$(make generate 2>&1)

# Parse and display each generator's output
echo "$GENERATE_OUTPUT" | while IFS= read -r line; do
  if echo "$line" | grep -q "^entgen:"; then
    positive "$line"
  elif echo "$line" | grep -q "^handlergen:"; then
    positive "$line"
  elif echo "$line" | grep -q "^eventgen:.*generated"; then
    positive "$line"
  elif echo "$line" | grep -q "^authzgen:"; then
    positive "$line"
  elif echo "$line" | grep -q "^agentgen:"; then
    positive "$line"
  elif echo "$line" | grep -q "^openapigen:"; then
    positive "$line"
  elif echo "$line" | grep -q "^uigen:"; then
    positive "$line"
  elif echo "$line" | grep -q "^uirender:"; then
    positive "$line"
  elif echo "$line" | grep -q "^testgen:"; then
    positive "$line"
  elif echo "$line" | grep -qE "^Generated|^go "; then
    info "$line"
  fi
done

echo ""
narrate "From a single CUE ontology:"
echo ""
label "  Ent schemas"         "18 database entities with constraints"
label "  HTTP handlers"       "6 service files + routes"
label "  Proto files"         "6 Connect-RPC service definitions"
label "  Event types"         "59 domain event constants"
label "  OPA policies"        "Authorization scaffolds for 17 entities"
label "  Agent context"       "ONTOLOGY.md, TOOLS.md, SIGNALS.md"
label "  OpenAPI spec"        "295KB, 76 paths, 47 schemas"
label "  UI schemas"          "18 JSON schemas for form generation"
label "  UI components"       "159 Svelte files (forms, lists, details)"
label "  Test cases"          "314 state machine transition tests"

pause

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
header "DEMO 6 — Runtime Jurisdiction Enforcement"
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

narrate "The ontology doesn't just validate at build time."
narrate "Jurisdiction rules are loaded from the database and enforced at runtime."
narrate "Every lease mutation passes through an Ent hook that checks compliance."
echo ""
narrate "Let's see what happens when a lease violates California law."
echo ""

# Build the server binary first (faster than go run).
info "Building server..."
go build -o "${PROJECT_ROOT}/tmp_demo_server" ./cmd/server 2>/dev/null

# Start server with --demo (seeds 4 jurisdictions + 11 rules) on a temp database.
DEMO_DB=$(mktemp /tmp/ontology_demo_XXXXXX)
DEMO_LOG=$(mktemp /tmp/ontology_demo_log_XXXXXX)
DEMO_PORT=18088

DATABASE_URL="file:${DEMO_DB}?_pragma=foreign_keys(1)" PORT=$DEMO_PORT \
  "${PROJECT_ROOT}/tmp_demo_server" --demo >"$DEMO_LOG" 2>&1 &
DEMO_PID=$!

# Wait for server to be ready.
DEMO_READY=false
for i in $(seq 1 30); do
  if curl -sf "http://localhost:${DEMO_PORT}/healthz" >/dev/null 2>&1; then
    DEMO_READY=true
    break
  fi
  sleep 0.5
done

if [ "$DEMO_READY" = false ]; then
  neutral "Server failed to start — skipping runtime demo"
  neutral "(This demo requires atlas CLI for migrations)"
  kill "$DEMO_PID" 2>/dev/null; wait "$DEMO_PID" 2>/dev/null || true
  DEMO_PID=""
  pause
else

BASE="http://localhost:${DEMO_PORT}/v1"
HDR_JSON="Content-Type: application/json"
HDR_ACTOR="X-Actor: demo"
HDR_SRC="X-Source: system"
NOW=$(date -u +%Y-%m-%dT%H:%M:%SZ)

# ── Silent setup: create prerequisite entities ──────────────────────
ORG_ID=$(curl -sf -X POST "$BASE/organizations" \
  -H "$HDR_JSON" -H "$HDR_ACTOR" -H "$HDR_SRC" \
  -d '{"legal_name":"Demo Properties LLC","org_type":"ownership_entity","status":"active"}' | jq -r '.id')

PORT_ID=$(curl -sf -X POST "$BASE/portfolios" \
  -H "$HDR_JSON" -H "$HDR_ACTOR" -H "$HDR_SRC" \
  -d "{\"name\":\"West Coast Portfolio\",\"management_type\":\"self_managed\",\"status\":\"onboarding\",\"owner_id\":\"$ORG_ID\"}" | jq -r '.id')

PROP_ID=$(curl -sf -X POST "$BASE/properties" \
  -H "$HDR_JSON" -H "$HDR_ACTOR" -H "$HDR_SRC" \
  -d "{\"name\":\"Sunset Apartments\",\"property_type\":\"multi_family\",\"status\":\"onboarding\",\"year_built\":1985,\"total_square_footage\":12000,\"total_spaces\":12,\"portfolio_id\":\"$PORT_ID\",\"address\":{\"street\":\"1234 Sunset Blvd\",\"city\":\"Santa Monica\",\"state\":\"CA\",\"postal_code\":\"90401\",\"country\":\"US\"}}" | jq -r '.id')

# Link property to California jurisdiction (seeded by --demo).
CA_ID=$(curl -sf "$BASE/jurisdictions" | jq -r '[.[] | select(.name == "California")][0].id')

curl -sf -X POST "$BASE/property-jurisdictions" \
  -H "$HDR_JSON" -H "$HDR_ACTOR" -H "$HDR_SRC" \
  -d "{\"property_id\":\"$PROP_ID\",\"jurisdiction_id\":\"$CA_ID\",\"effective_date\":\"$NOW\",\"lookup_source\":\"manual\",\"verified\":true}" >/dev/null

# ── Show the jurisdiction hierarchy ─────────────────────────────────
positive "Server started with seeded jurisdiction hierarchy"
echo ""

narrate "4 jurisdictions seeded from real statutes (Federal → State → County → City):"
echo ""

# Show hierarchy with indentation.
curl -sf "$BASE/jurisdictions" | jq -r '
  .[] | if .jurisdiction_type == "federal" then "      FEDERAL:  \(.name)"
       elif .jurisdiction_type == "state"  then "        STATE:  \(.name)"
       elif .jurisdiction_type == "county" then "       COUNTY:  \(.name)"
       elif .jurisdiction_type == "city"   then "         CITY:  \(.name)"
       else "                \(.name)" end' | sort

echo ""

# ── Show California's security deposit rule ─────────────────────────
narrate "California's security deposit limit (loaded from the database):"
echo ""

RULE_JSON=$(curl -sf "$BASE/jurisdiction-rules" | jq '[.[] | select(.rule_type == "security_deposit_limit")][0]')
RULE_STATUTE=$(echo "$RULE_JSON" | jq -r '.statute_reference // "n/a"')
RULE_MAX=$(echo "$RULE_JSON" | jq -r '.rule_definition.max_months // "n/a"')

label "    Rule type"    "security_deposit_limit"
label "    Statute"      "$RULE_STATUTE"
label "    Max deposit"  "${RULE_MAX} month(s) of rent"

echo ""

pause

# ── Violation: excessive security deposit ───────────────────────────
narrate "Scenario: Create a lease for Sunset Apartments in Santa Monica."
narrate "Monthly rent: \$2,000.  Security deposit: \$6,000 (3× rent)."
narrate "California law (AB 12) caps deposits at 1 month's rent."
echo ""

cmd "curl -X POST /v1/leases ... security_deposit: 600000, rent: 200000"
echo ""

VIOLATION_RESULT=$(curl -s -X POST "$BASE/leases" \
  -H "$HDR_JSON" -H "$HDR_ACTOR" -H "$HDR_SRC" \
  -d "{
    \"property_id\":\"$PROP_ID\",
    \"tenant_role_ids\":[],
    \"lease_type\":\"fixed_term\",
    \"status\":\"draft\",
    \"liability_type\":\"individual\",
    \"term\":{\"start\":\"2025-08-01T00:00:00Z\",\"end\":\"2026-07-31T00:00:00Z\"},
    \"base_rent_amount_cents\":200000,
    \"security_deposit_amount_cents\":600000,
    \"notice_required_days\":30,
    \"is_sublease\":false,
    \"sublease_billing\":\"direct_to_landlord\"
  }")

ERROR_CODE=$(echo "$VIOLATION_RESULT" | jq -r '.code // empty')
ERROR_MSG=$(echo "$VIOLATION_RESULT" | jq -r '.error // empty')

if [ "$ERROR_CODE" = "JURISDICTION_VIOLATION" ]; then
  negative "REJECTED: ${ERROR_CODE}"
  echo ""
  file_show "    $ERROR_MSG"
  echo ""
  narrate "${GREEN}The Ent mutation hook loaded California's security deposit rule${RESET}"
  narrate "${GREEN}from the database and blocked the mutation before it reached disk.${RESET}"
else
  neutral "Response: $(echo "$VIOLATION_RESULT" | jq -c .)"
fi

echo ""

# ── Compliant lease creation ────────────────────────────────────────
narrate "Now with a compliant deposit: \$2,000 (1 month — within California's limit)."
echo ""

cmd "curl -X POST /v1/leases ... security_deposit: 200000, rent: 200000"
echo ""

GOOD_RESULT=$(curl -s -X POST "$BASE/leases" \
  -H "$HDR_JSON" -H "$HDR_ACTOR" -H "$HDR_SRC" \
  -d "{
    \"property_id\":\"$PROP_ID\",
    \"tenant_role_ids\":[],
    \"lease_type\":\"fixed_term\",
    \"status\":\"draft\",
    \"liability_type\":\"individual\",
    \"term\":{\"start\":\"2025-08-01T00:00:00Z\",\"end\":\"2026-07-31T00:00:00Z\"},
    \"base_rent_amount_cents\":200000,
    \"security_deposit_amount_cents\":200000,
    \"notice_required_days\":30,
    \"is_sublease\":false,
    \"sublease_billing\":\"direct_to_landlord\"
  }")

GOOD_ID=$(echo "$GOOD_RESULT" | jq -r '.id // empty')
GOOD_STATUS=$(echo "$GOOD_RESULT" | jq -r '.status // empty')

if [ -n "$GOOD_ID" ] && [ "$GOOD_STATUS" = "draft" ]; then
  positive "ACCEPTED: Lease ${GOOD_ID:0:8}... created (status: $GOOD_STATUS)"
  echo ""
  narrate "${GREEN}Same property, same jurisdiction, same rules — but compliant.${RESET}"
else
  neutral "Unexpected: $(echo "$GOOD_RESULT" | jq -c .)"
fi

echo ""
narrate "The enforcement stack:"
echo ""
file_show "    API request → Ent mutation → jurisdiction.LeaseHook()"
file_show "        → loadActiveRules(property_id)"
file_show "            → PropertyJurisdiction → Jurisdiction → JurisdictionRule"
file_show "        → checkSecurityDepositLimit(deposit, rent, max_months)"
file_show "        → Violation{statute: \"Cal. Civ. Code §1950.5\"} or proceed"
echo ""
narrate "Rules come from the database, not hardcoded. Add a new jurisdiction,"
narrate "link it to a property, and enforcement happens automatically."

# ── Cleanup server ──────────────────────────────────────────────────
kill "$DEMO_PID" 2>/dev/null
wait "$DEMO_PID" 2>/dev/null || true
DEMO_PID=""

fi  # end DEMO_READY check

pause

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
header "THE BEFORE AND AFTER"
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

printf "  ${RED}${BOLD}WITHOUT ontological safety:${RESET}\n\n"
narrate "  Developer adds \"triple_net\" to the API response."
narrate "  Nobody updates the command. Nobody updates the event."
narrate "  Backend sends an enum the frontend doesn't know about."
narrate "  Mobile app crashes on unknown lease type."
narrate "  Bug found in production 3 weeks later."
echo ""

printf "  ${GREEN}${BOLD}WITH ontological safety:${RESET}\n\n"
narrate "  Developer adds \"triple_net\" to #LeaseType in the ontology."
narrate "  Runs make generate — all 11 generators pick it up."
narrate "  Runs make driftcheck — commands, events, API all validate."
narrate "  CI runs ci-check — confirms generated code is in sync."
narrate "  At runtime, jurisdiction rules enforce compliance on every mutation."
narrate "  ${BOLD}Drift is caught at build time. Violations are caught at runtime.${RESET}"

pause

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

printf "\n"
printf "  ${CYAN}╔══════════════════════════════════════════════════════════════╗${RESET}\n"
printf "  ${CYAN}║${RESET}                                                              ${CYAN}║${RESET}\n"
printf "  ${CYAN}║${RESET}  ${BOLD}V3.1 Safety Net — What We Built:${RESET}                            ${CYAN}║${RESET}\n"
printf "  ${CYAN}║${RESET}                                                              ${CYAN}║${RESET}\n"
printf "  ${CYAN}║${RESET}  ${WHITE}Build-time:${RESET}                                                 ${CYAN}║${RESET}\n"
printf "  ${CYAN}║${RESET}  ${WHITE}  close()${RESET}        Prevents unknown fields on entities        ${CYAN}║${RESET}\n"
printf "  ${CYAN}║${RESET}  ${WHITE}  MinRunes(1)${RESET}    Required strings validated in the schema   ${CYAN}║${RESET}\n"
printf "  ${CYAN}║${RESET}  ${WHITE}  Cross-package${RESET}  Commands/events/API import ontology types  ${CYAN}║${RESET}\n"
printf "  ${CYAN}║${RESET}  ${WHITE}  Enum safety${RESET}    Change an enum, all consumers must agree   ${CYAN}║${RESET}\n"
printf "  ${CYAN}║${RESET}  ${WHITE}  State machines${RESET} Unified map, 314 auto-generated tests      ${CYAN}║${RESET}\n"
printf "  ${CYAN}║${RESET}  ${WHITE}  Drift check${RESET}    One command validates 6 packages            ${CYAN}║${RESET}\n"
printf "  ${CYAN}║${RESET}  ${WHITE}  11 generators${RESET}  All derived from one CUE ontology           ${CYAN}║${RESET}\n"
printf "  ${CYAN}║${RESET}                                                              ${CYAN}║${RESET}\n"
printf "  ${CYAN}║${RESET}  ${WHITE}Runtime:${RESET}                                                    ${CYAN}║${RESET}\n"
printf "  ${CYAN}║${RESET}  ${WHITE}  Jurisdiction${RESET}   DB-driven rules enforce every lease write   ${CYAN}║${RESET}\n"
printf "  ${CYAN}║${RESET}  ${WHITE}  Ent hooks${RESET}      Deposit limits, notice periods, rent caps   ${CYAN}║${RESET}\n"
printf "  ${CYAN}║${RESET}  ${WHITE}  Event bus${RESET}      Commands emit events → signal consumers     ${CYAN}║${RESET}\n"
printf "  ${CYAN}║${RESET}  ${WHITE}  9 commands${RESET}     Transactional multi-entity mutations        ${CYAN}║${RESET}\n"
printf "  ${CYAN}║${RESET}                                                              ${CYAN}║${RESET}\n"
printf "  ${CYAN}║${RESET}  ${DIM}The ontology is the single source of truth.${RESET}                  ${CYAN}║${RESET}\n"
printf "  ${CYAN}║${RESET}  ${DIM}Drift is caught at build time. Violations at runtime.${RESET}        ${CYAN}║${RESET}\n"
printf "  ${CYAN}║${RESET}                                                              ${CYAN}║${RESET}\n"
printf "  ${CYAN}╚══════════════════════════════════════════════════════════════╝${RESET}\n"

printf "\n${GREEN}${BOLD}Demo complete.${RESET}\n\n"
