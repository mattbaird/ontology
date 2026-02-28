// cmd/replgen generates the REPL schema registry and entity dispatchers
// from the CUE ontology definitions.
//
// It produces two files:
//   - internal/repl/schema/gen_registry.go: entity metadata (fields, edges, state machines)
//   - internal/repl/schema/gen_dispatch.go: per-entity QueryHandle implementations
package main

import (
	"bytes"
	"fmt"
	"go/format"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
	"unicode"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/load"
)

// ── Data structures ─────────────────────────────────────────────────────────

type entityInfo struct {
	Name         string // PascalCase: "Lease"
	PQLName      string // snake_case: "lease"
	Fields       []fieldInfo
	Edges        []edgeInfo
	HasMachine   bool
	Immutable    bool
	Machine      map[string][]string // status -> targets
	EnumFields   map[string][]string // field name -> enum values
}

type fieldInfo struct {
	Name       string // PQL name (snake_case): "lease_type"
	EntColumn  string // Ent column constant value: "lease_type"
	Type       string // "String", "Int", "Int64", "Float", "Bool", "Time", "Enum", "UUID", "JSON"
	Optional   bool
	Sensitive  bool
	EnumValues []string
}

type edgeInfo struct {
	Name        string // edge name: "lease_spaces"
	Target      string // target PQL name: "lease_space"
	WithMethod  string // PascalCase: "LeaseSpaces" (for With*)
	Cardinality string // "O2O", "O2M", "M2O", "M2M"
	Unique      bool
}

// Known value types that map to Money (flattened in Ent).
var moneyTypes = map[string]bool{
	"#Money": true, "#NonNegativeMoney": true, "#PositiveMoney": true,
}

// Known value types that map to JSON in Ent.
var jsonTypes = map[string]bool{
	"#Money": true, "#NonNegativeMoney": true, "#PositiveMoney": true,
	"#Address": true, "#ContactMethod": true, "#DateRange": true,
	"#EntityRef": true, "#RentScheduleEntry": true, "#RecurringCharge": true,
	"#LateFeePolicy": true, "#CAMTerms": true, "#TenantImprovement": true,
	"#RenewalOption": true, "#SubsidyTerms": true, "#AccountDimensions": true,
	"#JournalLine": true, "#RoleAttributes": true, "#TenantAttributes": true,
	"#OwnerAttributes": true, "#ManagerAttributes": true, "#GuarantorAttributes": true,
	"#UsageBasedCharge": true, "#PercentageRent": true, "#RentAdjustment": true,
	"#ExpansionRight": true, "#ContractionRight": true, "#CAMCategoryTerms": true,
}

// ── CUE field-level attributes ──────────────────────────────────────────────

type fieldAttrs struct {
	sensitive bool
}

func extractAttributes(v cue.Value) fieldAttrs {
	var fa fieldAttrs
	for _, name := range []string{"sensitive", "pii"} {
		a := v.Attribute(name)
		if a.Err() == nil {
			fa.sensitive = true
		}
	}
	return fa
}

// ── Main ────────────────────────────────────────────────────────────────────

func main() {
	log.SetFlags(0)
	log.SetPrefix("replgen: ")

	projectRoot := findProjectRoot()
	ctx := cuecontext.New()

	// Load ontology CUE
	insts := load.Instances([]string{"./ontology"}, &load.Config{Dir: projectRoot})
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

	// Parse entities
	entities := parseEntities(val)

	// Parse relationships
	parseRelationships(val, entities)

	// Parse state machines
	parseStateMachines(val, entities)

	// Sort entities deterministically
	var sorted []*entityInfo
	for _, e := range entities {
		sorted = append(sorted, e)
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].PQLName < sorted[j].PQLName
	})

	// Generate files
	schemaDir := filepath.Join(projectRoot, "internal", "repl", "schema")

	if err := generateRegistryFile(schemaDir, sorted); err != nil {
		log.Fatalf("generating registry: %v", err)
	}

	execDir := filepath.Join(projectRoot, "internal", "repl", "executor")
	if err := generateDispatchFile(execDir, sorted); err != nil {
		log.Fatalf("generating dispatch: %v", err)
	}

	fmt.Printf("replgen: generated registry + dispatchers for %d entities\n", len(sorted))
}

// ── Entity parsing (follows cmd/entgen pattern) ─────────────────────────────

func parseEntities(val cue.Value) map[string]*entityInfo {
	entities := make(map[string]*entityInfo)

	iter, _ := val.Fields(cue.Definitions(true))
	for iter.Next() {
		label := iter.Selector().String()
		defVal := iter.Value()

		// Entity detection: has "id" and "audit" fields
		idField := defVal.LookupPath(cue.ParsePath("id"))
		auditField := defVal.LookupPath(cue.ParsePath("audit"))
		if idField.Err() != nil || auditField.Err() != nil {
			continue
		}

		name := strings.TrimPrefix(label, "#")
		if name == "BaseEntity" || name == "StatefulEntity" || name == "ImmutableEntity" {
			continue
		}

		ent := &entityInfo{
			Name:       name,
			PQLName:    toSnake(name),
			EnumFields: make(map[string][]string),
		}

		// Parse fields
		ent.Fields = parseFields(defVal)

		// Build enum field map
		for _, f := range ent.Fields {
			if len(f.EnumValues) > 0 {
				ent.EnumFields[f.Name] = f.EnumValues
			}
		}

		// Check immutability via hidden _immutable field
		immField := defVal.LookupPath(cue.ParsePath("_immutable"))
		if immField.Err() == nil {
			ent.Immutable = true
		}

		entities[name] = ent
	}

	return entities
}

func parseFields(structVal cue.Value) []fieldInfo {
	var fields []fieldInfo

	iter, _ := structVal.Fields(cue.Optional(true))
	for iter.Next() {
		label := iter.Selector().String()
		fieldVal := iter.Value()

		label = strings.TrimSuffix(label, "?")

		// Skip id, audit, hidden fields
		if label == "id" || label == "audit" || strings.HasPrefix(label, "_") {
			continue
		}

		// Check for money fields — these get flattened to two fields
		refName := findReference(fieldVal)
		if moneyTypes[refName] {
			attrs := extractAttributes(fieldVal)
			optional := iter.IsOptional()
			fields = append(fields, fieldInfo{
				Name:      label + "_amount_cents",
				EntColumn: label + "_amount_cents",
				Type:      "Int64",
				Optional:  optional,
				Sensitive: attrs.sensitive,
			})
			fields = append(fields, fieldInfo{
				Name:      label + "_currency",
				EntColumn: label + "_currency",
				Type:      "String",
				Optional:  optional,
				Sensitive: attrs.sensitive,
			})
			continue
		}

		fi := classifyField(label, fieldVal, iter.IsOptional())
		if fi != nil {
			attrs := extractAttributes(fieldVal)
			fi.Sensitive = attrs.sensitive
			fields = append(fields, *fi)
		}
	}

	return fields
}

// classifyField returns nil for money fields (handled separately in parseFields).
func classifyField(name string, val cue.Value, optional bool) *fieldInfo {
	fi := &fieldInfo{
		Name:      name,
		EntColumn: name,
		Optional:  optional,
	}

	// Check for time.Time
	if isTimeField(val) {
		fi.Type = "Time"
		return fi
	}

	// Check for references to known types
	refName := findReference(val)
	if refName != "" {
		// Money fields are flattened; return nil to signal special handling
		if moneyTypes[refName] {
			return nil
		}
		// Other known types → JSON
		if jsonTypes[refName] {
			fi.Type = "JSON"
			return fi
		}
	}

	// Check for list types
	if isList(val) {
		fi.Type = "JSON"
		return fi
	}

	// Check for enums
	if isEnum(val) {
		fi.Type = "Enum"
		fi.EnumValues = extractEnumValues(val)
		return fi
	}

	kind := val.IncompleteKind()
	if kind == cue.BottomKind {
		kind = inferKindFromExpr(val)
	}

	switch kind {
	case cue.StringKind:
		fi.Type = "String"
	case cue.IntKind:
		fi.Type = "Int"
	case cue.FloatKind, cue.NumberKind:
		fi.Type = "Float"
	case cue.BoolKind:
		fi.Type = "Bool"
	case cue.ListKind:
		fi.Type = "JSON"
	case cue.StructKind:
		fi.Type = "JSON"
	default:
		if kind != 0 && kind != cue.BottomKind {
			fi.Type = "JSON"
			return fi
		}
		return nil
	}

	return fi
}

// ── CUE type detection helpers (mirrored from cmd/entgen) ───────────────────

func findReference(val cue.Value) string {
	_, path := val.ReferencePath()
	if path.String() != "" {
		selectors := path.Selectors()
		if len(selectors) > 0 {
			return selectors[len(selectors)-1].String()
		}
	}
	op, args := val.Expr()
	if op == cue.AndOp || op == cue.OrOp {
		for _, arg := range args {
			if ref := findReference(arg); ref != "" {
				return ref
			}
		}
	}
	if op == cue.SelectorOp && len(args) >= 2 {
		if s, err := args[1].String(); err == nil && s == "Time" {
			return "time.Time"
		}
	}
	return ""
}

func isTimeField(val cue.Value) bool {
	ref := findReference(val)
	return ref == "time.Time" || ref == "Time"
}

func isList(val cue.Value) bool {
	if val.IncompleteKind() == cue.ListKind {
		return true
	}
	op, args := val.Expr()
	if op == cue.AndOp {
		for _, arg := range args {
			if arg.IncompleteKind() == cue.ListKind {
				return true
			}
		}
	}
	_ = args
	return false
}

func isEnum(val cue.Value) bool {
	op, args := val.Expr()
	if op == cue.OrOp && len(args) >= 2 {
		for _, arg := range args {
			if arg.IncompleteKind() == cue.StringKind {
				if _, err := arg.String(); err == nil {
					return true
				}
			}
		}
	}
	// Check through AndOp wrapping (from embedded types)
	if op == cue.AndOp {
		for _, arg := range args {
			if isEnum(arg) {
				return true
			}
		}
	}
	return false
}

func extractEnumValues(val cue.Value) []string {
	return findEnumDisjunction(val)
}

func findEnumDisjunction(val cue.Value) []string {
	op, args := val.Expr()
	if op == cue.OrOp {
		var values []string
		for _, arg := range args {
			// Skip default markers
			if dop, _ := arg.Expr(); dop == cue.OrOp {
				continue
			}
			if s, err := arg.String(); err == nil {
				values = append(values, s)
			}
		}
		if len(values) > 0 {
			return values
		}
	}
	if op == cue.AndOp {
		for _, arg := range args {
			if vals := findEnumDisjunction(arg); len(vals) > 0 {
				return vals
			}
		}
	}
	return nil
}

func inferKindFromExpr(val cue.Value) cue.Kind {
	op, args := val.Expr()
	if op == cue.AndOp {
		for _, arg := range args {
			k := arg.IncompleteKind()
			if k != cue.BottomKind && k != 0 {
				return k
			}
		}
	}
	if op == cue.OrOp && len(args) > 0 {
		return args[0].IncompleteKind()
	}
	return cue.BottomKind
}

// ── Relationship parsing ────────────────────────────────────────────────────

func parseRelationships(val cue.Value, entities map[string]*entityInfo) {
	rels := val.LookupPath(cue.ParsePath("relationships"))
	if rels.Err() != nil {
		log.Printf("warning: no relationships found")
		return
	}

	relIter, _ := rels.List()
	for relIter.Next() {
		rel := relIter.Value()

		from, _ := rel.LookupPath(cue.ParsePath("from")).String()
		to, _ := rel.LookupPath(cue.ParsePath("to")).String()
		edgeName, _ := rel.LookupPath(cue.ParsePath("edge_name")).String()
		card, _ := rel.LookupPath(cue.ParsePath("cardinality")).String()
		inverseName, _ := rel.LookupPath(cue.ParsePath("inverse_name")).String()

		// Add edge to "from" entity
		if ent, ok := entities[from]; ok {
			unique := card == "O2O" || card == "M2O"
			ent.Edges = append(ent.Edges, edgeInfo{
				Name:        edgeName,
				Target:      toSnake(to),
				WithMethod:  toPascal(edgeName),
				Cardinality: card,
				Unique:      unique,
			})
		}

		// Add inverse edge to "to" entity
		if ent, ok := entities[to]; ok {
			inverseCard := invertCardinality(card)
			unique := inverseCard == "O2O" || inverseCard == "M2O"
			ent.Edges = append(ent.Edges, edgeInfo{
				Name:        inverseName,
				Target:      toSnake(from),
				WithMethod:  toPascal(inverseName),
				Cardinality: inverseCard,
				Unique:      unique,
			})
		}
	}
}

func invertCardinality(card string) string {
	switch card {
	case "O2M":
		return "M2O"
	case "M2O":
		return "O2M"
	default:
		return card // O2O and M2M are symmetric
	}
}

// ── State machine parsing ───────────────────────────────────────────────────

func parseStateMachines(val cue.Value, entities map[string]*entityInfo) {
	for _, ent := range entities {
		smVal := val.LookupPath(cue.ParsePath("#StateMachines." + ent.PQLName))
		if smVal.Err() != nil {
			continue
		}

		ent.HasMachine = true
		ent.Machine = make(map[string][]string)

		smIter, _ := smVal.Fields()
		for smIter.Next() {
			fromStatus := smIter.Selector().String()
			var targets []string
			targetIter, _ := smIter.Value().List()
			for targetIter.Next() {
				if s, err := targetIter.Value().String(); err == nil {
					targets = append(targets, s)
				}
			}
			ent.Machine[fromStatus] = targets
		}
	}
}

// ── Code generation ─────────────────────────────────────────────────────────

func generateRegistryFile(dir string, entities []*entityInfo) error {
	tmpl := template.Must(template.New("registry").Funcs(template.FuncMap{
		"quote":    func(s string) string { return fmt.Sprintf("%q", s) },
		"fieldType": mapFieldType,
	}).Parse(registryTemplate))

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, entities); err != nil {
		return fmt.Errorf("template: %w", err)
	}

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		// Write unformatted for debugging
		os.WriteFile(filepath.Join(dir, "gen_registry.go"), buf.Bytes(), 0o644)
		return fmt.Errorf("gofmt: %w (wrote unformatted output for debugging)", err)
	}

	return os.WriteFile(filepath.Join(dir, "gen_registry.go"), formatted, 0o644)
}

func generateDispatchFile(dir string, entities []*entityInfo) error {
	tmpl := template.Must(template.New("dispatch").Funcs(template.FuncMap{
		"lower":    strings.ToLower,
		"pascal":   toPascal,
		"toPascal": toPascal,
		"quote":    func(s string) string { return fmt.Sprintf("%q", s) },
	}).Parse(dispatchTemplate))

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, entities); err != nil {
		return fmt.Errorf("template: %w", err)
	}

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		os.WriteFile(filepath.Join(dir, "gen_dispatch.go"), buf.Bytes(), 0o644)
		return fmt.Errorf("gofmt: %w (wrote unformatted output for debugging)", err)
	}

	return os.WriteFile(filepath.Join(dir, "gen_dispatch.go"), formatted, 0o644)
}

func mapFieldType(t string) string {
	switch t {
	case "String":
		return "FieldString"
	case "Int":
		return "FieldInt"
	case "Int64":
		return "FieldInt64"
	case "Float":
		return "FieldFloat"
	case "Bool":
		return "FieldBool"
	case "Time":
		return "FieldTime"
	case "Enum":
		return "FieldEnum"
	case "UUID":
		return "FieldUUID"
	case "JSON":
		return "FieldJSON"
	default:
		return "FieldString"
	}
}

// ── String conversion helpers ───────────────────────────────────────────────

func toSnake(s string) string {
	var result []rune
	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 {
				result = append(result, '_')
			}
			result = append(result, unicode.ToLower(r))
		} else {
			result = append(result, r)
		}
	}
	return string(result)
}

func toPascal(s string) string {
	parts := strings.Split(s, "_")
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, "")
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
			log.Fatal("could not find project root (no go.mod found)")
		}
		dir = parent
	}
}

// ── Templates ───────────────────────────────────────────────────────────────

var registryTemplate = `// Code generated by cmd/replgen from CUE ontology. DO NOT EDIT.
package schema

// InitRegistry populates the registry with all entity metadata.
func InitRegistry() *Registry {
	r := NewRegistry()
{{range .}}
	r.Register(&EntitySchema{
		Name:    {{quote .PQLName}},
		EntName: {{quote .Name}},
		Fields: map[string]*FieldMeta{
{{- range .Fields}}
			{{quote .Name}}: {
				Name:      {{quote .Name}},
				EntColumn: {{quote .EntColumn}},
				Type:      {{fieldType .Type}},
				Optional:  {{.Optional}},
				Sensitive: {{.Sensitive}},
{{- if .EnumValues}}
				EnumValues: []string{ {{- range $i, $v := .EnumValues}}{{if $i}}, {{end}}{{quote $v}}{{end -}} },
{{- end}}
			},
{{- end}}
		},
		FieldOrder: []string{
{{- range .Fields}}
			{{quote .Name}},
{{- end}}
		},
		Edges: map[string]*EdgeMeta{
{{- range .Edges}}
			{{quote .Name}}: {
				Name:        {{quote .Name}},
				Target:      {{quote .Target}},
				Cardinality: {{quote .Cardinality}},
				Unique:      {{.Unique}},
			},
{{- end}}
		},
		EdgeOrder: []string{
{{- range .Edges}}
			{{quote .Name}},
{{- end}}
		},
		HasStateMachine: {{.HasMachine}},
		Immutable:       {{.Immutable}},
{{- if .HasMachine}}
		StateMachine: map[string][]string{
{{- range $from, $targets := .Machine}}
			{{quote $from}}: { {{- range $i, $t := $targets}}{{if $i}}, {{end}}{{quote $t}}{{end -}} },
{{- end}}
		},
{{- end}}
	})
{{end}}
	return r
}
`

var dispatchTemplate = `// Code generated by cmd/replgen from CUE ontology. DO NOT EDIT.
package executor

import (
	"context"
	"fmt"

	"entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	"github.com/matthewbaird/ontology/ent"
	"github.com/matthewbaird/ontology/ent/predicate"
	"github.com/matthewbaird/ontology/internal/repl/planner"
{{- range .}}
	"github.com/matthewbaird/ontology/ent/{{lower .Name}}"
{{- end}}
)

// Ensure imports are used.
var (
	_ predicate.Account
	_ sql.Selector
	_ = fmt.Errorf
)

// InitDispatchers creates and registers dispatchers for all entities.
func InitDispatchers() *DispatchRegistry {
	dr := NewDispatchRegistry()
{{range .}}	dr.Register({{quote .PQLName}}, &{{lower .Name}}Dispatcher{})
{{end}}	return dr
}

// buildSQLPredicate converts a PredicateSpec to a raw SQL selector predicate.
func buildSQLPredicate(spec planner.PredicateSpec) func(*sql.Selector) {
	switch spec.Op {
	case planner.OpEQ:
		return sql.FieldEQ(spec.Field, spec.Value)
	case planner.OpNEQ:
		return sql.FieldNEQ(spec.Field, spec.Value)
	case planner.OpGT:
		return sql.FieldGT(spec.Field, spec.Value)
	case planner.OpLT:
		return sql.FieldLT(spec.Field, spec.Value)
	case planner.OpGTE:
		return sql.FieldGTE(spec.Field, spec.Value)
	case planner.OpLTE:
		return sql.FieldLTE(spec.Field, spec.Value)
	case planner.OpIn:
		return func(s *sql.Selector) {
			s.Where(sql.In(s.C(spec.Field), spec.Values...))
		}
	case planner.OpLike:
		if v, ok := spec.Value.(string); ok {
			return func(s *sql.Selector) {
				s.Where(sql.Like(s.C(spec.Field), v))
			}
		}
		return func(*sql.Selector) {} // no-op for non-string
	default:
		return func(*sql.Selector) {} // no-op
	}
}
{{range .}}
// ── {{.Name}} ───────────────────────────────────────────────────────────────

type {{lower .Name}}Dispatcher struct{}

func (d *{{lower .Name}}Dispatcher) Query(client *ent.Client) QueryHandle {
	return &{{lower .Name}}QueryHandle{q: client.{{.Name}}.Query()}
}

func (d *{{lower .Name}}Dispatcher) Get(ctx context.Context, client *ent.Client, id uuid.UUID) (any, error) {
	return client.{{.Name}}.Get(ctx, id)
}

type {{lower .Name}}QueryHandle struct {
	q *ent.{{.Name}}Query
}

func (h *{{lower .Name}}QueryHandle) Where(specs ...planner.PredicateSpec) QueryHandle {
	for _, spec := range specs {
		p := buildSQLPredicate(spec)
		h.q = h.q.Where(predicate.{{.Name}}(p))
	}
	return h
}

func (h *{{lower .Name}}QueryHandle) WithEdge(name string) QueryHandle {
	switch name {
{{- range .Edges}}
	case {{quote .Name}}:
		h.q = h.q.With{{.WithMethod}}()
{{- end}}
	}
	return h
}

func (h *{{lower .Name}}QueryHandle) OrderBy(field string, desc bool) QueryHandle {
	var opts []sql.OrderTermOption
	if desc {
		opts = append(opts, sql.OrderDesc())
	}
	h.q = h.q.Order({{lower .Name}}.OrderOption(sql.OrderByField(field, opts...).ToFunc()))
	return h
}

func (h *{{lower .Name}}QueryHandle) Limit(n int) QueryHandle {
	h.q = h.q.Limit(n)
	return h
}

func (h *{{lower .Name}}QueryHandle) Offset(n int) QueryHandle {
	h.q = h.q.Offset(n)
	return h
}

func (h *{{lower .Name}}QueryHandle) All(ctx context.Context) ([]any, error) {
	results, err := h.q.All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]any, len(results))
	for i, r := range results {
		out[i] = r
	}
	return out, nil
}

func (h *{{lower .Name}}QueryHandle) Count(ctx context.Context) (int, error) {
	return h.q.Count(ctx)
}
{{- if not .Immutable}}

func (d *{{lower .Name}}Dispatcher) Create(ctx context.Context, client *ent.Client, fields map[string]any) (any, error) {
	builder := client.{{.Name}}.Create()
	m := builder.Mutation()
	for name, val := range fields {
		if err := m.SetField(name, val); err != nil {
			return nil, fmt.Errorf("set %s: %w", name, err)
		}
	}
	return builder.Save(ctx)
}

func (d *{{lower .Name}}Dispatcher) Update(ctx context.Context, client *ent.Client, id uuid.UUID, fields map[string]any) (any, error) {
	builder := client.{{.Name}}.UpdateOneID(id)
	m := builder.Mutation()
	for name, val := range fields {
		if err := m.SetField(name, val); err != nil {
			return nil, fmt.Errorf("set %s: %w", name, err)
		}
	}
	return builder.Save(ctx)
}

func (d *{{lower .Name}}Dispatcher) Delete(ctx context.Context, client *ent.Client, id uuid.UUID) error {
	return client.{{.Name}}.DeleteOneID(id).Exec(ctx)
}
{{- end}}
{{end}}`
