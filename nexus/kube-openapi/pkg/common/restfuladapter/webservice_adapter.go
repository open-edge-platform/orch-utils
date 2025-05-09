// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package restfuladapter

import (
	"github.com/emicklei/go-restful"
	"github.com/vmware-tanzu/graph-framework-for-microservices/kube-openapi/pkg/common"
)

var _ common.RouteContainer = &WebServiceAdapter{}

// WebServiceAdapter adapts a restful.WebService to common.RouteContainer.
type WebServiceAdapter struct {
	WebService *restful.WebService
}

func (r *WebServiceAdapter) RootPath() string {
	return r.WebService.RootPath()
}

func (r *WebServiceAdapter) PathParameters() []common.Parameter {
	var params []common.Parameter
	for _, rParam := range r.WebService.PathParameters() {
		params = append(params, &ParamAdapter{rParam})
	}
	return params
}

func (r *WebServiceAdapter) Routes() []common.Route {
	var routes []common.Route
	for _, rRoute := range r.WebService.Routes() {
		localRoute := rRoute
		routes = append(routes, &RouteAdapter{&localRoute})
	}
	return routes
}
