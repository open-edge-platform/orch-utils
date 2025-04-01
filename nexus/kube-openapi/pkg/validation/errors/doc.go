// Copyright 2015 go-swagger maintainers
// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

/*
Package errors provides an Error interface and several concrete types
implementing this interface to manage API errors and JSON-schema validation
errors.

A middleware handler ServeError() is provided to serve the errors types
it defines.

It is used throughout the various go-openapi toolkit libraries
(https://github.com/go-openapi).
*/
package errors
