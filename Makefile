.PHONY: validate entgen entgenerate handlergen apigen eventgen authzgen agentgen openapigen uigen uirender generate-ui generate testgen replgen driftcheck ci-check clean serve migrate-diff migrate-apply migrate-status

# Validate CUE ontology
validate:
	cue vet ./ontology/...

# Generate Ent schemas from CUE ontology
entgen:
	go run ./cmd/entgen

# Run Ent code generation (generates Go CRUD code from schemas)
entgenerate:
	go generate ./ent

# Generate HTTP handlers and routes from CUE ontology + apigen.cue
handlergen:
	go run ./cmd/handlergen

# Generate proto files from CUE ontology
apigen:
	go run ./cmd/apigen

# Generate event type constants + catalog from CUE ontology
eventgen:
	go run ./cmd/eventgen

# Generate OPA/Rego policy scaffolds from CUE ontology
authzgen:
	go run ./cmd/authzgen

# Generate ONTOLOGY.md, TOOLS.md, propeller-tools.json from CUE ontology
agentgen:
	go run ./cmd/agentgen

# Generate OpenAPI 3.1 spec from CUE ontology + apigen.cue
openapigen:
	go run ./cmd/openapigen

# Generate UI schemas from CUE ontology (Layer 1)
uigen:
	go run ./cmd/uigen

# Generate Svelte components from UI schemas (Layer 2)
uirender:
	go run ./cmd/uirender

# Generate all UI artifacts (schemas + components)
generate-ui: uigen uirender

# Generate state machine transition tests from CUE ontology
testgen:
	go run ./cmd/testgen

# Generate REPL schema registry and entity dispatchers from CUE ontology
replgen:
	go run ./cmd/replgen

# Validate cross-boundary consistency (ontology, commands, events, api, policies)
driftcheck:
	go run ./cmd/driftcheck

# Full generation pipeline
generate: validate entgen entgenerate handlergen apigen eventgen authzgen agentgen openapigen generate-ui testgen replgen

# CI check: verify generated code matches ontology (no drift)
ci-check: generate
	@if [ -n "$$(git status --porcelain)" ]; then \
		echo "ERROR: Generated code is out of sync with ontology."; \
		echo "Run 'make generate' and commit the results."; \
		git diff --stat; \
		exit 1; \
	fi
	@echo "OK: Generated code matches ontology."

# Build all packages
build:
	go build ./...

# Run tests
test:
	go test ./...

# Run the REST API server (SQLite, port 8080)
serve:
	go run ./cmd/server

# Generate a new Atlas migration from schema changes
migrate-diff:
	atlas migrate diff --env local

# Apply pending Atlas migrations
migrate-apply:
	atlas migrate apply --env local

# Show Atlas migration status
migrate-status:
	atlas migrate status --env local

# Clean generated files
clean:
	rm -rf ent/generated
	rm -rf gen/proto/*.go
	rm -rf gen/connect/*.go
	rm -rf gen/ui/
