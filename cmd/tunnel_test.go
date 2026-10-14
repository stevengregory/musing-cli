package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/stevengregory/musing-cli/internal/health"
	"github.com/stevengregory/musing-cli/internal/process"
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
	originalRead := readProcessCommand
	t.Cleanup(func() {
		checkPort = originalCheck
		execCommand = originalExec
		readProcessCommand = originalRead
	})

	checkPort = func(port int) health.PortStatus {
		return health.PortStatus{Port: port, Open: false}
	}
	execCommand = func(name string, args ...string) *exec.Cmd {
		t.Fatalf("execCommand should not be called when tunnel is not running")
		return nil
	}
	readProcessCommand = func(pid int) (process.Command, error) {
		t.Fatalf("should not read process command when tunnel is not running")
		return process.Command{}, nil
	}

	if err := tunnelStop(); err != nil {
		t.Fatalf("tunnelStop returned error: %v", err)
	}
}

func TestTunnelStop_OnlyKillsConfiguredSSHTunnel(t *testing.T) {
	matchingArgs := []string{"ssh", "-f", "-N", "-L", "27019:localhost:27017", "root@example.com"}
	keyArgs := []string{"ssh", "-i", "/tmp/project-key", "-f", "-N", "-L", "27019:localhost:27017", "root@example.com"}
	flattenedCollision := []string{"ssh", "-i", "/tmp/project-key -f -N", "-L", "27019:localhost:27017", "root@example.com"}
	if strings.Join(keyArgs, " ") != strings.Join(flattenedCollision, " ") {
		t.Fatal("argv collision fixture should have identical flattened text")
	}

	tests := []struct {
		name        string
		pids        string
		executable  string
		commandArgs []string
		prodPort    int
		remotePort  int
		server      string
		keyPath     string
		failAt      string
		wantError   bool
		wantKill    bool
	}{
		{name: "configured tunnel", pids: "4321\n", executable: "ssh", commandArgs: matchingArgs, wantKill: true},
		{name: "absolute SSH executable", pids: "4321\n", executable: "/usr/bin/ssh", commandArgs: []string{"/usr/bin/ssh", "-f", "-N", "-L", "27019:localhost:27017", "root@example.com"}, wantKill: true},
		{name: "duplicate IPv4 and IPv6 listener", pids: "4321\n4321\n", executable: "ssh", commandArgs: matchingArgs, wantKill: true},
		{name: "normalized PID", pids: "  004321\n", executable: "ssh", commandArgs: matchingArgs, wantKill: true},
		{name: "custom ports", pids: "4321\n", executable: "ssh", commandArgs: []string{"ssh", "-f", "-N", "-L", "28019:localhost:28017", "root@example.com"}, prodPort: 28019, remotePort: 28017, wantKill: true},
		{name: "configured SSH key", pids: "4321\n", executable: "ssh", commandArgs: keyArgs, keyPath: "/tmp/project-key", wantKill: true},
		{name: "key path with spaces", pids: "4321\n", executable: "ssh", commandArgs: []string{"ssh", "-i", "/tmp/project key", "-f", "-N", "-L", "27019:localhost:27017", "root@example.com"}, keyPath: "/tmp/project key", wantKill: true},
		{name: "key path resembling SSH options", pids: "4321\n", executable: "ssh", commandArgs: []string{"ssh", "-i", "/tmp/key -o HostName=other", "-f", "-N", "-L", "27019:localhost:27017", "root@example.com"}, keyPath: "/tmp/key -o HostName=other", wantKill: true},
		{name: "key path with control character", pids: "4321\n", executable: "ssh", commandArgs: []string{"ssh", "-i", "/tmp/key\tname", "-f", "-N", "-L", "27019:localhost:27017", "root@example.com"}, keyPath: "/tmp/key\tname", wantKill: true},
		{name: "server with Unicode whitespace", pids: "4321\n", executable: "ssh", commandArgs: []string{"ssh", "-f", "-N", "-L", "27019:localhost:27017", "root@example.com\u00a0"}, server: "root@example.com\u00a0", wantKill: true},
		{name: "server with control character", pids: "4321\n", executable: "ssh", commandArgs: []string{"ssh", "-f", "-N", "-L", "27019:localhost:27017", "root@example.com\t"}, server: "root@example.com\t", wantKill: true},
		{name: "unrelated listener", pids: "4321\n", executable: "mongod", commandArgs: matchingArgs, wantError: true},
		{name: "wrong SSH server", pids: "4321\n", executable: "ssh", commandArgs: []string{"ssh", "-f", "-N", "-L", "27019:localhost:27017", "root@other.example.com"}, wantError: true},
		{name: "wrong SSH forward", pids: "4321\n", executable: "ssh", commandArgs: []string{"ssh", "-f", "-N", "-L", "27019:localhost:27018", "root@example.com"}, wantError: true},
		{name: "wrong SSH key", pids: "4321\n", executable: "ssh", commandArgs: []string{"ssh", "-i", "/tmp/other-key", "-f", "-N", "-L", "27019:localhost:27017", "root@example.com"}, keyPath: "/tmp/project-key", wantError: true},
		{name: "lossy argv collision", pids: "4321\n", executable: "ssh", commandArgs: flattenedCollision, keyPath: "/tmp/project-key", wantError: true},
		{name: "extra SSH destination override", pids: "4321\n", executable: "ssh", commandArgs: []string{"ssh", "-o", "HostName=other.example.com", "-f", "-N", "-L", "27019:localhost:27017", "root@example.com"}, wantError: true},
		{name: "misleading executable in args", pids: "4321\n", executable: "ssh", commandArgs: []string{"fake-ssh", "-f", "-N", "-L", "27019:localhost:27017", "root@example.com"}, wantError: true},
		{name: "empty executable", pids: "4321\n", commandArgs: matchingArgs, wantError: true},
		{name: "empty process args", pids: "4321\n", executable: "ssh", wantError: true},
		{name: "multiple listeners", pids: "4321\n5678\n", executable: "ssh", commandArgs: matchingArgs, wantError: true},
		{name: "no listener PID", wantError: true},
		{name: "malformed PID", pids: "4321x\n", wantError: true},
		{name: "malformed second PID", pids: "4321\ninvalid\n", executable: "ssh", commandArgs: matchingArgs, wantError: true},
		{name: "zero PID", pids: "0\n", wantError: true},
		{name: "negative PID", pids: "-4321\n", wantError: true},
		{name: "PID exceeds OS limit", pids: "2147483648\n", wantError: true},
		{name: "lsof failure", failAt: "lsof", wantError: true},
		{name: "process command read failure", pids: "4321\n", failAt: "process", wantError: true},
		{name: "kill failure", pids: "4321\n", executable: "ssh", commandArgs: matchingArgs, failAt: "kill", wantError: true, wantKill: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.server
			if server == "" {
				server = "root@example.com"
			}
			root := writeTunnelProject(t, fmt.Sprintf(`
database:
  prodPort: %d
production:
  server: %q
  remoteDBPort: %d
  sshKeyPath: %q
`, tt.prodPort, server, tt.remotePort, tt.keyPath))
			chdirCmdTest(t, root)

			originalCheck := checkPort
			originalExec := execCommand
			originalRead := readProcessCommand
			t.Cleanup(func() {
				checkPort = originalCheck
				execCommand = originalExec
				readProcessCommand = originalRead
			})

			port := tt.prodPort
			if port == 0 {
				port = 27019
			}
			checkPort = func(got int) health.PortStatus {
				if got != port {
					t.Fatalf("checked port %d, want %d", got, port)
				}
				return health.PortStatus{Port: got, Open: true}
			}

			var calls []string
			killCalled := false
			readProcessCommand = func(pid int) (process.Command, error) {
				calls = append(calls, "process")
				if pid != 4321 {
					t.Fatalf("process command queried PID %d, want 4321", pid)
				}
				if tt.failAt == "process" {
					return process.Command{}, fmt.Errorf("process unavailable")
				}
				return process.Command{Executable: tt.executable, Args: tt.commandArgs}, nil
			}
			execCommand = func(name string, args ...string) *exec.Cmd {
				calls = append(calls, name)
				switch name {
				case "lsof":
					want := []string{"-nP", "-a", fmt.Sprintf("-iTCP:%d", port), "-sTCP:LISTEN", "-t"}
					if !reflect.DeepEqual(args, want) {
						t.Fatalf("lsof args = %v, want listener-only query %v", args, want)
					}
					return fakeTunnelCommand(tt.pids, tt.failAt == "lsof")
				case "kill":
					if !tt.wantKill {
						t.Fatalf("attempted to kill an unverified listener: %v", args)
					}
					if !reflect.DeepEqual(args, []string{"4321"}) {
						t.Fatalf("kill args = %v, want one normalized positive PID", args)
					}
					killCalled = true
					return fakeTunnelCommand("", tt.failAt == "kill")
				default:
					t.Fatalf("unexpected command: %s %v", name, args)
				}
				return nil
			}

			err := tunnelStop()
			if (err != nil) != tt.wantError {
				t.Fatalf("tunnelStop error = %v, wantError %v", err, tt.wantError)
			}
			if killCalled != tt.wantKill {
				t.Fatalf("kill called = %v, want %v", killCalled, tt.wantKill)
			}
			if tt.wantKill && !reflect.DeepEqual(calls, []string{"lsof", "process", "kill"}) {
				t.Fatalf("unexpected command sequence: %v", calls)
			}
		})
	}
}

func TestTunnelStop_RequiresProductionConfig(t *testing.T) {
	tests := []struct {
		name       string
		production string
	}{
		{name: "no production configuration"},
		{name: "empty production configuration", production: "production: {}"},
		{name: "blank server", production: "production:\n  server: '   '"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := writeTunnelProject(t, "database:\n  prodPort: 27019\n"+tt.production+"\n")
			chdirCmdTest(t, root)

			originalCheck := checkPort
			originalExec := execCommand
			originalRead := readProcessCommand
			t.Cleanup(func() {
				checkPort = originalCheck
				execCommand = originalExec
				readProcessCommand = originalRead
			})
			checkPort = func(port int) health.PortStatus {
				t.Fatalf("should not check a port without production configuration")
				return health.PortStatus{}
			}
			execCommand = func(name string, args ...string) *exec.Cmd {
				t.Fatalf("should not execute %s without production configuration", name)
				return nil
			}
			readProcessCommand = func(pid int) (process.Command, error) {
				t.Fatalf("should not read process command without production configuration")
				return process.Command{}, nil
			}

			if err := tunnelStop(); err == nil || !strings.Contains(err.Error(), "production") {
				t.Fatalf("want useful production configuration error, got %v", err)
			}
		})
	}
}

func TestTunnelStop_ProjectDiscoveryError(t *testing.T) {
	chdirCmdTest(t, t.TempDir())
	if err := tunnelStop(); err == nil || !strings.Contains(err.Error(), "project root") {
		t.Fatalf("want useful project discovery error, got %v", err)
	}
}

// The command seam only launches harmless printf/exit processes during tests.
func fakeTunnelCommand(output string, fail bool) *exec.Cmd {
	if fail {
		return exec.Command("sh", "-c", "exit 1")
	}
	return exec.Command("sh", "-c", `printf '%s' "$1"`, "test", output)
}
