package main

import (
	"fmt"
	"time"

	"github.com/gherlein/jira-hours/internal/jira"
)

func main() {
	fmt.Println("================================================================")
	fmt.Println("Demo: Idempotent Add + Delete Mode Workflow")
	fmt.Println("================================================================")
	fmt.Println()

	client := jira.NewMockClient()
	userEmail := "test@example.com"

	issueKey := "NPU-25"
	date := time.Date(2026, 1, 5, 8, 0, 0, 0, time.FixedZone("PST", -8*3600))
	hours := 12

	fmt.Println("STEP 1: First Add (should succeed)")
	fmt.Println("-----------------------------------")
	client.AddWorklog(issueKey, hours, date)
	fmt.Println("  ✓ Worklog added: 12h on 2026-01-05")
	fmt.Println()

	fmt.Println("STEP 2: Second Add - Idempotent Behavior")
	fmt.Println("-----------------------------------------")
	exists, worklogID, _ := client.WorklogExists(issueKey, date, hours, userEmail)
	if exists {
		fmt.Printf("  ○ Already logged (worklog %s) - SKIPPED\n", worklogID)
		fmt.Println("  → No duplicate created!")
	} else {
		client.AddWorklog(issueKey, hours, date)
		fmt.Println("  ✗ Would create duplicate (idempotent check failed)")
	}
	fmt.Println()

	fmt.Println("STEP 3: Verify Worklog Exists")
	fmt.Println("------------------------------")
	worklogs, _ := client.GetWorklogs(issueKey)
	fmt.Printf("  Found %d worklog(s) for %s:\n", len(worklogs), issueKey)
	for _, wl := range worklogs {
		fmt.Printf("    - ID %s: %s on %s (author: %s)\n", wl.ID, wl.TimeSpent, wl.Started[:10], wl.Author.EmailAddress)
	}
	fmt.Println()

	fmt.Println("STEP 4: Delete Mode - Find and Remove")
	fmt.Println("--------------------------------------")
	matches, _ := client.FindMatchingWorklogs(issueKey, date, userEmail)
	fmt.Printf("  Found %d matching worklog(s) to delete:\n", len(matches))
	for _, wl := range matches {
		fmt.Printf("    - Deleting worklog %s (%s)\n", wl.ID, wl.TimeSpent)
		client.DeleteWorklog(issueKey, wl.ID)
	}
	if len(matches) > 0 {
		fmt.Println("  ✓ Deleted successfully")
	} else {
		fmt.Println("  - No worklogs to delete")
	}
	fmt.Println()

	fmt.Println("STEP 5: Verify Deletion")
	fmt.Println("------------------------")
	worklogs, _ = client.GetWorklogs(issueKey)
	fmt.Printf("  Remaining worklogs: %d\n", len(worklogs))
	if len(worklogs) == 0 {
		fmt.Println("  ✓ All worklogs removed")
	} else {
		fmt.Printf("  - Still have %d worklog(s)\n", len(worklogs))
	}
	fmt.Println()

	fmt.Println("STEP 6: Re-add After Deletion")
	fmt.Println("------------------------------")
	exists, _, _ = client.WorklogExists(issueKey, date, hours, userEmail)
	if exists {
		fmt.Println("  ○ Already logged (skipped)")
	} else {
		client.AddWorklog(issueKey, hours, date)
		fmt.Println("  ✓ Worklog re-added: 12h on 2026-01-05")
	}
	worklogs, _ = client.GetWorklogs(issueKey)
	fmt.Printf("  Final count: %d worklog(s)\n", len(worklogs))
	fmt.Println()

	fmt.Println("================================================================")
	fmt.Println("Workflow Demonstration Complete!")
	fmt.Println("================================================================")
	fmt.Println()
	fmt.Println("This demonstrates:")
	fmt.Println("  1. Idempotent add prevents duplicates")
	fmt.Println("  2. Delete mode removes existing worklogs")
	fmt.Println("  3. Can delete and re-add for error correction")
	fmt.Println("  4. All operations are safe and reversible")
	fmt.Println()
}
