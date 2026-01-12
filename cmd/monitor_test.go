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
