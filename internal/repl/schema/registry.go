// Package schema provides the entity metadata registry for the REPL.
//
// The registry is populated at init time by generated code (gen_registry.go)
// and consumed by the parser (autocomplete), planner (validation),
// and executor (dispatch).
package schema

// FieldType classifies how PQL treats a field for comparison operators
// and value coercion.
type FieldType int

const (
	FieldString FieldType = iota
	FieldInt
	FieldInt64
	FieldFloat
	FieldBool
	FieldTime
	FieldEnum
	FieldUUID
	FieldJSON
)

// String returns the PQL-visible type name.
func (ft FieldType) String() string {
	switch ft {
	case FieldString:
		return "string"
	case FieldInt:
		return "int"
	case FieldInt64:
		return "int64"
	case FieldFloat:
		return "float"
	case FieldBool:
		return "bool"
	case FieldTime:
		return "time"
	case FieldEnum:
		return "enum"
	case FieldUUID:
		return "uuid"
	case FieldJSON:
		return "json"
	default:
		return "unknown"
	}
}

// Comparable returns true if the field type supports comparison operators (>, <, >=, <=).
func (ft FieldType) Comparable() bool {
	switch ft {
	case FieldInt, FieldInt64, FieldFloat, FieldTime:
		return true
	default:
		return false
	}
}

// FieldMeta describes a single field on an entity.
type FieldMeta struct {
	Name      string    // PQL name (snake_case, e.g. "lease_type")
	EntColumn string    // Ent column constant value (e.g. "lease_type")
	Type      FieldType // Logical type for operator validation
	Optional  bool      // Whether the field is nullable
	Sensitive bool      // Whether the field is PII/@sensitive
	EnumValues []string // Non-nil for enum fields
}

// EdgeMeta describes a relationship edge on an entity.
type EdgeMeta struct {
	Name        string // PQL name (snake_case, e.g. "lease_spaces")
	Target      string // Target entity PQL name (e.g. "lease_space")
	Cardinality string // "O2O", "O2M", "M2O", "M2M"
	Unique      bool   // True for O2O and M2O (single result)
}

// EntitySchema holds the complete metadata for one entity.
type EntitySchema struct {
	Name            string                // PQL name (snake_case, e.g. "lease")
	EntName         string                // Go type name (PascalCase, e.g. "Lease")
	Fields          map[string]*FieldMeta // field name -> metadata
	Edges           map[string]*EdgeMeta  // edge name -> metadata
	FieldOrder      []string              // fields in ontology order
	EdgeOrder       []string              // edges in ontology order
	HasStateMachine bool
	Immutable       bool
	StateMachine    map[string][]string   // from_status -> valid targets (nil if !HasStateMachine)
}

// Registry holds schema metadata for all entities. It is populated at init
// time by generated code and is safe for concurrent read access.
type Registry struct {
	entities      map[string]*EntitySchema   // pql_name -> schema
	entityOrder   []string                   // sorted entity names
	stateMachines map[string]map[string][]string // entity -> status -> targets
}

// NewRegistry creates an empty registry.
func NewRegistry() *Registry {
	return &Registry{
		entities:      make(map[string]*EntitySchema),
		entityOrder:   nil,
		stateMachines: make(map[string]map[string][]string),
	}
}

// Register adds an entity schema to the registry.
func (r *Registry) Register(es *EntitySchema) {
	r.entities[es.Name] = es
	r.entityOrder = append(r.entityOrder, es.Name)
	if es.HasStateMachine && es.StateMachine != nil {
		r.stateMachines[es.Name] = es.StateMachine
	}
}

// Entity returns the schema for a named entity, or nil if not found.
func (r *Registry) Entity(name string) *EntitySchema {
	return r.entities[name]
}

// EntityNames returns all registered entity names in sorted order.
func (r *Registry) EntityNames() []string {
	return r.entityOrder
}

// StateMachine returns the state machine transitions for an entity, or nil.
func (r *Registry) StateMachine(entity string) map[string][]string {
	return r.stateMachines[entity]
}

// AllEntities returns all entity schemas.
func (r *Registry) AllEntities() map[string]*EntitySchema {
	return r.entities
}
