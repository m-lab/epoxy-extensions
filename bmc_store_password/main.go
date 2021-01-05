// Copyright 2020 M-Lab Authors
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

// bmc-store-password implements the epoxy extension API and provides a way for
// machines booting with epoxy to store the configured BMC password to GCD.
//
// To deploy the bmc-password, the ePoxy server must have an extension
// registered that maps an operation name to this server, e.g.:
//     "bmc-store-password" -> "http://localhost:8801/bmc-store-password"
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/m-lab/epoxy/extension"
	"github.com/m-lab/go/host"
	"github.com/m-lab/reboot-service/creds"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	gcdNamespace = "reboot-api"
)

var (
	credsNewProvider = creds.NewProvider
	fListenAddress   = flag.String("listen-address", ":8801", "Address on which to listen for requests")
	localPassword    password
	// requestDuration provides a histogram of processing times. The buckets should
	// use periods that are intuitive for people.
	//
	// Provides metrics:
	//   bmc_password_request_duration_seconds{code="...", le="..."}
	//   ...
	//   bmc_password_request_duration_seconds{code="..."}
	//   bmc_password_request_duration_seconds{code="..."}
	// Usage example:
	requestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "bmc_password_request_duration_seconds",
			Help: "Request status codes and execution times.",
			Buckets: []float64{
				0.001, 0.01, 0.1, 1.0, 5.0, 10.0, 30.0, 60.0, 120.0, 300.0, math.Inf(+1),
			},
		},
		[]string{"method", "code"},
	)
)

// password defines the interface for storing BMC passwords.
type password interface {
	Store(target string, password string) error
}

type bmcPassword struct{}

// Store stores a BMC password in GCD.
func (p *bmcPassword) Store(hostname string, password string) error {
	parts, err := host.Parse(hostname)
	if err != nil {
		return fmt.Errorf("Could not parse hostname: %s", hostname)
	}

	bmcHostname := strings.Replace(hostname, parts.Machine, parts.Machine+"d", 1)

	bmcAddr, err := net.LookupHost(bmcHostname)
	if err != nil {
		return fmt.Errorf("Could not resolve BMC hostname: %s", bmcHostname)
	}

	c := &creds.Credentials{
		Address:  bmcAddr[0],
		Hostname: bmcHostname,
		Model:    "DRAC",
		Username: "admin",
		Password: password,
	}

	provider, err := credsNewProvider(&creds.DatastoreConnector{}, parts.Project, gcdNamespace)
	if err != nil {
		return fmt.Errorf("Could not connect to Google Cloud Datastore: %v", err)
	}

	err = provider.AddCredentials(context.Background(), bmcHostname, c)
	if err != nil {
		return fmt.Errorf("Error while adding credentials to GCD: %v", err)
	}

	return nil
}

// passwordHandler is an http.HandlerFunc for responding to an ePoxy extension
// Request.
func passwordHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: verify this is from a trusted source (admin or epoxy source)
	// else return HTTP 401 (Unauthorized) and fire an alert (since this should never happen)

	var reqPassword string

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
		log.Println("The requesting machine booted more than two hours ago. Rejecting.")
		w.WriteHeader(http.StatusRequestTimeout)
		// Write no response.
		return
	}

	log.Println("Request: ", ext.Encode())

	// Parse query parameters from the request.
	queryParams, err := url.ParseQuery(ext.V1.RawQuery)
	if err != nil {
		log.Printf("Failed to parse RawQuery field: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		// Write no response.
		return
	}

	reqPassword = queryParams.Get("p")
	if reqPassword == "" {
		log.Println("Query parameter 'p' missing in request, or is empty.")
		w.WriteHeader(http.StatusBadRequest)
		// Write no response.
		return
	}

	err = localPassword.Store(ext.V1.Hostname, reqPassword)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		// Write no response.
		return
	}

	// Write response to caller.
	w.WriteHeader(http.StatusOK)
	return
}

func init() {
	prometheus.MustRegister(requestDuration)
}

func main() {
	flag.Parse()

	localPassword = &bmcPassword{}

	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/v1/bmc_store_password",
		promhttp.InstrumentHandlerDuration(
			requestDuration, http.HandlerFunc(passwordHandler)))
	log.Printf("Listening on interface: %s", *fListenAddress)
	log.Fatal(http.ListenAndServe(*fListenAddress, nil))
}
