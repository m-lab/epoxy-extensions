// Copyright 2016 k8s-support Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//////////////////////////////////////////////////////////////////////////////

// The token_server implements the epoxy extension API and provides a way for
// machines booting with epoxy to allocate a k8s token, necessary for joining
// the cluster.
//
// To deploy the token_server, the ePoxy server must have an extension
// registered that maps an operation name to this server, e.g.:
//
//	"allocate_k8s_token" -> "http://localhost:8800/allocate_k8s_token"
package main

import (
	"encoding/json"
	"flag"
	"log"
	"math"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/m-lab/epoxy/extension"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	fKubeadmCommand string
	fPort           string

	// requestDuration provides a histogram of processing times. The buckets should
	// use periods that are intuitive for people.
	//
	// Provides metrics:
	//   token_server_request_duration_seconds{code="...", le="..."}
	//   ...
	//   token_server_request_duration_seconds{code="..."}
	//   token_server_request_duration_seconds{code="..."}
	// Usage example:
	requestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "token_server_request_duration_seconds",
			Help: "Request status codes and execution times.",
			Buckets: []float64{
				0.001, 0.01, 0.1, 1.0, 5.0, 10.0, 30.0, 60.0, 120.0, 300.0, math.Inf(+1),
			},
		},
		[]string{"method", "code"},
	)

	localGenerator tokenGenerator
)

func init() {
	flag.StringVar(&fKubeadmCommand, "command", "/usr/bin/kubeadm",
		"Absolute path to the kubeadm command used to create tokens")
	flag.StringVar(&fPort, "port", "8800",
		"Accept connection on this port.")
	prometheus.MustRegister(requestDuration)
}

// tokenGenerator defines the interface for creating tokens.
type tokenGenerator interface {
	Token(target string) error // Generate a new token.
	Response(version string) ([]byte, error)
}

type k8sTokenGenerator struct {
	Command       string
	TokenResponse tokenResponse
}

type tokenResponse struct {
	APIAddress string `json:"apiaddress"`
	Token      string `json:"token"`
	CAHash     string `json:"cahash"`
}

// Token generates a new k8s token.
func (g *k8sTokenGenerator) Token(target string) error {
	// Allocate the token for the given hostname.
	cmd := exec.Command(
		g.Command, "token", "create", "--ttl", "5m", "--print-join-command",
		"--description", "Allow "+target+" to join the cluster")
	output, err := cmd.Output()
	if err != nil {
		return err
	}
	fields := strings.Fields(string(output))
	g.TokenResponse.APIAddress = fields[2]
	g.TokenResponse.Token = fields[4]
	g.TokenResponse.CAHash = fields[6]

	return nil
}

func (g *k8sTokenGenerator) Response(version string) ([]byte, error) {
	if version == "v1" {
		return []byte(g.TokenResponse.Token), nil
	} else {
		return json.Marshal(g.TokenResponse)
	}
}

// allocateTokenHandler is an http.HandlerFunc for responding to an epoxy extension
// Request.
func allocateTokenHandler(w http.ResponseWriter, r *http.Request, v string) {
	var resp []byte

	// TODO: verify this is from a trusted source (admin or epoxy source)
	// else return HTTP 401 (Unauthorized) and fire an alert (since this should never happen)

	// Require requests to be POSTs.
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		// Write no response.
		return
	}

	// Decode the extension request.
	ext := &extension.Request{}
	err := ext.Decode(r.Body)
	if err != nil || ext.V1 == nil {
		log.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		// Write no response.
		return
	}
	if time.Now().UTC().Sub(ext.V1.LastBoot) > 120*time.Minute {
		// According to ePoxy the machine booted over 2 hours ago,
		// which is longer than we're willing to support.
		w.WriteHeader(http.StatusRequestTimeout)
		// Write no response.
		return
	}

	log.Println("Request:", ext.Encode())

	err = localGenerator.Token(ext.V1.Hostname)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		// Write no response.
		return
	}

	if v == "v1" {
		// Write response to caller.
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		resp, _ = localGenerator.Response(v)
	} else {
		// Write response to caller.
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		resp, err = localGenerator.Response(v)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}

func main() {
	flag.Parse()

	localGenerator = &k8sTokenGenerator{
		fKubeadmCommand,
		tokenResponse{},
	}

	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/v1/allocate_k8s_token",
		promhttp.InstrumentHandlerDuration(requestDuration, http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				allocateTokenHandler(w, r, "v1")
			})))

	http.HandleFunc("/v2/allocate_k8s_token",
		promhttp.InstrumentHandlerDuration(requestDuration, http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				allocateTokenHandler(w, r, "v2")
			})))
	log.Fatal(http.ListenAndServe(":"+fPort, nil))
}
