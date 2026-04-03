package main

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/gherlein/jira-hours/internal/config"
	"github.com/gherlein/jira-hours/internal/dates"
	"github.com/gherlein/jira-hours/internal/jira"
	"github.com/gherlein/jira-hours/internal/parser"
	"github.com/spf13/cobra"
)

type JiraClient interface {
	AddWorklog(issueKey string, hours int, startDate time.Time) error
	TestConnection() error
	GetWorklogs(issueKey string) ([]jira.Worklog, error)
	DeleteWorklog(issueKey, worklogID string) error
	FindMatchingWorklogs(issueKey string, targetDate time.Time, userEmail string) ([]jira.Worklog, error)
	WorklogExists(issueKey string, targetDate time.Time, hours int, userEmail string) (bool, string, error)
}

func newLogCmd() *cobra.Command {
	var month string
	var week int
	var code string
	var dryRun bool
	var deleteMode bool
	var mock bool
	var timezone string

	cmd := &cobra.Command{
		Use:   "log",
		Short: "Log or delete hours in Jira",
		Long:  "Log hours from monthly time log to Jira tickets via API, or delete existing worklogs",
		RunE: func(cmd *cobra.Command, args []string) error {
			return logHours(month, week, code, dryRun, deleteMode, mock, timezone)
		},
	}

	cmd.Flags().StringVarP(&month, "month", "m", "", "Month in YYYY-MM format")
	cmd.Flags().IntVarP(&week, "week", "w", 0, "Week number (1-4, 0 for all weeks)")
	cmd.Flags().StringVarP(&code, "code", "c", "", "Project code filter")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be logged without making API calls")
	cmd.Flags().BoolVarP(&deleteMode, "delete", "d", false, "Delete mode (remove worklogs instead of adding)")
	cmd.Flags().BoolVar(&mock, "mock", false, "Use mock client instead of real Jira API")
	cmd.Flags().StringVarP(&timezone, "timezone", "t", "America/Los_Angeles", "Timezone for date calculations")

	cmd.MarkFlagRequired("month")

	return cmd
}

func logHours(month string, week int, code string, dryRun bool, deleteMode bool, mock bool, timezone string) error {
	year, monthNum, err := dates.ParseMonth(month)
	if err != nil {
		return err
	}

	logFile := filepath.Join("data", month+".yaml")
	monthlyLog, err := parser.ParseMonthlyLog(logFile)
	if err != nil {
		return fmt.Errorf("loading monthly log: %w", err)
	}

	projectConfig, err := config.LoadProjectConfig("configs/projects.yaml")
	if err != nil {
		return fmt.Errorf("loading project config: %w", err)
	}

	var client JiraClient
	var userEmail string

	if mock || dryRun {
		client = jira.NewMockClient()
		userEmail = "test@example.com"
	} else {
		creds, err := loadCredentials()
		if err != nil {
			return fmt.Errorf("loading credentials: %w", err)
		}

		client = jira.NewClient(creds)
		userEmail = creds.Email

		fmt.Println("Testing Jira connection...")
		if err := client.TestConnection(); err != nil {
			return fmt.Errorf("jira connection failed: %w", err)
		}
		fmt.Println("✓ Connected to Jira")
	}

	if deleteMode {
		return deleteHours(month, year, monthNum, week, code, dryRun, client, userEmail, timezone, monthlyLog, projectConfig)
	}

	return addHours(month, year, monthNum, week, code, dryRun, client, userEmail, timezone, monthlyLog, projectConfig)
}

func addHours(month string, year int, monthNum int, week int, code string, dryRun bool, client JiraClient, userEmail string, timezone string, monthlyLog *parser.MonthlyLog, projectConfig *config.ProjectConfig) error {
	fmt.Printf("\nLogging Hours: %s\n", month)
	fmt.Println("=====================\n")

	stats := &logStats{}
	weeks := []int{1, 2, 3, 4}
	if week > 0 {
		weeks = []int{week}
	}

	for _, w := range weeks {
		monday, err := dates.GetWeekMonday(year, monthNum, w, timezone)
		if err != nil {
			fmt.Printf("⚠ Week %d: %v\n", w, err)
			continue
		}

		fmt.Printf("Week %d (Monday, %s):\n", w, monday.Format("2006-01-02"))

		for _, entry := range monthlyLog.Hours {
			if code != "" && entry.Code != code {
				continue
			}

			hours := entry.GetWeekHours(w)

			project, exists := projectConfig.Projects[entry.Code]
			if !exists {
				fmt.Printf("  ✗ %s: unknown project code\n", entry.Code)
				stats.failed++
				continue
			}

			if hours == 0 {
				if !dryRun {
					fmt.Printf("  - %-10s (%-6s): %dh (skipped)\n", project.Ticket, entry.Code, hours)
				}
				stats.skipped++
				continue
			}

			if dryRun {
				fmt.Printf("  ✓ %-10s (%-6s): %dh (would log)\n", project.Ticket, entry.Code, hours)
				stats.wouldLog++
				stats.totalHours += hours
			} else {
				alreadyExists, worklogID, err := client.WorklogExists(project.Ticket, monday, hours, userEmail)
				if err != nil {
					fmt.Printf("  ⚠ %-10s (%-6s): error checking existing: %v\n", project.Ticket, entry.Code, err)
				}
				if alreadyExists {
					fmt.Printf("  ○ %-10s (%-6s): %dh already logged (worklog %s)\n", project.Ticket, entry.Code, hours, worklogID)
					stats.alreadyLogged++
					stats.totalHours += hours
					continue
				}

				err = client.AddWorklog(project.Ticket, hours, monday)
				if err != nil {
					fmt.Printf("  ✗ %-10s (%-6s): %dh failed: %v\n", project.Ticket, entry.Code, hours, err)
					stats.failed++
				} else {
					fmt.Printf("  ✓ %-10s (%-6s): %dh logged\n", project.Ticket, entry.Code, hours)
					stats.success++
					stats.totalHours += hours
				}
			}
		}
		fmt.Println()
	}

	printSummary(stats, dryRun, false)

	if stats.failed > 0 {
		return fmt.Errorf("some worklogs failed to log")
	}

	return nil
}

func deleteHours(month string, year int, monthNum int, week int, code string, dryRun bool, client JiraClient, userEmail string, timezone string, monthlyLog *parser.MonthlyLog, projectConfig *config.ProjectConfig) error {
	fmt.Printf("\nDeleting Hours: %s\n", month)
	fmt.Println("====================\n")

	stats := &logStats{}
	weeks := []int{1, 2, 3, 4}
	if week > 0 {
		weeks = []int{week}
	}

	for _, w := range weeks {
		monday, err := dates.GetWeekMonday(year, monthNum, w, timezone)
		if err != nil {
			fmt.Printf("⚠ Week %d: %v\n", w, err)
			continue
		}

		fmt.Printf("Week %d (Monday, %s):\n", w, monday.Format("2006-01-02"))

		for _, entry := range monthlyLog.Hours {
			if code != "" && entry.Code != code {
				continue
			}

			project, exists := projectConfig.Projects[entry.Code]
			if !exists {
				fmt.Printf("  ✗ %s: unknown project code\n", entry.Code)
				stats.failed++
				continue
			}

			matchingWorklogs, err := client.FindMatchingWorklogs(project.Ticket, monday, userEmail)
			if err != nil {
				fmt.Printf("  ✗ %-10s (%-6s): error finding worklogs: %v\n", project.Ticket, entry.Code, err)
				stats.failed++
				continue
			}

			if len(matchingWorklogs) == 0 {
				fmt.Printf("  - %-10s (%-6s): no worklog found (skipped)\n", project.Ticket, entry.Code)
				stats.notFound++
				continue
			}

			for _, wl := range matchingWorklogs {
				if dryRun {
					fmt.Printf("  ✓ %-10s (%-6s): would delete worklog %s (%s)\n", project.Ticket, entry.Code, wl.ID, wl.TimeSpent)
					stats.wouldDelete++
					stats.totalHours += wl.TimeSpentSeconds / 3600
				} else {
					err := client.DeleteWorklog(project.Ticket, wl.ID)
					if err != nil {
						fmt.Printf("  ✗ %-10s (%-6s): failed to delete worklog %s: %v\n", project.Ticket, entry.Code, wl.ID, err)
						stats.failed++
					} else {
						fmt.Printf("  ✓ %-10s (%-6s): deleted worklog %s (%s)\n", project.Ticket, entry.Code, wl.ID, wl.TimeSpent)
						stats.deleted++
						stats.totalHours += wl.TimeSpentSeconds / 3600
					}
				}
			}
		}
		fmt.Println()
	}

	printSummary(stats, dryRun, true)

	if stats.failed > 0 {
		return fmt.Errorf("some operations failed")
	}

	return nil
}

type logStats struct {
	success       int
	failed        int
	skipped       int
	wouldLog      int
	alreadyLogged int
	deleted       int
	wouldDelete   int
	notFound      int
	totalHours    int
}

func printSummary(stats *logStats, dryRun bool, deleteMode bool) {
	fmt.Println("Summary:")
	fmt.Println("========")

	if deleteMode {
		if dryRun {
			fmt.Printf("  Would delete: %d worklogs\n", stats.wouldDelete)
			fmt.Printf("  Not found: %d entries\n", stats.notFound)
		} else {
			fmt.Printf("  Deleted: %d worklogs\n", stats.deleted)
			fmt.Printf("  Not found: %d entries\n", stats.notFound)
			if stats.failed > 0 {
				fmt.Printf("  Failed: %d\n", stats.failed)
			}
		}
	} else {
		if dryRun {
			fmt.Printf("  Would log: %d entries\n", stats.wouldLog)
			fmt.Printf("  Would skip: %d entries (zero hours)\n", stats.skipped)
		} else {
			fmt.Printf("  Success: %d new\n", stats.success)
			if stats.alreadyLogged > 0 {
				fmt.Printf("  Already logged: %d (skipped)\n", stats.alreadyLogged)
			}
			if stats.failed > 0 {
				fmt.Printf("  Failed: %d\n", stats.failed)
			}
			fmt.Printf("  Skipped: %d (zero hours)\n", stats.skipped)
		}
	}

	fmt.Printf("  Total hours: %d\n", stats.totalHours)
}

func loadCredentials() (*config.JiraCredentials, error) {
	creds, err := config.LoadCredentialsFromEnv()
	if err != nil {
		return nil, fmt.Errorf("credentials not found in environment variables (.envrc or .env): %w", err)
	}

	return creds, nil
}
