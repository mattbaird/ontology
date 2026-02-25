#!/usr/bin/env bash
# Build script for Air. Runs make generate when CUE files have changed,
# then builds the signaldemo server.
set -e

STAMP="tmp/.cue_stamp"
mkdir -p tmp

# Check if any .cue file is newer than the last generation stamp.
if [ ! -f "$STAMP" ] || find ontology/ codegen/ -name '*.cue' -newer "$STAMP" 2>/dev/null | grep -q .; then
  echo "==> CUE files changed, running make generate..."
  make generate
  touch "$STAMP"
fi

go build -o ./tmp/signaldemo ./cmd/signaldemo
