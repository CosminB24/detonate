package main

import (
	"fmt"
	"os"
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
}
