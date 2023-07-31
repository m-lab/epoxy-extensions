// node implements the ePoxy extension API and provides a way for machines to
// manage their k8s nodes. For example, this package allows a machine to request
// to have its node deleted from the cluster when it reboots or shuts down.
// This is especially important for managed instance group (MIG) instances. As
// the instance group manager scales in and out the group, instances that are
// going to be deleted need to also be removed from the cluster.
package node

import (
	"log"
	"os/exec"
)

// Commander is an interface that is used to wrap os/exec.Command() for testing purposes.
type Commander interface {
	Run(args ...string) ([]byte, error)
}

// Command implements the Commander interface.
type Command struct {
	Path string
}

func (c *Command) Run(args ...string) ([]byte, error) {
	cmd := exec.Command(c.Path, args...)
	return cmd.Output()
}

// Manager mediates operations for a given node.
type Manager struct {
	Command Commander
}

// Delete deletes a node from the cluster.
func (m *Manager) Delete(target string) error {
	args := []string{
		"delete", "node", target,
	}

	// Delete the node
	output, err := m.Command.Run(args...)
	log.Println(string(output))
	if err != nil {
		return err
	}

	return nil
}

// NewManager returns a handler.NodeManager
func NewManager(bindir string) *Manager {
	return &Manager{
		Command: &Command{
			Path: bindir + "/kubectl",
		},
	}
}
