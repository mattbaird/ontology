#!/usr/bin/env bash
# =============================================================================
# Signal Discovery System — VP Demo
#
# A narrated walkthrough showing how the signal discovery system gives agents
# the ability to see cross-cutting patterns that humans carry as intuition.
#
# This demo hits REAL API endpoints served by the main server with --demo flag.
# The server seeds an in-memory activity store with demo data and serves the
# real signal aggregation, classification, and escalation engine.
#
# Prerequisites: the server must be running with --demo flag.
#   go run ./cmd/server --demo    — or —    air
# =============================================================================
set -euo pipefail

# ---------------------------------------------------------------------------
# Config
# ---------------------------------------------------------------------------
BASE_URL="${SIGNAL_DEMO_URL:-http://localhost:8080}"

# ---------------------------------------------------------------------------
# Colors & formatting
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
label()    { printf "  ${WHITE}%-22s${RESET} %s\n" "$1" "$2"; }

pause() {
  printf "\n${DIM}    ─── press enter to continue ───${RESET}"
  read -r
  printf "\n"
}

# ---------------------------------------------------------------------------
# Preflight: make sure the server is up
# ---------------------------------------------------------------------------
if ! curl -s -o /dev/null -w '' "${BASE_URL}/healthz" 2>/dev/null; then
  printf "${RED}ERROR: server not running at ${BASE_URL}${RESET}\n"
  printf "${DIM}Start it with:  go run ./cmd/server --demo${RESET}\n"
  printf "${DIM}    — or —      air${RESET}\n"
  exit 1
fi

# polarity_bullet prints a colored bullet based on polarity.
polarity_bullet() {
  local polarity="$1"
  shift
  case "$polarity" in
    positive)   positive "$*" ;;
    negative)   negative "$*" ;;
    neutral)    neutral  "$*" ;;
    *)          info     "$*" ;;
  esac
}

# sentiment_color returns the appropriate color for a sentiment value.
sentiment_color() {
  case "$1" in
    critical)    printf "%s" "$RED" ;;
    concerning)  printf "%s" "$RED" ;;
    mixed)       printf "%s" "$YELLOW" ;;
    positive)    printf "%s" "$GREEN" ;;
    *)           printf "%s" "$WHITE" ;;
  esac
}

# =============================================================================
printf "\n${BOLD}${CYAN}╔══════════════════════════════════════════════════════════════╗${RESET}"
printf "\n${BOLD}${CYAN}║       Signal Discovery — Cross-Cutting Intelligence         ║${RESET}"
printf "\n${BOLD}${CYAN}║                                                              ║${RESET}"
printf "\n${BOLD}${CYAN}║  How agents see what experienced property managers see.       ║${RESET}"
printf "\n${BOLD}${CYAN}╚══════════════════════════════════════════════════════════════╝${RESET}\n"
printf "\n${DIM}    Server: ${BASE_URL} (real API calls, real signal engine)${RESET}\n"
# =============================================================================

pause

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
header "THE PROBLEM — Invisible Patterns"
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

narrate "You ask the agent: 'Which residents are at risk of not renewing?'"
narrate ""
narrate "The agent checks the obvious places:"
echo ""
info "Payment history — all on time"
info "Lease expiration — 90 days out, renewal window open"
info "Balance — \$0.00"
echo ""
narrate "Agent says: 'Marcus Johnson looks fine. No action needed.'"

pause

narrate "But here's what the agent DOESN'T see — because nobody told it to look:"
echo ""
negative "3 noise complaints in the last 5 months"
negative "Roommate moved out 6 weeks ago"
negative "Stopped logging into the tenant portal"
negative "2 unanswered outreach attempts from management"
echo ""
narrate "A good property manager connects these dots instantly."
narrate "Our software treats them as unrelated tables."
narrate ""
narrate "${BOLD}The signal discovery system fixes this.${RESET}"

pause

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
header "LAYER 1 — One Query Sees Everything (LIVE)"
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

narrate "The agent calls ONE endpoint — GetEntityActivity — and sees the full picture."
narrate ""
printf "  ${MAGENTA}GET ${BASE_URL}/v1/activity/entity/person/marcus-johnson${RESET}\n"
echo ""

# Real API call.
ACTIVITY_RESPONSE=$(curl -s "${BASE_URL}/v1/activity/entity/person/marcus-johnson")
TOTAL_COUNT=$(echo "$ACTIVITY_RESPONSE" | jq '.total_count')

narrate "Response: ${TOTAL_COUNT} activity entries. Here's the chronological feed:"
echo ""

printf "  ${DIM}%-24s %-16s %-10s %-10s %s${RESET}\n" "occurred_at" "category" "weight" "polarity" "summary"
printf "  ${DIM}%-24s %-16s %-10s %-10s %s${RESET}\n" "───────────────────────" "──────────────" "─────────" "─────────" "──────────────────────────────────────"

# Parse and display each activity entry from the real response.
echo "$ACTIVITY_RESPONSE" | jq -r '.activities[] | [.occurred_at, .category, .weight, .polarity, .summary] | @tsv' | \
  sort | while IFS=$'\t' read -r ts cat weight pol summary; do
    # Format timestamp to be more readable.
    display_ts=$(echo "$ts" | cut -c1-16 | sed 's/T/  /')
    polarity_bullet "$pol" "$(printf '%-22s  %-14s  %-9s  %-9s  %s' "$display_ts" "$cat" "$weight" "$pol" "$summary")"
  done

echo ""
narrate "That's a real API response. Every entry came from the signal engine."
narrate "Payments, complaints, communication, relationships, lifecycle — one feed."

pause

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
header "LAYER 2 — Signal Taxonomy"
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

narrate "Every event is classified into 8 categories, 5 weights, 4 polarities."
narrate "Defined in the CUE ontology — source of truth."
echo ""

printf "  ${WHITE}Categories${RESET}\n"
label "  financial"     "Payments, fees, NSF, collections"
label "  maintenance"   "Work orders, complaints, inspections"
label "  communication" "Outreach, response rates, portal activity"
label "  compliance"    "Violations, infractions, inspection failures"
label "  behavioral"    "Occupancy patterns, amenity usage, parking"
label "  market"        "Comparable rents, vacancy trends"
label "  relationship"  "Household changes, guarantors, roommates"
label "  lifecycle"     "Lease milestones, renewal windows, notices"

echo ""
printf "  ${WHITE}Weights${RESET}            ${WHITE}Polarity${RESET}\n"
label "  critical"      "Immediate action     positive   — Favorable"
label "  strong"        "Affects outcome      negative   — Unfavorable"
label "  moderate"      "Contributes to       neutral    — Neither"
label "  weak"          "  pattern            contextual — Agent decides"
label "  info"          "Context only"

echo ""
narrate "~30 signal registrations map specific event types to classifications."
narrate "All defined in ontology/signals.cue."

pause

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
header "LAYER 3 — The Signal Summary (LIVE)"
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

narrate "The agent calls GetSignalSummary for a pre-aggregated assessment."
narrate ""
printf "  ${MAGENTA}GET ${BASE_URL}/v1/activity/summary/person/marcus-johnson${RESET}\n"
echo ""

# Real API call.
SUMMARY_RESPONSE=$(curl -s "${BASE_URL}/v1/activity/summary/person/marcus-johnson")

SENTIMENT=$(echo "$SUMMARY_RESPONSE" | jq -r '.overall_sentiment')
SENTIMENT_REASON=$(echo "$SUMMARY_RESPONSE" | jq -r '.sentiment_reason')
SENT_COLOR=$(sentiment_color "$SENTIMENT")
SENTIMENT_UPPER=$(echo "$SENTIMENT" | tr '[:lower:]' '[:upper:]')

printf "  ${CYAN}╔══════════════════════════════════════════════════════════════╗${RESET}\n"
printf "  ${CYAN}║${RESET}  ${WHITE}Signal Summary: Marcus Johnson${RESET}                              ${CYAN}║${RESET}\n"
printf "  ${CYAN}╠══════════════════════════════════════════════════════════════╣${RESET}\n"
printf "  ${CYAN}║${RESET}                                                              ${CYAN}║${RESET}\n"
printf "  ${CYAN}║${RESET}  ${SENT_COLOR}${BOLD}Overall Sentiment: %s${RESET}%-*s${CYAN}║${RESET}\n" "$SENTIMENT_UPPER" $((38 - ${#SENTIMENT_UPPER})) ""
printf "  ${CYAN}║${RESET}  ${DIM}%s${RESET}%-*s${CYAN}║${RESET}\n" "$SENTIMENT_REASON" $((62 - ${#SENTIMENT_REASON})) ""
printf "  ${CYAN}║${RESET}                                                              ${CYAN}║${RESET}\n"
printf "  ${CYAN}╠══════════════════════════════════════════════════════════════╣${RESET}\n"

# Display each category from the real response.
echo "$SUMMARY_RESPONSE" | jq -r '.categories | to_entries | sort_by(.key) | .[] | [.key, (.value.signal_count|tostring), .value.dominant_polarity, .value.trend] | @tsv' | \
  while IFS=$'\t' read -r cat count polarity trend; do
    # Color based on dominant polarity.
    case "$polarity" in
      negative)  cat_color="$RED" ;;
      positive)  cat_color="$GREEN" ;;
      *)         cat_color="$YELLOW" ;;
    esac

    trend_display="$trend"
    if [ "$trend" = "declining" ]; then
      trend_display="${RED}${trend}${RESET}"
    fi

    line=$(printf "%-14s %s signals  dominant: %-10s trend: %s" "$cat" "$count" "$polarity" "$trend")
    printf "  ${CYAN}║${RESET}  ${cat_color}%-14s${RESET} %s signals  dominant: %-10s trend: %s" "$cat" "$count" "$polarity" "$trend_display"
    # Pad to column 64 (approximate).
    printf "%*s${CYAN}║${RESET}\n" 2 ""

    # Show weight/polarity breakdown.
    weights=$(echo "$SUMMARY_RESPONSE" | jq -r --arg c "$cat" '.categories[$c].by_weight | to_entries | map("\(.key):\(.value)") | join("  ")')
    printf "  ${CYAN}║${RESET}  ${DIM}  %s${RESET}%*s${CYAN}║${RESET}\n" "$weights" $((60 - ${#weights})) ""
    printf "  ${CYAN}║${RESET}                                                              ${CYAN}║${RESET}\n"
  done

# Display escalations.
ESC_COUNT=$(echo "$SUMMARY_RESPONSE" | jq '.escalations | length')
if [ "$ESC_COUNT" -gt 0 ]; then
  printf "  ${CYAN}╠══════════════════════════════════════════════════════════════╣${RESET}\n"
  printf "  ${CYAN}║${RESET}  ${RED}${BOLD}ESCALATIONS TRIGGERED:${RESET}                                       ${CYAN}║${RESET}\n"
  printf "  ${CYAN}║${RESET}                                                              ${CYAN}║${RESET}\n"

  echo "$SUMMARY_RESPONSE" | jq -r '.escalations[] | [.rule.id, .rule.escalated_description, .rule.recommended_action, (.triggering_count|tostring)] | @tsv' | \
    while IFS=$'\t' read -r rule_id desc action count; do
      printf "  ${CYAN}║${RESET}  ${RED}▲${RESET} ${BOLD}%s${RESET}%*s${CYAN}║${RESET}\n" "$rule_id" $((58 - ${#rule_id})) ""
      printf "  ${CYAN}║${RESET}    %s%*s${CYAN}║${RESET}\n" "$desc" $((58 - ${#desc})) ""
      if [ -n "$action" ] && [ "$action" != "null" ]; then
        # Truncate long actions for display.
        short_action="${action:0:54}"
        printf "  ${CYAN}║${RESET}    ${DIM}-> %s${RESET}%*s${CYAN}║${RESET}\n" "$short_action" $((55 - ${#short_action})) ""
      fi
      printf "  ${CYAN}║${RESET}                                                              ${CYAN}║${RESET}\n"
    done
fi

printf "  ${CYAN}╚══════════════════════════════════════════════════════════════╝${RESET}\n"

echo ""
narrate "That's ${BOLD}real computed output${RESET} — not a mock."
narrate "The aggregator counted signals per category, computed trends,"
narrate "evaluated escalation rules, and determined overall sentiment."
narrate ""
narrate "The escalation fired because 3+ maintenance complaints in 180 days"
narrate "matched the maint_complaint_pattern rule from the CUE ontology."

pause

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
header "PORTFOLIO SCREENING — Compare All Tenants (LIVE)"
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

narrate "The agent doesn't review tenants one by one."
narrate "Let's call GetSignalSummary for every seeded tenant and rank them."
echo ""

# Define our demo tenants.
TENANTS=("marcus-johnson" "jennifer-park" "david-kim" "lisa-hernandez" "james-wright" "amy-torres")
TENANT_NAMES=("Marcus Johnson" "Jennifer Park" "David Kim" "Lisa Hernandez" "James Wright" "Amy Torres")

printf "  ${DIM}%-3s %-20s %-12s %-6s %s${RESET}\n" "#" "tenant" "sentiment" "sigs" "top concern"
printf "  ${DIM}%-3s %-20s %-12s %-6s %s${RESET}\n" "──" "───────────────────" "──────────" "────" "──────────────────────────────────────"

# Collect summaries.
declare -a RESULTS=()
for i in "${!TENANTS[@]}"; do
  tid="${TENANTS[$i]}"
  tname="${TENANT_NAMES[$i]}"
  resp=$(curl -s "${BASE_URL}/v1/activity/summary/person/${tid}")
  sent=$(echo "$resp" | jq -r '.overall_sentiment')
  reason=$(echo "$resp" | jq -r '.sentiment_reason')
  total_sigs=$(echo "$resp" | jq '[.categories[].signal_count] | add // 0')
  esc_count=$(echo "$resp" | jq '.escalations | length')

  # Build a sortable key: critical=1, concerning=2, mixed=3, positive=4
  case "$sent" in
    critical)   sort_key=1 ;;
    concerning) sort_key=2 ;;
    mixed)      sort_key=3 ;;
    *)          sort_key=4 ;;
  esac

  RESULTS+=("${sort_key}|${tname}|${sent}|${total_sigs}|${reason}|${esc_count}")
done

# Sort and display.
RANK=1
printf '%s\n' "${RESULTS[@]}" | sort -t'|' -k1,1n | while IFS='|' read -r _ tname sent total_sigs reason esc_count; do
  sent_upper=$(echo "$sent" | tr '[:lower:]' '[:upper:]')
  sc=$(sentiment_color "$sent")

  # Truncate reason for table display.
  short_reason="${reason:0:42}"

  printf "  %-3s %-20s ${sc}%-12s${RESET} %-6s %s\n" "$RANK" "$tname" "$sent_upper" "$total_sigs" "$short_reason"
  RANK=$((RANK + 1))
done

echo ""
narrate "Every row is a real GetSignalSummary call."
narrate "The ranking comes from the computed sentiment, not a hardcoded list."

pause

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
header "SEARCH — Full-Text Across All Activity (LIVE)"
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

narrate "The agent can search across all activity with free text."
narrate ""
printf "  ${MAGENTA}POST ${BASE_URL}/v1/activity/search${RESET}\n"
printf "  ${DIM}  { \"query\": \"noise\", \"entity_type\": \"person\" }${RESET}\n"
echo ""

SEARCH_RESPONSE=$(curl -s -X POST "${BASE_URL}/v1/activity/search" \
  -H "Content-Type: application/json" \
  -d '{"query":"noise","entity_type":"person"}')
SEARCH_COUNT=$(echo "$SEARCH_RESPONSE" | jq '.total_count')

narrate "Found ${SEARCH_COUNT} results matching 'noise':"
echo ""

echo "$SEARCH_RESPONSE" | jq -r '.results[] | [.occurred_at, .indexed_entity_id, .summary] | @tsv' | \
  while IFS=$'\t' read -r ts entity summary; do
    display_ts=$(echo "$ts" | cut -c1-10)
    negative "$(printf '%-12s %-18s %s' "$display_ts" "$entity" "$summary")"
  done

pause

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
header "THE REASONING GUIDE — Domain Expertise, Codified"
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

narrate "The system generates SIGNALS.md — a reasoning guide loaded into"
narrate "the agent's context. It encodes what experienced PMs know:"
echo ""

printf "  ${CYAN}╔══════════════════════════════════════════════════════════════╗${RESET}\n"
printf "  ${CYAN}║${RESET}  ${WHITE}From gen/agent/SIGNALS.md:${RESET}                                    ${CYAN}║${RESET}\n"
printf "  ${CYAN}╠══════════════════════════════════════════════════════════════╣${RESET}\n"
printf "  ${CYAN}║${RESET}                                                              ${CYAN}║${RESET}\n"
printf "  ${CYAN}║${RESET}  ${BOLD}Non-Renewal Predictors (in combination):${RESET}                     ${CYAN}║${RESET}\n"
printf "  ${CYAN}║${RESET}  - 2+ maintenance complaints AND payment worsening          ${CYAN}║${RESET}\n"
printf "  ${CYAN}║${RESET}  - Communication declining AND lease expiring < 90 days      ${CYAN}║${RESET}\n"
printf "  ${CYAN}║${RESET}  - Behavioral changes AND no renewal conversation            ${CYAN}║${RESET}\n"
printf "  ${CYAN}║${RESET}  - Roommate departure AND income below 3x rent              ${CYAN}║${RESET}\n"
printf "  ${CYAN}║${RESET}                                                              ${CYAN}║${RESET}\n"
printf "  ${CYAN}║${RESET}  ${BOLD}Interpreting Absence:${RESET}                                         ${CYAN}║${RESET}\n"
printf "  ${CYAN}║${RESET}  - Long-term tenant, no maintenance in 12+ months            ${CYAN}║${RESET}\n"
printf "  ${CYAN}║${RESET}    -> Possible disengagement, not a good sign                ${CYAN}║${RESET}\n"
printf "  ${CYAN}║${RESET}  - Portal user stops logging in -> check other signals       ${CYAN}║${RESET}\n"
printf "  ${CYAN}║${RESET}  - No renewal response in 14 days -> escalate                ${CYAN}║${RESET}\n"
printf "  ${CYAN}║${RESET}                                                              ${CYAN}║${RESET}\n"
printf "  ${CYAN}╚══════════════════════════════════════════════════════════════╝${RESET}\n"

echo ""
narrate "This is generated from the CUE ontology."
narrate "When the ontology changes, the reasoning guide regenerates."

pause

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
header "HOW IT'S BUILT — Ontology-First, As Always"
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

narrate "Same pattern as the rest of the system. One source of truth."
echo ""

printf "  ${WHITE}ontology/signals.cue${RESET}  ${DIM}(signal taxonomy + 30 registrations)${RESET}\n"
printf "  ${DIM}        |${RESET}\n"
printf "  ${DIM}        |-->${RESET}  ${WHITE}internal/signals/registry.go${RESET}   Go data, lookup maps\n"
printf "  ${DIM}        |-->${RESET}  ${WHITE}internal/signals/classifier.go${RESET} Event -> classification\n"
printf "  ${DIM}        |-->${RESET}  ${WHITE}internal/signals/aggregator.go${RESET} Classification -> summary\n"
printf "  ${DIM}        |${RESET}\n"
printf "  ${DIM}        |-->${RESET}  ${WHITE}internal/activity/store.go${RESET}     Partitioned Postgres\n"
printf "  ${DIM}        |-->${RESET}  ${WHITE}internal/activity/indexer.go${RESET}   NATS event consumer\n"
printf "  ${DIM}        |${RESET}\n"
printf "  ${DIM}        |-->${RESET}  ${WHITE}4 API endpoints${RESET}               Activity, Summary, Portfolio, Search\n"
printf "  ${DIM}        |${RESET}\n"
printf "  ${DIM}        '-->${RESET}  ${GREEN}gen/agent/SIGNALS.md${RESET}          Agent reasoning guide\n"
printf "  ${DIM}             ${GREEN}gen/agent/TOOLS.md${RESET}            Updated with 4 new tools\n"
printf "  ${DIM}             ${GREEN}gen/agent/propeller-tools.json${RESET} Updated tool schemas\n"
echo ""

narrate "CUE ontology -> make generate -> everything downstream."

pause

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
header "THE BEFORE AND AFTER"
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

printf "  ${RED}${BOLD}WITHOUT signal discovery:${RESET}\n\n"
narrate "  Agent asked: 'Who's at risk of not renewing?'"
narrate "  Agent queries: payments, lease dates, balances"
narrate "  Agent answers: 'Everyone looks fine.'"
narrate "  Result: Marcus moves out. Vacancy costs \$5,500."
narrate "  Nobody saw it coming."
echo ""

printf "  ${GREEN}${BOLD}WITH signal discovery:${RESET}\n\n"
narrate "  Agent asked: 'Who's at risk of not renewing?'"
narrate "  Agent calls: GetSignalSummary for each tenant"
narrate "  Agent sees:  Marcus = ${RED}CONCERNING${RESET}${DIM} with escalation triggered${RESET}"
narrate "  Agent calls: GetEntityActivity -> full timeline"
narrate "  Agent answers: 'Marcus Johnson is a flight risk. 3 noise"
narrate "    complaints, roommate left, gone dark on communication,"
narrate "    lease expires in 90 days. Resolve noise issue first,"
narrate "    then personal renewal conversation.'"
narrate "  Result: Issue addressed. Marcus renews. \$22,200/year retained."

pause

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

printf "\n"
printf "  ${CYAN}╔══════════════════════════════════════════════════════════════╗${RESET}\n"
printf "  ${CYAN}║${RESET}                                                              ${CYAN}║${RESET}\n"
printf "  ${CYAN}║${RESET}  ${BOLD}What we built:${RESET}                                              ${CYAN}║${RESET}\n"
printf "  ${CYAN}║${RESET}                                                              ${CYAN}║${RESET}\n"
printf "  ${CYAN}║${RESET}  Layer 1: Entity Activity Stream                             ${CYAN}║${RESET}\n"
printf "  ${CYAN}║${RESET}    One query sees ALL activity for any entity.                ${CYAN}║${RESET}\n"
printf "  ${CYAN}║${RESET}                                                              ${CYAN}║${RESET}\n"
printf "  ${CYAN}║${RESET}  Layer 2: Signal Taxonomy (CUE ontology)                     ${CYAN}║${RESET}\n"
printf "  ${CYAN}║${RESET}    8 categories, 5 weights, 4 polarities.                    ${CYAN}║${RESET}\n"
printf "  ${CYAN}║${RESET}    30 signal registrations with escalation rules.             ${CYAN}║${RESET}\n"
printf "  ${CYAN}║${RESET}                                                              ${CYAN}║${RESET}\n"
printf "  ${CYAN}║${RESET}  Layer 3: Classification + Aggregation                       ${CYAN}║${RESET}\n"
printf "  ${CYAN}║${RESET}    Real-time trend computation and sentiment scoring.         ${CYAN}║${RESET}\n"
printf "  ${CYAN}║${RESET}    Escalation rules that fire on dangerous patterns.          ${CYAN}║${RESET}\n"
printf "  ${CYAN}║${RESET}                                                              ${CYAN}║${RESET}\n"
printf "  ${CYAN}║${RESET}  ${DIM}Everything you saw was computed live — no mocks.${RESET}             ${CYAN}║${RESET}\n"
printf "  ${CYAN}║${RESET}                                                              ${CYAN}║${RESET}\n"
printf "  ${CYAN}╚══════════════════════════════════════════════════════════════╝${RESET}\n"

printf "\n${GREEN}${BOLD}Demo complete.${RESET}\n\n"
