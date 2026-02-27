// cmd/uigen generates framework-agnostic UI schemas from CUE ontology definitions.
//
// This is Layer 1 of the UI generation system. It reads CUE definitions from
// ontology/*.cue and codegen/*.cue, then outputs JSON UI schemas to gen/ui/schema/.
//
// Layer 2 (cmd/uirender) reads these schemas and generates Svelte components.
//
// Field type mapping:
//
//	CUE Type                     UI Field Type
//	string (short)               "string"
//	string (description/notes)   "text"
//	int                          "int"
//	float                        "float"
//	bool                         "bool"
//	time.Time (date context)     "date"
//	time.Time                    "datetime"
//	enum ("a" | "b")             "enum"
//	#Money                       "money" variant:"any"
//	#NonNegativeMoney            "money" variant:"non_negative"
//	#PositiveMoney               "money" variant:"positive"
//	#Address                     "address"
//	#DateRange                   "date_range"
//	#ContactMethod               "contact_method"
//	[...string]                  "string_list"
//	[...#Struct]                 "embedded_array"
//	#Struct                      "embedded_object"
//	name_id + relationship       "entity_ref"
//	name_ids                     "entity_ref_list"
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

// ── Data structures ──────────────────────────────────────────────────────────

// UISchema is the top-level schema for a single entity.
type UISchema struct {
	Entity              string              `json:"entity"`
	DisplayName         string              `json:"display_name"`
	DisplayNamePlural   string              `json:"display_name_plural"`
	PrimaryDisplayField string              `json:"primary_display_field"`
	PrimaryDisplay      string              `json:"primary_display_template"`
	Fields              []UIFieldDef        `json:"fields"`
	Enums               map[string]UIEnum   `json:"enums"`
	Form                UIForm              `json:"form"`
	Detail              UIDetail            `json:"detail"`
	List                UIList              `json:"list"`
	Status              *UIStatus           `json:"status,omitempty"`
	StateMachine        *UIStateMachine     `json:"state_machine,omitempty"`
	Relationships       []UIRelationship    `json:"relationships"`
	Validation          UIValidation        `json:"validation"`
	API                 UIAPI               `json:"api"`
}

type UIFieldDef struct {
	Name                  string      `json:"name"`
	Type                  string      `json:"type"`
	EnumRef               string      `json:"enum_ref,omitempty"`
	ObjectRef             string      `json:"object_ref,omitempty"`
	RefEntity             string      `json:"ref_entity,omitempty"`
	RefDisplay            string      `json:"ref_display,omitempty"`
	RefFilter             any         `json:"ref_filter,omitempty"`
	MoneyVariant          string      `json:"money_variant,omitempty"`
	Required              bool        `json:"required"`
	Default               any         `json:"default"`
	Immutable             bool        `json:"immutable,omitempty"`
	ConditionallyRequired bool        `json:"conditionally_required,omitempty"`
	Label                 string      `json:"label"`
	HelpText              string      `json:"help_text,omitempty"`
	ControlsVisibility    bool        `json:"controls_visibility,omitempty"`
	ShowInCreate          bool        `json:"show_in_create"`
	ShowInUpdate          bool        `json:"show_in_update"`
	ShowInList            bool        `json:"show_in_list"`
	ShowInDetail          bool        `json:"show_in_detail"`
	Sortable              bool        `json:"sortable"`
	Filterable            bool        `json:"filterable"`
	FilterType            string      `json:"filter_type,omitempty"`
	EndRequired           *bool       `json:"end_required,omitempty"`
	EndConditional        *bool       `json:"end_conditionally_required,omitempty"`
	ListDisplayField      string      `json:"list_display_field,omitempty"`
	ListColumnLabel       string      `json:"list_column_label,omitempty"`
	MinItems              *int        `json:"min_items,omitempty"`
	Min                   any         `json:"min,omitempty"`
	Max                   any         `json:"max,omitempty"`
	Pattern               string      `json:"pattern,omitempty"`
	IsDisplayName         bool        `json:"is_display_name,omitempty"`
	IsSensitive           bool        `json:"is_sensitive,omitempty"`
	IsPII                 bool        `json:"is_pii,omitempty"`
	IsComputed            bool        `json:"is_computed,omitempty"`
	IsDeprecated          bool        `json:"is_deprecated,omitempty"`
	DeprecatedReason      string      `json:"deprecated_reason,omitempty"`
	DeprecatedSince       string      `json:"deprecated_since,omitempty"`
}

type UIEnum struct {
	Values []UIEnumValue `json:"values"`
	Groups []UIEnumGroup `json:"groups,omitempty"`
}

type UIEnumValue struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

type UIEnumGroup struct {
	Label  string   `json:"label"`
	Values []string `json:"values"`
}

type UIForm struct {
	Sections       []UIFormSection `json:"sections"`
	FieldOrderRule string          `json:"field_order_rule"`
}

type UIFormSection struct {
	ID                 string           `json:"id"`
	Title              string           `json:"title"`
	Collapsible        bool             `json:"collapsible"`
	InitiallyCollapsed *bool            `json:"initially_collapsed,omitempty"`
	Fields             []string         `json:"fields,omitempty"`
	EmbeddedObject     string           `json:"embedded_object,omitempty"`
	EmbeddedArray      string           `json:"embedded_array,omitempty"`
	VisibleWhen        *VisibilityRule  `json:"visible_when,omitempty"`
	RequiredWhen       *VisibilityRule  `json:"required_when,omitempty"`
}

type VisibilityRule struct {
	Field    string   `json:"field"`
	Operator string   `json:"operator"`
	Value    any      `json:"value,omitempty"`
	Values   []string `json:"values,omitempty"`
}

type UIDetail struct {
	Header          UIDetailHeader    `json:"header"`
	Sections        []UIDetailSection `json:"sections"`
	RelatedSections []UIRelatedSection `json:"related_sections,omitempty"`
}

type UIDetailHeader struct {
	TitleTemplate string `json:"title_template"`
	StatusField   string `json:"status_field,omitempty"`
	Actions       bool   `json:"actions"`
}

type UIDetailSection struct {
	ID             string          `json:"id"`
	Title          string          `json:"title"`
	Layout         string          `json:"layout"`
	Fields         []string        `json:"fields,omitempty"`
	EmbeddedObject string          `json:"embedded_object,omitempty"`
	DisplayMode    string          `json:"display_mode,omitempty"`
	VisibleWhen    *VisibilityRule `json:"visible_when,omitempty"`
}

type UIRelatedSection struct {
	Title         string   `json:"title"`
	Relationship  string   `json:"relationship"`
	Entity        string   `json:"entity"`
	Display       string   `json:"display"`
	IncludeFields []string `json:"include_fields,omitempty"`
}

type UIList struct {
	DefaultColumns    []UIListColumn `json:"default_columns"`
	MaxDefaultColumns int            `json:"max_default_columns"`
	Filters           []UIListFilter `json:"filters"`
	DefaultSort       UISort         `json:"default_sort"`
	RowClickAction    string         `json:"row_click_action"`
	BulkActions       bool           `json:"bulk_actions"`
}

type UIListColumn struct {
	Field     string `json:"field"`
	Label     string `json:"label,omitempty"`
	Width     string `json:"width"`
	Align     string `json:"align,omitempty"`
	DisplayAs string `json:"display_as,omitempty"`
	Component string `json:"component,omitempty"`
}

type UIListFilter struct {
	Field    string `json:"field"`
	Type     string `json:"type"`
	Label    string `json:"label,omitempty"`
	EnumRef  string `json:"enum_ref,omitempty"`
	RefEntity string `json:"ref_entity,omitempty"`
}

type UISort struct {
	Field     string `json:"field"`
	Direction string `json:"direction"`
}

type UIStatus struct {
	Field        string            `json:"field"`
	ColorMapping map[string]string `json:"color_mapping"`
}

type UIStateMachine struct {
	Transitions map[string][]UITransition `json:"transitions"`
}

type UITransition struct {
	Target         string   `json:"target"`
	Label          string   `json:"label"`
	Variant        string   `json:"variant"`
	Confirm        bool     `json:"confirm"`
	ConfirmMessage string   `json:"confirm_message,omitempty"`
	APIEndpoint    string   `json:"api_endpoint"`
	RequiresFields []string `json:"requires_fields,omitempty"`
}

type UIRelationship struct {
	Name          string `json:"name"`
	TargetEntity  string `json:"target_entity"`
	Cardinality   string `json:"cardinality"`
	APIEndpoint   string `json:"api_endpoint,omitempty"`
	DisplayInDetail bool `json:"display_in_detail"`
	DisplayMode   string `json:"display_mode,omitempty"`
}

type UIValidation struct {
	FieldRules     []UIFieldRule      `json:"field_rules"`
	CrossFieldRules []UICrossFieldRule `json:"cross_field_rules"`
}

type UIFieldRule struct {
	Field string `json:"field"`
	Rule  string `json:"rule"`
	Value any    `json:"value,omitempty"`
}

type UICrossFieldRule struct {
	ID          string          `json:"id"`
	Description string          `json:"description"`
	Condition   *VisibilityRule `json:"condition"`
	Then        UIFieldRule     `json:"then"`
	Message     string          `json:"message"`
}

type UIAPI struct {
	BasePath    string                    `json:"base_path"`
	Operations  map[string]UIAPIEndpoint  `json:"operations"`
	Transitions map[string]UIAPIEndpoint  `json:"transitions,omitempty"`
	Related     map[string]UIAPIEndpoint  `json:"related,omitempty"`
}

type UIAPIEndpoint struct {
	Method string `json:"method"`
	Path   string `json:"path"`
}

// ── Internal parse structures ────────────────────────────────────────────────

type entityInfo struct {
	name       string
	fields     []fieldInfo
	hasMachine bool
	machine    map[string][]string // status -> []target_status
}

type fieldInfo struct {
	name          string
	cueVal        cue.Value
	optional      bool
	uiType        string
	enumRef       string
	objectRef     string
	refEntity     string
	moneyVariant  string
	enumValues    []string
	min           string
	max           string
	pattern       string
	hasDefault    bool
	defaultVal    string
	isList        bool
	listElemRef   string
	isDisplayName bool
	attrs         fieldAttrs
}

type relationshipInfo struct {
	name        string
	from        string
	to          string
	edgeType    string // "o2m", "m2o", "o2o", "m2m"
	fromField   string
	toField     string
	required    bool
}

type operationInfo struct {
	name       string
	entity     string
	opType     string
	entityPath string
	action     string
	toStatus   string
	extraFields []string
	custom     bool
}

type serviceInfo struct {
	name      string
	basePath  string
	entities  []string
	operations []operationInfo
}

type uiOverride struct {
	displayName        string
	displayNamePlural  string
	primaryDisplay     string
}

// ── Known constants ──────────────────────────────────────────────────────────

var knownValueTypes = map[string]string{
	"#Money":              "Money",
	"#NonNegativeMoney":   "NonNegativeMoney",
	"#PositiveMoney":      "PositiveMoney",
	"#Address":            "Address",
	"#ContactMethod":      "ContactMethod",
	"#DateRange":          "DateRange",
	"#EntityRef":          "EntityRef",
	"#RentScheduleEntry":  "RentScheduleEntry",
	"#RecurringCharge":    "RecurringCharge",
	"#LateFeePolicy":      "LateFeePolicy",
	"#CAMTerms":           "CAMTerms",
	"#TenantImprovement":  "TenantImprovement",
	"#RenewalOption":      "RenewalOption",
	"#SubsidyTerms":       "SubsidyTerms",
	"#AccountDimensions":  "AccountDimensions",
	"#JournalLine":        "JournalLine",
	"#RoleAttributes":     "RoleAttributes",
	"#TenantAttributes":   "TenantAttributes",
	"#OwnerAttributes":    "OwnerAttributes",
	"#ManagerAttributes":  "ManagerAttributes",
	"#GuarantorAttributes":"GuarantorAttributes",
	"#UsageBasedCharge":   "UsageBasedCharge",
	"#PercentageRent":     "PercentageRent",
	"#RentAdjustment":     "RentAdjustment",
	"#ExpansionRight":     "ExpansionRight",
	"#ContractionRight":   "ContractionRight",
	"#CAMCategoryTerms":   "CAMCategoryTerms",
}

var moneyTypes = map[string]string{
	"#Money":            "any",
	"#NonNegativeMoney": "non_negative",
	"#PositiveMoney":    "positive",
}

// State machines are read from the unified #StateMachines map in CUE.

// knownEntityNames is populated in main() before field classification runs.
// Used to validate that _id/_ids suffixed fields actually reference entities.
var knownEntityNames = map[string]bool{}

// edgeToEntity maps relationship edge names to their target entity snake_case names.
// Populated in main() from parsed relationships. Used as fallback when _id fields
// reference entities by edge name rather than entity name (e.g., owner_id → organization).
var edgeToEntity = map[string]string{}

// entityDisplayField maps entity snake_case name to its primary display field.
// Populated in main() after parsing entities. Used to set ref_display on entity_ref fields.
var entityDisplayField = map[string]string{}

// textFieldIndicators is no longer used — text fields are now identified by
// the @text() attribute in CUE rather than type references or name heuristics.

// Known abbreviations for enum label generation.
var knownAbbreviations = map[string]string{
	"nnn": "NNN", "nn": "NN", "cam": "CAM", "ach": "ACH",
	"cpi": "CPI", "nsf": "NSF", "hud": "HUD", "lihtc": "LIHTC",
	"ami": "AMI", "id": "ID", "uuid": "UUID", "url": "URL",
	"ssn": "SSN", "dba": "DBA", "ein": "EIN", "itin": "ITIN",
	"hoa": "HOA", "ada": "ADA", "hvac": "HVAC", "pbv": "PBV",
	"vash": "VASH", "gl": "GL",
}

// Known phrases for enum labels.
var knownPhrases = map[string]string{
	"section_8":                 "Section 8",
	"month_to_month":            "Month to Month",
	"month_to_month_holdover":   "Month-to-Month",
	"joint_and_several":         "Joint and Several",
	"by_the_bed":                "By the Bed",
	"not_started":               "Not Started",
	"in_progress":               "In Progress",
	"single_family":             "Single Family",
	"multi_family":              "Multi-Family",
	"mixed_use":                 "Mixed Use",
	"notice_given":              "Notice Given",
	"make_ready":                "Make Ready",
	"owner_occupied":            "Owner Occupied",
	"common_area":               "Common Area",
	"lot_pad":                   "Lot / Pad",
	"bed_space":                 "Bed Space",
	"desk_space":                "Desk Space",
	"per_day":                   "Per Day",
	"full_service":              "Full Service",
}

// Active states → success color.
var activeStates = map[string]bool{
	"active": true, "occupied": true, "posted": true,
	"approved": true, "balanced": true, "verified": true,
}

// Initial/draft states → surface color.
var initialStates = map[string]bool{
	"draft": true, "submitted": true, "onboarding": true,
}

// Negative states → error color.
var negativeStates = map[string]bool{
	"eviction": true, "denied": true, "down": true,
	"frozen": true, "voided": true,
}

// Warning states → warning color.
var warningStates = map[string]bool{
	"expired": true, "notice_given": true, "make_ready": true,
	"unbalanced": true, "month_to_month_holdover": true,
}

// Terminal/negative transition targets → danger variant.
var dangerTargets = map[string]bool{
	"terminated": true, "eviction": true, "denied": true,
	"voided": true, "dissolved": true, "closed": true,
}

// Forward transition targets → primary variant.
var primaryTargets = map[string]bool{
	"active": true, "approved": true, "posted": true,
	"pending_approval": true, "pending_signature": true,
	"occupied": true, "balanced": true, "renewed": true,
	"screening": true, "under_review": true,
	"conditionally_approved": true,
}

// ── CUE attribute extraction ─────────────────────────────────────────────────

// fieldAttrs holds cross-cutting metadata read from CUE @attr() annotations.
type fieldAttrs struct {
	display          bool
	text             bool
	immutable        bool
	computed         bool
	sensitive        bool
	pii              bool
	deprecated       bool
	deprecatedReason string
	deprecatedSince  string
}

// extractAttributes reads CUE field-level attributes from a value.
func extractAttributes(v cue.Value) fieldAttrs {
	var fa fieldAttrs
	for _, name := range []string{"display", "text", "immutable", "computed", "sensitive", "pii"} {
		a := v.Attribute(name)
		if a.Err() != nil {
			continue
		}
		switch name {
		case "display":
			fa.display = true
		case "text":
			fa.text = true
		case "immutable":
			fa.immutable = true
		case "computed":
			fa.computed = true
		case "sensitive":
			fa.sensitive = true
		case "pii":
			fa.pii = true
		}
	}
	if a := v.Attribute("deprecated"); a.Err() == nil {
		fa.deprecated = true
		fa.deprecatedReason, _, _ = a.Lookup(0, "reason")
		fa.deprecatedSince, _, _ = a.Lookup(0, "since")
	}
	return fa
}

// ── Name conversion utilities ────────────────────────────────────────────────

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
		if p == "" {
			continue
		}
		upper := strings.ToUpper(p)
		if _, ok := knownAbbreviations[strings.ToLower(p)]; ok {
			parts[i] = knownAbbreviations[strings.ToLower(p)]
		} else {
			parts[i] = strings.ToUpper(upper[:1]) + strings.ToLower(upper[1:])
		}
	}
	return strings.Join(parts, "")
}

func toCamel(s string) string {
	p := toPascal(s)
	if len(p) == 0 {
		return p
	}
	return strings.ToLower(p[:1]) + p[1:]
}

func toLabel(s string) string {
	// Check known phrases first
	if label, ok := knownPhrases[s]; ok {
		return label
	}
	return generateEnumLabel(s)
}

func generateEnumLabel(value string) string {
	if value == "" {
		return ""
	}

	// Check known phrases
	if label, ok := knownPhrases[value]; ok {
		return label
	}

	parts := strings.Split(value, "_")
	for i, p := range parts {
		if p == "" {
			continue
		}
		if abbr, ok := knownAbbreviations[strings.ToLower(p)]; ok {
			parts[i] = abbr
		} else {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, " ")
}

// fieldLabel converts a snake_case field name to a human-readable label.
func fieldLabel(name string, isEntityRef bool) string {
	clean := name
	// Only remove _id/_ids suffix for actual entity reference fields
	if isEntityRef {
		clean = strings.TrimSuffix(strings.TrimSuffix(name, "_ids"), "_id")
	}
	return generateEnumLabel(clean)
}

// ── CUE parsing utilities ────────────────────────────────────────────────────

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

func isEnum(val cue.Value) bool {
	op, args := findEnumDisjunction(val)
	if op != cue.OrOp || len(args) < 2 {
		return false
	}
	for _, arg := range args {
		aOp, aArgs := arg.Expr()
		check := arg
		if aOp == cue.SelectorOp && len(aArgs) > 0 {
			check = aArgs[0]
		}
		if check.IncompleteKind() != cue.StringKind {
			return false
		}
		if _, err := check.String(); err != nil {
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
	return true
}

// findEnumDisjunction extracts the OrOp disjunction from a value that may be
// wrapped in an AndOp (e.g., when a field from an embedded base type is
// unified with enum values in the entity).
func findEnumDisjunction(val cue.Value) (cue.Op, []cue.Value) {
	op, args := val.Expr()
	if op == cue.OrOp {
		return op, args
	}
	// Resolve definition references (e.g., #USState)
	if op != cue.OrOp {
		deref := cue.Dereference(val)
		dOp, dArgs := deref.Expr()
		if dOp == cue.OrOp {
			return dOp, dArgs
		}
	}
	// When embedding introduces `string & ("a" | "b" | "c")`, look through AndOp
	if op == cue.AndOp {
		for _, arg := range args {
			argOp, argArgs := arg.Expr()
			if argOp == cue.OrOp {
				return argOp, argArgs
			}
		}
	}
	return op, args
}

func extractEnumValues(val cue.Value) []string {
	op, args := findEnumDisjunction(val)
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
	if op == cue.OrOp {
		if len(args) > 0 {
			for _, arg := range args {
				k := arg.IncompleteKind()
				if k != cue.BottomKind {
					return k
				}
			}
		}
	}
	return cue.BottomKind
}

func inferListElementKind(val cue.Value) (cue.Value, bool) {
	elemVal := val.LookupPath(cue.MakePath(cue.AnyIndex))
	if elemVal.Err() == nil {
		return elemVal, true
	}
	op, args := val.Expr()
	if op == cue.AndOp {
		for _, arg := range args {
			if arg.IncompleteKind() == cue.ListKind || inferKindFromExpr(arg) == cue.ListKind {
				ev := arg.LookupPath(cue.MakePath(cue.AnyIndex))
				if ev.Err() == nil {
					return ev, true
				}
			}
		}
	}
	return cue.Value{}, false
}

func extractNumericBounds(val cue.Value) (string, string) {
	op, args := val.Expr()
	lo, hi := "", ""
	if op == cue.AndOp {
		for _, arg := range args {
			aLo, aHi := extractNumericBounds(arg)
			if aLo != "" {
				lo = aLo
			}
			if aHi != "" {
				hi = aHi
			}
		}
		return lo, hi
	}
	switch op {
	case cue.GreaterThanEqualOp, cue.GreaterThanOp:
		if len(args) >= 2 {
			lo = fmt.Sprint(args[1])
		}
	case cue.LessThanEqualOp, cue.LessThanOp:
		if len(args) >= 2 {
			hi = fmt.Sprint(args[1])
		}
	}
	return lo, hi
}

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
	return ""
}

func hasNonEmpty(val cue.Value) bool {
	op, args := val.Expr()
	if op == cue.NotEqualOp && len(args) > 1 {
		if s, err := args[1].String(); err == nil && s == "" {
			return true
		}
	}
	// Detect strings.MinRunes(1) CallOp pattern
	if op == cue.CallOp && len(args) >= 2 {
		if n, err := args[1].Int64(); err == nil && n >= 1 {
			return true
		}
	}
	if op == cue.AndOp {
		for _, arg := range args {
			if hasNonEmpty(arg) {
				return true
			}
		}
	}
	return false
}

// ── Entity parsing ───────────────────────────────────────────────────────────

func parseEntities(val cue.Value) map[string]*entityInfo {
	entities := make(map[string]*entityInfo)

	iter, _ := val.Fields(cue.Definitions(true))
	for iter.Next() {
		label := iter.Selector().String()
		defVal := iter.Value()

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
		ent := &entityInfo{
			name: name,
		}
		ent.fields = parseEntityFields(name, defVal)
		entities[name] = ent
	}
	return entities
}

func parseEntityFields(entityName string, structVal cue.Value) []fieldInfo {
	var fields []fieldInfo
	iter, _ := structVal.Fields(cue.Optional(true))
	for iter.Next() {
		label := iter.Selector().String()
		fieldVal := iter.Value()
		label = strings.TrimSuffix(label, "?")

		if label == "id" || label == "audit" {
			continue
		}
		if strings.HasPrefix(label, "_") {
			continue
		}

		fi := classifyUIField(label, fieldVal, iter.IsOptional())
		if fi != nil {
			fi.attrs = extractAttributes(fieldVal)
			// Override type classification from attributes
			if fi.attrs.text && fi.uiType == "string" {
				fi.uiType = "text"
			}
			if fi.attrs.display {
				fi.isDisplayName = true
			}
			fields = append(fields, *fi)
		}
	}
	return fields
}

// classifyUIField maps a CUE field to one of the 16 UI field types.
func classifyUIField(name string, val cue.Value, optional bool) *fieldInfo {
	fi := &fieldInfo{
		name:     name,
		cueVal:   val,
		optional: optional,
	}

	// Check for time.Time
	if isTimeField(val) {
		// Distinguish date vs datetime by name heuristic
		lower := strings.ToLower(name)
		if strings.Contains(lower, "date") || strings.HasSuffix(lower, "_date") {
			fi.uiType = "date"
		} else {
			fi.uiType = "datetime"
		}
		return fi
	}

	// Check for references to known types
	refName := findReference(val)
	if refName != "" {
		// Money types
		if variant, ok := moneyTypes[refName]; ok {
			fi.uiType = "money"
			fi.moneyVariant = variant
			return fi
		}

		// Other known value types
		if typeName, ok := knownValueTypes[refName]; ok {
			if isList(val) {
				fi.uiType = "embedded_array"
				fi.objectRef = typeName
				fi.isList = true
				fi.listElemRef = typeName
			} else {
				switch typeName {
				case "Address":
					fi.uiType = "address"
				case "DateRange":
					fi.uiType = "date_range"
				case "ContactMethod":
					// Contact methods are usually lists, check
					if isList(val) {
						fi.uiType = "embedded_array"
						fi.objectRef = typeName
						fi.isList = true
					} else {
						fi.uiType = "contact_method"
					}
				default:
					fi.uiType = "embedded_object"
					fi.objectRef = typeName
				}
			}
			return fi
		}
	}

	// Check for list types
	if isList(val) {
		elemVal, found := inferListElementKind(val)
		if found {
			elemRef := findReference(elemVal)
			if elemRef != "" {
				if typeName, ok := knownValueTypes[elemRef]; ok {
					fi.uiType = "embedded_array"
					fi.objectRef = typeName
					fi.isList = true
					fi.listElemRef = typeName
					return fi
				}
			}
			// Check for contact methods list
			if elemRef == "#ContactMethod" {
				fi.uiType = "embedded_array"
				fi.objectRef = "ContactMethod"
				fi.isList = true
				return fi
			}
			// String list
			kind := elemVal.IncompleteKind()
			if kind == cue.BottomKind {
				kind = inferKindFromExpr(elemVal)
			}
			if kind == cue.StringKind {
				fi.uiType = "string_list"
				return fi
			}
		}
		// Fallback for list
		fi.uiType = "string_list"
		return fi
	}

	// Check for enum
	if isEnum(val) {
		fi.uiType = "enum"
		fi.enumValues = extractEnumValues(val)
		fi.enumRef = toPascal(name)
		// Check for default
		if d, ok := val.Default(); ok {
			if s, err := d.String(); err == nil {
				fi.hasDefault = true
				fi.defaultVal = s
			}
		}
		return fi
	}

	// Classify by kind
	kind := val.IncompleteKind()
	if kind == cue.BottomKind {
		kind = inferKindFromExpr(val)
	}

	switch kind {
	case cue.StringKind:
		// Check if _id or _ids → entity_ref (only if the referenced entity exists)
		if strings.HasSuffix(name, "_ids") {
			ref := strings.TrimSuffix(name, "_ids")
			if knownEntityNames[ref] {
				fi.uiType = "entity_ref_list"
				fi.refEntity = ref
				return fi
			}
			// Fallback: check relationship edge names (e.g., owner_ids → organization)
			if target, ok := edgeToEntity[ref]; ok {
				fi.uiType = "entity_ref_list"
				fi.refEntity = target
				return fi
			}
		}
		if strings.HasSuffix(name, "_id") {
			ref := strings.TrimSuffix(name, "_id")
			if knownEntityNames[ref] {
				fi.uiType = "entity_ref"
				fi.refEntity = ref
				return fi
			}
			// Fallback: check relationship edge names (e.g., owner_id → organization)
			if target, ok := edgeToEntity[ref]; ok {
				fi.uiType = "entity_ref"
				fi.refEntity = target
				return fi
			}
		}

		fi.uiType = "string"
		fi.pattern = extractPattern(val)
		return fi

	case cue.IntKind:
		fi.uiType = "int"
		fi.min, fi.max = extractNumericBounds(val)
		return fi

	case cue.FloatKind, cue.NumberKind:
		fi.uiType = "float"
		fi.min, fi.max = extractNumericBounds(val)
		return fi

	case cue.BoolKind:
		fi.uiType = "bool"
		if d, ok := val.Default(); ok {
			b, _ := d.Bool()
			fi.hasDefault = true
			if b {
				fi.defaultVal = "true"
			} else {
				fi.defaultVal = "false"
			}
		}
		return fi

	case cue.StructKind:
		fi.uiType = "embedded_object"
		fi.objectRef = "Unknown"
		return fi

	default:
		// Top type (_) or mixed kind → embedded_object (JSON)
		if kind != 0 && kind != cue.BottomKind {
			fi.uiType = "embedded_object"
			fi.objectRef = "JSON"
			return fi
		}
	}

	return nil
}

// ── Relationship parsing ─────────────────────────────────────────────────────

func parseRelationships(val cue.Value) []relationshipInfo {
	var rels []relationshipInfo

	relList := val.LookupPath(cue.ParsePath("relationships"))
	if relList.Err() != nil {
		return rels
	}

	iter, _ := relList.List()
	for iter.Next() {
		r := iter.Value()

		var rel relationshipInfo

		if v := r.LookupPath(cue.ParsePath("edge_name")); v.Err() == nil {
			rel.name, _ = v.String()
		}
		if v := r.LookupPath(cue.ParsePath("from")); v.Err() == nil {
			rel.from, _ = v.String()
		}
		if v := r.LookupPath(cue.ParsePath("to")); v.Err() == nil {
			rel.to, _ = v.String()
		}
		if v := r.LookupPath(cue.ParsePath("cardinality")); v.Err() == nil {
			c, _ := v.String()
			rel.edgeType = strings.ToLower(c)
		}
		if v := r.LookupPath(cue.ParsePath("inverse_name")); v.Err() == nil {
			rel.toField, _ = v.String()
		}
		if v := r.LookupPath(cue.ParsePath("required")); v.Err() == nil {
			rel.required, _ = v.Bool()
		}

		if rel.name == "" || rel.from == "" || rel.to == "" {
			continue
		}

		rels = append(rels, rel)
	}
	return rels
}

// ── State machine parsing ────────────────────────────────────────────────────

func parseStateMachines(val cue.Value, entities map[string]*entityInfo) {
	for entityName, ent := range entities {
		transVal := val.LookupPath(cue.ParsePath("#StateMachines." + toSnake(entityName)))
		if transVal.Err() != nil {
			continue
		}

		machine := make(map[string][]string)
		iter, _ := transVal.Fields()
		for iter.Next() {
			state := iter.Selector().String()
			var targets []string

			listIter, _ := iter.Value().List()
			for listIter.Next() {
				if s, err := listIter.Value().String(); err == nil {
					targets = append(targets, s)
				}
			}
			machine[state] = targets
		}

		ent.hasMachine = true
		ent.machine = machine
	}
}

// ── Operations parsing from codegen/apigen.cue ──────────────────────────────

func parseOperations(codegenVal cue.Value) []serviceInfo {
	var services []serviceInfo

	svcList := codegenVal.LookupPath(cue.ParsePath("services"))
	if svcList.Err() != nil {
		return services
	}

	iter, _ := svcList.List()
	for iter.Next() {
		svcVal := iter.Value()
		svc := serviceInfo{}

		if v := svcVal.LookupPath(cue.ParsePath("name")); v.Err() == nil {
			svc.name, _ = v.String()
		}
		if v := svcVal.LookupPath(cue.ParsePath("base_path")); v.Err() == nil {
			svc.basePath, _ = v.String()
		}

		entIter, _ := svcVal.LookupPath(cue.ParsePath("entities")).List()
		for entIter.Next() {
			if s, err := entIter.Value().String(); err == nil {
				svc.entities = append(svc.entities, s)
			}
		}

		opIter, _ := svcVal.LookupPath(cue.ParsePath("operations")).List()
		for opIter.Next() {
			opVal := opIter.Value()
			op := operationInfo{}

			if v := opVal.LookupPath(cue.ParsePath("name")); v.Err() == nil {
				op.name, _ = v.String()
			}
			if v := opVal.LookupPath(cue.ParsePath("entity")); v.Err() == nil {
				op.entity, _ = v.String()
			}
			if v := opVal.LookupPath(cue.ParsePath("type")); v.Err() == nil {
				op.opType, _ = v.String()
			}
			if v := opVal.LookupPath(cue.ParsePath("entity_path")); v.Err() == nil {
				op.entityPath, _ = v.String()
			}
			if v := opVal.LookupPath(cue.ParsePath("action")); v.Err() == nil {
				op.action, _ = v.String()
			}
			if v := opVal.LookupPath(cue.ParsePath("to_status")); v.Err() == nil {
				op.toStatus, _ = v.String()
			}
			if v := opVal.LookupPath(cue.ParsePath("custom")); v.Err() == nil {
				op.custom, _ = v.Bool()
			}

			efIter, _ := opVal.LookupPath(cue.ParsePath("extra_fields")).List()
			for efIter.Next() {
				if s, err := efIter.Value().String(); err == nil {
					op.extraFields = append(op.extraFields, s)
				}
			}

			svc.operations = append(svc.operations, op)
		}

		services = append(services, svc)
	}
	return services
}

// ── UI overrides parsing from codegen/uigen.cue ─────────────────────────────

func parseUIOverrides(codegenVal cue.Value) map[string]uiOverride {
	overrides := make(map[string]uiOverride)

	ov := codegenVal.LookupPath(cue.ParsePath("ui_entity_overrides"))
	if ov.Err() != nil {
		return overrides
	}

	iter, _ := ov.Fields()
	for iter.Next() {
		name := iter.Selector().String()
		v := iter.Value()

		o := uiOverride{}
		if dn := v.LookupPath(cue.ParsePath("display_name")); dn.Err() == nil {
			o.displayName, _ = dn.String()
		}
		if dp := v.LookupPath(cue.ParsePath("display_name_plural")); dp.Err() == nil {
			o.displayNamePlural, _ = dp.String()
		}
		if pt := v.LookupPath(cue.ParsePath("primary_display_template")); pt.Err() == nil {
			o.primaryDisplay, _ = pt.String()
		}
		overrides[name] = o
	}
	return overrides
}

// parseEnumGroupings reads enum grouping overrides from uigen.cue.
func parseEnumGroupings(codegenVal cue.Value) map[string]UIEnum {
	groupings := make(map[string]UIEnum)

	eg := codegenVal.LookupPath(cue.ParsePath("ui_enum_groupings"))
	if eg.Err() != nil {
		return groupings
	}

	iter, _ := eg.Fields()
	for iter.Next() {
		name := iter.Selector().String()
		v := iter.Value()

		e := UIEnum{}

		valIter, _ := v.LookupPath(cue.ParsePath("values")).List()
		for valIter.Next() {
			ev := UIEnumValue{}
			if vv := valIter.Value().LookupPath(cue.ParsePath("value")); vv.Err() == nil {
				ev.Value, _ = vv.String()
			}
			if lv := valIter.Value().LookupPath(cue.ParsePath("label")); lv.Err() == nil {
				ev.Label, _ = lv.String()
			}
			e.Values = append(e.Values, ev)
		}

		grpIter, _ := v.LookupPath(cue.ParsePath("groups")).List()
		for grpIter.Next() {
			g := UIEnumGroup{}
			if gl := grpIter.Value().LookupPath(cue.ParsePath("label")); gl.Err() == nil {
				g.Label, _ = gl.String()
			}
			gvIter, _ := grpIter.Value().LookupPath(cue.ParsePath("values")).List()
			for gvIter.Next() {
				if s, err := gvIter.Value().String(); err == nil {
					g.Values = append(g.Values, s)
				}
			}
			e.Groups = append(e.Groups, g)
		}

		groupings[name] = e
	}
	return groupings
}

// ── Schema building ──────────────────────────────────────────────────────────

func buildUISchema(
	ent *entityInfo,
	relationships []relationshipInfo,
	services []serviceInfo,
	overrides map[string]uiOverride,
	enumGroupings map[string]UIEnum,
	allEnums map[string]UIEnum,
) UISchema {
	snake := toSnake(ent.name)

	schema := UISchema{
		Entity:            snake,
		DisplayName:       ent.name,
		DisplayNamePlural: ent.name + "s",
		PrimaryDisplayField: "id",
		Enums:             make(map[string]UIEnum),
		Relationships:     []UIRelationship{},
	}

	// Apply overrides
	if o, ok := overrides[ent.name]; ok {
		schema.DisplayName = o.displayName
		schema.DisplayNamePlural = o.displayNamePlural
		if o.primaryDisplay != "" {
			schema.PrimaryDisplay = o.primaryDisplay
		}
	}

	// Build fields
	schema.Fields = buildFieldDefs(ent, relationships)

	// Build enums (from this entity's enum fields)
	for _, f := range ent.fields {
		if f.uiType == "enum" && len(f.enumValues) > 0 {
			enumName := toPascal(ent.name) + toPascal(f.name)
			// Special case: use standalone enum name if it's a "status" or well-known field
			if f.name == "status" {
				enumName = toPascal(ent.name) + "Status"
			} else if f.name == "lease_type" || f.name == "space_type" || f.name == "property_type" ||
				f.name == "account_type" || f.name == "org_type" || f.name == "building_type" ||
				f.name == "role_type" || f.name == "entry_type" || f.name == "source_type" ||
				f.name == "account_subtype" || f.name == "normal_balance" {
				enumName = toPascal(f.name)
			}

			// Check for override grouping — merge with ontology values so new
			// enum values added to CUE always appear even before the override is updated.
			if ge, ok := enumGroupings[enumName]; ok {
				// Build set of values already in the override
				overrideSet := make(map[string]bool, len(ge.Values))
				for _, v := range ge.Values {
					overrideSet[v.Value] = true
				}
				// Append any ontology values missing from the override
				for _, v := range f.enumValues {
					if !overrideSet[v] {
						ge.Values = append(ge.Values, UIEnumValue{
							Value: v,
							Label: generateEnumLabel(v),
						})
					}
				}
				schema.Enums[enumName] = ge
				allEnums[enumName] = ge
			} else {
				e := UIEnum{}
				for _, v := range f.enumValues {
					e.Values = append(e.Values, UIEnumValue{
						Value: v,
						Label: generateEnumLabel(v),
					})
				}
				schema.Enums[enumName] = e
				allEnums[enumName] = e
			}

			// Update field's enum_ref
			for i := range schema.Fields {
				if schema.Fields[i].Name == f.name {
					schema.Fields[i].EnumRef = enumName
				}
			}
		}
	}

	// Build form
	schema.Form = buildFormSchema(ent, schema.Fields)

	// Build detail
	schema.Detail = buildDetailSchema(ent, schema.Fields, relationships)

	// Build list
	schema.List = buildListSchema(ent, schema.Fields)

	// Build status
	if ent.hasMachine {
		schema.Status = buildStatusSchema(ent)
	}

	// Build state machine
	if ent.hasMachine {
		schema.StateMachine = buildStateMachineSchema(ent, services)
	}

	// Build relationships
	schema.Relationships = buildRelationships(ent, relationships, services)

	// Build validation
	schema.Validation = buildValidation(ent, schema.Fields)

	// Build API
	schema.API = buildAPISchema(ent, services)

	return schema
}

// buildFieldDefs builds UIFieldDef from parsed field info.
func buildFieldDefs(ent *entityInfo, relationships []relationshipInfo) []UIFieldDef {
	var fields []UIFieldDef

	for _, f := range ent.fields {
		fd := UIFieldDef{
			Name:          f.name,
			Type:          f.uiType,
			Required:      !f.optional,
			Label:         fieldLabel(f.name, f.uiType == "entity_ref" || f.uiType == "entity_ref_list"),
			IsDisplayName: f.isDisplayName,
			ShowInCreate:  true,
			ShowInUpdate:  true,
			ShowInList:    false,
			ShowInDetail:  true,
			Sortable:      false,
			Filterable:    false,
		}

		// Set default
		if f.hasDefault {
			fd.Default = f.defaultVal
		}

		// Type-specific enrichment
		switch f.uiType {
		case "enum":
			fd.EnumRef = f.enumRef
			fd.Sortable = true
			fd.Filterable = true
			fd.ShowInList = true
			// Type fields control visibility
			if strings.HasSuffix(f.name, "_type") || f.name == "is_sublease" || f.name == "requires_trust_accounting" || f.name == "is_trust" {
				fd.ControlsVisibility = true
			}

		case "money":
			fd.MoneyVariant = f.moneyVariant
			fd.Sortable = true
			fd.Filterable = true
			fd.FilterType = "money_range"
			fd.ShowInList = true

		case "entity_ref":
			fd.RefEntity = f.refEntity
			if df, ok := entityDisplayField[f.refEntity]; ok {
				fd.RefDisplay = df
			} else {
				fd.RefDisplay = "name"
			}
			fd.Sortable = true
			fd.Filterable = true
			fd.FilterType = "entity_ref"
			fd.ShowInList = true

			// Resolve display field from relationship
			for _, rel := range relationships {
				refSnake := strings.TrimSuffix(f.name, "_id")
				if toSnake(rel.to) == refSnake || rel.fromField == f.name || toSnake(rel.name) == refSnake {
					fd.RefEntity = toSnake(rel.to)
					if df, ok := entityDisplayField[fd.RefEntity]; ok {
						fd.RefDisplay = df
					}
					break
				}
			}

		case "entity_ref_list":
			fd.RefEntity = f.refEntity
			if df, ok := entityDisplayField[f.refEntity]; ok {
				fd.RefDisplay = df
			} else {
				fd.RefDisplay = "name"
			}
			fd.ShowInList = false
			one := 1
			fd.MinItems = &one

		case "date", "datetime":
			fd.Sortable = true
			fd.Filterable = true
			fd.FilterType = "date_range"

		case "date_range":
			fd.Sortable = true
			fd.Filterable = true
			fd.FilterType = "date_range"

		case "address":
			fd.ShowInList = false

		case "contact_method":
			fd.ShowInList = false

		case "embedded_object":
			fd.ObjectRef = f.objectRef
			fd.ShowInList = false
			fd.ShowInCreate = true
			fd.ShowInUpdate = true

		case "embedded_array":
			fd.ObjectRef = f.objectRef
			fd.ShowInList = false
			fd.ShowInCreate = true
			fd.ShowInUpdate = true

		case "string_list":
			fd.ShowInList = false

		case "string":
			fd.Sortable = true
			if f.pattern != "" {
				fd.Pattern = f.pattern
			}

		case "text":
			fd.ShowInList = false

		case "int":
			fd.Sortable = true
			if f.min != "" {
				fd.Min = f.min
			}
			if f.max != "" {
				fd.Max = f.max
			}

		case "float":
			fd.Sortable = true
			if f.min != "" {
				fd.Min = f.min
			}
			if f.max != "" {
				fd.Max = f.max
			}

		case "bool":
			fd.Sortable = true
			fd.Filterable = true
		}

		// Status field special handling
		if f.name == "status" {
			fd.ShowInCreate = false
			fd.ShowInUpdate = false
			fd.ShowInList = true
			fd.Sortable = true
			fd.Filterable = true
			fd.Immutable = true
		}

		// Attribute-driven metadata
		if f.attrs.computed {
			fd.IsComputed = true
			fd.ShowInCreate = false
			fd.ShowInUpdate = false
		}
		if f.attrs.immutable {
			fd.Immutable = true
			fd.ShowInUpdate = false
		}
		if f.attrs.sensitive {
			fd.IsSensitive = true
		}
		if f.attrs.pii {
			fd.IsPII = true
		}
		if f.attrs.deprecated {
			fd.IsDeprecated = true
			fd.DeprecatedReason = f.attrs.deprecatedReason
			fd.DeprecatedSince = f.attrs.deprecatedSince
		}

		fields = append(fields, fd)
	}

	return fields
}

// ── Form schema building ─────────────────────────────────────────────────────

// Hardcoded constraint map — same approach as entgen's assignConstraints.
// CUE v0.15.4 doesn't expose if blocks via API.
type constraintDef struct {
	field    string
	operator string
	value    any
	values   []string
	requires []string // fields that become visible/required
}

func getEntityConstraints(entityName string) []constraintDef {
	switch entityName {
	case "Lease":
		return []constraintDef{
			{field: "lease_type", operator: "in", values: []string{"fixed_term", "student"}, requires: []string{"term.end"}},
			{field: "lease_type", operator: "in", values: []string{"commercial_nnn", "commercial_nn", "commercial_n", "commercial_gross", "commercial_modified_gross"}, requires: []string{"cam_terms"}},
			{field: "lease_type", operator: "eq", value: "commercial_nnn", requires: []string{"cam_terms.includes_property_tax", "cam_terms.includes_insurance", "cam_terms.includes_utilities"}},
			{field: "lease_type", operator: "eq", value: "commercial_nn", requires: []string{"cam_terms.includes_property_tax", "cam_terms.includes_insurance"}},
			{field: "lease_type", operator: "eq", value: "commercial_n", requires: []string{"cam_terms.includes_property_tax"}},
			{field: "lease_type", operator: "in", values: []string{"section_8", "affordable"}, requires: []string{"subsidy"}},
			{field: "lease_type", operator: "eq", value: "section_8", requires: []string{"subsidy"}},
			{field: "is_sublease", operator: "eq", value: true, requires: []string{"parent_lease_id"}},
			{field: "status", operator: "in", values: []string{"active"}, requires: []string{"move_in_date"}},
			{field: "status", operator: "in", values: []string{"active", "expired", "renewed"}, requires: []string{"signed_at"}},
		}
	case "Space":
		return []constraintDef{
			{field: "space_type", operator: "eq", value: "residential_unit", requires: []string{"bedrooms", "bathrooms"}},
			{field: "space_type", operator: "in", values: []string{"parking", "storage", "lot_pad"}, requires: []string{"bedrooms", "bathrooms"}},
			{field: "space_type", operator: "eq", value: "common_area", requires: []string{"leasable"}},
			{field: "status", operator: "eq", value: "occupied", requires: []string{"active_lease_id"}},
		}
	case "Property":
		return []constraintDef{
			{field: "property_type", operator: "eq", value: "single_family", requires: []string{"total_spaces"}},
			{field: "property_type", operator: "eq", value: "affordable_housing", requires: []string{"compliance_programs"}},
			{field: "rent_controlled", operator: "eq", value: true, requires: []string{"jurisdiction_id"}},
		}
	case "Portfolio":
		return nil // Trust fields removed in v3
	case "Account":
		return []constraintDef{
			{field: "account_type", operator: "in", values: []string{"asset", "expense"}, requires: []string{"normal_balance"}},
			{field: "account_type", operator: "in", values: []string{"liability", "equity", "revenue"}, requires: []string{"normal_balance"}},
			{field: "is_header", operator: "eq", value: true, requires: []string{"allows_direct_posting"}},
			{field: "is_trust_account", operator: "eq", value: true, requires: []string{"trust_type"}},
		}
	case "LedgerEntry":
		return []constraintDef{
			{field: "entry_type", operator: "in", values: []string{"payment", "refund", "nsf"}, requires: []string{"person_id"}},
			{field: "entry_type", operator: "in", values: []string{"charge", "late_fee"}, requires: []string{"lease_id"}},
			{field: "entry_type", operator: "eq", value: "adjustment", requires: []string{"adjusts_entry_id"}},
			{field: "reconciled", operator: "eq", value: true, requires: []string{"reconciliation_id", "reconciled_at"}},
		}
	case "JournalEntry":
		return []constraintDef{
			{field: "source_type", operator: "eq", value: "manual", requires: []string{"approved_by", "approved_at"}},
			{field: "status", operator: "eq", value: "voided", requires: []string{"reversed_by_journal_id"}},
		}
	case "Application":
		return []constraintDef{
			{field: "status", operator: "in", values: []string{"approved", "conditionally_approved", "denied"}, requires: []string{"decision_by", "decision_at"}},
			{field: "status", operator: "eq", value: "denied", requires: []string{"decision_reason"}},
		}
	case "BankAccount":
		return nil // Trust fields removed in v3
	case "Reconciliation":
		return []constraintDef{
			{field: "status", operator: "in", values: []string{"balanced", "approved"}, requires: []string{"reconciled_by", "reconciled_at"}},
		}
	}
	return nil
}

func buildFormSchema(ent *entityInfo, fields []UIFieldDef) UIForm {
	form := UIForm{
		FieldOrderRule: "required_first",
	}

	constraints := getEntityConstraints(ent.name)

	// Build entity-specific form sections
	sections := buildEntityFormSections(ent, fields, constraints)
	form.Sections = sections
	return form
}

func buildEntityFormSections(ent *entityInfo, fields []UIFieldDef, constraints []constraintDef) []UIFormSection {
	var sections []UIFormSection
	switch ent.name {
	case "Lease":
		sections = buildLeaseFormSections(fields, constraints)
	case "Space":
		sections = buildSpaceFormSections(fields, constraints)
	case "Property":
		sections = buildPropertyFormSections(fields, constraints)
	case "Portfolio":
		sections = buildPortfolioFormSections(fields, constraints)
	case "Account":
		sections = buildAccountFormSections(fields, constraints)
	case "Application":
		sections = buildApplicationFormSections(fields, constraints)
	case "BankAccount":
		sections = buildBankAccountFormSections(fields, constraints)
	default:
		return buildGenericFormSections(ent, fields, constraints)
	}
	// Append any fields from the ontology not explicitly assigned to a section
	return appendUnassignedFields(sections, fields)
}

// appendUnassignedFields adds any schema fields not present in existing sections
// to an "Additional Details" catch-all section. This ensures new fields added to
// the CUE ontology always appear in the form even before the section builder is updated.
func appendUnassignedFields(sections []UIFormSection, fields []UIFieldDef) []UIFormSection {
	assigned := make(map[string]bool)
	for _, s := range sections {
		for _, f := range s.Fields {
			assigned[f] = true
		}
	}
	var extra []string
	for _, f := range fields {
		if assigned[f.Name] || !f.ShowInCreate {
			continue
		}
		// Skip embedded types — they use slots, not direct field rendering
		if f.Type == "embedded_object" || f.Type == "embedded_array" {
			continue
		}
		extra = append(extra, f.Name)
	}
	if len(extra) > 0 {
		sections = append(sections, UIFormSection{
			ID: "additional", Title: "Additional Details", Collapsible: true, Fields: extra,
		})
	}
	return sections
}

// commercialLeaseTypes is the list of commercial lease types for visibility rules.
var commercialLeaseTypes = []string{"commercial_nnn", "commercial_nn", "commercial_n", "commercial_gross", "commercial_modified_gross"}
var nnnLeaseTypes = []string{"commercial_nnn", "commercial_nn", "commercial_n"}

func boolPtr(b bool) *bool { return &b }

func buildLeaseFormSections(fields []UIFieldDef, constraints []constraintDef) []UIFormSection {
	return []UIFormSection{
		{ID: "identity", Title: "Lease Details", Fields: []string{"lease_type", "liability_type", "property_id", "tenant_role_ids", "guarantor_role_ids"}},
		{ID: "term", Title: "Lease Term", Fields: []string{"term", "lease_commencement_date", "rent_commencement_date"}},
		{ID: "financial", Title: "Financial Terms", Fields: []string{"base_rent", "security_deposit"}},
		{
			ID: "cam", Title: "Common Area Maintenance", Collapsible: true,
			VisibleWhen: &VisibilityRule{Field: "lease_type", Operator: "in", Values: commercialLeaseTypes},
			RequiredWhen: &VisibilityRule{Field: "lease_type", Operator: "in", Values: nnnLeaseTypes},
			EmbeddedObject: "CAMTerms",
		},
		{
			ID: "percentage_rent", Title: "Percentage Rent", Collapsible: true, InitiallyCollapsed: boolPtr(true),
			VisibleWhen: &VisibilityRule{Field: "lease_type", Operator: "in", Values: commercialLeaseTypes},
			EmbeddedObject: "PercentageRent",
		},
		{
			ID: "subsidy", Title: "Subsidy Terms", Collapsible: true,
			VisibleWhen: &VisibilityRule{Field: "lease_type", Operator: "in", Values: []string{"section_8", "affordable"}},
			RequiredWhen: &VisibilityRule{Field: "lease_type", Operator: "eq", Value: "section_8"},
			EmbeddedObject: "SubsidyTerms",
		},
		{
			ID: "short_term", Title: "Short-Term Rental", Collapsible: true,
			VisibleWhen: &VisibilityRule{Field: "lease_type", Operator: "eq", Value: "short_term"},
			Fields: []string{"check_in_time", "check_out_time", "cleaning_fee", "platform_booking_id"},
		},
		{
			ID: "membership", Title: "Membership", Collapsible: true,
			VisibleWhen: &VisibilityRule{Field: "lease_type", Operator: "eq", Value: "membership"},
			Fields: []string{"membership_tier"},
		},
		{ID: "rent_schedule", Title: "Rent Schedule", Collapsible: true, InitiallyCollapsed: boolPtr(true), EmbeddedArray: "RentScheduleEntry"},
		{ID: "recurring_charges", Title: "Recurring Charges", Collapsible: true, InitiallyCollapsed: boolPtr(true), EmbeddedArray: "RecurringCharge"},
		{
			ID: "usage_charges", Title: "Usage-Based Charges", Collapsible: true, InitiallyCollapsed: boolPtr(true),
			VisibleWhen: &VisibilityRule{Field: "lease_type", Operator: "in", Values: commercialLeaseTypes},
			EmbeddedArray: "UsageBasedCharge",
		},
		{ID: "renewal_options", Title: "Renewal Options", Collapsible: true, InitiallyCollapsed: boolPtr(true), EmbeddedArray: "RenewalOption"},
		{
			ID: "expansion", Title: "Expansion Rights", Collapsible: true, InitiallyCollapsed: boolPtr(true),
			VisibleWhen: &VisibilityRule{Field: "lease_type", Operator: "in", Values: commercialLeaseTypes},
			EmbeddedArray: "ExpansionRight",
		},
		{
			ID: "contraction", Title: "Contraction Rights", Collapsible: true, InitiallyCollapsed: boolPtr(true),
			VisibleWhen: &VisibilityRule{Field: "lease_type", Operator: "in", Values: commercialLeaseTypes},
			EmbeddedArray: "ContractionRight",
		},
		{
			ID: "tenant_improvement", Title: "Tenant Improvement Allowance", Collapsible: true, InitiallyCollapsed: boolPtr(true),
			VisibleWhen: &VisibilityRule{Field: "lease_type", Operator: "in", Values: commercialLeaseTypes},
			EmbeddedObject: "TenantImprovement",
		},
		{
			ID: "sublease", Title: "Sublease Details", Collapsible: true, InitiallyCollapsed: boolPtr(true),
			VisibleWhen: &VisibilityRule{Field: "is_sublease", Operator: "eq", Value: true},
			Fields: []string{"parent_lease_id", "sublease_billing"},
		},
		{ID: "late_fee", Title: "Late Fee Policy", Collapsible: true, InitiallyCollapsed: boolPtr(true), EmbeddedObject: "LateFeePolicy"},
		{ID: "signing", Title: "Signing", Collapsible: true, InitiallyCollapsed: boolPtr(true), Fields: []string{"signing_method", "document_id"}},
	}
}

func buildSpaceFormSections(fields []UIFieldDef, constraints []constraintDef) []UIFormSection {
	return []UIFormSection{
		{ID: "identity", Title: "Space Details", Fields: []string{"space_number", "space_type", "property_id", "building_id"}},
		{ID: "physical", Title: "Physical Attributes", Fields: []string{"square_footage", "floor"}},
		{
			ID: "residential", Title: "Residential Details", Collapsible: true,
			VisibleWhen: &VisibilityRule{Field: "space_type", Operator: "eq", Value: "residential_unit"},
			Fields: []string{"bedrooms", "bathrooms"},
		},
		{ID: "features", Title: "Features & Amenities", Collapsible: true, InitiallyCollapsed: boolPtr(true),
			Fields: []string{"amenities", "floor_plan", "ada_accessible", "pet_friendly", "furnished", "specialized_infrastructure"}},
		{ID: "financial", Title: "Financial", Collapsible: true, Fields: []string{"market_rent", "ami_restriction"}},
		{ID: "hierarchy", Title: "Hierarchy", Collapsible: true, InitiallyCollapsed: boolPtr(true),
			Fields: []string{"parent_space_id", "leasable", "shared_with_parent"}},
	}
}

func buildPropertyFormSections(fields []UIFieldDef, constraints []constraintDef) []UIFormSection {
	return []UIFormSection{
		{ID: "identity", Title: "Property Details", Fields: []string{"name", "property_type", "portfolio_id"}},
		{ID: "address", Title: "Address", Fields: []string{"address"}},
		{ID: "physical", Title: "Physical Details", Fields: []string{"year_built", "total_square_footage", "total_spaces", "lot_size_sqft", "stories", "parking_spaces"}},
		{ID: "regulatory", Title: "Regulatory", Collapsible: true,
			Fields: []string{"jurisdiction_id", "rent_controlled", "compliance_programs", "requires_lead_disclosure"}},
		{ID: "financial", Title: "Financial", Collapsible: true, Fields: []string{"chart_of_accounts_id", "bank_account_id"}},
		{ID: "insurance", Title: "Insurance", Collapsible: true, InitiallyCollapsed: boolPtr(true),
			Fields: []string{"insurance_policy_number", "insurance_expiry"}},
	}
}

func buildPortfolioFormSections(fields []UIFieldDef, constraints []constraintDef) []UIFormSection {
	return []UIFormSection{
		{ID: "identity", Title: "Portfolio Details", Fields: []string{"name", "owner_id", "management_type", "description"}},
		{ID: "financial", Title: "Financial Defaults", Collapsible: true,
			Fields: []string{"default_chart_of_accounts_id", "default_bank_account_id"}},
	}
}

func buildAccountFormSections(fields []UIFieldDef, constraints []constraintDef) []UIFormSection {
	return []UIFormSection{
		{ID: "identity", Title: "Account Details", Fields: []string{"account_number", "name", "description", "account_type", "account_subtype"}},
		{ID: "behavior", Title: "Behavior", Fields: []string{"normal_balance", "is_header", "is_system", "allows_direct_posting"}},
		{ID: "hierarchy", Title: "Hierarchy", Collapsible: true, Fields: []string{"parent_account_id", "depth", "dimensions"}},
		{ID: "trust", Title: "Trust", Collapsible: true,
			VisibleWhen: &VisibilityRule{Field: "is_trust_account", Operator: "eq", Value: true},
			Fields: []string{"is_trust_account", "trust_type"}},
		{ID: "budget", Title: "Budget & Tax", Collapsible: true, InitiallyCollapsed: boolPtr(true),
			Fields: []string{"budget_amount", "tax_line"}},
	}
}

func buildApplicationFormSections(fields []UIFieldDef, constraints []constraintDef) []UIFormSection {
	return []UIFormSection{
		{ID: "identity", Title: "Application Details", Fields: []string{"property_id", "space_id", "applicant_person_id"}},
		{ID: "timeline", Title: "Timeline", Fields: []string{"desired_move_in", "desired_lease_term_months"}},
		{ID: "screening", Title: "Screening", Collapsible: true,
			Fields: []string{"screening_request_id", "screening_completed", "credit_score", "background_clear", "income_verified", "income_to_rent_ratio"}},
		{ID: "decision", Title: "Decision", Collapsible: true,
			Fields: []string{"decision_by", "decision_at", "decision_reason", "conditions"}},
		{ID: "financial", Title: "Financial", Fields: []string{"application_fee", "fee_paid"}},
	}
}

func buildBankAccountFormSections(fields []UIFieldDef, constraints []constraintDef) []UIFormSection {
	return []UIFormSection{
		{ID: "identity", Title: "Bank Account Details", Fields: []string{"name", "account_type", "gl_account_id"}},
		{ID: "bank", Title: "Bank Information", Fields: []string{"institution_name", "routing_number", "account_mask", "account_number_encrypted"}},
		{ID: "scope", Title: "Scope", Collapsible: true, Fields: []string{"portfolio_id", "property_id", "entity_id"}},
		{ID: "capabilities", Title: "Capabilities", Collapsible: true,
			Fields: []string{"is_default", "accepts_deposits", "accepts_payments"}},
		{ID: "integration", Title: "Integration", Collapsible: true, InitiallyCollapsed: boolPtr(true),
			Fields: []string{"plaid_account_id", "plaid_access_token"}},
	}
}

func buildGenericFormSections(ent *entityInfo, fields []UIFieldDef, constraints []constraintDef) []UIFormSection {
	// Group fields into identity, main, and secondary sections
	var identityFields, mainFields, secondaryFields []string

	for _, f := range fields {
		if !f.ShowInCreate {
			continue
		}
		name := f.Name
		switch {
		case name == "status":
			// Skip status in forms
		case strings.HasSuffix(name, "_type") || strings.HasSuffix(name, "_id") || name == "name" || name == "legal_name":
			identityFields = append(identityFields, name)
		case f.Type == "embedded_object" || f.Type == "embedded_array":
			secondaryFields = append(secondaryFields, name)
		default:
			mainFields = append(mainFields, name)
		}
	}

	var sections []UIFormSection
	if len(identityFields) > 0 {
		sections = append(sections, UIFormSection{
			ID: "identity", Title: ent.name + " Details", Fields: identityFields,
		})
	}
	if len(mainFields) > 0 {
		sections = append(sections, UIFormSection{
			ID: "main", Title: "Details", Fields: mainFields,
		})
	}
	if len(secondaryFields) > 0 {
		sections = append(sections, UIFormSection{
			ID: "additional", Title: "Additional", Collapsible: true, InitiallyCollapsed: boolPtr(true), Fields: secondaryFields,
		})
	}
	return sections
}

// ── Detail schema building ───────────────────────────────────────────────────

func buildDetailSchema(ent *entityInfo, fields []UIFieldDef, relationships []relationshipInfo) UIDetail {
	detail := UIDetail{
		Header: UIDetailHeader{
			TitleTemplate: ent.name,
			Actions:       ent.hasMachine,
		},
	}

	// Set status field in header
	for _, f := range fields {
		if f.Name == "status" {
			detail.Header.StatusField = "status"
			break
		}
	}

	// Overview section — key fields
	var overviewFields []string
	for _, f := range fields {
		if f.ShowInDetail && f.Type != "embedded_object" && f.Type != "embedded_array" && f.Type != "text" && len(overviewFields) < 8 {
			overviewFields = append(overviewFields, f.Name)
		}
	}
	detail.Sections = append(detail.Sections, UIDetailSection{
		ID: "overview", Title: "Overview", Layout: "grid_2col", Fields: overviewFields,
	})

	// Add embedded object sections
	for _, f := range fields {
		if f.Type == "embedded_object" && f.ShowInDetail {
			detail.Sections = append(detail.Sections, UIDetailSection{
				ID:             toSnake(f.Name),
				Title:          f.Label,
				Layout:         "grid_2col",
				EmbeddedObject: f.ObjectRef,
				DisplayMode:    "readonly",
				VisibleWhen:    &VisibilityRule{Field: f.Name, Operator: "truthy"},
			})
		}
	}

	// Add related sections from relationships
	for _, rel := range relationships {
		if rel.from != ent.name {
			continue
		}
		displayMode := "list"
		if rel.edgeType == "o2m" {
			displayMode = "table"
		}
		detail.RelatedSections = append(detail.RelatedSections, UIRelatedSection{
			Title:        generateEnumLabel(rel.name),
			Relationship: rel.name,
			Entity:       toSnake(rel.to),
			Display:      displayMode,
		})
	}

	return detail
}

// ── List schema building ─────────────────────────────────────────────────────

func buildListSchema(ent *entityInfo, fields []UIFieldDef) UIList {
	list := UIList{
		MaxDefaultColumns: 7,
		DefaultSort:       UISort{Field: "updated_at", Direction: "desc"},
		RowClickAction:    "navigate_to_detail",
		BulkActions:       false,
	}

	// Select columns by priority
	var columns []UIListColumn
	var filters []UIListFilter
	addedFields := map[string]bool{}

	// Priority 0: display name — always include the entity's @display() field
	for _, f := range fields {
		if f.IsDisplayName && len(columns) < 7 {
			columns = append(columns, UIListColumn{
				Field: f.Name, Label: f.Label, Width: "200px",
			})
			addedFields[f.Name] = true
			break // Only the first display name field
		}
	}

	// Priority 1: status
	for _, f := range fields {
		if f.Name == "status" && len(columns) < 7 {
			statusComponent := "enum_badge"
			if ent.hasMachine {
				statusComponent = "status_badge"
			}
			columns = append(columns, UIListColumn{
				Field: "status", Width: "100px", Component: statusComponent,
			})
			filters = append(filters, UIListFilter{
				Field: "status", Type: "multi_enum", EnumRef: f.EnumRef, Label: "Status",
			})
			addedFields["status"] = true
		}
	}

	// Priority 2: type enum
	for _, f := range fields {
		if f.Type == "enum" && !addedFields[f.Name] && f.Name != "status" && strings.Contains(f.Name, "type") && len(columns) < 7 {
			columns = append(columns, UIListColumn{
				Field: f.Name, Label: f.Label, Width: "140px", Component: "enum_badge",
			})
			filters = append(filters, UIListFilter{
				Field: f.Name, Type: "multi_enum", EnumRef: f.EnumRef, Label: f.Label,
			})
			addedFields[f.Name] = true
			break // Only one type column
		}
	}

	// Priority 3: entity refs
	for _, f := range fields {
		if f.Type == "entity_ref" && !addedFields[f.Name] && len(columns) < 7 {
			displayAs := f.RefEntity + "." + f.RefDisplay
			if f.RefDisplay == "" {
				displayAs = f.RefEntity + ".name"
			}
			columns = append(columns, UIListColumn{
				Field: f.Name, Label: f.Label, Width: "180px",
				DisplayAs: displayAs,
			})
			filters = append(filters, UIListFilter{
				Field: f.Name, Type: "entity_ref", RefEntity: f.RefEntity, Label: f.Label,
			})
			addedFields[f.Name] = true
		}
	}

	// Priority 4: money fields
	for _, f := range fields {
		if f.Type == "money" && !addedFields[f.Name] && len(columns) < 7 {
			columns = append(columns, UIListColumn{
				Field: f.Name, Label: f.Label, Width: "120px", Align: "right", Component: "money",
			})
			filters = append(filters, UIListFilter{
				Field: f.Name, Type: "money_range", Label: f.Label,
			})
			addedFields[f.Name] = true
		}
	}

	// Priority 5: date fields
	for _, f := range fields {
		if (f.Type == "date" || f.Type == "datetime" || f.Type == "date_range") && !addedFields[f.Name] && f.Name != "updated_at" && f.Name != "created_at" && len(columns) < 7 {
			col := UIListColumn{
				Field: f.Name, Label: f.Label, Width: "120px", Component: "date",
			}
			if f.Type == "date_range" {
				col.Field = f.Name + ".end"
				col.Label = "End Date"
			}
			columns = append(columns, col)
			addedFields[f.Name] = true
		}
	}

	// Always add updated_at if room
	if len(columns) < 7 {
		columns = append(columns, UIListColumn{
			Field: "updated_at", Label: "Last Updated", Width: "140px", Component: "datetime",
		})
	}

	list.DefaultColumns = columns
	list.Filters = filters
	return list
}

// ── Status schema building ───────────────────────────────────────────────────

func buildStatusSchema(ent *entityInfo) *UIStatus {
	if !ent.hasMachine {
		return nil
	}

	status := &UIStatus{
		Field:        "status",
		ColorMapping: make(map[string]string),
	}

	// Determine terminal states (no outgoing transitions)
	terminalStates := make(map[string]bool)
	for state, targets := range ent.machine {
		if len(targets) == 0 {
			terminalStates[state] = true
		}
		_ = state
	}

	// Determine initial states (not a target of any transition)
	targetStates := make(map[string]bool)
	for _, targets := range ent.machine {
		for _, t := range targets {
			targetStates[t] = true
		}
	}

	for state := range ent.machine {
		color := classifyStateColor(state, terminalStates, targetStates)
		status.ColorMapping[state] = color
	}

	return status
}

func classifyStateColor(state string, terminalStates, targetStates map[string]bool) string {
	if activeStates[state] {
		return "success"
	}
	if negativeStates[state] {
		return "error"
	}
	if warningStates[state] {
		return "warning"
	}
	if terminalStates[state] {
		return "surface"
	}
	if initialStates[state] {
		return "surface"
	}
	// Initial states (not target of any transition)
	if !targetStates[state] {
		return "surface"
	}
	return "secondary"
}

// ── State machine schema building ────────────────────────────────────────────

func buildStateMachineSchema(ent *entityInfo, services []serviceInfo) *UIStateMachine {
	if !ent.hasMachine {
		return nil
	}

	sm := &UIStateMachine{
		Transitions: make(map[string][]UITransition),
	}

	// Find entity path and base path from services
	entityPath := ""
	basePath := ""
	transitionOps := make(map[string]operationInfo) // toStatus -> operation

	for _, svc := range services {
		for _, op := range svc.operations {
			if op.entity == ent.name && op.opType == "transition" {
				entityPath = op.entityPath
				basePath = svc.basePath
				transitionOps[op.toStatus] = op
			}
			if op.entity == ent.name && op.entityPath != "" {
				entityPath = op.entityPath
				basePath = svc.basePath
			}
		}
	}

	constraints := getEntityConstraints(ent.name)

	// Build transitions map
	states := sortedKeys(ent.machine)
	for _, state := range states {
		targets := ent.machine[state]
		var transitions []UITransition

		for _, target := range targets {
			variant := classifyTransitionVariant(target)
			label := generateTransitionLabel(state, target, ent.name)
			confirm := variant == "danger"

			t := UITransition{
				Target:  target,
				Label:   label,
				Variant: variant,
				Confirm: confirm,
			}

			if confirm {
				t.ConfirmMessage = generateConfirmMessage(target, ent.name)
			}

			// Find API endpoint
			if op, ok := transitionOps[target]; ok {
				t.APIEndpoint = fmt.Sprintf("POST %s/%s/{id}/%s", basePath, entityPath, op.action)
				t.RequiresFields = op.extraFields
			} else if entityPath != "" {
				// Default endpoint
				action := strings.ReplaceAll(target, "_", "-")
				t.APIEndpoint = fmt.Sprintf("POST %s/%s/{id}/%s", basePath, entityPath, action)
			}

			// Check for requires_fields from constraints
			for _, c := range constraints {
				if c.field == "status" {
					for _, v := range c.values {
						if v == target {
							t.RequiresFields = append(t.RequiresFields, c.requires...)
						}
					}
					if cVal, ok := c.value.(string); ok && cVal == target {
						t.RequiresFields = append(t.RequiresFields, c.requires...)
					}
				}
			}
			// Deduplicate requires_fields
			t.RequiresFields = dedup(t.RequiresFields)

			transitions = append(transitions, t)
		}

		sm.Transitions[state] = transitions
	}

	return sm
}

func classifyTransitionVariant(target string) string {
	if dangerTargets[target] {
		return "danger"
	}
	if primaryTargets[target] {
		return "primary"
	}
	return "secondary"
}

func generateTransitionLabel(from, to, entityName string) string {
	// Special case labels
	switch to {
	case "active":
		return "Activate"
	case "inactive":
		return "Deactivate"
	case "terminated":
		if from == "draft" || from == "pending_approval" || from == "pending_signature" {
			return "Cancel"
		}
		return "Terminate"
	case "eviction":
		return "Initiate Eviction"
	case "pending_approval":
		return "Submit for Approval"
	case "pending_signature":
		if from == "pending_approval" {
			return "Approve"
		}
		return "Send for Signature"
	case "draft":
		return "Return to Draft"
	case "renewed":
		return "Renew"
	case "month_to_month_holdover":
		return "Convert to Month-to-Month"
	case "expired":
		return "Mark Expired"
	case "voided":
		return "Void"
	case "posted":
		return "Post"
	case "approved":
		return "Approve"
	case "denied":
		return "Deny"
	case "dissolved":
		return "Dissolve"
	case "suspended":
		return "Suspend"
	case "closed":
		return "Close"
	case "frozen":
		return "Freeze"
	case "balanced":
		return "Mark Balanced"
	case "unbalanced":
		return "Mark Unbalanced"
	case "withdrawn":
		return "Withdraw"
	case "screening":
		return "Begin Screening"
	case "under_review":
		return "Move to Review"
	case "conditionally_approved":
		return "Conditionally Approve"
	case "occupied":
		return "Occupy"
	case "notice_given":
		return "Record Notice"
	case "make_ready":
		return "Start Make Ready"
	case "vacant":
		return "Mark Vacant"
	case "down":
		return "Mark Down"
	case "model":
		return "Mark as Model"
	case "reserved":
		return "Reserve"
	case "owner_occupied":
		return "Mark Owner Occupied"
	case "onboarding":
		return "Start Onboarding"
	case "offboarding":
		return "Start Offboarding"
	case "for_sale":
		return "List for Sale"
	case "under_renovation":
		return "Start Renovation"
	case "in_progress":
		return "Start"
	}
	return generateEnumLabel(to)
}

func generateConfirmMessage(target, entityName string) string {
	lower := strings.ToLower(entityName)
	switch target {
	case "terminated":
		return fmt.Sprintf("Are you sure you want to terminate this %s? This cannot be undone.", lower)
	case "eviction":
		return fmt.Sprintf("Are you sure you want to initiate eviction proceedings on this %s?", lower)
	case "voided":
		return fmt.Sprintf("Are you sure you want to void this %s?", lower)
	case "dissolved":
		return fmt.Sprintf("Are you sure you want to dissolve this %s?", lower)
	case "closed":
		return fmt.Sprintf("Are you sure you want to close this %s?", lower)
	case "denied":
		return fmt.Sprintf("Are you sure you want to deny this %s?", lower)
	}
	return fmt.Sprintf("Are you sure you want to change this %s to %s?", lower, generateEnumLabel(target))
}

// ── Relationship building ────────────────────────────────────────────────────

func buildRelationships(ent *entityInfo, relationships []relationshipInfo, services []serviceInfo) []UIRelationship {
	var rels []UIRelationship

	for _, r := range relationships {
		if r.from != ent.name {
			continue
		}

		rel := UIRelationship{
			Name:            r.name,
			TargetEntity:    toSnake(r.to),
			Cardinality:     r.edgeType,
			DisplayInDetail: true,
		}

		switch r.edgeType {
		case "o2m":
			rel.DisplayMode = "table"
		case "m2m":
			rel.DisplayMode = "list"
		default:
			rel.DisplayMode = "list"
		}

		rels = append(rels, rel)
	}
	return rels
}

// ── Validation building ──────────────────────────────────────────────────────

func buildValidation(ent *entityInfo, fields []UIFieldDef) UIValidation {
	v := UIValidation{}

	// Field-level rules
	for _, f := range fields {
		if f.Required && f.ShowInCreate {
			v.FieldRules = append(v.FieldRules, UIFieldRule{
				Field: f.Name, Rule: "required",
			})
		}
		if f.Min != nil {
			v.FieldRules = append(v.FieldRules, UIFieldRule{
				Field: f.Name, Rule: "min", Value: f.Min,
			})
		}
		if f.Max != nil {
			v.FieldRules = append(v.FieldRules, UIFieldRule{
				Field: f.Name, Rule: "max", Value: f.Max,
			})
		}
		if f.Pattern != "" {
			v.FieldRules = append(v.FieldRules, UIFieldRule{
				Field: f.Name, Rule: "pattern", Value: f.Pattern,
			})
		}
		if f.MinItems != nil {
			v.FieldRules = append(v.FieldRules, UIFieldRule{
				Field: f.Name, Rule: "min_length", Value: *f.MinItems,
			})
		}
		if f.Type == "money" && (f.MoneyVariant == "non_negative" || f.MoneyVariant == "positive") {
			min := 0
			if f.MoneyVariant == "positive" {
				min = 1
			}
			v.FieldRules = append(v.FieldRules, UIFieldRule{
				Field: f.Name + ".amount_cents", Rule: "min", Value: min,
			})
		}
	}

	// Cross-field rules from constraints
	constraints := getEntityConstraints(ent.name)
	for i, c := range constraints {
		for _, req := range c.requires {
			rule := UICrossFieldRule{
				ID: fmt.Sprintf("%s_%s_%d", toSnake(ent.name), toSnake(req), i),
				Condition: &VisibilityRule{
					Field:    c.field,
					Operator: c.operator,
				},
				Then:    UIFieldRule{Field: req, Rule: "required"},
				Message: fmt.Sprintf("%s is required when %s is %s", fieldLabel(req, false), fieldLabel(c.field, false), formatConditionValue(c)),
			}

			if c.value != nil {
				rule.Condition.Value = c.value
			}
			if len(c.values) > 0 {
				rule.Condition.Values = c.values
			}

			rule.Description = rule.Message
			v.CrossFieldRules = append(v.CrossFieldRules, rule)
		}
	}

	return v
}

func formatConditionValue(c constraintDef) string {
	if c.value != nil {
		return fmt.Sprintf("%v", c.value)
	}
	if len(c.values) > 0 {
		return strings.Join(c.values, ", ")
	}
	return ""
}

// ── API schema building ──────────────────────────────────────────────────────

func buildAPISchema(ent *entityInfo, services []serviceInfo) UIAPI {
	api := UIAPI{
		Operations:  make(map[string]UIAPIEndpoint),
		Transitions: make(map[string]UIAPIEndpoint),
		Related:     make(map[string]UIAPIEndpoint),
	}

	for _, svc := range services {
		for _, op := range svc.operations {
			if op.entity != ent.name {
				continue
			}
			if op.custom {
				continue
			}

			path := fmt.Sprintf("%s/%s", svc.basePath, op.entityPath)
			api.BasePath = path

			switch op.opType {
			case "create":
				api.Operations["create"] = UIAPIEndpoint{Method: "POST", Path: path}
			case "get":
				api.Operations["get"] = UIAPIEndpoint{Method: "GET", Path: path + "/{id}"}
			case "list":
				api.Operations["list"] = UIAPIEndpoint{Method: "GET", Path: path}
			case "update":
				api.Operations["update"] = UIAPIEndpoint{Method: "PATCH", Path: path + "/{id}"}
			case "delete":
				api.Operations["delete"] = UIAPIEndpoint{Method: "DELETE", Path: path + "/{id}"}
			case "transition":
				if op.action != "" {
					api.Transitions[op.action] = UIAPIEndpoint{
						Method: "POST",
						Path:   path + "/{id}/" + op.action,
					}
				}
			}
		}
	}

	return api
}

// ── Embedded type parsing ────────────────────────────────────────────────────

type embeddedTypeDef struct {
	name   string
	fields []fieldInfo
}

func parseEmbeddedTypes(val cue.Value) map[string]*embeddedTypeDef {
	types := make(map[string]*embeddedTypeDef)

	iter, _ := val.Fields(cue.Definitions(true))
	for iter.Next() {
		label := iter.Selector().String()
		defVal := iter.Value()

		name := strings.TrimPrefix(label, "#")

		// Skip entities (they have id + audit)
		idField := defVal.LookupPath(cue.ParsePath("id"))
		auditField := defVal.LookupPath(cue.ParsePath("audit"))
		if idField.Err() == nil && auditField.Err() == nil {
			continue
		}

		// Only include known value types that are structs
		if _, ok := knownValueTypes[label]; !ok {
			continue
		}

		// Skip non-struct types
		kind := defVal.IncompleteKind()
		if kind != cue.StructKind {
			continue
		}

		td := &embeddedTypeDef{name: name}
		fIter, _ := defVal.Fields(cue.Optional(true))
		for fIter.Next() {
			fname := strings.TrimSuffix(fIter.Selector().String(), "?")
			if strings.HasPrefix(fname, "_") {
				continue
			}

			fi := classifyUIField(fname, fIter.Value(), fIter.IsOptional())
			if fi != nil {
				td.fields = append(td.fields, *fi)
			}
		}

		types[name] = td
	}

	return types
}

// ── Utility functions ────────────────────────────────────────────────────────

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func dedup(s []string) []string {
	if len(s) == 0 {
		return s
	}
	seen := make(map[string]bool)
	var result []string
	for _, v := range s {
		if !seen[v] {
			seen[v] = true
			result = append(result, v)
		}
	}
	return result
}

func writeJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling JSON: %w", err)
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0644)
}

// ── Main ─────────────────────────────────────────────────────────────────────

func main() {
	log.SetFlags(0)
	log.SetPrefix("uigen: ")

	ctx := cuecontext.New()
	projectRoot := findProjectRoot()

	// Load ontology CUE package
	ontInsts := load.Instances([]string{"./ontology"}, &load.Config{Dir: projectRoot})
	if len(ontInsts) == 0 {
		log.Fatal("no CUE instances found in ./ontology")
	}
	if ontInsts[0].Err != nil {
		log.Fatalf("loading ontology CUE: %v", ontInsts[0].Err)
	}
	ontVal := ctx.BuildInstance(ontInsts[0])
	if ontVal.Err() != nil {
		log.Fatalf("building ontology CUE value: %v", ontVal.Err())
	}

	// Load codegen CUE package
	cgInsts := load.Instances([]string{"./codegen"}, &load.Config{Dir: projectRoot})
	if len(cgInsts) == 0 {
		log.Fatal("no CUE instances found in ./codegen")
	}
	if cgInsts[0].Err != nil {
		log.Fatalf("loading codegen CUE: %v", cgInsts[0].Err)
	}
	cgVal := ctx.BuildInstance(cgInsts[0])
	if cgVal.Err() != nil {
		log.Fatalf("building codegen CUE value: %v", cgVal.Err())
	}

	// Parse all data sources
	entities := parseEntities(ontVal)
	relationships := parseRelationships(ontVal)
	parseStateMachines(ontVal, entities)
	services := parseOperations(cgVal)
	overrides := parseUIOverrides(cgVal)
	enumGroupings := parseEnumGroupings(cgVal)
	_ = parseEmbeddedTypes(ontVal) // Collected but schemas capture them via field types

	// Populate knownEntityNames for field classifier to validate _id references
	for name := range entities {
		knownEntityNames[toSnake(name)] = true
	}

	// Populate edgeToEntity from relationships for _id fields that use edge names
	// (e.g., owner_id on Portfolio → Organization via "owner" edge)
	for _, rel := range relationships {
		edgeToEntity[rel.name] = toSnake(rel.to)
	}

	// Populate entityDisplayField — uses the first @display()-annotated field per entity.
	// Falls back to "name" or "id" if no field is marked with @display().
	for eName, ent := range entities {
		snake := toSnake(eName)
		best := "id"
		for _, f := range ent.fields {
			if f.isDisplayName {
				best = f.name
				break
			}
		}
		if best == "id" {
			// Fallback for entities without @display()
			for _, f := range ent.fields {
				if f.name == "name" {
					best = "name"
					break
				}
			}
		}
		entityDisplayField[snake] = best
	}

	// Reclassify _id/_ids fields now that knownEntityNames and edgeToEntity are populated.
	// parseEntities runs classifyUIField before these maps exist, so entity_ref detection
	// for edge-named fields (e.g., owner_id → organization) needs a second pass.
	for _, ent := range entities {
		for i, f := range ent.fields {
			if f.uiType != "string" {
				continue
			}
			if strings.HasSuffix(f.name, "_ids") {
				ref := strings.TrimSuffix(f.name, "_ids")
				if knownEntityNames[ref] {
					ent.fields[i].uiType = "entity_ref_list"
					ent.fields[i].refEntity = ref
				} else if target, ok := edgeToEntity[ref]; ok {
					ent.fields[i].uiType = "entity_ref_list"
					ent.fields[i].refEntity = target
				}
			} else if strings.HasSuffix(f.name, "_id") {
				ref := strings.TrimSuffix(f.name, "_id")
				if knownEntityNames[ref] {
					ent.fields[i].uiType = "entity_ref"
					ent.fields[i].refEntity = ref
				} else if target, ok := edgeToEntity[ref]; ok {
					ent.fields[i].uiType = "entity_ref"
					ent.fields[i].refEntity = target
				}
			}
		}
	}

	// Ensure output directory exists
	outDir := filepath.Join(projectRoot, "gen", "ui", "schema")
	if err := os.MkdirAll(outDir, 0755); err != nil {
		log.Fatalf("creating output directory: %v", err)
	}

	// Collect all enums across entities
	allEnums := make(map[string]UIEnum)

	// Generate schema for each entity
	entityNames := sortedKeys(entities)
	for _, name := range entityNames {
		ent := entities[name]
		schema := buildUISchema(ent, relationships, services, overrides, enumGroupings, allEnums)

		outPath := filepath.Join(outDir, toSnake(name)+".schema.json")
		if err := writeJSON(outPath, schema); err != nil {
			log.Fatalf("writing schema for %s: %v", name, err)
		}
		fmt.Printf("Generated gen/ui/schema/%s.schema.json\n", toSnake(name))
	}

	// Write combined enums file
	enumPath := filepath.Join(outDir, "_enums.schema.json")
	if err := writeJSON(enumPath, allEnums); err != nil {
		log.Fatalf("writing enums schema: %v", err)
	}
	fmt.Printf("Generated gen/ui/schema/_enums.schema.json\n")

	fmt.Printf("uigen: generated %d entity schemas + 1 enums schema\n", len(entities))
}
