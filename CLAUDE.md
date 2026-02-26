# Propeller Project Instructions

## Skills
Before generating tests, read: .claude/skills/test-generation.md
Before generating documentation, read: .claude/skills/doc-generation.md
Before generating signal annotations, read: .claude/skills/signal-annotation.md

## Ontology Files
The source of truth is in ontology/*.cue. Never hand-write types that 
represent domain entities. Everything derives from these files.

## Generation Rule
After any change to ontology/*.cue, run both skills: regenerate tests 
AND documentation. Both are projections of the ontology and must stay in sync.
