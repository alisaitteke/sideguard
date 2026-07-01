package main

import (
	"os"

	"github.com/alisaitteke/vibeguard/cmd/vibeguard/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
