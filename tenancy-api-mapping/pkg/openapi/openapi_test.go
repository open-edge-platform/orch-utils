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

package openapi_test

import (
	"io/fs"
	"os"
	"path/filepath"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/open-edge-platform/orch-utils/tenancy-api-mapping/pkg/config"
	"github.com/open-edge-platform/orch-utils/tenancy-api-mapping/pkg/openapi"
)

var _ = ginkgo.Describe("OpenAPISpecProcessor", func() {
	const (
		apiMappingDir               = "apimappingconfigcrs"
		apiMappingName              = "sample_api_mapping"
		defaultFileMode fs.FileMode = 0o600
		defaultDirMode  fs.FileMode = 0o755
	)
	var (
		processor    *openapi.SpecProcessor
		specFilePath string
		mappingCR    config.APIMappingConfig
		cfg          config.Config
	)

	ginkgo.BeforeEach(func() {
		// Set up a temporary directory for testing
		tmpDir, err := os.MkdirTemp("", "openapi_test")
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		localPath := filepath.Join(tmpDir, "gitsubmodules")
		specOpDir := filepath.Join(tmpDir, "genspecs")

		err = os.MkdirAll(localPath, defaultDirMode)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		err = os.MkdirAll(filepath.Join(localPath, apiMappingName), defaultDirMode)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		err = os.MkdirAll(specOpDir, defaultDirMode)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		specFilePath = "test_spec.yaml"

		// Create a test OpenAPI spec file
		testSpecContent := []byte(`openapi: 3.0.0
info:
  title: Simple Pet Store API
  description: A simple API to demonstrate OpenAPI Specification
  version: 1.0.0
servers:
  - url: http://petstore.example.com/api
    description: Production server
paths:
  /pets:
    get:
      summary: List all pets
      operationId: listPets
      tags:
        - pets
      parameters:
        - name: limit
          in: query
          description: How many items to return at one time (max 100)
          required: false
          schema:
            type: integer
            format: int32
      responses:
        '200':
          description: An array of pets
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: '#/components/schemas/Pet'
        default:
          description: Unexpected error
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Error'
    post:
      summary: Create a pet
      operationId: createPets
      tags:
        - pets
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/NewPet'
      responses:
        '201':
          description: Null response
        default:
          description: Unexpected error
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Error'
components:
  schemas:
    Pet:
      type: object
      required:
        - id
        - name
      properties:
        id:
          type: integer
          format: int64
        name:
          type: string
        tag:
          type: string
          nullable: true
    NewPet:
      type: object
      required:
        - name
      properties:
        name:
          type: string
        tag:
          type: string
          nullable: true
    Error:
      type: object
      required:
        - code
        - message
      properties:
        code:
          type: integer
          format: int32
        message:
          type: string`)

		testSpecPath := filepath.Join(localPath, apiMappingName, specFilePath)

		gomega.Expect(os.WriteFile(testSpecPath, testSpecContent, defaultFileMode)).To(gomega.Succeed())

		// Set up test configuration
		cfg = config.Config{
			Global: config.Global{
				LocalSubModsDir:        localPath,
				SpecOutputDir:          specOpDir,
				APImappingConfigCrsDir: apiMappingDir,
				Servers: []config.Server{
					{
						URL: "{apiRoot}",
						Variables: []config.Variable{
							{
								Key:   "apiRoot",
								Value: "https://multi-tenancy-apis.intel.com",
							},
						},
					},
				},
			},
		}
		sampleMappingBytes := []byte(`apiVersion: apimappingconfig.edge-orchestrator.intel.com/v1
kind: APIMappingConfig
metadata:
    name: ` + apiMappingName + `
    labels:
      configs.config.edge-orchestrator.intel.com: default
spec:
    specGenEnabled: true
    repoConf:
        url: "https://github.com/open-edge-platform/infra-core.git"
        tag: "main"
        specFilePath: ` + specFilePath + `
    mappings:
        - externalURI: /v1/projects/{projectName}/telemetry/loggroups/{telemetryLogsGroupId}/logprofiles
          serviceURI: /telemetry/profiles/logs
        - externalURI: /v1/projects/{projectName}/telemetry/loggroups/{telemetryLogsGroupId}/logprofiles/{telemetryLogsProfileId}
          serviceURI: /telemetry/profiles/logs/{telemetryLogsProfileId}
        - externalURI: /v1/projects/{projectName}/telemetry/metricgroups
          serviceURI: /telemetry/groups/metrics
    backend:
        service: "mi-api.orch-ui.cluster.local"
        port: 8080`)

		err = os.MkdirAll(filepath.Join(tmpDir, apiMappingDir), defaultDirMode)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		sampleMappingPath := filepath.Join(tmpDir, apiMappingDir, "sample_api_mapping.yaml")
		gomega.Expect(os.WriteFile(sampleMappingPath, sampleMappingBytes, defaultFileMode)).To(gomega.Succeed())

		// Parse the YAML content to get the RepoConf object
		mappingCR, err = config.ParseCRContent(sampleMappingBytes)
		gomega.Expect(err).To(gomega.BeNil())

		processor, err = openapi.NewOpenAPISpecProcessor(mappingCR, cfg.Global)
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
	})

	ginkgo.AfterEach(func() {
		os.RemoveAll(filepath.Dir(cfg.Global.LocalSubModsDir))
	})

	ginkgo.Describe("Creating a new OpenAPISpecProcessor", func() {
		ginkgo.Context("with valid input parameters", func() {
			ginkgo.It("should not return an error", func() {
				respErr := processor.Process()
				gomega.Expect(respErr).NotTo(gomega.HaveOccurred())
			})

			ginkgo.It("should create a processor with a non-nil spec", func() {
			})
		})

		// Add more Contexts and It blocks to test different scenarios
	})

	ginkgo.Describe("Processing the OpenAPI spec", func() {
		ginkgo.Context("with a valid spec", func() {
			ginkgo.BeforeEach(func() {
				// Set up the processor with a valid spec
			})

			ginkgo.It("should process the spec without errors", func() {
				err := processor.Process()
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
			})

			// Add more tests to check the spec after processing
		})

		// Add more Contexts and It blocks to test different scenarios
	})

	// Add more Describe blocks to test other methods like processPaths, updateSecuritySection, and updateServers
})
