// SPDX-FileCopyrightText: 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"context"
	"fmt"

	"slices"
	"strings"
	"time"

	"github.com/Nerzal/gocloak/v13"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"crypto/rand"
	"math/big"

	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
	folderv1 "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/apis/folder.edge-orchestrator.intel.com/v1"
	orgsv1 "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/apis/org.edge-orchestrator.intel.com/v1"
	projectv1 "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/apis/project.edge-orchestrator.intel.com/v1"
	nexus_client "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/nexus-client"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	KeycloakRealm = "master"
	adminUser     = "admin"
	tenantAdmin   = "tenant-admin"
	charset       = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789!@#$%^&*()_+|:<>?="
)

var (
	log = logging.GetLogger("tenant-init-job")
)

// CreateSingleTenant creates one Org, one Project and one Project admin user for the Org.
func CreateSingleTenant(ctx context.Context, orgName, projectName string) error {
	log.Info().Msg("Creating organization...")
	err := CreateOrg(ctx, orgName)
	if err != nil {
		return fmt.Errorf("failed to create org: %w", err)
	}
	log.Info().Msg("Organization created successfully.")

	log.Info().Msgf("Creating project admin user '%s' in organization '%s'...\n", tenantAdmin, orgName)
	client, token, err := KeycloakLogin(ctx)
	if err != nil {
		return err
	}
	tenantAdminUserId, orgId, err := createKeycloakUser(ctx, client, token, tenantAdmin, orgName)
	if err != nil {
		return err
	}
	log.Info().Msg("Project admin user created successfully.")

	log.Info().Msgf("Assigning organization roles to project admin user '%s'...\n", tenantAdmin)
	err = AddProjectAdminUserToOrg(ctx, client, token, orgId, tenantAdminUserId)
	if err != nil {
		return fmt.Errorf("failed to assign project admin roles for org: %w", err)
	}
	log.Info().Msg("Assigned organization roles to project admin user successfully")

	log.Info().Msgf("Creating project '%s' in organization '%s'...\n", projectName, orgName)
	err = CreateProjectInOrg(ctx, orgName, projectName)
	if err != nil {
		return fmt.Errorf("failed to create project in org: %w", err)
	}
	log.Info().Msg("Project created successfully.")

	log.Info().Msgf("Checking Project active watchers for project '%s'...\n", projectName)

	_, err = WaitUntilProjectWatchersReady(ctx, orgName, projectName)
	if err != nil {
		return fmt.Errorf("failed to wait for project active watchers to be ready: %w", err)
	}
	log.Info().Msg("Project active watchers are ready.")

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

// AddProjectAdminUserToOrg assigns Org level access to a Project Admin user
func AddProjectAdminUserToOrg(ctx context.Context, client *gocloak.GoCloak, token *gocloak.JWT, orgId string, projectAdminUserId string) error {
	groups := []string{orgId + "_Project-Manager-Group"}

	err := addUserToGroups(ctx, client, token, KeycloakRealm, groups, projectAdminUserId)
	if err != nil {
		log.Error().Msgf("error adding org roles to user with id %s", projectAdminUserId)
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

func GetKeycloakSecret() (string, error) {
	config := ctrl.GetConfigOrDie()
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return "", fmt.Errorf("error creating kubernetes client: %w", err)
	}

	secret, err := clientset.CoreV1().Secrets("orch-platform").Get(context.TODO(), "platform-keycloak", metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("error getting secret platform-keycloak: %w", err)
	}

	encodedPass, exists := secret.Data["admin-password"]
	if !exists {
		return "", fmt.Errorf("admin-password key not found in secret")
	}

	// Decode the Base64 string
	adminPass := string(encodedPass)
	if adminPass == "" {
		return "", fmt.Errorf("password string empty")
	}

	return adminPass, nil
}

func KeycloakLogin(ctx context.Context) (*gocloak.GoCloak, *gocloak.JWT, error) {
	keycloakURL := "https://keycloak.orch-platform.svc.cluster.local"

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

func createKeycloakUser(ctx context.Context, client *gocloak.GoCloak, token *gocloak.JWT, tenantUser string, orgName string) (string, string, error) {
	var user *gocloak.User
	tenantUser = strings.ToLower(tenantUser)
	params := gocloak.GetUsersParams{
		Username: &tenantUser,
	}

	users, err := client.GetUsers(ctx, token.AccessToken, KeycloakRealm, params)
	if err != nil {
		log.Error().Msgf("error getting user %s: %v", tenantUser, err)
		return "", "", err
	}
	for _, user = range users {
		if *user.Username == tenantUser {
			return "", "", status.Errorf(codes.AlreadyExists, "user %s already found in realm %s", tenantUser, KeycloakRealm)
		}
	}

	orgId, _ := getOrgId(ctx, orgName)
	if orgId == "" {
		return "", "", err
	}

	user = &gocloak.User{
		Username:      &tenantUser,
		Email:         gocloak.StringP(tenantUser + "@" + orgName + ".com"),
		FirstName:     &tenantUser,
		LastName:      &tenantUser,
		Enabled:       gocloak.BoolP(true),
		EmailVerified: gocloak.BoolP(true),
	}

	userId, err := client.CreateUser(ctx, token.AccessToken, KeycloakRealm, *user)
	if err != nil {
		log.Error().Msgf("error creating user %s", tenantUser)
		return "", "", err
	}

	userPassword, err := generateRandomPassword(14)
	if err != nil {
		log.Error().Msgf("error generating password for user %s", tenantUser)
		return "", "", err
	}

	err = client.SetPassword(ctx, token.AccessToken, userId, KeycloakRealm, userPassword, false)
	if err != nil {
		log.Error().Msgf("error setting password for user %s", tenantUser)
		return "", "", err
	}

	log.Info().Msgf("created user %s\n", tenantUser)
	return userId, orgId, nil
}

func generateRandomPassword(length int) (string, error) {
	password := make([]byte, length)
	max := big.NewInt(int64(len(charset)))

	for i := 0; i < length; i++ {
		idx, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		password[i] = charset[idx.Int64()]
	}
	return string(password), nil
}
