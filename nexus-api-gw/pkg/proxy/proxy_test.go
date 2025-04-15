// Copyright (C) 2025 Intel Corporation
// SPDX-FileCopyrightText: 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package proxy_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/open-edge-platform/orch-utils/nexus-api-gw/pkg/apiremap"
	"github.com/open-edge-platform/orch-utils/nexus-api-gw/pkg/proxy"
	"github.com/stretchr/testify/assert"
)

func TestApiGwToProxy(t *testing.T) {
	// Setup Echo context
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", bytes.NewBuffer([]byte("test body")))
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	var jsonMarshalResponse []byte
	// Setup mock HTTP server
	testServer := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		body, err := io.ReadAll(req.Body)
		assert.NoError(t, err)
		assert.Equal(t, "test body", string(body))
		resp := struct {
			Applications  []string `json:"applications"`
			TotalElements int      `json:"totalElements"`
		}{
			TotalElements: 0,
		}

		jsonMarshalResponse, err = json.Marshal(resp)
		assert.Nil(t, err)
		rw.Write(jsonMarshalResponse)
	}))
	defer testServer.Close()

	// Determined backend server host and port.
	hostNamePort := strings.Split(testServer.URL[len("http://"):], ":")
	u64, err := strconv.ParseUint(hostNamePort[1], 10, 32)
	assert.Nil(t, err)

	//
	// Failure Test.
	//

	// Mock the apiremap.Output to value that has incorrect port of backend server.
	apiremapOutput := apiremap.Output{
		ServiceURI: "test-service-uri",
		Backendinfo: apiremap.BackendInfo{
			SvcName: hostNamePort[0],
			PortStr: 0,
		},
	}

	// Invoke proxy.
	status, msg, _ := proxy.APIGwToProxy(c, apiremapOutput)
	fmt.Println("status :", status)
	fmt.Println("msg :", msg)

	// Assertion that proxy returns an error.
	assert.Equal(t, http.StatusInternalServerError, status)
	assert.Equal(t, "Failed to send new request", msg)

	//
	// Success Test.
	//
	apiremapOutput = apiremap.Output{
		ServiceURI: "test-service-uri",
		Backendinfo: apiremap.BackendInfo{
			SvcName: hostNamePort[0],
			PortStr: uint32(u64),
		},
	}

	// Invoke Proxy.
	status, msg, response := proxy.APIGwToProxy(c, apiremapOutput)

	// Assertion that proxy returns success.
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, "", msg)
	assert.Equal(t, jsonMarshalResponse, response)
}
