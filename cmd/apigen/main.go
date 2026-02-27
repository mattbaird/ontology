// cmd/apigen generates Connect-RPC .proto service definitions from the CUE ontology
// and codegen/apigen.cue service mappings.
//
// It reads entity definitions from ontology/*.cue and service definitions from
// codegen/apigen.cue, then generates .proto files in gen/proto/.
package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/load"
)

type serviceDef struct {
	Name       string
	BasePath   string
	Entities   []string
	Operations []operationDef
}

type operationDef struct {
	Name        string
	Entity      string
	Type        string // create, get, list, update, delete, transition
	FromStatus  []string
	ToStatus    string
	ExtraFields []string
	Description string
}

type entityField struct {
	Name       string
	ProtoType  string
	Number     int
	Optional   bool
	Repeated   bool
	Comment    string
	Deprecated bool // @deprecated() — add proto comment
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("apigen: ")

	projectRoot := findProjectRoot()
	ctx := cuecontext.New()

	// Load ontology
	ontInsts := load.Instances([]string{"./ontology"}, &load.Config{Dir: projectRoot})
	if len(ontInsts) == 0 || ontInsts[0].Err != nil {
		log.Fatalf("loading ontology: %v", ontInsts[0].Err)
	}
	ontVal := ctx.BuildInstance(ontInsts[0])
	if ontVal.Err() != nil {
		log.Fatalf("building ontology: %v", ontVal.Err())
	}

	// Load apigen config
	apiInsts := load.Instances([]string{"./codegen"}, &load.Config{Dir: projectRoot})
	if len(apiInsts) == 0 || apiInsts[0].Err != nil {
		log.Fatalf("loading codegen: %v", apiInsts[0].Err)
	}
	apiVal := ctx.BuildInstance(apiInsts[0])
	if apiVal.Err() != nil {
		log.Fatalf("building codegen: %v", apiVal.Err())
	}

	// Parse services
	services := parseServices(apiVal)

	// Parse entities for message generation
	entities := parseEntityMessages(ontVal)

	// Generate proto files
	for _, svc := range services {
		if err := generateProto(projectRoot, svc, entities); err != nil {
			log.Fatalf("generating proto for %s: %v", svc.Name, err)
		}
		fmt.Printf("Generated gen/proto/%s.proto\n", toSnake(svc.Name))
	}
}

func parseServices(val cue.Value) []serviceDef {
	var services []serviceDef
	svcList := val.LookupPath(cue.ParsePath("services"))
	if svcList.Err() != nil {
		log.Fatalf("no services found: %v", svcList.Err())
	}

	iter, _ := svcList.List()
	for iter.Next() {
		svc := iter.Value()
		s := serviceDef{}
		s.Name, _ = svc.LookupPath(cue.ParsePath("name")).String()
		s.BasePath, _ = svc.LookupPath(cue.ParsePath("base_path")).String()

		// Parse entities
		entIter, _ := svc.LookupPath(cue.ParsePath("entities")).List()
		for entIter.Next() {
			e, _ := entIter.Value().String()
			s.Entities = append(s.Entities, e)
		}

		// Parse operations
		opIter, _ := svc.LookupPath(cue.ParsePath("operations")).List()
		for opIter.Next() {
			op := opIter.Value()
			o := operationDef{}
			o.Name, _ = op.LookupPath(cue.ParsePath("name")).String()
			o.Entity, _ = op.LookupPath(cue.ParsePath("entity")).String()
			o.Type, _ = op.LookupPath(cue.ParsePath("type")).String()
			o.Description, _ = op.LookupPath(cue.ParsePath("description")).String()
			o.ToStatus, _ = op.LookupPath(cue.ParsePath("to_status")).String()

			fromIter, _ := op.LookupPath(cue.ParsePath("from_status")).List()
			for fromIter.Next() {
				st, _ := fromIter.Value().String()
				o.FromStatus = append(o.FromStatus, st)
			}

			extraIter, _ := op.LookupPath(cue.ParsePath("extra_fields")).List()
			for extraIter.Next() {
				ef, _ := extraIter.Value().String()
				o.ExtraFields = append(o.ExtraFields, ef)
			}

			s.Operations = append(s.Operations, o)
		}

		services = append(services, s)
	}

	return services
}

// parseEntityMessages extracts proto message field info from CUE entity definitions.
func parseEntityMessages(val cue.Value) map[string][]entityField {
	entities := make(map[string][]entityField)

	iter, _ := val.Fields(cue.Definitions(true))
	for iter.Next() {
		label := iter.Selector().String()
		defVal := iter.Value()

		// Check if entity (has id + audit)
		idField := defVal.LookupPath(cue.ParsePath("id"))
		auditField := defVal.LookupPath(cue.ParsePath("audit"))
		if idField.Err() != nil || auditField.Err() != nil {
			continue
		}

		name := strings.TrimPrefix(label, "#")
		// Skip base entity types — not domain entities
		if name == "BaseEntity" || name == "StatefulEntity" || name == "ImmutableEntity" {
			continue
		}
		var fields []entityField
		fieldNum := 1

		// Add id field
		fields = append(fields, entityField{
			Name:      "id",
			ProtoType: "string",
			Number:    fieldNum,
		})
		fieldNum++

		// Parse remaining fields
		fIter, _ := defVal.Fields(cue.Optional(true))
		for fIter.Next() {
			fname := strings.TrimSuffix(fIter.Selector().String(), "?")
			if fname == "id" || fname == "audit" || strings.HasPrefix(fname, "_") {
				continue
			}

			ef := entityField{
				Name:     fname,
				Number:   fieldNum,
				Optional: fIter.IsOptional(),
			}
			ef.ProtoType = cueToProtoType(fIter.Value())
			if a := fIter.Value().Attribute("deprecated"); a.Err() == nil {
				ef.Deprecated = true
				reason, _, _ := a.Lookup(0, "reason")
				if reason != "" {
					ef.Comment = "Deprecated: " + reason
				}
			}
			fields = append(fields, ef)
			fieldNum++
		}

		// Add audit fields
		for _, af := range []string{"created_at", "updated_at", "created_by", "updated_by", "source"} {
			fields = append(fields, entityField{
				Name:      af,
				ProtoType: protoTypeForAuditField(af),
				Number:    fieldNum,
			})
			fieldNum++
		}

		entities[name] = fields
	}

	return entities
}

func cueToProtoType(val cue.Value) string {
	// Check for reference to known types
	ref := findReference(val)
	switch ref {
	case "#Money", "#NonNegativeMoney", "#PositiveMoney":
		return "Money"
	case "#Address":
		return "Address"
	case "#DateRange":
		return "DateRange"
	case "#ContactMethod":
		return "ContactMethod"
	case "time.Time", "Time":
		return "google.protobuf.Timestamp"
	}

	kind := val.IncompleteKind()
	if kind == cue.BottomKind {
		kind = inferKindFromExpr(val)
	}

	switch kind {
	case cue.StringKind:
		return "string"
	case cue.IntKind:
		return "int64"
	case cue.FloatKind, cue.NumberKind:
		return "double"
	case cue.BoolKind:
		return "bool"
	case cue.ListKind:
		return "repeated string" // simplified
	default:
		return "string"
	}
}

func protoTypeForAuditField(name string) string {
	switch name {
	case "created_at", "updated_at":
		return "google.protobuf.Timestamp"
	default:
		return "string"
	}
}

// findReference recursively searches CUE value for type references.
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

func inferKindFromExpr(val cue.Value) cue.Kind {
	op, args := val.Expr()
	if op == cue.AndOp {
		for _, arg := range args {
			k := arg.IncompleteKind()
			if k != cue.BottomKind {
				return k
			}
			if inferred := inferKindFromExpr(arg); inferred != cue.BottomKind {
				return inferred
			}
		}
	}
	if op == cue.OrOp && len(args) > 0 {
		for _, arg := range args {
			k := arg.IncompleteKind()
			if k != cue.BottomKind {
				return k
			}
		}
	}
	return cue.BottomKind
}

func generateProto(projectRoot string, svc serviceDef, entities map[string][]entityField) error {
	var buf bytes.Buffer
	tmpl, err := template.New("proto").Funcs(template.FuncMap{
		"toSnake":     toSnake,
		"toPascal":    toPascal,
		"toLower":     strings.ToLower,
		"hasTimestamp": func(fields []entityField) bool {
			for _, f := range fields {
				if f.ProtoType == "google.protobuf.Timestamp" {
					return true
				}
			}
			return false
		},
		"hasCustomTypes": func(fields []entityField) bool {
			for _, f := range fields {
				switch f.ProtoType {
				case "Money", "Address", "DateRange", "ContactMethod":
					return true
				}
			}
			return false
		},
		"sortedEntities": func(ents []string) []string {
			sorted := make([]string, len(ents))
			copy(sorted, ents)
			sort.Strings(sorted)
			return sorted
		},
		"hasPrefix": strings.HasPrefix,
		"add": func(a, b int) int { return a + b },
	}).Parse(protoTemplate)
	if err != nil {
		return fmt.Errorf("parsing template: %w", err)
	}

	data := struct {
		Service  serviceDef
		Entities map[string][]entityField
	}{svc, entities}

	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("executing template: %w", err)
	}

	outPath := filepath.Join(projectRoot, "gen", "proto", toSnake(svc.Name)+".proto")
	return os.WriteFile(outPath, buf.Bytes(), 0644)
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

func toPascal(s string) string {
	parts := strings.Split(s, "_")
	for i, p := range parts {
		if len(p) > 0 {
			upper := strings.ToUpper(p)
			switch upper {
			case "ID", "URL", "API", "HTTP":
				parts[i] = upper
			default:
				parts[i] = strings.ToUpper(p[:1]) + p[1:]
			}
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
			log.Fatal("could not find project root")
		}
		dir = parent
	}
}

const protoTemplate = `// Code generated by cmd/apigen from CUE ontology. DO NOT EDIT.
syntax = "proto3";

package propeller.v1;

option go_package = "github.com/matthewbaird/ontology/gen/proto;propellerv1";

import "google/protobuf/timestamp.proto";

// ─── Common Types ────────────────────────────────────────────────────────────

message Money {
  int64 amount_cents = 1;
  string currency = 2; // ISO 4217
}

message Address {
  string line1 = 1;
  string line2 = 2;
  string city = 3;
  string state = 4;
  string postal_code = 5;
  string country = 6;
  double latitude = 7;
  double longitude = 8;
  string county = 9;
}

message DateRange {
  google.protobuf.Timestamp start = 1;
  google.protobuf.Timestamp end = 2;
}

message ContactMethod {
  string type = 1;
  string value = 2;
  bool primary = 3;
  bool verified = 4;
  bool opt_out = 5;
  string label = 6;
}

message SideEffect {
  string entity_type = 1;
  string entity_id = 2;
  string description = 3;
}

// ─── Entity Messages ─────────────────────────────────────────────────────────
{{- range $entName := sortedEntities .Service.Entities}}
{{- $fields := index $.Entities $entName}}

message {{$entName}} {
{{- range $fields}}
{{- if eq .ProtoType "google.protobuf.Timestamp"}}
  {{.ProtoType}} {{.Name}} = {{.Number}};
{{- else if eq .ProtoType "Money"}}
  Money {{.Name}} = {{.Number}};
{{- else if eq .ProtoType "Address"}}
  Address {{.Name}} = {{.Number}};
{{- else if eq .ProtoType "DateRange"}}
  DateRange {{.Name}} = {{.Number}};
{{- else if eq .ProtoType "ContactMethod"}}
  repeated ContactMethod {{.Name}} = {{.Number}};
{{- else if hasPrefix .ProtoType "repeated"}}
  {{.ProtoType}} {{.Name}} = {{.Number}};
{{- else}}
  {{.ProtoType}} {{.Name}} = {{.Number}};
{{- end}}
{{- end}}
}
{{- end}}

// ─── Service Definition ──────────────────────────────────────────────────────

service {{.Service.Name}} {
{{- range .Service.Operations}}
  // {{.Description}}
  rpc {{.Name}}({{.Name}}Request) returns ({{.Name}}Response);
{{- end}}
}

// ─── Request/Response Messages ───────────────────────────────────────────────
{{range .Service.Operations}}
{{- if eq .Type "create"}}
message {{.Name}}Request {
  {{.Entity}} {{toSnake .Entity}} = 1;
}

message {{.Name}}Response {
  {{.Entity}} {{toSnake .Entity}} = 1;
}
{{- else if eq .Type "get"}}
message {{.Name}}Request {
  string id = 1;
  repeated string include = 2; // Edge names to include
}

message {{.Name}}Response {
  {{.Entity}} {{toSnake .Entity}} = 1;
}
{{- else if eq .Type "list"}}
message {{.Name}}Request {
  int32 page_size = 1;
  string page_token = 2;
  string filter = 3;
  string order_by = 4;
}

message {{.Name}}Response {
  repeated {{.Entity}} {{toSnake .Entity}}s = 1;
  string next_page_token = 2;
  int32 total_count = 3;
}
{{- else if eq .Type "update"}}
message {{.Name}}Request {
  {{.Entity}} {{toSnake .Entity}} = 1;
  repeated string update_mask = 2; // Fields to update
}

message {{.Name}}Response {
  {{.Entity}} {{toSnake .Entity}} = 1;
}
{{- else if eq .Type "delete"}}
message {{.Name}}Request {
  string id = 1;
}

message {{.Name}}Response {}
{{- else if eq .Type "transition"}}
message {{.Name}}Request {
  string id = 1;
{{- range $i, $f := .ExtraFields}}
  string {{$f}} = {{add $i 2}};
{{- end}}
}

message {{.Name}}Response {
  {{.Entity}} {{toSnake .Entity}} = 1;
  repeated SideEffect side_effects = 2;
}
{{- end}}
{{end}}
`
