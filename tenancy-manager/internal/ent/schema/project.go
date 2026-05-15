// SPDX-FileCopyrightText: 2026 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// Project represents a tenant project, scoped to a folder within an org.
type Project struct {
	ent.Schema
}

func (Project) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New),
		field.String("name").MaxLen(253).NotEmpty(),
		field.String("description").Default(""),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
		field.Time("deleted_at").Optional().Nillable(),
	}
}

func (Project) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("folder", Folder.Type).Ref("projects").Unique().Required(),
	}
}

func (Project) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("name").Edges("folder").Unique(),
	}
}
