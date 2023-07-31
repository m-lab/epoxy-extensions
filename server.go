package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/m-lab/epoxy-extensions/bmc"
	"github.com/m-lab/epoxy-extensions/handler"
	"github.com/m-lab/epoxy-extensions/metrics"
	"github.com/m-lab/epoxy-extensions/node"
	"github.com/m-lab/epoxy-extensions/token"
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

	log.SetFlags(log.LUTC | log.LstdFlags | log.Lshortfile)

	tc := &token.TokenCommand{}
	tokenManager := token.New(fBinDir, tc)
	bmcPasswordStore := bmc.New()
	nodeManager := node.NewManager(fBinDir)

	http.HandleFunc("/", rootHandler)
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/v1/allocate_k8s_token",
		promhttp.InstrumentHandlerDuration(metrics.TokenRequestDuration,
			handler.NewTokenHandler("v1", tokenManager)))

	http.HandleFunc("/v2/allocate_k8s_token",
		promhttp.InstrumentHandlerDuration(metrics.TokenRequestDuration,
			handler.NewTokenHandler("v2", tokenManager)))

	http.HandleFunc("/v1/bmc_store_password",
		promhttp.InstrumentHandlerDuration(metrics.BMCRequestDuration,
			handler.NewBmcHandler(bmcPasswordStore)))

	http.HandleFunc("/v1/node/delete",
		promhttp.InstrumentHandlerDuration(metrics.NodeRequestDuration,
			handler.NewNodeHandler(nodeManager, "delete")))

	log.Printf("Listening on interface: %s", fListenAddress)
	log.Fatal(http.ListenAndServe(fListenAddress, nil))
}
