// SPDX-FileCopyrightText: 2026 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// TenancyEvent records tenancy lifecycle events (created/deleted) for orgs
// and projects. Written transactionally with the data change so that if
// the data committed, the event committed.
type TenancyEvent struct {
	ent.Schema
}

func (TenancyEvent) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").SchemaType(map[string]string{
			dialect.Postgres: "BIGSERIAL",
		}),
		field.String("event_type").NotEmpty(),    // "created", "deleted"
		field.String("resource_type").NotEmpty(),  // "org", "project"
		field.UUID("resource_id", uuid.UUID{}),
		field.String("resource_name").MaxLen(253),
		field.UUID("org_id", uuid.UUID{}).Optional().Nillable(),
		field.String("org_name").MaxLen(253).Optional().Nillable(),
		field.UUID("folder_id", uuid.UUID{}).Optional().Nillable(),
		field.Time("created_at").Default(time.Now).Immutable(),
	}
}

func (TenancyEvent) Edges() []ent.Edge {
	return nil
}

func (TenancyEvent) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("resource_type", "resource_id"),
	}
}
