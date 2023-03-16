package token

import (
	"fmt"
	"testing"
)

var (
	testAPIAddress string = "api.example.com:6443"
	testCAHash     string = "sha256:hash"
	testToken      string = "012345.abcdefghijklmnop"
)

type fakeTokenCommand struct {
	result string
}

func (c *fakeTokenCommand) Command(prog string, args ...string) ([]byte, error) {
	if c.result == "" {
		return nil, fmt.Errorf("command failed")
	}
	return []byte(c.result), nil
}

// There are failure checks for Response(), since the only possible error would
// be returned by json.Marshal(), which for the most part won't return an error
// since the input is type constrained.
func Test_Response(t *testing.T) {
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
			g := &TokenManager{
				Details: Details{
					APIAddress: testAPIAddress,
					CAHash:     testCAHash,
					Token:      testToken,
				},
			}

			resp, err := g.Response(tt.version)
			r := string(resp)
			if (err != nil) != tt.wantErr {
				t.Errorf("Response(): error = %v, wantErr %v", err, tt.wantErr)
			}
			if r != tt.expect {
				t.Errorf("Response() = %#v, want %#v", r, tt.expect)
			}
		})
	}
}

func Test_Create(t *testing.T) {
	tests := []struct {
		name    string
		expect  Details
		result  string
		wantErr bool
	}{
		{
			name: "success",
			expect: Details{
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
			g := &TokenManager{
				Commander: &fakeTokenCommand{
					result: tt.result,
				},
			}
			err := g.Create("test-host")
			if (err != nil) != tt.wantErr {
				t.Errorf("Create(): error = %v, wantErr %v", err, tt.wantErr)
			}
			if g.Details != tt.expect {
				t.Errorf("Create() = %q, want %q", g.Details, tt.expect)
			}
		})
	}
}
