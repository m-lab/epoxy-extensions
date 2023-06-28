// delete implements the ePoxy extension API and provides a way for nodes to
// leave the cluster. This is especially important for managed instance group
// (MIG) instances. As the instance group manager scales in and out the group,
// instances that are going to be deleted need to also be removed from the
// cluster.
package delete

import (
	"os/exec"
)

var commandArgs []string = []string{
	"delete", "node",
}

// Commander is an interface that is used to wrap os/exec.Command() for testing purposes.
type Commander interface {
	Command(prog string, args ...string) ([]byte, error)
}

// DeleteCommand implements the Commander interface.
type DeleteCommand struct{}

func (dc *DeleteCommand) Command(prog string, args ...string) ([]byte, error) {
	cmd := exec.Command(prog, args...)
	return cmd.Output()
}

// Manager defines the interface for deleting a node.
type Manager interface {
	Delete(target string) error // Delete a node
}

// DeleteManager implements the Manager interface.
type DeleteManager struct {
	Command   string
	Commander Commander
}

// Delete deletes a node from the cluster.
func (d *DeleteManager) Delete(target string) error {
	// Append the target to the slice of arguments, since it is only after the
	// request has been handled that we know which node the request is for.
	args := append(commandArgs, target)

	// Delete the node
	_, err := d.Commander.Command(d.Command, args...)
	if err != nil {
		return err
	}

	return nil
}

// New returns a DeleteManager.
func New(bindir string, commander Commander) Manager {
	return &DeleteManager{
		Command:   bindir + "/kubectl",
		Commander: commander,
	}
}
