// SPDX-FileCopyrightText: 2026 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package mage

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/magefile/mage/sh"
	"gopkg.in/yaml.v3"
)

const binaryName = "tenancy-manager"

var goEnvs = map[string]string{
	"GOPRIVATE":   "github.com/open-edge-platform/*",
	"GOTOOLCHAIN": "local",
}

// goBuild compiles the binary to ./bin/tenancy-manager.
func goBuild() error {
	if err := sh.RunV("mkdir", "-p", "bin"); err != nil {
		return err
	}

	commit, err := gitCommit()
	if err != nil {
		return err
	}

	ldFlags := "-ldflags=-w -s -buildid= -X 'main.Commit=" + commit + "'"
	return sh.RunWithV(goEnvs, "go", "build", ldFlags, "-trimpath", "-o", filepath.Join("bin", binaryName), "./cmd/")
}

// dockerBuild builds the tenancy-manager container image, tagged with the
// appVersion from charts/tenancy-manager/Chart.yaml.
func dockerBuild() error {
	appVersion, err := chartAppVersion()
	if err != nil {
		return err
	}

	registry := os.Getenv("DOCKER_REGISTRY")
	if registry == "" {
		registry = "080137407410.dkr.ecr.us-west-2.amazonaws.com"
	}
	repository := os.Getenv("DOCKER_REPOSITORY")
	if repository == "" {
		repository = "edge-orch"
	}

	tag := registry + "/" + repository + "/common/" + binaryName + ":" + appVersion
	fmt.Printf("Building image: %s\n", tag)

	return sh.RunV(
		"docker", "build",
		"--load",
		"--build-arg", "HTTPS_PROXY="+os.Getenv("HTTPS_PROXY"),
		"--build-arg", "HTTP_PROXY="+os.Getenv("HTTP_PROXY"),
		"--build-arg", "NO_PROXY="+os.Getenv("NO_PROXY"),
		"--build-arg", "https_proxy="+os.Getenv("https_proxy"),
		"--build-arg", "http_proxy="+os.Getenv("http_proxy"),
		"--build-arg", "no_proxy="+os.Getenv("no_proxy"),
		"--tag", tag,
		"--file", "Dockerfile",
		".",
	)
}

// chartAppVersion reads the appVersion from the Helm chart at
// ../charts/tenancy-manager/Chart.yaml (relative to the module root).
func chartAppVersion() (string, error) {
	contents, err := os.ReadFile(filepath.Join("..", "charts", "tenancy-manager", "Chart.yaml"))
	if err != nil {
		return "", fmt.Errorf("read Chart.yaml: %w", err)
	}

	var chart struct {
		AppVersion string `yaml:"appVersion"`
	}
	if err := yaml.Unmarshal(contents, &chart); err != nil {
		return "", fmt.Errorf("parse Chart.yaml: %w", err)
	}
	if chart.AppVersion == "" {
		return "", fmt.Errorf("appVersion in Chart.yaml is empty")
	}
	return chart.AppVersion, nil
}

// gitCommit returns the current HEAD commit hash.
func gitCommit() (string, error) {
	out, err := exec.Command("git", "rev-parse", "HEAD").Output()
	if err != nil {
		return "", fmt.Errorf("git rev-parse HEAD: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}
