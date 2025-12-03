package cmd

import (
	"errors"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Agyn-sandbox/gh-pr-review/internal/resolver"
	"github.com/Agyn-sandbox/gh-pr-review/internal/threads"
)

func newThreadsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "threads",
		Short: "Inspect and resolve pull request review threads",
	}

	cmd.AddCommand(newThreadsListCommand())
	cmd.AddCommand(newThreadsResolveCommand())
	cmd.AddCommand(newThreadsUnresolveCommand())
	cmd.AddCommand(newThreadsFindCommand())

	return cmd
}

func newThreadsListCommand() *cobra.Command {
	opts := &threadsListOptions{}

	cmd := &cobra.Command{
		Use:   "list [<number> | <url> | <owner>/<repo>#<number>]",
		Short: "List review threads for a pull request",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.Selector = args[0]
			}
			if !opts.JSON {
				return errors.New("specify --json to select JSON output")
			}
			return runThreadsList(cmd, opts)
		},
	}

	cmd.Flags().BoolVar(&opts.UnresolvedOnly, "unresolved", false, "Filter to unresolved threads only")
	cmd.Flags().BoolVar(&opts.MineOnly, "mine", false, "Show only threads involving or resolvable by the viewer")
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "Emit machine-readable JSON output")
	cmd.PersistentFlags().StringVarP(&opts.Repo, "repo", "R", "", "Repository in 'owner/repo' format")
	cmd.PersistentFlags().IntVar(&opts.Pull, "pr", 0, "Pull request number")

	return cmd
}

type threadsListOptions struct {
	Repo           string
	Pull           int
	Selector       string
	UnresolvedOnly bool
	MineOnly       bool
	JSON           bool
}

func runThreadsList(cmd *cobra.Command, opts *threadsListOptions) error {
	selector, err := resolver.NormalizeSelector(opts.Selector, opts.Pull)
	if err != nil {
		return err
	}

	hostEnv := os.Getenv("GH_HOST")
	identity, err := resolver.Resolve(selector, opts.Repo, hostEnv)
	if err != nil {
		return err
	}

	service := threads.NewService(apiClientFactory(identity.Host))
	payload, err := service.List(identity, threads.ListOptions{
		OnlyUnresolved: opts.UnresolvedOnly,
		MineOnly:       opts.MineOnly,
	})
	if err != nil {
		return err
	}

	return encodeJSON(cmd, payload)
}

func newThreadsResolveCommand() *cobra.Command {
	return newThreadsMutationCommand(true)
}

func newThreadsUnresolveCommand() *cobra.Command {
	return newThreadsMutationCommand(false)
}

func newThreadsMutationCommand(resolve bool) *cobra.Command {
	opts := &threadsMutationOptions{}

	use := "resolve"
	short := "Resolve a review thread"
	if !resolve {
		use = "unresolve"
		short = "Reopen a review thread"
	}

	cmd := &cobra.Command{
		Use:   use + " [<number> | <url> | <owner>/<repo>#<number>]",
		Short: short,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.Selector = args[0]
			}
			if !opts.JSON {
				return errors.New("specify --json to select JSON output")
			}
			if err := opts.Validate(); err != nil {
				return err
			}
			if resolve {
				return runThreadsResolve(cmd, opts)
			}
			return runThreadsUnresolve(cmd, opts)
		},
	}

	cmd.Flags().StringVar(&opts.ThreadID, "thread-id", "", "GraphQL node ID for the review thread")
	cmd.Flags().Int64Var(&opts.CommentID, "comment-id", 0, "Pull request review comment identifier")
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "Emit machine-readable JSON output")
	cmd.PersistentFlags().StringVarP(&opts.Repo, "repo", "R", "", "Repository in 'owner/repo' format")
	cmd.PersistentFlags().IntVar(&opts.Pull, "pr", 0, "Pull request number")

	return cmd
}

type threadsMutationOptions struct {
	Repo      string
	Pull      int
	Selector  string
	ThreadID  string
	CommentID int64
	JSON      bool
}

func (o *threadsMutationOptions) Validate() error {
	hasThread := strings.TrimSpace(o.ThreadID) != ""
	hasComment := o.CommentID > 0

	switch {
	case hasThread && hasComment:
		return errors.New("specify either --thread-id or --comment-id, not both")
	case !hasThread && !hasComment:
		return errors.New("must provide --thread-id or --comment-id")
	}
	return nil
}

func runThreadsResolve(cmd *cobra.Command, opts *threadsMutationOptions) error {
	return runThreadsMutation(cmd, opts, true)
}

func runThreadsUnresolve(cmd *cobra.Command, opts *threadsMutationOptions) error {
	return runThreadsMutation(cmd, opts, false)
}

func runThreadsMutation(cmd *cobra.Command, opts *threadsMutationOptions, resolve bool) error {
	selector, err := resolver.NormalizeSelector(opts.Selector, opts.Pull)
	if err != nil {
		return err
	}

	hostEnv := os.Getenv("GH_HOST")
	identity, err := resolver.Resolve(selector, opts.Repo, hostEnv)
	if err != nil {
		return err
	}

	service := threads.NewService(apiClientFactory(identity.Host))
	action := threads.ActionOptions{ThreadID: strings.TrimSpace(opts.ThreadID), CommentID: opts.CommentID}

	var result threads.ActionResult
	if resolve {
		result, err = service.Resolve(identity, action)
	} else {
		result, err = service.Unresolve(identity, action)
	}
	if err != nil {
		return err
	}
	return encodeJSON(cmd, result)
}

func newThreadsFindCommand() *cobra.Command {
	opts := &threadsFindOptions{}

	cmd := &cobra.Command{
		Use:   "find [<number> | <url> | <owner>/<repo>#<number>]",
		Short: "Locate a review thread by thread or comment identifier",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.Selector = args[0]
			}
			return runThreadsFind(cmd, opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Repo, "repo", "R", "", "Repository in 'owner/repo' format")
	cmd.Flags().IntVar(&opts.Pull, "pr", 0, "Pull request number")
	cmd.Flags().StringVar(&opts.ThreadID, "thread_id", "", "GraphQL thread node ID")
	cmd.Flags().Int64Var(&opts.CommentID, "comment_id", 0, "Review comment identifier")

	return cmd
}

type threadsFindOptions struct {
	Repo      string
	Pull      int
	Selector  string
	ThreadID  string
	CommentID int64
}

func runThreadsFind(cmd *cobra.Command, opts *threadsFindOptions) error {
	threadIDProvided := strings.TrimSpace(opts.ThreadID) != ""
	commentProvided := opts.CommentID > 0

	switch {
	case threadIDProvided && commentProvided:
		return errors.New("specify either --thread_id or --comment_id, not both")
	case !threadIDProvided && !commentProvided:
		return errors.New("must provide --thread_id or --comment_id")
	}

	selector, err := resolver.NormalizeSelector(opts.Selector, opts.Pull)
	if err != nil {
		return err
	}

	hostEnv := os.Getenv("GH_HOST")
	identity, err := resolver.Resolve(selector, opts.Repo, hostEnv)
	if err != nil {
		return err
	}

	service := threads.NewService(apiClientFactory(identity.Host))
	result, err := service.Find(identity, threads.FindOptions{
		ThreadID:  opts.ThreadID,
		CommentID: opts.CommentID,
	})
	if err != nil {
		return err
	}

	return encodeJSON(cmd, result)
}
