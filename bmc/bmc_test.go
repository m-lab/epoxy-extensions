package bmc

import (
	"fmt"
	"testing"

	"github.com/m-lab/go/host"
)

type fakePasswordStore struct{}

func (p *fakePasswordStore) Put(hostname string, password string) error {
	_, err := host.Parse(hostname)
	if err != nil {
		return fmt.Errorf("could not parse hostname: %s", hostname)
	}
	return nil
}

// It is debatable whether this unit test is even worthwhile. There is not much
// to test in package bmc, as almost the entire functionality of the package is
// handled by github.com/m-lab/reboot-service/creds. About the only
// thing to test is whether Put() properly returns an error when it receives a
// malformed M-Lab hostname. The snippet of code in fakePasswordStore.Put() was
// copied directly from the real Put() function.
func Test_Put(t *testing.T) {
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
			fp := &fakePasswordStore{}
			err := fp.Put(tt.hostname, "")
			if (err != nil) != tt.wantErr {
				t.Errorf("Store(): want err %v, got %v", tt.wantErr, err)
			}
		})
	}
}
