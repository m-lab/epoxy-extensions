// token implements the ePoxy extension API and provides a way for machines
// booting with ePoxy to work with cluster bootstrap tokens, generally to create
// one for joining the cluster.
package token

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

var commandArgs []string = []string{
	"token", "create", "--ttl", "5m", "--print-join-command",
}

// Commander is an interface that is used to wrap os/exec.Command() for testing purposes.
type Commander interface {
	Command(prog string, args ...string) ([]byte, error)
}

// TokenCommand implements the Commander interface.
type TokenCommand struct{}

func (tc *TokenCommand) Command(prog string, args ...string) ([]byte, error) {
	cmd := exec.Command(prog, args...)
	return cmd.Output()
}

// Manager defines the interface for working with tokens.
type Manager interface {
	Create(target string) error // Generate a new token.
	Response(version string) ([]byte, error)
}

// TokenManager implements the Manager interface.
type TokenManager struct {
	Command   string
	Commander Commander
	Details   Details
}

// Details represents data used in responses to allocate_k8s_token extension
// requests. For v1, only Token will be populated/returned, and for v2 all
// fields should have values and will be returned as JSON.
type Details struct {
	APIAddress string `json:"api_address"`
	Token      string `json:"token"`
	CAHash     string `json:"ca_hash"`
}

// Create generates a new k8s token.
func (t *TokenManager) Create(target string) error {
	// Append the --description flag to the slice of arguments, since it is only
	// after the request has been handled that we know which host the request is for.
	desc := fmt.Sprintf("Allow %s to join the cluster", target)
	args := append(commandArgs, "--description", desc)

	// Allocate the token for the given hostname.
	output, err := t.Commander.Command(t.Command, args...)
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
	t.Details.APIAddress = fields[2]
	t.Details.Token = fields[4]
	t.Details.CAHash = fields[6]

	return nil
}

// Response returns an appropriate response body for the incoming request, based
// on the API version.
func (t *TokenManager) Response(version string) ([]byte, error) {
	if version == "v1" {
		return []byte(t.Details.Token), nil
	}
	return json.Marshal(t.Details)
}

// New returns a TokenManager.
func New(bindir string, commander Commander) Manager {
	return &TokenManager{
		Command:   bindir + "/kubeadm",
		Commander: commander,
		Details:   Details{},
	}

}
