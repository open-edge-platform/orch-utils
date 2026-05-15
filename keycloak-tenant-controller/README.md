# Keycloak Tenancy Controller (KTC)

## Description

Keycloak Tenancy Controller (KTC) is an application designed to facilitate the integration between the Tenancy Manager (TM) and Keycloak. It automates the creation of roles and groups in Keycloak when an organization or project is created in the TM.
KTC consumes tenancy lifecycle events from the Tenant Manager REST API using the shared polling library in `orch-library/go/pkg/tenancy`, and uses the event payload UUIDs to create the necessary roles and groups in Keycloak.
These mappings are configured through environmental variables which are populated by Helm values `keycloak_org_groups` and `keycloak_proj_groups`.
This gives the end user flexible and customizable role/group definitions tailored to specific organizational needs. An example configuration can be found in `orch-utils/charts/keycloak-tenant-controller/values.yaml`.

## Features

- **Automated Role/Group Creation**: Automatically creates necessary roles and groups in Keycloak based on organization or project creation events in the TM.
- **Polling-based event consumption**: Uses the `orch-library/go/pkg/tenancy` poller (replay + steady-state) so the controller recovers safely on restart without missing events.
- **Idempotent Init**: Calling `Init()` a second time gracefully stops the previous poller before starting a new one, preventing goroutine leaks.
- **Input validation**: `NewMTClient` and `Init` validate inputs eagerly (empty app name, nil client, invalid or non-HTTP URL) and return structured errors.

## Building the container

From the `orch-utils` directory run `build:keycloakTenantController`

## Configuration

| Environment Variable  | Default                                   | Description                                |
|-----------------------|-------------------------------------------|--------------------------------------------|
| `TENANT_MANAGER_URL`  | `http://tenancy-manager.orch-iam:8080`    | Base URL of the Tenant Manager REST API.   |
| `KEYCLOAK_URL`        | `http://platform-keycloak.orch-platform:8080` | Base URL of the Keycloak instance.     |
| `KEYCLOAK_REALM`      | `master`                                  | Keycloak realm to manage.                  |
| `KEYCLOAK_ORG_GROUPS` | _(set via Helm)_                          | JSON definition of org roles/groups.       |
| `KEYCLOAK_PROJ_GROUPS`| _(set via Helm)_                          | JSON definition of project roles/groups.   |

## Package: `pkg/tdmclient`

`tdmclient` is the bridge between the tenancy polling library and the Keycloak client. It implements `tenancy.Handler` and routes each lifecycle event to the appropriate Keycloak operation.

### Construction

```go
client, err := tdmclient.NewMTClient(appName, kcClient)
if err != nil {
    // appName was empty or kcClient was nil
}
```

### Lifecycle

```go
// Start the poller (validates TENANT_MANAGER_URL, then begins polling).
if err := client.Init(); err != nil {
    // URL invalid, or poller construction failed
}

// Graceful shutdown (safe to call multiple times and before Init).
client.Stop()
```

### Event handling

`HandleEvent` is called for every replayed and incremental tenancy event. The routing table is:

| `ResourceType` | `EventType` | Keycloak operation          |
|----------------|-------------|-----------------------------|
| `org`          | `created`   | `CreateOrg(resourceID)`     |
| `org`          | `deleted`   | `DeleteOrg(resourceID)`     |
| `project`      | `created`   | `CreateProject(orgID, resourceID)` |
| `project`      | `deleted`   | `DeleteProject(resourceID)` |
| _(any other)_  | _(any)_     | logged and ignored (no error) |

All handler methods are idempotent — replay on restart will re-deliver events for existing resources without causing failures.

### Error handling

- `NewMTClient` returns an error for an empty `appName` or a `nil` `kcClient`.
- `Init` validates that `TENANT_MANAGER_URL` is an absolute `http`/`https` URL before creating the poller.
- Keycloak errors from `CreateOrg`, `DeleteOrg`, `CreateProject`, `DeleteProject` are wrapped with context and returned to the poller, which retries on the next poll interval.
- Unexpected poller exits (non-cancellation errors) are logged with `controller=<appName>` context.

## Running tests

```bash
cd keycloak-tenant-controller
go test ./pkg/tdmclient/... -v -count=1
```

The test suite covers:

- `NewMTClient` construction validation (empty name, nil client).
- `validateURL` for valid HTTP/HTTPS, invalid scheme, missing host, and malformed URLs.
- `Init` with a real in-process HTTP test server, default URL fallback, invalid URL rejection, and idempotent re-init.
- `Stop` before `Init`, multiple calls, and context cancellation.
- `HandleEvent` routing for all four event types, error propagation from Keycloak, and graceful handling of unknown events.
- End-to-end event delivery: a live HTTP test server serves a single org-created event and the test asserts `CreateOrg` is called.
