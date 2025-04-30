package schema

import (
	"rscc/internal/common/utils"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// Listener holds the schema definition for the Listener entity.
type Listener struct {
	ent.Schema
}

// Fields of the Listener.
func (Listener) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").DefaultFunc(utils.GenID).Immutable().Unique(),
		field.String("name").Unique().NotEmpty(),
		field.Bytes("private_key").NotEmpty(),
	}
}

// Edges of the Listener.
func (Listener) Edges() []ent.Edge {
	return nil
}
