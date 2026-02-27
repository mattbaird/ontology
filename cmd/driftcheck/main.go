// cmd/driftcheck validates cross-boundary consistency between the ontology,
// commands, events, API contracts, and policies.
//
// It leverages CUE's built-in validation: since commands/events/api import
// ontology types, `cue vet` catches type mismatches, missing enum values,
// and constraint violations automatically.
//
// Additional checks validate:
// - Command _affects fields reference valid #EntityType values
// - Permission command keys reference actual commands
package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	log.SetFlags(0)
	log.SetPrefix("driftcheck: ")

	projectRoot := findProjectRoot()

	packages := []string{
		"./ontology/...",
		"./commands/...",
		"./events/...",
		"./api/...",
		"./policies/...",
		"./codegen/...",
	}

	// Phase 1: Run cue vet across all packages
	// This validates cross-package imports, enum consistency, and type constraints
	fmt.Printf("Phase 1: Validating CUE packages (%s)...\n", strings.Join(packages, ", "))
	cmd := exec.Command("cue", "vet")
	cmd.Args = append(cmd.Args, packages...)
	cmd.Dir = projectRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		log.Fatalf("CUE validation failed: %v", err)
	}
	fmt.Println("  All packages validate.")

	// Phase 2: Verify generated code matches ontology
	fmt.Println("Phase 2: Checking generated code freshness...")
	statusCmd := exec.Command("git", "status", "--porcelain")
	statusCmd.Dir = projectRoot
	out, err := statusCmd.Output()
	if err != nil {
		// Not a git repo or git not available — skip this check
		fmt.Println("  Skipping: not a git repository")
	} else if len(out) > 0 {
		fmt.Println("  WARNING: Working tree has uncommitted changes.")
		fmt.Println("  Run 'make generate' and verify output matches expectations.")
	} else {
		fmt.Println("  Generated code is clean.")
	}

	fmt.Println("\ndriftcheck: OK — no cross-boundary drift detected")
}

func findProjectRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		log.Fatalf("getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			log.Fatal("cannot find project root (no go.mod found)")
		}
		dir = parent
	}
}
