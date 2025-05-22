// SPDX-FileCopyrightText: 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/spf13/afero"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v3"
)

const (
	targetVersion = "3.0.3"
	inputDir      = "openapispecs/generated"
	convertedDir  = "openapispecs/converted_specs"
	combinedDir   = "openapispecs/combined"
	versionFile   = "VERSION"
	dirPerm       = 0o755
	filePerm      = 0o600
)

func main() {
	app := &cli.App{
		Name:  "OpenAPI Combiner",
		Usage: "Combine multiple OpenAPI spec files into one",
		Action: func(_ *cli.Context) error {
			return run()
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func run() error {
	fs := afero.NewOsFs()

	// Create output directories if they don't exist
	if err := createDir(fs, convertedDir); err != nil {
		return err
	}
	if err := createDir(fs, combinedDir); err != nil {
		return err
	}

	// Read the version from the VERSION file
	version, err := readVersionFile(versionFile)
	if err != nil {
		return fmt.Errorf("failed to read version file: %w", err)
	}

	// Find all YAML files in the input directory
	specFiles, err := afero.Glob(fs, filepath.Join(inputDir, "*.yaml"))
	if err != nil {
		return fmt.Errorf("failed to find YAML files: %w", err)
	}
	if len(specFiles) == 0 {
		return fmt.Errorf("no OpenAPI spec files found in %s", inputDir)
	}

	// Load and process each spec file
	combinedSpec, err := processSpecFiles(specFiles)
	if err != nil {
		return err
	}

	// Add common BearerAuth to the combined spec
	addCommonBearerAuth(combinedSpec)

	// Update the title, info, etc.
	combinedSpec.Info.Title = "Multi tenancy APIs"
	combinedSpec.Info.Description = "Tenancy aware APIs for the Open Edge Platform services"
	combinedSpec.Info.Version = version

	// Save the combined spec
	combinedSpecFile := filepath.Join(combinedDir, "combined_spec.yaml")
	if err := saveSpec(combinedSpec, combinedSpecFile); err != nil {
		return fmt.Errorf("failed to save combined spec: %w", err)
	}

	fmt.Printf("Conversion and combination complete. Combined spec saved to %s.\n", combinedSpecFile)
	return nil
}

func createDir(fs afero.Fs, dir string) error {
	if err := fs.MkdirAll(dir, dirPerm); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}
	return nil
}

func readVersionFile(file string) (string, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func loadSpec(file string) (*openapi3.T, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true
	return loader.LoadFromData(data)
}

func saveSpec(spec *openapi3.T, file string) error {
	var yspec bytes.Buffer
	yenc := yaml.NewEncoder(&yspec)
	yenc.SetIndent(2) //nolint:mnd // this is # of spaces to indent, not a magic number
	defer yenc.Close()

	err := yenc.Encode(&spec)
	if err != nil {
		return err
	}

	// copyright header
	header := []byte(`---
# SPDX-FileCopyrightText: 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

`)

	return os.WriteFile(file, append(header, yspec.Bytes()...), filePerm)
}

func processSpecFiles(specFiles []string) (*openapi3.T, error) {
	var combinedSpec *openapi3.T
	for _, specFile := range specFiles {
		fmt.Printf("Processing %s...\n", specFile)
		spec, err := loadSpec(specFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load spec %s: %w", specFile, err)
		}

		// Update OpenAPI version
		spec.OpenAPI = targetVersion

		// Remove BearerAuth
		delete(spec.Components.SecuritySchemes, "BearerAuth")

		// Add custom query parameter to /vx/projects GET method, datamodel spec file
		addCustomQueryParamToProjectsGet(spec)

		// Save the processed spec
		outputFile := filepath.Join(convertedDir, filepath.Base(specFile))
		if err := saveSpec(spec, outputFile); err != nil {
			return nil, fmt.Errorf("failed to save spec %s: %w", outputFile, err)
		}

		// Combine specs
		if combinedSpec == nil {
			combinedSpec = spec
		} else {
			mergeSpecs(combinedSpec, spec)
		}
	}
	return combinedSpec, nil
}

func mergeSpecs(base, addition *openapi3.T) {
	// Merge paths
	if base.Paths == nil {
		base.Paths = openapi3.NewPaths()
	}
	if addition.Paths != nil {
		for path, item := range addition.Paths.Map() {
			base.Paths.Set(path, item)
		}
	}

	// Merge components
	mergeComponents(base.Components, addition.Components)
}

func mergeComponents(base, addition *openapi3.Components) {
	mergeSchemas(base, addition)
	mergeResponses(base, addition)
	mergeParameters(base, addition)
	mergeExamples(base, addition)
	mergeRequestBodies(base, addition)
	mergeHeaders(base, addition)
	mergeSecuritySchemes(base, addition)
	mergeLinks(base, addition)
	mergeCallbacks(base, addition)
}

func mergeSchemas(base, addition *openapi3.Components) {
	if base.Schemas == nil {
		base.Schemas = make(map[string]*openapi3.SchemaRef)
	}
	for name, schema := range addition.Schemas {
		base.Schemas[name] = schema
	}
}

func mergeResponses(base, addition *openapi3.Components) {
	if base.Responses == nil {
		base.Responses = make(map[string]*openapi3.ResponseRef)
	}
	for name, response := range addition.Responses {
		base.Responses[name] = response
	}
}

func mergeParameters(base, addition *openapi3.Components) {
	if base.Parameters == nil {
		base.Parameters = make(map[string]*openapi3.ParameterRef)
	}
	for name, parameter := range addition.Parameters {
		base.Parameters[name] = parameter
	}
}

func mergeExamples(base, addition *openapi3.Components) {
	if base.Examples == nil {
		base.Examples = make(map[string]*openapi3.ExampleRef)
	}
	for name, example := range addition.Examples {
		base.Examples[name] = example
	}
}

func mergeRequestBodies(base, addition *openapi3.Components) {
	if base.RequestBodies == nil {
		base.RequestBodies = make(map[string]*openapi3.RequestBodyRef)
	}
	for name, requestBody := range addition.RequestBodies {
		base.RequestBodies[name] = requestBody
	}
}

func mergeHeaders(base, addition *openapi3.Components) {
	if base.Headers == nil {
		base.Headers = make(map[string]*openapi3.HeaderRef)
	}
	for name, header := range addition.Headers {
		base.Headers[name] = header
	}
}

func mergeSecuritySchemes(base, addition *openapi3.Components) {
	if base.SecuritySchemes == nil {
		base.SecuritySchemes = make(map[string]*openapi3.SecuritySchemeRef)
	}
	for name, securityScheme := range addition.SecuritySchemes {
		base.SecuritySchemes[name] = securityScheme
	}
}

func mergeLinks(base, addition *openapi3.Components) {
	if base.Links == nil {
		base.Links = make(map[string]*openapi3.LinkRef)
	}
	for name, link := range addition.Links {
		base.Links[name] = link
	}
}

func mergeCallbacks(base, addition *openapi3.Components) {
	if base.Callbacks == nil {
		base.Callbacks = make(map[string]*openapi3.CallbackRef)
	}
	for name, callback := range addition.Callbacks {
		base.Callbacks[name] = callback
	}
}

func addCommonBearerAuth(spec *openapi3.T) {
	if spec.Components.SecuritySchemes == nil {
		spec.Components.SecuritySchemes = make(map[string]*openapi3.SecuritySchemeRef)
	}
	spec.Components.SecuritySchemes["BearerAuth"] = &openapi3.SecuritySchemeRef{
		Value: &openapi3.SecurityScheme{
			Type:         "http",
			Scheme:       "bearer",
			BearerFormat: "JWT",
		},
	}
}

func addCustomQueryParamToProjectsGet(spec *openapi3.T) {
	// Define the new query parameter
	newQueryParam := &openapi3.ParameterRef{
		Value: &openapi3.Parameter{
			Name:     "member-role",
			In:       "query",
			Required: false,
			Schema: &openapi3.SchemaRef{
				Value: &openapi3.Schema{
					Type: &openapi3.Types{openapi3.TypeString},
				},
			},
		},
	}

	// Define the regex pattern to match paths like /v1/projects, /v2/projects, etc.
	pattern := regexp.MustCompile(`^/v\d+/projects$`)

	// Iterate over the paths in the spec
	for path, pathItem := range spec.Paths.Map() {
		if pattern.MatchString(path) {
			// Check if the GET method exists
			if getOperation := pathItem.Get; getOperation != nil {
				// Add the new query parameter to the GET method
				getOperation.Parameters = append(getOperation.Parameters, newQueryParam)
			}
		}
	}
}
