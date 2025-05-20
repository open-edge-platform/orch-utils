# SPDX-FileCopyrightText: 2025 Intel Corporation
#
# SPDX-License-Identifier: Apache-2.0

package authz

import future.keywords.if
import future.keywords.in

# rules for specific objects in data model
rules := {
	"org-read-role": {"resource": `^/v[a-zA-Z0-9]+/orgs(/[^/]+(/status)?)?$`, "methods": ["get"]},
	"org-write-role": {"resource": `^/v[a-zA-Z0-9]+/orgs/[^/]+$`, "methods": ["put"]},
	"org-delete-role": {"resource": `^/v[a-zA-Z0-9]+/orgs/[^/]+$`, "methods": ["delete"]},
	"project-read-role": {"resource": `^/v[a-zA-Z0-9]+/projects(/[^/]+(/status)?)?$`, "methods": ["get"]},
	"project-write-role": {"resource": `^/v[a-zA-Z0-9]+/projects/[^/]+$`, "methods": ["put"]},
	"project-delete-role": {"resource": `^/v[a-zA-Z0-9]+/projects/[^/]+$`, "methods": ["delete"]},
	"app-deployment-manager-read-role": {"resource": `^/v[a-zA-Z0-9]+/projects/[^/]+/networks(/[^/]+(/status)?)?$`, "methods": ["get"]},
	"app-deployment-manager-write-role": {"resource": `^/v[a-zA-Z0-9]+/projects/[^/]+/networks/[^/]+$`, "methods": ["put","delete"]},
}

# rules for generic objects in data model (not addressed by a specific rule)
member_rules := {
    "member-role": {"resource": `^/v[a-zA-Z0-9]+/projects(/[^/]+(/.*)?)?$`, "methods": ["get","put","post","delete","patch"]},
}

hasSpecificRule if {
    some roleName
    rule := rules[roleName]
    regex.match(rule.resource, input.resource)
}

getMemberRolePattern() = pattern if {
	input.projectId != null
    input.projectId != ""
	pattern = sprintf("^%s_%s_m(ember-role)?$",[input.orgId,input.projectId])
} else = sprintf("^%s_.*_m(ember-role)?$",[input.orgId])

getAppDeployMgrPattern(role) = pattern {
	pattern = sprintf("^%s_(ao-rw|%s)$",[input.projectId,role])
}

first_matching_role(pattern, roles) = item {
	items := [r | r := roles[_]; regex.match(pattern,r)]
	item:= items[0]
} else = null

get_claim_name(roleName) = name if {
    regex.match(`^project-.*$`, roleName)
    input.orgId != null
    input.orgId != ""
    name = sprintf("%s_%s",[input.orgId,roleName])
} else = name if {
    regex.match(`^app-deployment-manager-.*$`, roleName)
    input.orgId != null
    input.orgId != ""
    input.projectId != null
    input.projectId != ""
    memberPattern = getMemberRolePattern
    item := first_matching_role(memberPattern, input.roles)
    item != null
    name = getAppDeployMgrPattern(roleName)
} else = name if {
    regex.match(`^member-role$`, roleName)
    input.orgId != null
    input.orgId != ""
    input.projectId != null
    input.projectId != ""
    name = getMemberRolePattern
#     name = sprintf("%s_%s_%s",[input.orgId,input.projectId,roleName])
} else = roleName

getValidClaim(claim_name) = claimRole if {
   	regex.match(`^.*project-read-role$`, claim_name)
    item := first_matching_role(claim_name, input.roles)
    item != null
    claimRole = {"claim": item, "present": true}
} else = claimRole if {
    input.method == "get"
    memberPattern = getMemberRolePattern
	item := first_matching_role(memberPattern, input.roles)
    item != null
    claimRole = {"claim": item, "present": true}
} else = claimRole if {
	item := first_matching_role(claim_name, input.roles)
    item != null
    claimRole = {"claim": item, "present": true}
} else = {"claim": "", "present": false}

# Rule to check if the input matches the required pattern
allow = result if {
    # check for org specific rules
	some roleName
    rule := rules[roleName]
	regex.match(rule.resource, input.resource)
	rule.methods[_] == input.method
    regex.match(`^org-.*-role$`, roleName)
    item := first_matching_role(roleName, input.roles)
    item != null
    result = {"allow": true, "claim": item}
} else = result if {
    # check for a specific rule first
	some roleName
    rule := rules[roleName]
	regex.match(rule.resource, input.resource)
	rule.methods[_] == input.method
    not regex.match(`^org-.*-role$`, roleName)
    claim_name = get_claim_name(roleName)
    claim_name != roleName
    item := first_matching_role(claim_name, input.roles)
    item != null
    result = {"allow": true, "claim": item}
} else = result if {
    # check for a specific rule first
	some roleName
    rule := rules[roleName]
	regex.match(rule.resource, input.resource)
	rule.methods[_] == input.method
    not regex.match(`^org-.*-role$`, roleName)
    not regex.match(`^app-deployment-manager.*$`, roleName)
    claim_name = get_claim_name(roleName)
    claimRole = getValidClaim(claim_name)
    claimRole.present
    result = {"allow": true, "claim": claimRole.claim}
} else = result if {
    # if no specific rule matched, check the member_rules
    some roleName
    rule := member_rules[roleName]
	regex.match(rule.resource, input.resource)
    not hasSpecificRule
	rule.methods[_] == input.method
    claim_name = get_claim_name(roleName)
    claimRole = getValidClaim(claim_name)
    claimRole.present
    result = {"allow": true, "claim": claimRole.claim}
} else = {"allow": false, "claim": ""}
