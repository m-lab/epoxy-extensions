package handler

import (
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/m-lab/epoxy-extensions/allocate_k8s_token"
	"github.com/m-lab/epoxy-extensions/bmc_store_password"
	"github.com/m-lab/epoxy/extension"
)

// k8sToken is the struct used to interact with the allocate_k8s_token package.
type k8sToken struct {
	generator allocate_k8s_token.TokenGenerator
	version   string
}

// ServeHTTP is the request handler for the allocate_k8s_token requests.
func (kt *k8sToken) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
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

	if time.Now().UTC().Sub(ext.V1.LastBoot) > 120*time.Minute {
		// According to ePoxy the machine booted over 2 hours ago,
		// which is longer than we're willing to support.
		resp.WriteHeader(http.StatusRequestTimeout)
		return
	}

	log.Println("Request:", ext.Encode())

	err = kt.generator.Create(ext.V1.Hostname)
	if err != nil {
		log.Println(err)
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	// A v1 response is just a string (the token), whereas a v2 response will be JSON.
	if kt.version == "v1" {
		resp.Header().Set("Content-Type", "text/plain; charset=utf-8")
	} else {
		resp.Header().Set("Content-Type", "application/json; charset=utf-8")
	}

	body, err = kt.generator.Response(kt.version)
	if err != nil {
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	resp.WriteHeader(http.StatusOK)
	resp.Write(body)
}

// bmcPassword is the struct used to interact with the bmc_store_password package.
type bmcPassword struct {
	password bmc_store_password.Password
}

// ServeHTTP is the request handler for the allocate_k8s_token requests.
func (bp *bmcPassword) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
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

	if time.Now().UTC().Sub(ext.V1.LastBoot) > 120*time.Minute {
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

	err = bp.password.Store(ext.V1.Hostname, reqPassword)
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

// NewK8sToken returns a new k8sToken object.
func NewK8sToken(version string, generator allocate_k8s_token.TokenGenerator) http.Handler {
	return &k8sToken{
		generator: generator,
		version:   version,
	}
}

// NewBmcPassword return a new bmcPassword object.
func NewBmcPassword(password bmc_store_password.Password) http.Handler {
	return &bmcPassword{
		password: password,
	}
}
