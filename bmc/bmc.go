// bmc_store_password implements the epoxy extension API and provides a way for
// machines booting with epoxy to work with BMSs (iDRACs).
package bmc

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/m-lab/go/host"
	"github.com/m-lab/reboot-service/creds"
)

const (
	gcdNamespace = "reboot-api"
)

var (
	credsNewProvider = creds.NewProvider
	netLookupHost    = net.LookupHost
)

// PasswordStore defines the interface for storing BMC passwords.
type PasswordStore interface {
	Put(target string, password string) error
}

type gcdPasswordStore struct{}

// Put stores a BMC password in GCD.
func (g *gcdPasswordStore) Put(hostname string, password string) error {
	parts, err := host.Parse(hostname)
	if err != nil {
		return fmt.Errorf("could not parse hostname: %s", hostname)
	}

	bmcHostname := strings.Replace(hostname, parts.Machine, parts.Machine+"d", 1)

	bmcAddr, err := netLookupHost(bmcHostname)
	if err != nil {
		return fmt.Errorf("could not resolve BMC hostname: %s", bmcHostname)
	}

	c := &creds.Credentials{
		Address:  bmcAddr[0],
		Hostname: bmcHostname,
		Model:    "DRAC",
		Username: "admin",
		Password: password,
	}

	provider, err := credsNewProvider(&creds.DatastoreConnector{}, parts.Project, gcdNamespace)
	if err != nil {
		return fmt.Errorf("could not connect to Google Cloud Datastore: %v", err)
	}

	err = provider.AddCredentials(context.Background(), bmcHostname, c)
	if err != nil {
		return fmt.Errorf("error while adding credentials to GCD: %v", err)
	}

	return nil
}

// New returns a new PasswordStore.
func New() PasswordStore {
	return &gcdPasswordStore{}
}
