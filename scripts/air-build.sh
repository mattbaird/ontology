#!/usr/bin/env bash
# Build script for Air. Runs make generate when CUE files have changed,
# then auto-diffs Atlas migrations if schemas changed, then builds the target server.
#
# Usage:
#   air              # full server with Ent + SQLite (default)
#   air -- --demo    # same, with seeded activity demo data
set -e

TARGET="${AIR_TARGET:-server}"
STAMP=".cue_stamp"
mkdir -p tmp

# Check if any .cue file is newer than the last generation stamp.
if [ ! -f "$STAMP" ] || find ontology/ codegen/ -name '*.cue' -newer "$STAMP" 2>/dev/null | grep -q .; then
  # Touch stamp FIRST so Air-triggered rebuilds (from generated file changes)
  # don't re-enter this block.
  touch "$STAMP"
  echo "==> CUE files changed, running make generate..."
  make generate
  echo "==> Checking for schema changes..."
  atlas migrate diff --env local 2>/dev/null || true
fi

go build -o ./tmp/app "./cmd/${TARGET}"
