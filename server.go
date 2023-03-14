package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	token "github.com/m-lab/epoxy-extensions/allocate_k8s_token"
	bmc "github.com/m-lab/epoxy-extensions/bmc_store_password"
	"github.com/m-lab/epoxy-extensions/handler"
	"github.com/m-lab/epoxy-extensions/metrics"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	fBinDir        string
	fListenAddress string
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
	flag.Parse()

	kc := &token.K8sCommand{}
	args := []string{
		"token", "create", "--ttl", "5m", "--print-join-command",
		"--description", "Allow machine to join the cluster",
	}
	k8sToken := token.New(fBinDir, kc, args)
	bmcPassword := bmc.New()

	http.HandleFunc("/", rootHandler)
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/v1/allocate_k8s_token",
		promhttp.InstrumentHandlerDuration(metrics.TokenRequestDuration,
			handler.NewTokenHandler("v1", k8sToken)))

	http.HandleFunc("/v2/allocate_k8s_token",
		promhttp.InstrumentHandlerDuration(metrics.TokenRequestDuration,
			handler.NewTokenHandler("v2", k8sToken)))

	http.HandleFunc("/v1/bmc_store_password",
		promhttp.InstrumentHandlerDuration(metrics.BMCRequestDuration,
			handler.NewBmcHandler(bmcPassword)))

	log.Printf("Listening on interface: %s", fListenAddress)
	log.Fatal(http.ListenAndServe(fListenAddress, nil))
}
