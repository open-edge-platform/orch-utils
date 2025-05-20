// Copyright (C) 2025 Intel Corporation
// SPDX-FileCopyrightText: 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package declarative

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/labstack/echo/v4"
	"sigs.k8s.io/yaml"

	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
)

var (
	supportedOperations = []string{"GET", "DELETE", "PUT"}
	appName             = "nexus-api-gw-openapi"
	log                 = logging.GetLogger(appName)
)

const (
	constStrObject     = "object"
	constStrArray      = "array"
	constStrString     = "string"
	constNumber        = 1.2
	constInt       int = 1
)

const (
	NexusKindName     = "x-nexus-kind-name"
	NexusGroupName    = "x-nexus-group-name"
	NexusListEndpoint = "x-nexus-list-endpoint"
	NexusShortName    = "x-nexus-short-name"
	OpenAPISpecFile   = "/openapi/openapi.yaml"
	OpenAPISpecDir    = "/openapi"
)

var (
	Paths              = make(map[string]*openapi3.PathItem)
	ApisList           = make(map[string]map[string]interface{})
	apisListMutex      = sync.Mutex{}
	Schema             = openapi3.T{}
	Schemas            openapi3.Schemas
	parsedSchemas      = make(map[string]interface{})
	parsedSchemasMutex = sync.Mutex{}
	CrdToSchema        = make(map[string]string)
	crdToSchemaMutex   = sync.Mutex{}
)

func Setup(openAPISpecFile string) error {
	_, err := os.Stat(openAPISpecFile)
	if err == nil {
		f, err := os.ReadFile(openAPISpecFile)
		if err != nil {
			return err
		}

		return Load(f)
	}
	log.InfraError("File %v is not present at setup", openAPISpecFile).Msg("")
	return nil
}

func Load(data []byte) error {
	doc, err := openapi3.NewLoader().LoadFromData(data)
	if err != nil {
		return err
	}

	Schemas = doc.Components.Schemas
	Schema = *doc

	for uri, pathInfo := range doc.Paths {
		if !ValidateNexusAnnotations(pathInfo) {
			continue
		}
		Paths[uri] = pathInfo
	}

	ParseSchemas()

	return nil
}

func ValidateNexusAnnotations(item *openapi3.PathItem) bool {
	for _, supportedOperation := range supportedOperations {
		op := item.GetOperation(supportedOperation)
		if op != nil {
			if GetExtensionVal(op, NexusKindName) == "" {
				return false
			}

			if GetExtensionVal(op, NexusGroupName) == "" {
				return false
			}
		}
	}

	return true
}

func GetExtensionVal(operation *openapi3.Operation, key string) string {
	val, ok := operation.Extensions[key]
	if !ok || val == nil {
		return ""
	}

	rawMsg, ok := val.(json.RawMessage)
	if !ok {
		// val is not of type json.RawMessage, log and assume as string
		rawMsg = json.RawMessage(fmt.Sprint(val))
	}

	out, err := rawMsg.MarshalJSON()
	if err != nil {
		// handle the error or return an empty string
		return ""
	}

	outStr := string(out)

	if strings.HasPrefix(outStr, `"`) && strings.HasSuffix(outStr, `"`) && len(outStr) > 2 {
		return outStr[1 : len(outStr)-1]
	}

	return outStr
}

func AddApisEndpoint(ec *EndpointContext) {
	apisListMutex.Lock()
	crdToSchemaMutex.Lock()
	defer func() {
		apisListMutex.Unlock()
		crdToSchemaMutex.Unlock()
	}()

	if ApisList[ec.URI] == nil {
		ApisList[ec.URI] = make(map[string]interface{})
	}

	params := make([]string, 0, len(ec.Params))
	for _, param := range ec.Params {
		params = append(params, param[1])
	}

	ApisList[ec.URI][ec.Method] = map[string]interface{}{
		"group":  ec.GroupName,
		"kind":   ec.KindName,
		"params": params,
		"uri":    ec.SpecURI,
	}

	if ec.SchemaName != "" {
		schema := ConvertSchemaToYaml(ec, params)
		ApisList[ec.URI]["yaml"] = schema
		CrdToSchema[fmt.Sprintf("%s.%s", ec.ResourceName, ec.GroupName)] = schema
	}

	if ec.ShortURI != "" {
		ApisList[ec.URI]["short"] = map[string]interface{}{
			"name": ec.ShortName,
			"uri":  ec.ShortURI,
		}
	}
}

func ConvertSchemaToYaml(ec *EndpointContext, params []string) string {
	labels := map[string]interface{}{}
	for _, param := range params {
		if param != ec.Identifier {
			labels[param] = constStrString
		}
	}

	obj := map[string]interface{}{
		"apiVersion": ec.GroupName + "/v1",
		"kind":       ec.KindName,
		"metadata": map[string]interface{}{
			"name":   constStrString,
			"labels": labels,
		},
	}
	obj["spec"] = parsedSchemas[ec.SchemaName]
	yamlObj, err := yaml.Marshal(obj)
	if err != nil {
		log.Warn().Msg(err.Error())
	}
	return string(yamlObj)
}

func parseSchema(schemaName string, wg *sync.WaitGroup) {
	parsedSchemasMutex.Lock()
	defer func() {
		parsedSchemasMutex.Unlock()
		wg.Done()
	}()

	spec := make(map[string]interface{})

	for field, val := range Schemas[schemaName].Value.Properties {
		spec[field] = parseField(schemaName, val)
	}

	parsedSchemas[schemaName] = spec
}

func parseField(schemaName string, val *openapi3.SchemaRef) interface{} {
	switch val.Value.Type {
	case constStrString:
		return parseStringField(val)
	case "boolean":
		return true
	case "number":
		return constNumber
	case "integer":
		return constInt
	case constStrArray:
		return parseArrayField(schemaName, val)
	case constStrObject:
		return constStrObject
	default:
		return parseRefField(schemaName, val)
	}
}

func parseStringField(val *openapi3.SchemaRef) interface{} {
	if len(val.Value.Enum) > 0 {
		return val.Value.Enum[0]
	}
	return constStrString
}

func parseArrayField(schemaName string, val *openapi3.SchemaRef) interface{} {
	if val.Value.Items.Ref != "" {
		ref := openapi3.DefaultRefNameResolver(val.Value.Items.Ref)
		if ref == schemaName {
			return constStrObject
		}
		return map[string]interface{}{
			"ref":  ref,
			"type": constStrArray,
		}
	} else if val.Value.Items.Value.Type == constStrString {
		return []string{val.Value.Items.Value.Type}
	}
	return nil
}

func parseRefField(schemaName string, val *openapi3.SchemaRef) interface{} {
	if val.Ref != "" {
		ref := openapi3.DefaultRefNameResolver(val.Ref)
		if ref == schemaName {
			return constStrObject
		}
		return map[string]interface{}{
			"ref": ref,
		}
	}
	return nil
}

func parseSchemaRefs(schemaName string, wg *sync.WaitGroup) {
	parsedSchemasMutex.Lock()
	defer func() {
		parsedSchemasMutex.Unlock()
		wg.Done()
	}()

	schemas, ok := parsedSchemas[schemaName].(map[string]interface{})
	if !ok {
		log.Warn().Msg("parsedSchemas[schemaName] is not of type map[string]interface{}")
		return
	}
	for fieldName, fieldVal := range schemas {
		if _, ok := fieldVal.(map[string]interface{}); !ok {
			continue
		}

		fv, ok := fieldVal.(map[string]interface{})
		if !ok {
			log.Warn().Msg("fieldVal is not of type map")
			continue
		}
		ref := fv["ref"]
		refType := fv["type"]

		if ref == nil || ref == schemaName {
			continue
		}

		refStr, ok := ref.(string)
		if !ok {
			log.Warn().Msg("ref is not of type string")
			continue
		}
		if refType == constStrArray {
			schemasRefStr, ok := parsedSchemas[refStr].(map[string]interface{})
			if !ok {
				log.Warn().Msg("parsedSchemas[refStr] is not of type map[string]interface{}")
				continue
			}
			schemas[fieldName] = []map[string]interface{}{
				schemasRefStr,
			}
			continue
		}

		schemas[fieldName] = parsedSchemas[refStr]
	}
}

func ParseSchemas() {
	wg := &sync.WaitGroup{}
	for schemaName := range Schemas {
		wg.Add(1)
		log.Debug().Msgf("Parsing %s schema", schemaName)
		go parseSchema(schemaName, wg)
	}
	wg.Wait()

	for schemaName := range parsedSchemas {
		wg.Add(1)
		log.Debug().Msgf("Parsing %s schema refs", schemaName)
		go parseSchemaRefs(schemaName, wg)
	}
	wg.Wait()
	log.Debug().Msgf("Finished parsing schemas")
}

func Middleware(endpointContext *EndpointContext, single bool) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			endpointContext.Context = c
			endpointContext.Single = single
			return next(endpointContext)
		}
	}
}
