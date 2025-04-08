// SPDX-FileCopyrightText: 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

module github.com/vmware-tanzu/graph-framework-for-microservices/compiler

go 1.23.2

require (
	github.com/fatih/structtag v1.2.0
	github.com/ghodss/yaml v1.0.0
	github.com/gogo/protobuf v1.3.2
	github.com/onsi/ginkgo v1.16.5
	github.com/onsi/gomega v1.37.0
	github.com/sirupsen/logrus v1.9.3
	github.com/vmware-tanzu/graph-framework-for-microservices/common-library v0.0.0-20231031085545-baa1f0ece453
	github.com/vmware-tanzu/graph-framework-for-microservices/kube-openapi v0.0.0-20231031085545-baa1f0ece453
	github.com/vmware-tanzu/graph-framework-for-microservices/nexus v0.0.0-20231031085545-baa1f0ece453
	golang.org/x/mod v0.24.0
	golang.org/x/text v0.24.0
	golang.org/x/tools v0.31.0
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/apiextensions-apiserver v0.32.3
	k8s.io/gengo v0.0.0-20250207200755-1244d31929d7
	k8s.io/utils v0.0.0-20250321185631-1f6e0b77f77e
)

require (
	github.com/BurntSushi/toml v1.5.0 // indirect
	github.com/emicklei/go-restful v2.16.0+incompatible // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/fxamacker/cbor/v2 v2.8.0 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-openapi/jsonpointer v0.21.1 // indirect
	github.com/go-openapi/jsonreference v0.21.0 // indirect
	github.com/go-openapi/swag v0.23.1 // indirect
	github.com/gonvenience/bunt v1.4.1 // indirect
	github.com/gonvenience/idem v0.0.2 // indirect
	github.com/gonvenience/neat v1.3.16 // indirect
	github.com/gonvenience/term v1.0.4 // indirect
	github.com/gonvenience/text v1.0.9 // indirect
	github.com/gonvenience/ytbx v1.4.7 // indirect
	github.com/google/gnostic v0.7.0 // indirect
	github.com/google/gnostic-models v0.6.9 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/homeport/dyff v1.10.1 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/lucasb-eyer/go-colorful v1.2.0 // indirect
	github.com/mailru/easyjson v0.9.0 // indirect
	github.com/mattn/go-ciede2000 v0.0.0-20170301095244-782e8c62fec3 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mitchellh/go-ps v1.0.0 // indirect
	github.com/mitchellh/hashstructure v1.1.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/nxadm/tail v1.4.11 // indirect
	github.com/rogpeppe/go-internal v1.14.1 // indirect
	github.com/sergi/go-diff v1.3.2-0.20230802210424-5b0b94c5c0d3 // indirect
	github.com/texttheater/golang-levenshtein v1.0.1 // indirect
	github.com/virtuald/go-ordered-json v0.0.0-20170621173500-b18e6e673d74 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	golang.org/x/net v0.38.0 // indirect
	golang.org/x/sync v0.13.0 // indirect
	golang.org/x/sys v0.32.0 // indirect
	golang.org/x/term v0.31.0 // indirect
	google.golang.org/protobuf v1.36.6 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/tomb.v1 v1.0.0-20141024135613-dd632973f1e7 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	k8s.io/apimachinery v0.32.3 // indirect
	k8s.io/klog/v2 v2.130.1 // indirect
	sigs.k8s.io/json v0.0.0-20241014173422-cfa47c3a1cc8 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.6.0 // indirect
	sigs.k8s.io/yaml v1.4.0 // indirect
)

replace github.com/vmware-tanzu/graph-framework-for-microservices/kube-openapi => ../kube-openapi

replace github.com/vmware-tanzu/graph-framework-for-microservices/gqlgen => ../gqlgen

replace github.com/vmware-tanzu/graph-framework-for-microservices/nexus => ../nexus

replace github.com/vmware-tanzu/graph-framework-for-microservices/common-library => ../common-library
