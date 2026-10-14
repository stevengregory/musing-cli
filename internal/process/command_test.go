package process

import (
	"encoding/binary"
	"reflect"
	"testing"
)

func darwinData(argc uint32, body string) []byte {
	data := make([]byte, 4, 4+len(body))
	binary.NativeEndian.PutUint32(data, argc)
	return append(data, body...)
}

func TestParseDarwinCommand_PreservesArgumentBoundaries(t *testing.T) {
	for _, args := range [][]string{
		{"ssh", "-i", "/tmp/key -f -N -L 27019:localhost:27017", "", "user@example.com", ""},
		{"", "ssh", "-f", "-N", "-L", "27019:localhost:27017"},
	} {
		data := darwinData(uint32(len(args)), "/usr/bin/ssh\x00\x00\x00\x00")
		for _, arg := range args {
			data = append(data, arg...)
			data = append(data, 0)
		}
		data = append(data, "SECRET=ignored\x00PATH=/usr/bin\x00"...)
		got, err := parseDarwinCommand(data)
		if err != nil {
			t.Fatalf("parseDarwinCommand: %v", err)
		}
		want := Command{Executable: "/usr/bin/ssh", Args: args}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("command mismatch: got %#v, want %#v", got, want)
		}
	}
}

func TestParseDarwinCommand_RejectsMalformedData(t *testing.T) {
	cases := map[string][]byte{
		"missing header":       nil,
		"short header":         {1, 0, 0},
		"missing executable":   darwinData(1, "\x00ssh\x00"),
		"unterminated path":    darwinData(1, "/usr/bin/ssh"),
		"invalid padding":      darwinData(1, "/usr/bin/ssh\x00ssh\x00"),
		"truncated padding":    darwinData(1, "/usr/bin/ssh\x00\x00"),
		"zero argc":            darwinData(0, "/usr/bin/ssh\x00\x00\x00\x00ssh\x00"),
		"negative argc":        darwinData(^uint32(0), "/usr/bin/ssh\x00\x00\x00\x00ssh\x00"),
		"impossible argc":      darwinData(100, "/usr/bin/ssh\x00\x00\x00\x00ssh\x00"),
		"missing argv":         darwinData(1, "/usr/bin/ssh\x00\x00\x00\x00"),
		"unterminated argv":    darwinData(1, "/usr/bin/ssh\x00\x00\x00\x00ssh"),
		"truncated later argv": darwinData(2, "/usr/bin/ssh\x00\x00\x00\x00ssh\x00-l"),
	}
	for name, data := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := parseDarwinCommand(data); err == nil {
				t.Fatal("expected malformed data error")
			}
		})
	}
}

func TestParseLinuxArgs_PreservesArgumentBoundaries(t *testing.T) {
	data := []byte("ssh\x00-i\x00/tmp/key -f -N -L 27019:localhost:27017\x00\x00user@example.com\x00\x00")
	want := []string{"ssh", "-i", "/tmp/key -f -N -L 27019:localhost:27017", "", "user@example.com", ""}
	got, err := parseLinuxArgs(data)
	if err != nil {
		t.Fatalf("parseLinuxArgs: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("argv mismatch: got %#v, want %#v", got, want)
	}
}

func TestParseLinuxArgs_RejectsUnavailableOrTruncatedData(t *testing.T) {
	for _, data := range [][]byte{nil, []byte("ssh"), []byte("ssh\x00-i")} {
		if _, err := parseLinuxArgs(data); err == nil {
			t.Fatalf("expected argument error for %q", data)
		}
	}
}

func TestParsers_RejectOversizedDataAndArgumentCounts(t *testing.T) {
	oversized := make([]byte, maxCommandBytes+1)
	if _, err := parseLinuxArgs(oversized); err == nil {
		t.Fatal("expected oversized Linux argument error")
	}
	if _, err := parseDarwinCommand(oversized); err == nil {
		t.Fatal("expected oversized Darwin argument error")
	}
	if _, err := parseLinuxArgs(make([]byte, maxCommandArgs+1)); err == nil {
		t.Fatal("expected excessive Linux argument count error")
	}
	data := darwinData(maxCommandArgs+1, "/usr/bin/ssh\x00\x00\x00\x00ssh\x00")
	data = append(data, make([]byte, maxCommandArgs)...)
	if _, err := parseDarwinCommand(data); err == nil {
		t.Fatal("expected excessive Darwin argument count error")
	}
}
