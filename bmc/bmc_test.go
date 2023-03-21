package bmc

import (
	"context"
	"fmt"
	"testing"

	"github.com/m-lab/reboot-service/creds"
)

// fakeCredsProviders implemenst the creds.Provider interface.
type fakeCredsProvider struct {
	wantErr bool
}

func (f fakeCredsProvider) AddCredentials(context.Context, string, *creds.Credentials) error {
	if f.wantErr {
		return fmt.Errorf("Error!")
	}
	return nil
}

func (f fakeCredsProvider) Close() error {
	return nil
}

func (f fakeCredsProvider) DeleteCredentials(context.Context, string) error {
	return nil
}

func (f fakeCredsProvider) FindCredentials(context.Context, string) (*creds.Credentials, error) {
	c := &creds.Credentials{}
	return c, nil
}

func (f fakeCredsProvider) ListCredentials(context.Context) ([]*creds.Credentials, error) {
	c := []*creds.Credentials{}
	return c, nil
}

// It is debatable whether this unit test is even worthwhile. There is not much
// to test in package bmc, as almost the entire functionality of the package is
// handled by github.com/m-lab/reboot-service/creds. About the only
// thing to test is whether Put() properly returns an error when it receives a
// malformed M-Lab hostname. The snippet of code in fakePasswordStore.Put() was
// copied directly from the real Put() function.
func Test_Put(t *testing.T) {
	tests := []struct {
		name         string
		hostname     string
		password     string
		wantErr      bool
		hostParseErr bool
		dnsErr       bool
		newCredsErr  bool
		addCredsErr  bool
	}{
		{
			name:     "success",
			hostname: "mlab1-foo01.mlab-oti.measurement-lab.org",
			password: "password",
		},
		{
			name:         "failure-invalid-mlab-hostname",
			hostname:     "lol-foo01.mlab-oti.measurement-lab.org",
			wantErr:      true,
			hostParseErr: true,
		},
		{
			name:     "failure-name-lookup",
			hostname: "mlab1-not01.mlab-oti.measurement-lab.org",
			wantErr:  true,
			dnsErr:   true,
		},
		{
			name:        "failure-new-creds-provider-error",
			hostname:    "mlab1-foo01.mlab-oti.measurement-lab.org",
			wantErr:     true,
			newCredsErr: true,
		},
		{
			name:        "failure-add-creds-error",
			hostname:    "mlab1-foo01.mlab-oti.measurement-lab.org",
			wantErr:     true,
			addCredsErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fc := fakeCredsProvider{
				wantErr: tt.wantErr,
			}
			ps := &gcdPasswordStore{}
			credsNewProvider = func(connector creds.Connector, projectID, namespace string) (creds.Provider, error) {
				if tt.newCredsErr {
					return nil, fmt.Errorf("Error!")
				}
				return fc, nil
			}
			netLookupHost = func(host string) (addrs []string, err error) {
				if tt.dnsErr {
					return nil, fmt.Errorf("Error!")
				}
				return []string{"192.168.0.1"}, nil
			}
			err := ps.Put(tt.hostname, tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("Put(): want err %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func Test_New(t *testing.T) {
	ps := New()
	var i interface{} = ps
	_, ok := i.(PasswordStore)
	if !ok {
		t.Errorf("New(): expected type PasswordStore, but got %T", ps)
	}
}
