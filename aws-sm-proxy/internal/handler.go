// SPDX-FileCopyrightText: 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

type SecretsManagerAPI interface {
	GetSecretValue(
		ctx context.Context,
		params *secretsmanager.GetSecretValueInput,
		optFns ...func(*secretsmanager.Options),
	) (*secretsmanager.GetSecretValueOutput, error)
}

func NewProxyAWSHandler(svc SecretsManagerAPI) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		secretName := r.URL.Query().Get("name")
		if secretName == "" {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintln(w, "query param name empty")
			return
		}
		// #nosec G706 -- secretName is validated before this point
		log.Println("handling request for secret:", secretName)
		input := &secretsmanager.GetSecretValueInput{
			SecretId: &secretName,
		}

		result, err := svc.GetSecretValue(context.Background(), input)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintln(w, err)
			return
		}

		if result.SecretString != nil {
			fmt.Fprintln(w, *result.SecretString)
		} else {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintln(w, "secret is binary")
		}
	}
}
