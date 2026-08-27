package collect

import (
	"bufio"
	"os"
	"strings"
)

type Event struct {
	Seq int
	PID string
	TS string
	Syscall string
	Raw string
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
		})
	}

	return events, skipped, scanner.Err()
}