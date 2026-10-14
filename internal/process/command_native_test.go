//go:build darwin || linux

package process

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestReadCommand_CurrentProcess(t *testing.T) {
	command, err := ReadCommand(os.Getpid())
	if err != nil {
		t.Fatalf("ReadCommand: %v", err)
	}
	if !reflect.DeepEqual(command.Args, os.Args) {
		t.Fatalf("argv mismatch: got %#v, want %#v", command.Args, os.Args)
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable: %v", err)
	}
	if filepath.Base(command.Executable) != filepath.Base(executable) {
		t.Fatalf("executable mismatch: got %q, want %q", command.Executable, executable)
	}
}

func TestReadCommand_RejectsInvalidPID(t *testing.T) {
	oversized := int64(1 << 31)
	for _, pid := range []int{0, -1, int(oversized)} {
		if _, err := ReadCommand(pid); err == nil {
			t.Fatalf("expected invalid PID error for %d", pid)
		}
	}
}
