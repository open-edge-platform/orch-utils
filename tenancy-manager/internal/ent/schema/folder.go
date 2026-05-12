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

// Folder groups projects within an org. A "default" folder is created
// automatically with each org; there is no folder CRUD API yet.
type Folder struct {
	ent.Schema
}

func (Folder) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New),
		field.String("name").MaxLen(253).NotEmpty(),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (Folder) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("org", Org.Type).Ref("folders").Unique().Required(),
		edge.To("projects", Project.Type),
	}
}

func (Folder) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("name").Edges("org").Unique(),
	}
}
