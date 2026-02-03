package main

import (
	"fmt"
	"path/filepath"

	"github.com/gherlein/jira-hours/internal/config"
	"github.com/gherlein/jira-hours/internal/dates"
	"github.com/gherlein/jira-hours/internal/parser"
	"github.com/spf13/cobra"
)

func newValidateCmd() *cobra.Command {
	var month string
	var timezone string

	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate time log files",
		Long:  "Validate monthly time log and project configuration files",
		RunE: func(cmd *cobra.Command, args []string) error {
			return validateFiles(month, timezone)
		},
	}

	cmd.Flags().StringVarP(&month, "month", "m", "", "Month in YYYY-MM format")
	cmd.Flags().StringVarP(&timezone, "timezone", "t", "America/Los_Angeles", "Timezone for date calculations")

	cmd.MarkFlagRequired("month")

	return cmd
}

func validateFiles(month string, timezone string) error {
	fmt.Printf("Validating: %s\n", month)
	fmt.Println("==================\n")

	year, monthNum, err := dates.ParseMonth(month)
	if err != nil {
		return err
	}
	fmt.Println("✓ Month format valid")

	logFile := filepath.Join("data", month+".yaml")
	monthlyLog, err := parser.ParseMonthlyLog(logFile)
	if err != nil {
		return fmt.Errorf("loading monthly log: %w", err)
	}
	fmt.Println("✓ Time log file valid")

	projectConfig, err := config.LoadProjectConfig("configs/projects.yaml")
	if err != nil {
		return fmt.Errorf("loading project config: %w", err)
	}
	fmt.Println("✓ Project config valid")

	unknownCodes := make([]string, 0)
	for _, entry := range monthlyLog.Hours {
		if _, exists := projectConfig.Projects[entry.Code]; !exists {
			unknownCodes = append(unknownCodes, entry.Code)
		}
	}

	if len(unknownCodes) > 0 {
		return fmt.Errorf("unknown project codes: %v", unknownCodes)
	}
	fmt.Println("✓ All project codes found")

	for w := 1; w <= 4; w++ {
		_, err := dates.GetWeekMonday(year, monthNum, w, timezone)
		if err != nil {
			fmt.Printf("⚠ Week %d: %v\n", w, err)
		}
	}
	fmt.Println("✓ Date calculations OK")

	total := monthlyLog.GetTotalHours()

	fmt.Println()
	fmt.Println("Hours Summary:")
	fmt.Println("--------------")

	nonZeroCount := 0
	for _, entry := range monthlyLog.Hours {
		projectTotal := entry.Week1 + entry.Week2 + entry.Week3 + entry.Week4
		if projectTotal > 0 {
			project, exists := projectConfig.Projects[entry.Code]
			ticketInfo := entry.Code
			if exists {
				ticketInfo = fmt.Sprintf("%s (%s)", project.Ticket, entry.Code)
			}
			fmt.Printf("  %-20s: %3d hours (W1:%2d W2:%2d W3:%2d W4:%2d)\n",
				ticketInfo,
				projectTotal,
				entry.Week1,
				entry.Week2,
				entry.Week3,
				entry.Week4)
			nonZeroCount++
		}
	}

	fmt.Println()
	fmt.Printf("Total: %d hours across %d projects\n", total, nonZeroCount)

	if total > 200 {
		fmt.Printf("⚠ Total hours (%d) exceeds 200 (may be unreasonable)\n", total)
	}

	fmt.Println("\n✓ Validation passed!")
	return nil
}
