// cmd/entgen generates Ent schemas from CUE ontology definitions.
//
// This is the core code generator in the Propeller ontological architecture.
// It reads CUE definitions from ontology/*.cue and generates Go files in ent/schema/.
//
// Type mapping:
//
//	CUE Type                     Ent Field
//	string & !=""                field.String(name).NotEmpty()
//	int & >= 0                   field.Int(name).NonNegative()
//	bool | *false                field.Bool(name).Default(false)
//	time.Time                    field.Time(name)
//	"a" | "b" | "c"             field.Enum(name).Values("a","b","c")
//	=~"^pattern$"                field.String(name).Match(regexp)
//	#Money (top-level)           Flatten to _amount_cents int64 + _currency string
//	[...#Foo] (list of structs)  field.JSON(name, []types.Foo{})
//	#StructType (sub-object)     field.JSON(name, &types.StructType{})
package main

import (
	"bytes"
	"fmt"
	"go/format"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/template"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/load"
)

// entityDef holds the parsed definition of a domain entity from CUE.
type entityDef struct {
	Name               string
	Fields             []fieldDef
	Edges              []edgeDef
	Immutable          bool // LedgerEntry, JournalEntry
	HasMachine         bool
	Machine            map[string][]string // status -> valid next statuses
	EdgeField          map[string]string   // edge name -> field name (for .Field() binding)
	HasConstraints     bool                // true if entity has cross-field constraints
	ConstraintHookCode string              // pre-rendered Go code for Hooks() + validation function
}

// fieldDef holds the parsed definition of an entity field.
type fieldDef struct {
	Name         string
	EntType      string // "String", "Int", "Int64", "Float64", "Bool", "Time", "Enum", "JSON", "UUID"
	Optional     bool
	Nillable     bool
	NotEmpty     bool
	Immutable    bool
	Sensitive    bool
	Default      string   // Go expression for default value
	EnumValues   []string // For Enum fields
	MatchPattern string   // For String fields with regex constraint
	Validators   []string // Additional validator expressions
	Comment      string
	JSONType     string // For JSON fields, the Go type expression
	NonNegative  bool
	Positive     bool
	Min          string // numeric min
	Max          string // numeric max
}

// edgeDef holds the parsed definition of a relationship edge.
type edgeDef struct {
	Name         string
	Target       string // Target entity type name
	Type         string // "To" or "From"
	Unique       bool
	Required     bool
	RefName      string // For "From" edges, the inverse edge name
	Comment      string
	FieldBinding string // If set, edge uses this field column via .Field()
}

// knownValueTypes maps CUE definition names to their Go type expressions.
var knownValueTypes = map[string]string{
	"#Money":             "types.Money",
	"#NonNegativeMoney":  "types.Money",
	"#PositiveMoney":     "types.Money",
	"#Address":           "types.Address",
	"#ContactMethod":     "types.ContactMethod",
	"#DateRange":         "types.DateRange",
	"#EntityRef":         "types.EntityRef",
	"#RentScheduleEntry": "types.RentScheduleEntry",
	"#RecurringCharge":   "types.RecurringCharge",
	"#LateFeePolicy":     "types.LateFeePolicy",
	"#CAMTerms":          "types.CAMTerms",
	"#TenantImprovement": "types.TenantImprovement",
	"#RenewalOption":     "types.RenewalOption",
	"#SubsidyTerms":      "types.SubsidyTerms",
	"#AccountDimensions": "types.AccountDimensions",
	"#JournalLine":       "types.JournalLine",
	"#RoleAttributes":    "json.RawMessage",
	"#TenantAttributes":    "types.TenantAttributes",
	"#OwnerAttributes":     "types.OwnerAttributes",
	"#ManagerAttributes":   "types.ManagerAttributes",
	"#GuarantorAttributes": "types.GuarantorAttributes",
	"#UsageBasedCharge":    "types.UsageBasedCharge",
	"#PercentageRent":      "types.PercentageRent",
	"#RentAdjustment":      "types.RentAdjustment",
	"#ExpansionRight":      "types.ExpansionRight",
	"#ContractionRight":    "types.ContractionRight",
	"#CAMCategoryTerms":    "types.CAMCategoryTerms",
}

// moneyFieldNames are field names recognized as #Money that get flattened.
var moneyFieldNames = map[string]bool{
	"#Money": true, "#NonNegativeMoney": true, "#PositiveMoney": true,
}

// entities that are immutable (no updates or deletes allowed).
var immutableEntities = map[string]bool{
	"LedgerEntry":  true,
	"JournalEntry": true,
}

// entityTransitionMap maps entity names to their CUE state machine definition names.
var entityTransitionMap = map[string]string{
	"Lease":          "#LeaseTransitions",
	"Space":          "#SpaceTransitions",
	"Building":       "#BuildingTransitions",
	"Application":    "#ApplicationTransitions",
	"JournalEntry":   "#JournalEntryTransitions",
	"Portfolio":      "#PortfolioTransitions",
	"Property":       "#PropertyTransitions",
	"PersonRole":     "#PersonRoleTransitions",
	"Organization":   "#OrganizationTransitions",
	"BankAccount":    "#BankAccountTransitions",
	"Reconciliation": "#ReconciliationTransitions",
}

// findProjectRoot walks up from cwd to find the directory containing go.mod.
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
			log.Fatal("could not find project root (no go.mod found)")
		}
		dir = parent
	}
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("entgen: ")

	ctx := cuecontext.New()

	// Determine project root (where go.mod lives)
	projectRoot := findProjectRoot()

	// Load ontology CUE package
	insts := load.Instances([]string{"./ontology"}, &load.Config{
		Dir: projectRoot,
	})
	if len(insts) == 0 {
		log.Fatal("no CUE instances found in ./ontology")
	}
	if insts[0].Err != nil {
		log.Fatalf("loading CUE: %v", insts[0].Err)
	}

	val := ctx.BuildInstance(insts[0])
	if val.Err() != nil {
		log.Fatalf("building CUE value: %v", val.Err())
	}

	// Parse entities from CUE definitions
	entities := parseEntities(val)

	// Parse relationships
	parseRelationships(val, entities)

	// Parse state machines
	parseStateMachines(val, entities)

	// Remove fields that would conflict with edges (FK fields like property_id, unit_id)
	for _, ent := range entities {
		removeFKFields(ent)
	}

	// Add cross-field constraint hooks
	assignConstraints(entities)

	// Generate Ent schema files
	for _, ent := range entities {
		if err := generateSchema(projectRoot, ent); err != nil {
			log.Fatalf("generating schema for %s: %v", ent.Name, err)
		}
		fmt.Printf("Generated ent/schema/%s.go\n", toSnake(ent.Name))
	}

	fmt.Printf("entgen: generated %d entity schemas\n", len(entities))
}

// parseEntities identifies domain entities in the CUE value.
// An entity is a CUE definition that has both "id" and "audit" fields.
func parseEntities(val cue.Value) map[string]*entityDef {
	entities := make(map[string]*entityDef)

	// Iterate over all definitions in the CUE package
	iter, _ := val.Fields(cue.Definitions(true))
	for iter.Next() {
		label := iter.Selector().String()
		defVal := iter.Value()

		// Check if this is an entity (has "id" and "audit" fields)
		// Note: Some entities with conditional constraints have IncompleteKind() == BottomKind,
		// so we check for id/audit fields directly instead of filtering by struct kind.
		idField := defVal.LookupPath(cue.ParsePath("id"))
		auditField := defVal.LookupPath(cue.ParsePath("audit"))
		if idField.Err() != nil || auditField.Err() != nil {
			continue
		}

		name := strings.TrimPrefix(label, "#")
		ent := &entityDef{
			Name:      name,
			Immutable: immutableEntities[name],
		}

		// Parse fields
		ent.Fields = parseFields(name, defVal)

		entities[name] = ent
	}

	return entities
}

// parseFields extracts field definitions from a CUE struct value.
func parseFields(entityName string, structVal cue.Value) []fieldDef {
	var fields []fieldDef

	iter, _ := structVal.Fields(cue.Optional(true))
	for iter.Next() {
		label := iter.Selector().String()
		fieldVal := iter.Value()

		// Strip optional marker from label if present
		label = strings.TrimSuffix(label, "?")

		// Skip "id" (handled as UUID PK), "audit" (handled by mixin), and constraint blocks
		if label == "id" || label == "audit" {
			continue
		}

		// Skip hidden fields (start with _)
		if strings.HasPrefix(label, "_") {
			continue
		}

		// Detect the CUE type reference
		fd := classifyField(label, fieldVal, iter.IsOptional())
		if fd != nil {
			fields = append(fields, *fd)
		}
	}

	return fields
}

// findReference recursively searches a CUE value expression tree for references
// to known types (like #Money, #Address, time.Time).
func findReference(val cue.Value) string {
	// Direct reference
	_, path := val.ReferencePath()
	if path.String() != "" {
		selectors := path.Selectors()
		if len(selectors) > 0 {
			return selectors[len(selectors)-1].String()
		}
	}

	// Check expression tree for references within unifications
	op, args := val.Expr()
	if op == cue.AndOp || op == cue.OrOp {
		for _, arg := range args {
			if ref := findReference(arg); ref != "" {
				return ref
			}
		}
	}
	// Check for time.Time specifically: it shows as a selector on time package
	if op == cue.SelectorOp && len(args) >= 2 {
		if s, err := args[1].String(); err == nil && s == "Time" {
			return "time.Time"
		}
	}
	return ""
}

// isTimeField checks if a CUE value represents a time.Time field.
func isTimeField(val cue.Value) bool {
	ref := findReference(val)
	return ref == "time.Time" || ref == "Time"
}

// classifyField determines the Ent field type for a CUE value.
func classifyField(name string, val cue.Value, optional bool) *fieldDef {
	fd := &fieldDef{
		Name:     name,
		Optional: optional,
		Nillable: optional,
	}

	// Check for time.Time first (from CUE time import)
	if isTimeField(val) {
		fd.EntType = "Time"
		return fd
	}

	// Check for references to known types
	refName := findReference(val)
	if refName != "" {
		// Money fields at top level get flattened
		if moneyFieldNames[refName] {
			return flattenMoney(name, optional, refName)
		}

		// Known value types become JSON fields
		if goType, ok := knownValueTypes[refName]; ok {
			fd.EntType = "JSON"
			if isList(val) {
				fd.JSONType = "[]" + goType + "{}"
			} else {
				fd.JSONType = "&" + goType + "{}"
			}
			return fd
		}
	}

	// Check for list types
	if isList(val) {
		return classifyListField(name, val, optional)
	}

	// Check for disjunction (enum) — must check before string since enums are string-like
	if isEnum(val) {
		fd.EntType = "Enum"
		fd.EnumValues = extractEnumValues(val)
		// Check for default value on enum
		if d, ok := val.Default(); ok {
			if s, err := d.String(); err == nil {
				fd.Default = s
			}
		}
		return fd
	}

	// Classify by incomplete kind
	kind := val.IncompleteKind()

	// If kind is bottom (_|_), the field has conflicting constraints from
	// conditional blocks. Try to infer the actual type from the expression tree.
	if kind == cue.BottomKind {
		kind = inferKindFromExpr(val)
	}

	switch kind {
	case cue.StringKind:
		fd.EntType = "String"
		// Check for regex constraint
		if pattern := extractPattern(val); pattern != "" {
			fd.MatchPattern = pattern
		}
		// Check for non-empty constraint
		if hasNonEmpty(val) {
			fd.NotEmpty = true
		}

	case cue.IntKind:
		fd.EntType = "Int"
		// Check for constraints
		lo, hi := extractNumericBounds(val)
		if lo != "" {
			fd.Min = lo
		}
		if hi != "" {
			fd.Max = hi
		}
		if fd.Min == "0" {
			fd.NonNegative = true
		}

	case cue.FloatKind, cue.NumberKind:
		fd.EntType = "Float64"
		lo, hi := extractNumericBounds(val)
		if lo != "" {
			fd.Min = lo
		}
		if hi != "" {
			fd.Max = hi
		}

	case cue.BoolKind:
		fd.EntType = "Bool"
		// Check for default value
		if d, ok := val.Default(); ok {
			b, _ := d.Bool()
			if b {
				fd.Default = "true"
			} else {
				fd.Default = "false"
			}
		}

	case cue.ListKind:
		return classifyListField(name, val, optional)

	case cue.StructKind:
		// Struct without a known reference → JSON
		fd.EntType = "JSON"
		fd.JSONType = "json.RawMessage{}"

	default:
		// Skip fields we can't classify
		return nil
	}

	return fd
}

// inferKindFromExpr walks the expression tree to find the underlying type
// of a field that CUE reports as BottomKind due to conditional constraints.
func inferKindFromExpr(val cue.Value) cue.Kind {
	op, args := val.Expr()

	if op == cue.AndOp {
		for _, arg := range args {
			k := arg.IncompleteKind()
			if k != cue.BottomKind {
				return k
			}
			// Recurse
			if inferred := inferKindFromExpr(arg); inferred != cue.BottomKind {
				return inferred
			}
		}
	}

	if op == cue.OrOp {
		// For a disjunction, check the first arg
		if len(args) > 0 {
			for _, arg := range args {
				k := arg.IncompleteKind()
				if k != cue.BottomKind {
					return k
				}
			}
		}
	}

	// Check for NoOp with a single value
	if op == cue.NoOp && len(args) == 0 {
		// Try evaluating with Allow
		return val.IncompleteKind()
	}

	return cue.BottomKind
}

// flattenMoney converts a #Money field into two columns: _amount_cents and _currency.
func flattenMoney(name string, optional bool, moneyType string) *fieldDef {
	// We return nil here and the caller should handle the expansion.
	// For now, we return a special marker that the template expands.
	return &fieldDef{
		Name:     name,
		EntType:  "Money",
		Optional: optional,
		Nillable: optional,
		Comment:  fmt.Sprintf("Flattened from %s", moneyType),
	}
}

// classifyListField handles CUE list types.
func classifyListField(name string, val cue.Value, optional bool) *fieldDef {
	fd := &fieldDef{
		Name:     name,
		EntType:  "JSON",
		Optional: optional,
		Nillable: optional,
	}

	// Try to determine the element type.
	// First try direct lookup, then walk expression tree for BottomKind values.
	elemFound := false
	elemVal := val.LookupPath(cue.MakePath(cue.AnyIndex))
	if elemVal.Err() == nil {
		elemFound = true
	} else {
		// For BottomKind list values (from conditional constraints), try to find
		// the list in the expression tree
		op, args := val.Expr()
		if op == cue.AndOp {
			for _, arg := range args {
				if arg.IncompleteKind() == cue.ListKind || inferKindFromExpr(arg) == cue.ListKind {
					ev := arg.LookupPath(cue.MakePath(cue.AnyIndex))
					if ev.Err() == nil {
						elemVal = ev
						elemFound = true
						break
					}
				}
			}
		}
	}

	if elemFound {
		// Check for reference to a known type
		ref := findReference(elemVal)
		if ref != "" {
			if goType, ok := knownValueTypes[ref]; ok {
				fd.JSONType = "[]" + goType + "{}"
				return fd
			}
		}

		// Check if it's a simple string list (including enum strings)
		kind := elemVal.IncompleteKind()
		if kind == cue.BottomKind {
			kind = inferKindFromExpr(elemVal)
		}
		if kind == cue.StringKind {
			fd.JSONType = "[]string{}"
			return fd
		}
	}

	// Fallback to raw JSON
	fd.JSONType = "json.RawMessage{}"
	return fd
}

// isEnum checks if a CUE value is a disjunction of string literals.
func isEnum(val cue.Value) bool {
	op, args := val.Expr()
	if op != cue.OrOp {
		return false
	}
	for _, arg := range args {
		// Each arg should be a string literal, possibly with a default marker
		aOp, aArgs := arg.Expr()
		check := arg
		if aOp == cue.SelectorOp && len(aArgs) > 0 {
			check = aArgs[0]
		}
		if check.IncompleteKind() != cue.StringKind {
			return false
		}
		// Must be a concrete string
		if _, err := check.String(); err != nil {
			// Could be a default expression
			d, ok := check.Default()
			if ok {
				if _, err := d.String(); err != nil {
					return false
				}
			} else {
				return false
			}
		}
	}
	return len(args) >= 2
}

// extractEnumValues pulls string literal values from a disjunction.
func extractEnumValues(val cue.Value) []string {
	op, args := val.Expr()
	if op != cue.OrOp {
		return nil
	}
	var values []string
	for _, arg := range args {
		// Handle default markers
		if s, err := arg.String(); err == nil {
			values = append(values, s)
			continue
		}
		if d, ok := arg.Default(); ok {
			if s, err := d.String(); err == nil {
				values = append(values, s)
			}
		}
	}
	return values
}

// isList checks if a CUE value represents a list type.
func isList(val cue.Value) bool {
	return val.IncompleteKind() == cue.ListKind
}

// extractPattern extracts a regex pattern from a CUE string constraint.
func extractPattern(val cue.Value) string {
	op, args := val.Expr()
	if op == cue.AndOp {
		for _, arg := range args {
			if p := extractPattern(arg); p != "" {
				return p
			}
		}
	}
	if op == cue.RegexMatchOp && len(args) >= 2 {
		if s, err := args[1].String(); err == nil {
			return s
		}
	}
	// Check NoOp for single-value match
	if op == cue.NoOp || op == cue.SelectorOp {
		// Try to find a match constraint in the expression tree
		return ""
	}
	return ""
}

// hasNonEmpty checks if a string field has the !="" constraint.
func hasNonEmpty(val cue.Value) bool {
	op, args := val.Expr()
	if op == cue.AndOp {
		for _, arg := range args {
			if hasNonEmpty(arg) {
				return true
			}
		}
	}
	if op == cue.NotEqualOp && len(args) >= 2 {
		if s, err := args[1].String(); err == nil && s == "" {
			return true
		}
	}
	return false
}

// extractNumericBounds extracts min/max from numeric CUE constraints.
func extractNumericBounds(val cue.Value) (lo, hi string) {
	op, args := val.Expr()
	if op == cue.AndOp {
		for _, arg := range args {
			l, h := extractNumericBounds(arg)
			if l != "" {
				lo = l
			}
			if h != "" {
				hi = h
			}
		}
		return
	}
	switch op {
	case cue.GreaterThanEqualOp:
		if len(args) >= 2 {
			lo = fmt.Sprint(args[1])
		}
	case cue.GreaterThanOp:
		if len(args) >= 2 {
			lo = fmt.Sprint(args[1])
		}
	case cue.LessThanEqualOp:
		if len(args) >= 2 {
			hi = fmt.Sprint(args[1])
		}
	case cue.LessThanOp:
		if len(args) >= 2 {
			hi = fmt.Sprint(args[1])
		}
	}
	return
}

// parseRelationships reads the relationships list from CUE and assigns edges to entities.
func parseRelationships(val cue.Value, entities map[string]*entityDef) {
	relList := val.LookupPath(cue.ParsePath("relationships"))
	if relList.Err() != nil {
		log.Printf("warning: no relationships found: %v", relList.Err())
		return
	}

	iter, _ := relList.List()
	for iter.Next() {
		rel := iter.Value()

		from, _ := rel.LookupPath(cue.ParsePath("from")).String()
		to, _ := rel.LookupPath(cue.ParsePath("to")).String()
		edgeName, _ := rel.LookupPath(cue.ParsePath("edge_name")).String()
		cardinality, _ := rel.LookupPath(cue.ParsePath("cardinality")).String()
		required, _ := rel.LookupPath(cue.ParsePath("required")).Bool()
		semantic, _ := rel.LookupPath(cue.ParsePath("semantic")).String()
		inverseName, _ := rel.LookupPath(cue.ParsePath("inverse_name")).String()

		if from == "" || to == "" || edgeName == "" {
			continue
		}

		// Add "To" edge on the "from" entity
		if ent, ok := entities[from]; ok {
			edge := edgeDef{
				Name:    edgeName,
				Target:  to,
				Type:    "To",
				Comment: semantic,
			}
			switch cardinality {
			case "O2O":
				edge.Unique = true
			case "M2O":
				// M2O from the "from" perspective means this is actually a "From" edge
				// The "from" entity has many, the "to" entity has one
				// Actually: from->to with M2O means many "from" to one "to"
				// So from's perspective: edge.From(to).Ref(edgeName).Unique()
				// But we model it as To on "from" with Unique
				edge.Type = "To"
				edge.Unique = true
				edge.Required = required
			case "O2M":
				// Nothing special, To without Unique
			case "M2M":
				// Nothing special, To without Unique
			}
			if cardinality != "M2O" {
				edge.Required = required
			}
			ent.Edges = append(ent.Edges, edge)
		}

		// Add inverse edge on the "to" entity
		if ent, ok := entities[to]; ok {
			invEdge := edgeDef{
				Name:    inverseName,
				Target:  from,
				Comment: semantic + " (inverse)",
			}
			switch cardinality {
			case "O2O":
				invEdge.Type = "From"
				invEdge.Unique = true
				invEdge.RefName = edgeName
			case "O2M":
				invEdge.Type = "From"
				invEdge.Unique = true
				invEdge.RefName = edgeName
			case "M2O":
				invEdge.Type = "To"
				invEdge.Unique = false
			case "M2M":
				invEdge.Type = "From"
				invEdge.RefName = edgeName
			}
			ent.Edges = append(ent.Edges, invEdge)
		}
	}
}

// parseStateMachines reads state machine definitions from CUE.
func parseStateMachines(val cue.Value, entities map[string]*entityDef) {
	for entName, cueName := range entityTransitionMap {
		ent, ok := entities[entName]
		if !ok {
			continue
		}

		smVal := val.LookupPath(cue.ParsePath(cueName))
		if smVal.Err() != nil {
			continue
		}

		machine := make(map[string][]string)
		iter, _ := smVal.Fields()
		for iter.Next() {
			state := iter.Selector().String()
			transitions := iter.Value()

			var targets []string
			tIter, _ := transitions.List()
			for tIter.Next() {
				if s, err := tIter.Value().String(); err == nil {
					targets = append(targets, s)
				}
			}
			machine[state] = targets
		}

		if len(machine) > 0 {
			ent.HasMachine = true
			ent.Machine = machine
		}
	}
}

// removeFKFields handles the relationship between entity fields and edge foreign keys.
// When an entity has a field like "property_id" AND a "property" edge, there are two cases:
//  1. Simple name match (e.g., property_id matches edge "property"): remove the field,
//     Ent auto-generates the FK column from the edge.
//  2. Composite name match (e.g., applicant_person_id matches edge "applicant" → Person):
//     keep the field, add .Field() to the edge so they share the same column.
func removeFKFields(ent *entityDef) {
	if ent.EdgeField == nil {
		ent.EdgeField = make(map[string]string)
	}

	type fkMatch struct {
		fieldName string
		edgeIdx   int
		composite bool // true if matched via {edge}_{target}_id pattern
	}

	var matches []fkMatch
	matched := make(map[string]bool) // field name -> matched

	for i, e := range ent.Edges {
		if !e.Unique {
			continue
		}
		// Simple patterns: edge or target name + "_id"
		simple := []string{e.Name + "_id", toSnake(e.Target) + "_id"}
		// Composite pattern: {edge}_{target}_id
		composite := e.Name + "_" + toSnake(e.Target) + "_id"

		found := false
		for _, cand := range simple {
			for _, f := range ent.Fields {
				if f.Name == cand && !matched[f.Name] {
					matches = append(matches, fkMatch{fieldName: f.Name, edgeIdx: i, composite: false})
					matched[f.Name] = true
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			for _, f := range ent.Fields {
				if f.Name == composite && !matched[f.Name] {
					matches = append(matches, fkMatch{fieldName: f.Name, edgeIdx: i, composite: true})
					matched[f.Name] = true
					break
				}
			}
		}
	}

	// Apply matches
	removeFields := make(map[string]bool)
	bindFields := make(map[string]bool)
	for _, m := range matches {
		if m.composite {
			// Keep the field, bind edge to it via .Field()
			ent.EdgeField[ent.Edges[m.edgeIdx].Name] = m.fieldName
			ent.Edges[m.edgeIdx].FieldBinding = m.fieldName
			bindFields[m.fieldName] = true
		} else {
			// Remove the field, Ent creates the FK column from the edge
			removeFields[m.fieldName] = true
		}
	}

	var filtered []fieldDef
	for _, f := range ent.Fields {
		if removeFields[f.Name] {
			continue
		}
		if bindFields[f.Name] {
			// Change type to UUID since edge FK must match target's ID type
			f.EntType = "UUID"
			// Preserve original optionality from CUE
		}
		filtered = append(filtered, f)
	}
	ent.Fields = filtered
}

// assignConstraints attaches cross-field constraint hook code to entities that have them.
// Constraints are hardcoded from CUE ontology conditional blocks — they change rarely,
// and CUE vet catches any drift between the ontology and this map.
func assignConstraints(entities map[string]*entityDef) {
	for name, ent := range entities {
		code := buildConstraintCode(name)
		if code != "" {
			ent.HasConstraints = true
			ent.ConstraintHookCode = code
		}
	}
}

// buildConstraintCode returns pre-rendered Go source for cross-field constraint hooks,
// or empty string if the entity has no constraints.
func buildConstraintCode(entityName string) string {
	// Common helper code shared by all constraint functions.
	// getField returns the effective value: newly set in mutation or existing in DB.
	const helperGetField = `
			getField := func(name string) (interface{}, bool) {
				if v, ok := m.Field(name); ok {
					return v, true
				}
				if v, err := m.OldField(ctx, name); err == nil {
					return v, true
				}
				return nil, false
			}`

	// toString handles both string and *string (nillable fields return *string from OldField).
	const helperToString = `
			toString := func(v interface{}) string {
				if v == nil {
					return ""
				}
				switch s := v.(type) {
				case string:
					return s
				case *string:
					if s != nil {
						return *s
					}
					return ""
				}
				return fmt.Sprint(v)
			}`

	const helperToInt = `
			toInt := func(v interface{}) (int, bool) {
				switch i := v.(type) {
				case int:
					return i, true
				case *int:
					if i != nil {
						return *i, true
					}
				}
				return 0, false
			}`

	const helperToFloat = `
			toFloat := func(v interface{}) (float64, bool) {
				switch f := v.(type) {
				case float64:
					return f, true
				case *float64:
					if f != nil {
						return *f, true
					}
				}
				return 0, false
			}`

	wrap := func(name, helpers, checks string) string {
		return fmt.Sprintf(`

// Hooks returns cross-field constraint validation hooks.
// Generated from CUE ontology conditional blocks.
func (%s) Hooks() []ent.Hook {
	return []ent.Hook{
		validate%sConstraints(),
	}
}

func validate%sConstraints() ent.Hook {
	return func(next ent.Mutator) ent.Mutator {
		return ent.MutateFunc(func(ctx context.Context, m ent.Mutation) (ent.Value, error) {%s
%s
			return next.Mutate(ctx, m)
		})
	}
}`, name, name, name, helpers, checks)
	}

	switch entityName {
	case "Portfolio":
		return wrap("Portfolio",
			helperGetField+helperToString,
			`
			// requires_trust_accounting == true → trust_bank_account_id must be non-empty
			if v, ok := getField("requires_trust_accounting"); ok && fmt.Sprint(v) == "true" {
				tid, tidOk := getField("trust_bank_account_id")
				if !tidOk || toString(tid) == "" {
					return nil, fmt.Errorf("portfolio with requires_trust_accounting=true must have trust_bank_account_id set")
				}
			}`)

	case "Property":
		return wrap("Property",
			helperGetField+helperToString,
			`
			// single_family → total_spaces must be 1
			if v, ok := getField("property_type"); ok && fmt.Sprint(v) == "single_family" {
				if tu, ok := getField("total_spaces"); ok {
					if tuInt, isInt := tu.(int); isInt && tuInt != 1 {
						return nil, fmt.Errorf("single_family property must have total_spaces=1, got %d", tuInt)
					}
				}
			}
			// affordable_housing → compliance_programs must have ≥1 entry
			if v, ok := getField("property_type"); ok && fmt.Sprint(v) == "affordable_housing" {
				cp, cpOk := getField("compliance_programs")
				if !cpOk && m.Op().Is(ent.OpCreate) {
					return nil, fmt.Errorf("affordable_housing property must have at least one compliance_programs entry")
				}
				if cpOk {
					if list, isList := cp.([]string); isList && len(list) == 0 {
						return nil, fmt.Errorf("affordable_housing property must have at least one compliance_programs entry")
					}
				}
			}
			// rent_controlled == true → jurisdiction_id must be non-empty
			if v, ok := getField("rent_controlled"); ok && fmt.Sprint(v) == "true" {
				jid, jidOk := getField("jurisdiction_id")
				if !jidOk || toString(jid) == "" {
					return nil, fmt.Errorf("rent-controlled property must have jurisdiction_id set")
				}
			}
			// year_built < 1978 → requires_lead_disclosure must be true
			if v, ok := getField("year_built"); ok {
				if yb, isInt := v.(int); isInt && yb < 1978 {
					if rld, ok := getField("requires_lead_disclosure"); ok {
						if fmt.Sprint(rld) != "true" {
							return nil, fmt.Errorf("property built before 1978 must have requires_lead_disclosure=true")
						}
					}
				}
			}`)

	case "Space":
		return wrap("Space",
			helperGetField+helperToInt+helperToFloat,
			`
			// residential_unit → bedrooms and bathrooms must be set
			if v, ok := getField("space_type"); ok && fmt.Sprint(v) == "residential_unit" {
				if _, ok := getField("bedrooms"); !ok && m.Op().Is(ent.OpCreate) {
					return nil, fmt.Errorf("residential_unit space must have bedrooms set")
				}
				if _, ok := getField("bathrooms"); !ok && m.Op().Is(ent.OpCreate) {
					return nil, fmt.Errorf("residential_unit space must have bathrooms set")
				}
			}
			// parking or storage → bedrooms == 0 and bathrooms == 0
			if v, ok := getField("space_type"); ok {
				st := fmt.Sprint(v)
				if st == "parking" || st == "storage" {
					if bd, ok := getField("bedrooms"); ok {
						if bdInt, ok := toInt(bd); ok && bdInt != 0 {
							return nil, fmt.Errorf("%s space must have bedrooms=0, got %d", st, bdInt)
						}
					}
					if bt, ok := getField("bathrooms"); ok {
						if btFloat, ok := toFloat(bt); ok && btFloat != 0 {
							return nil, fmt.Errorf("%s space must have bathrooms=0, got %v", st, btFloat)
						}
					}
				}
			}`)

	default:
		return ""
	}
}

// generateSchema renders an Ent schema Go file for the given entity.
func generateSchema(projectRoot string, ent *entityDef) error {
	var buf bytes.Buffer

	tmpl, err := template.New("schema").Funcs(template.FuncMap{
		"toSnake":   toSnake,
		"toPascal":  toPascal,
		"toCamel":   toCamel,
		"hasJSON":    func(fields []fieldDef) bool { return fieldsHaveType(fields, "JSON") },
		"hasMoney":   func(fields []fieldDef) bool { return fieldsHaveType(fields, "Money") },
		"hasEnum":    func(fields []fieldDef) bool { return fieldsHaveType(fields, "Enum") },
		"hasTime":    func(fields []fieldDef) bool { return fieldsHaveType(fields, "Time") },
		"needsRegexp": func(fields []fieldDef) bool {
			for _, f := range fields {
				if f.EntType == "Money" || f.MatchPattern != "" {
					return true
				}
			}
			return false
		},
		"needsTypes": func(fields []fieldDef) bool {
			for _, f := range fields {
				if f.EntType == "JSON" && strings.Contains(f.JSONType, "types.") {
					return true
				}
			}
			return false
		},
		"needsJSON": func(fields []fieldDef) bool {
			for _, f := range fields {
				if f.EntType == "JSON" && strings.Contains(f.JSONType, "json.RawMessage") {
					return true
				}
			}
			return false
		},
		"sortedStates": func(m map[string][]string) []string {
			keys := make([]string, 0, len(m))
			for k := range m {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			return keys
		},
	}).Parse(schemaTemplate)
	if err != nil {
		return fmt.Errorf("parsing template: %w", err)
	}

	if err := tmpl.Execute(&buf, ent); err != nil {
		return fmt.Errorf("executing template: %w", err)
	}

	// Format the generated Go code
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		// Write unformatted for debugging
		outPath := filepath.Join(projectRoot, "ent", "schema", toSnake(ent.Name)+".go")
		os.WriteFile(outPath, buf.Bytes(), 0644)
		return fmt.Errorf("formatting generated code for %s: %w\nUnformatted code written to %s", ent.Name, err, outPath)
	}

	outPath := filepath.Join(projectRoot, "ent", "schema", toSnake(ent.Name)+".go")
	if err := os.WriteFile(outPath, formatted, 0644); err != nil {
		return fmt.Errorf("writing %s: %w", outPath, err)
	}

	return nil
}

func fieldsHaveType(fields []fieldDef, t string) bool {
	for _, f := range fields {
		if f.EntType == t {
			return true
		}
	}
	return false
}

// toSnake converts PascalCase to snake_case.
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

// toPascal converts snake_case to PascalCase.
func toPascal(s string) string {
	parts := strings.Split(s, "_")
	for i, p := range parts {
		if len(p) > 0 {
			// Handle common abbreviations
			upper := strings.ToUpper(p)
			switch upper {
			case "ID", "URL", "API", "HTTP", "SQL", "CAM", "HAP", "AMI", "GL", "NNN", "ACH", "SSN", "DBA":
				parts[i] = upper
			default:
				parts[i] = strings.ToUpper(p[:1]) + p[1:]
			}
		}
	}
	return strings.Join(parts, "")
}

// toCamel converts snake_case to camelCase.
func toCamel(s string) string {
	p := toPascal(s)
	if len(p) == 0 {
		return p
	}
	return strings.ToLower(p[:1]) + p[1:]
}

var matchPatternRe = regexp.MustCompile(`=~"([^"]+)"`)

const schemaTemplate = `// Code generated by cmd/entgen from CUE ontology. DO NOT EDIT.
package schema

import (
	{{- if .HasConstraints}}
	"context"
	"fmt"
	{{- end}}
	{{- if needsJSON .Fields}}
	"encoding/json"
	{{- end}}
	{{- if needsRegexp .Fields}}
	"regexp"
	{{- end}}

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	{{- if .Edges}}
	"entgo.io/ent/schema/edge"
	{{- end}}
	"entgo.io/ent/schema/mixin"
	"github.com/google/uuid"
	{{- if needsTypes .Fields}}
	"github.com/matthewbaird/ontology/internal/types"
	{{- end}}
)

// Ensure mixin import is used.
var _ mixin.Schema

// {{.Name}} holds the schema definition for the {{.Name}} entity.
type {{.Name}} struct {
	ent.Schema
}

// Mixin of the {{.Name}}.
func ({{.Name}}) Mixin() []ent.Mixin {
	return []ent.Mixin{
		AuditMixin{},
	}
}

// Fields of the {{.Name}}.
func ({{.Name}}) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New).Immutable().Comment("Primary key"),
{{- range .Fields}}
{{- if eq .EntType "Money"}}
		field.Int64("{{.Name}}_amount_cents"){{if .Optional}}.Optional().Nillable(){{end}}.Comment("{{.Name}} — amount in cents"),
		field.String("{{.Name}}_currency"){{if .Optional}}.Optional().Nillable(){{end}}.Default("USD").Match(regexp.MustCompile(` + "`" + `^[A-Z]{3}$` + "`" + `)).Comment("{{.Name}} — ISO 4217 currency code"),
{{- else if eq .EntType "String"}}
		field.String("{{.Name}}"){{if .Optional}}.Optional().Nillable(){{end}}{{if .NotEmpty}}.NotEmpty(){{end}}{{if .Sensitive}}.Sensitive(){{end}}{{if .MatchPattern}}.Match(regexp.MustCompile(` + "`" + `{{.MatchPattern}}` + "`" + `)){{end}}.SchemaType(map[string]string{"postgres": "varchar"}),
{{- else if eq .EntType "Int"}}
		field.Int("{{.Name}}"){{if .Optional}}.Optional().Nillable(){{end}}{{if .NonNegative}}.NonNegative(){{end}}{{if .Min}}.Min({{.Min}}){{end}}{{if .Max}}.Max({{.Max}}){{end}},
{{- else if eq .EntType "Int64"}}
		field.Int64("{{.Name}}"){{if .Optional}}.Optional().Nillable(){{end}},
{{- else if eq .EntType "Float64"}}
		field.Float("{{.Name}}"){{if .Optional}}.Optional().Nillable(){{end}},
{{- else if eq .EntType "Bool"}}
		field.Bool("{{.Name}}"){{if .Optional}}.Optional().Nillable(){{end}}{{if .Default}}.Default({{.Default}}){{end}},
{{- else if eq .EntType "Time"}}
		field.Time("{{.Name}}"){{if .Optional}}.Optional().Nillable(){{end}}{{if .Immutable}}.Immutable(){{end}},
{{- else if eq .EntType "Enum"}}
		field.Enum("{{.Name}}").Values({{range $i, $v := .EnumValues}}{{if $i}}, {{end}}"{{$v}}"{{end}}){{if .Optional}}.Optional().Nillable(){{end}}{{if .Default}}.Default("{{.Default}}"){{end}},
{{- else if eq .EntType "JSON"}}
		field.JSON("{{.Name}}", {{.JSONType}}){{if .Optional}}.Optional(){{end}},
{{- else if eq .EntType "UUID"}}
		field.UUID("{{.Name}}", uuid.UUID{}){{if .Optional}}.Optional().Nillable(){{end}},
{{- end}}
{{- end}}
	}
}

// Edges of the {{.Name}}.
func ({{.Name}}) Edges() []ent.Edge {
{{- if .Edges}}
	return []ent.Edge{
{{- range .Edges}}
{{- if eq .Type "To"}}
		edge.To("{{.Name}}", {{.Target}}.Type){{if .Unique}}.Unique(){{end}}{{if .Required}}.Required(){{end}}{{if .FieldBinding}}.Field("{{.FieldBinding}}"){{end}}.Comment("{{.Comment}}"),
{{- else}}
		edge.From("{{.Name}}", {{.Target}}.Type).Ref("{{.RefName}}"){{if .Unique}}.Unique(){{end}}.Comment("{{.Comment}}"),
{{- end}}
{{- end}}
	}
{{- else}}
	return nil
{{- end}}
}
{{- if .HasMachine}}

// Valid{{.Name}}Transitions defines the allowed state machine transitions.
// Generated from CUE ontology state_machines.cue.
var Valid{{.Name}}Transitions = map[string][]string{
{{- range $state := sortedStates .Machine}}
	"{{$state}}": { {{- range $i, $target := index $.Machine $state}}{{if $i}}, {{end}}"{{$target}}"{{end}} },
{{- end}}
}
{{- end}}
{{- if .HasConstraints}}
{{.ConstraintHookCode}}
{{- end}}
{{- if .Immutable}}

// Policy of the {{.Name}}.
// Immutable entity: updates and deletes are denied.
func ({{.Name}}) Policy() ent.Policy {
	return nil // Immutability enforced via hooks at handler level
}
{{- end}}
`
