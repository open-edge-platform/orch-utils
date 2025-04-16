// Copyright (C) 2025 Intel Corporation
// SPDX-FileCopyrightText: 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package server

type Router interface {
	Start()
	RegisterRouter(urlPath string)
	RoutesNotification(stopCh chan struct{})
	StopServer()
}
