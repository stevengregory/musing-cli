package process

import (
	"fmt"

	"golang.org/x/sys/unix"
)

// ReadCommand reads native process metadata without flattening argv boundaries.
func ReadCommand(pid int) (Command, error) {
	if pid <= 0 || pid > 1<<31-1 {
		return Command{}, fmt.Errorf("invalid process PID %d", pid)
	}
	data, err := unix.SysctlRaw("kern.procargs2", pid)
	if err != nil {
		return Command{}, fmt.Errorf("read process arguments: %w", err)
	}
	return parseDarwinCommand(data)
}
