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

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/m-lab/epoxy/extension"
)

var (
	testAPIAddress string = "api.example.com:6443"
	testCAHash     string = "sha256:hash"
	testToken      string = "012345.abcdefghijklmnop"
)

type fakeRunCommand struct {
	result string
}

func (c *fakeRunCommand) Command(prog string, args ...string) ([]byte, error) {
	if c.result == "" {
		return nil, fmt.Errorf("command failed")
	}
	return []byte(c.result), nil
}

type fakeTokenGenerator struct {
	response tokenResponse
	token    string
	version  string
}

func (g *fakeTokenGenerator) Token(target string) error {
	if g.response.Token == "" {
		return fmt.Errorf("failed to generate token")
	}
	g.response.Token = g.token
	return nil
}

func (g *fakeTokenGenerator) Response(version string) ([]byte, error) {
	if g.version == "v1" {
		return []byte(g.token), nil
	}
	return json.Marshal(g.response)
}

func Test_allocateTokenHandler(t *testing.T) {
	tests := []struct {
		expect  string
		method  string
		name    string
		status  int
		token   string
		v1      *extension.V1
		version string
	}{
		{
			name:   "success-v1",
			method: "POST",
			v1: &extension.V1{
				Hostname:    "mlab1-foo01.mlab-oti.measurement-lab.org",
				IPv4Address: "192.168.1.1",
				LastBoot:    time.Now().UTC().Add(-5 * time.Minute),
			},
			status:  http.StatusOK,
			token:   testToken,
			expect:  testToken,
			version: "v1",
		},
		{
			name:   "success-v2",
			method: "POST",
			v1: &extension.V1{
				Hostname:    "mlab1-foo01.mlab-oti.measurement-lab.org",
				IPv4Address: "192.168.1.1",
				LastBoot:    time.Now().UTC().Add(-5 * time.Minute),
			},
			status:  http.StatusOK,
			token:   testToken,
			expect:  `{"api_address":"` + testAPIAddress + `","token":"` + testToken + `","ca_hash":"` + testCAHash + `"}`,
			version: "v2",
		},
		{
			name:   "failure-bad-method",
			method: "GET",
			status: http.StatusMethodNotAllowed,
		},
		{
			name:   "failure-bad-requested",
			method: "POST",
			v1:     nil,
			status: http.StatusBadRequest,
		},
		{
			name:   "failure-last-boot-too-old",
			method: "POST",
			v1: &extension.V1{
				Hostname:    "mlab1-foo01.mlab-oti.measurement-lab.org",
				IPv4Address: "192.168.1.1",
				LastBoot:    time.Now().UTC().Add(-125 * time.Minute),
			},
			status: http.StatusRequestTimeout,
		},
		{
			name:   "failure-failure-to-generate-token",
			method: "POST",
			v1: &extension.V1{
				Hostname:    "mlab1-foo01.mlab-oti.measurement-lab.org",
				IPv4Address: "192.168.1.1",
				LastBoot:    time.Now().UTC().Add(-5 * time.Minute),
			},
			status: http.StatusInternalServerError,
			token:  "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			localGenerator = &fakeTokenGenerator{
				response: tokenResponse{
					APIAddress: testAPIAddress,
					CAHash:     testCAHash,
					Token:      tt.token,
				},
				token:   tt.token,
				version: tt.version,
			}
			ext := extension.Request{V1: tt.v1}
			req := httptest.NewRequest(
				tt.method, "/allocate_k8s_token", strings.NewReader(ext.Encode()))
			rec := httptest.NewRecorder()

			allocateTokenHandler(rec, req, tt.version)

			// v1 response should be a simple string, while a v2 response should
			// be JSON and have a content-type to support it.
			if tt.status == http.StatusOK {
				ct := rec.Result().Header["Content-Type"][0]
				if tt.version == "v1" {
					if ct != "text/plain; charset=utf-8" {
						t.Errorf("Expected Content-Type of text/plain, but got %v", ct)
					}
				} else {
					if ct != "application/json; charset=utf-8" {
						t.Errorf("Expected Content-Type of application/json, but got %v", ct)
					}

				}
			}
			if tt.status != rec.Code {
				t.Errorf("allocateTokenHandler() bad status code: got %d; want %d",
					rec.Code, tt.status)
			}
			if rec.Body.String() != tt.expect {
				t.Errorf("allocateTokenHandler() bad token returned: got %q; want %q",
					rec.Body.String(), tt.token)
			}
		})
	}
}

func Test_k8sTokenGenerator_Response(t *testing.T) {
	tests := []struct {
		name    string
		expect  string
		version string
		wantErr bool
	}{
		{
			name:    "success-v1",
			expect:  testToken,
			version: "v1",
			wantErr: false,
		},
		{
			name:    "success-v2",
			expect:  `{"api_address":"` + testAPIAddress + `","token":"` + testToken + `","ca_hash":"` + testCAHash + `"}`,
			version: "v2",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &k8sTokenGenerator{
				TokenResponse: tokenResponse{
					APIAddress: testAPIAddress,
					CAHash:     testCAHash,
					Token:      testToken,
				},
			}

			resp, err := g.Response(tt.version)
			r := string(resp)
			if (err != nil) != tt.wantErr {
				t.Errorf("k8sTokenGenerator.Response(): error = %v, wantErr %v", err, tt.wantErr)
			}
			if r != tt.expect {
				t.Errorf("k8sTokenGenerator.Response() = %#v, want %#v", r, tt.expect)
			}
		})
	}
}

func Test_k8sTokenGenerator_Token(t *testing.T) {
	tests := []struct {
		name    string
		expect  tokenResponse
		result  string
		wantErr bool
	}{
		{
			name: "success",
			expect: tokenResponse{
				APIAddress: "api.example.com:6443",
				CAHash:     "sha256:hash",
				Token:      "testtoken",
			},
			result:  "kubeadm join api.example.com:6443 --token testtoken --discovery-token-ca-cert-hash sha256:hash",
			wantErr: false,
		},
		{
			name:    "fail-command-error",
			result:  "",
			wantErr: true,
		},
		{
			name:    "fail-too-few-fields",
			result:  "kubeadm join api.example.com:6443 --discovery-token-ca-cert-hash sha256:hash",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			localCommander = &fakeRunCommand{
				result: tt.result,
			}
			g := &k8sTokenGenerator{}
			err := g.Token("test")
			if (err != nil) != tt.wantErr {
				t.Errorf("k8sTokenGenerator.Token(): error = %v, wantErr %v", err, tt.wantErr)
			}
			if g.TokenResponse != tt.expect {
				t.Errorf("k8sTokenGenerator.Token() = %q, want %q", g.TokenResponse, tt.expect)
			}
		})
	}
}
