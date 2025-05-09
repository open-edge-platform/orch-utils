// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package followschema

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vmware-tanzu/graph-framework-for-microservices/gqlgen/client"
	"github.com/vmware-tanzu/graph-framework-for-microservices/gqlgen/graphql/handler"
)

type FieldsOrderPayloadResults struct {
	OverrideValueViaInput struct {
		FirstFieldValue *string `json:"firstFieldValue"`
	} `json:"overrideValueViaInput"`
}

func TestFieldsOrder(t *testing.T) {
	resolvers := &Stub{}

	c := client.New(handler.NewDefaultServer(NewExecutableSchema(Config{Resolvers: resolvers})))
	resolvers.FieldsOrderInputResolver.OverrideFirstField = func(ctx context.Context, in *FieldsOrderInput, data *string) error {
		if data != nil {
			in.FirstField = data
		}
		return nil
	}
	resolvers.MutationResolver.OverrideValueViaInput = func(ctx context.Context, in FieldsOrderInput) (ret *FieldsOrderPayload, err error) {
		ret = &FieldsOrderPayload{
			FirstFieldValue: in.FirstField,
		}
		return
	}

	t.Run("firstField", func(t *testing.T) {
		var resp FieldsOrderPayloadResults

		err := c.Post(`mutation {
			overrideValueViaInput(input: { firstField:"newName" }) {
				firstFieldValue
			}
		}`, &resp)
		require.NoError(t, err)

		require.NotNil(t, resp.OverrideValueViaInput.FirstFieldValue)
		require.Equal(t, "newName", *resp.OverrideValueViaInput.FirstFieldValue)
	})

	t.Run("firstField/override", func(t *testing.T) {
		var resp FieldsOrderPayloadResults

		err := c.Post(`mutation { overrideValueViaInput(input: {
				firstField:"newName",
				overrideFirstField: "override"
			}) {
				firstFieldValue
			}
		}`, &resp)
		require.NoError(t, err)

		require.NotNil(t, resp.OverrideValueViaInput.FirstFieldValue)
		require.NotEqual(t, "newName", *resp.OverrideValueViaInput.FirstFieldValue)
		require.Equal(t, "override", *resp.OverrideValueViaInput.FirstFieldValue)
	})
}
