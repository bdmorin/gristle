// SPDX-FileCopyrightText: 2024 Ville Eurométropole Strasbourg
//
// SPDX-License-Identifier: MIT

// Main program
package main

import (
	"fmt"
	"os"

	"github.com/bdmorin/gristle/cmd"
)

var version = "dev" // Set via ldflags during build

func main() {
	// Set version for cmd package
	cmd.Version = version

	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
