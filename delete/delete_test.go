package delete

import (
	"fmt"
	"strings"
	"testing"
)

type fakeDeleteCommand struct {
	command string
}

func (d *fakeDeleteCommand) Command(prog string, args ...string) ([]byte, error) {
	if d.command == "" {
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
			dm := &DeleteManager{
				Commander: &fakeDeleteCommand{
					command: tt.command,
				},
			}
			err := dm.Delete(tt.hostname)
			if (err != nil) != tt.wantErr {
				t.Errorf("Delete(): error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_DeleteCommand(t *testing.T) {
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
			dc := &DeleteCommand{}
			output, err := dc.Command(tt.prog, tt.args...)
			result := strings.TrimSpace(string(output))
			if (err != nil) != tt.wantErr {
				t.Errorf("DeleteCommand(): error = %v, wantErr %v", err, tt.wantErr)
			}
			if result != tt.expect {
				t.Errorf("DeleteCommand(): expected '%s', got '%s'", tt.expect, result)
			}
		})
	}
}

func Test_New(t *testing.T) {
	dc := &DeleteCommand{}
	m := New("/fake/bin", dc)
	var i interface{} = m
	_, ok := i.(Manager)
	if !ok {
		t.Errorf("New(): expected type Manager, but got %T", m)
	}
}
