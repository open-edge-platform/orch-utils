// Copyright (C) 2025 Intel Corporation
// SPDX-FileCopyrightText: 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package mage

import (
	"strings"

	"github.com/magefile/mage/sh"
)

func (Test) golang() error {
	_ = sh.RunV(
		"ginkgo",
		"version",
	)

	// FIXME: Enable all these tests! This is tech debt.
	skippedPackages := []string{
		"auth-service/internal",
		"nexus/common-library/pkg/nexus-compare",
		"nexus/compiler/example/tests",
		"nexus/compiler/pkg/generator",
		"nexus/compiler/pkg/openapi_generator",
		"nexus/compiler/pkg/parser",
		"nexus/compiler/pkg/parser/rest",
		"nexus/compiler/pkg/preparser",
		"nexus/gqlgen/api",
		"nexus/gqlgen/codegen/templates",
		"nexus/gqlgen/codegen/testserver/followschema",
		"nexus/gqlgen/codegen/testserver/singlefile",
		"nexus/gqlgen/internal/code",
		"nexus/gqlgen/internal/rewrite",
		"nexus/gqlgen/graphql/handler/transport",
		"nexus/gqlgen/plugin/federation",
		"nexus/gqlgen/plugin/modelgen",
		"nexus/gqlgen/plugin/resolvergen",
		"nexus/install-validator/pkg/dir",
		"nexus/kube-openapi/pkg/idl",
		"nexus/kube-openapi/pkg/validation/spec",
		"nexus/kube-openapi/pkg/util/proto",
		"nexus/kube-openapi/pkg/util/proto/validation",
		"nexus/kube-openapi/pkg/schemamutation",
		"nexus/kube-openapi/test/integration",
		"tenancy-manager/fuzztest",
		"tenancy-api-mapping/cmd/specgen",
	}

	return sh.RunV(
		"ginkgo",
		"run",
		"-v",
		"-r",
		"-p",
		"--race",
		"-randomize-all",
		"-randomize-suites",
		"--skip-package="+strings.Join(skippedPackages, ","),
	)
}
