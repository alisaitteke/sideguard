// Copyright (c) 2026 Ali Sait Teke
// SPDX-License-Identifier: MIT

package main

import (
	"os"

	"github.com/alisaitteke/sideguard/cmd/sideguard/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
