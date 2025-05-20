// Copyright (C) 2025 Intel Corporation
// SPDX-FileCopyrightText: 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package echoserver

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/open-edge-platform/orch-utils/nexus-api-gw/pkg/client"
	"github.com/open-edge-platform/orch-utils/nexus-api-gw/pkg/common"
	"github.com/open-edge-platform/orch-utils/nexus-api-gw/pkg/config"
	"github.com/open-edge-platform/orch-utils/nexus-api-gw/pkg/model"
	"github.com/vmware-tanzu/graph-framework-for-microservices/common-library/pkg/nexus"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	labelSelector "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	httpTransportTimeoutCost = 10 * time.Second
	ctxDurationTimeoutConst  = 10 * time.Second
)

// kubeSetupProxy is used to set up reverse proxy to an API server.
func kubeSetupProxy(e *echo.Echo) *httputil.ReverseProxy {
	proxyURL, err := url.Parse(client.Host)
	if err != nil {
		log.Warn().Msgf("Could not parse proxy URL: %v", err)
	}
	proxy := httputil.NewSingleHostReverseProxy(proxyURL)
	if client.HostScheme == "https" {
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(client.HostTLSClientConfig.CAData)
		cert, err := tls.LoadX509KeyPair(client.HostTLSClientConfig.CertFile, client.HostTLSClientConfig.KeyFile)
		if err != nil {
			log.Warn().Msgf("Could not load client certficate: %y+v", err)
		}
		httpTransport := http.Transport{
			Proxy: http.ProxyFromEnvironment,
			Dial: (&net.Dialer{
				Timeout:   httpTransportTimeoutCost,
				KeepAlive: httpTransportTimeoutCost,
			}).Dial,
			TLSHandshakeTimeout: httpTransportTimeoutCost,
			TLSClientConfig: &tls.Config{
				MinVersion:         tls.VersionTLS13,
				MaxVersion:         tls.VersionTLS13,
				RootCAs:            caCertPool,
				Certificates:       []tls.Certificate{cert},
				InsecureSkipVerify: true, // #nosec G402: TLS InsecureSkipVerify set true
			},
		}
		proxy.Transport = &httpTransport
	}
	proxy.ModifyResponse = UpdateProxyResponse
	if common.IsModeAdmin() {
		e.Any("/api/*", echo.WrapHandler(proxy))
		e.Any("/apis/*", echo.WrapHandler(proxy))
		e.Any("/api", echo.WrapHandler(proxy))
		e.Any("/apis", echo.WrapHandler(proxy))
		e.Any("/readyz", echo.WrapHandler(proxy))
		e.Any("/openapi/*", echo.WrapHandler(proxy))
		e.Any("/openapi", echo.WrapHandler(proxy))
		e.Any("/healthz", echo.WrapHandler(proxy))
		e.Any("/readyz", echo.WrapHandler(proxy))
	} else {
		e.Any("/*", echo.WrapHandler(proxy))
	}
	return proxy
}

func UpdateProxyResponse(response *http.Response) error {
	if config.Cfg.CustomNotFoundPage != "" &&
		(response.StatusCode == http.StatusNotFound || response.StatusCode == http.StatusMovedPermanently) {
		// Create a context with a timeout
		ctx, cancel := context.WithTimeout(context.Background(), ctxDurationTimeoutConst)
		defer cancel()

		// Create a new HTTP request with the context
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, config.Cfg.CustomNotFoundPage, http.NoBody)
		if err != nil {
			log.Printf("Error creating request: %v", err)
			return err
		}

		// Create an HTTP client and execute the request
		httpClient := &http.Client{}
		resp, err := httpClient.Do(req)
		if err != nil {
			log.Printf("Error fetching custom not found page: %v", err)
			return err
		}
		defer resp.Body.Close()

		// Read the body from the response
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("Error reading response body: %v", err)
			return err
		}

		// Set the response body to a new ReadCloser
		response.Body = io.NopCloser(bytes.NewReader(body))
		response.Header = resp.Header
		response.StatusCode = resp.StatusCode
		return nil
	}
	return nil
}

// kubeGetByNameHandler is used to process 'kubectl get <resource> <name>' requests.
func KubeGetByNameHandler(c echo.Context) error {
	nc, ok := c.(*NexusContext)
	if !ok {
		return fmt.Errorf("context is not of type *NexusContext")
	}

	gvr := schema.GroupVersionResource{
		Group:    nc.GroupName,
		Version:  "v1",
		Resource: nc.Resource,
	}
	obj, err := client.GetObject(gvr, c.Param("name"), metav1.GetOptions{})
	if err != nil {
		if status := kerrors.APIStatus(nil); errors.As(err, &status) {
			return c.JSON(int(status.Status().Code), status.Status())
		}
		c.Error(err)
	}

	return c.JSON(http.StatusOK, obj)
}

// kubeGetHandler is used to process `kubectl get <resource>' requests.
func KubeGetHandler(c echo.Context) error {
	nc, ok := c.(*NexusContext)
	if !ok {
		return fmt.Errorf("context is not of type *NexusContext")
	}

	opts := metav1.ListOptions{}
	if c.QueryParams().Has("labelSelector") {
		opts.LabelSelector = c.QueryParams().Get("labelSelector")
	}

	if c.QueryParams().Has("limit") {
		i, err := strconv.ParseInt(c.QueryParams().Get("limit"), 10, 64)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, DefaultResponse{Message: err.Error()})
		}
		opts.Limit = i
	}

	if c.QueryParams().Has("continue") {
		opts.Continue = c.QueryParams().Get("continue")
	}

	gvr := schema.GroupVersionResource{
		Group:    nc.GroupName,
		Version:  "v1",
		Resource: nc.Resource,
	}
	log.Debug().Msgf("KubeGetHandler: received GET rquest for %+v", gvr)

	obj, err := client.Client.Resource(gvr).List(context.TODO(), opts)
	if err != nil {
		log.Debug().Msgf("KubeGetHandler: GetObject for %+v failed with error %+v", gvr, err)
		if status := kerrors.APIStatus(nil); errors.As(err, &status) {
			return c.JSON(int(status.Status().Code), status.Status())
		}
		c.Error(err)
	}
	return c.JSON(http.StatusOK, obj)
}

func processBody(body *unstructured.Unstructured, nc *NexusContext,
	crdInfo model.NodeInfo,
) (*unstructured.Unstructured, map[string]string, string, string) {
	displayName := body.GetName()
	labels := body.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}
	labels["nexus/is_name_hashed"] = "true"
	labels["nexus/display_name"] = displayName

	orderedLabels := nexus.ParseCRDLabels(crdInfo.ParentHierarchy, labels)
	for _, key := range orderedLabels.Keys() {
		value, _ := orderedLabels.Get(key)
		key, ok := key.(string)
		if !ok {
			log.InfraError("key is not of type string").Msg("")
		}

		val, ok := value.(string)
		if !ok {
			log.InfraError("val is not of type string").Msg("")
		}
		labels[key] = val
	}

	hashedName := nexus.GetHashedName(nc.CrdType, crdInfo.ParentHierarchy, labels, displayName)
	body.SetLabels(labels)
	body.SetName(hashedName)

	if crdInfo.DeferredDelete {
		finalizerVal := "nexus.com/nexus-deferred-delete"
		body.SetFinalizers([]string{finalizerVal})
		log.Debug().Msgf("Object %s is marked for deferred delete, added finalizer %v", body.GetName(), body.GetFinalizers())
	}

	return body, labels, hashedName, displayName
}

// KubePostHandler is used to process `kubectl apply` requests.

func KubePostHandler(c echo.Context) error {
	nc, ok := c.(*NexusContext)
	if !ok {
		return fmt.Errorf("context is not of type *NexusContext")
	}
	crdInfo := model.CrdTypeToNodeInfo[nc.CrdType]

	body := &unstructured.Unstructured{}
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, DefaultResponse{Message: err.Error()})
	}

	body, labels, hashedName, _ := processBody(body, nc, crdInfo)

	gvr := schema.GroupVersionResource{
		Group:    nc.GroupName,
		Version:  "v1",
		Resource: nc.Resource,
	}
	log.Debug().Msgf("KubePostHandler: received POST request for %+v", gvr)

	obj, err := getObjectOrCreate(gvr, hashedName, body, labels, crdInfo)
	if err != nil {
		return handleError(c, err)
	}

	body.SetResourceVersion(obj.GetResourceVersion())
	if err := updateSpec(body, obj, crdInfo); err != nil {
		return c.JSON(http.StatusInternalServerError, DefaultResponse{Message: err.Error()})
	}

	obj, err = client.Client.Resource(gvr).Update(context.TODO(), body, metav1.UpdateOptions{})
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(http.StatusOK, obj)
}

func getObjectOrCreate(gvr schema.GroupVersionResource, hashedName string, body *unstructured.Unstructured,
	labels map[string]string, crdInfo model.NodeInfo,
) (*unstructured.Unstructured, error) {
	obj, err := client.GetObject(gvr, hashedName, metav1.GetOptions{})
	if err != nil {
		if kerrors.IsNotFound(err) {
			if len(crdInfo.ParentHierarchy) > 0 {
				parentCrdName := crdInfo.ParentHierarchy[len(crdInfo.ParentHierarchy)-1]
				parentCrd := model.CrdTypeToNodeInfo[parentCrdName]
				if _, err := client.GetParent(parentCrdName, parentCrd, labels); err != nil {
					log.Debug().Msgf("KubePostHandler: GetParent failed with error %+v", err)
					return nil, err
				}
			}

			if _, ok := body.UnstructuredContent()["spec"]; !ok {
				content := body.UnstructuredContent()
				content["spec"] = map[string]interface{}{}
				body.SetUnstructuredContent(content)
			}
			return client.Client.Resource(gvr).Create(context.TODO(), body, metav1.CreateOptions{})
		}
		return nil, err
	}
	return obj, nil
}

func updateSpec(body, obj *unstructured.Unstructured, crdInfo model.NodeInfo) error {
	spec, ok := obj.Object["spec"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("spec is not of type map")
	}
	newSpec, ok := body.Object["spec"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("newSpec is not of type map")
	}

	for _, v := range crdInfo.Children {
		if value, ok := spec[v.FieldNameGvk]; ok {
			newSpec[v.FieldNameGvk] = value
		}
	}
	for _, v := range crdInfo.Links {
		if value, ok := spec[v.FieldNameGvk]; ok {
			newSpec[v.FieldNameGvk] = value
		}
	}
	body.Object["spec"] = newSpec
	return nil
}

func handleError(c echo.Context, err error) error {
	if status := kerrors.APIStatus(nil); errors.As(err, &status) {
		return c.JSON(int(status.Status().Code), status.Status())
	}
	return c.JSON(http.StatusInternalServerError, err.Error())
}

func KubeDeleteHandler(c echo.Context) error {
	nc, ok := c.(*NexusContext)
	if !ok {
		return fmt.Errorf("context is not of type *NexusContext")
	}
	crdInfo := model.CrdTypeToNodeInfo[nc.CrdType]
	gvr := schema.GroupVersionResource{
		Group:    nc.GroupName,
		Version:  "v1",
		Resource: nc.Resource,
	}
	labels := make(map[string]string)
	name := c.Param("name")
	log.Debug().Msgf("KubeDeleteHandler: display name: %s", name)

	if c.QueryParams().Has("labelSelector") {
		labelsMap, err := labelSelector.ConvertSelectorToLabelsMap(c.QueryParams().Get("labelSelector"))
		if err != nil {
			return c.JSON(http.StatusBadRequest, DefaultResponse{Message: err.Error()})
		}
		for key, val := range labelsMap {
			labels[key] = val
		}
	}

	name = nexus.GetHashedName(nc.CrdType, crdInfo.ParentHierarchy, labels, name)
	log.Debug().Msgf("KubeDeleteHandler: hashedName: %s, labels: %s", name, labels)

	err := client.DeleteObject(gvr, nc.CrdType, crdInfo, name)
	if err != nil {
		if status := kerrors.APIStatus(nil); errors.As(err, &status) {
			return c.JSON(int(status.Status().Code), status.Status())
		}
		c.Error(err)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"kind":       "Status",
		"apiVersion": "v1",
		"metadata":   map[string]interface{}{},
		"status":     "Success",
		"details": map[string]interface{}{
			"name":  c.Param("name"),
			"group": nc.GroupName,
			"kind":  nc.Resource,
		},
	})
}
