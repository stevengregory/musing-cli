package process

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
)

// ReadCommand reads native process metadata without flattening argv boundaries.
func ReadCommand(pid int) (Command, error) {
	if pid <= 0 || pid > 1<<31-1 {
		return Command{}, fmt.Errorf("invalid process PID %d", pid)
	}
	dir := filepath.Join("/proc", strconv.Itoa(pid))
	executable, err := os.Readlink(filepath.Join(dir, "exe"))
	if err != nil {
		return Command{}, fmt.Errorf("read process executable: %w", err)
	}
	file, err := os.Open(filepath.Join(dir, "cmdline"))
	if err != nil {
		return Command{}, fmt.Errorf("read process arguments: %w", err)
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxCommandBytes+1))
	if err != nil {
		return Command{}, fmt.Errorf("read process arguments: %w", err)
	}
	args, err := parseLinuxArgs(data)
	if err != nil {
		return Command{}, err
	}
	return Command{Executable: executable, Args: args}, nil
}
