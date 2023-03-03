// Copyright 2020 M-Lab Authors
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

// bmc_store_password implements the epoxy extension API and provides a way for
// machines booting with epoxy to store the configured BMC password to GCD.
//
// To deploy bmc_store_password, the ePoxy server must have an extension
// registered that maps an operation name to this server, e.g.:
//
//	"bmc_store_password" -> "http://localhost:8801/bmc_store_password"
package bmc_store_password

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
)

// password defines the interface for storing BMC passwords.
type Password interface {
	Store(target string, password string) error
}

type bmcPassword struct{}

// Store stores a BMC password in GCD.
func (p *bmcPassword) Store(hostname string, password string) error {
	parts, err := host.Parse(hostname)
	if err != nil {
		return fmt.Errorf("could not parse hostname: %s", hostname)
	}

	bmcHostname := strings.Replace(hostname, parts.Machine, parts.Machine+"d", 1)

	bmcAddr, err := net.LookupHost(bmcHostname)
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

func New() *bmcPassword {
	return &bmcPassword{}
}
