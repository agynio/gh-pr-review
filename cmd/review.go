package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Agyn-sandbox/gh-pr-review/internal/resolver"
	reviewsvc "github.com/Agyn-sandbox/gh-pr-review/internal/review"
)

func newReviewCommand() *cobra.Command {
	opts := &reviewOptions{Side: "RIGHT", Event: "COMMENT"}

	cmd := &cobra.Command{
		Use:   "review [<number> | <url> | <owner>/<repo>#<number>]",
		Short: "Manage pending reviews via GraphQL helpers",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.Selector = args[0]
			}
			return runReview(cmd, opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Repo, "repo", "R", "", "Repository in 'owner/repo' format")
	cmd.Flags().IntVar(&opts.Pull, "pr", 0, "Pull request number")

	cmd.Flags().BoolVar(&opts.Start, "start", false, "Open a pending review")
	cmd.Flags().BoolVar(&opts.AddComment, "add-comment", false, "Add an inline comment to a pending review")
	cmd.Flags().BoolVar(&opts.Submit, "submit", false, "Submit a pending review")

	cmd.Flags().StringVar(&opts.Commit, "commit", "", "Commit SHA for review start (defaults to current head)")
	cmd.Flags().StringVar(&opts.ReviewID, "review-id", "", "Review identifier")
	cmd.Flags().StringVar(&opts.Path, "path", "", "File path for inline comment")
	cmd.Flags().IntVar(&opts.Line, "line", 0, "Line number for inline comment")
	cmd.Flags().StringVar(&opts.Side, "side", opts.Side, "Diff side for inline comment (LEFT or RIGHT)")
	cmd.Flags().IntVar(&opts.StartLine, "start-line", 0, "Start line for multi-line comments")
	cmd.Flags().StringVar(&opts.StartSide, "start-side", "", "Start side for multi-line comments")
	cmd.Flags().StringVar(&opts.Body, "body", "", "Comment or review body")
	cmd.Flags().StringVar(&opts.Event, "event", opts.Event, "Review submission event (APPROVE, COMMENT, REQUEST_CHANGES)")

	return cmd
}

type reviewOptions struct {
	Repo     string
	Pull     int
	Selector string

	Start      bool
	AddComment bool
	Submit     bool

	Commit    string
	ReviewID  string
	Path      string
	Line      int
	Side      string
	StartLine int
	StartSide string
	Body      string
	Event     string
}

func runReview(cmd *cobra.Command, opts *reviewOptions) error {
	actions := []bool{opts.Start, opts.AddComment, opts.Submit}
	enabled := 0
	for _, flag := range actions {
		if flag {
			enabled++
		}
	}
	if enabled != 1 {
		return errors.New("specify exactly one of --start, --add-comment, or --submit")
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

	service := reviewsvc.NewService(apiClientFactory(identity.Host))

	switch {
	case opts.Start:
		return executeReviewStart(cmd, service, identity, opts)
	case opts.AddComment:
		return executeReviewAddComment(cmd, service, identity, opts)
	default: // Submit
		return executeReviewSubmit(cmd, service, identity, opts)
	}
}

func executeReviewStart(cmd *cobra.Command, service *reviewsvc.Service, pr resolver.Identity, opts *reviewOptions) error {
	state, err := service.Start(pr, strings.TrimSpace(opts.Commit))
	if err != nil {
		return err
	}
	return encodeJSON(cmd, state)
}

func executeReviewAddComment(cmd *cobra.Command, service *reviewsvc.Service, pr resolver.Identity, opts *reviewOptions) error {
	side, err := normalizeSide(opts.Side)
	if err != nil {
		return err
	}
	var startLine *int
	if opts.StartLine > 0 {
		startLine = &opts.StartLine
	}
	var startSide *string
	if opts.StartSide != "" {
		normalized, err := normalizeSide(opts.StartSide)
		if err != nil {
			return fmt.Errorf("invalid start-side: %w", err)
		}
		startSide = &normalized
	}

	input := reviewsvc.ThreadInput{
		ReviewID:  strings.TrimSpace(opts.ReviewID),
		Path:      strings.TrimSpace(opts.Path),
		Line:      opts.Line,
		Side:      side,
		StartLine: startLine,
		StartSide: startSide,
		Body:      opts.Body,
	}

	thread, err := service.AddThread(pr, input)
	if err != nil {
		return err
	}
	return encodeJSON(cmd, thread)
}

func executeReviewSubmit(cmd *cobra.Command, service *reviewsvc.Service, pr resolver.Identity, opts *reviewOptions) error {
	event, err := normalizeEvent(opts.Event)
	if err != nil {
		return err
	}
	input := reviewsvc.SubmitInput{
		ReviewID: strings.TrimSpace(opts.ReviewID),
		Event:    event,
		Body:     opts.Body,
	}
	state, err := service.Submit(pr, input)
	if err != nil {
		return err
	}
	return encodeJSON(cmd, state)
}

func normalizeSide(side string) (string, error) {
	s := strings.ToUpper(strings.TrimSpace(side))
	switch s {
	case "LEFT", "RIGHT":
		return s, nil
	case "":
		return "", errors.New("side is required")
	default:
		return "", fmt.Errorf("invalid side %q: must be LEFT or RIGHT", side)
	}
}

func normalizeEvent(event string) (string, error) {
	e := strings.ToUpper(strings.TrimSpace(event))
	switch e {
	case "APPROVE", "COMMENT", "REQUEST_CHANGES":
		return e, nil
	default:
		return "", fmt.Errorf("invalid event %q: must be APPROVE, COMMENT, or REQUEST_CHANGES", event)
	}
}
