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

type handler struct {
	extension   string
	bmcPassword bmc_store_password.Password
	k8sToken    allocate_k8s_token.TokenGenerator
	version     string
}

func (h *handler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	// TODO: verify this is from a trusted source (admin or epoxy source)
	// else return HTTP 401 (Unauthorized) and fire an alert (since this should
	// never happen)

	// Require requests to be POSTs.
	if req.Method != http.MethodPost {
		resp.WriteHeader(http.StatusMethodNotAllowed)
		// Write no response.
		return
	}

	// Decode the extension request.
	ext := &extension.Request{}
	err := ext.Decode(req.Body)
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

	log.Println("Request:", ext.Encode())

	switch h.extension {
	case "allocate_k8s_token":
		h.tokenHandler(resp, ext, h.version)
	case "bmc_store_password":
		h.passwordHandler(resp, ext)
	default:
		log.Printf("Unknown extension: %s", h.extension)
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	return
}

// tokenHandler
func (h *handler) tokenHandler(resp http.ResponseWriter, ext *extension.Request, version string) {
	var body []byte

	err := h.k8sToken.Create(ext.V1.Hostname)
	if err != nil {
		log.Println(err)
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	// A v1 response is just a string (the token), whereas a v2 response will be JSON.
	if version == "v1" {
		resp.Header().Set("Content-Type", "text/plain; charset=utf-8")
	} else {
		resp.Header().Set("Content-Type", "application/json; charset=utf-8")
	}

	body, err = h.k8sToken.Response(h.version)
	if err != nil {
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	resp.WriteHeader(http.StatusOK)
	resp.Write(body)
}

// passwordHandler
func (h *handler) passwordHandler(resp http.ResponseWriter, ext *extension.Request) {

	var reqPassword string

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

	err = h.bmcPassword.Store(ext.V1.Hostname, reqPassword)
	if err != nil {
		log.Println(err)
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	resp.WriteHeader(http.StatusOK)
}

func New(extension string, version string, k8sToken allocate_k8s_token.TokenGenerator, bmcPassword bmc_store_password.Password) http.Handler {
	return &handler{
		k8sToken:    k8sToken,
		bmcPassword: bmcPassword,
		extension:   extension,
		version:     version,
	}
}
