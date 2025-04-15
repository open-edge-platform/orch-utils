// Copyright (C) 2025 Intel Corporation
// SPDX-FileCopyrightText: 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package utils_test

import (
	"testing"

	"github.com/open-edge-platform/orch-utils/nexus-api-gateway/pkg/utils"
)

func TestPlural(t *testing.T) {
	cases := []struct {
		typeName string
		expected string
	}{
		{
			"I",
			"I",
		},
		{
			"Pod",
			"Pods",
		},
		{
			"Entry",
			"Entries",
		},
		{
			"Bus",
			"Buses",
		},
		{
			"Fizz",
			"Fizzes",
		},
		{
			"Search",
			"Searches",
		},
		{
			"Autograph",
			"Autographs",
		},
		{
			"Dispatch",
			"Dispatches",
		},
		{
			"Earth",
			"Earths",
		},
		{
			"City",
			"Cities",
		},
		{
			"Ray",
			"Rays",
		},
		{
			"Fountain",
			"Fountains",
		},
		{
			"Life",
			"Lives",
		},
		{
			"Leaf",
			"Leaves",
		},
	}
	for _, c := range cases {
		if e, a := c.expected, utils.ToPlural(c.typeName); e != a {
			t.Errorf("Unexpected result from plural namer. Expected: %s, Got: %s", e, a)
		}
	}
}
