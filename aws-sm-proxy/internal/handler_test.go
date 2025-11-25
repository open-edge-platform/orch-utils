// SPDX-FileCopyrightText: 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package internal_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/stretchr/testify/mock"

	"github.com/open-edge-platform/orch-utils/aws-sm-proxy/internal"
)

type mockSMClient struct {
	mock.Mock
}

func (m *mockSMClient) GetSecretValue(
	_ context.Context,
	_ *secretsmanager.GetSecretValueInput,
	_ ...func(*secretsmanager.Options),
) (*secretsmanager.GetSecretValueOutput, error) {
	args := m.Called()
	return args.Get(0).(*secretsmanager.GetSecretValueOutput), args.Error(1)
}

var _ = Describe("AWS Secrets Manager", func() {
	var (
		client *mockSMClient
	)
	BeforeEach(func() {
		client = &mockSMClient{}
	})
	Context("Secrets manager", func() {
		It("should return the secret", func() {
			secretString := "mockSecret"
			client.On("GetSecretValue").Return(
				&secretsmanager.GetSecretValueOutput{
					SecretString: &secretString,
				}, nil)
			req, err := http.NewRequest("GET", "/aws-secret?name=mockName", nil)
			Expect(err).ToNot(HaveOccurred())
			rr := httptest.NewRecorder()
			handler := internal.NewProxyAWSHandler(client)

			handler(rr, req)

			Expect(rr.Result().StatusCode).To(Equal(http.StatusOK))
			Expect(rr.Body.String()).To(ContainSubstring("mockSecret"))
		})

		It("should return error when no secret name specified", func() {
			secretString := "mockSecret"
			client.On("GetSecretValue").Return(
				&secretsmanager.GetSecretValueOutput{
					SecretString: &secretString,
				}, nil)
			req, err := http.NewRequest("GET", "/aws-secret?xyz=bad-param", nil)
			Expect(err).ToNot(HaveOccurred())
			rr := httptest.NewRecorder()
			handler := internal.NewProxyAWSHandler(client)

			handler(rr, req)

			Expect(rr.Result().StatusCode).To(Equal(http.StatusBadRequest))
			Expect(rr.Body.String()).To(ContainSubstring("query param name empty"))
		})

		It("should handle secrets manager returning error", func() {
			client.On("GetSecretValue").Return(
				&secretsmanager.GetSecretValueOutput{}, fmt.Errorf("some error"))
			req, err := http.NewRequest("GET", "/aws-secret?name=mockName", nil)
			Expect(err).ToNot(HaveOccurred())
			rr := httptest.NewRecorder()
			handler := internal.NewProxyAWSHandler(client)

			handler(rr, req)

			Expect(rr.Result().StatusCode).To(Equal(http.StatusInternalServerError))
			Expect(rr.Body.String()).To(ContainSubstring("some error"))
		})

		It("should handle returning nil secret which implies binary secret", func() {
			client.On("GetSecretValue").Return(
				&secretsmanager.GetSecretValueOutput{
					SecretString: nil,
				}, nil)
			req, err := http.NewRequest("GET", "/aws-secret?name=mockName", nil)
			Expect(err).ToNot(HaveOccurred())
			rr := httptest.NewRecorder()
			handler := internal.NewProxyAWSHandler(client)

			handler(rr, req)

			Expect(rr.Result().StatusCode).To(Equal(http.StatusNotFound))
			Expect(rr.Body.String()).To(ContainSubstring("secret is binary"))
		})
	})
})
