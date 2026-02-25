package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/mixin"
)

// AuditMixin provides standard audit fields for every domain entity.
// These fields map directly to the #AuditMetadata CUE type and provide
// full traceability for every change in the system.
type AuditMixin struct {
	mixin.Schema
}

// Fields of the AuditMixin.
func (AuditMixin) Fields() []ent.Field {
	return []ent.Field{
		field.Time("created_at").
			Default(time.Now).
			Immutable().
			Comment("When the entity was created"),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now).
			Comment("When the entity was last updated"),
		field.String("created_by").
			NotEmpty().
			Comment("User ID, agent ID, or 'system' who created this entity"),
		field.String("updated_by").
			NotEmpty().
			Comment("User ID, agent ID, or 'system' who last updated this entity"),
		field.Enum("source").
			Values("user", "agent", "import", "system", "migration").
			Comment("Origin of the change"),
		field.String("correlation_id").
			Optional().
			Nillable().
			Comment("Links related changes across entities"),
		field.String("agent_goal_id").
			Optional().
			Nillable().
			Comment("If source == 'agent', which goal triggered this change"),
	}
}
