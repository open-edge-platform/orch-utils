// SPDX-FileCopyrightText: 2026 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// ControllerStatus tracks each controller's current status for each
// resource. This is a persistent, per-controller, per-resource status
// that survives event cleanup and controller restarts.
//
// When a controller processes a "deleted" event successfully, it removes
// its row. The Tenant Manager hard-deletes the soft-deleted resource once
// no rows from registered controllers remain.
type ControllerStatus struct {
	ent.Schema
}

func (ControllerStatus) Fields() []ent.Field {
	return []ent.Field{
		field.String("controller_name").NotEmpty(),
		field.String("resource_type").NotEmpty(), // "org", "project"
		field.UUID("resource_id", uuid.UUID{}),
		field.Enum("status").Values("in_progress", "completed", "error").
			Default("in_progress"),
		field.String("message").Default(""),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (ControllerStatus) Edges() []ent.Edge {
	return nil
}

func (ControllerStatus) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("controller_name", "resource_type", "resource_id").Unique(),
		index.Fields("resource_type", "resource_id"),
	}
}
