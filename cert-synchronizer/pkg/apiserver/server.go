// SPDX-FileCopyrightText: 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package apiserver

import (
	"bytes"
	"context"
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/acm"
	"github.com/aws/aws-sdk-go-v2/service/acm/types"
	"github.com/aws/aws-sdk-go-v2/service/route53"

	// "github.com/labstack/gommon/log"

	// "github.com/labstack/gommon/log"

	route53types "github.com/aws/aws-sdk-go-v2/service/route53/types" //redeclared

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/google/uuid"
	ps "github.com/mitchellh/go-ps"
	"r53restapi.com/pkg/buildflags"
	"r53restapi.com/pkg/log"

	"flag"
	"path/filepath"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	DEFAULT_NAMESPACE           = "orch-gateway"
	DEFAULT_AUTOCERT_SECRETNAME = "kubernetes-docker-internal" // kubernetes-docker-internal
	DEFAULT_SECRETNAME          = "tls-orch"
	DEFAULT_TLS_CERT_PATH       = "/etc/ssl/cert/cert-man/tls.crt"
	DEFAULT_TLS_KEY_PATH        = "/etc/ssl/cert/cert-man/tls.key"
	DEFAULT_CA_CERT_PATH        = "/etc/ssl/cert/cert-man/ca.crt"

	DEBUG_TLS_CERT_PATH = "./certs/tls.crt"
	DEBUG_TLS_KEY_PATH  = "./certs/tls.key"
	DEBUG_CA_CERT_PATH  = "./certs/ca.crt"

	DEFAULT_SLEEP_TIMEOUT = 1000 * 120
	//DEFAULT_HTTP_TIMEOUT  = 10 * time.Second

	IMPORT_COMMENT = "Imported by Intel Open Edge Platform CertSynchronizer"

	//R10_URL  = "https://letsencrypt.org/certs/2024/r10.pem"
	//R11_URL  = "https://letsencrypt.org/certs/2024/r11.pem"
	//ROOT_URL = "https://letsencrypt.org/certs/isrgrootx1.pem"

	DEFAULT_INTER1_URL = "https://letsencrypt.org/certs/2024/r10.pem"
	DEFAULT_INTER2_URL = "https://letsencrypt.org/certs/2024/r11.pem"
	DEFAULT_ROOT_URL   = "https://letsencrypt.org/certs/isrgrootx1.pem"

	DEFAULT_INTER1_PEM = "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUZCVENDQXUyZ0F3SUJBZ0lRUzZoU2svZWFMNkp6Qmt1b0JJMTEwREFOQmdrcWhraUc5dzBCQVFzRkFEQlAKTVFzd0NRWURWUVFHRXdKVlV6RXBNQ2NHQTFVRUNoTWdTVzUwWlhKdVpYUWdVMlZqZFhKcGRIa2dVbVZ6WldGeQpZMmdnUjNKdmRYQXhGVEFUQmdOVkJBTVRERWxUVWtjZ1VtOXZkQ0JZTVRBZUZ3MHlOREF6TVRNd01EQXdNREJhCkZ3MHlOekF6TVRJeU16VTVOVGxhTURNeEN6QUpCZ05WQkFZVEFsVlRNUll3RkFZRFZRUUtFdzFNWlhRbmN5QkYKYm1OeWVYQjBNUXd3Q2dZRFZRUURFd05TTVRBd2dnRWlNQTBHQ1NxR1NJYjNEUUVCQVFVQUE0SUJEd0F3Z2dFSwpBb0lCQVFEUFYrWG14RlFTN2JSSC9za25XSFpHVUNpTUhUNkkzd1dkMWJVWUtiM2R0VnEvK3ZiT283NnZBQ0ZMCllscGFQQUV2eFZnRDlvbi9qaEZENjhHMTRCUUhsbzl2SDlmbnVvRTVDWFZsdDhLdkdGczNKaWpuby9RSEsyMGEKLzZ0WXZKV3VRUC9weTFmRXRWdC9lQTBZWWJ3WDUxVEd1MG1Selc0WTBZQ0Y3cVpsTnJ4MDZyeFFUT3I4SWZNNApGcE9VdXJEVGF6Z0d6UllTZXNwU2RjaXRkckxDbkYyWVJWeHZZWHZHTGU0OEUxS0dBZGxYNWpnYzM0MjFINUtSCm11ZEtITXhGcUhKVjhMRG1vd2ZzL2FjYlpwNC9TSXR4aEhGWXlUcjY3MTd5VzBRclBIVG5qN0pId1FkcXpacTMKRFpiM0VvRW1VVlFLN0dIMjkvWGk4b3JJbFEyTkFnTUJBQUdqZ2Znd2dmVXdEZ1lEVlIwUEFRSC9CQVFEQWdHRwpNQjBHQTFVZEpRUVdNQlFHQ0NzR0FRVUZCd01DQmdnckJnRUZCUWNEQVRBU0JnTlZIUk1CQWY4RUNEQUdBUUgvCkFnRUFNQjBHQTFVZERnUVdCQlM3dk1OSHBlUzhxY2JEcEhJTUVJMmlOZUhJNkRBZkJnTlZIU01FR0RBV2dCUjUKdEZubWU3Ymw1QUZ6Z0FpSXlCcFk5dW1iYmpBeUJnZ3JCZ0VGQlFjQkFRUW1NQ1F3SWdZSUt3WUJCUVVITUFLRwpGbWgwZEhBNkx5OTRNUzVwTG14bGJtTnlMbTl5Wnk4d0V3WURWUjBnQkF3d0NqQUlCZ1puZ1F3QkFnRXdKd1lEClZSMGZCQ0F3SGpBY29CcWdHSVlXYUhSMGNEb3ZMM2d4TG1NdWJHVnVZM0l1YjNKbkx6QU5CZ2txaGtpRzl3MEIKQVFzRkFBT0NBZ0VBa3JIblFUZnJlWjJCNXMzaUplRTZJT21RUkpXamdWelB3MTM5dmFCdzFiR1dLQ0lMMHZJbwp6d3puMU9aRGpDUWlIY0ZDa3RFSnI1OUw5TWh3VHlBV3NWcmRBZllmK0I5aGF4UW5zSEtOWTY3dTRzNUx6emZkCnU2UFV6ZWV0VUsyOXYrUHNQbUkyY0preHAraU4zZXBpNGhLdTlaelVQU3dNcXRDY2ViN3FQVnhFYnBZeFkxcDkKMW41UEpLQkxCWDllYjlMVTZsOHpTeFBXVjdiSzNsRzRYYU1KZ25UOXgzaWVzN21zRnRwS0s1YkR0b3Rpai9sMApHYUtlQTk3cGI1dXdEOUtnV3ZhRlhNSUV0OGpWVGpMRXZ3UmR2Q24yOTRHUERGMDhVOGxBa0l2N3RnaGx1YVFoCjFRbmxFNFNFTjRMT0VDajhkc0lHSlhwR1VrM2FVM0trSno5aWNLeSthVWdBKzJjUDIxdWg2TmNESVMzWHlmYVoKUWptRFE5OTNDaElJOFNYV3VwUVpWQmlJcGNXTzRScVprM2xyN0J6NU1VQ3d6RElBMzU5ZTU3U1NxNUNDa1kwTgo0QjZWdWxrN0xrdGZ3cmRHTlZJNUJzQzlxcXhTd1NLZ1JKZVo5d3lnSWFlaGJIRkhGaGNCYU1ES3BpWmxCSHl6CnJzbm5sRlhDYjVzOEhLbjVMc1VnR3ZCMjRMN3NHTlpQMkNYN2RoSG92K1loRCtqb3pMVzJwOVc0OTU5QnoyRWkKUm1xRHRtaVhMbnpxVHBYYkkrc3V5Q3NvaEtSZzZVbjBSQzQ3K2NwaVZ3SGlYWkFXK2NuOGVpTklqcWJWZ1hMeApLUHBkenZ2dFRuT1BsQzdTUVpTWW1kdW5yM0JmOWI3N0FpQy9aaWRzdEszNmRSSUxLejdPQTU0PQotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCg=="
	DEFAULT_INTER2_PEM = "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUZCakNDQXU2Z0F3SUJBZ0lSQUlwOVBoUFdMekR2STRhOUtRZHJOUGd3RFFZSktvWklodmNOQVFFTEJRQXcKVHpFTE1Ba0dBMVVFQmhNQ1ZWTXhLVEFuQmdOVkJBb1RJRWx1ZEdWeWJtVjBJRk5sWTNWeWFYUjVJRkpsYzJWaApjbU5vSUVkeWIzVndNUlV3RXdZRFZRUURFd3hKVTFKSElGSnZiM1FnV0RFd0hoY05NalF3TXpFek1EQXdNREF3CldoY05NamN3TXpFeU1qTTFPVFU1V2pBek1Rc3dDUVlEVlFRR0V3SlZVekVXTUJRR0ExVUVDaE1OVEdWMEozTWcKUlc1amNubHdkREVNTUFvR0ExVUVBeE1EVWpFeE1JSUJJakFOQmdrcWhraUc5dzBCQVFFRkFBT0NBUThBTUlJQgpDZ0tDQVFFQXVvZThYQnNBT2N2S0NzM1VaeEQ1QVR5bFRxVmh5eWJLVXZzVkFiZTVLUFVvSHUwbnN5UVlPV2NKCkRBanM0RHF3TzNjT3ZmUGxPVlJCREU2dVFkYVpkTjVSMis5Ny8xaTlxTGNUOXQ0eDFmSnl5WEpxQzROMGxaeEcKQUdRVW1mT3gyU0xaemFpU3Fod21lai8rNzFnRmV3aVZnZHR4RDQ3NzR6RUp1d20rVUUxZmo1RjJQVnFkbm9QeQo2Y1JtcytFR1prTklHSUJsb0RjWW1wdUVNcGV4c3IzRStCVUFuU2VJKytKakY1WnNteWRuUzhUYktGNXB3bm53ClNWemdKRkRoeEx5aEJheDdRRzBBdE1KQlA2ZFl1Qy9GWEp1bHV3bWU4Zjdyc0lVNS9hZ0s3MFhFZU90bEtzTFAKWHp6ZTQxeE5HL2NMSnl1cUMwSjNVMDk1YWgySDJRSURBUUFCbzRINE1JSDFNQTRHQTFVZER3RUIvd1FFQXdJQgpoakFkQmdOVkhTVUVGakFVQmdnckJnRUZCUWNEQWdZSUt3WUJCUVVIQXdFd0VnWURWUjBUQVFIL0JBZ3dCZ0VCCi93SUJBREFkQmdOVkhRNEVGZ1FVeGM5R3BPcjB3OEI2YkpYRUxiQmVraThtNDdrd0h3WURWUjBqQkJnd0ZvQVUKZWJSWjVudTI1ZVFCYzRBSWlNZ2FXUGJwbTI0d01nWUlLd1lCQlFVSEFRRUVKakFrTUNJR0NDc0dBUVVGQnpBQwpoaFpvZEhSd09pOHZlREV1YVM1c1pXNWpjaTV2Y21jdk1CTUdBMVVkSUFRTU1Bb3dDQVlHWjRFTUFRSUJNQ2NHCkExVWRId1FnTUI0d0hLQWFvQmlHRm1oMGRIQTZMeTk0TVM1akxteGxibU55TG05eVp5OHdEUVlKS29aSWh2Y04KQVFFTEJRQURnZ0lCQUU3aWlWMEtBeHlRT05EMUgvbHhYUGpEajdJM2lIcHZzQ1VmN2I2MzJJWUdqdWtKaE0xeQp2NEh6L01yUFUwanR2ZlpwUXRTbEVUNDF5Qk95a2gwRlgrb3UxTmo0U2NPdDlabVduTzhtMk9HMEpBdElJRTM4CjAxUzBxY1loeU9FMkcvOTNaQ2tYdWZCTDcxM3F6WG5RdjVDL3ZpT3lrTnBLcVVneGRLbEVDK0hpOWkyRGNhUjEKZTlLVXdRVVpSaHk1ai9QRWRFZ2xLZzNsOWR0RDR0dVRtN2tadEI4djMyb09qekhUWXcrN0tkemRaaXcvc0J0bgpVZmhCUE9STnVheTRwSnhtWS9XcmhTTWR6Rk8ycTNHdTNNVUJjZG8yN2dvWUtqTDlDVEY4ai9aejU1eWN0VW9WCmFuZUNXcy9halVYK0h5cGtCVEErYzhMR0RMbldPMk5LcTBZRC9wbkFSa0FuWUdQZlVEb0hSOWdWU3AvcVJ4K1oKV2doaURMWnNNd2hOMXpqdFNDMHVCV2l1Z0YzdlROellJRUZmYVBHN1dzM2pEckFNTVllYlE5NUpRK0hJQkQvUgpQQnVIUlRCcHFLbHlEbmtTSERIWVBpTlgzYWRQb1BBY2dkRjNIMi9XMHJtb3N3TVdnVGxMbjFXdTBtcmtzNy9xCnBkV2ZTNlBKMWp0eTgwcjJWS3NNL0RqM1lJRGZialhLZGFGVTVDKzhiaGZKR3FVM3RhS2F1dXowd0hWR1QzZW8KNkZsV2tXWXRidDRwZ2RhbWx3VmVaRVcrTE03cVpFSkVzTU5QcmZDMDNBUEttWnNKZ3BXQ0RXT0tadmtaY3ZqVgp1WWtRNG9tWUNUWDVvaHkra25NamRPbWRIOWM3U3BxRVdCREM4NmZpTmV4K08wWE9NRVpTYThEQQotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCg=="
	DEFAULT_ROOT_PEM   = "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUZhekNDQTFPZ0F3SUJBZ0lSQUlJUXo3RFNRT05aUkdQZ3UyT0Npd0F3RFFZSktvWklodmNOQVFFTEJRQXcKVHpFTE1Ba0dBMVVFQmhNQ1ZWTXhLVEFuQmdOVkJBb1RJRWx1ZEdWeWJtVjBJRk5sWTNWeWFYUjVJRkpsYzJWaApjbU5vSUVkeWIzVndNUlV3RXdZRFZRUURFd3hKVTFKSElGSnZiM1FnV0RFd0hoY05NVFV3TmpBME1URXdORE00CldoY05NelV3TmpBME1URXdORE00V2pCUE1Rc3dDUVlEVlFRR0V3SlZVekVwTUNjR0ExVUVDaE1nU1c1MFpYSnUKWlhRZ1UyVmpkWEpwZEhrZ1VtVnpaV0Z5WTJnZ1IzSnZkWEF4RlRBVEJnTlZCQU1UREVsVFVrY2dVbTl2ZENCWQpNVENDQWlJd0RRWUpLb1pJaHZjTkFRRUJCUUFEZ2dJUEFEQ0NBZ29DZ2dJQkFLM29KSFAwRkRmem01NHJWeWdjCmg3N2N0OTg0a0l4dVBPWlhvSGozZGNLaS92VnFidllBVHlqYjNtaUdiRVNUdHJGai9SUVNhNzhmMHVveG15RisKMFRNOHVrajEzWG5mczdqL0V2RWhta3ZCaW9aeGFVcG1abXlQZmp4d3Y2MHBJZ2J6NU1EbWdLN2lTNCszbVg2VQpBNS9UUjVkOG1VZ2pVK2c0cms4S2I0TXUwVWxYaklCMHR0b3YwRGlOZXdOd0lSdDE4akE4K28rdTNkcGpxK3NXClQ4S09FVXQrend2by83VjNMdlN5ZTByZ1RCSWxESENOQXltZzRWTWs3QlBaN2htL0VMTktqRCtKbzJGUjNxeUgKQjVUMFkzSHNMdUp2VzVpQjRZbGNOSGxzZHU4N2tHSjU1dHVrbWk4bXhkQVE0UTdlMlJDT0Z2dTM5NmozeCtVQwpCNWlQTmdpVjUrSTNsZzAyZFo3N0RuS3hIWnU4QS9sSkJkaUIzUVcwS3RaQjZhd0JkcFVLRDlqZjFiMFNIelV2CktCZHMwcGpCcUFsa2QyNUhON3JPckZsZWFKMS9jdGFKeFFaQktUNVpQdDBtOVNUSkVhZGFvMHhBSDBhaG1iV24KT2xGdWhqdWVmWEtuRWdWNFdlMCtVWGdWQ3dPUGpkQXZCYkkrZTBvY1MzTUZFdnpHNnVCUUUzeERrM1N6eW5UbgpqaDhCQ05BdzFGdHhOclFIdXNFd01GeEl0NEk3bUtaOVlJcWlveW1DekxxOWd3UWJvb01EUWFIV0JmRWJ3cmJ3CnFIeUdPMGFvU0NxSTNIYWFkcjhmYXFVOUdZL3JPUE5rM3NnckRRb28vL2ZiNGhWQzFDTFFKMTNoZWY0WTUzQ0kKclU3bTJZczZ4dDBuVVc3L3ZHVDFNME5QQWdNQkFBR2pRakJBTUE0R0ExVWREd0VCL3dRRUF3SUJCakFQQmdOVgpIUk1CQWY4RUJUQURBUUgvTUIwR0ExVWREZ1FXQkJSNXRGbm1lN2JsNUFGemdBaUl5QnBZOXVtYmJqQU5CZ2txCmhraUc5dzBCQVFzRkFBT0NBZ0VBVlI5WXFieXlxRkRRRExIWUdta2dKeWtJckdGMVhJcHUrSUxsYVMvVjlsWkwKdWJoekVGblRJWmQrNTB4eCs3TFNZSzA1cUF2cUZ5RldoZkZRRGxucnp1Qlo2YnJKRmUrR25ZK0VnUGJrNlpHUQozQmViWWh0RjhHYVYwbnh2d3VvNzd4L1B5OWF1Si9HcHNNaXUvWDErbXZvaUJPdi8yWC9xa1NzaXNSY09qL0tLCk5GdFkyUHdCeVZTNXVDYk1pb2d6aVV3dGhEeUMzKzZXVndXNkxMdjN4TGZIVGp1Q3ZqSElJbk56a3RIQ2dLUTUKT1JBekk0Sk1QSitHc2xXWUhiNHBob3dpbTU3aWF6dFhPb0p3VGR3Sng0bkxDZ2ROYk9oZGpzbnZ6cXZIdTdVcgpUa1hXU3RBbXpPVnl5Z2hxcFpYakZhSDNwTzNKTEYrbCsvK3NLQUl1dnRkN3UrTnhlNUFXMHdkZVJsTjhOd2RDCmpOUEVscHpWbWJVcTRKVWFnRWl1VERrSHpzeEhwRktWSzdxNCs2M1NNMU45NVIxTmJkV2hzY2RDYitaQUp6VmMKb3lpM0I0M25qVE9RNXlPZisxQ2NlV3hHMWJRVnM1WnVmcHNNbGpxNFVpMC8xbHZoK3dqQ2hQNGtxS09KMnF4cQo0Umdxc2FoRFlWdlRIOXc3alhieUxlaU5kZDhYTTJ3OVUvdDd5MEZmLzl5aTBHRTQ0WmE0ckYyTE45ZDExVFBBCm1SR3VuVUhCY25XRXZnSkJRbDluSkVpVTBac252Z2MvdWJoUGdYUlI0WHEzN1owajRyN2cxU2dFRXp3eEE1N2QKZW15UHhnY1l4bi9lUjQ0L0tKNEVCcytsVkRSM3ZleUptK2tYUTk5YjIxLytqaDVYb3MxQW5YNWlJdHJlR0NjPQotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0t"
)

// var env envVars
var stdOutLogging bool = true

// var log *logrus.Logger

const (
	ERROR_WEBPAGE  = "./html/error.html"
	HEADER_CONTENT = "content-type"
	HEADER_ACCEPT  = "accept"
	JSON_CONTENT   = "application/json"
	HTML_CONTENT   = "text/html"
	PLAIN_CONTENT  = "text/plain; charset=utf-8"
	MSG_200_OK     = "200 - OK."
	MSG_400_BAD_RQ = "400 - The validation request is malformed or missing required parameters."
	MSG_401_UNAUTH = "There was an error validating your license for this product. Check that your license has not expired. " +
		"Please open a support ticket including the following identifier at https://supporttickets.intel.com/supportrequest?lang=en-US&productId=236861:15711"
	MSG_403_FORBID = "403 - This resource is forbidden with your current license."
	MSG_405_METHOD = "405 - Method not allowed."
	MSG_418_TEAPOT = "418 - The server refuses the attempt to brew coffee with a teapot.!\n"
	MSG_503_UNAVAL = "503 - A backend error has occurred."
	PROC_LSERV     = "lserv64"
	PROC_SENTINEL  = "sntlcloudps64_i" //process names are truncated in /proc/<pid>/stat
)

var (
	server http.Server
)

type ServerConfig struct {
	VersionInfo string
	HttpPort    string
}

type JsonResponse struct {
	Message string `json:"message"`
}

type httpResponse struct {
	status          int
	acceptedContent string
	errWebpagePath  string
	errWebpageCmt   string
	errorUuid       string
	message         string
}

type certChain struct {
	tlsCrt []byte
	tlsKey []byte
	caCrt  []byte
	//inter1Crt   []byte
	//inter2Crt   []byte
	//rootCrt     []byte
	tlsCrtChain []byte
	caCrtChain  []byte
}

type fileTimes struct {
	tlsCrtModTime time.Time
	tlsKeyModTime time.Time
	caCrtModTime  time.Time
}

var certFileTimes fileTimes

type envVars struct {
	debugMode                 bool
	version                   string
	namespace                 string
	autoCertName              string
	tlsCrtPath                string
	tlsKeyPath                string
	caCrtPath                 string
	awsAccessKey              string
	awsSecretKey              string
	roleArn                   string
	region                    string
	vpc                       string
	domain                    string
	acmCertificateName        string
	certificateNameSpace      string
	inter1Cert                []byte
	inter2Cert                []byte
	rootCert                  []byte
	disableCertMatchChecks    bool
	importIntoACMIfNotExists  bool
	createK8sCertSecret       bool
	podFileUpdateSleepTimeout int
	k8sCertSecretName         string
	inter1CertUrl             string
	inter2CertUrl             string
	rootCertUrl               string
}

type CertEvent struct {
	Source    string   `json:"source"`
	Data      CertData `json:"data"`
	TimeStamp string   `json:"timeStamp"`
}
type CertData struct {
	APIVersion      string      `json:"APIVersion"`
	Action          string      `json:"Action"`
	Cluster         string      `json:"Cluster"`
	Count           int         `json:"Count"`
	Kind            string      `json:"Kind"`
	Level           string      `json:"Level"`
	Messages        []string    `json:"Messages"`
	Name            string      `json:"Name"`
	Namespace       string      `json:"Namespace"`
	Reason          string      `json:"Reason"`
	Recommendations interface{} `json:"Recommendations"`
	Resource        string      `json:"Resource"`
	TimeStamp       string      `json:"TimeStamp"`
	Title           string      `json:"Title"`
	Type            string      `json:"Type"`
	Warnings        interface{} `json:"Warnings"`
}

type DNSRecord struct {
	AWSRegion string `json:"region"`
	Domain    string `json:"domain"`
	Action    string `json:"action"`
	Type      string `json:"type"`
	Name      string `json:"name"`
	Value     string `json:"value"`
	TTL       int    `json:"ttl"`
	Policy    string `json:"policy"`
}

type DNSRecordParams struct {
	Region        string  `json:"region"` //Region = exported, region = not exported - stackoverflow.com/questions/28228393/json-unmarshal-returning-blank-structure
	Domain        string  `json:"domain"`
	VPC           string  `json:"vpc,omitempty"`
	IsPrivate     bool    `json:"isPrivate"`
	Recordtype    string  `json:"recordType"`
	Recordname    string  `json:"recordName"`
	Recordvalue   string  `json:"recordValue"`
	Policy        string  `json:"policy,omitempty"`
	SetIdentifier string  `json:"setIdentifier,omitempty"`
	Weight        int     `json:"weight,omitempty"`
	Failover      string  `json:"failover,omitempty"`
	CountryCode   string  `json:"countryCode,omitempty"`
	CIDR          string  `json:"cidr,omitempty"`
	Latitude      float64 `json:"latitude,omitempty"`
	Longitude     float64 `json:"longitude,omitempty"`
	fqdn          string
	escapequotes  bool
}

type debugParams struct {
	replyWith string
	delay     string
}

var env envVars

//var certs certChain

func (sc ServerConfig) validate() error {
	return validation.ValidateStruct(&sc,
		validation.Field(&sc.HttpPort, validation.Required, is.Port),
	)
}

func (dp debugParams) validate() error {
	var delay int
	var err error

	if err = validation.ValidateStruct(&dp,
		validation.Field(&dp.replyWith, validation.Required, validation.Length(3, 3), is.Int),
		validation.Field(&dp.delay, is.Int)); err != nil {
		return err
	}

	if dp.delay != "" {
		delay, err = strconv.Atoi(dp.delay)
		if err != nil {
			return err
		}

		if err = validation.Validate(&delay, validation.Min(0).Error("delay: must be 0 or greater"), validation.Max(120).Error("delay: must be 120 or less")); err != nil {
			return err
		}
	}
	return nil
}

/*
Init initialises the license server. Validates configurations. Initialises the license
cache. Initialises the function handlers. Sets global variables within the package and
creates the http server.
*/
func Init(cfg ServerConfig) error {

	flag.BoolVar(
		&stdOutLogging,
		"s",
		true,
		"Set logging format to stdOut only",
	)

	flag.Parse()
	getEnvVars()
	checkCertFiles(true, true)
	if env.debugMode {
		log.Infof("Debug mode=true")
		// log.SetLevel(logrus.DebugLevel)
	}

	if err := cfg.validate(); err != nil {
		log.Errorf("HTTP server config validation error: %v", err)
		return err
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/updatecert", UpdateCert)
	mux.HandleFunc("/updatecert/", UpdateCert)

	mux.HandleFunc("/forceupdatecert", UpdateCertWithoutChecks)
	mux.HandleFunc("/forceupdatecert/", UpdateCertWithoutChecks)

	mux.HandleFunc("/deletecert", DeleteCert)
	mux.HandleFunc("/deletecert/", DeleteCert)

	mux.HandleFunc("/forcedeletecert", DeleteCertWithoutChecks)
	mux.HandleFunc("/forcedeletecert/", DeleteCertWithoutChecks)

	mux.HandleFunc("/creatednsrecord", HTTPCreateDNSRecord)
	mux.HandleFunc("/creatednsrecord/", HTTPCreateDNSRecord)

	mux.HandleFunc("/deletednsrecord", HTTPDeleteDNSRecord)
	mux.HandleFunc("/deletednsrecord/", HTTPDeleteDNSRecord)

	mux.HandleFunc("/healthcheck", HealthCheckServer)
	mux.HandleFunc("/healthcheck/", HealthCheckServer)

	mux.HandleFunc("/debug", POSTDebug)
	mux.HandleFunc("/debug/", POSTDebug)

	if buildflags.DEBUG {
		mux.HandleFunc("/debugv1", GetDebug)
		mux.HandleFunc("/debugv1/", GetDebug)
	}

	server = http.Server{
		Addr:    ":" + cfg.HttpPort,
		Handler: mux,
	}

	env.version = cfg.VersionInfo

	return nil
}

/*
Run starts the server that was created by Init() and waits for the server to stop.
The server will stop if the application receives a SIGTERM or if the server encounters
an error. In any case. after the server stops this function will call shutdownServer(),
gracefully shutting down the http server and will also call licensecache.Cleanup(),
releasing all licenses in the cache.
*/
func Run() error {

	doInititalCertUpdate() //Assume we need to an initial cert upload after cert-manager and botkube have started.

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	var err error

	if server.Addr == "" {
		return fmt.Errorf("HTTP server was not initialised")
	}

	defer shutdownServer()

	go func() {
		if err = server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			close(done)
		}
	}()

	log.Infof("HTTP server started")
	<-done

	if err != nil && err != http.ErrServerClosed {
		log.Errorf("HTTP server error: %v", err)
		return err
	}

	return nil
}

func checkCertFiles(checkCA bool, getModTime bool) bool {
	certFileExists := false
	privateKeyFileExists := false
	caCertFileExists := false

	log.Infof("Certificate File path: %s\n", env.tlsCrtPath)
	if env.tlsCrtPath != "" {
		certFileExists = doesFileExist(env.tlsCrtPath, false)

		if certFileExists && getModTime {
			// Get the fileinfo
			fileInfo, err := os.Stat(env.tlsCrtPath)

			// Checks for the error
			if err != nil {
				log.Errorf("Error getting file info for %v : %v", env.tlsCrtPath, err)
			}
			// Gives the modification time
			certFileTimes.tlsCrtModTime = fileInfo.ModTime()
			log.Infof("Initial Cert file last modification time : %v\n", certFileTimes.tlsCrtModTime)
		}

	}

	log.Infof("Private key File: %s\n", env.tlsKeyPath)
	if env.tlsKeyPath != "" {
		privateKeyFileExists = doesFileExist(env.tlsKeyPath, false)

		if privateKeyFileExists && getModTime {
			// Get the fileinfo
			fileInfo, err := os.Stat(env.tlsKeyPath)

			// Checks for the error
			if err != nil {
				log.Errorf("Error getting file info for %v : %v", env.tlsKeyPath, err)
			}
			// Gives the modification time
			certFileTimes.tlsKeyModTime = fileInfo.ModTime()
			log.Infof("Initial Private key file last modification time : %v\n", certFileTimes.tlsKeyModTime)
		}

	}

	if checkCA {
		log.Infof("CA Certificate file path: %s\n", env.caCrtPath)
		if env.caCrtPath != "" {
			caCertFileExists = doesFileExist(env.caCrtPath, false) //This check is optional as the file contents might be in the tlsCert
			if caCertFileExists && getModTime {
				// Get the fileinfo
				fileInfo, err := os.Stat(env.caCrtPath)

				// Checks for the error
				if err != nil {
					log.Errorf("Error getting file info for %v : %v", env.caCrtPath, err)
				}
				// Gives the modification time
				certFileTimes.caCrtModTime = fileInfo.ModTime()
				log.Infof("Initial CA Cert file last modification time : %v\n", certFileTimes.caCrtModTime)
			}

		}
	} else {
		caCertFileExists = true // fib result
		if getModTime {
			certFileTimes.caCrtModTime = time.Now()
		}
	}

	return certFileExists && privateKeyFileExists && caCertFileExists

}

func haveCertFilesUpdated(checkCA bool) bool {
	certFileExists := false
	privateKeyFileExists := false
	caCertFileExists := false

	certFileUpdated := false
	privateKeyFileUpdated := false
	caCertFileUpdated := false

	log.Debugf("Checking for update to cert file : %s\n", env.tlsCrtPath)
	if env.tlsCrtPath != "" {
		certFileExists = doesFileExist(env.tlsCrtPath, false)

		if certFileExists {
			// Get the fileinfo
			fileInfo, err := os.Stat(env.tlsCrtPath)

			// Checks for the error
			if err != nil {
				log.Errorf("Error getting file info for %v : %v", env.tlsCrtPath, err)
			}
			// Gives the modification time

			certFileUpdated = (certFileTimes.tlsCrtModTime.Before(fileInfo.ModTime()))
			log.Debugf("Cert file last modification time : %v\n", certFileTimes.tlsCrtModTime)
			log.Debugf("Cert file current modification time : %v\n", fileInfo.ModTime())
			if certFileUpdated {
				certFileTimes.tlsCrtModTime = fileInfo.ModTime() //store updated file time
			}
		}

	}

	log.Debugf("Checking for update to private key File: %s\n", env.tlsKeyPath)
	if env.tlsKeyPath != "" {
		privateKeyFileExists = doesFileExist(env.tlsKeyPath, false)

		if privateKeyFileExists {
			// Get the fileinfo
			fileInfo, err := os.Stat(env.tlsKeyPath)

			// Checks for the error
			if err != nil {
				log.Errorf("Error getting file info for %v : %v", env.tlsKeyPath, err)
			}
			// Gives the modification time

			privateKeyFileUpdated = (certFileTimes.tlsKeyModTime.Before(fileInfo.ModTime()))
			log.Debugf("Private key file last modification time : %v\n", certFileTimes.tlsKeyModTime)
			log.Debugf("Private key file current modification time : %v\n", fileInfo.ModTime())
			if privateKeyFileUpdated {
				certFileTimes.tlsKeyModTime = fileInfo.ModTime() //store updated file time
			}
		}

	}

	if checkCA {
		log.Debugf("CA Certificate file path: %s\n", env.caCrtPath)
		if env.caCrtPath != "" {
			caCertFileExists = doesFileExist(env.caCrtPath, false) //This check is optional as the file contents might be in the tlsCert
			if caCertFileExists {
				// Get the fileinfo
				fileInfo, err := os.Stat(env.caCrtPath)

				// Checks for the error
				if err != nil {
					log.Errorf("Error getting file info for %v : %v", env.caCrtPath, err)
				}
				// Gives the modification time
				caCertFileUpdated = (certFileTimes.caCrtModTime.Before(fileInfo.ModTime()))
				log.Debugf("CA Cert file last modification time : %v\n", certFileTimes.caCrtModTime)
				log.Debugf("CA Cert file current modification time : %v\n", fileInfo.ModTime())
				if caCertFileUpdated {
					certFileTimes.caCrtModTime = fileInfo.ModTime()
				}

			} else {
				caCertFileUpdated = true
			}

		}
	} else {
		caCertFileUpdated = true // fib result
		certFileTimes.caCrtModTime = time.Now()
	}

	return certFileUpdated && privateKeyFileUpdated && caCertFileUpdated

}

/*
HealthCheckProcesses is used for kubernetes liveness/startup probes.
Returns either a 200 or 418 if dependent processes are found or not in the pod.
*/
func HealthCheckServer(w http.ResponseWriter, r *http.Request) {
	var (
		accContent = r.Header.Get(HEADER_ACCEPT)
	)

	log.Infof("HealthCheckServer()--------------->start")

	if checkCertFiles(true, false) == false {
		msg := "Required certificate files are missing from pod"
		log.Errorf(msg)
		httpResponse{acceptedContent: accContent, status: http.StatusInternalServerError, message: msg}.write(w)
		return
	}

	// Load the custom AWS configuration with the provided credentials
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(env.region),
	)

	if err != nil {
		msg := "Unable to load AWS SDK"
		log.Errorf(msg+", %v", err)
		httpResponse{acceptedContent: accContent, status: http.StatusInternalServerError, message: msg}.write(w)
		return
	}

	if (env.awsAccessKey != "") && (env.awsSecretKey != "") {
		cfg.Credentials = credentials.NewStaticCredentialsProvider(env.awsAccessKey, env.awsSecretKey, "")
	}

	// Create an ACM client
	svc := acm.NewFromConfig(cfg)

	err2 := checkExpiringCertificates(svc, 30)
	if err2 == nil {
		httpResponse{acceptedContent: accContent, status: http.StatusOK, message: MSG_200_OK}.write(w)
	} else {
		log.Errorf("Returning 418. Kubernetes will restart the pod.")
		httpResponse{acceptedContent: accContent, status: http.StatusTeapot, message: MSG_418_TEAPOT}.write(w)
	}

	log.Infof("HealthCheckServer()--------------->end")
}

func boolToString(input bool) string {
	retval := "false"
	if input {
		retval = "true"
	}
	return retval
}

func getEnvVars() {

	log.Infof("Read Env vars--->start")

	debugStr := strings.ToLower(os.Getenv("DEBUG"))
	env.debugMode = false
	if debugStr == "y" || debugStr == "yes" || debugStr == "t" || debugStr == "true" {
		env.debugMode = true
	}

	env.namespace = os.Getenv("SECRET_NAMESPACE")
	if env.namespace == "" {
		env.namespace = DEFAULT_NAMESPACE
	}

	env.autoCertName = os.Getenv("AUTOCERT_CERTSECRET_NAME")
	if env.autoCertName == "" {
		env.autoCertName = DEFAULT_AUTOCERT_SECRETNAME
	}

	// Retrieve AWS credentials from environment variables
	env.awsAccessKey = os.Getenv("AWS_ACCESS_KEY_ID")
	env.awsSecretKey = os.Getenv("AWS_SECRET_ACCESS_KEY")

	env.region = os.Getenv("AWS_REGION")
	if env.region == "" {
		env.region = "us-west-2"
	}

	env.vpc = os.Getenv("AWS_VPC")

	env.domain = strings.ToLower(os.Getenv("AWS_R53_DOMAIN"))
	if env.domain == "" {
		env.domain = "espdqa.infra-host.com"
	}

	env.acmCertificateName = os.Getenv("CSP_CERTIFICATE_NAME_TAG")

	// Define the certificate name (tag value)
	if env.acmCertificateName == "" {
		env.acmCertificateName = "ACM_Certificate_Importer"
	}

	env.certificateNameSpace = os.Getenv("K8S_CERTIFICATE_NAMESPACE")
	// Define the certificate namespace (tag value)
	if env.certificateNameSpace == "" {
		env.certificateNameSpace = "orch-gateway"
	}

	env.k8sCertSecretName = os.Getenv("K8S_CERT_SECRET_NAME")
	if env.k8sCertSecretName == "" {
		env.k8sCertSecretName = DEFAULT_SECRETNAME
	}

	envStr := strings.ToLower(os.Getenv("CREATE_K8S_CERT_SECRET"))
	env.createK8sCertSecret = false
	if envStr == "y" || envStr == "yes" || envStr == "t" || envStr == "true" || envStr == "1" {
		env.createK8sCertSecret = true
	}

	if env.debugMode {
		//env.tlsCrtPath = DEBUG_TLS_CERT_PATH
		//env.tlsKeyPath = DEBUG_TLS_KEY_PATH
		//env.caCrtPath = DEBUG_CA_CERT_PATH

		env.tlsCrtPath = os.Getenv("CERTIFICATE_FILE")
		if env.tlsCrtPath == "" {
			env.tlsCrtPath = DEBUG_TLS_CERT_PATH
		}

		env.tlsKeyPath = os.Getenv("PRIVATE_KEY_FILE")
		if env.tlsKeyPath == "" {
			env.tlsKeyPath = DEBUG_TLS_KEY_PATH
		}
		env.caCrtPath = os.Getenv("CA_CERTIFICATE_FILE")
		if env.caCrtPath == "" {
			env.caCrtPath = DEBUG_CA_CERT_PATH
		}

	} else {
		env.tlsCrtPath = os.Getenv("CERTIFICATE_FILE")
		if env.tlsCrtPath == "" {
			env.tlsCrtPath = DEFAULT_TLS_CERT_PATH
		}

		env.tlsKeyPath = os.Getenv("PRIVATE_KEY_FILE")
		if env.tlsKeyPath == "" {
			env.tlsKeyPath = DEFAULT_TLS_KEY_PATH
		}
		env.caCrtPath = os.Getenv("CA_CERTIFICATE_FILE")
		if env.caCrtPath == "" {
			env.caCrtPath = DEFAULT_CA_CERT_PATH
		}
	}

	log.Debugf("env.tlsCrtPath = %s", env.tlsCrtPath)
	log.Debugf("env.tlsKeyPath = %s", env.tlsKeyPath)
	log.Debugf("env.caCrtPath = %s", env.caCrtPath)

	env.inter1CertUrl = os.Getenv("INTERMEDIATE1_CERT_URL")
	if (len(env.inter1CertUrl) == 0) || env.inter1CertUrl == "" {
		env.inter1CertUrl = DEFAULT_INTER1_URL
	}
	env.inter2CertUrl = os.Getenv("INTERMEDIATE2_CERT_URL")
	if (len(env.inter2CertUrl) == 0) || env.inter2CertUrl == "" {
		env.inter2CertUrl = DEFAULT_INTER2_URL
	}

	env.rootCertUrl = os.Getenv("ROOT_CERT_URL")
	if (len(env.rootCertUrl) == 0) || env.rootCertUrl == "" {
		env.rootCertUrl = DEFAULT_ROOT_URL
	}

	inter1PEM, err := httpGetCertPEM(env.inter1CertUrl)
	if err != nil {
		env.inter1Cert, err = b64.StdEncoding.DecodeString(DEFAULT_INTER1_PEM) //UTF-8
		if err == nil {
			env.inter1Cert = slices.Concat([]byte("\n"), env.inter1Cert)
		} else {
			log.Errorf("Unable to set env.inter1Cert")
		}
	} else {
		if len(inter1PEM) > 0 {
			//certs.r10Crt, _ = b64.StdEncoding.DecodeString(r10Env) //UTF-8
			env.inter1Cert = []byte("\n" + inter1PEM)
		} else {
			env.inter1Cert, err = b64.StdEncoding.DecodeString(DEFAULT_INTER1_PEM) //UTF-8
			if err == nil {
				env.inter1Cert = slices.Concat([]byte("\n"), env.inter1Cert)
			} else {
				log.Errorf("Unable to set env.inter1Cert")
			}
		}
	}

	log.Debugf("inter1  PEM : %s", env.rootCert)

	inter2PEM, err := httpGetCertPEM(env.inter2CertUrl)
	if err != nil {
		env.inter2Cert, err = b64.StdEncoding.DecodeString(DEFAULT_INTER2_PEM) //UTF-8
		if err == nil {
			env.inter2Cert = slices.Concat([]byte("\n"), env.inter2Cert)
		} else {
			log.Errorf("Unable to set env.inter2Cert")
		}
	} else {
		if len(inter2PEM) > 0 {
			//certs.r10Crt, _ = b64.StdEncoding.DecodeString(r10Env) //UTF-8
			env.inter2Cert = []byte("\n" + inter2PEM)
		} else {
			env.inter2Cert, err = b64.StdEncoding.DecodeString(DEFAULT_INTER1_PEM) //UTF-8
			if err == nil {
				env.inter2Cert = slices.Concat([]byte("\n"), env.inter2Cert)
			} else {
				log.Errorf("Unable to set env.inter2Cert")
			}
		}
	}

	log.Debugf("Inter2 PEM : %s", env.rootCert)

	rootPEM, err := httpGetCertPEM(env.rootCertUrl)
	if err != nil {
		env.rootCert, err = b64.StdEncoding.DecodeString(DEFAULT_ROOT_PEM) //UTF-8
		if err == nil {
			env.rootCert = slices.Concat([]byte("\n"), env.rootCert)
		} else {
			log.Errorf("Unable to set env.rootCert")
		}
	} else {
		if len(rootPEM) > 0 {
			//certs.r10Crt, _ = b64.StdEncoding.DecodeString(r10Env) //UTF-8
			env.rootCert = []byte("\n" + rootPEM)
		} else {
			env.rootCert, err = b64.StdEncoding.DecodeString(DEFAULT_ROOT_PEM) //UTF-8
			if err == nil {
				env.rootCert = slices.Concat([]byte("\n"), env.rootCert)
			} else {
				log.Errorf("Unable to set env.rootCert")
			}
		}
	}

	log.Debugf("Root cert PEM : %s", rootPEM)

	log.Debugf("Root cert ENV PEM : %s", env.rootCert)

	sleepStr := os.Getenv("POD_FILE_UPDATE_TIMEOUT_SECS")
	if len(sleepStr) > 0 {
		podFileUpdateSleepTimeout, err := strconv.Atoi(sleepStr)
		if err != nil {
			env.podFileUpdateSleepTimeout = DEFAULT_SLEEP_TIMEOUT
		} else {
			env.podFileUpdateSleepTimeout = podFileUpdateSleepTimeout * 1000 //convert to millis
		}
	} else {
		env.podFileUpdateSleepTimeout = DEFAULT_SLEEP_TIMEOUT
	}

	env.disableCertMatchChecks = false
	dmc := strings.ToLower(os.Getenv("DISABLE_CERT_MATCH_CHECKS"))
	if (dmc == "1") || (dmc == "true") || (dmc == "t") || (dmc == "y") || (dmc == "yes") {
		env.disableCertMatchChecks = true
	}

	env.importIntoACMIfNotExists = true
	acmImp := strings.ToLower(os.Getenv("ACM_IMPORT_IF_NOT_EXISTS"))
	if (acmImp == "0") || (acmImp == "false") || (acmImp == "f") || (acmImp == "n") || (acmImp == "no") {
		env.importIntoACMIfNotExists = false
	}

	log.Infof("Read Env vars--->end")
}

func (lp DNSRecordParams) validate() error {
	return validation.ValidateStruct(&lp,
		validation.Field(&lp.Region, validation.Required, validation.Length(1, 50), is.ASCII),
		//validation.Field(&lp.Domain, validation.Required, validation.Length(1, 15), is.ASCII),
		validation.Field(&lp.fqdn, validation.Required, validation.Length(1, 255), is.ASCII),
		validation.Field(&lp.Recordtype, validation.Length(1, 10), is.ASCII),
		validation.Field(&lp.Recordvalue, validation.Length(1, 255), is.ASCII),
	)
}

func HTTPCreateDNSRecord(w http.ResponseWriter, r *http.Request) {
	var accContent = strings.ToLower(r.Header.Get(HEADER_ACCEPT))
	var params DNSRecordParams
	var dnsrecords []DNSRecordParams
	//params.escapequotes = false

	switch r.Method {
	case "GET":
		for key := range r.URL.Query() {
			switch strings.ToLower(key) {
			case "region":
				params.Region = r.URL.Query().Get(key)
			case "domain":
				params.Domain = strings.ToLower(r.URL.Query().Get(key))
			case "recordtype":
				params.Recordtype = strings.ToUpper(r.URL.Query().Get(key))
			case "recordname":
				params.Recordname = strings.ToLower(r.URL.Query().Get(key))
				params.Recordvalue = r.URL.Query().Get(key)
			}

		}

		if (params.Recordtype != "") && (params.Recordvalue != "") {
			dnsrecords = append(dnsrecords, sanitizeDNSRecord(params))
		}

		if err := params.validate(); err != nil {
			log.Warnf("Parameter validation error: %v", err)
			log.Warnf("Returning http Bad Request (400)")
			httpResponse{acceptedContent: accContent, status: http.StatusBadRequest, message: MSG_400_BAD_RQ}.write(w)
			return
		}

		log.Debugf("AWS Region: %s", params.Region)
		log.Debugf("DNS Domain: %s", params.Domain)
		log.Debugf("DNS Record Type: %s", params.Recordtype)
		log.Debugf("DNS Record Name: %s", params.Recordname)
		log.Debugf("DNS Record Value: %s", params.Recordvalue)
		log.Debugf("Accepted Content Type: %s", accContent)

		break
	case "POST":
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Error reading body", http.StatusInternalServerError)
			return
		}

		if len(body) > 0 {
			log.Infof("Got POST data: %s\n", string(body[:]))
			var dnsrecord DNSRecordParams
			err := json.Unmarshal(body, &dnsrecord)
			if err != nil {
				// If the single record parsing fails, try parsing as multiple records
				err := json.Unmarshal(body, &dnsrecords)
				if err != nil {
					msg := "Error parsing submitted body JSON"
					log.Errorf(msg+", err: %v", err)
					log.Errorf("Returning http Bad Request (400)")
					httpResponse{acceptedContent: accContent, status: http.StatusBadRequest, message: MSG_400_BAD_RQ}.write(w)
				}
			} else {
				// If the single record parsing succeeds, append it to the dnsrecords slice
				dnsrecords = append(dnsrecords, sanitizeDNSRecord(dnsrecord))
			}

		} else {
			http.Error(w, "Body is empty", http.StatusInternalServerError)
		}
		break
	default:
		log.Warnf("Disallowed http method call. Returning http Method Not Allowed (405)")
		httpResponse{acceptedContent: accContent, status: http.StatusMethodNotAllowed, message: MSG_405_METHOD}.write(w)
	}

	log.Infof("Number of DNS records to create/update: %v", len(dnsrecords))

	if len(dnsrecords) == 1 {
		if err := dnsrecords[0].validate(); err != nil {
			log.Warnf("Parameter validation error: %v", err)
			log.Warnf("Returning http Bad Request (400)")
			httpResponse{acceptedContent: accContent, status: http.StatusBadRequest, message: MSG_400_BAD_RQ}.write(w)
			return
		}
		// Load the AWS configuration
		cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(params.Region))
		if err != nil {
			msg := "Unable to load AWS SDK"
			log.Errorf(msg+", err: %v", err)
			httpResponse{acceptedContent: accContent, status: http.StatusInternalServerError, message: msg}.write(w)
			return
		}

		svc := route53.NewFromConfig(cfg)
		log.Infof("records: %v", dnsrecords)
		createSingleDNSRecord(svc, w, accContent, dnsrecords[0])
	} else {
		var invalidStrings strings.Builder
		var responseStrings strings.Builder
		var errorStrings strings.Builder
		var invalid = 0
		var errorcount = 0
		var successcount = 0
		var recCount = 0

		log.Debugf("DNS Records : %v", dnsrecords)

		for _, record := range dnsrecords {
			recCount++

			record = sanitizeDNSRecord(record)

			// Load the AWS configuration
			cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(record.Region))
			if err != nil {
				msg := "Unable to load AWS SDK"
				log.Errorf(msg+", err: %v", err)
				httpResponse{acceptedContent: accContent, status: http.StatusInternalServerError, message: msg}.write(w)
				return
			}

			svc := route53.NewFromConfig(cfg)

			log.Infof("Domain: %s, RecordName: %s, RecordType: %s, Value: %s, Region: %s\n", record.Domain, record.Recordname, record.Recordtype, record.Recordvalue, record.Region)
			if err := record.validate(); err != nil {
				log.Errorf("Parameter validation error: %v", err)
				//log.Warnf("Returning http Bad Request (400)")
				inputStr := fmt.Sprintf("Error %s, Domain: %s, RecordName: %s, RecordType: %s, Value: %s\n", err, record.Domain, record.Recordname, record.Recordtype, record.Recordvalue)
				invalidStrings.WriteString(inputStr)
				invalid++
				//httpResponse{acceptedContent: accContent, status: http.StatusBadRequest, message: MSG_400_BAD_RQ}.write(w)
				continue //Skip malformed records
			}

			result, err2 := createDNSRecord(svc, record)
			if err2 != nil {
				errorcount++
				errorStrings.WriteString(result)
			} else {
				successcount++
				responseStrings.WriteString(result)
			}
		}
		msg := "Completed " + strconv.Itoa(recCount) + " records, with " + strconv.Itoa(invalid) + " invalid records, " + strconv.Itoa(errorcount) + " errors, " + strconv.Itoa(successcount) + " successfully inserted/updated.\n\n"
		msg = msg + "Invalid transactions\n" + invalidStrings.String() + "\n"
		msg = msg + "Error transactions\n" + errorStrings.String() + "\n"
		msg = msg + "Sucessful transactions\n" + responseStrings.String() + "\n"

		httpResponse{acceptedContent: accContent, status: http.StatusOK, message: msg}.write(w)

	}

}

func sanitizeDNSRecord(record DNSRecordParams) DNSRecordParams {
	if record.Region == "" {
		record.Region = env.region
	}

	if record.Domain == "" {
		record.Domain = strings.ToLower(env.domain)
	}

	if !strings.HasSuffix(record.Recordname, ".") {
		//params.recordname = strings.TrimRight(params.recordname, ".")
		record.Recordname = record.Recordname + "."
		record.Recordname = strings.ToLower(record.Recordname)
	}

	if strings.HasPrefix(record.Domain, ".") {
		record.Domain = strings.TrimLeft(record.Domain, ".")
		record.Domain = strings.ToLower(record.Domain)
	}

	if record.Recordtype == "TXT" {
		record.Recordvalue = escapeString(record.Recordvalue)
	}

	record.fqdn = strings.ToLower(record.Recordname + record.Domain)

	return record

}

func readCertFiles(cert certChain) (certChain, error) {
	// Read certificate and key files

	// cert, err := os.ReadFile(env.certFile)

	tlsCrt, err := os.ReadFile(env.tlsCrtPath)
	if err != nil {
		log.Infof("Failed to read certificate  %s: %s", env.tlsCrtPath, err.Error())
	}
	tlsKey, err := os.ReadFile(env.tlsKeyPath)
	if err != nil {
		log.Infof("Failed to read private key  %s: %s", env.tlsKeyPath, err.Error())
	}
	fixCA := false
	caCrt, err := os.ReadFile(env.caCrtPath)
	if err != nil {
		log.Infof("Unable to read CA certificate file %s: %s", env.caCrtPath, err.Error())
		fixCA = true
		err = nil //We don't want to return this to the callee
	}

	if (caCrt == nil) || (len(caCrt) == 0) {
		log.Infof("CA certificate file %s is either empty or null ", env.caCrtPath)
		fixCA = true
	}

	if fixCA {
		log.Infof("Splitting %s: into tls.crt and ca.crt", env.tlsCrtPath)
		//cert-manager creates a cert with the ca cert as a 2nd cert in the tls cert, and the ca cert secret is empty
		//this splits the tls cert as assigns the certs correctly for import into AWS.
		certSeparator := []byte("-----END CERTIFICATE-----")
		certParts := bytes.SplitAfter(tlsCrt, certSeparator)

		log.Infof("Found %v parts in the tls.crt file", (len(certParts) - 1)) //ignore empty last part of slice
		if len(certParts) > 2 {
			tlsCrt = certParts[0] //extract the tls cert from the slice and assign to tlsCrt var
			caCrt = certParts[1]  //the 2nd cert is the ca cert
		} else {
			err = fmt.Errorf("tlsCrt file does not contain certificate chain to populate caCrt")
		}
	}

	//tlsCertChain := slices.Concat(tlsCrt, cert.inter1Crt, cert.inter2Crt, cert.rootCrt)
	//caCertChain := slices.Concat(caCrt, env.inter1Cert, env.inter2Cert, env.rootcert)

	log.Debugf("Root cert PEM : %s", env.rootCert)
	tlsCertChain := slices.Concat(tlsCrt, caCrt, env.rootCert)
	caCertChain := slices.Concat(caCrt, env.rootCert)
	cert.tlsCrt = tlsCrt
	cert.tlsKey = tlsKey
	cert.caCrt = caCrt
	cert.tlsCrtChain = tlsCertChain
	cert.caCrtChain = caCertChain

	log.Debugf("TLS cert chain PEM : %s", cert.tlsCrtChain)

	return cert, err

}

func createK8sCertificateSecret(cert certChain) error {

	/* 	if env.debugMode {
	   		log.Infof("tls.crt : " + string(cert.tlsCrt))
	   		log.Infof("tls.key : " + string(cert.tlsKey))
	   		log.Infof("ca.crt : " + string(cert.caCrt))
	   		// os.Exit(1)
	   	}
	*/

	dynamicClient, err := getKubernetesClient()
	if err != nil {
		err = fmt.Errorf("Unable to create client connection to Kubernetes cluster for secret creation.")
		return err
	}

	secretGVR := schema.GroupVersionResource{Version: "v1", Resource: "secrets"}
	secret := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Secret",
			"metadata": map[string]interface{}{
				"name":      env.k8sCertSecretName,
				"namespace": env.namespace,
			},
			"type": "kubernetes.io/tls",
			"data": map[string]interface{}{
				"tls.crt": cert.tlsCrtChain,
				"tls.key": cert.tlsKey,
				//"ca.crt":  cert.caCrt,
			},
		},
	}

	// Create the secret in Kubernetes
	_, err = dynamicClient.Resource(secretGVR).Namespace(env.namespace).Apply(context.TODO(), env.k8sCertSecretName, secret, metav1.ApplyOptions{FieldManager: "application/apply-patch", Force: true})

	if err != nil {
		log.Errorf("Error creating secret: %s", err.Error())
		return fmt.Errorf("failed to create/update secret %v : %v", env.k8sCertSecretName, err)
	} else {
		log.Infof("Sucessfully updated Kubernetes secret " + env.k8sCertSecretName + " in namespace:" + env.namespace)
	}
	return nil
}

func createOrUpdateK8sCertificateSecret(cert certChain) error {

	dynamicClient, err := getKubernetesClient()
	if err != nil {
		err = fmt.Errorf("Unable to create client connection to Kubernetes cluster for secret creation.")
		return err
	}

	secretGVR := schema.GroupVersionResource{Version: "v1", Resource: "secrets"}
	secret := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Secret",
			"metadata": map[string]interface{}{
				"name":      env.k8sCertSecretName,
				"namespace": env.namespace,
			},
			"type": "kubernetes.io/tls",
			"data": map[string]interface{}{
				"tls.crt": cert.tlsCrtChain,
				"tls.key": cert.tlsKey,
				//"ca.crt":  cert.caCrt,
			},
		},
	}

	existingSecret, err := dynamicClient.Resource(secretGVR).Namespace(env.namespace).Get(context.TODO(), env.k8sCertSecretName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Secret does not exist, create it
			log.Infof("Kubernetes Secret " + env.k8sCertSecretName + " does not exist, creating.")
			_, err = dynamicClient.Resource(secretGVR).Namespace(env.namespace).Create(context.TODO(), secret, metav1.CreateOptions{})
			//_, err = dynamicClient.Resource(secretGVR).Namespace(env.namespace).Apply(context.TODO(), env.k8sCertSecretName, secret, metav1.ApplyOptions{FieldManager: "application/apply-patch", Force: true})
			if err != nil {
				return fmt.Errorf("failed to create secret %v : %v", env.k8sCertSecretName, err)
			}
			log.Infof("Sucessfully created Kubernetes secret " + env.k8sCertSecretName + " in namespace:" + env.namespace)
			return nil
		}
		return fmt.Errorf("failed to get secret %v : %v", env.k8sCertSecretName, err)
	}
	// Secret exists, update it
	log.Infof("Kubernetes Secret " + env.k8sCertSecretName + " dalready exists, updating.")
	secret.SetResourceVersion(existingSecret.GetResourceVersion())
	_, err = dynamicClient.Resource(secretGVR).Namespace(env.namespace).Update(context.TODO(), secret, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update secret %v : %v", env.k8sCertSecretName, err)
	}
	log.Infof("Sucessfully updated Kubernetes secret " + env.k8sCertSecretName + " in namespace:" + env.namespace)
	return nil
}

func getKubernetesClient() (*dynamic.DynamicClient, error) {
	var clusterConfig *rest.Config
	var dynamicClient *dynamic.DynamicClient
	var err error

	clusterConfig, err = rest.InClusterConfig()
	if err == nil {
		dynamicClient, err = dynamic.NewForConfig(clusterConfig)
		if err != nil {
			log.Errorf("Error creating dynamic client: %v\n", err)
			return nil, err
		}
	} else {
		userHomeDir, err := os.UserHomeDir()
		if err != nil {
			log.Errorf("Error getting user home directory: %v\n", err)
			return nil, err
		}

		kubeConfigPath := filepath.Join(userHomeDir, ".kube", "config")
		log.Infof("Using kubeconfig: %s\n", kubeConfigPath)

		kubeConfig, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
		if err != nil {
			log.Errorf("Error getting Kubernetes config: %v\n", err)
			return nil, err
		}

		dynamicClient, err = dynamic.NewForConfig(kubeConfig)
		if err != nil {
			log.Errorf("Error creating dynamic client: %v\n", err)
			return nil, err
		}

	}
	return dynamicClient, err
}
func createSingleDNSRecord(svc *route53.Client, w http.ResponseWriter, accContent string, params DNSRecordParams) {

	var vpc string
	var region string
	if params.VPC == "" {
		vpc = env.vpc
	} else {
		vpc = params.VPC
	}

	if params.Region == "" {
		region = env.region
	} else {
		region = params.Region
	}

	log.Infof("params.VPC : " + params.VPC)
	log.Infof("env.vpc : " + env.vpc)
	log.Infof("vpc : " + vpc)

	// Check if the hosted zone exists, if not create it
	hostedZoneID, err := getOrCreateHostedZone(svc, params.Domain, vpc, region, params.IsPrivate, true)
	if err != nil {
		msg := "Failed to get or create Route53 hosted zone for domain : " + params.Domain
		log.Errorf(msg+", err: %v", err)
		httpResponse{acceptedContent: accContent, status: http.StatusInternalServerError, message: msg}.write(w)
		return
	}
	log.Infof("Got hostedZoneID : " + hostedZoneID + " for domain: " + params.Domain)

	// Check if the DNS record exists
	existingDNSRecord, err := getRoute53Record(svc, hostedZoneID, params.fqdn, params.Recordtype)
	if err != nil {
		msg := "Failed to get Route53 DNS record " + params.fqdn + " for hosted zone : " + hostedZoneID
		log.Errorf(msg+", err: %v", err)
		httpResponse{acceptedContent: accContent, status: http.StatusInternalServerError, message: msg}.write(w)
		return
	}

	if existingDNSRecord != nil {
		// Update the existing DNS record
		log.Infof("Updating existing DNS " + params.Recordtype + " record " + params.fqdn + " with value " + params.Recordvalue)
		err = updateRoute53Record(svc, hostedZoneID, params.fqdn, params.Recordtype, params.Recordvalue)
		if err != nil {
			msg := "Failed to update Route53 DNS " + params.Recordtype + " type record " + params.fqdn + " for hosted zone : " + hostedZoneID
			log.Errorf(msg+", err: %v", err)
			httpResponse{acceptedContent: accContent, status: http.StatusInternalServerError, message: msg}.write(w)
			return
		}
		msg := "Updated Route53 DNS " + params.Recordtype + " type record " + params.fqdn + " for hosted zone : " + hostedZoneID
		log.Infof(msg)
		httpResponse{acceptedContent: accContent, status: http.StatusOK, message: msg}.write(w)
	} else {
		// Create the DNS record
		log.Infof("Importing new DNS " + params.Recordtype + " record " + params.fqdn + " with value " + params.Recordvalue)
		err = createRoute53Record(svc, hostedZoneID, params.Recordname+params.Domain, params.Recordtype, params.Recordvalue)
		if err != nil {
			msg := "Failed to insert Route53 DNS record " + params.Recordtype + " type record " + params.fqdn + " for hosted zone : " + hostedZoneID
			log.Errorf(msg+", err: %v", err)
			httpResponse{acceptedContent: accContent, status: http.StatusInternalServerError, message: msg}.write(w)
			return
		}
		msg := "Created Route53 DNS " + params.Recordtype + " type record " + params.fqdn + " for hosted zone : " + hostedZoneID
		log.Infof(msg)
		httpResponse{acceptedContent: accContent, status: http.StatusOK, message: msg}.write(w)
	}
}

func createDNSRecord(svc *route53.Client, params DNSRecordParams) (string, error) {
	var retval = ""
	var vpc string
	var region string
	if params.VPC == "" {
		vpc = env.vpc
	} else {
		vpc = params.VPC
	}

	if params.Region == "" {
		region = env.region
	} else {
		region = params.Region
	}

	// Check if the hosted zone exists, if not create it
	hostedZoneID, err := getOrCreateHostedZone(svc, params.Domain, vpc, region, params.IsPrivate, true)
	if err != nil {
		msg := "Failed to get or create Route53 hosted zone for domain : " + params.Domain
		log.Errorf(msg+", err: %v", err)
		//httpResponse{acceptedContent: accContent, status: http.StatusInternalServerError, message: msg}.write(w)
		return msg + "\n", err
	}
	log.Infof("Got hostedZoneID : " + hostedZoneID + " for domain: " + params.Domain)

	// Check if the DNS record exists
	existingDNSRecord, err := getRoute53Record(svc, hostedZoneID, params.fqdn, params.Recordtype)
	if err != nil {
		msg := "Failed to get Route53 DNS record " + params.fqdn + " for hosted zone : " + hostedZoneID
		log.Errorf(msg+", err: %v", err)
		//httpResponse{acceptedContent: accContent, status: http.StatusInternalServerError, message: msg}.write(w)
		return msg + "\n", err
	}

	if existingDNSRecord != nil {
		// Update the existing DNS record
		log.Infof("Updating existing DNS " + params.Recordtype + " record " + params.fqdn + " with value " + params.Recordvalue)
		err = updateRoute53Record(svc, hostedZoneID, params.fqdn, params.Recordtype, params.Recordvalue)
		if err != nil {
			msg := "Failed to update Route53 DNS " + params.Recordtype + " type record " + params.fqdn + " for hosted zone : " + hostedZoneID
			log.Errorf(msg+", err: %v", err)
			//httpResponse{acceptedContent: accContent, status: http.StatusInternalServerError, message: msg}.write(w)
			return msg + "\n", err
		}
		msg := "Updated Route53 DNS " + params.Recordtype + " type record " + params.fqdn + " for hosted zone : " + hostedZoneID
		log.Infof(msg)
		retval = msg + "\n"
		//httpResponse{acceptedContent: accContent, status: http.StatusOK, message: msg}.write(w)
	} else {
		// Create the DNS record
		log.Infof("Importing new DNS " + params.Recordtype + " record " + params.fqdn + " with value " + params.Recordvalue)
		err = createRoute53Record(svc, hostedZoneID, params.Recordname+params.Domain, params.Recordtype, params.Recordvalue)
		if err != nil {
			msg := "Failed to insert Route53 DNS record " + params.Recordtype + " type record " + params.fqdn + " for hosted zone : " + hostedZoneID
			log.Errorf(msg+", err: %v", err)
			//httpResponse{acceptedContent: accContent, status: http.StatusInternalServerError, message: msg}.write(w)
			return msg + "\n", err
		}
		msg := "Created Route53 DNS " + params.Recordtype + " type record " + params.fqdn + " for hosted zone : " + hostedZoneID
		log.Infof(msg)
		retval = msg + "\n"
		//httpResponse{acceptedContent: accContent, status: http.StatusOK, message: msg}.write(w)
	}
	return retval, nil
}

func deleteSingleDNSRecord(svc *route53.Client, w http.ResponseWriter, accContent string, params DNSRecordParams) {
	// Check if the hosted zone exists, if not create it
	hostedZoneID, err := getOrCreateHostedZone(svc, params.Domain, "", "", params.IsPrivate, false)
	if err != nil {
		msg := "Failed to get or create Route53 hosted zone for domain : " + params.Domain
		log.Errorf(msg+", err: %v", err)
		httpResponse{acceptedContent: accContent, status: http.StatusInternalServerError, message: msg}.write(w)
		return
	}
	log.Infof("Got hostedZoneID : " + hostedZoneID + " for domain: " + params.Domain)

	// Check if the DNS record exists
	existingDNSRecord, err := getRoute53Record(svc, hostedZoneID, params.fqdn, params.Recordtype)
	if err != nil {
		msg := "Failed to get Route53 DNS record " + params.fqdn + " for hosted zone : " + hostedZoneID
		log.Errorf(msg+", err: %v", err)
		httpResponse{acceptedContent: accContent, status: http.StatusInternalServerError, message: msg}.write(w)
		return
	}

	if existingDNSRecord != nil {
		// Update the existing DNS record
		log.Infof("Deleting existing DNS record " + params.fqdn + " in hosted zone " + hostedZoneID)
		// Delete the DNS record
		err = deleteRoute53Record(svc, hostedZoneID, params.fqdn, params.Recordtype)
		if err != nil {
			msg := "Failed to delete Route53 DNS " + params.Recordtype + " type record " + params.fqdn + " for hosted zone : " + hostedZoneID
			log.Errorf(msg+", err: %v", err)
			httpResponse{acceptedContent: accContent, status: http.StatusInternalServerError, message: msg}.write(w)
			return
		}
		msg := "Deleted Route53 DNS " + params.Recordtype + " type record " + params.fqdn + " for hosted zone : " + hostedZoneID
		log.Infof(msg+", err: %v", err)
		httpResponse{acceptedContent: accContent, status: http.StatusOK, message: msg}.write(w)
	} else {
		msg := "No Route53 DNS record of type " + params.Recordtype + " called " + params.fqdn + " for hosted zone : " + hostedZoneID + " found to delete. Done."
		log.Infof(msg+", err: %v", err)
		httpResponse{acceptedContent: accContent, status: http.StatusOK, message: msg}.write(w)
	}
}

func deleteDNSRecord(svc *route53.Client, params DNSRecordParams) (string, error) {
	// Check if the hosted zone exists, if not create it
	hostedZoneID, err := getOrCreateHostedZone(svc, params.Domain, "", "", params.IsPrivate, false)
	if err != nil {
		msg := "Failed to get or create Route53 hosted zone for domain : " + params.Domain
		log.Errorf(msg+", err: %v", err)
		//httpResponse{acceptedContent: accContent, status: http.StatusInternalServerError, message: msg}.write(w)
		return msg + "\n", err

	}
	log.Infof("Got hostedZoneID : " + hostedZoneID + " for domain: " + params.Domain)

	// Check if the DNS record exists
	existingDNSRecord, err := getRoute53Record(svc, hostedZoneID, params.fqdn, params.Recordtype)
	if err != nil {
		msg := "Failed to get Route53 DNS record " + params.fqdn + " for hosted zone : " + hostedZoneID
		log.Errorf(msg+", err: %v", err)
		//httpResponse{acceptedContent: accContent, status: http.StatusInternalServerError, message: msg}.write(w)
		return msg + "\n", err
	}

	if existingDNSRecord != nil {
		// Update the existing DNS record
		log.Infof("Deleting existing DNS record " + params.fqdn + " in hosted zone " + hostedZoneID)
		// Delete the DNS record
		err = deleteRoute53Record(svc, hostedZoneID, params.fqdn, params.Recordtype)
		if err != nil {
			msg := "Failed to delete Route53 DNS " + params.Recordtype + " type record " + params.fqdn + " for hosted zone : " + hostedZoneID
			log.Errorf(msg+", err: %v", err)
			//httpResponse{acceptedContent: accContent, status: http.StatusInternalServerError, message: msg}.write(w)
			return msg + "\n", err
		}
		msg := "Deleted Route53 DNS " + params.Recordtype + " type record " + params.fqdn + " for hosted zone : " + hostedZoneID
		log.Infof(msg+", err: %v", err)
		//httpResponse{acceptedContent: accContent, status: http.StatusOK, message: msg}.write(w)
		return msg + "\n", nil
	} else {
		msg := "No Route53 DNS record of type " + params.Recordtype + " called " + params.fqdn + " for hosted zone : " + hostedZoneID + " found to delete. Done."
		log.Infof(msg+", err: %v", err)
		//httpResponse{acceptedContent: accContent, status: http.StatusOK, message: msg}.write(w)
		return msg + "\n", nil
	}
}

func HTTPDeleteDNSRecord(w http.ResponseWriter, r *http.Request) {
	var accContent = strings.ToLower(r.Header.Get(HEADER_ACCEPT))
	var params DNSRecordParams
	var dnsrecords []DNSRecordParams

	switch r.Method {
	case "GET":
		for key := range r.URL.Query() {
			switch strings.ToLower(key) {
			case "region":
				params.Region = r.URL.Query().Get(key)
			case "domain":
				params.Domain = r.URL.Query().Get(key)
			case "recordtype":
				params.Recordtype = strings.ToUpper(r.URL.Query().Get(key))
			case "recordname":
				params.Recordname = r.URL.Query().Get(key)
			case "recordvalue":
				params.Recordvalue = r.URL.Query().Get(key)
			}
		}

		if (params.Recordtype != "") && (params.Recordvalue != "") {
			dnsrecords = append(dnsrecords, sanitizeDNSRecord(params))
		}

		if err := params.validate(); err != nil {
			log.Warnf("Parameter validation error: %v", err)
			log.Warnf("Returning http Bad Request (400)")
			httpResponse{acceptedContent: accContent, status: http.StatusBadRequest, message: MSG_400_BAD_RQ}.write(w)
			return
		}

		log.Debugf("AWS Region: %s", params.Region)
		log.Debugf("DNS Domain: %s", params.Domain)
		log.Debugf("DNS Record Type: %s", params.Recordtype)
		log.Debugf("DNS Record Name: %s", params.Recordname)
		log.Debugf("DNS Record Value: %s", params.Recordvalue)
		log.Debugf("Accepted Content Type: %s", accContent)

		//log.Warnf("Missing required parameters. Returning http Bad Request (400)")
		//httpResponse{acceptedContent: accContent, status: http.StatusBadRequest, message: MSG_400_BAD_RQ}.write(w)
		break
	case "POST":
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Error reading body", http.StatusInternalServerError)
			return
		}

		if len(body) > 0 {
			log.Infof("Got POST data: %s\n", string(body[:]))
			var dnsrecord DNSRecordParams
			err := json.Unmarshal(body, &dnsrecord)
			if err != nil {
				// If the single record parsing fails, try parsing as multiple records
				err := json.Unmarshal(body, &dnsrecords)
				if err != nil {
					msg := "Error parsing submitted body JSON"
					log.Errorf(msg+", err: %v", err)
					log.Errorf("Returning http Bad Request (400)")
					httpResponse{acceptedContent: accContent, status: http.StatusBadRequest, message: MSG_400_BAD_RQ}.write(w)
				}
			} else {
				// If the single record parsing succeeds, append it to the dnsrecords slice
				dnsrecords = append(dnsrecords, sanitizeDNSRecord(dnsrecord))
			}

		} else {
			http.Error(w, "Body is empty", http.StatusInternalServerError)
		}

		break
	default:
		log.Warnf("Disallowed http method call. Returning http Method Not Allowed (405)")
		httpResponse{acceptedContent: accContent, status: http.StatusMethodNotAllowed, message: MSG_405_METHOD}.write(w)
	}

	log.Infof("Number of DNS records to delete: %v", len(dnsrecords))

	if len(dnsrecords) == 1 {
		if err := dnsrecords[0].validate(); err != nil {
			log.Warnf("Parameter validation error: %v", err)
			log.Warnf("Returning http Bad Request (400)")
			httpResponse{acceptedContent: accContent, status: http.StatusBadRequest, message: MSG_400_BAD_RQ}.write(w)
			return
		}
		// Load the AWS configuration
		cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(params.Region))
		if err != nil {
			msg := "Unable to load AWS SDK"
			log.Errorf(msg+", err: %v", err)
			httpResponse{acceptedContent: accContent, status: http.StatusInternalServerError, message: msg}.write(w)
			return
		}

		svc := route53.NewFromConfig(cfg)
		log.Infof("records: %v", dnsrecords)
		deleteSingleDNSRecord(svc, w, accContent, dnsrecords[0])
	} else {
		var invalidStrings strings.Builder
		var responseStrings strings.Builder
		var errorStrings strings.Builder
		var invalid = 0
		var errorcount = 0
		var successcount = 0
		var recCount = 0

		log.Debugf("DNS Records : %v", dnsrecords)

		for _, record := range dnsrecords {
			recCount++

			record = sanitizeDNSRecord(record)

			// Load the AWS configuration
			cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(env.region))
			if err != nil {
				msg := "Unable to load AWS SDK"
				log.Errorf(msg+", %v", err)
				httpResponse{acceptedContent: accContent, status: http.StatusInternalServerError, message: msg}.write(w)
				return

			}

			svc := route53.NewFromConfig(cfg)

			log.Infof("Domain: %s, RecordName: %s, RecordType: %s, Value: %s, Region: %s\n", record.Domain, record.Recordname, record.Recordtype, record.Recordvalue, record.Region)
			if err := record.validate(); err != nil {
				log.Errorf("Parameter validation error: %v", err)
				//log.Warnf("Returning http Bad Request (400)")
				inputStr := fmt.Sprintf("Error %s, Domain: %s, RecordName: %s, RecordType: %s, Value: %s\n", err, record.Domain, record.Recordname, record.Recordtype, record.Recordvalue)
				invalidStrings.WriteString(inputStr)
				invalid++
				//httpResponse{acceptedContent: accContent, status: http.StatusBadRequest, message: MSG_400_BAD_RQ}.write(w)
				continue //Skip malformed records
			}

			// Replace with your hosted zone ID
			//hostedZoneID := "Z02949312W0XK5WEA8LW6"
			//domain := "getta.club"
			//Route53RecordName := "test." + domain
			//Route53RecordType := "A"
			//Route53RecordValue := "192.0.2.44"
			result, err2 := deleteDNSRecord(svc, record)
			if err2 != nil {
				errorcount++
				errorStrings.WriteString(result)
			} else {
				successcount++
				responseStrings.WriteString(result)
			}
		}
		msg := "Completed " + strconv.Itoa(recCount) + " records, with " + strconv.Itoa(invalid) + " invalid records, " + strconv.Itoa(errorcount) + " errors, " + strconv.Itoa(successcount) + " successfully inserted/updated.\n\n"
		msg = msg + "Invalid transactions\n" + invalidStrings.String() + "\n"
		msg = msg + "Error transactions\n" + errorStrings.String() + "\n"
		msg = msg + "Sucessful transactions\n" + responseStrings.String() + "\n"
		httpResponse{acceptedContent: accContent, status: http.StatusOK, message: msg}.write(w)
	}

}

func UpdateCert(w http.ResponseWriter, r *http.Request) {
	var accContent = strings.ToLower(r.Header.Get(HEADER_ACCEPT))

	log.Debugf("Response content type: |%s|\n", accContent)
	log.Debugf("AWS Region: %s\n", env.region)
	log.Debugf("K8 Certificate Namespace: %s\n", env.certificateNameSpace)
	log.Debugf("Certificate Name: %s\n", env.acmCertificateName)
	log.Debugf("Certificate File path: %s\n", env.tlsCrtPath)
	log.Debugf("Private key File: %s\n", env.tlsKeyPath)
	log.Debugf("CA Certificate file path: %s\n", env.caCrtPath)
	log.Infof("Pod cert secret files to updated.")

	switch r.Method {
	case "POST":
		log.Infof("POST Request /updatecert at %v\n", time.Now())

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Error reading body", http.StatusInternalServerError)
			return
		}

		if len(body) > 0 {
			log.Infof("Input data: %s\n", string(body[:]))

			var certevent CertEvent
			var certevents []CertEvent

			err := json.Unmarshal([]byte(body), &certevent) //try parse a single certevent
			if err != nil {
				err := json.Unmarshal([]byte(body), &certevents) //fallback to array of events
				if err != nil {
					log.Errorf("Error parsing JSON %s", err)
				}
			} else {
				// If the single record parsing succeeds, append it to the certevents slice
				certevents = append(certevents, certevent)
			}

			for _, event := range certevents {

				layout := time.RFC3339Nano
				t, err := time.Parse(layout, event.Data.TimeStamp)
				var unixEpoch int64 = 0
				if err == nil {
					unixEpoch = t.Unix()
				}
				unixEpochStr := strconv.FormatInt(unixEpoch, 10)
				log.Infof("Namespace %s, Name: %s, Event Type: %s, Timestamp %s\n", event.Data.Namespace, event.Data.Name, event.Data.Type, unixEpochStr)
				log.Infof("Disable cert match check %v\n", env.disableCertMatchChecks)
				updateEvent := (strings.ToLower(event.Data.Type) == "create") || (strings.ToLower(event.Data.Type) == "update")
				eventNameSpace := strings.Trim(event.Data.Namespace, " ")
				eventName := strings.Trim(event.Data.Name, " ")
				if (env.disableCertMatchChecks == true) || ((eventNameSpace == env.certificateNameSpace) && (eventName == env.autoCertName) && updateEvent) {

					sleepTimeOut := env.podFileUpdateSleepTimeout // We've seen this take over 1 minute
					sleepInterval := 200
					timeOutExceeded := false
					sleepTime := 0
					log.Infof("Waiting for cert secret files to update in pod....")
					for (!haveCertFilesUpdated(true)) && (!timeOutExceeded) {
						time.Sleep(time.Duration(sleepInterval) * time.Millisecond)
						sleepTime += sleepInterval
						w.WriteHeader(http.StatusProcessing) //Send feedback to botKube
						if sleepTime > sleepTimeOut {
							timeOutExceeded = true
							log.Warnf("Timed out waiting for secret files to update in pod")
						}
					}
					log.Infof("Pod cert secret files to updated.")
					//https://aws.github.io/aws-sdk-go-v2/docs/configuring-sdk/#static-credentials

					var certs certChain

					certs, err := readCertFiles(certs)
					if err != nil {
						msg := "Failed to read certificate  files."
						log.Errorf(msg+", err: %v", err)
						httpResponse{acceptedContent: accContent, status: http.StatusInternalServerError, message: msg}.write(w)
						return
					}

					if env.createK8sCertSecret {
						createK8sCertificateSecret(certs)
					}

					// Load the custom AWS configuration with the provided credentials
					cfg, err := config.LoadDefaultConfig(context.TODO(),
						config.WithRegion(env.region),
						//config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(awsAccessKey, awsSecretKey, "")),
					)

					if err != nil {
						msg := "Unable to load AWS SDK"
						log.Errorf(msg+", %v", err)
						httpResponse{acceptedContent: accContent, status: http.StatusInternalServerError, message: msg}.write(w)
						return
					}

					if (env.awsAccessKey != "") && (env.awsSecretKey != "") {
						cfg.Credentials = credentials.NewStaticCredentialsProvider(env.awsAccessKey, env.awsSecretKey, "")
					}

					// Create an ACM client
					svc := acm.NewFromConfig(cfg)

					log.Infof("Searching for ARN of certificate with tag name " + env.acmCertificateName + " in ACM.")
					// Find the certificate by name
					certArn, err := findCertificateByName(svc, env.acmCertificateName)
					if err != nil {
						msg := "Failed to find certificate by name  " + env.acmCertificateName
						log.Errorf(msg+", err: %v", err)
						httpResponse{acceptedContent: accContent, status: http.StatusInternalServerError, message: msg}.write(w)
						return
					}

					log.Infof("ARN of certificate with tag name " + env.acmCertificateName + " is " + certArn)
					log.Infof("Existing certificate found, using PEM files to update in ACM.")

					log.Infof("Trying to update certArn Certificate: " + certArn)
					if certArn != "" {
						// Update existing certificate
						input := &acm.ImportCertificateInput{
							Certificate:    certs.tlsCrt,
							PrivateKey:     certs.tlsKey,
							CertificateArn: aws.String(certArn),
							//CertificateChain: certChain,
						}

						if certs.caCrtChain != nil {
							input.CertificateChain = certs.caCrtChain
						}

						result, err := svc.ImportCertificate(context.TODO(), input)
						if err != nil {
							msg := "Failed to import certificate"
							log.Errorf(msg+", err: %v", err)
							httpResponse{acceptedContent: accContent, status: http.StatusInternalServerError, message: msg}.write(w)
							return
						}
						log.Infof("Successfully updated certificate: %s\n", aws.ToString(result.CertificateArn))
						msg := "Successfully updated existing certificate in ACM, cert ARN = " + certArn
						httpResponse{acceptedContent: accContent, status: http.StatusOK, message: msg}.write(w)
					} else {
						if env.importIntoACMIfNotExists {
							log.Infof("No existing certificate found, Importing new certificate into ACM.")
							// Import new certificate
							input := &acm.ImportCertificateInput{
								Certificate: certs.tlsCrt,
								PrivateKey:  certs.tlsKey,
								//CertificateChain: certs.caCrt,
							}

							if certs.caCrtChain != nil {
								input.CertificateChain = certs.caCrtChain
							}

							result, err := svc.ImportCertificate(context.TODO(), input)
							if err != nil {
								msg := "Failed to import certificate"
								log.Errorf(msg+", err: %v", err)
								httpResponse{acceptedContent: accContent, status: http.StatusInternalServerError, message: msg}.write(w)
								return
							}

							log.Infof("Successfully imported new certificate: %s\n", aws.ToString(result.CertificateArn))

							// Add tags to the new certificate to identify it by name and add a comment
							tagInput := &acm.AddTagsToCertificateInput{
								CertificateArn: result.CertificateArn,
								Tags: []types.Tag{
									{
										Key:   aws.String("Name"),
										Value: aws.String(env.acmCertificateName),
									},
									{
										Key:   aws.String("Comment"),
										Value: aws.String(IMPORT_COMMENT),
									},
									{
										Key:   aws.String("Version"),
										Value: aws.String(env.version),
									},
								},
							}

							_, err = svc.AddTagsToCertificate(context.TODO(), tagInput)
							if err != nil {
								msg := "Failed to add tags to certificate"
								log.Errorf(msg+", err: %v", err)
								httpResponse{acceptedContent: accContent, status: http.StatusInternalServerError, message: msg}.write(w)
								return
							}
							log.Infof("Successfully added tags to imported certificate: %s\n", aws.ToString(result.CertificateArn))
							certArn = aws.ToString(result.CertificateArn)
							msg := "Successfully imported certificate into ACM, cert ARN = " + certArn
							httpResponse{acceptedContent: accContent, status: http.StatusOK, message: msg}.write(w)
						} else {
							msg := "Skipped import of certificate due to ACM_IMPORT_IF_NOT_EXISTS set to : " + boolToString(env.importIntoACMIfNotExists)
							log.Infof(msg)
							httpResponse{acceptedContent: accContent, status: http.StatusOK, message: msg}.write(w)
						}

					} //end import

				} else {
					msg := "Ignoring Certificate update event due to parameters mismatch"
					log.Warnf(msg)
					log.Warnf("Certificate event must be 'update' or 'create', got : %v event from botKube\n", (strings.ToLower(event.Data.Type)))
					log.Warnf("Certificate K8 namespace in event and configuration must match. Configured value : %v got %v from botKube \n", env.certificateNameSpace, event.Data.Namespace)
					log.Warnf("Certificate name in event and configuration must match. Configured value : %v got %v from botKube \n", env.autoCertName, event.Data.Name)
					httpResponse{acceptedContent: accContent, status: http.StatusInternalServerError, message: msg}.write(w)
					return
				}
				//httpResponse{acceptedContent: JSON_CONTENT, status: http.StatusOK, message: MSG_200_OK}.write(w)
			}
		} else {
			http.Error(w, "Body does not contain", http.StatusInternalServerError)
		}
		break
	case "GET":
		log.Warnf("Disallowed http GET method call")
		httpResponse{acceptedContent: accContent, status: http.StatusMethodNotAllowed, message: MSG_405_METHOD}.write(w)
		break
	case "DEFAULT":
		log.Warnf("Disallowed http method call")
		httpResponse{acceptedContent: accContent, status: http.StatusMethodNotAllowed, message: MSG_405_METHOD}.write(w)
		break
	}

}

func DeleteCert(w http.ResponseWriter, r *http.Request) {
	var accContent = r.Header.Get(HEADER_ACCEPT)

	log.Debugf("Response content type: |%s|\n", accContent)
	log.Debugf("AWS Region: %s\n", env.region)
	log.Debugf("Certificate Name: %s\n", env.acmCertificateName)
	log.Debugf("Certificate File path: %s\n", env.tlsCrtPath)
	log.Debugf("Private key File: %s\n", env.tlsKeyPath)
	log.Debugf("CA Certificate file path: %s\n", env.caCrtPath)

	//https://aws.github.io/aws-sdk-go-v2/docs/configuring-sdk/#static-credentials

	switch r.Method {
	case "POST":
		log.Infof("POST Request /updatecert at %v\n", time.Now())

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Error reading body", http.StatusInternalServerError)
			return
		}

		if len(body) > 0 {
			log.Infof("Input data: %s\n", string(body[:]))

			var certevent CertEvent
			var certevents []CertEvent

			err := json.Unmarshal([]byte(body), &certevent) //try parse a single certevent
			if err != nil {
				err := json.Unmarshal([]byte(body), &certevents) //fallback to array of events
				if err != nil {
					log.Errorf("Error parsing JSON %s", err)
				}
			} else {
				// If the single record parsing succeeds, append it to the certevents slice
				certevents = append(certevents, certevent)
			}

			for _, event := range certevents {

				layout := time.RFC3339Nano
				t, err := time.Parse(layout, event.Data.TimeStamp)
				var unixEpoch int64 = 0
				if err == nil {
					unixEpoch = t.Unix()
				}
				unixEpochStr := strconv.FormatInt(unixEpoch, 10)
				log.Infof("Namespace %s, Name: %s, Event Type: %s, Timestamp %s\n", event.Data.Namespace, event.Data.Name, event.Data.Type, unixEpochStr)
				updateEvent := (strings.ToLower(event.Data.Type) == "delete")
				if (env.disableCertMatchChecks == true) || ((event.Data.Namespace == env.certificateNameSpace) && (event.Data.Name == env.autoCertName) && updateEvent) {

					// Load the custom AWS configuration with the provided credentials
					cfg, err := config.LoadDefaultConfig(context.TODO(),
						config.WithRegion(env.region),
						//config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(awsAccessKey, awsSecretKey, "")),
					)

					if err != nil {
						msg := "Unable to load AWS SDK"
						log.Errorf(msg+", %v", err)
						httpResponse{acceptedContent: accContent, status: http.StatusInternalServerError, message: msg}.write(w)
						return
					}

					if (env.awsAccessKey != "") && (env.awsSecretKey != "") {
						cfg.Credentials = credentials.NewStaticCredentialsProvider(env.awsAccessKey, env.awsSecretKey, "")
					}

					// Create an ACM client
					svc := acm.NewFromConfig(cfg)

					if env.acmCertificateName == "" {
						msg := "Required Certificate name parameter missing."
						log.Errorf(msg)
						httpResponse{acceptedContent: accContent, status: http.StatusServiceUnavailable, message: msg}.write(w)
						return
					}

					certArn, err := findCertificateByName(svc, env.acmCertificateName)
					if err != nil {
						msg := "An error occurred trying to find existing certificate"
						log.Errorf(msg+", err: %v", err)
						httpResponse{acceptedContent: accContent, status: http.StatusServiceUnavailable, message: msg}.write(w)
						return
					}

					if certArn == "" {
						msg := "Certificate with name '" + env.acmCertificateName + "' not found."
						log.Errorf(msg)
						log.Errorf("Unable to find existing certificate: %v", err)
						httpResponse{acceptedContent: accContent, status: http.StatusServiceUnavailable, message: msg}.write(w)
						return
					}

					input := &acm.DeleteCertificateInput{
						CertificateArn: aws.String(certArn),
					}

					_, err = svc.DeleteCertificate(context.TODO(), input)
					if err != nil {
						msg := "An error occurred trying to delete existing certificate"
						log.Errorf(msg+", err: %v", err)
						httpResponse{acceptedContent: accContent, status: http.StatusServiceUnavailable, message: msg}.write(w)
						return
					}

					msg := "Successfully deleted certificate, ARN=" + certArn
					log.Infof(msg)

					httpResponse{acceptedContent: accContent, status: http.StatusOK, message: msg}.write(w)

				} else {
					msg := "Certificate event must be delete"
					log.Warnf("Certificate event must be delete, got : %v\n", (strings.ToLower(event.Data.Type)))
					log.Warnf("Certificate K8 namespace in event and config must match. Need : %v  got %v \n", env.certificateNameSpace, event.Data.Namespace)
					log.Warnf("Certificate name in event and config must match. Need : %v  got %v \n", env.acmCertificateName, event.Data.Name)
					httpResponse{acceptedContent: accContent, status: http.StatusInternalServerError, message: msg}.write(w)
					return
				}
				//httpResponse{acceptedContent: JSON_CONTENT, status: http.StatusOK, message: MSG_200_OK}.write(w)
			}
		} else {
			http.Error(w, "Body does not contain", http.StatusInternalServerError)
		}
		break
	case "GET":
		log.Warnf("Disallowed http GET method call")
		httpResponse{acceptedContent: accContent, status: http.StatusMethodNotAllowed, message: MSG_405_METHOD}.write(w)
		break
	case "DEFAULT":
		log.Warnf("Disallowed http method call")
		httpResponse{acceptedContent: accContent, status: http.StatusMethodNotAllowed, message: MSG_405_METHOD}.write(w)
		break
	}

}

func httpGetCertPEM(requestURL string) (string, error) {
	var retval = ""
	log.Infof("Requesting data from %v", requestURL)
	res, err := http.Get(requestURL)
	if err != nil {
		log.Errorf("error making http request to %v: %s\n", requestURL, err)
		return retval, err
	}
	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		log.Errorf("client: could not read HTTP response body from %v: %s\n", requestURL, err)
		return retval, err
	}
	retval = string(resBody[:])
	log.Infof("Returning retreived body from %v", requestURL)
	return retval, nil
}

func doInititalCertUpdate() {

	log.Debugf("AWS Region: %s\n", env.region)
	log.Debugf("K8 Certificate Namespace: %s\n", env.certificateNameSpace)
	log.Debugf("Certificate Name: %s\n", env.acmCertificateName)
	log.Debugf("Certificate File path: %s\n", env.tlsCrtPath)
	log.Debugf("Private key File: %s\n", env.tlsKeyPath)
	log.Debugf("CA Certificate file path: %s\n", env.caCrtPath)

	var certs certChain
	certs, err := readCertFiles(certs)
	if err != nil {
		msg := "Failed to read certificate  files."
		log.Errorf(msg+", err: %v", err)
		return
	}

	if env.createK8sCertSecret {
		createK8sCertificateSecret(certs)
	}

	//https://aws.github.io/aws-sdk-go-v2/docs/configuring-sdk/#static-credentials

	// Load the custom AWS configuration with the provided credentials

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(env.region),
	)

	if err != nil {
		msg := "Unable to load AWS SDK"
		log.Errorf(msg+", %v", err)
		return
	}

	if (env.awsAccessKey != "") && (env.awsSecretKey != "") {
		cfg.Credentials = credentials.NewStaticCredentialsProvider(env.awsAccessKey, env.awsSecretKey, "")
	}

	// Create an ACM client
	svc := acm.NewFromConfig(cfg)

	log.Infof("Searching for ARN of certificate with tag name " + env.acmCertificateName + " in ACM.")
	// Find the certificate by name
	log.Infof("Search for existing cert in ACM with name: %s\n", env.acmCertificateName)
	certArn, err := findCertificateByName(svc, env.acmCertificateName)
	if err != nil {
		msg := "Failed to find certificate by name  " + env.acmCertificateName
		log.Errorf(msg+", err: %v", err)
		return
	}

	log.Infof("ARN of certificate with tag name " + env.acmCertificateName + " is " + certArn)
	log.Infof("Existing certificate found, using PEM files to update in ACM.")

	log.Infof("Trying to update certArn Certificate: " + certArn)
	if certArn != "" {
		// Update existing certificate
		input := &acm.ImportCertificateInput{
			Certificate:    certs.tlsCrt,
			PrivateKey:     certs.tlsKey,
			CertificateArn: aws.String(certArn),
		}

		if certs.caCrtChain != nil {
			input.CertificateChain = certs.caCrtChain
		}

		//JOL should be able to remove all code up to here and use certs instead of certChain
		result, err := svc.ImportCertificate(context.TODO(), input)
		if err != nil {
			msg := "Failed to import certificate"
			log.Errorf(msg+", err: %v", err)
			return
		}
		log.Infof("Successfully updated certificate: %s\n", aws.ToString(result.CertificateArn))
	} else {
		log.Infof("No existing certificate found, Importing new certificate into ACM.")
		// Import new certificate
		input := &acm.ImportCertificateInput{
			Certificate: certs.tlsCrt,
			PrivateKey:  certs.tlsKey,
			//CertificateChain: certs.caCrtChain,
		}

		if certs.caCrtChain != nil {
			input.CertificateChain = certs.caCrtChain
		}

		result, err := svc.ImportCertificate(context.TODO(), input)
		if err != nil {
			msg := "Failed to import certificate"
			log.Errorf(msg+", err: %v", err)
			return
		}
		log.Infof("Successfully imported new certificate: %s\n", aws.ToString(result.CertificateArn))

		// Add tags to the new certificate to identify it by name and add a comment
		tagInput := &acm.AddTagsToCertificateInput{
			CertificateArn: result.CertificateArn,
			Tags: []types.Tag{
				{
					Key:   aws.String("Name"),
					Value: aws.String(env.acmCertificateName),
				},
				{
					Key:   aws.String("Comment"),
					Value: aws.String("Imported by Golang SDK"),
				},
			},
		}

		_, err = svc.AddTagsToCertificate(context.TODO(), tagInput)
		if err != nil {
			msg := "Failed to add tags to certificate"
			log.Errorf(msg+", err: %v", err)
			return
		}
		log.Infof("Successfully added tags to imported certificate: %s\n", aws.ToString(result.CertificateArn))
		certArn = aws.ToString(result.CertificateArn)
	}
}

func UpdateCertWithoutChecks(w http.ResponseWriter, r *http.Request) {
	var accContent = strings.ToLower(r.Header.Get(HEADER_ACCEPT))

	log.Debugf("Response content type: |%s|\n", accContent)
	log.Debugf("AWS Region: %s\n", env.region)
	log.Debugf("K8 Certificate Namespace: %s\n", env.certificateNameSpace)
	log.Debugf("Certificate Name: %s\n", env.acmCertificateName)
	log.Debugf("Certificate File path: %s\n", env.tlsCrtPath)
	log.Debugf("Private key File: %s\n", env.tlsKeyPath)
	log.Debugf("CA Certificate file path: %s\n", env.caCrtPath)

	//https://aws.github.io/aws-sdk-go-v2/docs/configuring-sdk/#static-credentials

	sleepTimeOut := env.podFileUpdateSleepTimeout // We've seen this take over 1 minute
	sleepInterval := 200
	timeOutExceeded := false
	sleepTime := 0
	log.Infof("Waiting for cert secret files to update in pod....")
	for (!haveCertFilesUpdated(true)) && (!timeOutExceeded) {
		time.Sleep(time.Duration(sleepInterval) * time.Millisecond)
		sleepTime += sleepInterval
		w.WriteHeader(http.StatusProcessing) //Send feedback to botKube
		if sleepTime > sleepTimeOut {
			timeOutExceeded = true
			log.Warnf("Timed out waiting for secret files to update in pod")
		}
	}
	log.Infof("Pod cert secret files to updated.")

	var certs certChain
	certs, err := readCertFiles(certs)
	if err != nil {
		msg := "Failed to read certificate  files."
		log.Errorf(msg+", err: %v", err)
		httpResponse{acceptedContent: accContent, status: http.StatusInternalServerError, message: msg}.write(w)
		return
	}

	// Load the custom AWS configuration with the provided credentials

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(env.region),
		//config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(awsAccessKey, awsSecretKey, "")),
	)

	if err != nil {
		msg := "Unable to load AWS SDK"
		log.Errorf(msg+", %v", err)
		httpResponse{acceptedContent: accContent, status: http.StatusInternalServerError, message: msg}.write(w)
		return
	}

	if (env.awsAccessKey != "") && (env.awsSecretKey != "") {
		cfg.Credentials = credentials.NewStaticCredentialsProvider(env.awsAccessKey, env.awsSecretKey, "")
	}

	//svc, err := InitializeACMClient(context.TODO())

	// Create an ACM client
	svc := acm.NewFromConfig(cfg)

	// Find the certificate by name
	log.Infof("Searching for ARN of certificate with tag name " + env.acmCertificateName + " in ACM.")
	certArn, err := findCertificateByName(svc, env.acmCertificateName)
	if err != nil {
		msg := "Failed to find certificate by name  " + env.acmCertificateName
		log.Errorf(msg+", err: %v", err)
		httpResponse{acceptedContent: accContent, status: http.StatusInternalServerError, message: msg}.write(w)
		return
	}

	log.Infof("ARN of certificate with tag name " + env.acmCertificateName + " is " + certArn)
	log.Infof("Existing certificate found, using PEM files to update in ACM.")

	log.Infof("Trying to update certArn Certificate: " + certArn)

	if certArn != "" {
		// Update existing certificate
		input := &acm.ImportCertificateInput{
			Certificate:    certs.tlsCrt,
			PrivateKey:     certs.tlsKey,
			CertificateArn: aws.String(certArn),
			//CertificateChain: certChain,
		}

		if certs.caCrtChain != nil {
			input.CertificateChain = certs.caCrtChain
		}

		result, err := svc.ImportCertificate(context.TODO(), input)
		if err != nil {
			msg := "Failed to import certificate"
			log.Errorf(msg+", err: %v", err)
			httpResponse{acceptedContent: accContent, status: http.StatusInternalServerError, message: msg}.write(w)
			return
		}
		log.Infof("Successfully updated certificate: %s\n", aws.ToString(result.CertificateArn))
		msg := "Successfully updated existing certificate in ACM, cert ARN = " + certArn
		httpResponse{acceptedContent: accContent, status: http.StatusOK, message: msg}.write(w)
	} else {
		if env.importIntoACMIfNotExists {
			log.Infof("No existing certificate found, Importing new certificate into ACM.")
			// Import new certificate
			input := &acm.ImportCertificateInput{
				Certificate: certs.tlsCrt,
				PrivateKey:  certs.tlsKey,
				//CertificateChain: certs.caCrt,
			}

			if certs.caCrtChain != nil {
				input.CertificateChain = certs.caCrtChain
			}

			result, err := svc.ImportCertificate(context.TODO(), input)
			if err != nil {
				msg := "Failed to import certificate"
				log.Errorf(msg+", err: %v", err)
				httpResponse{acceptedContent: accContent, status: http.StatusInternalServerError, message: msg}.write(w)
				return
			}

			log.Infof("Successfully imported new certificate: %s\n", aws.ToString(result.CertificateArn))

			// Add tags to the new certificate to identify it by name and add a comment
			tagInput := &acm.AddTagsToCertificateInput{
				CertificateArn: result.CertificateArn,
				Tags: []types.Tag{
					{
						Key:   aws.String("Name"),
						Value: aws.String(env.acmCertificateName),
					},
					{
						Key:   aws.String("Comment"),
						Value: aws.String(IMPORT_COMMENT),
					},
					{
						Key:   aws.String("Version"),
						Value: aws.String(env.version),
					},
				},
			}

			_, err = svc.AddTagsToCertificate(context.TODO(), tagInput)
			if err != nil {
				msg := "Failed to add tags to certificate"
				log.Errorf(msg+", err: %v", err)
				httpResponse{acceptedContent: accContent, status: http.StatusInternalServerError, message: msg}.write(w)
				return
			}
			log.Infof("Successfully added tags to imported certificate: %s\n", aws.ToString(result.CertificateArn))
			certArn = aws.ToString(result.CertificateArn)
			msg := "Successfully imported certificate into ACM, cert ARN = " + certArn
			httpResponse{acceptedContent: accContent, status: http.StatusOK, message: msg}.write(w)
		} else {
			msg := "Skipped import of certificate due to ACM_IMPORT_IF_NOT_EXISTS set to : " + boolToString(env.importIntoACMIfNotExists)
			log.Infof(msg)
			httpResponse{acceptedContent: accContent, status: http.StatusOK, message: msg}.write(w)
		}
	} //end of import

	//httpResponse{acceptedContent: JSON_CONTENT, status: http.StatusOK, message: MSG_200_OK}.write(w)
}

func DeleteCertWithoutChecks(w http.ResponseWriter, r *http.Request) {

	var accContent = r.Header.Get(HEADER_ACCEPT)

	log.Debugf("Response content type: |%s|\n", accContent)
	log.Debugf("AWS Region: %s\n", env.region)
	log.Debugf("Certificate Name: %s\n", env.acmCertificateName)
	log.Debugf("Certificate File path: %s\n", env.tlsCrtPath)
	log.Debugf("Private key File: %s\n", env.tlsKeyPath)
	log.Debugf("CA Certificate file path: %s\n", env.caCrtPath)

	//https://aws.github.io/aws-sdk-go-v2/docs/configuring-sdk/#static-credentials

	// Load the custom AWS configuration with the provided credentials
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(env.region),
		//config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(awsAccessKey, awsSecretKey, "")),
	)

	if err != nil {
		msg := "Unable to load AWS SDK"
		log.Errorf(msg+", %v", err)
		httpResponse{acceptedContent: accContent, status: http.StatusInternalServerError, message: msg}.write(w)
		return
	}

	if (env.awsAccessKey != "") && (env.awsSecretKey != "") {
		cfg.Credentials = credentials.NewStaticCredentialsProvider(env.awsAccessKey, env.awsSecretKey, "")
	}

	// Create an ACM client
	svc := acm.NewFromConfig(cfg)

	if env.acmCertificateName == "" {
		msg := "Required Certificate name parameter missing."
		log.Errorf(msg)
		httpResponse{acceptedContent: accContent, status: http.StatusServiceUnavailable, message: msg}.write(w)
		return
	}

	certArn, err := findCertificateByName(svc, env.acmCertificateName)
	if err != nil {
		msg := "An error occurred trying to find existing certificate"
		log.Errorf(msg+", err: %v", err)
		httpResponse{acceptedContent: accContent, status: http.StatusServiceUnavailable, message: msg}.write(w)
		return
	}

	if certArn == "" {
		msg := "Certificate with name '" + env.acmCertificateName + "' not found."
		log.Errorf(msg)
		log.Errorf("Unable to find existing certificate: %v", err)
		httpResponse{acceptedContent: accContent, status: http.StatusServiceUnavailable, message: msg}.write(w)
		return
	}

	input := &acm.DeleteCertificateInput{
		CertificateArn: aws.String(certArn),
	}

	_, err = svc.DeleteCertificate(context.TODO(), input)
	if err != nil {
		msg := "An error occurred trying to delete existing certificate"
		log.Errorf(msg+", err: %v", err)
		httpResponse{acceptedContent: accContent, status: http.StatusServiceUnavailable, message: msg}.write(w)
		return
	}

	msg := "Successfully deleted certificate, ARN=" + certArn
	log.Infof(msg)

	httpResponse{acceptedContent: accContent, status: http.StatusOK, message: msg}.write(w)
}

func getOrCreateHostedZone(svc *route53.Client, domain string, vpc string, region string, isPrivate bool, createZone bool) (string, error) {
	// List hosted zones and check if the domain exists

	listZonesInput := &route53.ListHostedZonesByNameInput{
		DNSName: aws.String(domain),
	}
	listZonesOutput, err := svc.ListHostedZonesByName(context.TODO(), listZonesInput)
	if err != nil {
		log.Errorf("Error listing Hosted Zones %s", err)
		return "", err
	}

	for _, zone := range listZonesOutput.HostedZones {
		if (strings.TrimSuffix(*zone.Name, ".") == domain) && (*&zone.Config.PrivateZone == isPrivate) {
			log.Infof("Found hosted zome %s for %s, private zone=%s", *zone.Id, domain, boolToString(isPrivate))
			return *zone.Id, nil
		}
	}

	// We don't create zones for delete actions
	// If not found and IsPrivate=false, create the hosted zone
	// to create a private zone, we need the region, which we have, and VPC which we don't have

	if createZone && ((!isPrivate) || (isPrivate && region != "" && vpc != "")) {
		createZoneInput := &route53.CreateHostedZoneInput{
			Name:            aws.String(domain),
			CallerReference: aws.String(fmt.Sprintf("%d", time.Now().UnixNano())),
		}

		if isPrivate {
			// Set the VPC parameter after initializing the params variable
			createZoneInput.VPC = &route53types.VPC{VPCId: aws.String(vpc), VPCRegion: route53types.VPCRegion(region)}
			createZoneInput.HostedZoneConfig = &route53types.HostedZoneConfig{PrivateZone: true}
		}

		createZoneOutput, err := svc.CreateHostedZone(context.TODO(), createZoneInput)
		if err != nil {
			log.Errorf("Error creating Hosted Zone %s", err)
			return "", err
		}

		log.Infof("Created hosted zone %s for %s, private zone=%s", *createZoneOutput.HostedZone.Id, domain, boolToString(isPrivate))

		return *createZoneOutput.HostedZone.Id, nil
	}
	return "", fmt.Errorf("Unable to find or create hosted zone")
}

func getRoute53Record(svc *route53.Client, hostedZoneID, fqdn string, recordType string) (*route53types.ResourceRecordSet, error) {

	if (hostedZoneID == "") || (fqdn == "") {
		return nil, fmt.Errorf("Required parameters missing.")
	}

	input := &route53.ListResourceRecordSetsInput{
		HostedZoneId:    aws.String(hostedZoneID),
		StartRecordName: aws.String(strings.ToLower(fqdn)),
		StartRecordType: route53types.RRType(recordType),
		//StartRecordType: types.RRTypeA,
	}

	log.Infof("Getting %v type DNS records for %v\n", recordType, fqdn)

	result, err := svc.ListResourceRecordSets(context.TODO(), input)
	if err != nil {
		log.Errorf("Error retreiving records or no Route53 record found %s", err)
		return nil, err
	}

	log.Infof("Search for existing record : " + fqdn + " of type " + recordType)

	for _, recordSet := range result.ResourceRecordSets {
		log.Infof("List Existing Record : " + *recordSet.Name)
		//if strings.TrimSuffix(*recordSet.Name, ".")  == recordName  && recordSet.Type == types.RRTypeA {
		if strings.TrimSuffix(*recordSet.Name, ".") == strings.ToLower(fqdn) && recordSet.Type == route53types.RRType(recordType) {
			log.Infof("Found Matching Existing Record : " + *recordSet.Name + " of type " + (string(recordSet.Type)))
			return &recordSet, nil
		}
	}

	return nil, nil
}

func createRoute53Record(svc *route53.Client, hostedZoneID, recordName, recordType string, recordValue string) error {
	input := &route53.ChangeResourceRecordSetsInput{
		HostedZoneId: aws.String(hostedZoneID),
		ChangeBatch: &route53types.ChangeBatch{
			Changes: []route53types.Change{
				{
					Action: route53types.ChangeActionCreate,
					ResourceRecordSet: &route53types.ResourceRecordSet{
						Name: aws.String(recordName),
						//Type: types.RRTypeA,
						Type: route53types.RRType(strings.ToUpper(recordType)),
						TTL:  aws.Int64(300),
						ResourceRecords: []route53types.ResourceRecord{
							{
								Value: aws.String(recordValue),
							},
						},
					},
				},
			},
		},
	}

	_, err := svc.ChangeResourceRecordSets(context.TODO(), input)
	return err
}

func updateRoute53Record(svc *route53.Client, hostedZoneID, recordName, recordType string, newValue string) error {
	input := &route53.ChangeResourceRecordSetsInput{
		HostedZoneId: aws.String(hostedZoneID),
		ChangeBatch: &route53types.ChangeBatch{
			Changes: []route53types.Change{
				{
					Action: route53types.ChangeActionUpsert,
					ResourceRecordSet: &route53types.ResourceRecordSet{
						Name: aws.String(recordName),
						Type: route53types.RRType(strings.ToUpper(recordType)),
						//Type: types.RRTypeA,
						TTL: aws.Int64(300),
						ResourceRecords: []route53types.ResourceRecord{
							{
								Value: aws.String(newValue),
							},
						},
					},
				},
			},
		},
	}

	_, err := svc.ChangeResourceRecordSets(context.TODO(), input)
	return err
}

func deleteRoute53Record(svc *route53.Client, hostedZoneID, recordName string, recordType string) error {
	existingRecord, err := getRoute53Record(svc, hostedZoneID, recordName, recordType)
	if err != nil {
		return err
	}
	if existingRecord == nil {
		log.Warnf("No existing record found to delete")
		return nil
	}

	input := &route53.ChangeResourceRecordSetsInput{
		HostedZoneId: aws.String(hostedZoneID),
		ChangeBatch: &route53types.ChangeBatch{
			Changes: []route53types.Change{
				{
					Action: route53types.ChangeActionDelete,
					ResourceRecordSet: &route53types.ResourceRecordSet{
						Name: aws.String(recordName),
						Type: route53types.RRType(recordType),
						//Type: types.RRTypeA,
						TTL: aws.Int64(300),
						ResourceRecords: []route53types.ResourceRecord{
							{
								Value: existingRecord.ResourceRecords[0].Value,
							},
						},
					},
				},
			},
		},
	}

	_, err = svc.ChangeResourceRecordSets(context.TODO(), input)
	return err
}

func POSTDebug(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		log.Infof("POST Request /debug at %v\n", time.Now())
		for k, v := range r.Header {
			log.Infof("%v: %v\n", k, v)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Error reading body", http.StatusInternalServerError)
			return
		}

		if len(body) > 0 {
			log.Infof("Input data: %s\n", string(body[:]))

			var certevent CertEvent
			var certevents []CertEvent

			err := json.Unmarshal([]byte(body), &certevent) //try parse a single certevent
			if err != nil {
				err := json.Unmarshal([]byte(body), &certevents) //fallback to array of events
				if err != nil {
					log.Errorf("Error parsing JSON %s", err)
				}
			} else {
				// If the single record parsing succeeds, append it to the certevents slice
				certevents = append(certevents, certevent)
			}

			for _, event := range certevents {

				layout := time.RFC3339Nano
				t, err := time.Parse(layout, event.Data.TimeStamp)
				var unixEpoch int64 = 0
				if err == nil {
					unixEpoch = t.Unix()
				}
				unixEpochStr := strconv.FormatInt(unixEpoch, 10)
				log.Infof("Namespace %s, Name: %s, Event Type: %s, Timestamp %s\n", event.Data.Namespace, event.Data.Name, event.Data.Type, unixEpochStr)
			}
		}

		//fmt.Println("POST data: ", string(body))
		contentType := r.Header.Get(HEADER_ACCEPT)
		contentType = strings.TrimSpace(contentType)
		contentType = strings.Trim(contentType, "\"") //Remove any quotes
		log.Infof("Formatted content-type: %s\n", contentType)

		w.Header().Set(HEADER_CONTENT, contentType)
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, string(body))

	case "GET":
		log.Infof("GET Request /debug at %v\n", time.Now())
		for k, v := range r.Header {
			log.Infof("%v: %v\n", k, v)
		}

	}

}

/*
GetDebug is a function used for debugging and testing. Based on the request url parameters it can be instructed
to return any status that GetLicense is capable of returning.
This function is not included in release builds.
*/
func GetDebug(w http.ResponseWriter, r *http.Request) {
	var params debugParams
	var accContent = r.Header.Get(HEADER_ACCEPT)
	var delay = 0 * time.Second
	var err error

	switch r.Method {
	case "GET":
		for key := range r.URL.Query() {
			switch strings.ToLower(key) {
			case "reply":
				params.replyWith = r.URL.Query().Get(key)
			case "delay":
				params.delay = r.URL.Query().Get(key)
			}
		}

		if err := params.validate(); err != nil {
			log.Warnf("Parameter validation error: %v", err)
			httpResponse{acceptedContent: accContent, status: http.StatusBadRequest, message: MSG_400_BAD_RQ}.write(w)
			return
		}

		if params.delay != "" {
			delay, err = time.ParseDuration(params.delay + "s")
			if err != nil {
				log.Warnf("Bad debug delay: %v", err)
				httpResponse{acceptedContent: accContent, status: http.StatusBadRequest, message: MSG_400_BAD_RQ}.write(w)
				return
			}
		} else {
			params.delay = "0"
		}

		if params.replyWith != "" {
			log.Infof("Debug Request to reply %s, with delay of %s seconds", params.replyWith, params.delay)
			time.Sleep(delay)

			switch params.replyWith {
			case "200":
				httpResponse{acceptedContent: accContent, status: http.StatusOK, message: MSG_200_OK}.write(w)
			case "400":
				httpResponse{acceptedContent: accContent, status: http.StatusBadRequest, message: MSG_400_BAD_RQ}.write(w)
			case "401":
				httpResponse{
					acceptedContent: accContent,
					status:          http.StatusUnauthorized,
					message:         MSG_401_UNAUTH,
					errWebpagePath:  ERROR_WEBPAGE,
					errWebpageCmt:   "example debug error",
					errorUuid:       uuid.New().String(),
				}.write(w)
			case "403":
				httpResponse{acceptedContent: accContent, status: http.StatusForbidden, message: MSG_403_FORBID}.write(w)
			case "405":
				httpResponse{acceptedContent: accContent, status: http.StatusMethodNotAllowed, message: MSG_405_METHOD}.write(w)
			case "503":
				httpResponse{acceptedContent: accContent, status: http.StatusServiceUnavailable, message: MSG_503_UNAVAL}.write(w)
			default:
				log.Warnf("Bad debug request: %s", params.replyWith)
				httpResponse{acceptedContent: accContent, status: http.StatusBadRequest, message: MSG_400_BAD_RQ}.write(w)
			}
			return
		}

		httpResponse{acceptedContent: accContent, status: http.StatusBadRequest, message: MSG_400_BAD_RQ}.write(w)

	default:
		log.Warnf("Disallowed http method call")
		httpResponse{acceptedContent: accContent, status: http.StatusMethodNotAllowed, message: MSG_405_METHOD}.write(w)
	}
}

// findCertificateByName searches for a certificate with a specific name tag and ECDSA 256 key type
func findCertificateByName(svc *acm.Client, name string) (string, error) {
	input := &acm.ListCertificatesInput{
		CertificateStatuses: []types.CertificateStatus{
			types.CertificateStatusIssued,
			types.CertificateStatusInactive,
			types.CertificateStatusExpired,
		},
		Includes: &types.Filters{
			KeyTypes: []types.KeyAlgorithm{
				types.KeyAlgorithmEcPrime256v1,
				types.KeyAlgorithmEcSecp384r1, // https://docs.aws.amazon.com/acm/latest/userguide/acm-certificate.html
				types.KeyAlgorithmEcSecp521r1,
				types.KeyAlgorithmRsa1024,
				types.KeyAlgorithmRsa2048, // Include other key types if needed
				types.KeyAlgorithmRsa3072,
				types.KeyAlgorithmRsa4096,
			},
		},
	}
	var certArn string

	//now := time.Now()
	//fmt.Println(now.UnixMilli())

	log.Debugf("Send request to AWS for cert list")
	paginator := acm.NewListCertificatesPaginator(svc, input)
	log.Debugf("Got response from AWS for cert list")
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			log.Errorf("Error iterating ACM ListCertificatesPaginator")
			return "", err
		}

		log.Infof("Processing page with %d certificates", len(page.CertificateSummaryList))

		for _, certSummary := range page.CertificateSummaryList {
			log.Infof("Found certificate ARN: %s", aws.ToString(certSummary.CertificateArn))

			// Describe the certificate to get its details
			describeInput := &acm.DescribeCertificateInput{
				CertificateArn: certSummary.CertificateArn,
			}

			describeOutput, err := svc.DescribeCertificate(context.TODO(), describeInput)
			if err != nil {
				log.Errorf("Failed to describe certificate: %v", err)
				continue //skip to next cert
				//return "Failed to describe certificate", err
			}

			// Check the key type of the certificate
			_ = describeOutput.Certificate
			//certDetails := describeOutput.Certificate

			tagInput := &acm.ListTagsForCertificateInput{
				CertificateArn: certSummary.CertificateArn,
			}
			tagResult, err := svc.ListTagsForCertificate(context.TODO(), tagInput)
			if err != nil {
				log.Errorf("failed to list tags for certificate: %v", err)
				continue //skip to next cert
				//return "", err
			}

			for _, tag := range tagResult.Tags {
				if aws.ToString(tag.Key) == "Name" && aws.ToString(tag.Value) == name {
					certArn = aws.ToString(certSummary.CertificateArn)
					return certArn, nil
				}
			}
		}
	}

	if certArn == "" {
		log.Infof("Certificate with name '%s' not found.", name)
	}

	return certArn, nil
}

// findCertificateByArn retrieves certificate details by its ARN
func findCertificateByArn(svc *acm.Client, arn string) (*types.CertificateDetail, error) {
	input := &acm.DescribeCertificateInput{
		CertificateArn: aws.String(arn),
	}

	result, err := svc.DescribeCertificate(context.TODO(), input)
	if err != nil {
		return nil, err
	}

	return result.Certificate, nil
}

func checkExpiringCertificates(svc *acm.Client, threshold int) error {
	var retVal string = ""
	input := &acm.ListCertificatesInput{
		CertificateStatuses: []types.CertificateStatus{
			types.CertificateStatusIssued,
			types.CertificateStatusInactive,
			types.CertificateStatusExpired,
		},
		Includes: &types.Filters{
			KeyTypes: []types.KeyAlgorithm{
				types.KeyAlgorithmEcPrime256v1,
				types.KeyAlgorithmEcSecp384r1, // https://docs.aws.amazon.com/acm/latest/userguide/acm-certificate.html
				types.KeyAlgorithmEcSecp521r1,
				types.KeyAlgorithmRsa1024,
				types.KeyAlgorithmRsa2048, // Include other key types if needed
				types.KeyAlgorithmRsa3072,
				types.KeyAlgorithmRsa4096,
			},
		},
	}

	//now := time.Now()
	//fmt.Println(now.UnixMilli())

	log.Infof("checkExpiringCertificates()------>start")

	log.Debugf("Send request to AWS for cert list")
	paginator := acm.NewListCertificatesPaginator(svc, input)
	log.Debugf("Got response from AWS for cert list")
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			log.Errorf("Error iterating ACM ListCertificatesPaginator")
			return err
		}

		log.Infof("Processing page with %d certificates", len(page.CertificateSummaryList))

		for _, certSummary := range page.CertificateSummaryList {
			log.Infof("Found certificate ARN: %s", aws.ToString(certSummary.CertificateArn))

			// Describe the certificate to get its details
			describeInput := &acm.DescribeCertificateInput{
				CertificateArn: certSummary.CertificateArn,
			}

			describeOutput, err := svc.DescribeCertificate(context.TODO(), describeInput)
			if err != nil {
				log.Errorf("Failed to describe certificate: %v", err)
				return err
			}

			// Check the key type of the certificate
			//_ = describeOutput.Certificate
			certDetails := describeOutput.Certificate

			expiryDays := int(certDetails.NotAfter.Sub(time.Now()).Hours() / 24)
			log.Infof("Certificate expires in %v days, warning threshold is %v days.", expiryDays, threshold)
			if expiryDays < threshold {
				//Report that the cert is within the expiry threshold
				domain := certDetails.DomainName
				arn := certDetails.CertificateArn

				t := time.Now()
				time := (t.Format(time.RFC3339))
				level := "WARN"

				if expiryDays > 1 {
					level = "WARN"
					retVal = "Certficate :" + *domain + " with ARN " + *arn + " is about to expire"
					retVal = `{
						"level": "` + level + `",
						"event": "auditmessage",
						"time": "` + time + `",
						"caller": "ACM Certificate Healthcheck",
						"message": "` + retVal + `"
					}`
					log.Warnf(retVal)
				} else {
					level = "ERROR"
					retVal = "Certficate :" + *domain + " with ARN " + *arn + " has expired!!"
					retVal = `{
						"level": "` + level + `",
						"event": "auditmessage",
						"time": "` + time + `",
						"caller": "ACM Certificate Healthcheck",
						"message": "` + retVal + `"
					}`
					log.Errorf(retVal)
				}
			}
		}
	}
	log.Infof("checkExpiringCertificates()------>end")
	return nil
}

// deleteCertificateByName deletes a certificate by its name tag if it exists
func deleteCertificateByName(svc *acm.Client, name string) error {
	certArn, err := findCertificateByName(svc, name)
	if err != nil {
		return err
	}

	if certArn == "" {
		fmt.Printf("Certificate with name '%s' not found.\n", name)
		return nil
	}

	input := &acm.DeleteCertificateInput{
		CertificateArn: aws.String(certArn),
	}

	_, err = svc.DeleteCertificate(context.TODO(), input)
	if err != nil {
		return err
	}

	fmt.Printf("Successfully deleted certificate: %s\n", certArn)
	return nil
}

// InitializeACMClient initializes and returns an ACM client
func InitializeACMClient(ctx context.Context) (*acm.Client, error) {
	var httpcfg config.LoadOptionsFunc

	customClient, err := ConfigureHTTPClient()
	if err == nil {
		httpcfg = config.WithHTTPClient(customClient)
	}

	cfg, err := config.LoadDefaultConfig(ctx, httpcfg)
	if err != nil {
		return nil, err
	}

	if env.region == "" {
		return nil, fmt.Errorf("AWS Region is not set in environment variable AWS_REGION")
	} else {
		cfg.Region = env.region
	}

	if (env.awsAccessKey != "") && (env.awsSecretKey != "") {
		cfg.Credentials = credentials.NewStaticCredentialsProvider(env.awsAccessKey, env.awsSecretKey, "")
	}

	//cfg.Credentials = credentials.NewStaticCredentialsProvider(env.awsAccessKey, env.awsSecretKey, "")

	return acm.NewFromConfig(cfg), nil
}

// ConfigureHTTPClient sets up an HTTP client with a proxy, respecting NO_PROXY environment variable
func ConfigureHTTPClient() (*http.Client, error) {
	proxyURL := os.Getenv("PROXY_URL")
	if proxyURL == "" {
		return nil, fmt.Errorf("PROXY_URL environment variable is not set")
	}

	noProxyHosts := strings.Split(os.Getenv("NO_PROXY"), ",")

	customTransport := &http.Transport{
		Proxy: func(req *http.Request) (*url.URL, error) {
			for _, host := range noProxyHosts {
				if strings.Contains(req.URL.Host, host) {
					return nil, nil
				}
			}
			return url.Parse(proxyURL)
		},
	}

	return &http.Client{
		Transport: customTransport,
	}, nil
}

// function to check if file exists
func doesFileExist(fileName string, displayinfo bool) bool {
	_, error := os.Stat(fileName)

	// check if error is "file not exists"
	if os.IsNotExist(error) {
		if displayinfo {
			log.Errorf("ERROR: The file %v does not exist!!\n", fileName)
		}
		return false
	} else {
		if displayinfo {
			log.Infof("INFO: %v file exists\n", fileName)
		}
		return true
	}
}

func TrimSuffix(s, suffix string) string {
	if strings.HasSuffix(s, suffix) {
		s = s[:len(s)-len(suffix)]
	}
	return s
}

func escapeString(s string) string {
	if strings.HasPrefix(s, "\"") && strings.HasSuffix(s, "\"") {
		// already escaped, return unchanged
		fmt.Println("Returning unchanged")
		return s
	}
	return "\"" + s + "\""
}

/*
doesProcessExist uses go-ps to search /proc/<pid>/stat for a binary name (truncated)
Returns false by default if the supply process name is not found.
*/
func doesProcessExist(processname string) bool {
	processList, err := ps.Processes()
	if err != nil {
		log.Errorf("ps.Processes() failed: %v", err)
		return false
	}

	var process ps.Process
	for i := range processList {
		process = processList[i]
		if process.Executable() == processname {
			return true
		}
	}
	return false
}

func shutdownServer() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	log.Infof("Shutting down HTTP server")
	if err := server.Shutdown(ctx); err != nil {
		log.Errorf("Error shutting down HTTP server: %v", err)
	} else {
		log.Infof("Shutdown of HTTP server complete")
	}
}

func (r httpResponse) write(writer http.ResponseWriter) {
	log.Infof("apiserver->httpResponse write()->start")
	log.Infof("Writing HTTP response")
	var (
		body        = []byte(html.EscapeString(r.message))
		contentType = PLAIN_CONTENT
		err         error
	)

	log.Infof("Setting content type: %v", r.acceptedContent)

	acceptedContent := strings.TrimSpace(r.acceptedContent)
	acceptedContent = strings.Trim(acceptedContent, "\"") //Remove any quotes

	if acceptedContent == strings.ToLower(JSON_CONTENT) {
		log.Infof("Building JSON response")

		if r.errorUuid != "" {
			r.message = r.message + " ID:" + r.errorUuid
		}

		jsonResponse := &JsonResponse{Message: r.message}

		if body, err = json.Marshal(jsonResponse); err != nil {
			log.Errorf("Error marshalling JSON response: %v", err)
			log.Errorf("Attempting plain text response")
			body = []byte(r.message)
		} else {
			contentType = strings.ToLower(JSON_CONTENT)
		}

	} else if r.errWebpagePath != "" {
		log.Debugf("Building webpage response")

		if body, err = os.ReadFile(r.errWebpagePath); err != nil {
			log.Errorf("Error reading HTML file: %v", err)
			log.Errorf("Attempting plain text response")
			body = []byte(r.message)
		} else {
			contentType = HTML_CONTENT

			if r.errorUuid != "" {
				body = bytes.Replace(body, []byte("UUIDSTR"), []byte(r.errorUuid), 1)
			}

			if r.errWebpageCmt != "" {
				errWebpageCmt := sanitizeString(r.errWebpageCmt)
				body = append(body, []byte("<!--\n"+errWebpageCmt+"\n-->")...)
			}
		}
	}

	writer.Header().Set(HEADER_CONTENT, contentType)
	writer.WriteHeader(r.status)

	if _, err = writer.Write(body); err != nil {
		log.Errorf("Error writing response body: %v", err)
	}

	log.Infof("apiserver->httpResponse write()->end")
}

func sanitizeString(str string) string {
	return regexp.MustCompile(`[^a-zA-Z0-9 ]+`).ReplaceAllString(str, " ")
}
