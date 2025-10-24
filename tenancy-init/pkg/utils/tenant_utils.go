// SPDX-FileCopyrightText: 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"context"
	"fmt"
	"os"

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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
)

var (
	log         = logging.GetLogger(AppName)
	keycloakURL = fmt.Sprintf("http://%s.%s.svc.cluster.local:%s", getEnv("KEYCLOAK_SERVICE", defaultKeycloakService),
		getEnv("KEYCLOAK_NAMESPACE", defaultKeycloakNamespace),
		getEnv("KEYCLOAK_PORT", defaultKeycloakPort))
	adminGroups = func() []string {
		projectAdminGroups := getEnv("PROJECT_ADMIN_GROUPS", defaultProjectAdminGroups)
		groups := strings.Split(projectAdminGroups, ",")
		for i, group := range groups {
			groups[i] = "_" + strings.TrimSpace(group)
		}
		return groups
	}()
	edgeGroups = func() []string {
		projectEdgeGroups := getEnv("PROJECT_EDGE_GROUPS", defaultProjectEdgeGroups)
		groups := strings.Split(projectEdgeGroups, ",")
		for i, group := range groups {
			groups[i] = "_" + strings.TrimSpace(group)
		}
		return groups
	}()
)

func getEnv(key, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultVal
}

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
	projectId, err := CreateProjectInOrg(ctx, orgName, projectName)
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

	err = AddProjectAdminUserToEdgeGroups(ctx, client, token, projectId, tenantAdminUserId)
	if err != nil {
		return fmt.Errorf("failed to assign project admin roles for org: %w", err)
	}
	log.Info().Msgf("Assigned Edge Manager, Onboarding, Operator and Host Manager roles to %s user successfully", tenantAdmin)

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
	groups := []string{}
	for _, group := range adminGroups {
		groups = append(groups, orgId+group)
	}

	err := addUserToGroups(ctx, client, token, KeycloakRealm, groups, projectAdminUserId)
	if err != nil {
		log.Error().Msgf("error adding org roles to user with id %s", projectAdminUserId)
		return err
	}

	return nil
}

// AddProjectAdminUserToEdgeGroups assigns Edge Manager, Edge Onboarding, Edge Operator and Host Manager Groups to a Project Admin user
func AddProjectAdminUserToEdgeGroups(ctx context.Context, client *gocloak.GoCloak, token *gocloak.JWT, projectId string, projectAdminUserId string) error {
	groups := []string{}
	for _, group := range edgeGroups {
		groups = append(groups, projectId+group)
	}

	err := addUserToGroups(ctx, client, token, KeycloakRealm, groups, projectAdminUserId)
	if err != nil {
		log.Error().Msgf("error adding Edge and Host roles to user with id %s", projectAdminUserId)
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
func CreateProjectInOrg(ctx context.Context, orgName string, projectName string) (string, error) {
	config := ctrl.GetConfigOrDie()
	nexusClient, err := nexus_client.NewForConfig(config)
	if err != nil {
		return "", fmt.Errorf("\nerror creating project (%s). Error: %w", projectName, err)
	}
	configNode := nexusClient.TenancyMultiTenancy().Config()
	if configNode == nil {
		return "", fmt.Errorf("\nerror creating project (%s). Error: %w", projectName, err)
	}

	org := configNode.Orgs(orgName)
	if org == nil {
		log.Info().Msgf("Org (%s) not found. Please create an org first\n", orgName)
		return "", fmt.Errorf("org not found")
	}

	log.Info().Msgf("Creating Project (%s)\n", projectName)

	folder, err := org.AddFolders(ctx, &folderv1.Folder{
		ObjectMeta: metav1.ObjectMeta{
			Name: "default",
		},
		Spec: folderv1.FolderSpec{},
	})

	if err != nil && !nexus_client.IsAlreadyExists(err) {
		return "", fmt.Errorf("\nerror creating project (%s). Error: %w", projectName, err)
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
		return "", fmt.Errorf("\nerror creating project (%s). Error: %w", projectName, err)
	}

	// wait until keycloak roles
	nexusClient.SubscribeAll()
	projectUUID, err := waitUntilProjectCreation(ctx, nexusClient, orgName, projectName)
	if err != nil {
		return "", fmt.Errorf("wait for project %s to go active failed with error %w", projectName, err)
	}
	log.Info().Msgf("\nProject (%s) has UID: %s\n", projectName, projectUUID)
	return projectUUID, nil
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
		// Add logic for already exists error
		if !strings.Contains(err.Error(), "409") {
			log.Error().Msgf("error creating user %s", tenantUser)
			return "", "", err
		}
		log.Info().Msgf("User %s already exists..skipping user creation", tenantUser)
	}

	userPassword, err := generatePassword()
	if err != nil {
		log.Error().Msgf("error generating password for user %s", tenantUser)
		return "", "", err
	}

	err = client.SetPassword(ctx, token.AccessToken, userId, KeycloakRealm, userPassword, false)
	if err != nil {
		log.Error().Msgf("error setting password for user %s", tenantUser)
		return "", "", err
	}

	// Create secret with user password
	err = createUserPasswordSecret(ctx, tenantUser, orgName, userPassword)
	if err != nil {
		log.Error().Msgf("error creating password secret for user %s: %v", tenantUser, err)
		return "", "", err
	}

	log.Info().Msgf("created user %s\n", tenantUser)
	return userId, orgId, nil
}

// createUserPasswordSecret creates a Kubernetes secret containing the user's password
func createUserPasswordSecret(ctx context.Context, username, orgName, password string) error {
	config := ctrl.GetConfigOrDie()
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("error creating kubernetes client: %w", err)
	}

	// Get current namespace where the pod is running
	namespace, err := getCurrentNamespace()
	if err != nil {
		return fmt.Errorf("error getting current namespace: %w", err)
	}

	secretName := fmt.Sprintf("%s-password", username)

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
			Labels: map[string]string{
				"app":      "tenant-init",
				"org":      orgName,
				"username": username,
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"admin-password": []byte(password),
		},
	}

	_, err = clientset.CoreV1().Secrets(namespace).Create(ctx, secret, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("error creating secret %s: %w", secretName, err)
	}

	log.Info().Msgf("created password secret %s in namespace %s\n", secretName, namespace)
	return nil
}

// getCurrentNamespace returns the namespace where the current pod is running
func getCurrentNamespace() (string, error) {
	// Try to read namespace from service account token (when running in pod)
	if data, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace"); err == nil {
		return strings.TrimSpace(string(data)), nil
	}

	// Fallback to orch-iam namespace if not running in pod
	return "orch-iam", nil
}

func secureRandomChar(charset string) (byte, error) {
	max := big.NewInt(int64(len(charset)))
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return 0, err
	}
	return charset[n.Int64()], nil
}

func shuffleBytes(data []byte) error {
	for i := len(data) - 1; i > 0; i-- {
		jBig, err := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		if err != nil {
			return err
		}
		j := int(jBig.Int64())
		data[i], data[j] = data[j], data[i]
	}
	return nil
}

func generatePassword() (string, error) {
	if passwordLength < 4 {
		return "", fmt.Errorf("password length must be at least 4 to fit character set requirements")
	}

	password := make([]byte, passwordLength)
	var err error

	// Guarantee one character from each category
	password[0], err = secureRandomChar(uppercaseChars)
	if err != nil {
		return "", err
	}
	password[1], err = secureRandomChar(lowercaseChars)
	if err != nil {
		return "", err
	}
	password[2], err = secureRandomChar(digitChars)
	if err != nil {
		return "", err
	}
	password[3], err = secureRandomChar(specialChars)
	if err != nil {
		return "", err
	}

	allChars := uppercaseChars + lowercaseChars + digitChars + specialChars
	for i := 4; i < passwordLength; i++ {
		password[i], err = secureRandomChar(allChars)
		if err != nil {
			return "", err
		}
	}

	err = shuffleBytes(password)
	if err != nil {
		return "", err
	}

	return string(password), nil
}
