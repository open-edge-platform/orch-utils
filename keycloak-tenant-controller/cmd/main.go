// SPDX-FileCopyrightText: 2026 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"flag"
	"math/rand/v2"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/open-edge-platform/orch-utils/keycloak-tenant-controller/pkg/auth/secrets"
	"github.com/open-edge-platform/orch-utils/keycloak-tenant-controller/pkg/common"
	"github.com/open-edge-platform/orch-utils/keycloak-tenant-controller/pkg/keycloak"
	"github.com/open-edge-platform/orch-utils/keycloak-tenant-controller/pkg/log"
	"github.com/open-edge-platform/orch-utils/keycloak-tenant-controller/pkg/tdmclient"
)

var (
	Commit         string
	logLevel       = flag.String("loglevel", "info", "Sets logging level.")
	ktcStressCalls = flag.Int("stressktc", 0, "KTC performs a stress test on startup. The number passed is the number of orgs and projects created.")
)

func main() {
	flag.Parse()
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	logConfig := log.Config{
		Level: *logLevel,
	}

	if err := log.Init(logConfig); err != nil {
		log.Errorf("Error initialising logging: %v", err)
		panic("Error initialising logging")
	}

	log.Infof("Commit: %s", Commit)

	if secretInitErr := secrets.Init(context.Background()); secretInitErr != nil {
		log.Errorf("Unable to initialize required secrets: %v", secretInitErr)
		panic("Error initialising required secrets")
	}

	kcClient := keycloak.NewClient()
	if err := kcClient.Init(); err != nil {
		log.Errorf("Error initialising Keycloak Client: %v", err)
		panic("Error initialising Keycloak Client")
	}

	if *ktcStressCalls > 0 {
		log.Infof("Starting KTC stress test")
		stressKtc(kcClient, *ktcStressCalls)
		log.Infof("KTC stress test complete. Application will continue as normal...")
	}

	client, err := tdmclient.NewMTClient(common.AppName, kcClient)
	if err != nil {
		log.Errorf("Error creating TDM Client: %v", err)
		panic("Error creating TDM Client")
	}
	if err := client.Init(); err != nil {
		log.Errorf("Error initialising TDM Client: %v", err)
		panic("Error initialising TDM Client")
	}

	<-done
	log.Infof("Shutting down")
	client.Stop()
}

func stressKtc(client keycloak.Client, numCreates int) {
	var waitGroup sync.WaitGroup

	randomSleep := func() {
		n := rand.IntN(10)
		time.Sleep(time.Duration(n) * time.Second)
	}

	for i := 0; i <= numCreates; i++ {
		orgId := uuid.New().String()
		projId := uuid.New().String()

		waitGroup.Add(1)
		go func(orgId string, projId string) {
			defer waitGroup.Done()

			randomSleep()
			if err := client.CreateOrg(orgId); err != nil {
				log.Errorf("Create org error: %v", err)
			}

			waitGroup.Add(1)
			go func() {
				defer waitGroup.Done()
				randomSleep()
				if err := client.DeleteOrg(orgId); err != nil {
					log.Errorf("Delete org error: %v", err)
				}
			}()

			randomSleep()
			if err := client.CreateProject(orgId, projId); err != nil {
				log.Errorf("Create proj error: %v", err)
			}

			waitGroup.Add(1)
			go func() {
				defer waitGroup.Done()
				randomSleep()
				if err := client.DeleteProject(projId); err != nil {
					log.Errorf("Delete proj error: %v", err)
				}
			}()

		}(orgId, projId)
	}

	waitGroup.Wait()
}
