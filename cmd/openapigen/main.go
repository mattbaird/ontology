// cmd/openapigen generates an OpenAPI 3.1 spec from the CUE ontology + codegen/apigen.cue.
//
// Output: gen/openapi/openapi.json
//
// This follows the same architecture as the other generators — load CUE,
// iterate entities + services, emit a single derived artifact.
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

// ─── Data types ──────────────────────────────────────────────────────────────

type fieldDef struct {
	Name       string
	FieldType  string // "string","int","float","bool","time","enum","json","money","uuid"
	Optional   bool
	EnumValues []string
	JSONType   string // Go type for JSON fields (e.g. "[]string", "types.Money")
	Deprecated bool   // @deprecated() — mark in OpenAPI output
}

type entityInfo struct {
	Name   string
	Fields []fieldDef
}

type serviceDef struct {
	Name       string
	Operations []operationDef
}

type operationDef struct {
	Name        string
	Entity      string
	Type        string // create, get, list, update, delete, transition
	EntityPath  string
	Action      string
	ToStatus    string
	Description string
}

// ─── Known type maps ─────────────────────────────────────────────────────────

var moneyFieldNames = map[string]bool{
	"#Money": true, "#NonNegativeMoney": true, "#PositiveMoney": true,
}

var knownValueTypes = map[string]bool{
	"#Money": true, "#NonNegativeMoney": true, "#PositiveMoney": true,
	"#Address": true, "#ContactMethod": true, "#DateRange": true,
	"#EntityRef": true, "#RentScheduleEntry": true, "#RecurringCharge": true,
	"#LateFeePolicy": true, "#CAMTerms": true, "#TenantImprovement": true,
	"#RenewalOption": true, "#SubsidyTerms": true, "#AccountDimensions": true,
	"#JournalLine": true, "#RoleAttributes": true, "#TenantAttributes": true,
	"#OwnerAttributes": true, "#ManagerAttributes": true, "#GuarantorAttributes": true,
}

// ─── CUE parsing (simplified from entgen/handlergen) ─────────────────────────

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

func extractEnumValues(val cue.Value) []string {
	op, args := val.Expr()
	if op != cue.OrOp {
		return nil
	}
	var values []string
	for _, arg := range args {
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

func inferListElementKind(val cue.Value) cue.Kind {
	op, args := val.Expr()
	if op == cue.AndOp || op == cue.OrOp {
		for _, a := range args {
			elem := a.LookupPath(cue.MakePath(cue.AnyIndex))
			if elem.Err() == nil {
				k := elem.IncompleteKind()
				if k != cue.BottomKind {
					return k
				}
			}
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
		fd.FieldType = "time"
		return fd
	}

	ref := findReference(val)
	if ref != "" {
		if moneyFieldNames[ref] {
			return &fieldDef{Name: name, FieldType: "money", Optional: optional}
		}
		if knownValueTypes[ref] {
			fd.FieldType = "json"
			fd.JSONType = ref
			return fd
		}
	}

	if val.IncompleteKind() == cue.ListKind || inferKindFromExpr(val) == cue.ListKind {
		fd.FieldType = "json"
		elem := val.LookupPath(cue.MakePath(cue.AnyIndex))
		if elem.Err() == nil {
			if elem.IncompleteKind() == cue.StringKind || isEnum(elem) {
				fd.JSONType = "[]string"
				return fd
			}
			eRef := findReference(elem)
			if eRef != "" && knownValueTypes[eRef] {
				fd.JSONType = "[]" + eRef
				return fd
			}
		}
		if inferListElementKind(val) == cue.StringKind {
			fd.JSONType = "[]string"
			return fd
		}
		fd.JSONType = "object"
		return fd
	}

	if isEnum(val) {
		fd.FieldType = "enum"
		fd.EnumValues = extractEnumValues(val)
		return fd
	}

	kind := val.IncompleteKind()
	if kind == cue.BottomKind {
		kind = inferKindFromExpr(val)
	}

	switch kind {
	case cue.StringKind:
		fd.FieldType = "string"
	case cue.IntKind:
		fd.FieldType = "int"
	case cue.FloatKind, cue.NumberKind:
		fd.FieldType = "float"
	case cue.BoolKind:
		fd.FieldType = "bool"
	default:
		return nil
	}
	return fd
}

// ─── Entity and service parsing ──────────────────────────────────────────────

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
				if a := fIter.Value().Attribute("deprecated"); a.Err() == nil {
					fd.Deprecated = true
				}
				ent.Fields = append(ent.Fields, *fd)
			}
		}
		entities[name] = ent
	}
	return entities
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
			op.Description, _ = o.LookupPath(cue.ParsePath("description")).String()
			svc.Operations = append(svc.Operations, op)
		}
		services = append(services, svc)
	}
	return services
}

// ─── OpenAPI spec building ───────────────────────────────────────────────────

type orderedMap struct {
	keys   []string
	values map[string]interface{}
}

func newOrderedMap() *orderedMap {
	return &orderedMap{values: make(map[string]interface{})}
}

func (om *orderedMap) Set(key string, value interface{}) {
	if _, exists := om.values[key]; !exists {
		om.keys = append(om.keys, key)
	}
	om.values[key] = value
}

func (om *orderedMap) MarshalJSON() ([]byte, error) {
	var buf strings.Builder
	buf.WriteString("{")
	for i, key := range om.keys {
		if i > 0 {
			buf.WriteString(",")
		}
		keyJSON, _ := json.Marshal(key)
		buf.Write(keyJSON)
		buf.WriteString(":")
		valJSON, err := json.Marshal(om.values[key])
		if err != nil {
			return nil, err
		}
		buf.Write(valJSON)
	}
	buf.WriteString("}")
	return []byte(buf.String()), nil
}

func fieldToSchema(f fieldDef) map[string]interface{} {
	switch f.FieldType {
	case "string":
		return map[string]interface{}{"type": "string"}
	case "int":
		return map[string]interface{}{"type": "integer", "format": "int32"}
	case "float":
		return map[string]interface{}{"type": "number", "format": "double"}
	case "bool":
		return map[string]interface{}{"type": "boolean"}
	case "time":
		return map[string]interface{}{"type": "string", "format": "date-time"}
	case "uuid":
		return map[string]interface{}{"type": "string", "format": "uuid"}
	case "enum":
		return map[string]interface{}{"type": "string", "enum": f.EnumValues}
	case "money":
		// Money is handled specially — expanded to two fields
		return nil
	case "json":
		if f.JSONType == "[]string" {
			return map[string]interface{}{
				"type":  "array",
				"items": map[string]interface{}{"type": "string"},
			}
		}
		if strings.HasPrefix(f.JSONType, "[]") {
			return map[string]interface{}{
				"type":  "array",
				"items": map[string]interface{}{"type": "object"},
			}
		}
		return map[string]interface{}{"type": "object"}
	}
	return map[string]interface{}{"type": "string"}
}

func buildEntitySchema(ent *entityInfo) *orderedMap {
	schema := newOrderedMap()
	schema.Set("type", "object")

	props := newOrderedMap()
	var required []string

	// Always include id
	props.Set("id", map[string]interface{}{"type": "string", "format": "uuid"})
	required = append(required, "id")

	for _, f := range ent.Fields {
		if f.FieldType == "money" {
			// Expand Money fields to two properties
			amtName := f.Name + "_amount_cents"
			curName := f.Name + "_currency"
			props.Set(amtName, map[string]interface{}{"type": "integer", "format": "int64", "description": f.Name + " amount in cents"})
			props.Set(curName, map[string]interface{}{"type": "string", "pattern": "^[A-Z]{3}$", "description": f.Name + " ISO 4217 currency code"})
			if !f.Optional {
				required = append(required, amtName)
			}
			continue
		}

		s := fieldToSchema(f)
		if s != nil {
			if f.Deprecated {
				s["deprecated"] = true
			}
			props.Set(f.Name, s)
			if !f.Optional {
				required = append(required, f.Name)
			}
		}
	}

	// Audit fields
	props.Set("created_at", map[string]interface{}{"type": "string", "format": "date-time"})
	props.Set("updated_at", map[string]interface{}{"type": "string", "format": "date-time"})
	props.Set("created_by", map[string]interface{}{"type": "string"})
	props.Set("updated_by", map[string]interface{}{"type": "string"})

	schema.Set("properties", props)
	if len(required) > 0 {
		schema.Set("required", required)
	}
	return schema
}

func httpMethod(opType string) string {
	switch opType {
	case "create":
		return "post"
	case "get":
		return "get"
	case "list":
		return "get"
	case "update":
		return "patch"
	case "delete":
		return "delete"
	case "transition":
		return "post"
	}
	return "post"
}

func buildPathItem(op operationDef, entities map[string]*entityInfo) map[string]interface{} {
	item := map[string]interface{}{
		"operationId": op.Name,
		"summary":     op.Description,
		"tags":        []string{op.Entity},
	}

	switch op.Type {
	case "create":
		item["requestBody"] = map[string]interface{}{
			"required": true,
			"content": map[string]interface{}{
				"application/json": map[string]interface{}{
					"schema": map[string]interface{}{
						"$ref": "#/components/schemas/" + op.Entity + "Create",
					},
				},
			},
		}
		item["responses"] = map[string]interface{}{
			"201": map[string]interface{}{
				"description": "Created",
				"content": map[string]interface{}{
					"application/json": map[string]interface{}{
						"schema": map[string]interface{}{"$ref": "#/components/schemas/" + op.Entity},
					},
				},
			},
			"400": map[string]interface{}{"description": "Bad Request"},
		}

	case "get":
		item["parameters"] = []map[string]interface{}{
			{"name": "id", "in": "path", "required": true, "schema": map[string]interface{}{"type": "string", "format": "uuid"}},
		}
		item["responses"] = map[string]interface{}{
			"200": map[string]interface{}{
				"description": "OK",
				"content": map[string]interface{}{
					"application/json": map[string]interface{}{
						"schema": map[string]interface{}{"$ref": "#/components/schemas/" + op.Entity},
					},
				},
			},
			"404": map[string]interface{}{"description": "Not Found"},
		}

	case "list":
		item["parameters"] = []map[string]interface{}{
			{"name": "limit", "in": "query", "schema": map[string]interface{}{"type": "integer", "default": 50}},
			{"name": "offset", "in": "query", "schema": map[string]interface{}{"type": "integer", "default": 0}},
		}
		item["responses"] = map[string]interface{}{
			"200": map[string]interface{}{
				"description": "OK",
				"content": map[string]interface{}{
					"application/json": map[string]interface{}{
						"schema": map[string]interface{}{
							"type":  "array",
							"items": map[string]interface{}{"$ref": "#/components/schemas/" + op.Entity},
						},
					},
				},
			},
		}

	case "update":
		item["parameters"] = []map[string]interface{}{
			{"name": "id", "in": "path", "required": true, "schema": map[string]interface{}{"type": "string", "format": "uuid"}},
		}
		item["requestBody"] = map[string]interface{}{
			"required": true,
			"content": map[string]interface{}{
				"application/json": map[string]interface{}{
					"schema": map[string]interface{}{
						"$ref": "#/components/schemas/" + op.Entity + "Update",
					},
				},
			},
		}
		item["responses"] = map[string]interface{}{
			"200": map[string]interface{}{
				"description": "OK",
				"content": map[string]interface{}{
					"application/json": map[string]interface{}{
						"schema": map[string]interface{}{"$ref": "#/components/schemas/" + op.Entity},
					},
				},
			},
			"400": map[string]interface{}{"description": "Bad Request"},
			"404": map[string]interface{}{"description": "Not Found"},
		}

	case "transition":
		item["parameters"] = []map[string]interface{}{
			{"name": "id", "in": "path", "required": true, "schema": map[string]interface{}{"type": "string", "format": "uuid"}},
		}
		desc := op.Description
		if op.ToStatus != "" {
			desc += fmt.Sprintf(" (transitions to %q)", op.ToStatus)
		}
		item["summary"] = desc
		item["responses"] = map[string]interface{}{
			"200": map[string]interface{}{
				"description": "OK",
				"content": map[string]interface{}{
					"application/json": map[string]interface{}{
						"schema": map[string]interface{}{"$ref": "#/components/schemas/" + op.Entity},
					},
				},
			},
			"409": map[string]interface{}{"description": "Invalid State Transition"},
		}
	}

	return item
}

func buildCreateSchema(ent *entityInfo) *orderedMap {
	schema := newOrderedMap()
	schema.Set("type", "object")
	props := newOrderedMap()
	var required []string

	for _, f := range ent.Fields {
		if f.FieldType == "money" {
			amtName := f.Name + "_amount_cents"
			curName := f.Name + "_currency"
			props.Set(amtName, map[string]interface{}{"type": "integer", "format": "int64"})
			props.Set(curName, map[string]interface{}{"type": "string", "pattern": "^[A-Z]{3}$"})
			if !f.Optional {
				required = append(required, amtName)
			}
			continue
		}
		s := fieldToSchema(f)
		if s != nil {
			props.Set(f.Name, s)
			if !f.Optional {
				required = append(required, f.Name)
			}
		}
	}

	schema.Set("properties", props)
	if len(required) > 0 {
		schema.Set("required", required)
	}
	return schema
}

func buildUpdateSchema(ent *entityInfo) *orderedMap {
	schema := newOrderedMap()
	schema.Set("type", "object")
	schema.Set("description", "All fields optional for partial updates")
	props := newOrderedMap()

	for _, f := range ent.Fields {
		if f.FieldType == "money" {
			amtName := f.Name + "_amount_cents"
			curName := f.Name + "_currency"
			props.Set(amtName, map[string]interface{}{"type": "integer", "format": "int64"})
			props.Set(curName, map[string]interface{}{"type": "string", "pattern": "^[A-Z]{3}$"})
			continue
		}
		s := fieldToSchema(f)
		if s != nil {
			props.Set(f.Name, s)
		}
	}

	schema.Set("properties", props)
	return schema
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

// ─── Main ────────────────────────────────────────────────────────────────────

func main() {
	log.SetFlags(0)
	log.SetPrefix("openapigen: ")

	ctx := cuecontext.New()
	projectRoot := findProjectRoot()

	// Load ontology
	insts := load.Instances([]string{"./ontology"}, &load.Config{Dir: projectRoot})
	if len(insts) == 0 || insts[0].Err != nil {
		log.Fatalf("loading ontology: %v", insts[0].Err)
	}
	val := ctx.BuildInstance(insts[0])
	if val.Err() != nil {
		log.Fatalf("building ontology: %v", val.Err())
	}

	entities := parseEntities(val)
	services := parseServices(ctx, projectRoot)

	// Build the OpenAPI spec using ordered maps for stable output
	spec := newOrderedMap()
	spec.Set("openapi", "3.1.0")
	spec.Set("info", map[string]interface{}{
		"title":       "Propeller Property Management API",
		"version":     "1.0.0",
		"description": "REST API generated from CUE ontology. All endpoints accept/return JSON.",
	})
	spec.Set("servers", []map[string]interface{}{
		{"url": "http://localhost:8080", "description": "Local development"},
	})

	// Build paths from operations
	paths := newOrderedMap()
	for _, svc := range services {
		for _, op := range svc.Operations {
			basePath := "/v1/" + op.EntityPath
			var path string
			switch op.Type {
			case "create":
				path = basePath
			case "get":
				path = basePath + "/{id}"
			case "list":
				path = basePath
			case "update":
				path = basePath + "/{id}"
			case "delete":
				path = basePath + "/{id}"
			case "transition":
				path = basePath + "/{id}/" + op.Action
			}

			method := httpMethod(op.Type)
			pathItem := buildPathItem(op, entities)

			// Get or create path entry
			var entry *orderedMap
			if existing, ok := paths.values[path]; ok {
				entry = existing.(*orderedMap)
			} else {
				entry = newOrderedMap()
				paths.Set(path, entry)
			}
			entry.Set(method, pathItem)
		}
	}
	spec.Set("paths", paths)

	// Build component schemas
	schemas := newOrderedMap()

	// Collect entity names used in operations, sorted
	entityNames := make([]string, 0, len(entities))
	for name := range entities {
		entityNames = append(entityNames, name)
	}
	sort.Strings(entityNames)

	// Track which entities need Create/Update schemas
	needsCreate := map[string]bool{}
	needsUpdate := map[string]bool{}
	for _, svc := range services {
		for _, op := range svc.Operations {
			switch op.Type {
			case "create":
				needsCreate[op.Entity] = true
			case "update":
				needsUpdate[op.Entity] = true
			}
		}
	}

	for _, name := range entityNames {
		ent := entities[name]
		schemas.Set(name, buildEntitySchema(ent))
		if needsCreate[name] {
			schemas.Set(name+"Create", buildCreateSchema(ent))
		}
		if needsUpdate[name] {
			schemas.Set(name+"Update", buildUpdateSchema(ent))
		}
	}

	// Error response schema
	schemas.Set("Error", map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"code":    map[string]interface{}{"type": "string"},
			"message": map[string]interface{}{"type": "string"},
		},
		"required": []string{"code", "message"},
	})

	components := newOrderedMap()
	components.Set("schemas", schemas)
	spec.Set("components", components)

	// Write output
	outPath := filepath.Join(projectRoot, "gen", "openapi", "openapi.json")
	data, err := json.MarshalIndent(spec, "", "    ")
	if err != nil {
		log.Fatalf("marshaling OpenAPI spec: %v", err)
	}
	// Add trailing newline
	data = append(data, '\n')
	if err := os.WriteFile(outPath, data, 0644); err != nil {
		log.Fatalf("writing %s: %v", outPath, err)
	}

	fmt.Printf("openapigen: generated %s (%d bytes, %d paths, %d schemas)\n",
		outPath, len(data), len(paths.keys), len(schemas.keys))
}
