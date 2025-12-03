package cmd

import (
	"errors"
	"os"

	"github.com/spf13/cobra"

	"github.com/Agyn-sandbox/gh-pr-review/internal/comments"
	"github.com/Agyn-sandbox/gh-pr-review/internal/resolver"
)

func newCommentsCommand() *cobra.Command {
	opts := &commentsOptions{}

	cmd := &cobra.Command{
		Use:   "comments [<number> | <url> | <owner>/<repo>#<number>]",
		Short: "List and reply to pull request review comments",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.Selector = args[0]
			}
			if !opts.ListFlag {
				return errors.New("specify --list or use the reply subcommand")
			}
			if opts.ReviewID == 0 && !opts.Latest {
				return errors.New("use --review-id or --latest to select a review")
			}
			return runCommentsList(cmd, opts)
		},
	}

	cmd.PersistentFlags().StringVarP(&opts.Repo, "repo", "R", "", "Repository in 'owner/repo' format")
	cmd.PersistentFlags().IntVar(&opts.Pull, "pr", 0, "Pull request number")

	cmd.Flags().BoolVar(&opts.ListFlag, "list", false, "List review comments (required when invoking this command directly)")
	cmd.Flags().Int64Var(&opts.ReviewID, "review-id", 0, "Review identifier to list comments from")
	cmd.Flags().BoolVar(&opts.Latest, "latest", false, "Resolve the latest submitted review for the authenticated reviewer")
	cmd.Flags().StringVar(&opts.Reviewer, "reviewer", "", "Reviewer login when using --latest")
	_ = cmd.MarkFlagRequired("list")

	cmd.AddCommand(newCommentsReplyCommand(opts))
	cmd.AddCommand(newCommentsIDsCommand())

	return cmd
}

type commentsOptions struct {
	Repo     string
	Pull     int
	Selector string
	ListFlag bool
	ReviewID int64
	Latest   bool
	Reviewer string
}

func runCommentsList(cmd *cobra.Command, opts *commentsOptions) error {
	selector, err := resolver.NormalizeSelector(opts.Selector, opts.Pull)
	if err != nil {
		return err
	}

	hostEnv := os.Getenv("GH_HOST")
	identity, err := resolver.Resolve(selector, opts.Repo, hostEnv)
	if err != nil {
		return err
	}

	service := comments.NewService(apiClientFactory(identity.Host))

	listOpts := comments.ListOptions{
		ReviewID: opts.ReviewID,
		Latest:   opts.Latest,
		Reviewer: opts.Reviewer,
	}

	data, err := service.List(identity, listOpts)
	if err != nil {
		return err
	}

	return encodeJSON(cmd, data)
}

func newCommentsReplyCommand(parent *commentsOptions) *cobra.Command {
	opts := &commentsReplyOptions{}

	cmd := &cobra.Command{
		Use:   "reply [<number> | <url> | <owner>/<repo>#<number>]",
		Short: "Reply to a pull request review comment",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.Selector = args[0]
			}
			// Inherit repo/pr from parent when unset locally.
			if opts.Repo == "" {
				opts.Repo = parent.Repo
			}
			if opts.Pull == 0 {
				opts.Pull = parent.Pull
			}
			return runCommentsReply(cmd, opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Repo, "repo", "R", "", "Repository in 'owner/repo' format")
	cmd.Flags().IntVar(&opts.Pull, "pr", 0, "Pull request number")
	cmd.Flags().Int64Var(&opts.CommentID, "comment-id", 0, "Review comment identifier to reply to")
	cmd.Flags().StringVar(&opts.Body, "body", "", "Reply text")
	_ = cmd.MarkFlagRequired("comment-id")
	_ = cmd.MarkFlagRequired("body")

	return cmd
}

type commentsReplyOptions struct {
	Repo      string
	Pull      int
	Selector  string
	CommentID int64
	Body      string
}

func runCommentsReply(cmd *cobra.Command, opts *commentsReplyOptions) error {
	selector, err := resolver.NormalizeSelector(opts.Selector, opts.Pull)
	if err != nil {
		return err
	}

	hostEnv := os.Getenv("GH_HOST")
	identity, err := resolver.Resolve(selector, opts.Repo, hostEnv)
	if err != nil {
		return err
	}

	service := comments.NewService(apiClientFactory(identity.Host))

	reply, err := service.Reply(identity, comments.ReplyOptions{
		CommentID: opts.CommentID,
		Body:      opts.Body,
	})
	if err != nil {
		return err
	}
	return encodeJSON(cmd, reply)
}

func newCommentsIDsCommand() *cobra.Command {
	opts := &commentsIDsOptions{}

	cmd := &cobra.Command{
		Use:   "ids [<number> | <url> | <owner>/<repo>#<number>]",
		Short: "List review comment identifiers with text",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.Selector = args[0]
			}
			return runCommentsIDs(cmd, opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Repo, "repo", "R", "", "Repository in 'owner/repo' format")
	cmd.Flags().IntVar(&opts.Pull, "pr", 0, "Pull request number")
	cmd.Flags().Int64Var(&opts.ReviewID, "review_id", 0, "Review identifier to target")
	cmd.Flags().BoolVar(&opts.Latest, "latest", false, "Resolve the latest submitted review (defaults to authenticated reviewer)")
	cmd.Flags().StringVar(&opts.Reviewer, "reviewer", "", "Reviewer login when using --latest")
	cmd.Flags().IntVar(&opts.Limit, "limit", 0, "Maximum number of comments to emit (0 = all)")
	cmd.Flags().IntVar(&opts.PerPage, "per_page", 0, "Number of comments per REST request")
	cmd.Flags().IntVar(&opts.Page, "page", 0, "Starting REST page index")

	return cmd
}

type commentsIDsOptions struct {
	Repo     string
	Pull     int
	Selector string
	ReviewID int64
	Latest   bool
	Reviewer string
	Limit    int
	PerPage  int
	Page     int
}

func runCommentsIDs(cmd *cobra.Command, opts *commentsIDsOptions) error {
	if opts.ReviewID == 0 && !opts.Latest {
		return errors.New("use --review_id or --latest to select a review")
	}
	if opts.ReviewID > 0 && opts.Latest {
		return errors.New("specify either --review_id or --latest, not both")
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

	service := comments.NewService(apiClientFactory(identity.Host))
	entries, err := service.IDs(identity, comments.IDsOptions{
		ReviewID: opts.ReviewID,
		Latest:   opts.Latest,
		Reviewer: opts.Reviewer,
		Limit:    opts.Limit,
		PerPage:  opts.PerPage,
		Page:     opts.Page,
	})
	if err != nil {
		return err
	}

	return encodeJSON(cmd, entries)
}
