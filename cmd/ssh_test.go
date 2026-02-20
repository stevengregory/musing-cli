package cmd

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/stevengregory/musing-cli/internal/config"
)

func TestBuildSSHArgs(t *testing.T) {
	cfg := &config.ProjectConfig{
		Database: config.DatabaseConfig{
			ProdPort: 27019,
		},
		Production: &config.ProductionConfig{
			Server:       "root@example.com",
			RemoteDBPort: 27017,
			SSHKeyPath:   "~/.ssh/id_ed25519",
		},
	}

	tests := []struct {
		name       string
		withTunnel bool
		want       []string
	}{
		{
			name:       "interactive session args",
			withTunnel: false,
			want:       []string{"-i", expandHomeDir("~/.ssh/id_ed25519"), "root@example.com"},
		},
		{
			name:       "tunnel args",
			withTunnel: true,
			want: []string{
				"-i", expandHomeDir("~/.ssh/id_ed25519"),
				"-f", "-N",
				"-L", "27019:localhost:27017",
				"root@example.com",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildSSHArgs(cfg, tt.withTunnel)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("buildSSHArgs() mismatch\nwant: %v\ngot:  %v", tt.want, got)
			}
		})
	}
}

func TestBuildSSHArgs_DefaultPorts(t *testing.T) {
	cfg := &config.ProjectConfig{
		Database: config.DatabaseConfig{
			ProdPort: 0,
		},
		Production: &config.ProductionConfig{
			Server:       "root@example.com",
			RemoteDBPort: 0,
		},
	}

	got := buildSSHArgs(cfg, true)
	want := []string{"-f", "-N", "-L", "27019:localhost:27017", "root@example.com"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("buildSSHArgs() defaults mismatch\nwant: %v\ngot:  %v", want, got)
	}
}

func TestExpandHomeDir(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get home dir: %v", err)
	}

	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "expands home prefix",
			in:   "~/test-key",
			want: filepath.Join(home, "test-key"),
		},
		{
			name: "keeps absolute path",
			in:   "/tmp/key",
			want: "/tmp/key",
		},
		{
			name: "keeps plain value",
			in:   "id_ed25519",
			want: "id_ed25519",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := expandHomeDir(tt.in)
			if got != tt.want {
				t.Fatalf("expandHomeDir(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
