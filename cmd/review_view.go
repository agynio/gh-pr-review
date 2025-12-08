package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/agynio/gh-pr-review/internal/report"
	"github.com/agynio/gh-pr-review/internal/resolver"
)

func newReviewViewCommand() *cobra.Command {
	opts := &reviewViewOptions{}

	cmd := &cobra.Command{
		Use:   "view [<number> | <url>]",
		Short: "View a structured review summary (GraphQL)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.Selector = args[0]
			}
			return runReviewView(cmd, opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Repo, "repo", "R", "", "Repository in 'owner/repo' format")
	cmd.Flags().IntVar(&opts.Pull, "pr", 0, "Pull request number")
	cmd.Flags().StringVar(&opts.Reviewer, "reviewer", "", "Filter to a specific reviewer (login)")
	cmd.Flags().StringSliceVar(&opts.States, "states", nil, "Comma-separated review states (APPROVED, CHANGES_REQUESTED, COMMENTED, DISMISSED)")
	cmd.Flags().BoolVar(&opts.Unresolved, "unresolved", false, "Only include unresolved threads")
	cmd.Flags().BoolVar(&opts.NotOutdated, "not_outdated", false, "Exclude outdated threads")
	cmd.Flags().IntVar(&opts.TailReplies, "tail", 0, "Limit to the last N replies per thread (0 = all)")
	cmd.Flags().BoolVar(&opts.IncludeCommentNodeID, "include-comment-node-id", false, "Include comment_node_id fields for parent comments and replies")

	return cmd
}

type reviewViewOptions struct {
	Repo                 string
	Pull                 int
	Selector             string
	Reviewer             string
	States               []string
	Unresolved           bool
	NotOutdated          bool
	TailReplies          int
	IncludeCommentNodeID bool
}

func runReviewView(cmd *cobra.Command, opts *reviewViewOptions) error {
	if opts.TailReplies < 0 {
		return fmt.Errorf("invalid --tail value %d: must be non-negative", opts.TailReplies)
	}

	selector, err := resolver.NormalizeSelector(opts.Selector, opts.Pull)
	if err != nil {
		return err
	}

	states, statesProvided, err := parseStateFilters(opts.States)
	if err != nil {
		return err
	}

	identity, err := resolver.Resolve(selector, opts.Repo, os.Getenv("GH_HOST"))
	if err != nil {
		return err
	}

	service := report.NewService(apiClientFactory(identity.Host))
	output, err := service.Fetch(identity, report.Options{
		Reviewer:             strings.TrimSpace(opts.Reviewer),
		States:               states,
		StatesProvided:       statesProvided,
		RequireUnresolved:    opts.Unresolved,
		RequireNotOutdated:   opts.NotOutdated,
		TailReplies:          opts.TailReplies,
		IncludeCommentNodeID: opts.IncludeCommentNodeID,
	})
	if err != nil {
		return err
	}

	return encodeJSON(cmd, output)
}

func parseStateFilters(raw []string) ([]report.State, bool, error) {
	if len(raw) == 0 {
		return nil, false, nil
	}

	valid := map[string]report.State{
		"APPROVED":          report.StateApproved,
		"CHANGES_REQUESTED": report.StateChangesRequested,
		"COMMENTED":         report.StateCommented,
		"DISMISSED":         report.StateDismissed,
	}
	allowed := make([]string, 0, len(valid))
	for key := range valid {
		allowed = append(allowed, key)
	}
	sort.Strings(allowed)

	temp := make(map[report.State]struct{})
	states := make([]report.State, 0, len(raw))
	for _, entry := range raw {
		parts := strings.Split(entry, ",")
		for _, part := range parts {
			candidate := strings.ToUpper(strings.TrimSpace(part))
			if candidate == "" {
				continue
			}
			state, ok := valid[candidate]
			if !ok {
				return nil, false, fmt.Errorf("invalid review state %q (allowed: %s)", part, strings.Join(allowed, ", "))
			}
			if _, seen := temp[state]; seen {
				continue
			}
			temp[state] = struct{}{}
			states = append(states, state)
		}
	}

	if len(states) == 0 {
		return nil, false, fmt.Errorf("no valid states provided")
	}

	return states, true, nil
}
