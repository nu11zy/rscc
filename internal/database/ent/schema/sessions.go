package schema

import (
	"rscc/internal/common/utils"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// Session holds the schema definition for the Session entity.
type Session struct {
	ent.Schema
}

// Fields of the Session.
func (Session) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").DefaultFunc(utils.GenID).Immutable().Unique(),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.String("agent_id").Immutable().NotEmpty(),
		field.String("username").Immutable().NotEmpty(),
		field.String("hostname").Immutable().NotEmpty(),
		field.String("domain").Immutable().Default(""),
		field.Bool("is_priv").Immutable().Default(false),
		field.Strings("ips").Immutable(),
		field.String("os_meta").Immutable().Default(""),
		field.String("proc_name").Immutable().Default(""),
		field.String("extra").Immutable().Default(""),
	}
}

// Edges of the Session.
func (Session) Edges() []ent.Edge {
	return nil
}
