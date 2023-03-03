// Copyright 2023 M-Lab Authors
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

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/m-lab/epoxy-extensions/allocate_k8s_token"
	"github.com/m-lab/epoxy-extensions/bmc_store_password"
	"github.com/m-lab/epoxy-extensions/handler"
	"github.com/m-lab/epoxy-extensions/metrics"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	fBinDir        string
	fListenAddress string

	mainCtx, mainCancel = context.WithCancel(context.Background())
)

// rootHandler implements the simplest possible handler for root requests,
// simply printing the name of the utility and returning a 200 status. This
// could be used by, for example, kubernetes aliveness checks.
func rootHandler(resp http.ResponseWriter, req *http.Request) {
	resp.WriteHeader(http.StatusOK)
	fmt.Fprintf(resp, "ePoxy Extensions")
}

func init() {
	flag.StringVar(&fBinDir, "bin-dir", "/usr/bin",
		"Absolute path to directory where required binaries are found.")
	flag.StringVar(&fListenAddress, "listen-address", ":8800",
		"Address on which to listen for requests.")
}

func main() {
	defer mainCancel()

	flag.Parse()

	k8sToken := allocate_k8s_token.New(fBinDir)
	bmcPassword := bmc_store_password.New()

	http.HandleFunc("/", rootHandler)
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/v1/allocate_k8s_token",
		promhttp.InstrumentHandlerDuration(metrics.TokenRequestDuration,
			handler.New("allocate_k8s_token", "v1", k8sToken, bmcPassword)))

	http.HandleFunc("/v2/allocate_k8s_token",
		promhttp.InstrumentHandlerDuration(metrics.TokenRequestDuration,
			handler.New("allocate_k8s_token", "v2", k8sToken, bmcPassword)))

	http.HandleFunc("/v1/bmc_store_password",
		promhttp.InstrumentHandlerDuration(metrics.BMCRequestDuration,
			handler.New("bmc_store_password", "v1", k8sToken, bmcPassword)))

	log.Printf("Listening on interface: %s", fListenAddress)
	log.Fatal(http.ListenAndServe(fListenAddress, nil))

}
