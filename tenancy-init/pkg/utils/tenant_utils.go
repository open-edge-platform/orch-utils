// SPDX-FileCopyrightText: 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"context"
	"encoding/base64"
	"fmt"

	"os"
	"os/exec"
	"slices"
	"strings"
	"time"

	"github.com/Nerzal/gocloak/v13"
	"github.com/bitfield/script"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
	folderv1 "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/apis/folder.edge-orchestrator.intel.com/v1"
	orgsv1 "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/apis/org.edge-orchestrator.intel.com/v1"
	projectv1 "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/apis/project.edge-orchestrator.intel.com/v1"
	nexus_client "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/nexus-client"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	KeycloakRealm = "master"

	defaultServicePort   = 443
	defaultClusterDomain = "kind.internal"
	adminUser            = "admin"
)

// autoCert is a package-level variable initialized during startup.
var autoCert = func() bool {
	value := os.Getenv("AUTO_CERT")
	return value == "1"
}()

var coderEnv = func() bool {
	value, exists := os.LookupEnv("CODER_WORKSPACE_NAME")
	return exists && value != ""
}()

var (
	serviceDomainWithPort = fmt.Sprintf("%s:%d", serviceDomain, defaultServicePort)
	log                   = logging.GetLogger("tenant-init-job")
)

// serviceDomain is a package-level variable initialized during startup.
var serviceDomain = func() string {
	sd := os.Getenv("E2E_SVC_DOMAIN")
	// retrieve svcdomain from configmap
	// if it does not exist there, then use the defaultservicedomain
	if sd == "" {
		domain, err := LookupOrchestratorDomain()
		if err != nil {
			// Could not locate in the configmap, also attempt to build the domain if this is an autocert (coder) deployment
			if autoCert && coderEnv {
				// retrieve the subdomain name
				subDomain := os.Getenv("ORCH_DOMAIN")
				if subDomain == "" {
					log.Error().Msgf("error retrieving the orchestrator domain from autocert aws lookup\n")
					return defaultClusterDomain
				}
				return subDomain
			}
			return defaultClusterDomain
		}

		if len(domain) > 0 {
			sd = domain
		} else {
			sd = defaultClusterDomain
		}
	}

	return sd
}()

// LookupOrchestratorDomain returns the domain for the orchestrator
// By default it will look for the domain in the configmap orchestrator-domain
func LookupOrchestratorDomain() (string, error) {
	kubeCmd := "kubectl -n orch-gateway get configmap orchestrator-domain -o json"
	data, err := script.NewPipe().Exec(kubeCmd).String()
	if err != nil {
		return "", fmt.Errorf("error getting k8s dns Names from configmap: %w", err)
	}
	domain, err := script.Echo(data).JQ(".data.orchestratorDomainName").Replace(`"`, "").String()
	domain = strings.ReplaceAll(domain, "\n", "")

	if err != nil {
		return "", fmt.Errorf("error getting k8s dns Names from configmap: %w", err)
	}
	return domain, nil
}

// CreateSingleTenant creates one Org, one Project, one Project admin, CO and Edge Infrastructure Manager users in the Project.
func CreateSingleTenant(ctx context.Context, orgName, projectName string) error {
	log.Info().Msg("Creating organization...")
	err := CreateOrg(ctx, orgName)
	if err != nil {
		return fmt.Errorf("failed to create org: %w", err)
	}
	log.Info().Msg("Organization created successfully.")

	/* projectAdminUser := fmt.Sprintf("%s-admin", orgName)
	log.Info().Msgf("Creating project admin user '%s' in organization '%s'...\n", projectAdminUser, orgName)
	err = CreateProjectAdminInOrg(ctx, orgName, projectAdminUser)
	if err != nil {
		return fmt.Errorf("failed to create project admin in org: %w", err)
	}
	log.Info().Msg("Project admin user created successfully.") */

	log.Info().Msgf("Creating project '%s' in organization '%s'...\n", projectName, orgName)
	err = CreateProjectInOrg(ctx, orgName, projectName)
	if err != nil {
		return fmt.Errorf("failed to create project in org: %w", err)
	}
	log.Info().Msg("Project created successfully.")

	/* log.Info().Msgf("Creating edge infra users in project '%s'...\n", projectName)
	err = CreateEdgeInfraUsers(ctx, orgName, projectName, projectName)
	if err != nil {
		return fmt.Errorf("failed to create edge infra users: %w", err)
	}
	log.Info().Msg("Edge infra users created successfully.")

	log.Info().Msgf("Creating cluster orchestration users in project '%s'...\n", projectName)
	err = CreateClusterOrchUsers(ctx, orgName, projectName, projectName)
	if err != nil {
		return fmt.Errorf("failed to create cluster orch users: %w", err)
	}
	log.Info().Msg("Cluster orchestration users created successfully.")
	*/
	return nil
}

// CreateOrg creates an Org in the system
func CreateOrg(ctx context.Context, org string) error {
	orgId, _ := getOrgId(ctx, org)
	if orgId != "" {
		log.Info().Msgf("Org (%s) already present with UID (%s)\n", org, orgId)
		return nil
	}
	log.Info().Msgf("Creating Org (%s)\n", org)
	config := ctrl.GetConfigOrDie()
	nexusClient, err := nexus_client.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("\nerror creating org (%s). Error: %w", org, err)
	}
	configNode := nexusClient.TenancyMultiTenancy().Config()
	if configNode == nil {
		return fmt.Errorf("\nerror creating org (%s). Error: %w", org, err)
	}

	_, err = configNode.AddOrgs(ctx, &orgsv1.Org{
		ObjectMeta: metav1.ObjectMeta{
			Name: org,
		},
		Spec: orgsv1.OrgSpec{
			Description: org,
		},
	})
	if err != nil {
		return fmt.Errorf("\nerror creating org (%s). Error: %w", org, err)
	}
	nexusClient.SubscribeAll()

	msg, err := waitUntilOrgWatcherCreation(ctx, nexusClient, org)
	if err != nil {
		return fmt.Errorf("ktc orgactivewatcher for org %s failed to be created with error %w", org, err)
	}
	if msg == "Created" {
		log.Info().Msgf("\nktc orgactivewatcher for org %s created\n", org)
	} else {
		log.Info().Msgf("\nktc orgactivewatcher for org %s status - %s\n", org, msg)
	}

	orgUUID, err := waitUntilOrgCreation(ctx, nexusClient, org)
	if err != nil {
		return fmt.Errorf("wait for org %s to go active failed with error %w", org, err)
	}
	log.Info().Msgf("\nOrg (%s) has UID: %s\n", org, orgUUID)
	return nil
}

func waitUntilOrgWatcherCreation(ctx context.Context, nexusClient *nexus_client.Clientset, org string) (string, error) {
	println("\nwaiting until org active watchers are completed")
	runtimeNode := nexusClient.TenancyMultiTenancy().Runtime()
	timeout := time.After(5 * time.Minute)
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			orgRuntimeNode, err := runtimeNode.GetOrgs(ctx, org)
			if err != nil {
				return "", err
			}
			ktcWatcher, err := orgRuntimeNode.GetActiveWatchers(ctx, "keycloak-tenant-controller")
			if err != nil {
				return "", err
			}
			if string(ktcWatcher.Spec.StatusIndicator) == string(orgsv1.StatusIndicationIdle) {
				return ktcWatcher.Spec.Message, nil
			}
		case <-timeout:
			return "", fmt.Errorf("KTC active watcher for org %s creation timed out", org)
		}
	}
}

func waitUntilOrgCreation(ctx context.Context, nexusClient *nexus_client.Clientset, org string) (string, error) {
	println("\nwaiting until org creation is completed")
	configNode := nexusClient.TenancyMultiTenancy().Config()
	timeout := time.After(5 * time.Minute)
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			orgNode, _ := configNode.GetOrgs(ctx, org)
			if orgNode.Status.OrgStatus.StatusIndicator == orgsv1.StatusIndicationIdle {
				return orgNode.Status.OrgStatus.UID, nil
			}
		case <-timeout:
			return "", fmt.Errorf("org %s creation timed out", org)
		}
	}
}

func getOrgId(ctx context.Context, orgName string) (string, error) {
	config := ctrl.GetConfigOrDie()
	nexusClient, err := nexus_client.NewForConfig(config)
	if err != nil {
		return "", fmt.Errorf("\nerror checking if the org (%s) already present. Error: %w", orgName, err)
	}
	configNode := nexusClient.TenancyMultiTenancy().Config()
	if configNode == nil {
		return "", fmt.Errorf("\nerror checking if the org (%s) already present: Error: %w", orgName, err)
	}

	org, _ := configNode.GetOrgs(ctx, orgName)
	if org == nil {
		log.Info().Msgf("org %s does not exist.\n", orgName)
		return "", nil
	}

	orgStatus, err := org.GetOrgStatus(ctx)
	if orgStatus == nil || err != nil {
		return "", fmt.Errorf("\nerror checking if the org (%s) already present: Error: %w", orgName, err)
	}

	return orgStatus.UID, nil
}

// GetOrg gets the Org in the system
func GetOrg(ctx context.Context, orgName string) error {
	config := ctrl.GetConfigOrDie()
	nexusClient, err := nexus_client.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("\nerror retrieving the org (%s). Error: %w", orgName, err)
	}
	configNode := nexusClient.TenancyMultiTenancy().Config()
	if configNode == nil {
		return fmt.Errorf("\nerror retrieving the org (%s). Error: %w", orgName, err)
	}

	org := configNode.Orgs(orgName)
	if org == nil {
		log.Info().Msgf("org %s does not exist.\n", orgName)
		return nil
	}

	orgStatus, err := org.GetOrgStatus(ctx)
	if orgStatus == nil || err != nil {
		return fmt.Errorf("\nerror retrieving the org (%s). Error: %w", orgName, err)
	}
	if strings.HasPrefix(orgStatus.Message, "Waiting for watchers") && strings.HasSuffix(orgStatus.Message, "to be deleted") {
		return fmt.Errorf("\norg (%s) is waiting for watchers to be deleted with status message '%s'", orgName, orgStatus.Message)
	}
	if orgStatus.Message != fmt.Sprintf("Org %s CREATE is complete", orgName) {
		return fmt.Errorf("\norg (%s) status message is set to %s", orgName, orgStatus.Message)
	}
	log.Info().Msgf("\nOrg status message - %s, Org status - %s\n", orgStatus.Message, orgStatus.StatusIndicator)
	// println("orgId: ", orgStatus.UID)
	return nil
}

// CreateProjectAdminInOrg creates a Project Admin in a given Org
func CreateProjectAdminInOrg(ctx context.Context, orgName string, projectAdminUser string) error {
	// TODO - Add a check to determine if ktc is running

	client, token, err := KeycloakLogin(ctx)
	if err != nil {
		return err
	}

	userId, orgId, err := createKeycloakUser(ctx, client, token, projectAdminUser, orgName)
	if err != nil {
		return err
	}

	groups := []string{orgId + "_Project-Manager-Group"}

	err = addUserToGroups(ctx, client, token, KeycloakRealm, groups, userId)
	if err != nil {
		log.Error().Msgf("error adding org roles to user %s", projectAdminUser)
		return err
	}

	return nil
}

func addUserToGroups(ctx context.Context, client *gocloak.GoCloak, token *gocloak.JWT, realm string, groupsList []string, userId string) error {
	for _, groupName := range groupsList {
		group, err := getGroup(ctx, client, token, realm, groupName)
		if err != nil {
			log.Error().Msgf("error fetching group %s", groupName)
			return err
		}
		err = client.AddUserToGroup(ctx, token.AccessToken, realm, userId, *group.ID)
		if err != nil {
			log.Error().Msgf("error adding org roles to the user %s", userId)
			return err
		}
		log.Info().Msgf("added user %s to group %s\n", userId, groupName)
	}

	return nil
}

func getGroup(ctx context.Context, client *gocloak.GoCloak, token *gocloak.JWT, realm string, groupName string) (*gocloak.Group, error) {
	group, err := client.GetGroupByPath(ctx, token.AccessToken, realm, groupName)
	if err != nil {
		log.Error().Msgf("Failed to retrieve group %s in realm %s: %v", groupName, realm, err)
		return nil, err
	}
	return group, nil
}

// Prints the default Orchestrator password.
func OrchPassword() error {
	pass, err := GetDefaultOrchPassword()
	if err != nil {
		return err
	}

	log.Info().Msgf("Default Orch Password: %s\n", pass)

	return nil
}

func GetDefaultOrchPassword() (string, error) {
	pass := os.Getenv("ORCH_DEFAULT_PASSWORD")
	if pass == "" {
		log.Info().Msg("Environment variable ORCH_DEFAULT_PASSWORD is empty, attempting to retrieve password using kubectl command...")

		output, err := exec.Command(
			"kubectl",
			"get",
			"secret",
			"platform-keycloak",
			"-n", "orch-platform",
			"-o", "jsonpath={.data.admin-password}",
		).CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("failed to get password from kubectl command: %w\noutput: %s", err, string(output))
		}
		encodedPass := string(output)

		log.Info().Msgf("Decoding base64 password... %s\n", encodedPass)

		decodedPass, err := base64.StdEncoding.DecodeString(strings.TrimSpace(encodedPass))
		if err != nil {
			return "", fmt.Errorf("failed to decode base64 password: %w", err)
		}

		pass = strings.TrimSpace(string(decodedPass))
		if pass == "" {
			return "", fmt.Errorf("empty password retrieved from kubectl command")
		}

		log.Info().Msg("Password successfully retrieved using kubectl command")
	}

	// Keycloak is configured to require: "passwordPolicy": "length(14) and digits(1) and specialChars(1) and upperCase(1) and lowerCase(1)",
	// and this variable is used in the MT setup to create the users so it must respect that policy.

	// check password length
	if len(pass) < 14 {
		return "", fmt.Errorf("password length less than 14 characters")
	}

	// check for at least one digit
	if !strings.ContainsAny(pass, "0123456789") {
		return "", fmt.Errorf("password does not contain a digit")
	}

	// check for at least one special character
	if !strings.ContainsAny(pass, "!@#$%^&*()_+-=[]{}|;:,.<>?") {
		return "", fmt.Errorf("password does not contain a special character")
	}

	// check for at least one upper case letter
	if strings.ToLower(pass) == pass {
		return "", fmt.Errorf("password does not contain an upper case letter")
	}

	// check for at least one lower case letter
	if strings.ToUpper(pass) == pass {
		return "", fmt.Errorf("password does not contain a lower case letter")
	}

	return pass, nil
}

func GetKeycloakSecret() (string, error) {
	kubecmd := fmt.Sprintf("kubectl get secret -n %s platform-keycloak -o jsonpath='{.data.admin-password}' ", "orch-platform")
	pass, err := script.Exec(kubecmd).String()
	if err != nil {
		return "", err
	}

	// Decode the Base64 string
	encodedPass, err := base64.StdEncoding.DecodeString(pass)
	if err != nil {
		return "", fmt.Errorf("error decoding Base64 string: %w", err)
	}

	// Convert the decoded bytes to a string
	adminPass := string(encodedPass)

	if adminPass == "" {
		return "", fmt.Errorf("password string empty")
	}

	return adminPass, nil
}

func KeycloakLogin(ctx context.Context) (*gocloak.GoCloak, *gocloak.JWT, error) {
	keycloakURL := "https://keycloak." + serviceDomainWithPort

	// retrieve admin user and password from keycloak secret
	adminPass, err := GetKeycloakSecret()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get keycloak admin password")
	}

	client := gocloak.NewClient(keycloakURL)

	jwtToken, err := client.LoginAdmin(ctx, adminUser, adminPass, KeycloakRealm)
	if err != nil {
		log.Error().Msgf("%v", err)
		return nil, nil, fmt.Errorf("failed to login to keycloak %s", keycloakURL)
	}
	return client, jwtToken, nil
}

// CreateProjectInOrg creates a Project in a given Org
func CreateProjectInOrg(ctx context.Context, orgName string, projectName string) error {
	config := ctrl.GetConfigOrDie()
	nexusClient, err := nexus_client.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("\nerror creating project (%s). Error: %w", projectName, err)
	}
	configNode := nexusClient.TenancyMultiTenancy().Config()
	if configNode == nil {
		return fmt.Errorf("\nerror creating project (%s). Error: %w", projectName, err)
	}

	org := configNode.Orgs(orgName)
	if org == nil {
		log.Info().Msgf("Org (%s) not found. Please create an org first\n", orgName)
		return nil
	}

	log.Info().Msgf("Creating Project (%s)\n", projectName)

	folder, err := org.AddFolders(ctx, &folderv1.Folder{
		ObjectMeta: metav1.ObjectMeta{
			Name: "default",
		},
		Spec: folderv1.FolderSpec{},
	})

	if err != nil && !nexus_client.IsAlreadyExists(err) {
		return fmt.Errorf("\nerror creating project (%s). Error: %w", projectName, err)
	}

	_, err = folder.AddProjects(ctx, &projectv1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name: projectName,
		},
		Spec: projectv1.ProjectSpec{
			Description: projectName,
		},
	})
	if err != nil && !nexus_client.IsAlreadyExists(err) {
		return fmt.Errorf("\nerror creating project (%s). Error: %w", projectName, err)
	}

	// wait until keycloak roles
	nexusClient.SubscribeAll()
	projectUUID, err := waitUntilProjectCreation(ctx, nexusClient, orgName, projectName)
	if err != nil {
		return fmt.Errorf("wait for project %s to go active failed with error %w", projectName, err)
	}
	log.Info().Msgf("\nProject (%s) has UID: %s\n", projectName, projectUUID)
	return nil
}

func WaitUntilProjectReady(ctx context.Context, orgName, projectName string) (string, error) {
	config := ctrl.GetConfigOrDie()
	nexusClient, err := nexus_client.NewForConfig(config)
	if err != nil {
		return "", fmt.Errorf("\nerror creating nexus client (%s). Error: %w", projectName, err)
	}

	// SubscribeAll otherwise GetFolders() and GetProjects may fail after client is restarted
	// Note that it may take a few moments for the subscription to populate the in-memory cache.
	nexusClient.SubscribeAll()

	return waitUntilProjectCreation(ctx, nexusClient, orgName, projectName)
}

func waitUntilProjectCreation(ctx context.Context, nexusClient *nexus_client.Clientset, orgName, projectName string) (string, error) {
	println("\nwaiting until project creation is completed")
	configNode := nexusClient.TenancyMultiTenancy().Config()
	orgNode, err := configNode.GetOrgs(ctx, orgName)
	if err != nil {
		return "", err
	}
	timeout := time.After(5 * time.Minute)
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			folder, err := orgNode.GetFolders(ctx, "default")
			if nexus_client.IsNotFound(err) {
				// not found most probably happened because nexus client cache is not loaded yet.
				continue
			}
			if err != nil {
				return "", err
			}

			project, err := folder.GetProjects(ctx, projectName)
			if nexus_client.IsNotFound(err) {
				// not found most probably happened because nexus client cache is not loaded yet.
				continue
			}
			if err != nil {
				return "", err
			}
			log.Info().Msgf("project %v status - %v (%s)\n", projectName, project.Status.ProjectStatus.StatusIndicator, project.Status.ProjectStatus.Message)
			if project.Status.ProjectStatus.StatusIndicator == projectv1.StatusIndicationIdle {
				return project.Status.ProjectStatus.UID, nil
			}
		case <-timeout:
			return "", fmt.Errorf("project %s creation timed out", projectName)
		}
	}
}

func WaitUntilProjectWatchersReady(ctx context.Context, orgName, projectName string) (string, error) {
	config := ctrl.GetConfigOrDie()
	nexusClient, err := nexus_client.NewForConfig(config)
	if err != nil {
		return "", fmt.Errorf("\nerror creating nexus client (%s). Error: %w", projectName, err)
	}

	return waitUntilProjectActiveWatchersCreated(ctx, nexusClient, orgName, projectName)
}

func waitUntilProjectActiveWatchersCreated(ctx context.Context, nexusClient *nexus_client.Clientset,
	orgName, projectName string,
) (string, error) {
	configNode, err := nexusClient.TenancyMultiTenancy().GetConfig(ctx)
	if err != nil {
		return "", err
	}
	projectWatchers, err := configNode.GetAllProjectWatchers(ctx)
	if err != nil {
		return "", err
	}
	var watchersList []string
	for _, watcher := range projectWatchers {
		watchersList = append(watchersList, watcher.GetLabels()["nexus/display_name"])
	}
	runtimeNode := nexusClient.TenancyMultiTenancy().Runtime()
	rtorgNode, err := runtimeNode.GetOrgs(ctx, orgName)
	if err != nil {
		return "", err
	}
	rtfolder, err := rtorgNode.GetFolders(ctx, "default")
	if err != nil {
		return "", err
	}
	rtproject, err := rtfolder.GetProjects(ctx, projectName)
	if err != nil {
		return "", err
	}
	timeout := time.After(5 * time.Minute)
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			activeWatchers, err := rtproject.GetAllActiveWatchers(ctx)
			if err != nil {
				return "", err
			}
			if len(activeWatchers) < len(projectWatchers) {
				log.Info().Msg("projectActiveWatchers count is lesser than projectWatchers. Waiting...")
				continue
			}
			var notReadyList []string
			for _, acw := range activeWatchers {
				if slices.Contains(watchersList, acw.GetLabels()["nexus/display_name"]) {
					if acw.Spec.StatusIndicator != "STATUS_INDICATION_IDLE" {
						notReadyList = append(notReadyList, acw.GetLabels()["nexus/display_name"])
					}
				}
			}
			log.Info().Msgf("Watchers [%v] are not yet set to STATUS_INDICATION_IDLE\n", notReadyList)
			if len(notReadyList) == 0 {
				log.Info().Msg("projectActiveWatchers created and in idle state")
				return "all watchers ready and in idle state", nil
			}
		case <-timeout:
			return "", fmt.Errorf("project active watchers %s creation timed out", projectName)
		}
	}
}

// GetProject gets the Project in the system
func GetProject(ctx context.Context, orgName, projectName string) error {
	config := ctrl.GetConfigOrDie()
	nexusClient, err := nexus_client.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("\nerror retrieving the project (%s). Error: %w", projectName, err)
	}
	configNode := nexusClient.TenancyMultiTenancy().Config()
	if configNode == nil {
		return fmt.Errorf("\nerror retrieving the project (%s). Error: %w", projectName, err)
	}

	org := configNode.Orgs(orgName)
	if org == nil {
		log.Info().Msgf("org %s does not exist.\n", orgName)
		return nil
	}

	folder := org.Folders("default")
	if folder == nil {
		return fmt.Errorf("\nerror retrieving the project (%s). Error: %w", projectName, err)
	}

	project := folder.Projects(projectName)
	projectStatus, err := project.GetProjectStatus(ctx)
	if projectStatus == nil || err != nil {
		return fmt.Errorf("\nerror retrieving the project (%s). Error: %w", projectName, err)
	}
	if strings.Contains(projectStatus.Message, "Waiting for watchers") {
		return fmt.Errorf("\nproject (%s) status message is set to %s", projectName, projectStatus.Message)
	}
	log.Info().Msgf("\nproject status message - %s, project status - %s\n", projectStatus.Message, projectStatus.StatusIndicator)
	println("projectId: ", projectStatus.UID)
	return nil
}

func GetProjectId(ctx context.Context, orgName, projectName string) (string, error) {
	config := ctrl.GetConfigOrDie()
	nexusClient, err := nexus_client.NewForConfig(config)
	if err != nil {
		return "", fmt.Errorf("\nerror retrieving the project (%s). Error: %w", projectName, err)
	}
	configNode := nexusClient.TenancyMultiTenancy().Config()
	if configNode == nil {
		return "", fmt.Errorf("\nerror retrieving the project (%s). Error: %w", projectName, err)
	}

	org := configNode.Orgs(orgName)
	if org == nil {
		log.Info().Msgf("org %s does not exist.\n", orgName)
		return "", nil
	}

	folder := org.Folders("default")
	if folder == nil {
		return "", fmt.Errorf("\nerror retrieving the project (%s). Error: %w", projectName, err)
	}

	project := folder.Projects(projectName)
	projectStatus, err := project.GetProjectStatus(ctx)
	if projectStatus == nil || err != nil {
		return "", fmt.Errorf("\nerror retrieving the project (%s). Error: %w", projectName, err)
	}

	return projectStatus.UID, nil
}

// CreateEdgeInfraUsers creates Edge Infra Manager users in a given Project
func CreateEdgeInfraUsers(ctx context.Context, orgName, projectName, edgeInfraUserPrefix string) error {
	// TODO - Add a check to determine if ktc is running

	client, token, err := KeycloakLogin(ctx)
	if err != nil {
		return err
	}

	projectId, err := GetProjectId(ctx, orgName, projectName)
	if projectId == "" || err != nil {
		return fmt.Errorf("error retrieving project %s. Error: %w", projectName, err)
	}

	// Create Edge Infra Manager NB API user
	user := edgeInfraUserPrefix + "-api-user"
	userId, orgId, err := createKeycloakUser(ctx, client, token, user, orgName)
	if err != nil && status.Code(err) != codes.AlreadyExists {
		return fmt.Errorf("error creating Keycloak user %s. Error: %w", user, err)
	}
	if status.Code(err) != codes.AlreadyExists {
		groups := []string{projectId + "_Host-Manager-Group"}

		err = addUserToGroups(ctx, client, token, KeycloakRealm, groups, userId)
		if err != nil {
			return fmt.Errorf("error adding org roles to user %s. Error: %w", user, err)
		}
		err = addProjectMemberRole(ctx, client, token, KeycloakRealm, orgId, projectId, userId)
		if err != nil {
			return fmt.Errorf("error adding member role to user %s. Error: %w", user, err)
		}
	}

	// Create EN agent user
	user = edgeInfraUserPrefix + "-en-svc-account"
	userId, _, err = createKeycloakUser(ctx, client, token, user, orgName)
	if err != nil && status.Code(err) != codes.AlreadyExists {
		return fmt.Errorf("error creating Keycloak user %s. Error: %w", user, err)
	}
	if status.Code(err) != codes.AlreadyExists {
		groups := []string{projectId + "_Edge-Node-M2M-Service-Account"}

		err = addUserToGroups(ctx, client, token, KeycloakRealm, groups, userId)
		if err != nil {
			return fmt.Errorf("error adding org roles to user %s. Error: %w", user, err)
		}
		err = addProjectMemberRole(ctx, client, token, KeycloakRealm, orgId, projectId, userId)
		if err != nil {
			return fmt.Errorf("error adding member role to user %s. Error: %w", user, err)
		}
	}

	// Create EN onboarding user
	user = edgeInfraUserPrefix + "-onboarding-user"
	userId, _, err = createKeycloakUser(ctx, client, token, user, orgName)
	if err != nil && status.Code(err) != codes.AlreadyExists {
		return fmt.Errorf("error creating Keycloak user %s. Error: %w", user, err)
	}
	if status.Code(err) != codes.AlreadyExists {
		groups := []string{projectId + "_Edge-Onboarding-Group"}

		err = addUserToGroups(ctx, client, token, KeycloakRealm, groups, userId)
		if err != nil {
			return fmt.Errorf("error adding org roles to user %s. Error: %w", user, err)
		}
		err = addProjectMemberRole(ctx, client, token, KeycloakRealm, orgId, projectId, userId)
		if err != nil {
			return fmt.Errorf("error adding member role to user %s. Error: %w", user, err)
		}
	}

	// Create Edge Infra Manager NB API user with service-admin which is needed for observability-admin access
	user = edgeInfraUserPrefix + "-service-admin-api-user"
	userId, orgId, err = createKeycloakUser(ctx, client, token, user, orgName)
	if err != nil && status.Code(err) != codes.AlreadyExists {
		return fmt.Errorf("error creating Keycloak user %s. Error: %w", user, err)
	}
	if status.Code(err) != codes.AlreadyExists {
		groups := []string{projectId + "_Host-Manager-Group", "service-admin-group"}

		err = addUserToGroups(ctx, client, token, KeycloakRealm, groups, userId)
		if err != nil {
			return fmt.Errorf("error adding org roles to user %s. Error: %w", user, err)
		}
		err = addProjectMemberRole(ctx, client, token, KeycloakRealm, orgId, projectId, userId)
		if err != nil {
			return fmt.Errorf("error adding member role to user %s. Error: %w", user, err)
		}
	}

	return nil
}

// CreateClusterOrchUsers creates Cluster Orch users in a given Project
func CreateClusterOrchUsers(ctx context.Context, orgName, projectName, coUserPrefix string) error {
	// TODO - Add a check to determine if ktc is running

	client, token, err := KeycloakLogin(ctx)
	if err != nil {
		return err
	}

	projectId, err := GetProjectId(ctx, orgName, projectName)
	if projectId == "" || err != nil {
		return fmt.Errorf("error retrieving project %s. Error: %w", projectName, err)
	}

	// Create Edge operator user
	user := coUserPrefix + "-edge-op"
	userId, orgId, err := createKeycloakUser(ctx, client, token, user, orgName)
	if err != nil && status.Code(err) != codes.AlreadyExists {
		return fmt.Errorf("error creating Keycloak user %s. Error: %w", user, err)
	}
	if status.Code(err) != codes.AlreadyExists {
		groups := []string{projectId + "_Edge-Operator-Group"}

		err = addUserToGroups(ctx, client, token, KeycloakRealm, groups, userId)
		if err != nil {
			return fmt.Errorf("error adding org roles to user %s. Error: %w", user, err)
		}
		err = addProjectMemberRole(ctx, client, token, KeycloakRealm, orgId, projectId, userId)
		if err != nil {
			return fmt.Errorf("error adding member role to user %s. Error: %w", user, err)
		}
	}

	// Create Edge Manager user
	user = coUserPrefix + "-edge-mgr"
	userId, _, err = createKeycloakUser(ctx, client, token, user, orgName)
	if err != nil && status.Code(err) != codes.AlreadyExists {
		return fmt.Errorf("error creating Keycloak user %s. Error: %w", user, err)
	}
	if status.Code(err) != codes.AlreadyExists {
		groups := []string{projectId + "_Edge-Manager-Group"}

		err = addUserToGroups(ctx, client, token, KeycloakRealm, groups, userId)
		if err != nil {
			return fmt.Errorf("error adding org roles to user %s. Error: %w", user, err)
		}
		err = addProjectMemberRole(ctx, client, token, KeycloakRealm, orgId, projectId, userId)
		if err != nil {
			return fmt.Errorf("error adding member role to user %s. Error: %w", user, err)
		}
	}

	return nil
}

func CreateDefaultKeyCloakUser(ctx context.Context, client *gocloak.GoCloak, token *gocloak.JWT, user *gocloak.User, orgName string) (string, error) {
	userName := strings.ToLower(*user.Username)
	params := gocloak.GetUsersParams{
		Username: &userName,
	}

	users, err := client.GetUsers(ctx, token.AccessToken, KeycloakRealm, params)
	if err != nil {
		log.Error().Msgf("error getting user %s: %v", userName, err)
		return "", err
	}
	for _, user = range users {
		if *user.Username == userName {
			return "", status.Errorf(codes.AlreadyExists, "user %s already found in realm %s", userName, KeycloakRealm)
		}
	}

	user = &gocloak.User{
		Username:      &userName,
		Email:         gocloak.StringP(userName + "@" + orgName + ".com"),
		Enabled:       gocloak.BoolP(true),
		EmailVerified: gocloak.BoolP(true),
		FirstName:     user.FirstName,
		LastName:      user.LastName,
	}

	userId, err := client.CreateUser(ctx, token.AccessToken, KeycloakRealm, *user)
	if err != nil {
		log.Error().Msgf("error creating user %s", userName)
		return "", err
	}

	defaultOrchPass, err := GetDefaultOrchPassword()
	if err != nil {
		log.Error().Msgf("error getting default orch password %v", err)
		return "", err
	}

	err = client.SetPassword(ctx, token.AccessToken, userId, KeycloakRealm, defaultOrchPass, false)
	if err != nil {
		log.Error().Msgf("error setting password for user %s", userName)
		return "", err
	}

	log.Info().Msgf("created user %s\n", userName)
	return userId, nil
}

func createKeycloakUser(ctx context.Context, client *gocloak.GoCloak, token *gocloak.JWT, edgeInfraUser string, orgName string) (string, string, error) {
	var user *gocloak.User
	edgeInfraUser = strings.ToLower(edgeInfraUser)
	params := gocloak.GetUsersParams{
		Username: &edgeInfraUser,
	}

	users, err := client.GetUsers(ctx, token.AccessToken, KeycloakRealm, params)
	if err != nil {
		log.Error().Msgf("error getting user %s: %v", edgeInfraUser, err)
		return "", "", err
	}
	for _, user = range users {
		if *user.Username == edgeInfraUser {
			return "", "", status.Errorf(codes.AlreadyExists, "user %s already found in realm %s", edgeInfraUser, KeycloakRealm)
		}
	}

	orgId, _ := getOrgId(ctx, orgName)
	if orgId == "" {
		return "", "", err
	}

	user = &gocloak.User{
		Username:      &edgeInfraUser,
		Email:         gocloak.StringP(edgeInfraUser + "@" + orgName + ".com"),
		FirstName:     &edgeInfraUser,
		LastName:      &edgeInfraUser,
		Enabled:       gocloak.BoolP(true),
		EmailVerified: gocloak.BoolP(true),
	}

	userId, err := client.CreateUser(ctx, token.AccessToken, KeycloakRealm, *user)
	if err != nil {
		log.Error().Msgf("error creating user %s", edgeInfraUser)
		return "", "", err
	}

	defaultOrchPass, err := GetDefaultOrchPassword()
	if err != nil {
		log.Error().Msgf("error getting default orch password %v", err)
		return "", "", err
	}

	err = client.SetPassword(ctx, token.AccessToken, userId, KeycloakRealm, defaultOrchPass, false)
	if err != nil {
		log.Error().Msgf("error setting password for user %s", edgeInfraUser)
		return "", "", err
	}

	log.Info().Msgf("created user %s\n", edgeInfraUser)
	return userId, orgId, nil
}

func addProjectMemberRole(ctx context.Context, client *gocloak.GoCloak, token *gocloak.JWT, realm, orgId, projectId, userId string) error {
	var roles []gocloak.Role
	prefixedRole := fmt.Sprintf("%s_%s_%s", orgId, projectId, "m")
	role, err := getRealmRole(ctx, client, token, realm, prefixedRole)
	if err != nil {
		return err
	}
	roles = append(roles, *role)

	err = client.AddRealmRoleToUser(ctx, token.AccessToken, realm, userId, roles)
	if err != nil {
		return err
	}
	log.Info().Msgf("added member roles to the user %s\n", userId)
	return nil
}

func getRealmRole(ctx context.Context, client *gocloak.GoCloak, token *gocloak.JWT, realm string, role string) (*gocloak.Role, error) {
	realmRole, err := client.GetRealmRole(ctx, token.AccessToken, realm, role)
	if err != nil {
		return nil, err
	}
	return realmRole, nil
}
