<!---
SPDX-FileCopyrightText: 2025 Intel Corporation
SPDX-License-Identifier: Apache-2.0
-->

## Tenant Initializer

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

## Overview

Tenant Initializer is a cloud-native job on the Edge Orchestrator. It
provides a bootstrap tenant during startup if the user wishes to avoid manually 
creating a tenant using scripts or commands. The initializer automatically creates
a tenant-admin user with a secure, randomly generated password that is stored as
a Kubernetes secret for retrieval.

## Get Started

Tenant Initializer gets deployed as a k8s job along with the deployment of
Edge Manageability Framework deployment if multiTenancy mode is disabled. But 
user can also install Tenant Initializer using the helm chart on their own 
k8s cluster using following command.

```shell
helm install -n orch-iam --create-namespace tenancy-init charts/tenancy-init
```

## Tenant Admin Password Management

The Tenant Initializer automatically generates a secure password for the `tenant-admin` user
during the tenant creation process. The password is generated with the following characteristics:

- 16 characters in length
- Contains at least one uppercase letter, lowercase letter, digit, and special character
- Uses cryptographically secure random generation
- Characters are shuffled for additional security

### Password Storage

The generated password is automatically stored as a Kubernetes secret in the same namespace
where the Tenant Initializer job runs (typically `orch-iam`). The secret is named 
`tenant-admin-password` and includes labels for easy identification:

- `app: tenant-init`
- `org: <organization-name>`
- `username: tenant-admin`

### Retrieving the Password

To retrieve the tenant-admin password after tenant initialization, use the following command:

```shell
kubectl get secret tenant-admin-password -n orch-iam -o jsonpath='{.data.admin-password}' | base64 -d
```

You can also view the secret details including labels:

```shell
kubectl describe secret tenant-admin-password -n orch-iam
```

**Note**: The password is base64 encoded in the secret and must be decoded for use.

## Contribute

We welcome contributions from the community! To contribute, please open a pull
request to have your changes reviewed and merged into the main branch. We
encourage you to add appropriate unit tests and e2e tests if your contribution
introduces a new feature. See the [CONTRIBUTING.md](../CONTRIBUTING.md) file
for more information.

Additionally, ensure the following commands are successful:

```shell
make lint
make build
make test
```

## Develop

Tenant Initializer is developed in the **Go** language and is built as a
Docker image through a `Dockerfile` in the `tenancy-init` folder. The CI
integration for this repository will publish the container image to the Edge
Orchestrator Release Service OCI registry upon merging to the `main` branch.

Tenant Initializer has a corresponding Helm chart in the
`charts/tenancy-init` folder. The CI integration for this repository
will publish this Helm chart to the Edge Orchestrator Release Service OCI
registry upon merging to the `main` branch. Tenant Initializer is deployed to
the Edge Orchestrator using this Helm chart, whose lifecycle is managed by Argo
CD (see [Foundational Platform]).

Instructions on how to build, install, and test.

### Prerequisites

This code requires the following tools to be installed on your development machine:

- [Go\* programming language](https://go.dev) - check the
  [Makefile](./Makefile) for usage
- [golangci-lint](https://github.com/golangci/golangci-lint) - check the
  [Makefile](./Makefile) for usage
- [hadolint](https://github.com/hadolint/hadolint) - check the
  [Makefile](./Makefile) for usage
- [yamllint](https://github.com/adrienverge/yamllint) - check the
  [Makefile](./Makefile) for usage
- [gocover-cobertura](https://github.com/boumenot/gocover-cobertura) - check
  the [Makefile](./Makefile) for usage
- [Docker](https://docs.docker.com/engine/install/) to build containers
- [Helm](https://helm.sh/docs/intro/install/) to install Helm charts for
  end-to-end tests

## Community and Support

To learn more about the project, its community, and governance, visit the [Edge
Orchestrator Community](https://github.com/open-edge-platform).

For support, start with
[Troubleshooting](https://github.com/open-edge-platform) or [contact
us](https://github.com/open-edge-platform).

## License

Tenant Initializer is licensed under [Apache
2.0](http://www.apache.org/licenses/LICENSE-2.0).
