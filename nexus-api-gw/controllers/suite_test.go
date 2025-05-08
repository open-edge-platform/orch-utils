// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-FileCopyrightText: 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers_test

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	ginkgo "github.com/onsi/ginkgo/v2"
	gomega "github.com/onsi/gomega"
	ctrls "github.com/open-edge-platform/orch-utils/nexus-api-gw/controllers"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var (
	cfg             *rest.Config
	k8sClient       client.Client
	dynamicClient   dynamic.Interface
	testEnv         *envtest.Environment
	ctx             context.Context
	cancel          context.CancelFunc
	absTestDataPath string
)

const defaultContextTimeout = 10 * time.Second

func initBinaries() {
	requiredBinaries := []string{
		"../test/bin/etcd",
		"../test/bin/kube-apiserver",
		"../test/bin/kubectl",
	}

	if anyFileMissing(requiredBinaries) {
		goos := runtime.GOOS
		goarch := runtime.GOARCH
		fmt.Printf("Detected OS: %s, Architecture: %s\n", goos, goarch)

		url := fmt.Sprintf("https://go.kubebuilder.io/test-tools/1.24.2/%s/%s", goos, goarch)
		fmt.Printf("Downloading envtest binaries from: %s\n", url)

		// Create a context with a timeout
		ctx, cancel := context.WithTimeout(context.Background(), defaultContextTimeout)

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
		if err != nil {
			fmt.Printf("Failed to create request for downloading file: %s, error: %v\n", url, err)
			cancel()
			os.Exit(1)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil || resp.StatusCode != http.StatusOK {
			fmt.Printf("Failed to download file: %s, error: %v, status code: %d\n", url, err, resp.StatusCode)
			cancel()
			os.Exit(1)
		}

		err = os.MkdirAll("../test/bin", os.ModePerm)
		if err != nil {
			fmt.Printf("Failed to create directory: ../test/bin, error: %v\n", err)
			resp.Body.Close()
			cancel()
			os.Exit(1)
		}
		envTestFilePath := "../test/bin/envtest-bins.tar.gz"
		out, err := os.Create(envTestFilePath)
		if err != nil {
			fmt.Printf("Failed to create file: %s, error: %v\n", envTestFilePath, err)
			resp.Body.Close()
			cancel()
			os.Exit(1)
		}

		if _, err = io.Copy(out, resp.Body); err != nil {
			fmt.Printf("Failed to write to file: %s, error: %v\n", envTestFilePath, err)
			resp.Body.Close()
			out.Close()
			cancel()
			os.Exit(1)
		}

		if err = extractTarGz(envTestFilePath, "../test/bin"); err != nil {
			fmt.Printf("Failed to extract tarball: %s, error: %v\n", envTestFilePath, err)
			resp.Body.Close()
			out.Close()
			cancel()
			os.Exit(1)
		}
		resp.Body.Close()
		out.Close()
		cancel()
	}
}

func anyFileMissing(paths []string) bool {
	for _, path := range paths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return true
		}
	}
	return false
}

func extractTarGz(gzipStream, dest string) error {
	file, err := os.Open(gzipStream)
	if err != nil {
		return err
	}
	defer file.Close()

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)

	for {
		header, err := tarReader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}

		if err := processTarHeader(header, tarReader, dest); err != nil {
			return err
		}
	}
	return nil
}

func processTarHeader(header *tar.Header, tarReader *tar.Reader, dest string) error {
	parts := strings.Split(header.Name, string(filepath.Separator))

	switch header.Typeflag {
	case tar.TypeDir:
		return createDir(dest)
	case tar.TypeReg:
		return createFile(dest, parts, tarReader)
	default:
		return fmt.Errorf("unknown type: %v in %s", header.Typeflag, header.Name)
	}
}

func createDir(dest string) error {
	return os.MkdirAll(dest, 0o755)
}

func createFile(dest string, parts []string, tarReader *tar.Reader) error {
	target := filepath.Join(dest, parts[len(parts)-1])
	outFile, err := os.Create(target)
	if err != nil {
		return err
	}
	defer outFile.Close()

	for {
		_, err := io.CopyN(outFile, tarReader, 1024)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return err
		}
	}

	if err := outFile.Chmod(0o755); err != nil {
		return err
	}

	return nil
}

func TestAPIs(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)

	ginkgo.RunSpecs(t, "Controller Suite")
}

var _ = ginkgo.SynchronizedBeforeSuite(func() {
	// run once
	logf.SetLogger(zap.New(zap.WriteTo(ginkgo.GinkgoWriter), zap.UseDevMode(true)))
	fmt.Println("Setting up the test environment")

	absTestDataDir, err := filepath.Abs("../test/tmpdata")
	if err != nil {
		fmt.Printf("Error getting absolute path: %v\n", err)
		os.Exit(1)
	}
	absTestDataPath = absTestDataDir
	if err := os.MkdirAll(absTestDataPath, os.ModePerm); err != nil {
		fmt.Printf("Error creating directory: %v\n", err)
		os.Exit(1)
	}
	initBinaries()

	relativePath := "../test/bin"
	absolutePath, pathErr := filepath.Abs(relativePath)
	if pathErr != nil {
		fmt.Println("Error:", pathErr)
		return
	}
	gomega.Expect(os.Setenv("TEST_ASSET_KUBE_APISERVER", absolutePath+"/kube-apiserver")).To(gomega.Succeed())
	gomega.Expect(os.Setenv("TEST_ASSET_ETCD", absolutePath+"/etcd")).To(gomega.Succeed())
	gomega.Expect(os.Setenv("TEST_ASSET_KUBECTL", absolutePath+"/kubectl")).To(gomega.Succeed())

	logf.SetLogger(zap.New(zap.WriteTo(ginkgo.GinkgoWriter), zap.UseDevMode(true)))
	ctx, cancel = context.WithCancel(context.TODO())

	testEnv = &envtest.Environment{}
	cfg, err = testEnv.Start()
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	// Install CRDs
	opts := envtest.CRDInstallOptions{
		Paths: []string{filepath.Join("..", "test", "crds", "bases")},
	}
	crds, err := envtest.InstallCRDs(cfg, opts)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	err = envtest.WaitForCRDs(cfg, crds, envtest.CRDInstallOptions{})
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	//+kubebuilder:scaffold:scheme
	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(k8sClient).NotTo(gomega.BeNil())

	dynamicClient, err = dynamic.NewForConfig(cfg)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(dynamicClient).NotTo(gomega.BeNil())

	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
		Metrics: metricsserver.Options{
			BindAddress: "0",
		},
	})
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	// DatamodelReconciler
	err = (&ctrls.DatamodelReconciler{
		Client:  k8sManager.GetClient(),
		Scheme:  k8sManager.GetScheme(),
		Dynamic: dynamicClient,
	}).SetupWithManager(k8sManager)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	// CustomResourceDefinitionReconciler
	err = (&ctrls.CustomResourceDefinitionReconciler{
		Client: k8sManager.GetClient(),
		Scheme: k8sManager.GetScheme(),
	}).SetupWithManager(k8sManager)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	go func() {
		defer ginkgo.GinkgoRecover()
		err = k8sManager.Start(ctx)
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
	}()
}, func() {
	// runs on all process nodes
})

var _ = ginkgo.SynchronizedAfterSuite(func() {
	// runs on all process node
}, func() {
	// run once
	// https://github.com/kubernetes-sigs/controller-runtime/issues/1571
	cancel()
	err := testEnv.Stop()
	if err != nil {
		time.Sleep(4 * time.Second)
	}
	err = testEnv.Stop()
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	gomega.Expect(os.Unsetenv("TEST_ASSET_KUBE_APISERVER")).To(gomega.Succeed())
	gomega.Expect(os.Unsetenv("TEST_ASSET_ETCD")).To(gomega.Succeed())
	gomega.Expect(os.Unsetenv("TEST_ASSET_KUBECTL")).To(gomega.Succeed())
})
