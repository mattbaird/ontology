// ontology/base.cue
package propeller

// ─── Base Entity Types ──────────────────────────────────────────────────────
// Every entity in the system embeds one of these base types.
// Cross-cutting concerns live here once, not repeated per entity.

#BaseEntity: {
	id:    string & !="" @immutable()
	audit: #AuditMetadata @computed()
}

// #StatefulEntity adds a status field with state machine enforcement.
// Embed this in entities that have lifecycle states.
#StatefulEntity: {
	#BaseEntity
	status:             string // refined to specific enum by each entity
	_has_state_machine: true
}

// #ImmutableEntity cannot be updated or deleted after creation.
// Used for ledger entries and audit logs.
#ImmutableEntity: {
	#BaseEntity
	_immutable: true
}
