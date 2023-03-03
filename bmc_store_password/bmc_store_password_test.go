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
	"testing"

	"github.com/m-lab/go/host"
)

type fakePassword struct{}

func (p *fakePassword) Store(hostname string, password string) error {
	_, err := host.Parse(hostname)
	if err != nil {
		return fmt.Errorf("could not parse hostname: %s", hostname)
	}
	return nil
}

// It is debatable whether this unit test is even worthwhile. There is not much
// to test in bmc_store_password, as almost the entire functionality of the
// package is hangled by github.com/m-lab/reboot-service/creds. About the only
// thing to test is whether Store() properly returns an error when it receives a
// malformed M-Lab hostname. The snipped of code in fakePassword.Store() was
// copied directly from the real Store() function.
func Test_Store(t *testing.T) {
	tests := []struct {
		name     string
		hostname string
		wantErr  bool
	}{
		{
			name:     "success",
			hostname: "mlab1-foo01.mlab-oti.measurement-lab.org",
			wantErr:  false,
		},
		{
			name:     "failure-bad-hostname",
			hostname: "lol-foo01.mlab-oti.measurement-lab.org",
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fp := &fakePassword{}
			err := fp.Store(tt.hostname, "")
			if (err != nil) != tt.wantErr {
				t.Errorf("Store(): want err %v, got %v", tt.wantErr, err)
			}
		})
	}
}
