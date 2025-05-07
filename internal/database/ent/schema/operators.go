package schema

import (
	"rscc/internal/common/utils"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// Operator holds the schema definition for the Operator entity.
type Operator struct {
	ent.Schema
}

// Fields of the Operator.
func (Operator) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").DefaultFunc(utils.GenID).Immutable().Unique(),
		field.String("name").NotEmpty().Unique(),
		field.Time("last_login").Optional().Nillable(),
		field.String("public_key").NotEmpty(),
		field.Bool("is_admin").Default(false),
	}
}

// Edges of the Operator.
func (Operator) Edges() []ent.Edge {
	return nil
}
