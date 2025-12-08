package main

import (
	"errors"
	"fmt"
	"os"
)

func main() {
	if err := newRootCommand().Execute(); err != nil {
		var exitErr *exitError
		if errors.As(err, &exitErr) {
			if !exitErr.silent && exitErr.msg != "" {
				fmt.Fprintln(os.Stderr, exitErr.msg)
			}
			os.Exit(exitErr.code)
		}

		fmt.Fprintf(os.Stderr, "goxsd: %v\n", err)
		os.Exit(1)
	}
}
