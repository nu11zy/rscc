package schema

import (
	"rscc/internal/common/utils"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// User holds the schema definition for the User entity.
type User struct {
	ent.Schema
}

// Fields of the User.
func (User) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").DefaultFunc(utils.GenID).Immutable().Unique(),
		field.String("name").NotEmpty(),
		field.Time("last_activity").Optional().Nillable(),
		field.String("public_key").NotEmpty(),
		field.Bool("is_admin").Default(false),
	}
}

// Edges of the User.
func (User) Edges() []ent.Edge {
	return nil
}
