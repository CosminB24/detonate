package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func main() {
	const requiredArgs = 3
	if len(os.Args) < requiredArgs {
		fmt.Fprintln(os.Stderr, "Usage: detonate npm <package>")
		os.Exit(1)
	}
	if os.Args[1] != "npm" {
		fmt.Fprintln(os.Stderr, "Only npm is supported")
		os.Exit(1)
	}

	ecosystem := os.Args[1]
	target := os.Args[2]
	fmt.Printf("%-12s %s\n", "ecosystem:", ecosystem)
	fmt.Printf("%-12s %s\n", "target:", target)

	// Working directories on the host.
	wd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	outPath := filepath.Join(wd, "out")
	if err := os.MkdirAll(outPath, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	workPath := filepath.Join(outPath, "work")
	if err := os.MkdirAll(workPath, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// PHASE A — download the package AND its full dependency tree on the host.
	// --ignore-scripts is the safety boundary: npm fetches every file but runs
	// no code, so nothing untrusted executes on the host. Needs network.
	install := exec.Command("npm", "install", target,
		"--ignore-scripts", "--no-audit", "--no-fund")
	install.Dir = workPath
	install.Stdout = os.Stdout
	install.Stderr = os.Stderr
	if err := install.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("%-12s %s\n", "installed:", workPath)

	// PHASE B — detonate inside the offline container. Reactivated next step.
	/*
		command := exec.Command("docker", "run", "--rm",
			"--network=none",
			"--cap-add=SYS_PTRACE",
			"--security-opt", "seccomp=unconfined",
			"-v", outPath+":/out",
			"-v", workPath+":/work:ro",
			"detonate-spike",
			"strace", "-f", "-tt", "-y", "-s", "512",
			"-e", "trace=%process,%file,%network",
			"-o", "/out/clean.log",
			"npm", "install", "--offline", "--no-audit", "--no-fund", "--no-save")
		command.Stdout = os.Stdout
		command.Stderr = os.Stderr
		if err := command.Run(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	*/
}
