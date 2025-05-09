// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package singlefile

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vmware-tanzu/graph-framework-for-microservices/gqlgen/client"
	"github.com/vmware-tanzu/graph-framework-for-microservices/gqlgen/graphql/handler"
)

func TestValidType(t *testing.T) {
	resolvers := &Stub{}
	resolvers.QueryResolver.ValidType = func(ctx context.Context) (validType *ValidType, e error) {
		return &ValidType{
			DifferentCase:    "new",
			DifferentCaseOld: "old",
		}, nil
	}

	c := client.New(handler.NewDefaultServer(NewExecutableSchema(Config{Resolvers: resolvers})))

	t.Run("fields with differing cases can be distinguished", func(t *testing.T) {
		var resp struct {
			ValidType struct {
				New string `json:"differentCase"`
				Old string `json:"different_case"`
			}
		}
		err := c.Post(`query { validType { differentCase, different_case } }`, &resp)
		require.NoError(t, err)

		require.Equal(t, "new", resp.ValidType.New)
		require.Equal(t, "old", resp.ValidType.Old)
	})
}
