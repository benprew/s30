package bugreport

import (
	"fmt"
	"os"

	"github.com/benprew/s30/game/world"
)

// HandleCrash captures a panic, writes crash diagnostics to disk and the worker, and logs details.
func HandleCrash(level *world.Level, screenName string, screen any, panicVal any, stack []byte) *SubmitResult {
	report := CollectCrashReport(level, screenName, screen, panicVal, stack)
	submitter := NewDefaultSubmitter()

	result, err := submitter.Submit(report)

	fmt.Fprintf(os.Stderr, "\n=======================================================\n")
	fmt.Fprintf(os.Stderr, "SHANDALAR 30 CRASH DETECTED\n")
	fmt.Fprintf(os.Stderr, "Error: %v\n", panicVal)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to submit crash report: %v\n", err)
	} else {
		if result.LocalFilePath != "" {
			fmt.Fprintf(os.Stderr, "Crash report saved to: %s\n", result.LocalFilePath)
		}
		if result.IssueURL != "" {
			fmt.Fprintf(os.Stderr, "GitHub Issue created: %s\n", result.IssueURL)
		}
	}
	fmt.Fprintf(os.Stderr, "=======================================================\n")
	fmt.Fprintf(os.Stderr, "%s\n", string(stack))
	fmt.Fprintf(os.Stderr, "=======================================================\n\n")

	return result
}
