package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

	// If the target is a local path, resolve it to an absolute path so it can
	// be mounted into the container regardless of the current directory.
	isLocalPath := strings.HasPrefix(target, ".") || strings.HasPrefix(target, "/")
	if isLocalPath {
		abs, err := filepath.Abs(target)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		target = abs
	}

	fmt.Printf("%-12s %s\n", "ecosystem:", ecosystem)
	fmt.Printf("%-12s %s\n", "target:", target)

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

	// Determine what to mount into the container and how npm will refer to it.
	//
	// LOCAL PATH  (e.g. ./spike/testdata/evil-pkg): mount the package folder at
	//   /pkg and install from there. Works for dependency-free packages.
	//
	// NAMED PACKAGE (e.g. lodash@4.17.21): fetch the tarball on the host with
	//   npm pack (network is safe: nothing executes), mount the folder holding
	//   it, and install that tarball offline in the container.
	//
	// NOTE: packages WITH dependencies are not handled yet — offline install in
	// the container has no cache for them. Tracked as a separate design task.
	var mountSource string // host path to mount
	var installArg string  // what npm installs, as seen inside the container

	if isLocalPath {
		mountSource = target
		installArg = "/pkg"
	} else {
		pkgPath := filepath.Join(outPath, "pkg")
		if err := os.MkdirAll(pkgPath, 0o755); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		pack := exec.Command("npm", "pack", target, "--pack-destination", pkgPath)
		out, err := pack.Output()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fileName := strings.TrimSpace(string(out))
		fmt.Printf("%-12s %s\n", "downloaded:", filepath.Join(pkgPath, fileName))
		mountSource = pkgPath
		installArg = "/pkg/" + fileName
	}

	// Detonate inside the offline container. strace records every process,
	// file and network syscall to /out/trace.log. Install scripts run
	// (--foreground-scripts) so their behaviour is captured.
	logName := "trace.log"
	command := exec.Command("docker", "run", "--rm",
		"--network=none",
		"--cap-add=SYS_PTRACE",
		"--security-opt", "seccomp=unconfined",
		"-v", outPath+":/out",
		"-v", mountSource+":/pkg:ro",
		"detonate-spike",
		"strace", "-f", "-tt", "-y", "-s", "512",
		"-e", "trace=%process,%file,%network",
		"-o", "/out/"+logName,
		"npm", "install", "--offline", "--no-audit", "--no-fund",
		"--foreground-scripts", installArg)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err := command.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Printf("%-12s %s\n", "trace:", filepath.Join(outPath, logName))

	tracePath := filepath.Join(outPath, logName)

	file, err := os.Open(tracePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	count := 0
	skipped := 0
	parsed := 0
	for scanner.Scan() {
		line := scanner.Text()
		count++

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
		parsed++

		if parsed <= 5 {
			fmt.Printf("%s | %s | %s\n", fields[0], fields[1], name)
		}
	}

	fmt.Printf("%-12s %d\n", "lines:", count)
	fmt.Printf("%-12s %d\n", "parsed:", parsed)
	fmt.Printf("%-12s %d\n", "skipped:", skipped)
}
