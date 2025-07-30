// SPDX-FileCopyrightText: 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package openapi

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/open-edge-platform/orch-utils/tenancy-api-mapping/pkg/config"
	"gopkg.in/yaml.v3"
)

const (
	NewFilePrefix     string      = "gen-mt"
	bearerAuthSecName string      = "BearerAuth"
	defaultFileMode   fs.FileMode = 0o600
	defaultDirMode    fs.FileMode = 0o755
	defaultMaxItems   uint64      = 100 // Default maximum items for arrays without explicit maxItems
)

type Spec struct {
	OpenAPI    string                        `json:"openapi" yaml:"openapi"`
	Info       openapi3.Info                 `json:"info" yaml:"info"`
	Servers    openapi3.Servers              `json:"servers,omitempty" yaml:"servers,omitempty"`
	Tags       openapi3.Tags                 `json:"tags,omitempty" yaml:"tags,omitempty"`
	Security   openapi3.SecurityRequirements `json:"security,omitempty" yaml:"security,omitempty"`
	Paths      *openapi3.Paths               `json:"paths" yaml:"paths"`
	Components openapi3.Components           `json:"components,omitempty" yaml:"components,omitempty"`
}

func (spec Spec) MarshalYAML() ([]byte, error) {
	var yspec bytes.Buffer
	yenc := yaml.NewEncoder(&yspec)
	yenc.SetIndent(2) //nolint:mnd // this is # of spaces to indent, not a magic number
	defer yenc.Close()

	err := yenc.Encode(&spec)
	if err != nil {
		return nil, err
	}

	// copyright header
	header := []byte(`---
# SPDX-FileCopyrightText: 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

`)

	return append(header, yspec.Bytes()...), nil
}

type SpecProcessor struct {
	spec      *openapi3.T
	mappingCR config.APIMappingConfig
	cnfGlbl   config.Global
}

func NewOpenAPISpecProcessor(mappingCR config.APIMappingConfig, cnfGlbl config.Global) (*SpecProcessor, error) {
	fullSpecPath := ""

	if mappingCR.Spec.RepoConf.SpecFilePath != "" { // openapi spec file is in a git repo
		fullSpecPath = filepath.Join(cnfGlbl.LocalSubModsDir, mappingCR.Metadata.Name, mappingCR.Spec.RepoConf.SpecFilePath)
	} else { // openapi spec file is downloaded
		dlFilename := mappingCR.Metadata.Name + "_" + mappingCR.Spec.DownloadConf.Version + ".yaml"
		fullSpecPath = filepath.Join(cnfGlbl.LocalDownloadsDir, dlFilename)
	}

	data, err := os.ReadFile(fullSpecPath)
	if err != nil {
		return nil, err
	}

	loader := openapi3.NewLoader()
	spec, err := loader.LoadFromData(data)
	if err != nil {
		return nil, err
	}

	return &SpecProcessor{
		spec:      spec,
		mappingCR: mappingCR,
		cnfGlbl:   cnfGlbl,
	}, nil
}

func (p *SpecProcessor) Process() error {
	if err := p.processPaths(); err != nil {
		return err
	}
	p.updateSecuritySection()
	p.updateServers()
	p.removeParameterDefinition("ActiveProjectIdHeader")
	p.removeParameterDefinition("Authorization")
	p.addMaxItemsToArrays()

	customSpecObj := Spec{
		OpenAPI:    p.spec.OpenAPI,
		Info:       *p.spec.Info,
		Servers:    p.spec.Servers,
		Tags:       p.spec.Tags,
		Security:   p.spec.Security,
		Paths:      p.spec.Paths,
		Components: *p.spec.Components,
	}

	modifiedData, err := customSpecObj.MarshalYAML()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(p.cnfGlbl.SpecOutputDir, defaultDirMode); err != nil {
		return err
	}

	newFileName := GetOutputSpecFileName(p.mappingCR)
	outputPath := filepath.Join(p.cnfGlbl.SpecOutputDir, newFileName)
	return os.WriteFile(outputPath, modifiedData, defaultFileMode)
}

func (p *SpecProcessor) processPaths() error {
	newPaths := openapi3.Paths{}

	// remove deprecated paths
	RemoveDeprecatedOperations(p.spec)

	fmt.Printf("\nabout to process %d paths from the existing openapi spec\n", p.spec.Paths.Len())
	keys := sortedPathKeys(p.spec.Paths)

	for _, path := range keys {
		existingPathItem := p.spec.Paths.Value(path)

		serverURL := getServerURL(p.spec.Servers)
		oldPathKey := constructOldPathKey(serverURL, path)
		if strings.HasPrefix(oldPathKey, "/") {
			oldPathKey = strings.Replace(oldPathKey, "/", "", 1)
		}

		fmt.Printf("\nkey to check : %s", oldPathKey)
		newPathKey, err := findExternalURIByServiceURI(p.mappingCR.Spec.Mappings, oldPathKey)
		if err != nil {
			fmt.Printf("\n \tmapping for url %s not found, hence skipping.", oldPathKey)
			continue
		}

		if err := p.ensurePathParams(existingPathItem, newPathKey); err != nil {
			return err
		}

		// Remove the specific parameter from the path item
		p.removeParameterFromPathItem(existingPathItem, "ActiveProjectIdHeader")
		p.removeParameterFromPathItem(existingPathItem, "Authorization")

		newPaths.Set(newPathKey, existingPathItem)
	}

	p.spec.Paths = &newPaths
	fmt.Printf("\nnumber of paths after process = %d\n", p.spec.Paths.Len())
	fmt.Printf("number of entries in mapping cr = %d\n", len(p.mappingCR.Spec.Mappings))
	return nil
}

func (p *SpecProcessor) removeParameterFromPathItem(pathItem *openapi3.PathItem, paramName string) {
	// Remove the parameter from the path item parameters
	if pathItem.Parameters != nil {
		pathItem.Parameters = removeParameter(pathItem.Parameters, paramName)
	}

	// Remove the parameter from each operation in the path item
	for _, operation := range pathItem.Operations() {
		if operation != nil {
			operation.Parameters = removeParameter(operation.Parameters, paramName)
		}
	}
}

func removeParameter(params openapi3.Parameters, paramName string) openapi3.Parameters {
	newParams := openapi3.Parameters{}
	for _, param := range params {
		if param.Ref != "" && strings.HasSuffix(param.Ref, paramName) {
			continue
		}
		if param.Value != nil && param.Value.Name == paramName {
			continue
		}
		newParams = append(newParams, param)
	}
	return newParams
}

func (p *SpecProcessor) removeParameterDefinition(paramName string) {
	if p.spec.Components.Parameters != nil {
		delete(p.spec.Components.Parameters, paramName)
	}
}

func sortedPathKeys(paths *openapi3.Paths) []string {
	keys := make([]string, 0, paths.Len())
	for key := range paths.Map() {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func getServerURL(servers openapi3.Servers) string {
	if servers != nil {
		return servers[0].URL
	}
	return ""
}

func constructOldPathKey(serverURL, path string) string {
	oldPathKey := GetPathPrefixFromGlobalServerDeclaration(serverURL)
	return strings.Trim(oldPathKey, " ") + path
}

func (p *SpecProcessor) ensurePathParams(existingPathItem *openapi3.PathItem, newPathKey string) error {
	newPPList := extractPathParamNamesFromPlaceholders(newPathKey)

	if existingPathItem.Parameters == nil {
		return p.ensureOperationLevelParams(existingPathItem, newPPList)
	}
	return p.ensurePathLevelParams(existingPathItem, newPPList)
}

func (p *SpecProcessor) ensureOperationLevelParams(pathItem *openapi3.PathItem, paramNames []string) error {
	for method, operation := range pathItem.Operations() {
		if operation != nil {
			fmt.Printf("\nabout to process operation %s", method)
			for _, paramName := range paramNames {
				eParam := containsParam(operation.Parameters, paramName)
				if eParam == nil {
					newParam := createParameterRefForParam(paramName)
					operation.Parameters = append(operation.Parameters, newParam)
				} else if eParam.Value.In != openapi3.ParameterInPath { // do nothing if its path param, else convert to path
					eParam.Value.In = openapi3.ParameterInPath
					eParam.Value.Required = true
				}
			}
		}
	}
	return nil
}

func (p *SpecProcessor) ensurePathLevelParams(pathItem *openapi3.PathItem, paramNames []string) error {
	for _, paramName := range paramNames {
		eParam := containsParam(pathItem.Parameters, paramName)
		if eParam == nil {
			newParam := createParameterRefForParam(paramName)
			pathItem.Parameters = append(pathItem.Parameters, newParam)
		} else if eParam.Value.In != openapi3.ParameterInPath { // do nothing if its path param, else convert to path
			eParam.Value.In = openapi3.ParameterInPath
		}
	}
	return nil
}

func (p *SpecProcessor) updateSecuritySection() {
	bearerAuthSr := map[string][]string{bearerAuthSecName: {}}
	p.spec.Security.With(bearerAuthSr)
	if p.spec.Components == nil {
		p.spec.Components = &openapi3.Components{}
	}
	if p.spec.Components.SecuritySchemes == nil {
		p.spec.Components.SecuritySchemes = make(openapi3.SecuritySchemes)
	}
	p.spec.Components.SecuritySchemes[bearerAuthSecName] = &openapi3.SecuritySchemeRef{Value: openapi3.NewJWTSecurityScheme()}
}

func (p *SpecProcessor) updateServers() {
	servers := make([]*openapi3.Server, 0)
	for _, entry := range p.cnfGlbl.Servers { // loop through servers (in config)
		vMap := make(map[string]*openapi3.ServerVariable)
		for _, v := range entry.Variables { // loop through variables
			vMap[v.Key] = &openapi3.ServerVariable{
				Default: v.Value,
			}
		}

		// Create the server object
		server := &openapi3.Server{
			URL:       entry.URL,
			Variables: vMap,
		}
		servers = append(servers, server)
	}

	p.spec.Servers = servers
}

func findExternalURIByServiceURI(mappings []config.MappingTuple, oldKey string) (string, error) {
	for _, mapping := range mappings {
		if mapping.ServiceURI == oldKey {
			return mapping.ExternalURI, nil
		}
	}
	return "", errors.New("no matching ServiceURI found")
}

func ProcessOpenAPISpec(mappingCR config.APIMappingConfig, cnfGlbl config.Global) error {
	fmt.Printf("\n === about to process openapi spec from repo : %s, file : %s",
		mappingCR.Spec.RepoConf.URL, mappingCR.Spec.RepoConf.SpecFilePath)
	processor, err := NewOpenAPISpecProcessor(mappingCR, cnfGlbl)
	if err != nil {
		return err
	}
	respErr := processor.Process()
	fmt.Printf("=== completed processing openapi spec from repo : %s, file : %s\n",
		mappingCR.Spec.RepoConf.URL, mappingCR.Spec.RepoConf.SpecFilePath)
	return respErr
}

func GetOutputSpecFileName(cfgCR config.APIMappingConfig) string {
	return fmt.Sprintf("%s.%s", cfgCR.Metadata.Name, "yaml")
}

func GetPathPrefixFromGlobalServerDeclaration(input string) string {
	// Find the last index of the '}' character. | workaround as this check is LCM for all the specs
	lastIndex := strings.LastIndex(input, "}")
	if lastIndex == -1 {
		return input
	}

	// Return the substring after the last '}' character.
	// If '}' is the last character in the string, this will return an empty string.
	return input[lastIndex+1:]
}

func extractPathParamNamesFromPlaceholders(url string) []string {
	// Define a regular expression to match path parameters in curly braces
	re := regexp.MustCompile(`{(\w+)}`)

	matches := re.FindAllStringSubmatch(url, -1)

	var params []string
	for _, match := range matches {
		if len(match) > 1 {
			params = append(params, match[1])
		}
	}

	return params
}

func containsParam(slice []*openapi3.ParameterRef, str string) *openapi3.ParameterRef {
	for _, item := range slice {
		if item.Value.Name == str {
			return item
		}
	}
	return nil
}

func createParameterRefForParam(paramName string) *openapi3.ParameterRef {
	return &openapi3.ParameterRef{
		Value: &openapi3.Parameter{
			In:          "path",
			Name:        paramName,
			Required:    true,
			Description: fmt.Sprintf("unique %s for the resource", paramName),
			Schema: &openapi3.SchemaRef{
				Value: openapi3.NewStringSchema(),
			},
		},
	}
}

// RemoveDeprecatedOperations removes deprecated operations from an OpenAPI specification.
func RemoveDeprecatedOperations(doc *openapi3.T) {
	for path, pathItem := range doc.Paths.Map() {
		for method, operation := range pathItem.Operations() {
			if operation.Deprecated {
				fmt.Printf("removed deprecated operation %s, in path %s\n", method, path)
				delete(pathItem.Operations(), method)
			}
		}
		// Remove empty path items
		if len(pathItem.Operations()) == 0 {
			fmt.Printf("removed path %s, as there are no active operations\n", path)
			delete(doc.Paths.Map(), path)
		}
	}
}

// addMaxItemsToArrays adds maxItems constraint to array schemas that don't have one
func (p *SpecProcessor) addMaxItemsToArrays() {
	fmt.Println("Adding maxItems constraints to array schemas...")
	
	// Use configured maxItems or default
	maxItems := p.cnfGlbl.DefaultMaxItems
	if maxItems == 0 {
		maxItems = defaultMaxItems // fallback to constant default
	}
	
	processed := make(map[*openapi3.Schema]bool) // Track processed schemas to avoid infinite recursion
	
	// Process schemas in components
	if p.spec.Components != nil && p.spec.Components.Schemas != nil {
		fmt.Printf("Processing %d component schemas...\n", len(p.spec.Components.Schemas))
		for schemaName, schemaRef := range p.spec.Components.Schemas {
			if schemaRef.Value != nil {
				p.processSchemaForMaxItems(schemaRef.Value, fmt.Sprintf("components.schemas.%s", schemaName), maxItems, processed)
			}
		}
	}
	
	// Process schemas in paths (request/response bodies, parameters)
	if p.spec.Paths != nil {
		fmt.Printf("Processing %d paths...\n", p.spec.Paths.Len())
		pathCount := 0
		for pathName, pathItem := range p.spec.Paths.Map() {
			pathCount++
			if pathCount%5 == 0 {
				fmt.Printf("  Processed %d/%d paths\n", pathCount, p.spec.Paths.Len())
			}
			if pathItem != nil {
				p.processPathItemForMaxItems(pathItem, pathName, maxItems, processed)
			}
		}
	}
	
	fmt.Println("Completed adding maxItems constraints.")
}

// processSchemaForMaxItems recursively processes a schema to add maxItems to arrays
func (p *SpecProcessor) processSchemaForMaxItems(schema *openapi3.Schema, schemaPath string, maxItems uint64, processed map[*openapi3.Schema]bool) {
	if schema == nil || processed[schema] {
		return // Avoid infinite recursion
	}
	processed[schema] = true
	
	// If this is an array schema without maxItems, add it
	if schema.Type.Is("array") && schema.MaxItems == nil {
		schema.MaxItems = &maxItems
		fmt.Printf("Added maxItems=%d to array schema at %s\n", maxItems, schemaPath)
	}
	
	// Process array items
	if schema.Items != nil && schema.Items.Value != nil {
		p.processSchemaForMaxItems(schema.Items.Value, schemaPath+".items", maxItems, processed)
	}
	
	// Process object properties (limit depth to avoid excessive recursion)
	if schema.Properties != nil && len(schemaPath) < 200 { // Depth limit
		for propName, propSchemaRef := range schema.Properties {
			if propSchemaRef.Value != nil {
				p.processSchemaForMaxItems(propSchemaRef.Value, fmt.Sprintf("%s.properties.%s", schemaPath, propName), maxItems, processed)
			}
		}
	}
	
	// Process allOf, anyOf, oneOf (with limits)
	if len(schema.AllOf) > 0 && len(schema.AllOf) < 10 { // Limit processing
		for i, schemaRef := range schema.AllOf {
			if schemaRef.Value != nil {
				p.processSchemaForMaxItems(schemaRef.Value, fmt.Sprintf("%s.allOf[%d]", schemaPath, i), maxItems, processed)
			}
		}
	}
	
	if len(schema.AnyOf) > 0 && len(schema.AnyOf) < 10 { // Limit processing
		for i, schemaRef := range schema.AnyOf {
			if schemaRef.Value != nil {
				p.processSchemaForMaxItems(schemaRef.Value, fmt.Sprintf("%s.anyOf[%d]", schemaPath, i), maxItems, processed)
			}
		}
	}
	
	if len(schema.OneOf) > 0 && len(schema.OneOf) < 10 { // Limit processing
		for i, schemaRef := range schema.OneOf {
			if schemaRef.Value != nil {
				p.processSchemaForMaxItems(schemaRef.Value, fmt.Sprintf("%s.oneOf[%d]", schemaPath, i), maxItems, processed)
			}
		}
	}
	
	// Process additionalProperties
	if schema.AdditionalProperties.Schema != nil && schema.AdditionalProperties.Schema.Value != nil {
		p.processSchemaForMaxItems(schema.AdditionalProperties.Schema.Value, schemaPath+".additionalProperties", maxItems, processed)
	}
}

// processPathItemForMaxItems processes all schemas in a path item
func (p *SpecProcessor) processPathItemForMaxItems(pathItem *openapi3.PathItem, pathName string, maxItems uint64, processed map[*openapi3.Schema]bool) {
	// Process parameters
	for i, paramRef := range pathItem.Parameters {
		if paramRef.Value != nil && paramRef.Value.Schema != nil && paramRef.Value.Schema.Value != nil {
			p.processSchemaForMaxItems(paramRef.Value.Schema.Value, fmt.Sprintf("paths.%s.parameters[%d].schema", pathName, i), maxItems, processed)
		}
	}
	
	// Process operations
	for method, operation := range pathItem.Operations() {
		if operation == nil {
			continue
		}
		
		// Process operation parameters
		for i, paramRef := range operation.Parameters {
			if paramRef.Value != nil && paramRef.Value.Schema != nil && paramRef.Value.Schema.Value != nil {
				p.processSchemaForMaxItems(paramRef.Value.Schema.Value, fmt.Sprintf("paths.%s.%s.parameters[%d].schema", pathName, method, i), maxItems, processed)
			}
		}
		
		// Process request body
		if operation.RequestBody != nil && operation.RequestBody.Value != nil {
			for mediaType, mediaTypeObj := range operation.RequestBody.Value.Content {
				if mediaTypeObj.Schema != nil && mediaTypeObj.Schema.Value != nil {
					p.processSchemaForMaxItems(mediaTypeObj.Schema.Value, fmt.Sprintf("paths.%s.%s.requestBody.content.%s.schema", pathName, method, mediaType), maxItems, processed)
				}
			}
		}
		
		// Process responses
		if operation.Responses != nil {
			for status, responseRef := range operation.Responses.Map() {
				if responseRef.Value != nil && responseRef.Value.Content != nil {
					for mediaType, mediaTypeObj := range responseRef.Value.Content {
						if mediaTypeObj.Schema != nil && mediaTypeObj.Schema.Value != nil {
							p.processSchemaForMaxItems(mediaTypeObj.Schema.Value, fmt.Sprintf("paths.%s.%s.responses.%s.content.%s.schema", pathName, method, status, mediaType), maxItems, processed)
						}
					}
				}
			}
		}
	}
}
