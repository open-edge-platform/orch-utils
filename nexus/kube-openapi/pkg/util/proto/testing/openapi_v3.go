// Copyright 2022 The Kubernetes Authors.
// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package testing

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	openapi_v3 "github.com/google/gnostic/openapiv3"
)

type FakeV3 struct {
	Path string

	lock      sync.Mutex
	documents map[string]*openapi_v3.Document
	errors    map[string]error
}

func (f *FakeV3) OpenAPIV3Schema(groupVersion string) (*openapi_v3.Document, error) {
	f.lock.Lock()
	defer f.lock.Unlock()

	if existing, ok := f.documents[groupVersion]; ok {
		return existing, nil
	} else if existingError, ok := f.errors[groupVersion]; ok {
		return nil, existingError
	}

	_, err := os.Stat(f.Path)
	if err != nil {
		return nil, err
	}
	spec, err := ioutil.ReadFile(filepath.Join(f.Path, groupVersion+".json"))
	if err != nil {
		return nil, err
	}

	if f.documents == nil {
		f.documents = make(map[string]*openapi_v3.Document)
	}

	if f.errors == nil {
		f.errors = make(map[string]error)
	}

	result, err := openapi_v3.ParseDocument(spec)
	if err != nil {
		f.errors[groupVersion] = err
		return nil, err
	}

	f.documents[groupVersion] = result
	return result, nil
}
