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

package bmc_store_password

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/m-lab/epoxy/extension"
	"github.com/m-lab/go/host"
)

type fakePassword struct {
	password string
}

func (p *fakePassword) Store(hostname string, password string) error {
	_, err := host.Parse(hostname)
	if err != nil {
		return fmt.Errorf("Bad hostname")
	}
	if p.password == "" {
		return fmt.Errorf("No password was provided")
	}
	return nil
}

func Test_passwordHandler(t *testing.T) {
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
			name:     "failure-bad-request",
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
			name:   "failure-no-password-provided",
			method: "POST",
			v1: &extension.V1{
				Hostname:    "mlab1-foo01.mlab-oti.measurement-lab.org",
				IPv4Address: "192.168.1.1",
				LastBoot:    time.Now().UTC().Add(-5 * time.Minute),
				RawQuery:    "y=rofl&z=lol",
			},
			status: http.StatusBadRequest,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			localPassword = &fakePassword{tt.password}
			ext := extension.Request{V1: tt.v1}
			req := httptest.NewRequest(
				tt.method, "/v1/bmc-store-password?p="+tt.password, strings.NewReader(ext.Encode()))
			rec := httptest.NewRecorder()

			passwordHandler(rec, req)

			if tt.status != rec.Code {
				t.Errorf("passwordHandler() bad status code: got %d; want %d",
					rec.Code, tt.status)
			}
		})
	}
}
