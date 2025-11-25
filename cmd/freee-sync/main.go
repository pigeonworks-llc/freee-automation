// Package main is the entry point for freee-sync CLI.
package main

import (
	"os"

	"github.com/shunichi-ikebuchi/accounting-system/cmd/freee-sync/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
