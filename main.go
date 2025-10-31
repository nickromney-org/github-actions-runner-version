package main

import (
	"os"

	"github.com/nickromney-org/github-actions-runner-version/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
