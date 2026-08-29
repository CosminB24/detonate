package runner

import (
	"os"
	"os/exec"
)

const image = "detonate-spike"

// Detonate analyses a package in two container phases sharing a work directory.
func Detonate(outDir, workDir, installArg, logName string) error {
	if err := fetch(workDir, installArg); err != nil {
		return err
	}
	return execute(outDir, workDir, logName)
}

// fetch downloads the package and its dependencies. Network is safe here:
// --ignore-scripts means nothing untrusted executes.
func fetch(workDir, installArg string) error {
	cmd := exec.Command("docker", "run", "--rm",
		"-v", workDir+":/work",
		"-w", "/work",
		image,
		"npm", "install", installArg,
		"--ignore-scripts", "--no-audit", "--no-fund")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// execute runs the install scripts offline, under strace.
func execute(outDir, workDir, logName string) error {
	cmd := exec.Command("docker", "run", "--rm",
		"--network=none",
		"--cap-add=SYS_PTRACE",
		"--security-opt", "seccomp=unconfined",
		"-v", outDir+":/out",
		"-v", workDir+":/work",
		"-w", "/work",
		image,
		"strace", "-f", "-tt", "-y", "-s", "512",
		"-e", "trace=%process,%file,%network",
		"-o", "/out/"+logName,
		"npm", "rebuild", "--foreground-scripts")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Clean removes the work directory. It runs inside a container because the
// files are owned by root (created by the container), not by the host user.
func Clean(outDir string) error {
	cmd := exec.Command("docker", "run", "--rm",
		"-v", outDir+":/out",
		image,
		"rm", "-rf", "/out/work")
	return cmd.Run()
}
