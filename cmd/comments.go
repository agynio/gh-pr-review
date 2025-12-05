package cmd

import (
	"errors"
	"os"

	"github.com/spf13/cobra"

	"github.com/Agyn-sandbox/gh-pr-review/internal/comments"
	"github.com/Agyn-sandbox/gh-pr-review/internal/resolver"
)

type commentsOptions struct {
	Repo string
	Pull int
}

func newCommentsCommand() *cobra.Command {
	opts := &commentsOptions{}

	cmd := &cobra.Command{
		Use:   "comments",
		Short: "Reply to pull request review threads",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := cmd.Help(); err != nil {
				return err
			}
			return errors.New("use 'gh pr-review comments reply' to respond to a review thread; run 'gh pr-review review report' to locate thread IDs")
		},
	}

	cmd.PersistentFlags().StringVarP(&opts.Repo, "repo", "R", "", "Repository in 'owner/repo' format")
	cmd.PersistentFlags().IntVar(&opts.Pull, "pr", 0, "Pull request number")

	cmd.AddCommand(newCommentsReplyCommand(opts))

	return cmd
}

func newCommentsReplyCommand(parent *commentsOptions) *cobra.Command {
	opts := &commentsReplyOptions{}

	cmd := &cobra.Command{
		Use:   "reply [<number> | <url> | <owner>/<repo>#<number>]",
		Short: "Reply to a pull request review thread",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.Selector = args[0]
			}
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
	cmd.Flags().StringVar(&opts.ThreadID, "thread-id", "", "Review thread identifier to reply to")
	cmd.Flags().StringVar(&opts.ReviewID, "review-id", "", "GraphQL review identifier when replying inside a pending review")
	cmd.Flags().StringVar(&opts.Body, "body", "", "Reply text")
	cmd.Flags().BoolVar(&opts.Concise, "concise", false, "Emit minimal reply payload { \"id\" }")
	_ = cmd.MarkFlagRequired("thread-id")
	_ = cmd.MarkFlagRequired("body")

	return cmd
}

type commentsReplyOptions struct {
	Repo     string
	Pull     int
	Selector string
	ThreadID string
	ReviewID string
	Body     string
	Concise  bool
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
		ThreadID: opts.ThreadID,
		ReviewID: opts.ReviewID,
		Body:     opts.Body,
	})
	if err != nil {
		return err
	}
	if opts.Concise {
		if reply.ID == "" {
			return errors.New("reply response missing id")
		}
		return encodeJSON(cmd, map[string]string{"id": reply.ID})
	}
	return encodeJSON(cmd, reply)
}
