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
	Command(prog string, args ...string) ([]byte, error)
}

// Command implements the Commander interface.
type Command struct{}

func (c *Command) Command(prog string, args ...string) ([]byte, error) {
	cmd := exec.Command(prog, args...)
	return cmd.Output()
}

// Manager defines the interface for managing a node.
type Manager interface {
	Delete(target string) error // Delete a node
}

// NodeManager implements the Manager interface.
type NodeManager struct {
	Command   string
	Commander Commander
}

// Delete deletes a node from the cluster.
func (n *NodeManager) Delete(target string) error {
	args := []string{
		"delete", "node", target,
	}

	// Delete the node
	output, err := n.Commander.Command(n.Command, args...)
	log.Println(string(output))
	if err != nil {
		return err
	}

	return nil
}

// New returns a NodeManager.
func New(bindir string, commander Commander) Manager {
	return &NodeManager{
		Command:   bindir + "/kubectl",
		Commander: commander,
	}
}
