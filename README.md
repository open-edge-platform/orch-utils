# Orchestrator Utilities

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0) [![Build](https://github.com/open-edge-platform/orch-utils/actions/workflows/lint-test-build-publish.yml/badge.svg)](https://github.com/open-edge-platform/orch-utils/actions/workflows/lint-test-build-publish.yml)

TODO: Update all links once the GitHub repository is created.

Test coverage


## Overview

The orch-utils repository is a crucial component of the Edge Orchestrator, providing various utility functions and tools
to facilitate the deployment and management of the Edge Orchestrator services. This repository includes utility Helm
charts, Dockerfiles, and Go code that support the Edge Orchestrator.

Key features include:

- Kubernetes Jobs: Facilitates deployments such as Harbor/Vault bootstrap and pod security patches.
- Namespace Creation: Manages the creation of Kubernetes namespaces.
- Release Service Utilities: Includes tools for token refresh and other release service-related tasks.
- Policies: Contains Istio and Kyverno policies for security and traffic management.
- Traefik Routes: Manages Traefik routes for ingress control.
- Traefik Plugins: Provides Traefik plugins for customizing Traefik behavior.
- Keycloak Tenant Controller (KTC): Manages multi-tenancy and user authentication.
- Squid Proxy: Provides a proxy for Edge Nodes in OT networks.

## Get Started

See the [Documentation](https://github.com/intel) to get started using orch-utils.

TODO: Use Make targets before releasing source code.

### Lint

```sh
mage lint:all
```

### Test

```sh
mage test:golang
```

### Build

```sh
mage build:SecretsConfig
mage build:awsSmProxy
mage build:tokenFS
mage build:authService
mage build:certSynchronizer
mage build:squidProxy
mage build:keycloakTenantController
mage ChartsBuild
```

### Release

```sh
echo TODO
```

## Develop

To develop orch-utils, the following development prerequisites are required:

- [Go](https://go.dev/doc/install)
- [Mage](https://magefile.org/)
- [asdf](https://asdf-vm.com/guide/getting-started.html)
- [Docker](https://docs.docker.com/get-docker/)

To build and test orch-utils, first clone the repository:

```sh
git clone https://github.com/open-edge-platform/orch-utils orch-utils

cd orch-utils
```

Then, install the required install tools:

```sh
mage asdfPlugins
```

To build the project, run the [build](#build) command.

## Contribute

To learn how to contribute to the project, see the [Contributor's Guide](/CONTRIBUTING.md).

## Community and Support

To learn more about the project, its community, and governance, visit the [Edge Orchestrator
Community](https://github.com/intel).

For support, start with [Troubleshooting](https://github.com/intel) or [contact us](https://github.com/intel).

## License

```text
Copyright 2025 Intel Corporation

Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the
License. You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an
"AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific
language governing permissions and limitations under the License.
```
