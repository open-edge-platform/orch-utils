// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package parser_test

import (
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/vmware-tanzu/graph-framework-for-microservices/compiler/pkg/config"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	examplePath    = "../../example/"
	exampleDSLPath = examplePath + "datamodel"
	baseGroupName  = "tsm.tanzu.vmware.com"
	crdModulePath  = "github.com/vmware-tanzu/graph-framework-for-microservices/compiler/example/output/generated/"
)

func TestParser(t *testing.T) {
	log.StandardLogger().ExitFunc = nil
	RegisterFailHandler(Fail)
	RunSpecs(t, "Parser Suite")
}

func init() {
	conf := &config.Config{
		GroupName:     baseGroupName,
		CrdModulePath: crdModulePath,
	}
	config.ConfigInstance = conf
}
