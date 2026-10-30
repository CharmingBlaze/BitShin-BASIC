package main

import (
	"fmt"
	"os"

	"moonbasic/internal/joltstage"
)

func main() {
	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := joltstage.Stage(root); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
