package health

import (
	"fmt"
	"net"
	"net/http"
	"time"
)

// PortStatus represents the status of a port check
type PortStatus struct {
	Port    int
	Open    bool
	Latency time.Duration
}

// HTTPStatus represents the status of an HTTP health check
type HTTPStatus struct {
	URL       string
	Available bool
	Latency   time.Duration
	Error     error
}

// CheckPort checks if a port is open on localhost
func CheckPort(port int) PortStatus {
	start := time.Now()
	addr := fmt.Sprintf("localhost:%d", port)

	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	latency := time.Since(start)

	if err != nil {
		return PortStatus{
			Port:    port,
			Open:    false,
			Latency: 0,
		}
	}

	conn.Close()
	return PortStatus{
		Port:    port,
		Open:    true,
		Latency: latency,
	}
}

// CheckHTTP performs an HTTP health check
func CheckHTTP(url string) HTTPStatus {
	start := time.Now()

	client := &http.Client{
		Timeout: 3 * time.Second,
	}

	resp, err := client.Get(url)
	latency := time.Since(start)

	if err != nil {
		return HTTPStatus{
			URL:       url,
			Available: false,
			Latency:   0,
			Error:     err,
		}
	}
	defer resp.Body.Close()

	available := resp.StatusCode >= 200 && resp.StatusCode < 300

	return HTTPStatus{
		URL:       url,
		Available: available,
		Latency:   latency,
		Error:     nil,
	}
}

// FormatLatency formats a duration for display (in milliseconds only)
func FormatLatency(d time.Duration) string {
	if d == 0 {
		return "timeout"
	}

	// Always show milliseconds with 1 decimal place for precision
	ms := float64(d.Microseconds()) / 1000.0
	return fmt.Sprintf("%.1fms", ms)
}
