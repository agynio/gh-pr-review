package cmd

import (
	"errors"
	"os"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/agynio/gh-pr-review/internal/draft"
	"github.com/agynio/gh-pr-review/internal/resolver"
)

func newDraftCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "draft",
		Short: "Manage pull request draft status",
	}

	cmd.AddCommand(newDraftMarkCommand())
	cmd.AddCommand(newDraftReadyCommand())
	cmd.AddCommand(newDraftStatusCommand())
	cmd.AddCommand(newDraftListCommand())

	return cmd
}

// draft mark [<number>]
func newDraftMarkCommand() *cobra.Command {
	opts := &draftActionOptions{}

	cmd := &cobra.Command{
		Use:   "mark [<number>]",
		Short: "Mark a pull request as draft",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				if prNum, err := strconv.Atoi(args[0]); err == nil {
					opts.PRNumber = prNum
				} else {
					opts.Selector = args[0]
				}
			}
			return runDraftMark(cmd, opts)
		},
	}

	cmd.PersistentFlags().StringVarP(&opts.Repo, "repo", "R", "", "Repository in 'owner/repo' format")
	cmd.PersistentFlags().IntVar(&opts.Pull, "pr", 0, "Pull request number")

	return cmd
}

// draft ready [<number>]
func newDraftReadyCommand() *cobra.Command {
	opts := &draftActionOptions{}

	cmd := &cobra.Command{
		Use:   "ready [<number>]",
		Short: "Mark a pull request as ready for review",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				if prNum, err := strconv.Atoi(args[0]); err == nil {
					opts.PRNumber = prNum
				} else {
					opts.Selector = args[0]
				}
			}
			return runDraftReady(cmd, opts)
		},
	}

	cmd.PersistentFlags().StringVarP(&opts.Repo, "repo", "R", "", "Repository in 'owner/repo' format")
	cmd.PersistentFlags().IntVar(&opts.Pull, "pr", 0, "Pull request number")

	return cmd
}

// draft status [<number>]
func newDraftStatusCommand() *cobra.Command {
	opts := &draftActionOptions{}

	cmd := &cobra.Command{
		Use:   "status [<number>]",
		Short: "Check if a pull request is a draft",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				if prNum, err := strconv.Atoi(args[0]); err == nil {
					opts.PRNumber = prNum
				} else {
					opts.Selector = args[0]
				}
			}
			return runDraftStatus(cmd, opts)
		},
	}

	cmd.PersistentFlags().StringVarP(&opts.Repo, "repo", "R", "", "Repository in 'owner/repo' format")
	cmd.PersistentFlags().IntVar(&opts.Pull, "pr", 0, "Pull request number")

	return cmd
}

// draft list
func newDraftListCommand() *cobra.Command {
	opts := &draftListOptions{}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all draft pull requests",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDraftList(cmd, opts)
		},
	}

	cmd.PersistentFlags().StringVarP(&opts.Repo, "repo", "R", "", "Repository in 'owner/repo' format")

	return cmd
}

type draftActionOptions struct {
	Repo     string
	Pull     int
	Selector string
	PRNumber int
}

type draftListOptions struct {
	Repo string
}

func runDraftMark(cmd *cobra.Command, opts *draftActionOptions) error {
	return runDraftAction(cmd, opts, true)
}

func runDraftReady(cmd *cobra.Command, opts *draftActionOptions) error {
	return runDraftAction(cmd, opts, false)
}

func runDraftAction(cmd *cobra.Command, opts *draftActionOptions, markAsDraft bool) error {
	var err error
	var selector string

	if opts.PRNumber != 0 {
		// Use the provided PR number directly
		selector = strconv.Itoa(opts.PRNumber)
	} else if opts.Selector != "" {
		// Use the provided selector
		selector = opts.Selector
	} else if opts.Pull != 0 {
		// Use the --pr flag
		selector = strconv.Itoa(opts.Pull)
	} else {
		return errors.New("pull request number is required")
	}

	inferPR(selector, &opts.Pull)
	normalizedSelector, err := resolver.NormalizeSelector(selector, opts.Pull)
	if err != nil {
		return err
	}

	inferRepo(&opts.Repo)
	hostEnv := os.Getenv("GH_HOST")
	identity, err := resolver.Resolve(normalizedSelector, opts.Repo, hostEnv)
	if err != nil {
		return err
	}

	service := draft.NewService(apiClientFactory(identity.Host))
	actionOpts := draft.ActionOptions{PRNumber: identity.Number}

	var result draft.ActionResult
	if markAsDraft {
		result, err = service.Draft(identity, actionOpts)
	} else {
		result, err = service.Ready(identity, actionOpts)
	}

	if err != nil {
		return err
	}

	return encodeJSON(cmd, result)
}

func runDraftStatus(cmd *cobra.Command, opts *draftActionOptions) error {
	var err error
	var selector string

	if opts.PRNumber != 0 {
		selector = strconv.Itoa(opts.PRNumber)
	} else if opts.Selector != "" {
		selector = opts.Selector
	} else if opts.Pull != 0 {
		selector = strconv.Itoa(opts.Pull)
	} else {
		return errors.New("pull request number is required")
	}

	inferPR(selector, &opts.Pull)
	normalizedSelector, err := resolver.NormalizeSelector(selector, opts.Pull)
	if err != nil {
		return err
	}

	inferRepo(&opts.Repo)
	hostEnv := os.Getenv("GH_HOST")
	identity, err := resolver.Resolve(normalizedSelector, opts.Repo, hostEnv)
	if err != nil {
		return err
	}

	service := draft.NewService(apiClientFactory(identity.Host))
	statusOpts := draft.ActionOptions{PRNumber: identity.Number}

	result, err := service.Status(identity, statusOpts)
	if err != nil {
		return err
	}

	return encodeJSON(cmd, result)
}

func runDraftList(cmd *cobra.Command, opts *draftListOptions) error {
	inferRepo(&opts.Repo)
	
	// Use a dummy selector for list operations
	selector := "1"
	normalizedSelector, err := resolver.NormalizeSelector(selector, 1)
	if err != nil {
		return err
	}

	hostEnv := os.Getenv("GH_HOST")
	identity, err := resolver.Resolve(normalizedSelector, opts.Repo, hostEnv)
	if err != nil {
		return err
	}

	service := draft.NewService(apiClientFactory(identity.Host))
	result, err := service.List(identity)
	if err != nil {
		return err
	}

	return encodeJSON(cmd, result)
}
