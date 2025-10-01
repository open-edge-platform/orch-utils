/*
 * Copyright (C) 2025 Intel Corporation
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions
 * and limitations under the License.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package main

import (
	"context"
	"flag"
	"os"

	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
	tenant "github.com/open-edge-platform/orch-utils/tenancy-init/pkg/utils"
	"github.com/rs/zerolog"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var log = logging.GetLogger("tenant-init-job")

func parseFlags(orgName, projectName *string) {
	flag.StringVar(orgName, "org", "", "The name of the organization.")
	flag.StringVar(projectName, "project", "", "The name of the project.")

	flag.Parse()
}

func setupLogging() {
	opts := zap.Options{Development: true}
	opts.BindFlags(flag.CommandLine)
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "error"
	}
	lvl, err := zerolog.ParseLevel(logLevel)
	if err != nil {
		log.Fatal().Msgf("Failed to configure logging: %v\n", err)
	}
	zerolog.SetGlobalLevel(lvl)
}

func main() {
	ctx := context.Background()
	var orgName, projectName string

	parseFlags(&orgName, &projectName)
	setupLogging()

	if err := tenant.CreateSingleTenant(ctx, orgName, projectName); err != nil {
		log.Fatal().Msgf("Failed to create tenant: %v", err)
	}
	log.Info().Msg("Tenant Created successfully")
}
