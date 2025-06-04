// SPDX-FileCopyrightText: 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"github.com/open-edge-platform/orch-utils/tenancy-datamodel/config/apimappingconfig"
	"github.com/open-edge-platform/orch-utils/tenancy-datamodel/config/org"
	"github.com/open-edge-platform/orch-utils/tenancy-datamodel/config/orgwatcher"
	"github.com/open-edge-platform/orch-utils/tenancy-datamodel/config/projectwatcher"
	"github.com/open-edge-platform/orch-utils/tenancy-datamodel/nexus/base/nexus"
)

// Config tree.
type Config struct {
	nexus.SingletonNode

	// Organizations created by User.
	Orgs org.Org `nexus:"children"`

	// Components to be notified of org create/delete.
	OrgWatchers orgwatcher.OrgWatcher `nexus:"children"`

	// APIMappings to support backend services.
	APIMappings apimappingconfig.APIMappingConfig `nexus:"children"`

	// Components to be notified of project create/delete.
	ProjectWatchers projectwatcher.ProjectWatcher `nexus:"children"`
}
