package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/spf13/cobra"

	"github.com/agynio/gh-pr-review/internal/resolver"
	"github.com/agynio/gh-pr-review/internal/watch"
)

func newWatchCommand() *cobra.Command {
	opts := &watchOptions{}

	cmd := &cobra.Command{
		Use:   "watch [<number> | <url>]",
		Short: "Watch for new PR comments and exit when they arrive",
		Long: `Watch for new review comments or issue comments on a pull request.

The command polls for new comments at a configurable interval (default 10s).
When new comments are detected, it debounces for 5 seconds to collect any
additional comments that arrive in quick succession (e.g., from a batch of
commits being reviewed together).

After the debounce period with no new comments, or when the timeout is reached,
the command exits and outputs all new comments as JSON.

Exit codes:
  0  New comments found
  1  Error occurred  
  2  Timed out with no new comments`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.Selector = args[0]
			}
			return runWatch(cmd, opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Repo, "repo", "R", "", "Repository in 'owner/repo' format")
	cmd.Flags().IntVar(&opts.Pull, "pr", 0, "Pull request number")
	cmd.Flags().IntVarP(&opts.Interval, "interval", "i", 10, "Polling interval in seconds")
	cmd.Flags().IntVar(&opts.Debounce, "debounce", 5, "Debounce duration in seconds")
	cmd.Flags().IntVar(&opts.Timeout, "timeout", 3600, "Maximum watch duration in seconds (default 1 hour)")
	cmd.Flags().BoolVar(&opts.IncludeIssue, "issue-comments", true, "Include issue comments (not just review comments)")

	return cmd
}

type watchOptions struct {
	Repo         string
	Pull         int
	Selector     string
	Interval     int
	Debounce     int
	Timeout      int
	IncludeIssue bool
}

func runWatch(cmd *cobra.Command, opts *watchOptions) error {
	selector, err := resolver.NormalizeSelector(opts.Selector, opts.Pull)
	if err != nil {
		return err
	}

	hostEnv := os.Getenv("GH_HOST")
	identity, err := resolver.Resolve(selector, opts.Repo, hostEnv)
	if err != nil {
		return err
	}

	service := watch.NewService(apiClientFactory(identity.Host))

	watchOpts := watch.WatchOptions{
		Interval:     time.Duration(opts.Interval) * time.Second,
		Debounce:     time.Duration(opts.Debounce) * time.Second,
		Timeout:      time.Duration(opts.Timeout) * time.Second,
		IncludeIssue: opts.IncludeIssue,
	}

	// Set up signal handling for graceful exit
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	go func() {
		<-sigCh
		cancel()
	}()

	fmt.Fprintf(os.Stderr, "Watching %s/%s#%d for new comments (interval=%ds, debounce=%ds, timeout=%ds)...\n",
		identity.Owner, identity.Repo, identity.Number,
		opts.Interval, opts.Debounce, opts.Timeout)

	result, err := service.Watch(ctx, identity, watchOpts)
	if err != nil {
		return err
	}

	if err := encodeJSON(cmd, result); err != nil {
		return err
	}

	// Exit with code 2 if timed out with no comments
	if result.TimedOut && len(result.Comments) == 0 {
		os.Exit(2)
	}

	return nil
}
