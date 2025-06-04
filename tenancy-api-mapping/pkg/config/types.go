/*
 * Copyright (C) 2025 Intel Corporation
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions
 * and limitations under the License.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package config

type Config struct {
	Global Global `yaml:"global"`
}

type Global struct {
	SpecOutputDir          string   `yaml:"specOutputDir"`
	LocalSubModsDir        string   `yaml:"localSubModsDir"`
	APImappingConfigCrsDir string   `yaml:"apimappingconfigcrsdir"`
	Servers                []Server `yaml:"servers"`
}

type Server struct {
	URL       string     `yaml:"url"`
	Variables []Variable `yaml:"variables"`
}

type Variable struct {
	Key   string `yaml:"key"`
	Value string `yaml:"value"`
}

// APIMappingConfig represents the structure of the CR YAML file.
type APIMappingConfig struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Metadata   struct {
		Name   string            `yaml:"name"`
		Labels map[string]string `yaml:"labels"`
	} `yaml:"metadata"`
	Spec struct {
		SpecGenEnabled bool `yaml:"specGenEnabled"`
		RepoConf       struct {
			URL          string `yaml:"url"`
			Tag          string `yaml:"tag"`
			SpecFilePath string `yaml:"specFilePath"`
		} `yaml:"repoConf"`

		Mappings []MappingTuple `yaml:"mappings"`
	} `yaml:"spec"`
}

type MappingTuple struct {
	ExternalURI string `yaml:"externalURI"` //nolint:tagliatelle // in externalURI, URI is an acronym
	ServiceURI  string `yaml:"serviceURI"`  //nolint:tagliatelle // in serviceURI, URI is an acronym
}
