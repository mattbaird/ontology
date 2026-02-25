// cmd/agentgen generates agent tool definitions and context documents from the CUE ontology.
//
// Output:
// - gen/agent/ONTOLOGY.md — Full domain model documentation for agent system prompts
// - gen/agent/STATE_MACHINES.md — All state machines with valid transitions
// - gen/agent/TOOLS.md — Human-readable tool documentation
// - gen/agent/propeller-tools.json — Anthropic function-calling format tool definitions
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/load"
)

// ToolDef represents an Anthropic function-calling tool definition.
type ToolDef struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"input_schema"`
}

// entityTransitionMap maps entity names to CUE state machine definition names.
var entityTransitionMap = map[string]string{
	"Lease":          "#LeaseTransitions",
	"Unit":           "#UnitTransitions",
	"Application":    "#ApplicationTransitions",
	"JournalEntry":   "#JournalEntryTransitions",
	"Portfolio":      "#PortfolioTransitions",
	"Property":       "#PropertyTransitions",
	"PersonRole":     "#PersonRoleTransitions",
	"Organization":   "#OrganizationTransitions",
	"BankAccount":    "#BankAccountTransitions",
	"Reconciliation": "#ReconciliationTransitions",
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("agentgen: ")

	projectRoot := findProjectRoot()
	ctx := cuecontext.New()

	// Load ontology
	insts := load.Instances([]string{"./ontology"}, &load.Config{Dir: projectRoot})
	if len(insts) == 0 || insts[0].Err != nil {
		log.Fatal("loading ontology failed")
	}
	val := ctx.BuildInstance(insts[0])
	if val.Err() != nil {
		log.Fatalf("building CUE: %v", val.Err())
	}

	// Load apigen for service definitions
	apiInsts := load.Instances([]string{"./codegen"}, &load.Config{Dir: projectRoot})
	if len(apiInsts) == 0 || apiInsts[0].Err != nil {
		log.Fatal("loading codegen failed")
	}
	apiVal := ctx.BuildInstance(apiInsts[0])
	if apiVal.Err() != nil {
		log.Fatalf("building codegen: %v", apiVal.Err())
	}

	outDir := filepath.Join(projectRoot, "gen", "agent")

	// Generate ONTOLOGY.md
	generateOntologyDoc(val, outDir)

	// Generate STATE_MACHINES.md
	generateStateMachinesDoc(val, outDir)

	// Generate TOOLS.md and propeller-tools.json
	generateToolDefs(apiVal, outDir)

	fmt.Println("agentgen: generated agent context documents")
}

func generateOntologyDoc(val cue.Value, outDir string) {
	var buf strings.Builder
	buf.WriteString("# Propeller Domain Ontology\n\n")
	buf.WriteString("This document describes the complete domain model for the Propeller property management system.\n")
	buf.WriteString("It is auto-generated from the CUE ontology and serves as the agent's world model.\n\n")

	buf.WriteString("## Entity Types\n\n")

	var entities []string
	entityFields := make(map[string][]string)

	iter, _ := val.Fields(cue.Definitions(true))
	for iter.Next() {
		label := iter.Selector().String()
		defVal := iter.Value()

		if defVal.LookupPath(cue.ParsePath("id")).Err() != nil ||
			defVal.LookupPath(cue.ParsePath("audit")).Err() != nil {
			continue
		}

		name := strings.TrimPrefix(label, "#")
		entities = append(entities, name)

		var fields []string
		fIter, _ := defVal.Fields(cue.Optional(true))
		for fIter.Next() {
			fname := strings.TrimSuffix(fIter.Selector().String(), "?")
			if fname == "audit" || strings.HasPrefix(fname, "_") {
				continue
			}
			optional := ""
			if fIter.IsOptional() {
				optional = " (optional)"
			}
			fields = append(fields, fname+optional)
		}
		entityFields[name] = fields
	}

	sort.Strings(entities)

	for _, ent := range entities {
		buf.WriteString(fmt.Sprintf("### %s\n\n", ent))
		buf.WriteString("Fields:\n")
		for _, f := range entityFields[ent] {
			buf.WriteString(fmt.Sprintf("- `%s`\n", f))
		}
		buf.WriteString("\n")
	}

	// Relationships
	buf.WriteString("## Relationships\n\n")
	relList := val.LookupPath(cue.ParsePath("relationships"))
	if relList.Err() == nil {
		rIter, _ := relList.List()
		for rIter.Next() {
			rel := rIter.Value()
			from, _ := rel.LookupPath(cue.ParsePath("from")).String()
			to, _ := rel.LookupPath(cue.ParsePath("to")).String()
			semantic, _ := rel.LookupPath(cue.ParsePath("semantic")).String()
			cardinality, _ := rel.LookupPath(cue.ParsePath("cardinality")).String()
			buf.WriteString(fmt.Sprintf("- **%s → %s** (%s): %s\n", from, to, cardinality, semantic))
		}
	}
	buf.WriteString("\n")

	outPath := filepath.Join(outDir, "ONTOLOGY.md")
	os.WriteFile(outPath, []byte(buf.String()), 0644)
	fmt.Println("Generated gen/agent/ONTOLOGY.md")
}

func generateStateMachinesDoc(val cue.Value, outDir string) {
	var buf strings.Builder
	buf.WriteString("# State Machines\n\n")
	buf.WriteString("Every entity with a `status` field has an explicit state machine.\n")
	buf.WriteString("Invalid transitions are rejected at the persistence layer.\n\n")

	// Sort entity names for deterministic output
	var entNames []string
	for k := range entityTransitionMap {
		entNames = append(entNames, k)
	}
	sort.Strings(entNames)

	for _, entName := range entNames {
		cueName := entityTransitionMap[entName]
		smVal := val.LookupPath(cue.ParsePath(cueName))
		if smVal.Err() != nil {
			continue
		}

		buf.WriteString(fmt.Sprintf("## %s\n\n", entName))
		buf.WriteString("| Current State | Valid Transitions |\n")
		buf.WriteString("|---|---|\n")

		var states []string
		stateMap := make(map[string][]string)
		sIter, _ := smVal.Fields()
		for sIter.Next() {
			state := sIter.Selector().String()
			states = append(states, state)

			var targets []string
			tIter, _ := sIter.Value().List()
			for tIter.Next() {
				t, _ := tIter.Value().String()
				targets = append(targets, t)
			}
			stateMap[state] = targets
		}

		for _, state := range states {
			targets := stateMap[state]
			if len(targets) == 0 {
				buf.WriteString(fmt.Sprintf("| %s | *(terminal)* |\n", state))
			} else {
				buf.WriteString(fmt.Sprintf("| %s | %s |\n", state, strings.Join(targets, ", ")))
			}
		}
		buf.WriteString("\n")
	}

	outPath := filepath.Join(outDir, "STATE_MACHINES.md")
	os.WriteFile(outPath, []byte(buf.String()), 0644)
	fmt.Println("Generated gen/agent/STATE_MACHINES.md")
}

func generateToolDefs(apiVal cue.Value, outDir string) {
	var tools []ToolDef
	var toolsMD strings.Builder

	toolsMD.WriteString("# Propeller API Tools\n\n")
	toolsMD.WriteString("Available operations for the Propeller property management system.\n")
	toolsMD.WriteString("Each tool corresponds to a Connect-RPC API method.\n\n")

	svcList := apiVal.LookupPath(cue.ParsePath("services"))
	if svcList.Err() != nil {
		log.Printf("warning: no services found")
		return
	}

	sIter, _ := svcList.List()
	for sIter.Next() {
		svc := sIter.Value()
		svcName, _ := svc.LookupPath(cue.ParsePath("name")).String()
		toolsMD.WriteString(fmt.Sprintf("## %s\n\n", svcName))

		opIter, _ := svc.LookupPath(cue.ParsePath("operations")).List()
		for opIter.Next() {
			op := opIter.Value()
			name, _ := op.LookupPath(cue.ParsePath("name")).String()
			desc, _ := op.LookupPath(cue.ParsePath("description")).String()
			opType, _ := op.LookupPath(cue.ParsePath("type")).String()
			entity, _ := op.LookupPath(cue.ParsePath("entity")).String()

			// Generate tool name in snake_case
			toolName := toSnake(name)

			// Build input schema based on operation type
			schema := buildInputSchema(opType, entity, op)

			tools = append(tools, ToolDef{
				Name:        toolName,
				Description: desc,
				InputSchema: schema,
			})

			toolsMD.WriteString(fmt.Sprintf("### `%s`\n\n", toolName))
			toolsMD.WriteString(fmt.Sprintf("%s\n\n", desc))
			toolsMD.WriteString(fmt.Sprintf("- **Type:** %s\n", opType))
			toolsMD.WriteString(fmt.Sprintf("- **Entity:** %s\n", entity))
			toolsMD.WriteString("\n")
		}
	}

	// Write propeller-tools.json
	toolsJSON, _ := json.MarshalIndent(tools, "", "  ")
	toolsPath := filepath.Join(outDir, "propeller-tools.json")
	os.WriteFile(toolsPath, toolsJSON, 0644)
	fmt.Println("Generated gen/agent/propeller-tools.json")

	// Write TOOLS.md
	toolsMDPath := filepath.Join(outDir, "TOOLS.md")
	os.WriteFile(toolsMDPath, []byte(toolsMD.String()), 0644)
	fmt.Println("Generated gen/agent/TOOLS.md")
}

func buildInputSchema(opType, entity string, op cue.Value) json.RawMessage {
	schema := map[string]interface{}{
		"type": "object",
	}

	properties := make(map[string]interface{})
	var required []string

	switch opType {
	case "get":
		properties["id"] = map[string]interface{}{
			"type":        "string",
			"description": fmt.Sprintf("The UUID of the %s to retrieve", entity),
		}
		properties["include"] = map[string]interface{}{
			"type":        "array",
			"items":       map[string]string{"type": "string"},
			"description": "Edge names to include in the response",
		}
		required = []string{"id"}

	case "list":
		properties["page_size"] = map[string]interface{}{
			"type":        "integer",
			"description": "Number of results per page (max 100)",
		}
		properties["page_token"] = map[string]interface{}{
			"type":        "string",
			"description": "Cursor token for pagination",
		}
		properties["filter"] = map[string]interface{}{
			"type":        "string",
			"description": "Filter expression",
		}

	case "create":
		properties["data"] = map[string]interface{}{
			"type":        "object",
			"description": fmt.Sprintf("The %s data to create", entity),
		}
		required = []string{"data"}

	case "update":
		properties["id"] = map[string]interface{}{
			"type":        "string",
			"description": fmt.Sprintf("The UUID of the %s to update", entity),
		}
		properties["data"] = map[string]interface{}{
			"type":        "object",
			"description": "Fields to update",
		}
		properties["update_mask"] = map[string]interface{}{
			"type":        "array",
			"items":       map[string]string{"type": "string"},
			"description": "Fields to update (if empty, all provided fields are updated)",
		}
		required = []string{"id", "data"}

	case "transition":
		properties["id"] = map[string]interface{}{
			"type":        "string",
			"description": fmt.Sprintf("The UUID of the %s to transition", entity),
		}
		required = []string{"id"}

		// Add extra fields
		extraIter, _ := op.LookupPath(cue.ParsePath("extra_fields")).List()
		for extraIter.Next() {
			ef, _ := extraIter.Value().String()
			properties[ef] = map[string]interface{}{
				"type":        "string",
				"description": fmt.Sprintf("Additional field: %s", ef),
			}
		}

	case "delete":
		properties["id"] = map[string]interface{}{
			"type":        "string",
			"description": fmt.Sprintf("The UUID of the %s to delete", entity),
		}
		required = []string{"id"}
	}

	schema["properties"] = properties
	if len(required) > 0 {
		schema["required"] = required
	}

	data, _ := json.Marshal(schema)
	return data
}

func toSnake(s string) string {
	var result []rune
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result = append(result, '_')
		}
		result = append(result, r)
	}
	return strings.ToLower(string(result))
}

func findProjectRoot() string {
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			log.Fatal("could not find project root")
		}
		dir = parent
	}
}
