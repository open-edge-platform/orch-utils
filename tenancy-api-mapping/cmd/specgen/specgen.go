// SPDX-FileCopyrightText: 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"log"

	"github.com/open-edge-platform/orch-utils/tenancy-api-mapping/pkg/config"
	"github.com/open-edge-platform/orch-utils/tenancy-api-mapping/pkg/openapi"
)

func main() {
	if err := Run("config.yaml"); err != nil {
		log.Fatalf("Error: %v", err)
	}
	log.Println("\nSpecGen Completed!")
}

func Run(configPath string) error {
	// read config
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// read APIMappingConfig CRs
	filePaths, err := config.GetCrYAMLFilePaths(cfg.Global.APImappingConfigCrsDir)
	if err != nil {
		return fmt.Errorf("failed to read filepaths from the directory = %s: %w", cfg.Global.APImappingConfigCrsDir, err)
	}

	for _, filePath := range filePaths {
		fmt.Println("about to start processing for filePath:", filePath)
		fBytes, err := config.ReadCRFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read contents from file = %s: %w", filePath, err)
		}

		// Parse the YAML content to get the RepoConf object
		mappingConfigCr, err := config.ParseCRContent(fBytes)
		if err != nil {
			return fmt.Errorf("error parsing cr content for file= %s: %w", filePath, err)
		}

		if !mappingConfigCr.Spec.SpecGenEnabled {
			fmt.Println("skipping generating spec for :", mappingConfigCr.Spec.RepoConf.URL, "as SpecGenEnabled is false")
			continue
		}

		repoConf := mappingConfigCr.Spec.RepoConf

		err = openapi.ProcessOpenAPISpec(mappingConfigCr, cfg.Global)
		if err != nil {
			return fmt.Errorf("failed to create multi-tenancy OpenAPI spec for mapping conf %s : %w",
				mappingConfigCr.Metadata.Name, err)
		}

		fmt.Printf("Created Multi-tenancy OpenAPI spec for repo: %s\n", repoConf.URL)
	}

	return nil
}
