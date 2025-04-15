// Copyright (C) 2025 Intel Corporation
// SPDX-FileCopyrightText: 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package declarative

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/open-edge-platform/orch-utils/nexus-api-gateway/pkg/config"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	httpClientTimeoutCost   = 5 * time.Second
	ctxTimeoutDurationConst = 10 * time.Second
)

var httpClient = &http.Client{
	Timeout: httpClientTimeoutCost,
}

type errorMessage struct {
	Message string `json:"message"`
}

func ApisHandler(c echo.Context) error {
	crdToSchemaMutex.Lock()
	defer crdToSchemaMutex.Unlock()

	crd := c.QueryParam("crd")
	if crd != "" {
		if val, ok := CrdToSchema[crd]; ok {
			return c.String(http.StatusOK, val)
		}

		return c.NoContent(http.StatusNotFound)
	}

	return c.JSON(http.StatusOK, ApisList)
}

func ListHandler(c echo.Context) error {
	ec, ok := c.(*EndpointContext)
	if !ok {
		log.InfraError("c is not of type '*EndpointContext'").Msg("")
		return fmt.Errorf("context is not of type *EndpointContext")
	}
	log.Debug().Msgf("ListHandler: %s <-> %s", c.Request().RequestURI, ec.SpecURI)

	url, err := BuildURLFromParams(ec)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorMessage{Message: err.Error()})
	}

	log.Debug().Msgf("Making a request to: %s", url)
	// Create a context with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), ctxTimeoutDurationConst)
	defer cancel()

	// Create a new HTTP request with the context
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		log.Printf("Error creating request: %v", err)
		return err
	}

	// Execute the request
	resp, err := httpClient.Do(req)
	if err != nil {
		log.Warn().Msg(err.Error())
		return c.NoContent(http.StatusInternalServerError)
	}
	defer resp.Body.Close()

	var respBody interface{}
	err = json.NewDecoder(resp.Body).Decode(&respBody)
	if err != nil {
		log.Warn().Msg(err.Error())
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.JSON(resp.StatusCode, respBody)
}

func GetHandler(c echo.Context) error {
	ec, ok := c.(*EndpointContext)
	if !ok {
		log.InfraError("c is not of type '*EndpointContext'").Msg("")
		return fmt.Errorf("context is not of type *EndpointContext")
	}
	log.Debug().Msgf("GetHandler: %s <-> %s", c.Request().RequestURI, ec.SpecURI)

	url, err := BuildURLFromParams(ec)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorMessage{Message: err.Error()})
	}

	log.Debug().Msgf("Making a request to: %s", url)
	// Create a context with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), ctxTimeoutDurationConst)
	defer cancel()

	// Create a new HTTP request with the context
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		log.Printf("Error creating request: %v", err)
		return err
	}

	// Execute the request
	resp, err := httpClient.Do(req)
	if err != nil {
		log.Warn().Msg(err.Error())
		return c.NoContent(http.StatusInternalServerError)
	}
	defer resp.Body.Close()

	var respBody interface{}
	err = json.NewDecoder(resp.Body).Decode(&respBody)
	if err != nil {
		log.Warn().Msg(err.Error())
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.JSON(resp.StatusCode, respBody)
}

func PutHandler(c echo.Context) error {
	ec, ok := c.(*EndpointContext)
	if !ok {
		log.InfraError("c is not of type '*EndpointContext'").Msg("")
		return fmt.Errorf("context is not of type *EndpointContext")
	}
	log.Debug().Msgf("PutHandler: %s <-> %s", c.Request().RequestURI, ec.SpecURI)

	body, err := parseRequestBody(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorMessage{Message: err.Error()})
	}

	metadata, err := validateMetadata(body)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorMessage{Message: err.Error()})
	}

	url, err := BuildURLFromBody(ec, metadata)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorMessage{Message: err.Error()})
	}

	req, err := buildPutRequest(url, body)
	if err != nil {
		return fmt.Errorf("unable to build request, err=%w", err)
	}

	return executeRequest(c, req)
}

func parseRequestBody(c echo.Context) (map[string]interface{}, error) {
	body := make(map[string]interface{})
	if err := c.Bind(&body); err != nil {
		log.Warn().Msg(err.Error())
		return nil, fmt.Errorf("unable to parse body")
	}
	return body, nil
}

func validateMetadata(body map[string]interface{}) (map[string]interface{}, error) {
	metadata, ok := body["metadata"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("metadata field not present or not of type map")
	}
	if _, ok := metadata["name"]; !ok {
		return nil, fmt.Errorf("metadata.name field not present")
	}
	return metadata, nil
}

func buildPutRequest(url string, body map[string]interface{}) (*http.Request, error) {
	req, err := http.NewRequest(http.MethodPut, url, http.NoBody)
	if err != nil {
		return nil, err
	}
	if spec, ok := body["spec"]; ok {
		jsonBody, err := json.Marshal(spec)
		if err != nil {
			log.Warn().Msg(err.Error())
			return nil, fmt.Errorf("unable to marshal spec")
		}
		reqBody := bytes.NewBuffer(jsonBody)
		req, err = http.NewRequest(http.MethodPut, url, reqBody)
		if err != nil {
			return nil, err
		}
		log.Debug().Msgf("Body: %s", reqBody.String())
	}
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

func executeRequest(c echo.Context, req *http.Request) error {
	log.Debug().Msgf("Making a request to: %s", req.URL.String())
	ctx, cancel := context.WithTimeout(context.Background(), ctxTimeoutDurationConst)
	defer cancel()

	req = req.WithContext(ctx)

	resp, err := httpClient.Do(req)
	if err != nil {
		log.Warn().Msg(err.Error())
		return c.NoContent(http.StatusInternalServerError)
	}
	defer resp.Body.Close()

	var respBody interface{}
	err = json.NewDecoder(resp.Body).Decode(&respBody)
	if err != nil {
		log.Warn().Msg(err.Error())
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.JSON(resp.StatusCode, respBody)
}

func DeleteHandler(c echo.Context) error {
	ec, ok := c.(*EndpointContext)
	if !ok {
		log.InfraError("c is not of type '*EndpointContext'").Msg("")
		return fmt.Errorf("context is not of type *EndpointContext")
	}
	log.Debug().Msgf("DeleteHandler: %s <-> %s", c.Request().RequestURI, ec.SpecURI)

	url, err := BuildURLFromParams(ec)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorMessage{Message: err.Error()})
	}

	ctx, cancel := context.WithTimeout(context.Background(), ctxTimeoutDurationConst)
	defer cancel()

	// Create a new HTTP request with the context
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, http.NoBody)
	if err != nil {
		log.Printf("Error creating request: %v", err)
		return err
	}

	log.Debug().Msgf("Making a request to: %s", url)

	httpClient := &http.Client{}
	resp, err := httpClient.Do(req)
	if err != nil {
		log.InfraErr(err).Msg("")
		return c.NoContent(http.StatusInternalServerError)
	}
	defer resp.Body.Close()
	return c.NoContent(resp.StatusCode)
}

func BuildURLFromParams(ec *EndpointContext) (string, error) {
	url := config.Cfg.BackendService + ec.SpecURI
	labelSelector, err := metav1.ParseToLabelSelector(ec.QueryParams().Get("labelSelector"))
	if err != nil {
		return "", err
	}
	for _, param := range ec.Params {
		if param[1] == ec.Identifier {
			continue
		}

		labelVal := "default"
		if val, ok := labelSelector.MatchLabels[param[1]]; ok {
			labelVal = val
		}

		url = strings.ReplaceAll(url, param[0], labelVal)
	}

	if ec.Single {
		url = strings.ReplaceAll(url, fmt.Sprintf("{%s}", ec.Identifier), ec.Param("name"))
	}

	return url, nil
}

func BuildURLFromBody(ec *EndpointContext, metadata map[string]interface{}) (string, error) {
	url := config.Cfg.BackendService + ec.SpecURI
	for _, param := range ec.Params {
		if param[1] == ec.Identifier {
			continue
		}

		labelVal := "default"

		if metadata["labels"] != nil {
			metadataLabels, ok := metadata["labels"].(map[string]interface{})
			if !ok {
				return "", fmt.Errorf("metadataLabels is not of type map[string]interface{}")
			}
			if val, ok := metadataLabels[param[1]]; ok {
				labelVal, ok = val.(string)
				if !ok {
					return "", fmt.Errorf("val is not of type string")
				}
			}
		}
		url = strings.ReplaceAll(url, param[0], labelVal)
	}
	metadataName, ok := metadata["name"].(string)
	if !ok {
		return "", fmt.Errorf("metadataName is not of type string")
	}
	url = strings.ReplaceAll(url, fmt.Sprintf("{%s}", ec.Identifier), metadataName)

	return url, nil
}
