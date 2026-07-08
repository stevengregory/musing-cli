package cmd

import "testing"

// TestGetStatus tests the getStatus function
func TestGetStatus(t *testing.T) {
	tests := []struct {
		name     string
		open     bool
		expected string
	}{
		{
			name:     "service is running",
			open:     true,
			expected: "running",
		},
		{
			name:     "service is down",
			open:     false,
			expected: "down",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getStatus(tt.open)
			if result != tt.expected {
				t.Errorf("getStatus(%v) = %q, want %q", tt.open, result, tt.expected)
			}
		})
	}
}

func TestMonitorServiceGroupsUseServiceTypes(t *testing.T) {
	model := monitorModel{
		services: []ServiceHealth{
			{Name: ServiceDockerDesktop, Type: serviceTypeDocker},
			{Name: "MongoDB", Port: 27018, Type: serviceTypeDatabase},
			{Name: "news-api", Port: 8080, Type: serviceTypeAPI},
			{Name: "Web", Port: 3000, Type: serviceTypeFrontend},
			{Name: "root@example.com", Port: 27019, Type: serviceTypeSSHTunnel},
		},
	}

	assertGroup := func(name string, got []ServiceHealth, wantName string) {
		t.Helper()
		if len(got) != 1 {
			t.Fatalf("%s group length = %d, want 1 (%v)", name, len(got), got)
		}
		if got[0].Name != wantName {
			t.Fatalf("%s group service = %q, want %q", name, got[0].Name, wantName)
		}
	}

	assertGroup("docker", model.getDockerServices(), ServiceDockerDesktop)
	assertGroup("database", model.getDatabaseServices(), "MongoDB")
	assertGroup("api", model.getAPIServices(), "news-api")
	assertGroup("frontend", model.getFrontendServices(), "Web")
	assertGroup("ssh", model.getSSHTunnelServices(), "root@example.com")
}
