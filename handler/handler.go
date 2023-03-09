package handler

import (
	"log"
	"net/http"
	"net/url"
	"time"

	token "github.com/m-lab/epoxy-extensions/allocate_k8s_token"
	bmc "github.com/m-lab/epoxy-extensions/bmc_store_password"
	"github.com/m-lab/epoxy/extension"
)

// The maximum amount of time since a machine has booted that extension will
// accept requests from that host.
const maxUptime time.Duration = 120 * time.Minute

// k8sToken implements the http.Handler interface and is the struct used to
// interact with the token package.
type tokenHandler struct {
	generator token.Generator
	version   string
}

// ServeHTTP is the request handler for allocate_k8s_token requests.
func (t *tokenHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	var body []byte

	// Require requests to be POSTs.
	if req.Method != http.MethodPost {
		resp.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	ext, err := decodeMessage(req)
	if err != nil || ext.V1 == nil {
		log.Println(err)
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	commandArgs := []string{
		"token", "create", "--ttl", "5m", "--print-join-command",
		"--description", "Allow " + ext.V1.Hostname + " to join the cluster",
	}

	if time.Since(ext.V1.LastBoot) > maxUptime {
		// According to ePoxy the machine booted over 2 hours ago,
		// which is longer than we're willing to support.
		resp.WriteHeader(http.StatusRequestTimeout)
		return
	}

	log.Println("Request:", ext.Encode())

	err = t.generator.Create(ext.V1.Hostname, commandArgs)
	if err != nil {
		log.Println(err)
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	// A v1 response is just a string (the token), whereas a v2 response will be JSON.
	if t.version == "v1" {
		resp.Header().Set("Content-Type", "text/plain; charset=utf-8")
	} else {
		resp.Header().Set("Content-Type", "application/json; charset=utf-8")
	}

	body, err = t.generator.Response(t.version)
	if err != nil {
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	resp.WriteHeader(http.StatusOK)
	resp.Write(body)
}

// bmcHandler implements the http.Handler interface and is the struct used to
// interact with the bmc_store_password package.
type bmcHandler struct {
	password bmc.Password
}

// ServeHTTP is the request handler for allocate_k8s_token requests.
func (b *bmcHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	var reqPassword string

	// Require requests to be POSTs.
	if req.Method != http.MethodPost {
		resp.WriteHeader(http.StatusMethodNotAllowed)
		// Write no response.
		return
	}

	ext, err := decodeMessage(req)
	if err != nil || ext.V1 == nil {
		log.Println(err)
		resp.WriteHeader(http.StatusBadRequest)
		// Write no response.
		return
	}

	if time.Since(ext.V1.LastBoot) > maxUptime {
		// According to ePoxy the machine booted over 2 hours ago,
		// which is longer than we're willing to support.
		resp.WriteHeader(http.StatusRequestTimeout)
		// Write no response.
		return
	}

	// Parse query parameters from the request.
	queryParams, err := url.ParseQuery(ext.V1.RawQuery)
	if err != nil {
		log.Printf("Failed to parse RawQuery field: %v", err)
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	reqPassword = queryParams.Get("p")
	if reqPassword == "" {
		log.Println("Query parameter 'p' missing in request, or is empty.")
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	err = b.password.Store(ext.V1.Hostname, reqPassword)
	if err != nil {
		log.Println(err)
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	resp.WriteHeader(http.StatusOK)
}

// decodeMessage takes and http request as input and returns the decoded
// extension request data.
func decodeMessage(req *http.Request) (*extension.Request, error) {
	ext := &extension.Request{}
	err := ext.Decode(req.Body)
	return ext, err
}

// NewTokenHandler returns a new tokenHandler, which implements the http.Handler
// interface.
func NewTokenHandler(version string, generator token.Generator) http.Handler {
	return &tokenHandler{
		generator: generator,
		version:   version,
	}
}

// NewBmcHandler returns a new bmcHandler, which implmements the
// http.Hanlder interface.
func NewBmcHandler(password bmc.Password) http.Handler {
	return &bmcHandler{
		password: password,
	}
}
