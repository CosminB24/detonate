package runner

import (
	"os"
	"os/exec"
)

func Detonate(outDir, mountSource, installArg, logName string) error {
	command := exec.Command("docker", "run", "--rm",
		"--network=none",
		"--cap-add=SYS_PTRACE",
		"--security-opt", "seccomp=unconfined",
		"-v", outDir+":/out",
		"-v", mountSource+":/pkg:ro",
		"detonate-spike",
		"strace", "-f", "-tt", "-y", "-s", "512",
		"-e", "trace=%process,%file,%network",
		"-o", "/out/"+logName,
		"npm", "install", "--offline", "--no-audit", "--no-fund",
		"--foreground-scripts", installArg)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr

	return command.Run()
}