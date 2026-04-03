package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var version = "0.1.0"

func main() {
	rootCmd := &cobra.Command{
		Use:   "jira-hours",
		Short: "Jira time logging automation tool",
		Long:  "A CLI tool to automate logging hours from monthly time sheets to Jira tickets",
		Version: version,
	}

	rootCmd.AddCommand(newLogCmd())
	rootCmd.AddCommand(newValidateCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
