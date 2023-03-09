// token implements the epoxy extension API and provides a way for machines
// booting with epoxy to obtain a bootstrap token to join the cluster.
package token

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// Commander is an interface that is used to wrap os/exec.Command() for testing purposes.
type Commander interface {
	Command(prog string, args ...string) ([]byte, error)
}

// K8sCommand implements the Commander interface.
type K8sCommand struct{}

func (tc *K8sCommand) Command(prog string, args ...string) ([]byte, error) {
	cmd := exec.Command(prog, args...)
	return cmd.Output()
}

// Generator defines the interface for creating tokens.
type Generator interface {
	Create(target string, args []string) error // Generate a new token.
	Response(version string) ([]byte, error)
}

// k8sGenerator implements the Generator interface.
type k8sGenerator struct {
	Command   string
	Commander Commander
	Details   Details
}

// details represents data used in responses to allocate_k8s_token extension
// requests. For v1, only Token will be populated/returned, and for v2 all
// fields should have values and will be returned as JSON.
type Details struct {
	APIAddress string `json:"api_address"`
	Token      string `json:"token"`
	CAHash     string `json:"ca_hash"`
}

// Create generates a new k8s token.
func (g *k8sGenerator) Create(target string, args []string) error {
	// Allocate the token for the given hostname.
	output, err := g.Commander.Command(g.Command, args...)
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
	g.Details.APIAddress = fields[2]
	g.Details.Token = fields[4]
	g.Details.CAHash = fields[6]

	return nil
}

// Response returns an appropriate response body for the incoming request, based
// on the API version.
func (g *k8sGenerator) Response(version string) ([]byte, error) {
	if version == "v1" {
		return []byte(g.Details.Token), nil
	}
	return json.Marshal(g.Details)
}

// New returns a partially populated k8sGenerator
func New(bindir string, commander Commander) Generator {
	return &k8sGenerator{
		Command:   bindir + "/kubeadm",
		Commander: commander,
		Details:   Details{},
	}

}
