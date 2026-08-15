package main

import (
	"fmt"
	"os"

	"bitshinbasic/internal/lsp"
)

func main() {
	if err := lsp.Serve(os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
