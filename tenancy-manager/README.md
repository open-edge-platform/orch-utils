# Tenancy Manager Overview

The **Tenancy Manager** is a core component responsible for monitoring multi-tenancy config object creation and
generating corresponding runtime objects for these events. It ensures the correct provisioning, tracking,
and status reporting of org and project creation/deletion.

The **Tenancy Manager** subscribes to config events related to **organization (org), project, orgActiveWatcher,**
and **projectActiveWatcher** creation/deletion. During startup, it registers **Org Watchers** and **Project Watchers**.

## Key Responsibilities

### Org/Project Creation Scenario

- When an org or project is created, the corresponding runtime entity is created.
- It iterates through all registered Org Watchers and verifies the creation of Org Active Watchers.
  Similarly, it checks for the creation of Project Active Watchers corresponding to registered Project Watchers.
- Once all required Active Watchers are successfully created, the Tenancy Manager marks the config status as Complete.
- If at least one Active Watcher is missing or not in a successful state, the configuration status is marked as In Progress.
- In case of any error during Org Active Watcher or Project Active Watcher processing, the status is marked as Error.
- If the Active Watchers are not created within a defined time interval, the org status is set to Timeout.

### Org/Project Deletion Scenario

- When an org or project is deleted, the corresponding runtime entity is deleted.
- The Tenancy Manager verifies that all associated Org Active Watchers and Project Active Watchers are deleted.
- If at least one Active Watcher is present, the configuration status is marked as In Progress.
- If any errors occur during the deletion process, the status is updated to Error.
- If the deletion does not complete within a defined time interval, the status is marked as Timeout.

### Status Reporting

The Tenancy Manager is responsible for reporting the status of org and project creation/deletion back to the user.
This status is communicated through the "status" section of the Org and Project objects in the data model.

The status is represented using well-defined enums that are consistently used across all the components to ensure clear communication.

| **Enum**                          | **Description**                                                                                                                |
|-----------------------------------|--------------------------------------------------------------------------------------------------------------------------------|
| **STATUS_INDICATION_ERROR**       | Error state. Indicates that the last request was not completed successfully.                                                   |
| **STATUS_INDICATION_IN_PROGRESS** | In progress. Indicates that the last request is still being processed.                                                         |
| **STATUS_INDICATION_IDLE**        | Steady state. Indicates that the last request was successfully completed, and the system is idle, waiting for future requests. |

## Get Started

Tenancy Manager gets deployed as a k8s pod along with the deployment of Edge Manageability Framework deployment. But user can also install Tenancy Manager using the helm chart on their own k8s cluster using following command.

```shell
helm install -n orch-iam --create-namespace tenancy-manager charts/tenancy-manager
```

## Develop

- Tenancy-Manager is developed in the **Go** language and is built as a Docker image through a `Dockerfile` in
  the `tenancy-manager` folder. The CI integration for this repository will publish the container image to
  the Edge Orchestrator Release Service OCI registry upon merging to the `main` branch.

- Tenancy-Manager has a corresponding Helm chart in the `charts/tenancy-manager` folder.
  The CI integration for this repository will publish these Helm charts to the Edge Orchestrator Release Service
  OCI registry upon merging to the `main` branch.

- Tenancy-Manager is deployed to the Edge Orchestrator using this Helm chart, whose lifecycle is managed by Argo CD.

### Prerequisites

This code requires the following tools to be installed on your development machine:

- [Go\* programming language](https://go.dev) - check the [Makefile](./Makefile) for usage
- [golangci-lint](https://github.com/golangci/golangci-lint) - check the [Makefile](./Makefile) for usage
- [hadolint](https://github.com/hadolint/hadolint) - check the [Makefile](./Makefile) for usage
- [yamllint](https://github.com/adrienverge/yamllint) - check the [Makefile](./Makefile) for usage
- [reuse](https://github.com/fsfe/reuse-tool) - check the [Makefile](./Makefile) for usage
- Python\* programming language version 3.10 or later
- [gocover-cobertura](https://github.com/boumenot/gocover-cobertura) - check the [Makefile](./Makefile) for usage
- [Docker](https://docs.docker.com/engine/install/) to build containers
- [Helm](https://helm.sh/docs/intro/install/) to install Helm charts for end-to-end tests

### Build, Scan, and Test

The basic workflow to make changes to the code, verify those changes, and create a GitHub pull request (PR) is:

1. Edit and build the code with the `make go-build` command.

2. Run linters with the `make lint` command.

   NOTE: As of now, `make lint` command returns errors. This will be fixed soon.

3. Run the unit tests with the `make test` command.

4. Build the code with the `make build` command to create the docker image.

## Contribute

We welcome contributions from the community! To contribute, please open a pull request to have your changes reviewed
and merged into the `main` branch. We encourage you to add appropriate unit tests and end-to-end tests
if your contribution introduces a new feature.

Additionally, ensure the following commands are successful:

```shell
make test
make lint
make license
make build
```
NOTE: As of now, `make lint` command returns errors. This will be fixed soon.

You can use `help` to see a list of makefile targets.
The following is a list of makefile targets that support developer activities:

- `fmt` to run `go fmt` against Go source files
- `vet` to run `go vet` against Go source files
- `lint` to run a list of linting targets
- `license` to run a scan for the License headers in source code files
- `hadolint` to lint Dockerfile(s) using hadolint
- `go-lint` to lint Go source files with golangci-lint
- `yamllint` to lint YAML files using yamllint
- `go-tidy` to update the Go dependencies and regenerate the `go.sum` file
- `go-build` to build the Go source code files
- `test` to run the unit tests
- `coverage` to run the unit test coverage
- `build` to build the tenancy-manager Docker image
- `release` to publish the built tenancy-manager Docker container to a pre-defined Docker container registry.
  This registry is set in an environment variable (`TENANCY_MANAGER_DOCKER_IMAGE_OEP`) in `tenancy-manager/Makefile`
