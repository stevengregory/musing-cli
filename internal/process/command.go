package process

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// Command contains the executable path and argv, including argv[0].
type Command struct {
	Executable string
	Args       []string
}

const (
	maxCommandBytes = 8 << 20
	maxCommandArgs  = 1 << 16
)

func parseLinuxArgs(data []byte) ([]string, error) {
	if len(data) == 0 || len(data) > maxCommandBytes || data[len(data)-1] != 0 {
		return nil, fmt.Errorf("missing, oversized, or unterminated process arguments")
	}
	if bytes.Count(data, []byte{0}) > maxCommandArgs {
		return nil, fmt.Errorf("too many process arguments")
	}
	values := bytes.Split(data[:len(data)-1], []byte{0})
	args := make([]string, len(values))
	for i, value := range values {
		args[i] = string(value)
	}
	return args, nil
}

func parseDarwinCommand(data []byte) (Command, error) {
	if len(data) < 4 || len(data) > maxCommandBytes {
		return Command{}, fmt.Errorf("missing or oversized process argument data")
	}
	argc := binary.NativeEndian.Uint32(data[:4])
	data = data[4:]
	end := bytes.IndexByte(data, 0)
	if end <= 0 {
		return Command{}, fmt.Errorf("missing or unterminated executable path")
	}
	command := Command{Executable: string(data[:end])}
	// On supported 64-bit Darwin platforms, XNU aligns the executable path
	// storage to eight bytes. Consume only that padding: further NUL bytes can
	// represent empty arguments, including argv[0]. The executable_path= prefix
	// stripped by the sysctl is sixteen bytes and leaves this alignment intact.
	start := (end + 1 + 7) &^ 7
	if start > len(data) {
		return Command{}, fmt.Errorf("truncated executable path padding")
	}
	for _, value := range data[end+1 : start] {
		if value != 0 {
			return Command{}, fmt.Errorf("invalid executable path padding")
		}
	}
	data = data[start:]
	if argc == 0 || argc > maxCommandArgs || uint64(argc) > uint64(len(data)) {
		return Command{}, fmt.Errorf("invalid process argument count %d", argc)
	}
	command.Args = make([]string, 0, int(argc))
	for i := uint32(0); i < argc; i++ {
		end = bytes.IndexByte(data, 0)
		if end < 0 {
			return Command{}, fmt.Errorf("unterminated process argument %d", i)
		}
		command.Args = append(command.Args, string(data[:end]))
		data = data[end+1:]
	}
	// Remaining strings are environment entries, not command arguments.
	return command, nil
}
