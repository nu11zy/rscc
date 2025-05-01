package schema

import (
	"rscc/internal/common/utils"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// Agent holds the schema definition for the Agent entity.
type Agent struct {
	ent.Schema
}

// Fields of the Agent.
func (Agent) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").DefaultFunc(utils.GenID).Immutable().Unique(),
		field.String("name").Immutable().Unique().NotEmpty(),
		field.String("os").Immutable().NotEmpty(),
		field.String("arch").Immutable().NotEmpty(),
		field.String("addr").Immutable().NotEmpty(),
		field.Bytes("public_key").Immutable().NotEmpty(),
		field.Uint64("xxhash").Immutable(),
		field.Int("status").Default(0), // 0: default, 1: modified, 2: deleted
	}
}

// Edges of the Agent.
func (Agent) Edges() []ent.Edge {
	return nil
}
