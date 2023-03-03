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

// The token_server implements the epoxy extension API and provides a way for
// machines booting with epoxy to allocate a k8s token, necessary for joining
// the cluster.
//
// To deploy the token_server, the ePoxy server must have an extension
// registered that maps an operation name to this server, e.g.:
//
//	"allocate_k8s_token" -> "http://localhost:8800/allocate_k8s_token"
package allocate_k8s_token

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

var (
	localCommander commander = &runCommand{}
)

type commander interface {
	Command(prog string, args ...string) ([]byte, error)
}

type runCommand struct{}

// Command takes a program name and arguments as parameters and hands those off
// to exec.Command. It exists as a wrapper to exec.Command to faciliate
// testing. It has the same return types as exec.Command: ([]byte, error).
func (c *runCommand) Command(prog string, args ...string) ([]byte, error) {
	cmd := exec.Command(prog, args...)
	return cmd.Output()
}

// TokenGenerator defines the interface for creating tokens.
type TokenGenerator interface {
	Create(target string) error // Generate a new token.
	Response(version string) ([]byte, error)
}

type k8sTokenGenerator struct {
	Command       string
	TokenResponse tokenResponse
}

type tokenResponse struct {
	APIAddress string `json:"api_address"`
	Token      string `json:"token"`
	CAHash     string `json:"ca_hash"`
}

// Token generates a new k8s token.
func (g *k8sTokenGenerator) Create(target string) error {
	args := []string{
		"token", "create", "--ttl", "5m", "--print-join-command",
		"--description", "Allow " + target + " to join the cluster",
	}
	// Allocate the token for the given hostname.
	output, err := localCommander.Command(g.Command, args...)
	if err != nil {
		return err
	}
	fields := strings.Fields(string(output))
	// The join command should have 7 fields, and we count on this to return the
	// right values. A sample join command:
	// kubeadm join <api address> --token <token> --discovery-token-ca-cert-hash <hash>
	if len(fields) != 7 {
		return fmt.Errorf("bad join command: %s", string(output))
	}
	g.TokenResponse.APIAddress = fields[2]
	g.TokenResponse.Token = fields[4]
	g.TokenResponse.CAHash = fields[6]

	return nil
}

// Response returns an appropriate response body for the incoming request, based
// on the API version.
func (g *k8sTokenGenerator) Response(version string) ([]byte, error) {
	if version == "v1" {
		return []byte(g.TokenResponse.Token), nil
	}
	return json.Marshal(g.TokenResponse)
}

func New(bindir string) *k8sTokenGenerator {
	return &k8sTokenGenerator{
		Command:       bindir + "/kubeadm",
		TokenResponse: tokenResponse{},
	}

}
