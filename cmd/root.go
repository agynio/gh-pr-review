package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Execute sets up the root command tree and executes it.
func Execute() error {
	root := newRootCommand()
	return root.Execute()
}

func newRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "gh-pr-review",
		Short:         "PR review helper commands for gh",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.AddCommand(newCommentsCommand())
	cmd.AddCommand(newReviewCommand())

	return cmd
}

// ExecuteOrExit runs the command tree and exits with a non-zero status on error.
func ExecuteOrExit() {
	if err := Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
