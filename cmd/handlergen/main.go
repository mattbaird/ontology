// Code generated handler generator — reads CUE ontology + apigen.cue to produce
// HTTP handler code and route registration. DO NOT EDIT individual handler files;
// edit the CUE ontology and re-run this generator.
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

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/load"
)

// ─── Data types ──────────────────────────────────────────────────────────────

type fieldDef struct {
	Name       string
	EntType    string // String, Int, Int64, Float64, Bool, Time, Enum, JSON, Money
	Optional   bool
	JSONType   string // Go type for JSON fields
	Default    string
}

type edgeDef struct {
	Name   string
	Target string
	Type   string // To or From
	Unique bool
}

type edgeFK struct {
	FieldName string // e.g. "unit_id"
	EdgeName  string // e.g. "unit"
	Target    string // e.g. "Unit"
	Optional  bool
}

type entityInfo struct {
	Name       string
	Fields     []fieldDef
	EdgeFKs    []edgeFK
	HasMachine bool
}

type serviceDef struct {
	Name       string
	Entities   []string
	Operations []operationDef
}

type operationDef struct {
	Name        string
	Entity      string
	Type        string // create, get, list, update, transition
	EntityPath  string
	Action      string
	ToStatus    string
	ExtraFields []string
	Custom      bool
}

// ─── Known types ─────────────────────────────────────────────────────────────

var moneyFieldNames = map[string]bool{
	"#Money": true, "#NonNegativeMoney": true, "#PositiveMoney": true,
}

var knownValueTypes = map[string]string{
	"#Money": "types.Money", "#NonNegativeMoney": "types.Money", "#PositiveMoney": "types.Money",
	"#Address": "types.Address", "#ContactMethod": "types.ContactMethod",
	"#DateRange": "types.DateRange", "#EntityRef": "types.EntityRef",
	"#RentScheduleEntry": "types.RentScheduleEntry", "#RecurringCharge": "types.RecurringCharge",
	"#LateFeePolicy": "types.LateFeePolicy", "#CAMTerms": "types.CAMTerms",
	"#TenantImprovement": "types.TenantImprovement", "#RenewalOption": "types.RenewalOption",
	"#SubsidyTerms": "types.SubsidyTerms", "#AccountDimensions": "types.AccountDimensions",
	"#JournalLine": "types.JournalLine", "#RoleAttributes": "json.RawMessage",
	"#TenantAttributes": "types.TenantAttributes", "#OwnerAttributes": "types.OwnerAttributes",
	"#ManagerAttributes": "types.ManagerAttributes", "#GuarantorAttributes": "types.GuarantorAttributes",
	"#UsageBasedCharge": "types.UsageBasedCharge", "#PercentageRent": "types.PercentageRent",
	"#RentAdjustment": "types.RentAdjustment", "#ExpansionRight": "types.ExpansionRight",
	"#ContractionRight": "types.ContractionRight", "#CAMCategoryTerms": "types.CAMCategoryTerms",
}

var entityTransitionMap = map[string]string{
	"Lease": "#LeaseTransitions", "Space": "#SpaceTransitions",
	"Building": "#BuildingTransitions",
	"Application": "#ApplicationTransitions", "JournalEntry": "#JournalEntryTransitions",
	"Portfolio": "#PortfolioTransitions", "Property": "#PropertyTransitions",
	"PersonRole": "#PersonRoleTransitions", "Organization": "#OrganizationTransitions",
	"BankAccount": "#BankAccountTransitions", "Reconciliation": "#ReconciliationTransitions",
}

// goInitialisms matches Ent's PascalCase behavior (only standard Go initialisms).
var goInitialisms = map[string]bool{
	"acl": true, "api": true, "ascii": true, "cpu": true, "css": true,
	"dns": true, "eof": true, "guid": true, "html": true, "http": true,
	"https": true, "id": true, "ip": true, "json": true, "lhs": true,
	"qps": true, "ram": true, "rhs": true, "rpc": true, "sla": true,
	"smtp": true, "sql": true, "ssh": true, "tcp": true, "tls": true,
	"ttl": true, "udp": true, "ui": true, "uid": true, "uuid": true,
	"uri": true, "url": true, "utf8": true, "vm": true, "xml": true,
	"xmpp": true, "xsrf": true, "xss": true,
}

// ─── CUE Parsing ────────────────────────────────────────────────────────────

func findProjectRoot() string {
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			log.Fatal("no go.mod found")
		}
		dir = parent
	}
}

func findReference(val cue.Value) string {
	_, path := val.ReferencePath()
	if path.String() != "" {
		sels := path.Selectors()
		if len(sels) > 0 {
			return sels[len(sels)-1].String()
		}
	}
	op, args := val.Expr()
	if op == cue.AndOp || op == cue.OrOp {
		for _, a := range args {
			if r := findReference(a); r != "" {
				return r
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
	r := findReference(val)
	return r == "time.Time" || r == "Time"
}

func isEnum(val cue.Value) bool {
	op, args := val.Expr()
	if op != cue.OrOp || len(args) < 2 {
		return false
	}
	for _, a := range args {
		check := a
		aOp, aArgs := a.Expr()
		if aOp == cue.SelectorOp && len(aArgs) > 0 {
			check = aArgs[0]
		}
		if check.IncompleteKind() != cue.StringKind {
			return false
		}
		if _, err := check.String(); err != nil {
			if d, ok := check.Default(); ok {
				if _, err := d.String(); err != nil {
					return false
				}
			} else {
				return false
			}
		}
	}
	return true
}

func inferKindFromExpr(val cue.Value) cue.Kind {
	op, args := val.Expr()
	if op == cue.AndOp || op == cue.OrOp {
		for _, a := range args {
			if k := a.IncompleteKind(); k != cue.BottomKind {
				return k
			}
			if k := inferKindFromExpr(a); k != cue.BottomKind {
				return k
			}
		}
	}
	return cue.BottomKind
}

// inferListElementKind attempts to determine the element kind of a CUE list
// by walking its expression tree. For lists like [...("A" | "B")], the element
// constraint may not be reachable via LookupPath when the value is an
// intersection (AndOp) of list constraints from conditional blocks.
func inferListElementKind(val cue.Value) cue.Kind {
	// Walk expression arguments looking for resolvable list elements
	op, args := val.Expr()
	if op == cue.AndOp || op == cue.OrOp {
		for _, a := range args {
			// Try element lookup on each arg
			elem := a.LookupPath(cue.MakePath(cue.AnyIndex))
			if elem.Err() == nil {
				k := elem.IncompleteKind()
				if k != cue.BottomKind {
					return k
				}
			}
			// Recurse
			if k := inferListElementKind(a); k != cue.BottomKind {
				return k
			}
		}
	}
	return cue.BottomKind
}

func classifyField(name string, val cue.Value, optional bool) *fieldDef {
	fd := &fieldDef{Name: name, Optional: optional}

	if isTimeField(val) {
		fd.EntType = "Time"
		return fd
	}

	ref := findReference(val)
	if ref != "" {
		if moneyFieldNames[ref] {
			return &fieldDef{Name: name, EntType: "Money", Optional: optional}
		}
		if goType, ok := knownValueTypes[ref]; ok {
			fd.EntType = "JSON"
			if val.IncompleteKind() == cue.ListKind {
				fd.JSONType = "[]" + goType
			} else {
				fd.JSONType = goType
			}
			return fd
		}
	}

	if val.IncompleteKind() == cue.ListKind || inferKindFromExpr(val) == cue.ListKind {
		fd.EntType = "JSON"
		elem := val.LookupPath(cue.MakePath(cue.AnyIndex))
		if elem.Err() == nil {
			eRef := findReference(elem)
			if eRef != "" {
				if goType, ok := knownValueTypes[eRef]; ok {
					fd.JSONType = "[]" + goType
					return fd
				}
			}
			if elem.IncompleteKind() == cue.StringKind || isEnum(elem) {
				fd.JSONType = "[]string"
				return fd
			}
		}
		// Fallback: walk the expression tree to find element type
		if elemKind := inferListElementKind(val); elemKind == cue.StringKind {
			fd.JSONType = "[]string"
			return fd
		}
		fd.JSONType = "json.RawMessage"
		return fd
	}

	if isEnum(val) {
		fd.EntType = "Enum"
		if d, ok := val.Default(); ok {
			if s, err := d.String(); err == nil {
				fd.Default = s
			}
		}
		return fd
	}

	kind := val.IncompleteKind()
	if kind == cue.BottomKind {
		kind = inferKindFromExpr(val)
	}

	switch kind {
	case cue.StringKind:
		fd.EntType = "String"
	case cue.IntKind:
		fd.EntType = "Int"
	case cue.FloatKind, cue.NumberKind:
		fd.EntType = "Float64"
	case cue.BoolKind:
		fd.EntType = "Bool"
	default:
		return nil
	}
	return fd
}

func parseEntities(val cue.Value) map[string]*entityInfo {
	entities := make(map[string]*entityInfo)
	iter, _ := val.Fields(cue.Definitions(true))
	for iter.Next() {
		label := iter.Selector().String()
		defVal := iter.Value()
		if defVal.LookupPath(cue.ParsePath("id")).Err() != nil {
			continue
		}
		if defVal.LookupPath(cue.ParsePath("audit")).Err() != nil {
			continue
		}
		name := strings.TrimPrefix(label, "#")
		ent := &entityInfo{Name: name}
		fIter, _ := defVal.Fields(cue.Optional(true))
		for fIter.Next() {
			fLabel := strings.TrimSuffix(fIter.Selector().String(), "?")
			if fLabel == "id" || fLabel == "audit" || strings.HasPrefix(fLabel, "_") {
				continue
			}
			fd := classifyField(fLabel, fIter.Value(), fIter.IsOptional())
			if fd != nil {
				ent.Fields = append(ent.Fields, *fd)
			}
		}
		entities[name] = ent
	}
	return entities
}

func parseRelationships(val cue.Value, entities map[string]*entityInfo) {
	relList := val.LookupPath(cue.ParsePath("relationships"))
	if relList.Err() != nil {
		return
	}
	iter, _ := relList.List()
	for iter.Next() {
		rel := iter.Value()
		from, _ := rel.LookupPath(cue.ParsePath("from")).String()
		to, _ := rel.LookupPath(cue.ParsePath("to")).String()
		edgeName, _ := rel.LookupPath(cue.ParsePath("edge_name")).String()
		cardinality, _ := rel.LookupPath(cue.ParsePath("cardinality")).String()
		inverseName, _ := rel.LookupPath(cue.ParsePath("inverse_name")).String()
		if from == "" || to == "" || edgeName == "" {
			continue
		}
		if ent, ok := entities[from]; ok {
			e := edgeDef{Name: edgeName, Target: to, Type: "To"}
			if cardinality == "O2O" || cardinality == "M2O" {
				e.Unique = true
			}
			ent.EdgeFKs = appendEdge(ent.EdgeFKs, ent.Fields, e)
		}
		if ent, ok := entities[to]; ok {
			e := edgeDef{Name: inverseName, Target: from}
			switch cardinality {
			case "O2O", "O2M":
				e.Type = "From"
				e.Unique = true
			case "M2O":
				e.Type = "To"
			case "M2M":
				e.Type = "From"
			}
			ent.EdgeFKs = appendEdge(ent.EdgeFKs, ent.Fields, e)
		}
	}
	// Remove FK fields from entity field lists
	for _, ent := range entities {
		fkNames := make(map[string]bool)
		for _, efk := range ent.EdgeFKs {
			fkNames[efk.FieldName] = true
		}
		var filtered []fieldDef
		for _, f := range ent.Fields {
			if !fkNames[f.Name] {
				filtered = append(filtered, f)
			}
		}
		ent.Fields = filtered
	}
}

func appendEdge(fks []edgeFK, fields []fieldDef, e edgeDef) []edgeFK {
	if !e.Unique {
		return fks
	}
	// Try multiple FK naming conventions:
	// 1. {edge_name}_id (e.g., "unit_id" for edge "unit")
	// 2. {target_snake}_id (e.g., "person_id" for target "Person")
	// 3. {edge_name}_{target_snake}_id (e.g., "applicant_person_id" for edge "applicant" → "Person")
	candidates := []string{
		e.Name + "_id",
		toSnake(e.Target) + "_id",
		e.Name + "_" + toSnake(e.Target) + "_id",
	}
	for _, f := range fields {
		for _, cand := range candidates {
			if f.Name == cand {
				// Deduplicate: skip if this field is already tracked
				for _, existing := range fks {
					if existing.FieldName == f.Name {
						return fks
					}
				}
				fks = append(fks, edgeFK{
					FieldName: f.Name,
					EdgeName:  e.Name,
					Target:    e.Target,
					Optional:  f.Optional,
				})
				return fks
			}
		}
	}
	return fks
}

func parseStateMachines(val cue.Value, entities map[string]*entityInfo) {
	for entName, cueName := range entityTransitionMap {
		ent, ok := entities[entName]
		if !ok {
			continue
		}
		smVal := val.LookupPath(cue.ParsePath(cueName))
		if smVal.Err() != nil {
			continue
		}
		iter, _ := smVal.Fields()
		for iter.Next() {
			ent.HasMachine = true
			break
		}
	}
}

func parseServices(ctx *cue.Context, projectRoot string) []serviceDef {
	insts := load.Instances([]string{"./codegen"}, &load.Config{Dir: projectRoot})
	if len(insts) == 0 || insts[0].Err != nil {
		log.Fatalf("loading codegen: %v", insts[0].Err)
	}
	val := ctx.BuildInstance(insts[0])
	if val.Err() != nil {
		log.Fatalf("building codegen: %v", val.Err())
	}
	svcList := val.LookupPath(cue.ParsePath("services"))
	if svcList.Err() != nil {
		log.Fatalf("no services in codegen: %v", svcList.Err())
	}
	var services []serviceDef
	iter, _ := svcList.List()
	for iter.Next() {
		s := iter.Value()
		svc := serviceDef{}
		svc.Name, _ = s.LookupPath(cue.ParsePath("name")).String()
		entList := s.LookupPath(cue.ParsePath("entities"))
		eIter, _ := entList.List()
		for eIter.Next() {
			e, _ := eIter.Value().String()
			svc.Entities = append(svc.Entities, e)
		}
		opList := s.LookupPath(cue.ParsePath("operations"))
		oIter, _ := opList.List()
		for oIter.Next() {
			o := oIter.Value()
			op := operationDef{}
			op.Name, _ = o.LookupPath(cue.ParsePath("name")).String()
			op.Entity, _ = o.LookupPath(cue.ParsePath("entity")).String()
			op.Type, _ = o.LookupPath(cue.ParsePath("type")).String()
			op.EntityPath, _ = o.LookupPath(cue.ParsePath("entity_path")).String()
			op.Action, _ = o.LookupPath(cue.ParsePath("action")).String()
			op.ToStatus, _ = o.LookupPath(cue.ParsePath("to_status")).String()
			op.Custom, _ = o.LookupPath(cue.ParsePath("custom")).Bool()
			efList := o.LookupPath(cue.ParsePath("extra_fields"))
			if efList.Err() == nil {
				efIter, _ := efList.List()
				for efIter.Next() {
					ef, _ := efIter.Value().String()
					op.ExtraFields = append(op.ExtraFields, ef)
				}
			}
			svc.Operations = append(svc.Operations, op)
		}
		services = append(services, svc)
	}
	return services
}

// ─── Naming helpers ──────────────────────────────────────────────────────────

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

// entPascal converts snake_case to PascalCase matching Ent's convention.
// Only standard Go initialisms are uppercased (e.g., "id" -> "ID").
func entPascal(s string) string {
	parts := strings.Split(s, "_")
	for i, p := range parts {
		lower := strings.ToLower(p)
		if goInitialisms[lower] {
			parts[i] = strings.ToUpper(p)
		} else if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, "")
}

// entPkg returns the Ent package name for an entity (lowercase, no separators).
func entPkg(name string) string {
	return strings.ToLower(name)
}

// ─── Code Generation ─────────────────────────────────────────────────────────

type cw struct{ bytes.Buffer }

func (w *cw) line(format string, args ...interface{}) {
	fmt.Fprintf(&w.Buffer, format+"\n", args...)
}

func generateHandlerFile(projectRoot string, svc serviceDef, entities map[string]*entityInfo) error {
	handlerType := strings.TrimSuffix(svc.Name, "Service") + "Handler"
	svcBase := strings.TrimSuffix(svc.Name, "Service")
	fileName := "gen_" + toSnake(svcBase) + ".go"

	// Collect entity packages and import needs
	entPkgs := map[string]bool{}
	needUUID := false
	needTime := false
	needTypes := false
	needSchema := false
	needJSON := false

	// Pre-scan what we need
	for _, entName := range svc.Entities {
		ent, ok := entities[entName]
		if !ok {
			continue
		}
		entPkgs[entPkg(entName)] = true
		if len(ent.EdgeFKs) > 0 {
			needUUID = true
		}
		for _, f := range ent.Fields {
			if f.EntType == "Time" {
				needTime = true
			}
			if f.EntType == "JSON" && strings.Contains(f.JSONType, "types.") {
				needTypes = true
			}
			if f.EntType == "JSON" && (f.JSONType == "json.RawMessage" || f.JSONType == "") {
				needJSON = true
			}
		}
		// Check for non-custom transitions
		for _, op := range svc.Operations {
			if op.Entity == entName && op.Type == "transition" && !op.Custom {
				needSchema = true
				// Check extra fields for time types
				for _, ef := range op.ExtraFields {
					if strings.Contains(ef, "date") {
						needTime = true
					}
				}
			}
		}
	}

	var buf cw
	buf.line("// Code generated by cmd/handlergen from CUE ontology. DO NOT EDIT.")
	buf.line("package handler")
	buf.line("")
	buf.line("import (")
	if needJSON {
		buf.line("\t\"encoding/json\"")
	}
	buf.line("\t\"net/http\"")
	if needTime {
		buf.line("\t\"time\"")
	}
	buf.line("")
	if needUUID {
		buf.line("\t\"github.com/google/uuid\"")
	}
	buf.line("\t\"github.com/matthewbaird/ontology/ent\"")
	sortedPkgs := make([]string, 0, len(entPkgs))
	for p := range entPkgs {
		sortedPkgs = append(sortedPkgs, p)
	}
	sort.Strings(sortedPkgs)
	for _, p := range sortedPkgs {
		buf.line("\t\"github.com/matthewbaird/ontology/ent/%s\"", p)
	}
	if needSchema {
		buf.line("\t\"github.com/matthewbaird/ontology/ent/schema\"")
	}
	if needTypes {
		buf.line("\t\"github.com/matthewbaird/ontology/internal/types\"")
	}
	buf.line(")")
	buf.line("")

	// Suppress unused import warnings — use a schema var that exists for this service
	buf.line("// Ensure imports are used.")
	buf.line("var (")
	if needJSON {
		buf.line("\t_ json.RawMessage")
	}
	if needTime {
		buf.line("\t_ time.Time")
	}
	if needUUID {
		buf.line("\t_ uuid.UUID")
	}
	if needSchema {
		// Use the first entity in this service that has a state machine
		for _, entName := range svc.Entities {
			if e, ok := entities[entName]; ok && e.HasMachine {
				buf.line("\t_ = schema.Valid%sTransitions", entName)
				break
			}
		}
	}
	if needTypes {
		buf.line("\t_ types.Money")
	}
	buf.line(")")
	buf.line("")

	// Handler struct
	buf.line("// %s implements HTTP handlers for %s entities.", handlerType, svc.Name)
	buf.line("type %s struct {", handlerType)
	buf.line("\tclient *ent.Client")
	buf.line("}")
	buf.line("")
	buf.line("// New%s creates a new %s.", handlerType, handlerType)
	buf.line("func New%s(client *ent.Client) *%s {", handlerType, handlerType)
	buf.line("\treturn &%s{client: client}", handlerType)
	buf.line("}")
	buf.line("")

	// Per entity
	for _, entName := range svc.Entities {
		ent, ok := entities[entName]
		if !ok {
			continue
		}
		pkg := entPkg(entName)
		ops := opsForEntity(svc.Operations, entName)

		writeEntitySection(&buf, handlerType, ent, pkg, ops)
	}

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		// Write unformatted for debugging
		outPath := filepath.Join(projectRoot, "internal", "handler", fileName)
		os.WriteFile(outPath, buf.Bytes(), 0644)
		return fmt.Errorf("formatting %s: %w\nUnformatted file written for debugging.", fileName, err)
	}

	outPath := filepath.Join(projectRoot, "internal", "handler", fileName)
	return os.WriteFile(outPath, formatted, 0644)
}

func opsForEntity(ops []operationDef, entity string) []operationDef {
	var out []operationDef
	for _, op := range ops {
		if op.Entity == entity {
			out = append(out, op)
		}
	}
	return out
}

func writeEntitySection(buf *cw, handlerType string, ent *entityInfo, pkg string, ops []operationDef) {
	buf.line("// ============================================================================")
	buf.line("// %s", ent.Name)
	buf.line("// ============================================================================")
	buf.line("")

	// Find operation names
	var createOp, getOp, listOp, updateOp string
	var transitions []operationDef
	for _, op := range ops {
		if op.Custom {
			if op.Type == "transition" {
				continue // Custom transitions are hand-written
			}
			continue // Custom CRUD ops are hand-written
		}
		switch op.Type {
		case "create":
			createOp = op.Name
		case "get":
			getOp = op.Name
		case "list":
			listOp = op.Name
		case "update":
			updateOp = op.Name
		case "transition":
			transitions = append(transitions, op)
		}
	}

	if createOp != "" {
		writeCreateStruct(buf, ent, pkg)
		writeCreateHandler(buf, handlerType, ent, pkg, createOp)
	}

	if getOp != "" {
		writeGetHandler(buf, handlerType, ent, getOp)
	}

	if listOp != "" {
		writeListHandler(buf, handlerType, ent, pkg, listOp)
	}

	if updateOp != "" {
		writeUpdateStruct(buf, ent, pkg)
		writeUpdateHandler(buf, handlerType, ent, pkg, updateOp)
	}

	if len(transitions) > 0 {
		writeTransitionHelper(buf, handlerType, ent, pkg)
		for _, tr := range transitions {
			writeTransitionMethod(buf, handlerType, ent, pkg, tr)
		}
	}
}

// ─── Create ──────────────────────────────────────────────────────────────────

func writeCreateStruct(buf *cw, ent *entityInfo, pkg string) {
	buf.line("type create%sRequest struct {", ent.Name)
	for _, f := range ent.Fields {
		writeStructField(buf, f, false)
	}
	for _, efk := range ent.EdgeFKs {
		if efk.Optional {
			buf.line("\t%s *string `json:\"%s,omitempty\"`", entPascal(efk.FieldName), efk.FieldName)
		} else {
			buf.line("\t%s string `json:\"%s\"`", entPascal(efk.FieldName), efk.FieldName)
		}
	}
	buf.line("}")
	buf.line("")
}

func writeCreateHandler(buf *cw, handlerType string, ent *entityInfo, pkg, opName string) {
	buf.line("func (h *%s) %s(w http.ResponseWriter, r *http.Request) {", handlerType, opName)
	buf.line("\taudit, ok := parseAuditContext(w, r)")
	buf.line("\tif !ok { return }")
	buf.line("\tvar req create%sRequest", ent.Name)
	buf.line("\tif err := decodeJSON(r, &req); err != nil {")
	buf.line("\t\twriteError(w, http.StatusBadRequest, \"INVALID_JSON\", err.Error())")
	buf.line("\t\treturn")
	buf.line("\t}")
	buf.line("\tbuilder := h.client.%s.Create()", ent.Name)

	// Set fields
	for _, f := range ent.Fields {
		writeCreateSetter(buf, f, pkg)
	}
	// Set edge FK fields
	for _, efk := range ent.EdgeFKs {
		writeEdgeFKSetter(buf, efk, false)
	}

	// Audit fields
	buf.line("\tbuilder.SetCreatedBy(audit.Actor).SetUpdatedBy(audit.Actor).SetSource(%s.Source(audit.Source))", pkg)
	buf.line("\tif audit.CorrelationID != nil {")
	buf.line("\t\tbuilder.SetCorrelationID(*audit.CorrelationID)")
	buf.line("\t}")
	buf.line("\tresult, err := builder.Save(r.Context())")
	buf.line("\tif err != nil {")
	buf.line("\t\tentErrorToHTTP(w, err)")
	buf.line("\t\treturn")
	buf.line("\t}")
	buf.line("\twriteJSON(w, http.StatusCreated, result)")
	buf.line("}")
	buf.line("")
}

// ─── Get ─────────────────────────────────────────────────────────────────────

func writeGetHandler(buf *cw, handlerType string, ent *entityInfo, opName string) {
	buf.line("func (h *%s) %s(w http.ResponseWriter, r *http.Request) {", handlerType, opName)
	buf.line("\tid, ok := parseUUID(w, r, \"id\")")
	buf.line("\tif !ok { return }")
	buf.line("\tresult, err := h.client.%s.Get(r.Context(), id)", ent.Name)
	buf.line("\tif err != nil {")
	buf.line("\t\tentErrorToHTTP(w, err)")
	buf.line("\t\treturn")
	buf.line("\t}")
	buf.line("\twriteJSON(w, http.StatusOK, result)")
	buf.line("}")
	buf.line("")
}

// ─── List ────────────────────────────────────────────────────────────────────

func writeListHandler(buf *cw, handlerType string, ent *entityInfo, pkg, opName string) {
	buf.line("func (h *%s) %s(w http.ResponseWriter, r *http.Request) {", handlerType, opName)
	buf.line("\tpg := parsePagination(r)")
	buf.line("\titems, err := h.client.%s.Query().", ent.Name)
	buf.line("\t\tLimit(pg.Limit).Offset(pg.Offset).")
	buf.line("\t\tOrder(ent.Desc(%s.FieldCreatedAt)).", pkg)
	buf.line("\t\tAll(r.Context())")
	buf.line("\tif err != nil {")
	buf.line("\t\tentErrorToHTTP(w, err)")
	buf.line("\t\treturn")
	buf.line("\t}")
	buf.line("\twriteJSON(w, http.StatusOK, items)")
	buf.line("}")
	buf.line("")
}

// ─── Update ──────────────────────────────────────────────────────────────────

func writeUpdateStruct(buf *cw, ent *entityInfo, pkg string) {
	buf.line("type update%sRequest struct {", ent.Name)
	for _, f := range ent.Fields {
		writeStructField(buf, f, true)
	}
	for _, efk := range ent.EdgeFKs {
		buf.line("\t%s *string `json:\"%s,omitempty\"`", entPascal(efk.FieldName), efk.FieldName)
	}
	buf.line("}")
	buf.line("")
}

func writeUpdateHandler(buf *cw, handlerType string, ent *entityInfo, pkg, opName string) {
	buf.line("func (h *%s) %s(w http.ResponseWriter, r *http.Request) {", handlerType, opName)
	buf.line("\tid, ok := parseUUID(w, r, \"id\")")
	buf.line("\tif !ok { return }")
	buf.line("\taudit, ok := parseAuditContext(w, r)")
	buf.line("\tif !ok { return }")
	buf.line("\tvar req update%sRequest", ent.Name)
	buf.line("\tif err := decodeJSON(r, &req); err != nil {")
	buf.line("\t\twriteError(w, http.StatusBadRequest, \"INVALID_JSON\", err.Error())")
	buf.line("\t\treturn")
	buf.line("\t}")
	buf.line("\tbuilder := h.client.%s.UpdateOneID(id)", ent.Name)

	// Set fields
	for _, f := range ent.Fields {
		writeUpdateSetter(buf, f, pkg)
	}
	// Set edge FK fields
	for _, efk := range ent.EdgeFKs {
		writeEdgeFKSetter(buf, efk, true)
	}

	// Audit fields
	buf.line("\tbuilder.SetUpdatedBy(audit.Actor).SetSource(%s.Source(audit.Source))", pkg)
	buf.line("\tif audit.CorrelationID != nil {")
	buf.line("\t\tbuilder.SetCorrelationID(*audit.CorrelationID)")
	buf.line("\t}")
	buf.line("\tresult, err := builder.Save(r.Context())")
	buf.line("\tif err != nil {")
	buf.line("\t\tentErrorToHTTP(w, err)")
	buf.line("\t\treturn")
	buf.line("\t}")
	buf.line("\twriteJSON(w, http.StatusOK, result)")
	buf.line("}")
	buf.line("")
}

// ─── Transitions ─────────────────────────────────────────────────────────────

func writeTransitionHelper(buf *cw, handlerType string, ent *entityInfo, pkg string) {
	buf.line("func (h *%s) transition%s(w http.ResponseWriter, r *http.Request, targetStatus string, applyExtra func(*ent.%sUpdateOne)) {", handlerType, ent.Name, ent.Name)
	buf.line("\tid, ok := parseUUID(w, r, \"id\")")
	buf.line("\tif !ok { return }")
	buf.line("\taudit, ok := parseAuditContext(w, r)")
	buf.line("\tif !ok { return }")
	buf.line("\tcurrent, err := h.client.%s.Get(r.Context(), id)", ent.Name)
	buf.line("\tif err != nil {")
	buf.line("\t\tentErrorToHTTP(w, err)")
	buf.line("\t\treturn")
	buf.line("\t}")
	buf.line("\tif err := ValidateTransition(schema.Valid%sTransitions, string(current.Status), targetStatus); err != nil {", ent.Name)
	buf.line("\t\twriteError(w, http.StatusConflict, \"INVALID_TRANSITION\", err.Error())")
	buf.line("\t\treturn")
	buf.line("\t}")
	buf.line("\tbuilder := h.client.%s.UpdateOneID(id).", ent.Name)
	buf.line("\t\tSetStatus(%s.Status(targetStatus)).", pkg)
	buf.line("\t\tSetUpdatedBy(audit.Actor).")
	buf.line("\t\tSetSource(%s.Source(audit.Source))", pkg)
	buf.line("\tif audit.CorrelationID != nil {")
	buf.line("\t\tbuilder.SetCorrelationID(*audit.CorrelationID)")
	buf.line("\t}")
	buf.line("\tif applyExtra != nil {")
	buf.line("\t\tapplyExtra(builder)")
	buf.line("\t}")
	buf.line("\tupdated, err := builder.Save(r.Context())")
	buf.line("\tif err != nil {")
	buf.line("\t\tentErrorToHTTP(w, err)")
	buf.line("\t\treturn")
	buf.line("\t}")
	buf.line("\twriteJSON(w, http.StatusOK, updated)")
	buf.line("}")
	buf.line("")
}

func writeTransitionMethod(buf *cw, handlerType string, ent *entityInfo, pkg string, op operationDef) {
	buf.line("func (h *%s) %s(w http.ResponseWriter, r *http.Request) {", handlerType, op.Name)
	if len(op.ExtraFields) > 0 {
		// Define an inline struct for extra fields
		buf.line("\ttype extraFields struct {")
		for _, ef := range op.ExtraFields {
			goType := "string"
			if strings.Contains(ef, "date") {
				goType = "*time.Time"
			}
			buf.line("\t\t%s %s `json:\"%s,omitempty\"`", entPascal(ef), goType, ef)
		}
		buf.line("\t}")
		buf.line("\tvar extra extraFields")
		buf.line("\t_ = decodeJSON(r, &extra)")
		buf.line("\th.transition%s(w, r, \"%s\", func(b *ent.%sUpdateOne) {", ent.Name, op.ToStatus, ent.Name)
		for _, ef := range op.ExtraFields {
			goName := entPascal(ef)
			entName := entPascal(ef)
			if strings.Contains(ef, "date") {
				// Time field — use SetNillable
				buf.line("\t\tif extra.%s != nil { b.SetNillable%s(extra.%s) }", goName, entName, goName)
			}
			// Non-date extra fields (like "reason") are accepted but not persisted
			// unless the entity has a matching field
		}
		buf.line("\t})")
	} else {
		buf.line("\th.transition%s(w, r, \"%s\", nil)", ent.Name, op.ToStatus)
	}
	buf.line("}")
	buf.line("")
}

// ─── Struct field helpers ────────────────────────────────────────────────────

func writeStructField(buf *cw, f fieldDef, isUpdate bool) {
	goName := entPascal(f.Name)

	switch f.EntType {
	case "Money":
		amtName := f.Name + "_amount_cents"
		curName := f.Name + "_currency"
		if isUpdate {
			buf.line("\t%s *int64 `json:\"%s,omitempty\"`", entPascal(amtName), amtName)
			buf.line("\t%s *string `json:\"%s,omitempty\"`", entPascal(curName), curName)
		} else if f.Optional {
			buf.line("\t%s *int64 `json:\"%s,omitempty\"`", entPascal(amtName), amtName)
			buf.line("\t%s *string `json:\"%s,omitempty\"`", entPascal(curName), curName)
		} else {
			buf.line("\t%s int64 `json:\"%s\"`", entPascal(amtName), amtName)
			buf.line("\t%s string `json:\"%s,omitempty\"`", entPascal(curName), curName)
		}

	case "String":
		if isUpdate {
			buf.line("\t%s *string `json:\"%s,omitempty\"`", goName, f.Name)
		} else if f.Optional {
			buf.line("\t%s *string `json:\"%s,omitempty\"`", goName, f.Name)
		} else {
			buf.line("\t%s string `json:\"%s\"`", goName, f.Name)
		}

	case "Int":
		if isUpdate {
			buf.line("\t%s *int `json:\"%s,omitempty\"`", goName, f.Name)
		} else if f.Optional {
			buf.line("\t%s *int `json:\"%s,omitempty\"`", goName, f.Name)
		} else {
			buf.line("\t%s int `json:\"%s\"`", goName, f.Name)
		}

	case "Int64":
		if isUpdate {
			buf.line("\t%s *int64 `json:\"%s,omitempty\"`", goName, f.Name)
		} else if f.Optional {
			buf.line("\t%s *int64 `json:\"%s,omitempty\"`", goName, f.Name)
		} else {
			buf.line("\t%s int64 `json:\"%s\"`", goName, f.Name)
		}

	case "Float64":
		if isUpdate {
			buf.line("\t%s *float64 `json:\"%s,omitempty\"`", goName, f.Name)
		} else if f.Optional {
			buf.line("\t%s *float64 `json:\"%s,omitempty\"`", goName, f.Name)
		} else {
			buf.line("\t%s float64 `json:\"%s\"`", goName, f.Name)
		}

	case "Bool":
		if isUpdate {
			buf.line("\t%s *bool `json:\"%s,omitempty\"`", goName, f.Name)
		} else if f.Optional {
			buf.line("\t%s *bool `json:\"%s,omitempty\"`", goName, f.Name)
		} else {
			buf.line("\t%s bool `json:\"%s\"`", goName, f.Name)
		}

	case "Time":
		if isUpdate {
			buf.line("\t%s *time.Time `json:\"%s,omitempty\"`", goName, f.Name)
		} else if f.Optional {
			buf.line("\t%s *time.Time `json:\"%s,omitempty\"`", goName, f.Name)
		} else {
			buf.line("\t%s time.Time `json:\"%s\"`", goName, f.Name)
		}

	case "Enum":
		if isUpdate {
			buf.line("\t%s *string `json:\"%s,omitempty\"`", goName, f.Name)
		} else if f.Optional {
			buf.line("\t%s *string `json:\"%s,omitempty\"`", goName, f.Name)
		} else {
			buf.line("\t%s string `json:\"%s\"`", goName, f.Name)
		}

	case "JSON":
		goType := f.JSONType
		if goType == "" {
			goType = "json.RawMessage"
		}
		isSlice := strings.HasPrefix(goType, "[]")
		if isUpdate {
			if isSlice {
				buf.line("\t%s %s `json:\"%s,omitempty\"`", goName, goType, f.Name)
			} else {
				buf.line("\t%s *%s `json:\"%s,omitempty\"`", goName, goType, f.Name)
			}
		} else if f.Optional {
			if isSlice {
				buf.line("\t%s %s `json:\"%s,omitempty\"`", goName, goType, f.Name)
			} else {
				buf.line("\t%s *%s `json:\"%s,omitempty\"`", goName, goType, f.Name)
			}
		} else {
			if isSlice {
				buf.line("\t%s %s `json:\"%s\"`", goName, goType, f.Name)
			} else {
				buf.line("\t%s %s `json:\"%s\"`", goName, goType, f.Name)
			}
		}
	}
}

// ─── Create setter helpers ───────────────────────────────────────────────────

func writeCreateSetter(buf *cw, f fieldDef, pkg string) {
	goName := entPascal(f.Name)
	entName := entPascal(f.Name)

	switch f.EntType {
	case "Money":
		amtEntName := entPascal(f.Name + "_amount_cents")
		curEntName := entPascal(f.Name + "_currency")
		amtGoName := entPascal(f.Name + "_amount_cents")
		curGoName := entPascal(f.Name + "_currency")
		if f.Optional {
			buf.line("\tif req.%s != nil { builder.SetNillable%s(req.%s) }", amtGoName, amtEntName, amtGoName)
			buf.line("\tif req.%s != nil { builder.SetNillable%s(req.%s) }", curGoName, curEntName, curGoName)
		} else {
			buf.line("\tbuilder.Set%s(req.%s)", amtEntName, amtGoName)
			buf.line("\tif req.%s != \"\" { builder.Set%s(req.%s) }", curGoName, curEntName, curGoName)
		}

	case "String":
		if f.Optional {
			buf.line("\tif req.%s != nil { builder.SetNillable%s(req.%s) }", goName, entName, goName)
		} else {
			buf.line("\tbuilder.Set%s(req.%s)", entName, goName)
		}

	case "Int", "Int64":
		if f.Optional {
			buf.line("\tif req.%s != nil { builder.SetNillable%s(req.%s) }", goName, entName, goName)
		} else {
			buf.line("\tbuilder.Set%s(req.%s)", entName, goName)
		}

	case "Float64":
		if f.Optional {
			buf.line("\tif req.%s != nil { builder.SetNillable%s(req.%s) }", goName, entName, goName)
		} else {
			buf.line("\tbuilder.Set%s(req.%s)", entName, goName)
		}

	case "Bool":
		if f.Optional {
			buf.line("\tif req.%s != nil { builder.Set%s(*req.%s) }", goName, entName, goName)
		} else {
			buf.line("\tbuilder.Set%s(req.%s)", entName, goName)
		}

	case "Time":
		if f.Optional {
			buf.line("\tif req.%s != nil { builder.SetNillable%s(req.%s) }", goName, entName, goName)
		} else {
			buf.line("\tbuilder.Set%s(req.%s)", entName, goName)
		}

	case "Enum":
		if f.Optional {
			buf.line("\tif req.%s != nil { builder.Set%s(%s.%s(*req.%s)) }", goName, entName, pkg, entName, goName)
		} else if f.Default != "" {
			// Field has an Ent default; only set when the client provides a value.
			buf.line("\tif req.%s != \"\" { builder.Set%s(%s.%s(req.%s)) }", goName, entName, pkg, entName, goName)
		} else {
			buf.line("\tbuilder.Set%s(%s.%s(req.%s))", entName, pkg, entName, goName)
		}

	case "JSON":
		isSlice := strings.HasPrefix(f.JSONType, "[]")
		if f.Optional {
			if isSlice {
				buf.line("\tif len(req.%s) > 0 { builder.Set%s(req.%s) }", goName, entName, goName)
			} else {
				buf.line("\tif req.%s != nil { builder.Set%s(req.%s) }", goName, entName, goName)
			}
		} else {
			if isSlice {
				buf.line("\tbuilder.Set%s(req.%s)", entName, goName)
			} else {
				// Non-optional single value: setter takes pointer, pass &req.X
				buf.line("\tbuilder.Set%s(&req.%s)", entName, goName)
			}
		}
	}
}

// ─── Update setter helpers ───────────────────────────────────────────────────

func writeUpdateSetter(buf *cw, f fieldDef, pkg string) {
	goName := entPascal(f.Name)
	entName := entPascal(f.Name)

	switch f.EntType {
	case "Money":
		amtEntName := entPascal(f.Name + "_amount_cents")
		curEntName := entPascal(f.Name + "_currency")
		amtGoName := entPascal(f.Name + "_amount_cents")
		curGoName := entPascal(f.Name + "_currency")
		buf.line("\tif req.%s != nil { builder.Set%s(*req.%s) }", amtGoName, amtEntName, amtGoName)
		buf.line("\tif req.%s != nil { builder.Set%s(*req.%s) }", curGoName, curEntName, curGoName)

	case "String":
		if f.Optional {
			// Optional strings use SetNillable in update too
			buf.line("\tif req.%s != nil { builder.SetNillable%s(req.%s) }", goName, entName, goName)
		} else {
			buf.line("\tif req.%s != nil { builder.Set%s(*req.%s) }", goName, entName, goName)
		}

	case "Int", "Int64":
		if f.Optional {
			buf.line("\tif req.%s != nil { builder.SetNillable%s(req.%s) }", goName, entName, goName)
		} else {
			buf.line("\tif req.%s != nil { builder.Set%s(*req.%s) }", goName, entName, goName)
		}

	case "Float64":
		if f.Optional {
			buf.line("\tif req.%s != nil { builder.SetNillable%s(req.%s) }", goName, entName, goName)
		} else {
			buf.line("\tif req.%s != nil { builder.Set%s(*req.%s) }", goName, entName, goName)
		}

	case "Bool":
		buf.line("\tif req.%s != nil { builder.Set%s(*req.%s) }", goName, entName, goName)

	case "Time":
		buf.line("\tif req.%s != nil { builder.SetNillable%s(req.%s) }", goName, entName, goName)

	case "Enum":
		buf.line("\tif req.%s != nil { builder.Set%s(%s.%s(*req.%s)) }", goName, entName, pkg, entName, goName)

	case "JSON":
		isSlice := strings.HasPrefix(f.JSONType, "[]")
		if isSlice {
			buf.line("\tif req.%s != nil { builder.Set%s(req.%s) }", goName, entName, goName)
		} else {
			buf.line("\tif req.%s != nil { builder.Set%s(req.%s) }", goName, entName, goName)
		}
	}
}

// ─── Edge FK setter helpers ──────────────────────────────────────────────────

func writeEdgeFKSetter(buf *cw, efk edgeFK, isUpdate bool) {
	goName := entPascal(efk.FieldName)
	edgeSetter := "Set" + entPascal(efk.EdgeName) + "ID"

	if efk.Optional || isUpdate {
		buf.line("\tif req.%s != nil {", goName)
		buf.line("\t\tuid, err := uuid.Parse(*req.%s)", goName)
		buf.line("\t\tif err != nil {")
		buf.line("\t\t\twriteError(w, http.StatusBadRequest, \"INVALID_ID\", \"invalid %s\")", efk.FieldName)
		buf.line("\t\t\treturn")
		buf.line("\t\t}")
		buf.line("\t\tbuilder.%s(uid)", edgeSetter)
		buf.line("\t}")
	} else {
		buf.line("\t{")
		buf.line("\t\tuid, err := uuid.Parse(req.%s)", goName)
		buf.line("\t\tif err != nil {")
		buf.line("\t\t\twriteError(w, http.StatusBadRequest, \"INVALID_ID\", \"invalid %s\")", efk.FieldName)
		buf.line("\t\t\treturn")
		buf.line("\t\t}")
		buf.line("\t\tbuilder.%s(uid)", edgeSetter)
		buf.line("\t}")
	}
}

// ─── Routes generation ───────────────────────────────────────────────────────

func generateRoutesFile(projectRoot string, services []serviceDef, handlerTypes map[string]string) error {
	var buf cw
	buf.line("// Code generated by cmd/handlergen from CUE ontology. DO NOT EDIT.")
	buf.line("package server")
	buf.line("")
	buf.line("import (")
	buf.line("\t\"net/http\"")
	buf.line("")
	buf.line("\t\"github.com/go-chi/chi/v5\"")
	buf.line("\t\"github.com/matthewbaird/ontology/ent\"")
	buf.line("\t\"github.com/matthewbaird/ontology/internal/handler\"")
	buf.line(")")
	buf.line("")
	buf.line("// RegisterRoutes registers all generated HTTP routes on the given router.")
	buf.line("func RegisterRoutes(r chi.Router, client *ent.Client) {")
	buf.line("\t// Health check")
	buf.line("\tr.Get(\"/healthz\", func(w http.ResponseWriter, r *http.Request) {")
	buf.line("\t\tw.Header().Set(\"Content-Type\", \"application/json\")")
	buf.line("\t\tw.Write([]byte(`{\"status\":\"ok\"}`))")
	buf.line("\t})")
	buf.line("")

	// Handler variable names
	handlerVars := map[string]string{
		"PersonService":     "ph",
		"PropertyService":   "proph",
		"LeaseService":      "lh",
		"AccountingService": "ah",
	}

	// Instantiate handlers — skip services where all operations are custom
	// (their routes are registered manually, not generated).
	for _, svc := range services {
		if !hasGeneratedRoutes(svc) {
			continue
		}
		v := handlerVars[svc.Name]
		ht := handlerTypes[svc.Name]
		buf.line("\t%s := handler.New%s(client)", v, ht)
	}
	buf.line("")

	// Register routes
	for _, svc := range services {
		if !hasGeneratedRoutes(svc) {
			continue
		}
		v := handlerVars[svc.Name]
		for _, op := range svc.Operations {
			// Custom non-transition operations have unique paths that don't fit
			// the standard CRUD pattern — they are registered manually.
			if op.Custom && op.Type != "transition" {
				continue
			}
			basePath := "/v1/" + op.EntityPath
			var chiMethod, path string
			switch op.Type {
			case "create":
				chiMethod, path = "Post", basePath
			case "get":
				chiMethod, path = "Get", basePath+"/{id}"
			case "list":
				chiMethod, path = "Get", basePath
			case "update":
				chiMethod, path = "Patch", basePath+"/{id}"
			case "delete":
				chiMethod, path = "Delete", basePath+"/{id}"
			case "transition":
				chiMethod, path = "Post", basePath+"/{id}/"+op.Action
			}
			buf.line("\tr.%s(\"%s\", %s.%s)", chiMethod, path, v, op.Name)
		}
	}

	buf.line("}")

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		outPath := filepath.Join(projectRoot, "internal", "server", "gen_routes.go")
		os.WriteFile(outPath, buf.Bytes(), 0644)
		return fmt.Errorf("formatting routes: %w", err)
	}

	outPath := filepath.Join(projectRoot, "internal", "server", "gen_routes.go")
	return os.WriteFile(outPath, formatted, 0644)
}

// hasGeneratedRoutes returns true if the service has at least one operation
// that produces a generated route (i.e., not all operations are custom).
func hasGeneratedRoutes(svc serviceDef) bool {
	for _, op := range svc.Operations {
		if !op.Custom || op.Type == "transition" {
			return true
		}
	}
	return false
}

// ─── Main ────────────────────────────────────────────────────────────────────

func main() {
	log.SetFlags(0)
	log.SetPrefix("handlergen: ")

	ctx := cuecontext.New()
	projectRoot := findProjectRoot()

	// Parse ontology entities
	insts := load.Instances([]string{"./ontology"}, &load.Config{Dir: projectRoot})
	if len(insts) == 0 || insts[0].Err != nil {
		log.Fatalf("loading ontology: %v", insts[0].Err)
	}
	val := ctx.BuildInstance(insts[0])
	if val.Err() != nil {
		log.Fatalf("building ontology: %v", val.Err())
	}

	entities := parseEntities(val)
	parseRelationships(val, entities)
	parseStateMachines(val, entities)

	// Parse service definitions
	services := parseServices(ctx, projectRoot)

	// Generate handler files
	handlerTypes := map[string]string{}
	for _, svc := range services {
		ht := strings.TrimSuffix(svc.Name, "Service") + "Handler"
		handlerTypes[svc.Name] = ht
		svcBase := strings.TrimSuffix(svc.Name, "Service")
		fileName := "gen_" + toSnake(svcBase) + ".go"

		// Skip handler file generation for services where all operations are custom.
		if !hasGeneratedRoutes(svc) {
			continue
		}

		if err := generateHandlerFile(projectRoot, svc, entities); err != nil {
			log.Fatalf("generating %s: %v", fileName, err)
		}
		fmt.Printf("Generated internal/handler/%s\n", fileName)
	}

	// Generate routes
	if err := generateRoutesFile(projectRoot, services, handlerTypes); err != nil {
		log.Fatalf("generating routes: %v", err)
	}
	fmt.Println("Generated internal/server/gen_routes.go")

	fmt.Printf("handlergen: generated %d handler files + routes\n", len(services))
}
