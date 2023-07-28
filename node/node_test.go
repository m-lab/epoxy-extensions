package node

import (
	"fmt"
	"strings"
	"testing"
)

type fakeCommand struct {
	Path    string
	command string
}

func (fc *fakeCommand) Run(args ...string) ([]byte, error) {
	if fc.command == "" {
		return nil, fmt.Errorf("command failed")
	}
	return []byte("lol"), nil
}

func Test_Delete(t *testing.T) {
	tests := []struct {
		name     string
		hostname string
		command  string
		wantErr  bool
	}{
		{
			name:     "success",
			hostname: "mlab4-abc0t.mlab-sandbox.measurement-lab.org",
			command:  "kubectl delete node",
			wantErr:  false,
		},
		{
			name:     "fail-command-error",
			hostname: "mlab4-abc0t.mlab-sandbox.measurement-lab.org",
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Manager{
				Command: &fakeCommand{
					command: tt.command,
				},
			}
			err := m.Delete(tt.hostname)
			if (err != nil) != tt.wantErr {
				t.Errorf("Delete(): error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_Command(t *testing.T) {
	tests := []struct {
		name    string
		prog    string
		args    []string
		expect  string
		wantErr bool
	}{
		{
			name:    "success",
			prog:    "date",
			args:    []string{"--date=@1679083030", "--utc", "+%FT%T"},
			expect:  "2023-03-17T19:57:10",
			wantErr: false,
		},
		{
			name:    "failure",
			prog:    "nosuchfile",
			args:    []string{"lol", ";-)"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dc := &Command{
				Path: tt.prog,
			}
			output, err := dc.Run(tt.args...)
			result := strings.TrimSpace(string(output))
			if (err != nil) != tt.wantErr {
				t.Errorf("Command(): error = %v, wantErr %v", err, tt.wantErr)
			}
			if result != tt.expect {
				t.Errorf("Command(): expected '%s', got '%s'", tt.expect, result)
			}
		})
	}
}

func Test_New(t *testing.T) {
	m := NewManager("/fake/bin")
	var i interface{} = m
	_, ok := i.(*Manager)
	if !ok {
		t.Errorf("New(): expected type Manager, but got %T", m)
	}
}
