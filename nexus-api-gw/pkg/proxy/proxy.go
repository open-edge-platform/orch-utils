// Copyright (C) 2025 Intel Corporation
// SPDX-FileCopyrightText: 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package proxy

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/open-edge-platform/orch-utils/nexus-api-gw/pkg/apiremap"

	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
)

const defaultContextTimeout = 10 * time.Second

var (
	appName = "nexus-api-gw-proxy"
	log     = logging.GetLogger(appName)
)

func APIGwToProxy(c echo.Context, apiremapOp apiremap.Output) (int, string, []byte) {
	log.Info().Msgf("Invoking the NexusApiGwTotenancyProxy with  userURI: %s\n", c.Request().URL)
	// Read ServiceURI

	if strings.Trim(apiremapOp.ServiceURI, " ") == "" {
		log.Info().Msgf("Backend ServiceURI is empty, No mapping found")
		return http.StatusInternalServerError, "unable to determine service backend", nil
	}

	// Read SvcName from Backendinfo
	tenancysvcName := apiremapOp.Backendinfo.SvcName
	if tenancysvcName == "" {
		log.InfraError("tenancy SvcName is not a string").Msg("")
		return http.StatusInternalServerError, "SvcName is not a string", nil
	}

	tenancyPortStr := strconv.FormatUint(uint64(apiremapOp.Backendinfo.PortStr), 10)
	if tenancyPortStr == "" {
		log.InfraError("tenancy PortStr is not a string").Msg("")
		return http.StatusInternalServerError, "PortStr is not a string", nil
	}
	url := fmt.Sprintf("http://%s:%s/%s", tenancysvcName, tenancyPortStr, apiremapOp.ServiceURI)

	// Read the body from the original request
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return http.StatusInternalServerError, "Failed to read request body", nil
	}

	// Restore the original request body so it can be read again
	c.Request().Body = io.NopCloser(bytes.NewBuffer(body))

	// Create a new HTTP method request with the same body
	ctx, cancel := context.WithTimeout(context.Background(), defaultContextTimeout)
	defer cancel()

	// Create a new HTTP request with the context
	newReq, err := http.NewRequestWithContext(ctx, c.Request().Method, url, bytes.NewBuffer(body))
	if err != nil {
		return http.StatusInternalServerError, "Failed to create new request", nil
	}

	// Copy headers from the original request to the new request
	newReq.Header = c.Request().Header.Clone()

	// Send the new request
	client := &http.Client{}
	resp, err := client.Do(newReq)
	if err != nil {
		return http.StatusInternalServerError, "Failed to send new request", nil
	}
	defer resp.Body.Close()

	// Read the response from the new request
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return http.StatusInternalServerError, "Failed to read response body", nil
	}
	log.Info().Msgf("Response from backend services: %s\n", respBody)
	return resp.StatusCode, "", respBody
}
