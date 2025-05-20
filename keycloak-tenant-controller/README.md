# Keycloak Tenancy Controller (KTC)

## Description

Keycloak Tenancy Controller (KTC) is an application designed to facilitate the integration between the Tenancy Manager (TM) and Keycloak. It automates the creation of roles and groups in Keycloak when an organization or project is created in the TM. This is achieved through event triggers using the Nexus API.
KTC retrieves the organization or project UUID from the Nexus API and uses predefined mappings to create the necessary roles and groups in Keycloak.
These mappings are configured through environmental variables which are populated by Helm values `keycloak_org_groups` and `keycloak_proj_groups`.
This gives the end user flexible and customizable role/group definitions tailored to specific organizational needs. An example configuration can be found in `orch-utils/charts/keycloak-tenant-controller/values.yaml`.

## Features

- **Automated Role/Group Creation**: Automatically creates necessary roles and groups in Keycloak based on organization or project creation events in the TM.

## Building the container

From the `orch-utils` directory run `build:keycloakTenantController`
