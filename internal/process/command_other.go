//go:build !darwin && !linux

package process

import (
	"fmt"
	"runtime"
)

// ReadCommand reads native process metadata without flattening argv boundaries.
func ReadCommand(pid int) (Command, error) {
	return Command{}, fmt.Errorf("process command verification is unsupported on %s", runtime.GOOS)
}
