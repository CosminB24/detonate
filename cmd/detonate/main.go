package main

import (
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

	fmt.Printf("%-10s %s\n", "ecosystem:", os.Args[1])
	fmt.Printf("%-10s %s\n", "target:", os.Args[2])

	wd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	outPath := filepath.Join(wd, "out")

	if err = os.MkdirAll(outPath, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	pkgPath := filepath.Join(outPath, "pkg")

	if err = os.MkdirAll(pkgPath, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	npmPack := exec.Command("npm", "pack", os.Args[2], "--pack-destination", pkgPath)

	out, err := npmPack.Output()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fileName := strings.TrimSpace(string(out))

	file := filepath.Join(pkgPath, fileName)
	fmt.Printf("%-10s %s\n", "downloaded:", file)

	command := exec.Command("docker", "run", "--rm", "--network=none", "--cap-add=SYS_PTRACE",
		"--security-opt", "seccomp=unconfined", "-v", outPath+":/out",
		"detonate-spike",
		"strace", "-f", "-tt", "-y", "-s", "512",
		"-e", "trace=%process,%file,%network",
		"-o", "/out/clean.log",
		"npm", "install", "--offline", "--no-audit", "--no-fund", "--no-save",
		"--ignore-scripts", "/fixtures/lodash-4.17.21.tgz")

	command.Stdout = os.Stdout
	command.Stderr = os.Stderr

	err = command.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
