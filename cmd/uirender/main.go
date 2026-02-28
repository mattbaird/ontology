// cmd/uirender generates Svelte + Skeleton UI + Tailwind components from UI schemas.
//
// This is Layer 2 of the UI generation system. It reads JSON schemas from
// gen/ui/schema/ (produced by cmd/uigen) and generates Svelte components,
// TypeScript types, API clients, validation, stores, and shared components.
package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
)

//go:embed templates/*
var templateFS embed.FS

// entityBasePaths maps entity names to their API base paths, populated during schema loading.
var entityBasePaths = map[string]string{}

// ── Schema types (mirrors uigen output) ──────────────────────────────────────

type UISchema struct {
	Entity            string                      `json:"entity"`
	DisplayName       string                      `json:"display_name"`
	DisplayNamePlural string                      `json:"display_name_plural"`
	PrimaryDisplay    string                      `json:"primary_display_template"`
	Fields            []UIFieldDef                `json:"fields"`
	Enums             map[string]UIEnum           `json:"enums"`
	Form              UIForm                      `json:"form"`
	Detail            UIDetail                    `json:"detail"`
	List              UIList                      `json:"list"`
	Status            *UIStatus                   `json:"status"`
	StateMachine      *UIStateMachine             `json:"state_machine"`
	Relationships     []UIRelationship            `json:"relationships"`
	Validation        UIValidation                `json:"validation"`
	API               UIAPI                       `json:"api"`
}

type UIFieldDef struct {
	Name             string `json:"name"`
	Type             string `json:"type"`
	EnumRef          string `json:"enum_ref,omitempty"`
	ObjectRef        string `json:"object_ref,omitempty"`
	RefEntity        string `json:"ref_entity,omitempty"`
	RefDisplay       string `json:"ref_display,omitempty"`
	MoneyVariant     string `json:"money_variant,omitempty"`
	Required         bool   `json:"required"`
	Default          any    `json:"default"`
	Immutable        bool   `json:"immutable,omitempty"`
	Label            string `json:"label"`
	HelpText         string `json:"help_text,omitempty"`
	ShowInCreate     bool   `json:"show_in_create"`
	ShowInUpdate     bool   `json:"show_in_update"`
	ShowInList       bool   `json:"show_in_list"`
	ShowInDetail     bool   `json:"show_in_detail"`
	Sortable         bool   `json:"sortable"`
	Filterable       bool   `json:"filterable"`
	FilterType       string `json:"filter_type,omitempty"`
	Pattern          string `json:"pattern,omitempty"`
	Min              any    `json:"min,omitempty"`
	Max              any    `json:"max,omitempty"`
	MinItems         *int   `json:"min_items,omitempty"`
	IsSensitive      bool   `json:"is_sensitive,omitempty"`
	IsPII            bool   `json:"is_pii,omitempty"`
	IsComputed       bool   `json:"is_computed,omitempty"`
	IsDeprecated     bool   `json:"is_deprecated,omitempty"`
	DeprecatedReason string `json:"deprecated_reason,omitempty"`
	DeprecatedSince  string `json:"deprecated_since,omitempty"`
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
	ID                 string          `json:"id"`
	Title              string          `json:"title"`
	Collapsible        bool            `json:"collapsible"`
	InitiallyCollapsed *bool           `json:"initially_collapsed,omitempty"`
	Fields             []string        `json:"fields,omitempty"`
	EmbeddedObject     string          `json:"embedded_object,omitempty"`
	EmbeddedArray      string          `json:"embedded_array,omitempty"`
	VisibleWhen        *VisibilityRule `json:"visible_when,omitempty"`
	RequiredWhen       *VisibilityRule `json:"required_when,omitempty"`
}

type VisibilityRule struct {
	Field    string   `json:"field"`
	Operator string   `json:"operator"`
	Value    any      `json:"value,omitempty"`
	Values   []string `json:"values,omitempty"`
}

type UIDetail struct {
	Header          UIDetailHeader     `json:"header"`
	Sections        []UIDetailSection  `json:"sections"`
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
	Title        string `json:"title"`
	Relationship string `json:"relationship"`
	Entity       string `json:"entity"`
	Display      string `json:"display"`
}

type UIList struct {
	DefaultColumns []UIListColumn `json:"default_columns"`
	Filters        []UIListFilter `json:"filters"`
	DefaultSort    UISort         `json:"default_sort"`
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
	Field     string `json:"field"`
	Type      string `json:"type"`
	Label     string `json:"label,omitempty"`
	EnumRef   string `json:"enum_ref,omitempty"`
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
	Name         string `json:"name"`
	TargetEntity string `json:"target_entity"`
	Cardinality  string `json:"cardinality"`
	APIEndpoint  string `json:"api_endpoint,omitempty"`
}

type UIValidation struct {
	FieldRules      []UIFieldRule      `json:"field_rules"`
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
	BasePath    string                   `json:"base_path"`
	Operations  map[string]UIAPIEndpoint `json:"operations"`
	Transitions map[string]UIAPIEndpoint `json:"transitions,omitempty"`
	Related     map[string]UIAPIEndpoint `json:"related,omitempty"`
}

type UIAPIEndpoint struct {
	Method string `json:"method"`
	Path   string `json:"path"`
}

// ── Template context types ───────────────────────────────────────────────────

type templateData struct {
	UISchema
	PascalName      string
	CamelName       string
	HasStatus       bool
	HasStateMachine bool
	StatusType      string
	InitialStatus   string // First status enum value — used as default on create
	RoutePath       string // e.g., "/properties" — API.BasePath with /v1 prefix stripped
	Imports         []importDef
}

type importDef struct {
	Name string
	Path string
}

type enumsTemplateData struct {
	Enums map[string]UIEnum
}

type sectionTemplateData struct {
	TypeName    string
	FieldPrefix string
	Fields      []UIFieldDef
}

// ── Name utilities ───────────────────────────────────────────────────────────

func toPascal(s string) string {
	parts := strings.Split(s, "_")
	for i, p := range parts {
		if p == "" {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
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

func toCamelHyphen(s string) string {
	parts := strings.Split(s, "-")
	for i := 1; i < len(parts); i++ {
		if parts[i] != "" {
			parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}
	return strings.Join(parts, "")
}

func toScreamingSnake(s string) string {
	var result []rune
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result = append(result, '_')
		}
		result = append(result, r)
	}
	return strings.ToUpper(string(result))
}

func fieldLabel(s string) string {
	if s == "" {
		return ""
	}
	clean := strings.TrimSuffix(strings.TrimSuffix(s, "_ids"), "_id")
	clean = strings.ReplaceAll(clean, ".", " ")
	parts := strings.Split(clean, "_")
	for i, p := range parts {
		if p == "" {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	return strings.Join(parts, " ")
}

func skeletonVariant(color string) string {
	switch color {
	case "success":
		return "variant-soft-success"
	case "error":
		return "variant-soft-error"
	case "warning":
		return "variant-soft-warning"
	case "secondary":
		return "variant-soft-secondary"
	case "surface":
		return "variant-soft"
	default:
		return "variant-soft"
	}
}

// ── Template functions ───────────────────────────────────────────────────────

func tsType(f UIFieldDef) string {
	switch f.Type {
	case "string", "text", "date", "datetime":
		return "string"
	case "int", "float":
		return "number"
	case "bool":
		return "boolean"
	case "enum":
		if f.EnumRef != "" {
			return f.EnumRef
		}
		return "string"
	case "money":
		return "{ amount_cents: number; currency: string }"
	case "address":
		return "Address"
	case "date_range":
		return "{ start: string; end?: string }"
	case "contact_method":
		return "ContactMethod"
	case "entity_ref":
		return "string"
	case "entity_ref_list":
		return "string[]"
	case "embedded_object":
		return "Record<string, any>"
	case "embedded_array":
		return "Array<Record<string, any>>"
	case "string_list":
		return "string[]"
	default:
		return "any"
	}
}

func commonTypeImports(data templateData) string {
	needed := map[string]bool{}
	for _, f := range data.Fields {
		switch f.Type {
		case "address":
			needed["Address"] = true
		case "contact_method":
			needed["ContactMethod"] = true
		}
	}
	if len(needed) == 0 {
		return ""
	}
	names := make([]string, 0, len(needed))
	for n := range needed {
		names = append(names, n)
	}
	sort.Strings(names)
	return fmt.Sprintf("import type { %s } from './common.types';", strings.Join(names, ", "))
}

func replaceID(s string) string {
	return strings.ReplaceAll(s, "{id}", "${id}")
}

func replaceIDTemplate(s string) string {
	return strings.ReplaceAll(s, "{id}", "${entityId}")
}

func escapeJS(s string) string {
	s = strings.ReplaceAll(s, "'", "\\'")
	return s
}

func dotToOptional(s string) string {
	return strings.ReplaceAll(s, ".", "?.")
}

func derefBool(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}

func visibilityCheck(rule *VisibilityRule, varName string) string {
	if rule == nil {
		return "return true;"
	}
	switch rule.Operator {
	case "eq":
		switch v := rule.Value.(type) {
		case bool:
			return fmt.Sprintf("return %s.%s === %v;", varName, rule.Field, v)
		default:
			return fmt.Sprintf("return %s.%s === '%v';", varName, rule.Field, v)
		}
	case "in":
		vals := make([]string, len(rule.Values))
		for i, v := range rule.Values {
			vals[i] = fmt.Sprintf("'%s'", v)
		}
		return fmt.Sprintf("return [%s].includes(%s.%s ?? '');", strings.Join(vals, ", "), varName, rule.Field)
	case "truthy":
		return fmt.Sprintf("return !!%s.%s;", varName, rule.Field)
	default:
		return "return true;"
	}
}

func formFieldRender(data any, fieldName string) string {
	// Extract fields from either UISchema or templateData
	var fields []UIFieldDef
	switch d := data.(type) {
	case templateData:
		fields = d.Fields
	case UISchema:
		fields = d.Fields
	default:
		return fmt.Sprintf("    <!-- Unknown field: %s -->", fieldName)
	}

	// Find the field definition
	var fd *UIFieldDef
	for i := range fields {
		if fields[i].Name == fieldName {
			fd = &fields[i]
			break
		}
	}
	if fd == nil {
		return fmt.Sprintf("    <!-- Unknown field: %s -->", fieldName)
	}

	req := ""
	if fd.Required {
		req = " required"
	}

	switch fd.Type {
	case "string":
		return fmt.Sprintf(`    <FormField label="%s"%s error={errors['%s']}>
      <input type="text" class="input" value={values.%s ?? ''} on:input={(e) => handleChange('%s', inputValue(e))} />
    </FormField>`, fd.Label, req, fd.Name, fd.Name, fd.Name)
	case "text":
		return fmt.Sprintf(`    <FormField label="%s"%s error={errors['%s']}>
      <textarea class="textarea" value={values.%s ?? ''} on:input={(e) => handleChange('%s', textareaValue(e))} />
    </FormField>`, fd.Label, req, fd.Name, fd.Name, fd.Name)
	case "int":
		return fmt.Sprintf(`    <FormField label="%s"%s error={errors['%s']}>
      <input type="number" step="1" class="input" value={values.%s ?? ''} on:input={(e) => handleChange('%s', parseInt(inputValue(e)))} />
    </FormField>`, fd.Label, req, fd.Name, fd.Name, fd.Name)
	case "float":
		return fmt.Sprintf(`    <FormField label="%s"%s error={errors['%s']}>
      <input type="number" step="any" class="input" value={values.%s ?? ''} on:input={(e) => handleChange('%s', parseFloat(inputValue(e)))} />
    </FormField>`, fd.Label, req, fd.Name, fd.Name, fd.Name)
	case "bool":
		return fmt.Sprintf(`    <FormField label="%s" error={errors['%s']}>
      <input type="checkbox" class="checkbox" checked={values.%s ?? false} on:change={(e) => handleChange('%s', inputChecked(e))} />
    </FormField>`, fd.Label, fd.Name, fd.Name, fd.Name)
	case "date":
		return fmt.Sprintf(`    <FormField label="%s"%s error={errors['%s']}>
      <input type="date" class="input" value={values.%s ?? ''} on:change={(e) => handleChange('%s', inputValue(e))} />
    </FormField>`, fd.Label, req, fd.Name, fd.Name, fd.Name)
	case "datetime":
		return fmt.Sprintf(`    <FormField label="%s"%s error={errors['%s']}>
      <input type="datetime-local" class="input" value={values.%s ?? ''} on:change={(e) => handleChange('%s', inputValue(e))} />
    </FormField>`, fd.Label, req, fd.Name, fd.Name, fd.Name)
	case "enum":
		optConst := toScreamingSnake(fd.EnumRef) + "_OPTIONS"
		return fmt.Sprintf(`    <FormField label="%s"%s error={errors['%s']}>
      <EnumSelect options={%s} value={values.%s} on:change={(e) => handleChange('%s', e.detail)} />
    </FormField>`, fd.Label, req, fd.Name, optConst, fd.Name, fd.Name)
	case "money":
		return fmt.Sprintf(`    <FormField label="%s"%s error={errors['%s']}>
      <MoneyInput value={values.%s} on:change={(e) => handleChange('%s', e.detail)} />
    </FormField>`, fd.Label, req, fd.Name, fd.Name, fd.Name)
	case "entity_ref":
		bp := entityBasePaths[fd.RefEntity]
		df := fd.RefDisplay
		if df == "" {
			df = "name"
		}
		return fmt.Sprintf(`    <FormField label="%s"%s error={errors['%s']}>
      <EntityRefSelect entityType="%s" basePath="%s" displayField="%s" value={values.%s} on:change={(e) => handleChange('%s', e.detail)} />
    </FormField>`, fd.Label, req, fd.Name, fd.RefEntity, bp, df, fd.Name, fd.Name)
	case "entity_ref_list":
		bp := entityBasePaths[fd.RefEntity]
		df := fd.RefDisplay
		if df == "" {
			df = "name"
		}
		return fmt.Sprintf(`    <FormField label="%s"%s error={errors['%s']}>
      <EntityRefSelect entityType="%s" basePath="%s" displayField="%s" multiple value={values.%s} on:change={(e) => handleChange('%s', e.detail)} />
    </FormField>`, fd.Label, req, fd.Name, fd.RefEntity, bp, df, fd.Name, fd.Name)
	case "date_range":
		return fmt.Sprintf(`    <FormField label="%s"%s error={errors['%s']}>
      <DateRangeInput value={values.%s} on:change={(e) => handleChange('%s', e.detail)} />
    </FormField>`, fd.Label, req, fd.Name, fd.Name, fd.Name)
	case "address":
		return fmt.Sprintf(`    <FormField label="%s"%s error={errors['%s']}>
      <AddressForm value={values.%s} on:change={(e) => handleChange('%s', e.detail)} />
    </FormField>`, fd.Label, req, fd.Name, fd.Name, fd.Name)
	case "string_list":
		return fmt.Sprintf(`    <FormField label="%s" error={errors['%s']}>
      <ArrayEditor value={values.%s ?? []} on:change={(e) => handleChange('%s', e.detail)} />
    </FormField>`, fd.Label, fd.Name, fd.Name, fd.Name)
	default:
		return fmt.Sprintf("    <!-- %s: %s (%s) -->", fd.Name, fd.Label, fd.Type)
	}
}

// computeFormImports scans form sections to determine which shared components
// and enum option constants the form template needs to import.
func computeFormImports(schema UISchema) []importDef {
	neededComponents := map[string]bool{}
	neededEnumConsts := []string{}

	// Collect all field names referenced in form sections
	formFieldNames := map[string]bool{}
	for _, sec := range schema.Form.Sections {
		for _, fn := range sec.Fields {
			formFieldNames[fn] = true
		}
	}

	// Map field types to component imports
	for _, fd := range schema.Fields {
		if !formFieldNames[fd.Name] {
			continue
		}
		switch fd.Type {
		case "enum":
			neededComponents["EnumSelect"] = true
			if fd.EnumRef != "" {
				neededEnumConsts = append(neededEnumConsts, toScreamingSnake(fd.EnumRef)+"_OPTIONS")
			}
		case "money":
			neededComponents["MoneyInput"] = true
		case "entity_ref", "entity_ref_list":
			neededComponents["EntityRefSelect"] = true
		case "date_range":
			neededComponents["DateRangeInput"] = true
		case "address":
			neededComponents["AddressForm"] = true
		case "contact_method":
			neededComponents["ContactMethodInput"] = true
		case "string_list":
			neededComponents["ArrayEditor"] = true
		}
	}

	var imports []importDef

	// Enum option constants (single import from enums.ts)
	if len(neededEnumConsts) > 0 {
		sort.Strings(neededEnumConsts)
		imports = append(imports, importDef{
			Name: "{ " + strings.Join(neededEnumConsts, ", ") + " }",
			Path: "../../../types/enums",
		})
	}

	// Sorted component imports for deterministic output
	componentPaths := map[string]string{
		"AddressForm":        "../../shared/AddressForm.svelte",
		"ArrayEditor":        "../../shared/ArrayEditor.svelte",
		"ContactMethodInput": "../../shared/ContactMethodInput.svelte",
		"DateRangeInput":     "../../shared/DateRangeInput.svelte",
		"EntityRefSelect":    "../../shared/EntityRefSelect.svelte",
		"EnumSelect":         "../../shared/EnumSelect.svelte",
		"MoneyInput":         "../../shared/MoneyInput.svelte",
	}
	names := make([]string, 0, len(neededComponents))
	for n := range neededComponents {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, name := range names {
		imports = append(imports, importDef{Name: name, Path: componentPaths[name]})
	}
	return imports
}

// requiredCheck generates a type-appropriate required-field check for validation templates.
// It looks up the field type from the schema to decide whether to use === '' (string-like) or just == null.
func requiredCheck(rule UIFieldRule, data templateData) string {
	accessor := dotToOptional(rule.Field)
	label := fieldLabel(rule.Field)
	// Find the field type
	fieldType := ""
	for _, f := range data.Fields {
		if f.Name == rule.Field {
			fieldType = f.Type
			break
		}
	}
	switch fieldType {
	case "string", "text", "date", "datetime":
		return fmt.Sprintf(`  if (input.%s == null || input.%s === '') {
    errors['%s'] = '%s is required';
  }`, accessor, accessor, rule.Field, escapeJS(label))
	default:
		return fmt.Sprintf(`  if (input.%s == null) {
    errors['%s'] = '%s is required';
  }`, accessor, rule.Field, escapeJS(label))
	}
}

func crossFieldCheck(rule UICrossFieldRule) string {
	if rule.Condition == nil {
		return ""
	}

	// Use bracket notation for condition field to avoid TS errors when
	// the field (e.g. status) is not in CreateInput
	condField := fmt.Sprintf("(input as any)['%s']", rule.Condition.Field)
	var condCheck string
	switch rule.Condition.Operator {
	case "eq":
		switch v := rule.Condition.Value.(type) {
		case bool:
			condCheck = fmt.Sprintf("%s === %v", condField, v)
		default:
			condCheck = fmt.Sprintf("%s === '%v'", condField, v)
		}
	case "in":
		vals := make([]string, len(rule.Condition.Values))
		for i, v := range rule.Condition.Values {
			vals[i] = fmt.Sprintf("'%s'", v)
		}
		condCheck = fmt.Sprintf("[%s].includes(%s ?? '')", strings.Join(vals, ", "), condField)
	case "truthy":
		condCheck = fmt.Sprintf("!!%s", condField)
	default:
		return ""
	}

	return fmt.Sprintf(`  if (%s && (input as any)['%s'] == null) {
    errors['%s'] = '%s';
  }`, condCheck, rule.Then.Field, rule.Then.Field, escapeJS(rule.Message))
}

// ── Shared component content ─────────────────────────────────────────────────

var sharedComponents = map[string]string{
	"MoneyInput.svelte": `<!-- GENERATED FROM PROPELLER ONTOLOGY. DO NOT HAND-EDIT. -->
<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  export let value: { amount_cents: number; currency: string } | null = null;
  export let min: number | null = null;
  export let currencies: string[] = ['USD'];
  export let disabled = false;
  export let readonly = false;
  const dispatch = createEventDispatcher();
  let displayValue = value ? (value.amount_cents / 100).toFixed(2) : '';
  let currency = value?.currency ?? 'USD';
  function handleBlur() {
    const parsed = parseFloat(displayValue);
    if (isNaN(parsed)) return;
    if (min !== null && parsed * 100 < min) return;
    dispatch('change', { amount_cents: Math.round(parsed * 100), currency });
  }
</script>
<div class="input-group input-group-divider grid-cols-[auto_1fr_auto]">
  <div class="input-group-shim">$</div>
  <input type="text" inputmode="decimal" bind:value={displayValue} on:blur={handleBlur} disabled={disabled || readonly} class="input" placeholder="0.00" />
  <div class="input-group-shim">{currency}</div>
</div>`,

	"MoneyDisplay.svelte": `<!-- GENERATED FROM PROPELLER ONTOLOGY. DO NOT HAND-EDIT. -->
<script lang="ts">
  export let value: { amount_cents: number; currency: string } | null = null;
</script>
{#if value}
  <span>{(value.amount_cents / 100).toLocaleString('en-US', { style: 'currency', currency: value.currency ?? 'USD' })}</span>
{:else}
  <span class="text-surface-400">—</span>
{/if}`,

	"AddressForm.svelte": `<!-- GENERATED FROM PROPELLER ONTOLOGY. DO NOT HAND-EDIT. -->
<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  export let value: Record<string, any> | null = null;
  export let readonly = false;
  const dispatch = createEventDispatcher();
  $: addr = value ?? {};
  function iv(e: Event): string { return (e.target as HTMLInputElement).value; }
  function update(field: string, val: string) {
    const updated = { ...addr, [field]: val };
    dispatch('change', updated);
  }
</script>
<div class="space-y-2">
  <input type="text" class="input" placeholder="Address Line 1" value={addr.line1 ?? ''} on:input={(e) => update('line1', iv(e))} disabled={readonly} />
  <input type="text" class="input" placeholder="Address Line 2" value={addr.line2 ?? ''} on:input={(e) => update('line2', iv(e))} disabled={readonly} />
  <div class="grid grid-cols-3 gap-2">
    <input type="text" class="input" placeholder="City" value={addr.city ?? ''} on:input={(e) => update('city', iv(e))} disabled={readonly} />
    <input type="text" class="input" placeholder="State" maxlength="2" value={addr.state ?? ''} on:input={(e) => update('state', iv(e))} disabled={readonly} />
    <input type="text" class="input" placeholder="ZIP" value={addr.postal_code ?? ''} on:input={(e) => update('postal_code', iv(e))} disabled={readonly} />
  </div>
</div>`,

	"AddressDisplay.svelte": `<!-- GENERATED FROM PROPELLER ONTOLOGY. DO NOT HAND-EDIT. -->
<script lang="ts">
  export let value: Record<string, any> | null = null;
</script>
{#if value}
  <address class="not-italic">
    {value.line1}{#if value.line2}<br />{value.line2}{/if}<br />
    {value.city}, {value.state} {value.postal_code}
  </address>
{:else}
  <span class="text-surface-400">—</span>
{/if}`,

	"DateRangeInput.svelte": `<!-- GENERATED FROM PROPELLER ONTOLOGY. DO NOT HAND-EDIT. -->
<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  export let value: { start?: string; end?: string } | null = null;
  export let requireEnd = false;
  const dispatch = createEventDispatcher();
  $: range = value ?? {};
  function iv(e: Event): string { return (e.target as HTMLInputElement).value; }
  function update(field: string, val: string) {
    const updated = { ...range, [field]: val };
    dispatch('change', updated);
  }
</script>
<div class="grid grid-cols-2 gap-2">
  <div>
    <label class="text-sm">Start</label>
    <input type="date" class="input" value={range.start ?? ''} on:change={(e) => update('start', iv(e))} />
  </div>
  <div>
    <label class="text-sm">End{#if requireEnd} <span class="text-error-500">*</span>{/if}</label>
    <input type="date" class="input" value={range.end ?? ''} on:change={(e) => update('end', iv(e))} />
  </div>
</div>`,

	"DateRangeDisplay.svelte": `<!-- GENERATED FROM PROPELLER ONTOLOGY. DO NOT HAND-EDIT. -->
<script lang="ts">
  export let value: { start?: string; end?: string } | null = null;
</script>
{#if value}
  <span>{value.start ? new Date(value.start).toLocaleDateString() : '?'} — {value.end ? new Date(value.end).toLocaleDateString() : 'Ongoing'}</span>
{:else}
  <span class="text-surface-400">—</span>
{/if}`,

	"ContactMethodInput.svelte": `<!-- GENERATED FROM PROPELLER ONTOLOGY. DO NOT HAND-EDIT. -->
<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  export let value: Record<string, any> | null = null;
  const dispatch = createEventDispatcher();
  $: cm = value ?? {};
  function iv(e: Event): string { return (e.target as HTMLInputElement).value; }
  function sv(e: Event): string { return (e.target as HTMLSelectElement).value; }
  function update(field: string, val: any) {
    const updated = { ...cm, [field]: val };
    dispatch('change', updated);
  }
</script>
<div class="grid grid-cols-[auto_1fr] gap-2">
  <select class="select" value={cm.type ?? 'email'} on:change={(e) => update('type', sv(e))}>
    <option value="email">Email</option>
    <option value="phone">Phone</option>
    <option value="sms">SMS</option>
  </select>
  <input type="text" class="input" value={cm.value ?? ''} on:input={(e) => update('value', iv(e))} />
</div>`,

	"ContactMethodDisplay.svelte": `<!-- GENERATED FROM PROPELLER ONTOLOGY. DO NOT HAND-EDIT. -->
<script lang="ts">
  export let value: Record<string, any> | null = null;
</script>
{#if value}
  <span class="badge variant-soft">{value.type}</span> {value.value}
{:else}
  <span class="text-surface-400">—</span>
{/if}`,

	"EntityRefSelect.svelte": `<!-- GENERATED FROM PROPELLER ONTOLOGY. DO NOT HAND-EDIT. -->
<script lang="ts">
  import { createEventDispatcher, onMount } from 'svelte';
  import { apiClient } from '../../api/client';
  export let entityType: string;
  export let basePath: string = '';
  export let displayField: string = 'name';
  export let filter: Record<string, any> = {};
  export let multiple = false;
  export let value: string | string[] | null = null;
  const dispatch = createEventDispatcher();
  let allItems: Array<{ label: string; value: string }> = [];
  let options: Array<{ label: string; value: string }> = [];
  let showDropdown = false;
  let inputValue = '';
  onMount(async () => {
    if (!basePath) return;
    try {
      const response = await apiClient.get<any>(basePath, { ...filter, limit: 100 });
      const items = Array.isArray(response) ? response : (response.data ?? []);
      allItems = items.map((item: any) => ({ label: item[displayField] ?? item.id, value: item.id }));
      // If we have a current value, show its label
      if (value) {
        const match = allItems.find(i => i.value === value);
        if (match) inputValue = match.label;
      }
    } catch {
      allItems = [];
    }
  });
  function handleInput(event: Event) {
    const query = (event.target as HTMLInputElement).value;
    inputValue = query;
    showDropdown = true;
    if (basePath) {
      const lower = query.toLowerCase();
      options = lower ? allItems.filter(i => i.label.toLowerCase().includes(lower)) : allItems;
    } else {
      dispatch('change', query);
    }
  }
  function handleFocus() {
    showDropdown = true;
    options = allItems;
  }
  function select(item: { label: string; value: string }) {
    inputValue = item.label;
    showDropdown = false;
    options = [];
    dispatch('change', item.value);
  }
</script>
<div class="relative">
  <input type="text" class="input" placeholder="Search {entityType}..." value={inputValue} on:input={handleInput} on:focus={handleFocus} on:blur={() => setTimeout(() => showDropdown = false, 200)} />
  {#if showDropdown && options.length > 0}
    <ul class="card list p-1 mt-1 max-h-40 overflow-y-auto absolute z-10 w-full shadow-lg">
      {#each options as opt}
        <li>
          <button type="button" class="btn btn-sm w-full text-left hover:variant-soft" on:click={() => select(opt)}>
            {opt.label}
          </button>
        </li>
      {/each}
    </ul>
  {/if}
  {#if showDropdown && options.length === 0 && inputValue}
    <div class="card p-2 mt-1 text-sm text-surface-500 absolute z-10 w-full">No {entityType} found</div>
  {/if}
</div>`,

	"EnumSelect.svelte": `<!-- GENERATED FROM PROPELLER ONTOLOGY. DO NOT HAND-EDIT. -->
<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  export let options: Array<{ value: string; label: string }> = [];
  export let value: string | null = null;
  export let disabled = false;
  const dispatch = createEventDispatcher();
  function sv(e: Event): string { return (e.target as HTMLSelectElement).value; }
</script>
<select class="select" {disabled} value={value ?? ''} on:change={(e) => dispatch('change', sv(e))}>
  <option value="">Select...</option>
  {#each options as opt}
    <option value={opt.value}>{opt.label}</option>
  {/each}
</select>`,

	"EnumBadge.svelte": `<!-- GENERATED FROM PROPELLER ONTOLOGY. DO NOT HAND-EDIT. -->
<script lang="ts">
  export let value: string = '';
  export let labels: Record<string, string> = {};
</script>
<span class="badge variant-soft">{labels[value] ?? value.replace(/_/g, ' ')}</span>`,

	"FormField.svelte": `<!-- GENERATED FROM PROPELLER ONTOLOGY. DO NOT HAND-EDIT. -->
<script lang="ts">
  export let label: string;
  export let required = false;
  export let error: string | undefined = undefined;
  export let helpText: string | undefined = undefined;
</script>
<label class="label">
  <span class="text-sm font-medium">
    {label}{#if required}<span class="text-error-500 ml-0.5">*</span>{/if}
  </span>
  <slot />
  {#if error}
    <p class="text-sm text-error-500 mt-1">{error}</p>
  {/if}
  {#if helpText}
    <p class="text-sm text-surface-400 mt-1">{helpText}</p>
  {/if}
</label>`,

	"FormSection.svelte": `<!-- GENERATED FROM PROPELLER ONTOLOGY. DO NOT HAND-EDIT. -->
<script lang="ts">
  export let title: string;
  export let collapsible = false;
  export let initiallyCollapsed = false;
  export let required = false;
  let collapsed = initiallyCollapsed;
</script>
<div class="card p-4 space-y-4">
  {#if collapsible}
    <button type="button" class="flex items-center justify-between w-full" on:click={() => collapsed = !collapsed}>
      <h3 class="h4">{title}{#if required}<span class="text-error-500 ml-1">*</span>{/if}</h3>
      <span class="text-sm">{collapsed ? '+' : '−'}</span>
    </button>
  {:else}
    <h3 class="h4">{title}{#if required}<span class="text-error-500 ml-1">*</span>{/if}</h3>
  {/if}
  {#if !collapsed || !collapsible}
    <div class="space-y-4">
      <slot />
    </div>
  {/if}
</div>`,

	"ArrayEditor.svelte": `<!-- GENERATED FROM PROPELLER ONTOLOGY. DO NOT HAND-EDIT. -->
<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  export let value: any[] = [];
  export let itemLabel = 'Item';
  const dispatch = createEventDispatcher();
  function iv(e: Event): string { return (e.target as HTMLInputElement).value; }
  function addItem() {
    value = [...value, ''];
    dispatch('change', value);
  }
  function removeItem(index: number) {
    value = value.filter((_, i) => i !== index);
    dispatch('change', value);
  }
  function updateItem(index: number, val: any) {
    value = value.map((v, i) => i === index ? val : v);
    dispatch('change', value);
  }
</script>
<div class="space-y-2">
  {#each value as item, i}
    <div class="flex gap-2 items-center">
      <input type="text" class="input flex-1" value={item} on:input={(e) => updateItem(i, iv(e))} />
      <button type="button" class="btn-icon btn-icon-sm variant-soft-error" on:click={() => removeItem(i)}>×</button>
    </div>
  {/each}
  <button type="button" class="btn btn-sm variant-soft" on:click={addItem}>+ Add {itemLabel}</button>
</div>`,

	"StatusBadge.svelte": `<!-- GENERATED FROM PROPELLER ONTOLOGY. DO NOT HAND-EDIT. -->
<script lang="ts">
  export let status: string;
  export let colorMap: Record<string, string> = {};
</script>
<span class="badge {colorMap[status] ?? 'variant-soft'}">
  {status.replace(/_/g, ' ').replace(/\b\w/g, (c) => c.toUpperCase())}
</span>`,

	"TransitionButton.svelte": `<!-- GENERATED FROM PROPELLER ONTOLOGY. DO NOT HAND-EDIT. -->
<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  export let label: string;
  export let variant: 'primary' | 'secondary' | 'danger' = 'primary';
  const dispatch = createEventDispatcher();
  const variantClass = variant === 'danger' ? 'variant-filled-error' : variant === 'primary' ? 'variant-filled-primary' : 'variant-soft';
</script>
<button type="button" class="btn {variantClass}" on:click={() => dispatch('click')}>{label}</button>`,

	"ConfirmDialog.svelte": `<!-- GENERATED FROM PROPELLER ONTOLOGY. DO NOT HAND-EDIT. -->
<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  export let open = false;
  export let message = 'Are you sure?';
  export let confirmLabel = 'Confirm';
  export let cancelLabel = 'Cancel';
  const dispatch = createEventDispatcher();
</script>
{#if open}
  <div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" role="presentation" on:click|self={() => dispatch('cancel')}>
    <div class="card p-6 max-w-md space-y-4">
      <p>{message}</p>
      <div class="flex justify-end gap-2">
        <button class="btn variant-soft" on:click={() => dispatch('cancel')}>{cancelLabel}</button>
        <button class="btn variant-filled-error" on:click={() => dispatch('confirm')}>{confirmLabel}</button>
      </div>
    </div>
  </div>
{/if}`,
}

// ── Store content ────────────────────────────────────────────────────────────

var storeFiles = map[string]string{
	"entity.ts": `// GENERATED FROM PROPELLER ONTOLOGY. DO NOT HAND-EDIT.
import { writable } from 'svelte/store';
import { apiClient } from '../api/client';

export function entityStore<T>(basePath: string, id: string, include?: string[]) {
  const { subscribe, set, update } = writable<{
    data: T | null;
    loading: boolean;
    error: Error | null;
  }>({ data: null, loading: true, error: null });

  async function fetch() {
    update(s => ({ ...s, loading: true, error: null }));
    try {
      const params = include ? { include: include.join(',') } : {};
      const data = await apiClient.get<T>(` + "`" + `${basePath}/${id}` + "`" + `, params);
      set({ data, loading: false, error: null });
    } catch (error) {
      set({ data: null, loading: false, error: error as Error });
    }
  }

  fetch();
  return { subscribe, refetch: fetch };
}`,

	"entityList.ts": `// GENERATED FROM PROPELLER ONTOLOGY. DO NOT HAND-EDIT.
import { writable, derived } from 'svelte/store';
import { apiClient } from '../api/client';

interface ListConfig {
  basePath: string;
  defaultSort: { field: string; direction: string };
  pageSize?: number;
}

export function entityListStore<T>(config: ListConfig) {
  const pageSize = config.pageSize ?? 25;
  const state = writable<{
    data: T[];
    total: number;
    loading: boolean;
    error: Error | null;
    page: number;
    sort: { field: string; direction: string };
    filters: Record<string, any>;
    pagination: { page: number; limit: number; size: number; amounts: number[] };
  }>({
    data: [],
    total: 0,
    loading: true,
    error: null,
    page: 0,
    sort: config.defaultSort,
    filters: {},
    pagination: { page: 0, limit: pageSize, size: 0, amounts: [10, 25, 50, 100] },
  });

  async function fetch() {
    state.update(s => ({ ...s, loading: true }));
    try {
      let s: any;
      state.subscribe(v => s = v)();
      const params: Record<string, any> = {
        offset: s.page * pageSize,
        limit: pageSize,
        sort: s.sort.field,
        order: s.sort.direction,
        ...s.filters,
      };
      const result = await apiClient.get<T[] | { data: T[]; total: number }>(config.basePath, params);
      // Handle both plain array and { data, total } response formats
      const items = Array.isArray(result) ? result : (result.data ?? []);
      const total = Array.isArray(result) ? items.length : (result.total ?? items.length);
      state.update(st => ({
        ...st,
        data: items,
        total,
        loading: false,
        error: null,
        pagination: { ...st.pagination, size: total },
      }));
    } catch (error) {
      state.update(s => ({ ...s, loading: false, error: error as Error }));
    }
  }

  function setPage(page: number) {
    state.update(s => ({ ...s, page }));
    fetch();
  }

  function toggleSort(field: string) {
    state.update(s => ({
      ...s,
      sort: s.sort.field === field
        ? { field, direction: s.sort.direction === 'asc' ? 'desc' : 'asc' }
        : { field, direction: 'asc' },
    }));
    fetch();
  }

  function setFilters(filters: Record<string, any>) {
    state.update(s => ({ ...s, filters, page: 0 }));
    fetch();
  }

  fetch();
  return { subscribe: state.subscribe, set: state.set, update: state.update, setPage, toggleSort, setFilters, refetch: fetch };
}`,

	"entityMutation.ts": `// GENERATED FROM PROPELLER ONTOLOGY. DO NOT HAND-EDIT.
import { writable } from 'svelte/store';
import { apiClient } from '../api/client';

export function entityMutationStore<TCreate, TUpdate, TResult>(
  entityType: string,
  basePath: string,
) {
  const state = writable<{
    loading: boolean;
    error: Error | null;
  }>({ loading: false, error: null });

  async function create(input: TCreate): Promise<TResult> {
    state.set({ loading: true, error: null });
    try {
      const result = await apiClient.post<TResult>(basePath, input);
      state.set({ loading: false, error: null });
      return result;
    } catch (error) {
      state.set({ loading: false, error: error as Error });
      throw error;
    }
  }

  async function update(id: string, input: TUpdate): Promise<TResult> {
    state.set({ loading: true, error: null });
    try {
      const result = await apiClient.patch<TResult>(` + "`" + `${basePath}/${id}` + "`" + `, input);
      state.set({ loading: false, error: null });
      return result;
    } catch (error) {
      state.set({ loading: false, error: error as Error });
      throw error;
    }
  }

  return { subscribe: state.subscribe, create, update };
}`,

	"stateMachine.ts": `// GENERATED FROM PROPELLER ONTOLOGY. DO NOT HAND-EDIT.
import { writable, derived } from 'svelte/store';
import { apiClient } from '../api/client';

interface Transition {
  target: string;
  label: string;
  variant: 'primary' | 'secondary' | 'danger';
  confirm: boolean;
  confirmMessage?: string;
  endpoint: string;
  requiresFields?: string[];
}

export function stateMachineStore(
  entityType: string,
  entityId: string,
  currentStatus: string,
  transitions: Record<string, Transition[]>,
) {
  const status = writable(currentStatus);
  const available = derived(status, ($status) => transitions[$status] ?? []);

  async function executeTransition(transition: Transition, body?: Record<string, any>): Promise<void> {
    const url = transition.endpoint.replace('{id}', entityId);
    await apiClient.post(url, body);
    status.set(transition.target);
  }

  return { status, available: { subscribe: available.subscribe }, executeTransition };
}`,

	"related.ts": `// GENERATED FROM PROPELLER ONTOLOGY. DO NOT HAND-EDIT.
import { writable } from 'svelte/store';
import { apiClient } from '../api/client';

export function relatedStore<T>(entityType: string, entityId: string, relationship: string) {
  const { subscribe, set, update } = writable<{
    data: T[];
    loading: boolean;
    error: Error | null;
  }>({ data: [], loading: true, error: null });

  async function fetch() {
    update(s => ({ ...s, loading: true }));
    try {
      const result = await apiClient.get<{ data: T[] }>(` + "`" + `/v1/${entityType}s/${entityId}/${relationship}` + "`" + `);
      set({ data: result.data ?? [], loading: false, error: null });
    } catch (error) {
      set({ data: [], loading: false, error: error as Error });
    }
  }

  fetch();
  return { subscribe, refetch: fetch };
}`,
}

// ── Main ─────────────────────────────────────────────────────────────────────

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
	log.SetPrefix("uirender: ")

	projectRoot := findProjectRoot()
	schemaDir := filepath.Join(projectRoot, "gen", "ui", "schema")
	outDir := filepath.Join(projectRoot, "gen", "ui")

	// Load all schema files
	schemas, err := loadSchemas(schemaDir)
	if err != nil {
		log.Fatalf("loading schemas: %v", err)
	}

	// Load enum schema
	allEnums, err := loadEnums(filepath.Join(schemaDir, "_enums.schema.json"))
	if err != nil {
		log.Fatalf("loading enums: %v", err)
	}

	// Build template functions
	funcMap := template.FuncMap{
		"tsType":             tsType,
		"toPascal":           toPascal,
		"toCamel":            toCamel,
		"toCamelHyphen":      toCamelHyphen,
		"toScreamingSnake":   toScreamingSnake,
		"fieldLabel":         fieldLabel,
		"skeletonVariant":    skeletonVariant,
		"replaceID":          replaceID,
		"replaceIDTemplate":  replaceIDTemplate,
		"escapeJS":           escapeJS,
		"dotToOptional":      dotToOptional,
		"derefBool":          derefBool,
		"visibilityCheck":    visibilityCheck,
		"formFieldRender":    formFieldRender,
		"crossFieldCheck":    crossFieldCheck,
		"commonTypeImports":  commonTypeImports,
		"requiredCheck":      requiredCheck,
	}

	// Parse templates
	tmplTypes := mustParseTemplate("types.ts.tmpl", funcMap)
	tmplAPI := mustParseTemplate("api.ts.tmpl", funcMap)
	tmplValidation := mustParseTemplate("validation.ts.tmpl", funcMap)
	tmplForm := mustParseTemplate("form.svelte.tmpl", funcMap)
	tmplDetail := mustParseTemplate("detail.svelte.tmpl", funcMap)
	tmplList := mustParseTemplate("list.svelte.tmpl", funcMap)
	tmplStatusBadge := mustParseTemplate("status_badge.svelte.tmpl", funcMap)
	tmplActions := mustParseTemplate("actions.svelte.tmpl", funcMap)
	tmplEnums := mustParseTemplate("enums.ts.tmpl", funcMap)

	// Ensure output directories
	dirs := []string{
		filepath.Join(outDir, "types"),
		filepath.Join(outDir, "api"),
		filepath.Join(outDir, "validation"),
		filepath.Join(outDir, "stores"),
		filepath.Join(outDir, "components", "shared"),
		filepath.Join(outDir, "components", "sections"),
	}
	for _, d := range dirs {
		os.MkdirAll(d, 0755)
	}

	componentCount := 0

	// Generate per-entity files
	for _, schema := range schemas {
		pascal := toPascal(schema.Entity)
		camel := toCamel(schema.Entity)
		entityDir := filepath.Join(outDir, "components", "entities", schema.Entity)
		os.MkdirAll(entityDir, 0755)

		routePath := strings.TrimPrefix(schema.API.BasePath, "/v1")
		data := templateData{
			UISchema:        schema,
			PascalName:      pascal,
			CamelName:       camel,
			HasStatus:       schema.Status != nil,
			HasStateMachine: schema.StateMachine != nil,
			RoutePath:       routePath,
		}

		if schema.Status != nil {
			for _, f := range schema.Fields {
				if f.Name == "status" && f.EnumRef != "" {
					data.StatusType = f.EnumRef
					if e, ok := schema.Enums[f.EnumRef]; ok && len(e.Values) > 0 {
						data.InitialStatus = e.Values[0].Value
					}
					break
				}
			}
		}

		// Compute imports needed for form based on field types used in form sections
		data.Imports = computeFormImports(schema)

		// Types
		renderTemplate(tmplTypes, data, filepath.Join(outDir, "types", schema.Entity+".types.ts"))
		componentCount++

		// API
		renderTemplate(tmplAPI, data, filepath.Join(outDir, "api", schema.Entity+".api.ts"))
		componentCount++

		// Validation
		renderTemplate(tmplValidation, data, filepath.Join(outDir, "validation", schema.Entity+".validation.ts"))
		componentCount++

		// Form
		renderTemplate(tmplForm, data, filepath.Join(outDir, "components", "entities", schema.Entity, pascal+"Form.svelte"))
		componentCount++

		// Detail
		renderTemplate(tmplDetail, data, filepath.Join(outDir, "components", "entities", schema.Entity, pascal+"Detail.svelte"))
		componentCount++

		// List
		renderTemplate(tmplList, data, filepath.Join(outDir, "components", "entities", schema.Entity, pascal+"List.svelte"))
		componentCount++

		// Status badge (only for entities with status)
		if schema.Status != nil {
			renderTemplate(tmplStatusBadge, data, filepath.Join(outDir, "components", "entities", schema.Entity, pascal+"StatusBadge.svelte"))
			componentCount++
		}

		// Actions (only for entities with state machine)
		if schema.StateMachine != nil {
			renderTemplate(tmplActions, data, filepath.Join(outDir, "components", "entities", schema.Entity, pascal+"Actions.svelte"))
			componentCount++
		}

		fmt.Printf("Generated %s components\n", pascal)
	}

	// Generate enums
	enumData := enumsTemplateData{Enums: allEnums}
	renderTemplate(tmplEnums, enumData, filepath.Join(outDir, "types", "enums.ts"))
	componentCount++
	fmt.Println("Generated types/enums.ts")

	// Generate common types
	writeFile(filepath.Join(outDir, "types", "common.types.ts"), commonTypesContent)
	componentCount++
	fmt.Println("Generated types/common.types.ts")

	// Generate API client
	clientContent, _ := templateFS.ReadFile("templates/client.ts.tmpl")
	writeFile(filepath.Join(outDir, "api", "client.ts"), string(clientContent))
	componentCount++
	fmt.Println("Generated api/client.ts")

	// Generate shared components
	for name, content := range sharedComponents {
		writeFile(filepath.Join(outDir, "components", "shared", name), content)
		componentCount++
	}
	fmt.Printf("Generated %d shared components\n", len(sharedComponents))

	// Generate stores
	for name, content := range storeFiles {
		writeFile(filepath.Join(outDir, "stores", name), content)
		componentCount++
	}
	fmt.Printf("Generated %d stores\n", len(storeFiles))

	fmt.Printf("uirender: generated %d files across %d entities\n", componentCount, len(schemas))
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func loadSchemas(dir string) ([]UISchema, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var schemas []UISchema
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "_") || !strings.HasSuffix(e.Name(), ".schema.json") {
			continue
		}

		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", e.Name(), err)
		}

		var schema UISchema
		if err := json.Unmarshal(data, &schema); err != nil {
			return nil, fmt.Errorf("parsing %s: %w", e.Name(), err)
		}

		schemas = append(schemas, schema)
		entityBasePaths[schema.Entity] = schema.API.BasePath
	}

	sort.Slice(schemas, func(i, j int) bool {
		return schemas[i].Entity < schemas[j].Entity
	})

	return schemas, nil
}

func loadEnums(path string) (map[string]UIEnum, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var enums map[string]UIEnum
	if err := json.Unmarshal(data, &enums); err != nil {
		return nil, err
	}
	return enums, nil
}

func mustParseTemplate(name string, funcMap template.FuncMap) *template.Template {
	data, err := templateFS.ReadFile("templates/" + name)
	if err != nil {
		log.Fatalf("reading template %s: %v", name, err)
	}

	tmpl, err := template.New(name).Funcs(funcMap).Parse(string(data))
	if err != nil {
		log.Fatalf("parsing template %s: %v", name, err)
	}
	return tmpl
}

func renderTemplate(tmpl *template.Template, data any, outPath string) {
	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		log.Fatalf("executing template for %s: %v", outPath, err)
	}
	writeFile(outPath, buf.String())
}

func writeFile(path, content string) {
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		log.Fatalf("writing %s: %v", path, err)
	}
}

const commonTypesContent = `// GENERATED FROM PROPELLER ONTOLOGY. DO NOT HAND-EDIT.

export interface Money {
  amount_cents: number;
  currency: string;
}

export interface Address {
  line1: string;
  line2?: string;
  city: string;
  state: string;
  postal_code: string;
  country?: string;
  county?: string;
  latitude?: number;
  longitude?: number;
}

export interface DateRange {
  start: string;
  end?: string;
}

export interface ContactMethod {
  type: 'email' | 'phone' | 'sms' | 'mail' | 'portal';
  value: string;
  primary: boolean;
  verified: boolean;
  opt_out: boolean;
  label?: string;
}

export interface EntityRef {
  entity_type: string;
  entity_id: string;
  relationship: string;
}

export interface AuditMetadata {
  created_by: string;
  updated_by: string;
  created_at: string;
  updated_at: string;
  source: string;
  correlation_id?: string;
  agent_goal_id?: string;
}

export interface PaginatedResponse<T> {
  data: T[];
  total: number;
  offset: number;
  limit: number;
}
`
