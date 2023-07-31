package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/m-lab/epoxy-extensions/node"
	"github.com/m-lab/epoxy-extensions/token"
	"github.com/m-lab/epoxy/extension"
	"github.com/m-lab/go/host"
)

var (
	testAPIAddress string = "api.example.com:6443"
	testCAHash     string = "sha256:hash"
	testToken      string = "012345.abcdefghijklmnop"
)

type fakeTokenManager struct {
	response token.Details
	token    string
	version  string
	wantErr  bool
}

func (ft *fakeTokenManager) Create(target string) error {
	if ft.response.Token == "" {
		return fmt.Errorf("failed to generate token")
	}
	ft.response.Token = ft.token
	return nil
}

func (ft *fakeTokenManager) Response(version string) ([]byte, error) {
	if ft.wantErr {
		return nil, fmt.Errorf("Error!")
	}
	if ft.version == "v1" {
		return []byte(ft.token), nil
	}
	return json.Marshal(ft.response)
}

func Test_tokenHandler(t *testing.T) {
	tests := []struct {
		expect  string
		method  string
		name    string
		status  int
		token   string
		v1      *extension.V1
		version string
		wantErr bool
	}{
		{
			name:   "success-v1",
			method: "POST",
			v1: &extension.V1{
				Hostname:    "mlab1-foo01.mlab-sandbox.measurement-lab.org",
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
				Hostname:    "mlab1-foo01.mlab-sandbox.measurement-lab.org",
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
				Hostname:    "mlab1-foo01.mlab-sandbox.measurement-lab.org",
				IPv4Address: "192.168.1.1",
				LastBoot:    time.Now().UTC().Add(-125 * time.Minute),
			},
			status: http.StatusRequestTimeout,
		},
		{
			name:   "failure-to-generate-token",
			method: "POST",
			v1: &extension.V1{
				Hostname:    "mlab1-foo01.mlab-sandbox.measurement-lab.org",
				IPv4Address: "192.168.1.1",
				LastBoot:    time.Now().UTC().Add(-5 * time.Minute),
			},
			status: http.StatusInternalServerError,
			token:  "",
		},
		{
			name:   "failure-json-marshall-error",
			method: "POST",
			v1: &extension.V1{
				Hostname:    "mlab1-foo01.mlab-sandbox.measurement-lab.org",
				IPv4Address: "192.168.1.1",
				LastBoot:    time.Now().UTC().Add(-5 * time.Minute),
			},
			status:  http.StatusInternalServerError,
			token:   testToken,
			version: "v2",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ft := &fakeTokenManager{
				response: token.Details{
					APIAddress: testAPIAddress,
					CAHash:     testCAHash,
					Token:      tt.token,
				},
				token:   tt.token,
				version: tt.version,
				wantErr: tt.wantErr,
			}
			th := NewTokenHandler(tt.version, ft)
			ext := extension.Request{V1: tt.v1}
			req := httptest.NewRequest(
				tt.method, "/allocate_k8s_token", strings.NewReader(ext.Encode()))
			rec := httptest.NewRecorder()

			th.ServeHTTP(rec, req)

			// v1 response should be a simple string, while a v2 response should
			// be JSON and have a content-type to support it.
			if tt.status == http.StatusOK {
				ct := rec.Result().Header["Content-Type"][0]
				if tt.version == "v1" {
					if ct != "text/plain; charset=utf-8" {
						t.Errorf("TokenHandler: expected Content-Type of text/plain, but got %v", ct)
					}
				} else {
					if ct != "application/json; charset=utf-8" {
						t.Errorf("TokenHandler: expected Content-Type of application/json, but got %v", ct)
					}

				}
			}
			if tt.status != rec.Code {
				t.Errorf("TokenHandler: bad status code: got %d; want %d",
					rec.Code, tt.status)
			}
			if rec.Body.String() != tt.expect {
				t.Errorf("TokenHandler: bad token returned: got %q; want %q",
					rec.Body.String(), tt.token)
			}
		})
	}
}

type fakePasswordStore struct{}

func (p *fakePasswordStore) Put(hostname string, password string) error {
	_, err := host.Parse(hostname)
	if err != nil {
		return fmt.Errorf("bad hostname")
	}
	return nil
}

func Test_bmcHandler(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		body     string
		v1       *extension.V1
		status   int
		password string
	}{
		{
			name:   "success",
			method: "POST",
			v1: &extension.V1{
				Hostname:    "mlab1-foo01.mlab-oti.measurement-lab.org",
				IPv4Address: "192.168.1.1",
				LastBoot:    time.Now().UTC().Add(-5 * time.Minute),
				RawQuery:    "p=somepass&z=lol",
			},
			status:   http.StatusOK,
			password: "012345abcdefghijklmnop",
		},
		{
			name:   "failure-bad-hostname",
			method: "POST",
			v1: &extension.V1{
				Hostname:    "lol-foo01.mlab-oti.measurement-lab.org",
				IPv4Address: "192.168.1.1",
				LastBoot:    time.Now().UTC().Add(-5 * time.Minute),
				RawQuery:    "p=somepass&z=lol",
			},
			status:   http.StatusInternalServerError,
			password: "012345abcdefghijklmnop",
		},
		{
			name:     "failure-bad-method",
			method:   "GET",
			status:   http.StatusMethodNotAllowed,
			password: "54321abcdefghijklmnop",
		},
		{
			name:     "failure-bad-request-v1-nil",
			method:   "POST",
			v1:       nil,
			status:   http.StatusBadRequest,
			password: "54321zyxwvu",
		},
		{
			name:   "failure-last-boot-too-old",
			method: "POST",
			v1: &extension.V1{
				Hostname:    "mlab1-foo01.mlab-oti.measurement-lab.org",
				IPv4Address: "192.168.1.1",
				LastBoot:    time.Now().UTC().Add(-125 * time.Minute),
				RawQuery:    "p=somepass&z=lol",
			},
			status:   http.StatusRequestTimeout,
			password: "testpassword",
		},
		{
			name:   "failure-invalid-query-with-semicolon",
			method: "POST",
			v1: &extension.V1{
				Hostname:    "mlab1-foo01.mlab-oti.measurement-lab.org",
				IPv4Address: "192.168.1.1",
				LastBoot:    time.Now().UTC().Add(-5 * time.Minute),
				RawQuery:    "p=somepass&;z=lol",
			},
			status:   http.StatusInternalServerError,
			password: "testpassword",
		},
		{
			name:   "failure-missing-query-param-p",
			method: "POST",
			v1: &extension.V1{
				Hostname:    "mlab1-foo01.mlab-oti.measurement-lab.org",
				IPv4Address: "192.168.1.1",
				LastBoot:    time.Now().UTC().Add(-5 * time.Minute),
				RawQuery:    "y=somepass&z=lol",
			},
			status:   http.StatusBadRequest,
			password: "testpassword",
		},
	}
	for _, tt := range tests {
		fp := &fakePasswordStore{}
		t.Run(tt.name, func(t *testing.T) {
			f := NewBmcHandler(fp)
			ext := extension.Request{V1: tt.v1}
			req := httptest.NewRequest(
				tt.method, "/v1/bmc-store-password?p="+tt.password, strings.NewReader(ext.Encode()))
			rec := httptest.NewRecorder()

			f.ServeHTTP(rec, req)

			if tt.status != rec.Code {
				t.Errorf("bmcPasswordStore: bad status code: got %d; want %d",
					rec.Code, tt.status)
			}
		})
	}
}

func Test_nodeHandler(t *testing.T) {
	tests := []struct {
		action  string
		command string
		method  string
		name    string
		status  int
		v1      *extension.V1
	}{
		{
			name:    "success",
			action:  "delete",
			command: "/bin/true",
			method:  "POST",
			v1: &extension.V1{
				Hostname: "mlab1-foo01.mlab-sandbox.measurement-lab.org",
				LastBoot: time.Now().UTC().Add(-5 * time.Minute),
			},
			status: http.StatusOK,
		},
		{
			name:    "failure-command-failed",
			action:  "delete",
			command: "/bin/doesnt/exist",
			method:  "POST",
			v1: &extension.V1{
				Hostname: "mlab1-foo01.mlab-sandbox.measurement-lab.org",
				LastBoot: time.Now().UTC().Add(-5 * time.Minute),
			},
			status: http.StatusInternalServerError,
		},
		{
			action: "bad-action",
			name:   "failure-bad-action",
			method: "POST",
			v1: &extension.V1{
				Hostname: "mlab1-foo01.mlab-oti.measurement-lab.org",
				LastBoot: time.Now().UTC().Add(-5 * time.Minute),
			},
			status: http.StatusInternalServerError,
		},
		{
			name:   "failure-bad-method",
			method: "GET",
			status: http.StatusMethodNotAllowed,
		},
		{
			name:   "failure-bad-request-v1-nil",
			method: "POST",
			v1:     nil,
			status: http.StatusBadRequest,
		},
		{
			name:   "failure-last-boot-too-old",
			method: "POST",
			v1: &extension.V1{
				Hostname: "mlab1-foo01.mlab-oti.measurement-lab.org",
				LastBoot: time.Now().UTC().Add(-125 * time.Minute),
			},
			status: http.StatusRequestTimeout,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nm := &node.Manager{
				Command: &node.Command{
					Path: tt.command,
				},
			}
			nh := NewNodeHandler(nm, tt.action)
			ext := extension.Request{V1: tt.v1}
			req := httptest.NewRequest(
				tt.method, "/v1/node/delete", strings.NewReader(ext.Encode()))
			rec := httptest.NewRecorder()

			nh.ServeHTTP(rec, req)

			if tt.status != rec.Code {
				t.Errorf("Nodeandler: bad status code: got %d; want %d",
					rec.Code, tt.status)
			}
		})
	}
}
