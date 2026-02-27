// cmd/agentgen generates agent tool definitions and context documents from the CUE ontology.
//
// Output:
// - gen/agent/ONTOLOGY.md — Full domain model documentation for agent system prompts
// - gen/agent/STATE_MACHINES.md — All state machines with valid transitions
// - gen/agent/TOOLS.md — Human-readable tool documentation
// - gen/agent/propeller-tools.json — Anthropic function-calling format tool definitions
// - gen/agent/SIGNALS.md — Signal reasoning guide for agent context
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

// State machines are read from the unified #StateMachines map in CUE.
// statefulEntities lists all entities that have state machines, for deterministic ordering.
var statefulEntities = []string{
	"Application", "BankAccount", "Building", "Jurisdiction", "JurisdictionRule",
	"JournalEntry", "Lease", "Organization", "PersonRole", "Portfolio",
	"Property", "Reconciliation", "Space",
}

// ── CUE attribute extraction ─────────────────────────────────────────────────

// extractAttributes reads CUE field-level attributes for agent context.
func extractAttributes(v cue.Value) (sensitive, pii bool) {
	if a := v.Attribute("sensitive"); a.Err() == nil {
		sensitive = true
	}
	if a := v.Attribute("pii"); a.Err() == nil {
		pii = true
	}
	return
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

	// Generate SIGNALS.md
	generateSignalsDoc(val, outDir)

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
		// Skip base entity types — not domain entities
		if name == "BaseEntity" || name == "StatefulEntity" || name == "ImmutableEntity" {
			continue
		}
		entities = append(entities, name)

		var fields []string
		fIter, _ := defVal.Fields(cue.Optional(true))
		for fIter.Next() {
			fname := strings.TrimSuffix(fIter.Selector().String(), "?")
			if fname == "audit" || strings.HasPrefix(fname, "_") {
				continue
			}
			sensitive, pii := extractAttributes(fIter.Value())
			if pii {
				continue // @pii() fields are never exposed in agent context
			}
			annotation := ""
			if fIter.IsOptional() {
				annotation = " (optional)"
			}
			if sensitive {
				annotation += " ⚠️ sensitive"
			}
			fields = append(fields, fname+annotation)
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

	for _, entName := range statefulEntities {
		smVal := val.LookupPath(cue.ParsePath("#StateMachines." + toSnake(entName)))
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

func generateSignalsDoc(val cue.Value, outDir string) {
	var buf strings.Builder

	buf.WriteString("# Signal Reasoning Guide\n\n")
	buf.WriteString("This document teaches the agent how to reason about signals from the entity activity stream.\n")
	buf.WriteString("It is auto-generated from the CUE ontology signal definitions.\n\n")

	// Read signal category descriptions.
	buf.WriteString("## Signal Categories\n\n")
	categories := []string{"financial", "maintenance", "communication", "compliance", "behavioral", "market", "relationship", "lifecycle"}
	catDescs := val.LookupPath(cue.ParsePath("signal_category_descriptions"))
	for _, cat := range categories {
		desc := ""
		if catDescs.Err() == nil {
			if d, err := catDescs.LookupPath(cue.ParsePath(cat)).String(); err == nil {
				desc = d
			}
		}
		buf.WriteString(fmt.Sprintf("### %s\n%s\n\n", cat, desc))
	}

	// Read signal weight descriptions.
	buf.WriteString("## Signal Weights\n\n")
	weights := []string{"critical", "strong", "moderate", "weak", "info"}
	weightDescs := val.LookupPath(cue.ParsePath("signal_weight_descriptions"))
	buf.WriteString("| Weight | Meaning |\n|---|---|\n")
	for _, w := range weights {
		desc := ""
		if weightDescs.Err() == nil {
			if d, err := weightDescs.LookupPath(cue.ParsePath(w)).String(); err == nil {
				desc = d
			}
		}
		buf.WriteString(fmt.Sprintf("| %s | %s |\n", w, desc))
	}
	buf.WriteString("\n")

	// Read signal polarity descriptions.
	buf.WriteString("## Signal Polarity\n\n")
	polarities := []string{"positive", "negative", "neutral", "contextual"}
	polDescs := val.LookupPath(cue.ParsePath("signal_polarity_descriptions"))
	buf.WriteString("| Polarity | Meaning |\n|---|---|\n")
	for _, p := range polarities {
		desc := ""
		if polDescs.Err() == nil {
			if d, err := polDescs.LookupPath(cue.ParsePath(p)).String(); err == nil {
				desc = d
			}
		}
		buf.WriteString(fmt.Sprintf("| %s | %s |\n", p, desc))
	}
	buf.WriteString("\n")

	// Read signal registrations and list them.
	buf.WriteString("## Signal Registrations\n\n")
	regList := val.LookupPath(cue.ParsePath("signal_registrations"))
	if regList.Err() == nil {
		rIter, _ := regList.List()
		// Group by category.
		type regEntry struct {
			id, eventType, weight, polarity, description string
		}
		grouped := make(map[string][]regEntry)
		for rIter.Next() {
			r := rIter.Value()
			id, _ := r.LookupPath(cue.ParsePath("id")).String()
			eventType, _ := r.LookupPath(cue.ParsePath("event_type")).String()
			category, _ := r.LookupPath(cue.ParsePath("category")).String()
			weight, _ := r.LookupPath(cue.ParsePath("weight")).String()
			polarity, _ := r.LookupPath(cue.ParsePath("polarity")).String()
			description, _ := r.LookupPath(cue.ParsePath("description")).String()
			grouped[category] = append(grouped[category], regEntry{id, eventType, weight, polarity, description})
		}
		for _, cat := range categories {
			entries := grouped[cat]
			if len(entries) == 0 {
				continue
			}
			buf.WriteString(fmt.Sprintf("### %s signals\n\n", cat))
			buf.WriteString("| Signal | Event Type | Weight | Polarity | Description |\n")
			buf.WriteString("|---|---|---|---|---|\n")
			for _, e := range entries {
				buf.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s |\n",
					e.id, e.eventType, e.weight, e.polarity, e.description))
			}
			buf.WriteString("\n")
		}
	}

	// Assessment workflow.
	buf.WriteString("## Assessment Workflow\n\n")
	buf.WriteString("When evaluating risk, health, or status of any entity:\n\n")
	buf.WriteString("1. Start with GetSignalSummary for overall sentiment and category breakdown.\n")
	buf.WriteString("2. Drill into concerning categories with GetEntityActivity.\n")
	buf.WriteString("3. Look for patterns ACROSS categories — single signals are rarely actionable.\n")
	buf.WriteString("4. Check for ABSENCE of expected signals — silence from an occupied unit is itself data.\n")
	buf.WriteString("5. Evaluate trajectory, not just current state — improving vs declining matters more than absolute counts.\n\n")

	// Cross-category pattern recognition.
	buf.WriteString("## Cross-Category Pattern Recognition\n\n")
	buf.WriteString("### Non-Renewal Predictors (in combination):\n")
	buf.WriteString("- 2+ maintenance complaints AND payment pattern worsening\n")
	buf.WriteString("- Communication responsiveness declining AND lease expiring within 90 days\n")
	buf.WriteString("- Behavioral changes (parking, amenity) AND no renewal conversation initiated\n")
	buf.WriteString("- Roommate departure AND remaining income below 3x rent\n\n")

	buf.WriteString("### Retention Opportunities:\n")
	buf.WriteString("- Long tenancy (2+ years) AND good payment AND recent complaint → Fast resolution retains high-value tenant.\n")
	buf.WriteString("- First-year tenant AND perfect payment AND lease expiring → Standard renewal with minor gesture has high ROI.\n\n")

	buf.WriteString("### Escalation Required:\n")
	buf.WriteString("- Any \"critical\" signal → Manager attention within 24 hours\n")
	buf.WriteString("- 3+ \"strong\" signals across different categories in 90 days → Proactive intervention\n")
	buf.WriteString("- Financial \"critical\" + Communication \"strong\" → In-person visit\n\n")

	// Category-specific reasoning.
	buf.WriteString("## Category-Specific Reasoning\n\n")

	buf.WriteString("### Financial\n")
	buf.WriteString("- Day-of-month consistency is stronger than occasional lateness.\n")
	buf.WriteString("- Partial payments: may indicate effort or decline. Check trend direction.\n")
	buf.WriteString("- NSF is stronger than simple lateness — attempted payment with no funds.\n\n")

	buf.WriteString("### Maintenance\n")
	buf.WriteString("- Complaint frequency matters more than severity.\n")
	buf.WriteString("- Unresolved complaints are exponentially worse than resolved ones.\n")
	buf.WriteString("- Zero maintenance requests from a long-term tenant is unusual — possible disengagement.\n")
	buf.WriteString("- Maintenance requests (not complaints) are POSITIVE — tenant is engaged.\n\n")

	buf.WriteString("### Communication\n")
	buf.WriteString("- Response time TREND matters more than individual response times.\n")
	buf.WriteString("- Tenant-initiated contact is almost always positive regardless of content.\n")
	buf.WriteString("- Silence is the most dangerous communication signal.\n\n")

	buf.WriteString("### Behavioral\n")
	buf.WriteString("- Individual behavioral signals are weak. Only meaningful in combination.\n")
	buf.WriteString("- Parking violations: often first visible sign of norm disengagement.\n")
	buf.WriteString("- Portal activity: leading indicator — drops precede other changes.\n\n")

	buf.WriteString("### Relationship\n")
	buf.WriteString("- Roommate departure: check remaining tenant's income against rent.\n")
	buf.WriteString("- Occupant additions: positive (growing household) or compliance concern.\n")
	buf.WriteString("- Guarantor removal: may signal changed family dynamics.\n\n")

	buf.WriteString("### Lifecycle\n")
	buf.WriteString("- 90-day pre-expiration: when most renewal decisions are made.\n")
	buf.WriteString("- No renewal response by 30 days out: likely leaving.\n")
	buf.WriteString("- Option exercise deadlines: legally binding, never miss.\n\n")

	// Interpreting absence.
	buf.WriteString("## Interpreting Absence\n\n")
	buf.WriteString("These \"non-events\" carry signal value:\n")
	buf.WriteString("- Long-term tenant, no maintenance requests in 12+ months: possible disengagement\n")
	buf.WriteString("- Previously active portal user stops logging in: check for other negative signals\n")
	buf.WriteString("- No response to renewal offer within 14 days: escalate\n")
	buf.WriteString("- Tenant who always paid early now pays on time: subtle trend shift, monitor\n")

	outPath := filepath.Join(outDir, "SIGNALS.md")
	os.WriteFile(outPath, []byte(buf.String()), 0644)
	fmt.Println("Generated gen/agent/SIGNALS.md")
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
