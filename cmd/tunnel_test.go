package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stevengregory/musing-cli/internal/health"
)

func writeTunnelProject(t *testing.T, yaml string) string {
	t.Helper()

	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "compose.yaml"), []byte("services: {}\n"), 0o644); err != nil {
		t.Fatalf("failed writing compose.yaml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, ".musing.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatalf("failed writing .musing.yaml: %v", err)
	}
	return root
}

func chdirCmdTest(t *testing.T, dir string) {
	t.Helper()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })

	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to chdir to %s: %v", dir, err)
	}
}

func TestTunnelStart_AlreadyRunning_SkipsSSHCommand(t *testing.T) {
	root := writeTunnelProject(t, `
database:
  type: MongoDB
  name: mydb
  devPort: 27018
  prodPort: 27019
  dataDir: data
production:
  server: root@example.com
services: []
`)
	chdirCmdTest(t, root)

	originalCheck := checkPort
	originalExec := execCommand
	t.Cleanup(func() {
		checkPort = originalCheck
		execCommand = originalExec
	})

	checkPort = func(port int) health.PortStatus {
		return health.PortStatus{Port: port, Open: true}
	}
	execCommand = func(name string, args ...string) *exec.Cmd {
		t.Fatalf("execCommand should not be called when tunnel is already running")
		return nil
	}

	if err := tunnelStart(); err != nil {
		t.Fatalf("tunnelStart returned error: %v", err)
	}
}

func TestTunnelStop_NotRunning_SkipsLsofAndKill(t *testing.T) {
	root := writeTunnelProject(t, `
database:
  type: MongoDB
  name: mydb
  devPort: 27018
  prodPort: 27019
  dataDir: data
production:
  server: root@example.com
services: []
`)
	chdirCmdTest(t, root)

	originalCheck := checkPort
	originalExec := execCommand
	t.Cleanup(func() {
		checkPort = originalCheck
		execCommand = originalExec
	})

	checkPort = func(port int) health.PortStatus {
		return health.PortStatus{Port: port, Open: false}
	}
	execCommand = func(name string, args ...string) *exec.Cmd {
		t.Fatalf("execCommand should not be called when tunnel is not running")
		return nil
	}

	if err := tunnelStop(); err != nil {
		t.Fatalf("tunnelStop returned error: %v", err)
	}
}

func TestTunnelStop_Running_KillsPid(t *testing.T) {
	root := writeTunnelProject(t, `
database:
  type: MongoDB
  name: mydb
  devPort: 27018
  prodPort: 27019
  dataDir: data
production:
  server: root@example.com
services: []
`)
	chdirCmdTest(t, root)

	originalCheck := checkPort
	originalExec := execCommand
	t.Cleanup(func() {
		checkPort = originalCheck
		execCommand = originalExec
	})

	checkPort = func(port int) health.PortStatus {
		return health.PortStatus{Port: port, Open: true}
	}

	var calls []string
	execCommand = func(name string, args ...string) *exec.Cmd {
		calls = append(calls, name)

		switch name {
		case "lsof":
			return exec.Command("sh", "-c", "echo 4321")
		case "kill":
			if len(args) != 1 || args[0] != "4321" {
				t.Fatalf("kill called with wrong pid: %v", args)
			}
			return exec.Command("sh", "-c", "exit 0")
		default:
			t.Fatalf("unexpected command: %s %v", name, args)
			return nil
		}
	}

	if err := tunnelStop(); err != nil {
		t.Fatalf("tunnelStop returned error: %v", err)
	}

	if len(calls) != 2 || calls[0] != "lsof" || calls[1] != "kill" {
		t.Fatalf("unexpected command sequence: %v", calls)
	}
}
