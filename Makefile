.PHONY: validate entgen entgenerate apigen bufgen generate ci-check clean serve

# Validate CUE ontology
validate:
	cue vet ./ontology/...

# Generate Ent schemas from CUE ontology
entgen:
	go run ./cmd/entgen

# Run Ent code generation (generates Go CRUD code from schemas)
entgenerate:
	go generate ./ent

# Generate proto files from CUE ontology
apigen:
	go run ./cmd/apigen

# Run buf to generate Go/TS stubs from proto files
bufgen:
	buf generate

# Full generation pipeline
generate: validate entgen entgenerate apigen bufgen

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

# Clean generated files
clean:
	rm -rf ent/generated
	rm -rf gen/proto/*.go
	rm -rf gen/connect/*.go
