package cmd

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/agynio/gh-pr-review/internal/await"
	"github.com/agynio/gh-pr-review/internal/resolver"
)

func newAwaitCommand() *cobra.Command {
	opts := &awaitOptions{}

	cmd := &cobra.Command{
		Use:   "await <number> | <url>",
		Short: "Poll a pull request until it needs attention",
		Long: `Poll a pull request until review comments, merge conflicts, or CI failures appear.

Exit codes:
  0  Work detected — PR needs attention
  1  Nothing to do — PR is clean (or timeout expired)
  2  Input error or API failure`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.Selector = args[0]
			}
			return runAwait(cmd, opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Repo, "repo", "R", "", "Repository in 'owner/repo' format")
	cmd.Flags().IntVar(&opts.Pull, "pr", 0, "Pull request number")
	cmd.Flags().StringVar(&opts.Mode, "mode", "all", "Watch mode: all, comments, conflicts, actions")
	cmd.Flags().IntVarP(&opts.Timeout, "timeout", "t", 86400, "Maximum polling time in seconds (default: 86400 = 1 day)")
	cmd.Flags().IntVarP(&opts.Interval, "interval", "i", 300, "Polling interval in seconds (default: 300 = 5 minutes)")
	cmd.Flags().BoolVarP(&opts.CheckOnly, "check-only", "c", false, "Check once and exit (no polling)")
	cmd.Flags().BoolVarP(&opts.Quiet, "quiet", "q", false, "Suppress progress output")

	return cmd
}

type awaitOptions struct {
	Repo     string
	Pull     int
	Selector string
	Mode     string
	Timeout  int
	Interval int
	CheckOnly bool
	Quiet    bool
}

func runAwait(cmd *cobra.Command, opts *awaitOptions) error {
	// Validate
	if err := opts.Validate(); err != nil {
		return err
	}

	// Resolve selector
	selector, err := resolver.NormalizeSelector(opts.Selector, opts.Pull)
	if err != nil {
		return err
	}

	// Get identity
	hostEnv := os.Getenv("GH_HOST")
	identity, err := resolver.Resolve(selector, opts.Repo, hostEnv)
	if err != nil {
		return err
	}

	// Parse mode
	mode, err := await.ParseMode(opts.Mode)
	if err != nil {
		return err
	}

	// Create service
	service := &await.Service{API: apiClientFactory(identity.Host)}

	// Start polling
	startTime := time.Now()
	iteration := 0

	if !opts.Quiet {
		fmt.Fprintf(os.Stderr, "[%s] gh await: checking %s/%s #%d (mode=%s)\n",
			await.Now(), identity.Owner, identity.Repo, identity.Number, mode)
		fmt.Fprintf(os.Stderr, "         timeout=%s, interval=%s\n",
			await.SecondsToHuman(opts.Timeout), await.SecondsToHuman(opts.Interval))
		if opts.CheckOnly {
			fmt.Fprintf(os.Stderr, "         check-only mode — will exit immediately on first fetch\n")
		}
		fmt.Fprintf(os.Stderr, "\n")
	}

	for {
		iteration++
		data, err := service.Fetch(&identity, identity.Number)
		if err != nil {
			return fmt.Errorf("%w (exit code 2)", err)
		}

		pr := data.Repository.PullRequest

		// Print state
		if !opts.Quiet {
			printState(cmd, &identity, pr, mode)
		}

		// Check for work
		conditions := await.WorkConditions(pr, mode)
		if len(conditions) > 0 {
			if !opts.Quiet {
				fmt.Fprintf(os.Stderr, "\nWork detected: %s\n", joinConditions(conditions))
			}
			return nil // exit code 0
		}

		// Check-only mode: PR is clean
		if opts.CheckOnly {
			if !opts.Quiet {
				fmt.Fprintf(os.Stderr, "\nNothing to do. PR is clean.\n")
			}
			os.Exit(int(await.ExitClean))
		}

		// Check timeout
		elapsed := int(time.Since(startTime).Seconds())
		remaining := opts.Timeout - elapsed
		if remaining <= 0 {
			if !opts.Quiet {
				fmt.Fprintf(os.Stderr, "\nNothing to do — timeout reached without work detected.\n")
			}
			os.Exit(int(await.ExitClean))
		}

		if !opts.Quiet {
			fmt.Fprintf(os.Stderr, "  (will retry in %s, %s remaining)\n",
				await.SecondsToHuman(opts.Interval), await.SecondsToHuman(remaining))
		}

		time.Sleep(time.Duration(opts.Interval) * time.Second)
	}
}

func (o *awaitOptions) Validate() error {
	if o.Timeout < 0 {
		return errors.New("--timeout must be a non-negative integer")
	}
	if o.Interval <= 0 {
		return errors.New("--interval must be a positive integer (> 0)")
	}
	if o.Selector == "" && o.Pull == 0 {
		return errors.New("pull request number or URL is required")
	}
	return nil
}

func printState(cmd *cobra.Command, identity *resolver.Identity, pr *await.PullRequest, mode await.Mode) {
	ts := await.Now()

	unresolved := await.CountUnresolvedThreads(pr)
	general := len(pr.Comments.Nodes)
	failing := await.FailingChecks(pr)
	pending := await.PendingChecks(pr)

	fmt.Fprintf(os.Stderr, "[%s] PR %s/%s #%d state:\n", ts, identity.Owner, identity.Repo, identity.Number)

	if mode == await.ModeAll || mode == await.ModeComments {
		if unresolved > 0 {
			fmt.Fprintf(os.Stderr, "  ✗ Unresolved review threads: %d\n", unresolved)
		} else {
			fmt.Fprintf(os.Stderr, "  ✓ Unresolved review threads: %d\n", unresolved)
		}
		if general > 0 {
			fmt.Fprintf(os.Stderr, "  ○ General discussion comments: %d\n", general)
		}
	}

	if mode == await.ModeAll || mode == await.ModeConflicts {
		if pr.Mergeable == "CONFLICTING" {
			fmt.Fprintf(os.Stderr, "  ✗ Merge conflicts detected (mergeable=%s)\n", pr.Mergeable)
		} else {
			fmt.Fprintf(os.Stderr, "  ✓ No merge conflicts (mergeable=%s, status=%s)\n",
				pr.Mergeable, pr.MergeState)
		}
	}

	if mode == await.ModeAll || mode == await.ModeActions {
		if len(failing) > 0 {
			fmt.Fprintf(os.Stderr, "  ✗ GitHub Actions: %d failing, %d pending\n",
				len(failing), len(pending))
			for _, name := range failing {
				fmt.Fprintf(os.Stderr, "       - %s\n", name)
			}
		} else if len(pending) > 0 {
			fmt.Fprintf(os.Stderr, "  ◌ GitHub Actions: %d checks pending\n", len(pending))
		} else {
			fmt.Fprintf(os.Stderr, "  ✓ GitHub Actions: all checks passed or none configured\n")
		}
	}
}

func joinConditions(conditions []string) string {
	if len(conditions) == 0 {
		return ""
	}
	result := conditions[0]
	for i := 1; i < len(conditions); i++ {
		result += " " + conditions[i]
	}
	return result
}
