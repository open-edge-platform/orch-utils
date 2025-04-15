// Copyright (C) 2025 Intel Corporation
// SPDX-FileCopyrightText: 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"flag"
	"os"

	"github.com/open-edge-platform/orch-utils/nexus-api-gateway/controllers"
	"github.com/open-edge-platform/orch-utils/nexus-api-gateway/pkg/cache"
	"github.com/open-edge-platform/orch-utils/nexus-api-gateway/pkg/client"
	"github.com/open-edge-platform/orch-utils/nexus-api-gateway/pkg/common"
	"github.com/open-edge-platform/orch-utils/nexus-api-gateway/pkg/config"
	"github.com/open-edge-platform/orch-utils/nexus-api-gateway/pkg/openapi/api"
	"github.com/open-edge-platform/orch-utils/nexus-api-gateway/pkg/server/echoserver"
	nexusClient "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/nexus-client"
	"github.com/rs/zerolog"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
	apiregistrationv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
)

var (
	scheme  = runtime.NewScheme()
	appName = "nexus-api-gw"
	log     = logging.GetLogger(appName)
)

func initializeSchemes() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	addSchemes()
}

func addSchemes() {
	addToScheme(apiextensionsv1.AddToScheme, "apiextensionsv1")
	addToScheme(apiregistrationv1.AddToScheme, "apiregistrationv1")
	addToScheme(corev1.AddToScheme, "corev1")
	//+kubebuilder:scaffold:scheme
}

func addToScheme(addFunc func(*runtime.Scheme) error, name string) {
	if err := addFunc(scheme); err != nil {
		log.Fatal().Msgf("Failed to add %s to scheme: %v", name, err)
	}
}

func main() {
	var metricsAddr, probeAddr string
	var enableLeaderElection bool

	parseFlags(&metricsAddr, &probeAddr, &enableLeaderElection)
	setupLogging()

	initializeSchemes()
	cache.InitializeCaches()

	k8sConfig, k8sClientSet := initializeK8sClient()
	nexusClientSet := initializeNexusClient(k8sConfig)
	tenancyNexusClientSet := initializeTenancyNexusClient(k8sConfig)

	loadConfigurations()

	stopCh := make(chan struct{})
	initializeEchoServer(stopCh, k8sClientSet, nexusClientSet, tenancyNexusClientSet)

	if config.Cfg.EnableNexusRuntime {
		overrideAddresses(&metricsAddr, &probeAddr)
		InitManager(metricsAddr, probeAddr, enableLeaderElection, stopCh)
	}

	// Block indefinitely to keep the main function running
	select {}
}

func parseFlags(metricsAddr, probeAddr *string, enableLeaderElection *bool) {
	flag.StringVar(metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(enableLeaderElection, "leader-elect", false, "Enable leader election for controller manager.")
	flag.Parse()
}

func setupLogging() {
	opts := zap.Options{Development: true}
	opts.BindFlags(flag.CommandLine)
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "error"
	}
	lvl, err := zerolog.ParseLevel(logLevel)
	if err != nil {
		log.Fatal().Msgf("Failed to configure logging: %v\n", err)
	}
	zerolog.SetGlobalLevel(lvl)
}

func initializeK8sClient() (*rest.Config, *kubernetes.Clientset) {
	k8sConfig := ctrl.GetConfigOrDie()
	k8sClientSet, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		log.Fatal().Msgf("Failed to create K8sclient: %v", err)
	}
	return k8sConfig, k8sClientSet
}

func initializeNexusClient(k8sConfig *rest.Config) *nexusClient.Clientset {
	nexusClientSet, err := nexusClient.NewForConfig(k8sConfig)
	if err != nil {
		log.Fatal().Msgf("Failed to create nexusclient: %v", err)
	}
	return nexusClientSet
}

func initializeTenancyNexusClient(k8sConfig *rest.Config) *nexusClient.Clientset {
	k8sConfig.Burst = 1000
	k8sConfig.QPS = 1000
	tenancyNexusClientSet, err := nexusClient.NewForConfig(k8sConfig)
	if err != nil {
		log.Fatal().Msgf("Failed to create tenancy nexusclient: %v", err)
	}
	tenancyNexusClientSet.SubscribeAll()
	return tenancyNexusClientSet
}

func loadConfigurations() {
	conf, err := config.LoadConfig("")
	if err != nil {
		log.Warn().Msgf("Error loading config: %v\n", err)
	}
	config.Cfg = conf

	common.Mode = os.Getenv("GATEWAY_MODE")
	log.Info().Msgf("Gateway Mode: %s", common.Mode)
}

func initializeEchoServer(stopCh chan struct{}, k8sClientSet *kubernetes.Clientset,
	nexusClientSet *nexusClient.Clientset, tenancyNexusClientSet *nexusClient.Clientset,
) {
	log.Info().Msg("Init Echo Server")
	echoserver.InitEcho(stopCh, config.Cfg, k8sClientSet, nexusClientSet, tenancyNexusClientSet)
}

func overrideAddresses(metricsAddr, probeAddr *string) {
	if config.Cfg.Server.HealthProbeAddrress != "" {
		*probeAddr = config.Cfg.Server.HealthProbeAddrress
	}

	if config.Cfg.Server.MetricsAddress != "" {
		*metricsAddr = config.Cfg.Server.MetricsAddress
	}
}

func InitManager(metricsAddr, probeAddr string, enableLeaderElection bool, stopCh chan struct{}) {
	if err := setupClients(); err != nil {
		log.Fatal().Msgf("unable to set up clients : %v", err)
		// os.Exit(1)
	}

	mgr, err := createManager(metricsAddr, probeAddr, enableLeaderElection)
	if err != nil {
		log.Fatal().Msgf("unable to start manager: %v", err)
		// os.Exit(1)
	}

	if err := addHealthChecks(mgr); err != nil {
		log.Fatal().Msgf("unable to set up health checks: %v", err)
		// os.Exit(1)
	}

	if err := setupControllers(mgr, stopCh); err != nil {
		log.Fatal().Msgf("unable to create controllers: %v", err)
		// os.Exit(1)
	}

	setupOpenAPI()

	startManager(mgr)
}

func createManager(metricsAddr, probeAddr string, enableLeaderElection bool) (ctrl.Manager, error) {
	return ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		Metrics:                metricsserver.Options{BindAddress: metricsAddr},
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "7b10c258.api-gw.com",
	})
}

func addHealthChecks(mgr ctrl.Manager) error {
	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		return err
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		return err
	}
	return nil
}

func setupControllers(mgr ctrl.Manager, stopCh chan struct{}) error {
	if err := (&controllers.CustomResourceDefinitionReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		StopCh: stopCh,
	}).SetupWithManager(mgr); err != nil {
		return err
	}

	if err := (&controllers.DatamodelReconciler{
		Client:  mgr.GetClient(),
		Scheme:  mgr.GetScheme(),
		Dynamic: client.Client,
	}).SetupWithManager(mgr); err != nil {
		return err
	}

	return nil
}

func setupClients() error {
	if err := client.New(ctrl.GetConfigOrDie()); err != nil {
		return err
	}

	if err := client.NewNexusClient(ctrl.GetConfigOrDie()); err != nil {
		return err
	}

	return nil
}

func setupOpenAPI() {
	// load combined oas file, override dm "edge-orchestrator.intel.com"
	api.LoadCombinedSpec()

	common.SSLEnabled = os.Getenv("SSL_ENABLED")
	log.Info().Msgf("SSL CertsEnabled: %s", common.SSLEnabled)
}

func startManager(mgr ctrl.Manager) {
	log.Info().Msgf("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		log.Fatal().Msgf("problem running manager: %v", err)
	}
}
