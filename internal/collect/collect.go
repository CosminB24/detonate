package collect

import (
	"bufio"
	"os"
	"strings"
)

var syscallKinds = map[string]string{
	"execve":      "process.exec",
	"clone":       "process.create",
	"connect":     "net.connect",
	"socket":      "net.socket",
	"sendto":      "net.send",
	"unlink":      "file.delete",
	"mkdir":       "file.mkdir",
	"statx":       "file.stat",
	"newfstatat":  "file.stat",
	"access":      "file.stat",
	"clone3":      "process.create",
	"vfork":       "process.create",
	"fork":        "process.create",
	"exit":        "process.exit",
	"exit_group":  "process.exit",
	"wait4":       "process.wait",
	"readlink":    "file.stat",
	"getcwd":      "file.stat",
	"rmdir":       "file.delete",
	"recvmsg":     "net.recv",
	"shutdown":    "net.close",
	"socketpair":  "net.socket",
	"setsockopt":  "net.socket",
	"getsockname": "net.socket",
	"getsockopt":  "net.socket",
}

type Event struct {
	Seq         int
	PID         string
	TS          string
	Syscall     string
	Raw         string
	Kind        string
	Target      string
	Failed      bool
	Partial     bool
	Result      string
	Phase       string
	ParentPhase string
}

const (
	PhaseSetup         = "setup"          // npm doing its own work
	PhaseInstallScript = "install_script" // code the package controls
)

const maxTreeDepth = 64

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

		var name string
		if fields[2] == "<..." {
			// "<... vfork resumed>) = 22" — the name is the next field.
			// Arguments were on the "<unfinished>" line, but the result is here.
			if len(fields) < 4 {
				skipped++
				continue
			}
			name = fields[3]
		} else {
			n, _, found := strings.Cut(fields[2], "(")
			if !found {
				skipped++
				continue
			}
			name = n
		}

		res := result(line)
		kind := classify(name, line)
		target := extractTarget(kind, line)

		events = append(events, Event{
			Seq:     len(events) + 1,
			PID:     fields[0],
			TS:      fields[1],
			Syscall: name,
			Kind:    kind,
			Target:  target,
			Raw:     line,
			Result:  res,
			Failed:  failed(res),
			Partial: partial(line),
		})
	}

	return MarkPhases(events), skipped, scanner.Err()
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

func extractTarget(kind, raw string) string {
	parts := strings.Split(raw, "\"")

	if len(parts) < 2 {
		return ""
	}

	return parts[1]
}

func failed(res string) bool {
	return strings.HasPrefix(res, "-")
}

func partial(raw string) bool {
	return strings.Contains(raw, "<unfinished")
}

func result(raw string) string {
	i := strings.LastIndex(raw, "= ")
	if i < 0 {
		return ""
	}

	rest := raw[i+2:]

	// Cut at the first space: "-1 ENOENT (...)" → "-1"
	if j := strings.IndexByte(rest, ' '); j >= 0 {
		rest = rest[:j]
	}
	// Cut at "<": "3</etc/passwd>" → "3"
	if j := strings.IndexByte(rest, '<'); j >= 0 {
		rest = rest[:j]
	}

	return rest
}

func ProcessTree(events []Event) map[string]string {
	tree := map[string]string{}

	for _, e := range events {
		if e.Kind != "process.create" {
			continue
		}
		if e.Failed {
			continue
		}
		if e.Partial {
			continue
		}
		if e.Result == "0" {
			continue
		}
		if e.Result == "" {
			continue
		}

		tree[e.Result] = e.PID
	}

	return tree
}

func ScriptRoots(events []Event) map[string]bool {
	roots := map[string]bool{}

	for _, e := range events {
		if e.Kind != "process.exec" {
			continue
		}
		if !strings.HasSuffix(e.Target, "/sh") && !strings.HasSuffix(e.Target, "/bash") {
			continue
		}
		if e.Partial {
			continue
		}
		if e.Failed {
			continue
		}

		roots[e.PID] = true
	}

	return roots
}

func MarkPhases(events []Event) []Event {
	tree := ProcessTree(events)
	roots := ScriptRoots(events)

	for i := range events {
		parent := tree[events[i].PID]
		events[i].Phase = phaseOf(events[i].PID, tree, roots)
		events[i].ParentPhase = phaseOf(parent, tree, roots)
	}

	return events
}

func phaseOf(pid string, tree map[string]string, roots map[string]bool) string {
	for depth := 0; depth < maxTreeDepth; depth++ {
		if roots[pid] {
			return PhaseInstallScript
		}

		parent, ok := tree[pid]
		if !ok {
			return PhaseSetup // no parent recorded: top of the tree
		}
		pid = parent
	}

	return PhaseSetup // depth limit hit; treat as setup rather than loop
}
