package cmd

import (
	"errors"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/agynio/gh-pr-review/internal/resolver"
	"github.com/agynio/gh-pr-review/internal/threads"
)

func newThreadsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "threads",
		Short: "Inspect and resolve pull request review threads",
	}

	cmd.AddCommand(newThreadsListCommand())
	cmd.AddCommand(newThreadsResolveCommand())
	cmd.AddCommand(newThreadsUnresolveCommand())

	return cmd
}

func newThreadsListCommand() *cobra.Command {
	opts := &threadsListOptions{}

	cmd := &cobra.Command{
		Use:   "list [<number> | <url>]",
		Short: "List review threads for a pull request",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.Selector = args[0]
			}
			return runThreadsList(cmd, opts)
		},
	}

	cmd.Flags().BoolVar(&opts.UnresolvedOnly, "unresolved", false, "Filter to unresolved threads only")
	cmd.Flags().BoolVar(&opts.MineOnly, "mine", false, "Show only threads involving or resolvable by the viewer")
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
		Use:   use + " [<number> | <url>]",
		Short: short,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.Selector = args[0]
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
	cmd.PersistentFlags().StringVarP(&opts.Repo, "repo", "R", "", "Repository in 'owner/repo' format")
	cmd.PersistentFlags().IntVar(&opts.Pull, "pr", 0, "Pull request number")

	return cmd
}

type threadsMutationOptions struct {
	Repo     string
	Pull     int
	Selector string
	ThreadID string
}

func (o *threadsMutationOptions) Validate() error {
	if strings.TrimSpace(o.ThreadID) == "" {
		return errors.New("--thread-id is required")
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
	action := threads.ActionOptions{ThreadID: strings.TrimSpace(opts.ThreadID)}

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
