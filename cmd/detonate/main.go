package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/CosminB24/detonate/internal/collect"
	"github.com/CosminB24/detonate/internal/runner"
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
	if err := runner.Detonate(outPath, mountSource, installArg, logName); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	tracePath := filepath.Join(outPath, logName)
	fmt.Printf("%-12s %s\n", "trace:", tracePath)

	events, skipped, err := collect.Parse(tracePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Printf("%-12s %d\n", "events:", len(events))
	fmt.Printf("%-12s %d\n", "skipped:", skipped)

	counts := map[string]int{}
	for _, e := range events {
		counts[e.Kind]++
	}

	fmt.Println()
	for kind, n := range counts {
		fmt.Printf("%-16s %d\n", kind, n)
	}
}
