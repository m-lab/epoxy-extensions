package handler

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/m-lab/epoxy-extensions/bmc"
	"github.com/m-lab/epoxy-extensions/node"
	"github.com/m-lab/epoxy-extensions/token"
	"github.com/m-lab/epoxy/extension"
)

// The maximum amount of time since a machine has booted that extensions will
// accept requests from that host.
const maxUptime time.Duration = 120 * time.Minute

// tokenHandler implements the http.Handler interface and is the struct used to
// interact with the token package.
type tokenHandler struct {
	manager token.Manager
	version string
}

// ServeHTTP is the request handler for token requests.
func (t *tokenHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	var body []byte

	log.Printf("context %p: %s", req.Context(), req.RequestURI)

	// Require requests to be POSTs.
	if req.Method != http.MethodPost {
		resp.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	ext, err := decodeMessage(req)
	if err != nil || ext.V1 == nil {
		log.Printf("context %p: %v", req.Context(), err)
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	if time.Since(ext.V1.LastBoot) > maxUptime {
		// According to ePoxy the machine booted over 2 hours ago,
		// which is longer than we're willing to support.
		resp.WriteHeader(http.StatusRequestTimeout)
		return
	}

	log.Printf("context %p: %s", req.Context(), ext.Encode())

	err = t.manager.Create(ext.V1.Hostname)
	if err != nil {
		log.Printf("context %p: %v", req.Context(), err)
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	// A v1 response is just a string (the token), whereas a v2 response will be JSON.
	if t.version == "v1" {
		resp.Header().Set("Content-Type", "text/plain; charset=utf-8")
	} else {
		resp.Header().Set("Content-Type", "application/json; charset=utf-8")
	}

	body, err = t.manager.Response(t.version)
	if err != nil {
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	resp.WriteHeader(http.StatusOK)
	resp.Write(body)
}

// bmcHandler implements the http.Handler interface and is the struct used to
// interact with the bmc package.
type bmcHandler struct {
	passwordStore bmc.PasswordStore
}

// ServeHTTP is the request handler for token requests.
func (b *bmcHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	var reqPassword string

	log.Printf("context %p: %s", req.Context(), req.RequestURI)

	// Require requests to be POSTs.
	if req.Method != http.MethodPost {
		resp.WriteHeader(http.StatusMethodNotAllowed)
		// Write no response.
		return
	}

	ext, err := decodeMessage(req)
	if err != nil || ext.V1 == nil {
		log.Printf("context %p: %v", req.Context(), err)
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

	log.Printf("context %p: %s", req.Context(), ext.Encode())

	// Parse query parameters from the request.
	queryParams, err := url.ParseQuery(ext.V1.RawQuery)
	if err != nil {
		log.Printf("context %p: failed to parse RawQuery field: %v", req.Context(), err)
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	reqPassword = queryParams.Get("p")
	if reqPassword == "" {
		log.Printf("context %p: query parameter 'p' missing in request or empty", req.Context())
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	err = b.passwordStore.Put(ext.V1.Hostname, reqPassword)
	if err != nil {
		log.Printf("context %p: %v", req.Context(), err)
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	resp.WriteHeader(http.StatusOK)
}

// nodeHandler implements the http.Handler interface and is the struct used to
// interact with the node package.
type nodeHandler struct {
	manager *node.Manager
	action  string
}

// ServeHTTP is the request handler for node requests.
func (nh *nodeHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	log.Printf("context %p: %s", req.Context(), req.RequestURI)

	// Require requests to be POSTs.
	if req.Method != http.MethodPost {
		resp.WriteHeader(http.StatusMethodNotAllowed)
		// Write no response.
		return
	}

	ext, err := decodeMessage(req)
	if err != nil || ext.V1 == nil {
		log.Printf("context %p: %v", req.Context(), err)
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

	log.Printf("context %p: %s", req.Context(), ext.Encode())

	switch nh.action {
	case "delete":
		err = nh.manager.Delete(ext.V1.Hostname)
	default:
		log.Printf("context %p: unknown action '%s'", req.Context(), nh.action)
		err = fmt.Errorf("unknown node action '%s'", nh.action)
	}

	if err != nil {
		log.Printf("context %p: %v", req.Context(), err)
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
func NewTokenHandler(version string, manager token.Manager) http.Handler {
	return &tokenHandler{
		manager: manager,
		version: version,
	}
}

// NewBmcHandler returns a new bmcHandler, which implmements the
// http.Hanlder interface.
func NewBmcHandler(store bmc.PasswordStore) http.Handler {
	return &bmcHandler{
		passwordStore: store,
	}
}

// NewDeleteHandler returns a new deleteHandler, which implmements the
// http.Hanlder interface.
func NewNodeHandler(manager *node.Manager, action string) http.Handler {
	return &nodeHandler{
		manager: manager,
		action:  action,
	}
}
