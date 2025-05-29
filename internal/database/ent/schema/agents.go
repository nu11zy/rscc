package schema

import (
	"rscc/internal/common/utils"
	"time"

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
		field.Time("created_at").Default(time.Now).Immutable(),
		field.String("name").Immutable().Unique().NotEmpty(),
		field.String("os").Immutable().NotEmpty(),
		field.String("arch").Immutable().NotEmpty(),
		field.Strings("servers").Immutable(),
		field.Bool("shared").Immutable().Default(false),
		field.Bool("pie").Immutable().Default(false),
		field.Bool("garble").Immutable().Default(false),
		field.Strings("subsystems").Immutable().Default([]string{}),
		field.String("xxhash").Immutable().NotEmpty(),
		field.String("path").Immutable().NotEmpty(),
		field.Bytes("public_key").Immutable().NotEmpty(),
		field.String("url").Immutable().NotEmpty(),
		field.Int("hits").Default(0),
	}
}

// Edges of the Agent.
func (Agent) Edges() []ent.Edge {
	return nil
}
