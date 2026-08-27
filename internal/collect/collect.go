package collect

import (
	"bufio"
	"os"
	"strings"
)

var syscallKinds = map[string]string{
	"execve":     "process.exec",
	"clone":      "process.create",
	"connect":    "net.connect",
	"socket":     "net.socket",
	"sendto":     "net.send",
	"unlink":     "file.delete",
	"mkdir":      "file.mkdir",
	"statx":      "file.stat",
	"newfstatat": "file.stat",
	"access":     "file.stat",
}

type Event struct {
	Seq int
	PID string
	TS string
	Syscall string
	Raw string
	Kind string
}

func Parse(path string) ([]Event, int, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, 0, err
	}
	defer file.Close()

	var events []Event
	skipped := 0

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		fields := strings.Fields(line)
		if len(fields) < 3 {
			skipped++
			continue
		}

		name, _, found := strings.Cut(fields[2], "(")
		if !found {
			skipped++
			continue
		}

		events = append(events, Event{
			Seq:     len(events) + 1,
			PID:     fields[0],
			TS:      fields[1],
			Syscall: name,
			Raw:     line,
			Kind:	 classify(name, line),
		})
	}

	return events, skipped, scanner.Err()
}

func classify(syscall, raw string) string {
	if syscall == "openat" || syscall == "open" {
		if strings.Contains(raw, "O_WRONLY") ||
			strings.Contains(raw, "O_RDWR") ||
			strings.Contains(raw, "O_CREAT") ||
			strings.Contains(raw, "O_APPEND") {
			return "file.write"
		}
		return "file.read"
	}

	if kind, ok := syscallKinds[syscall]; ok {
		return kind
	}

	return "other"
}
