package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/agynio/gh-pr-review/internal/reactions"
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
	cmd.AddCommand(newThreadsResolveAllCommand())

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
	cmd.Flags().StringVar(&opts.Author, "author", "", "Filter threads to those containing a comment by this author login (case-insensitive)")
	cmd.Flags().StringVar(&opts.Since, "since", "", "Only include threads updated at or after this RFC3339 timestamp")
	cmd.Flags().StringVarP(&opts.Output, "output", "o", "", "Output format: 'ids' prints one thread ID per line")
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
	Author         string
	Since          string
	Output         string
}


func runThreadsList(cmd *cobra.Command, opts *threadsListOptions) error {
	inferPR(opts.Selector, &opts.Pull)
	selector, err := resolver.NormalizeSelector(opts.Selector, opts.Pull)
	if err != nil {
		return err
	}

	inferRepo(&opts.Repo)
	hostEnv := os.Getenv("GH_HOST")
	identity, err := resolver.Resolve(selector, opts.Repo, hostEnv)
	if err != nil {
		return err
	}

	listOpts := threads.ListOptions{
		OnlyUnresolved: opts.UnresolvedOnly,
		MineOnly:       opts.MineOnly,
		Author:         strings.TrimSpace(opts.Author),
	}

	if opts.Since != "" {
		t, err := time.Parse(time.RFC3339, opts.Since)
		if err != nil {
			return fmt.Errorf("--since: invalid RFC3339 timestamp %q: %w", opts.Since, err)
		}
		listOpts.Since = t
	}

	service := threads.NewService(apiClientFactory(identity.Host))
	payload, err := service.List(identity, listOpts)
	if err != nil {
		return err
	}

	if strings.TrimSpace(opts.Output) == "ids" {
		ids := make([]string, len(payload))
		for i, t := range payload {
			ids[i] = t.ThreadID
		}
		fmt.Fprintln(cmd.OutOrStdout(), strings.Join(ids, "\n"))
		return nil
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
	if resolve {
		cmd.Flags().StringVar(&opts.Commit, "commit", "", "Post a reply linking to this commit SHA before resolving")
		cmd.Flags().StringVar(&opts.React, "react", "", "Add a reaction to the first comment (thumbs_up, thumbs_down, laugh, hooray, confused, heart, rocket, eyes)")
		cmd.Flags().StringVar(&opts.Message, "message", "", "Post a reply message before resolving (used with --react thumbs_down)")
	}

	return cmd
}

type threadsMutationOptions struct {
	Repo     string
	Pull     int
	Selector string
	ThreadID string
	Commit   string
	React    string
	Message  string
}

func (o *threadsMutationOptions) Validate() error {
	if strings.TrimSpace(o.ThreadID) == "" {
		return errors.New("--thread-id is required")
	}
	if o.React != "" {
		if _, ok := reactions.ValidReactions[o.React]; !ok {
			return fmt.Errorf("--react: invalid reaction %q; valid values: thumbs_up, thumbs_down, laugh, hooray, confused, heart, rocket, eyes", o.React)
		}
	}
	return nil
}

func runThreadsResolve(cmd *cobra.Command, opts *threadsMutationOptions) error {
	if opts.Commit == "" && opts.React != "thumbs_down" {
		return fmt.Errorf("must provide --commit (bug was fixed) or --react thumbs_down --message 'reason' (not a bug)")
	}
	return runThreadsMutation(cmd, opts, true)
}

func runThreadsUnresolve(cmd *cobra.Command, opts *threadsMutationOptions) error {
	return runThreadsMutation(cmd, opts, false)
}

func runThreadsMutation(cmd *cobra.Command, opts *threadsMutationOptions, resolve bool) error {
	inferPR(opts.Selector, &opts.Pull)
	selector, err := resolver.NormalizeSelector(opts.Selector, opts.Pull)
	if err != nil {
		return err
	}

	inferRepo(&opts.Repo)
	hostEnv := os.Getenv("GH_HOST")
	identity, err := resolver.Resolve(selector, opts.Repo, hostEnv)
	if err != nil {
		return err
	}

	service := threads.NewService(apiClientFactory(identity.Host))
	commit := strings.TrimSpace(opts.Commit)
	if commit != "" {
		commit, err = resolveCommitRef(commit)
		if err != nil {
			return fmt.Errorf("--commit: %w", err)
		}
	}
	var reaction string
	if opts.React != "" {
		reaction = reactions.ValidReactions[opts.React]
	}
	action := threads.ActionOptions{
		ThreadID: strings.TrimSpace(opts.ThreadID),
		Commit:   commit,
		React:    reaction,
		Message:  strings.TrimSpace(opts.Message),
	}

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


// newThreadsResolveAllCommand creates the resolve-all subcommand.
func newThreadsResolveAllCommand() *cobra.Command {
	opts := &threadsResolveAllOptions{}

	cmd := &cobra.Command{
		Use:   "resolve-all [<number> | <url>]",
		Short: "Resolve all matching review threads for a pull request",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.Selector = args[0]
			}
			return runThreadsResolveAll(cmd, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Author, "author", "", "Only resolve threads by this author")
	cmd.Flags().StringVar(&opts.Commit, "commit", "", "Attach commit SHA to each resolution reply")
	cmd.Flags().StringVar(&opts.React, "react", "", "Add a reaction to each thread's first comment after resolving (thumbs_up, thumbs_down, laugh, hooray, confused, heart, rocket, eyes)")
	cmd.Flags().BoolVar(&opts.IncludeResolved, "include-resolved", false, "Also resolve already-resolved threads")
	cmd.PersistentFlags().StringVarP(&opts.Repo, "repo", "R", "", "Repository in 'owner/repo' format (required)")
	cmd.PersistentFlags().IntVar(&opts.Pull, "pr", 0, "Pull request number")

	return cmd
}

type threadsResolveAllOptions struct {
	Repo            string
	Pull            int
	Selector        string
	Author          string
	Commit          string
	React           string
	IncludeResolved bool
}

func runThreadsResolveAll(cmd *cobra.Command, opts *threadsResolveAllOptions) error {
	if opts.Commit == "" {
		return fmt.Errorf("--commit is required for resolve-all")
	}

	if opts.React != "" {
		if _, ok := reactions.ValidReactions[opts.React]; !ok {
			return fmt.Errorf("--react: invalid reaction %q; valid values: thumbs_up, thumbs_down, laugh, hooray, confused, heart, rocket, eyes", opts.React)
		}
	}

	inferPR(opts.Selector, &opts.Pull)
	selector, err := resolver.NormalizeSelector(opts.Selector, opts.Pull)
	if err != nil {
		return err
	}

	inferRepo(&opts.Repo)
	hostEnv := os.Getenv("GH_HOST")
	identity, err := resolver.Resolve(selector, opts.Repo, hostEnv)
	if err != nil {
		return err
	}

	commit := strings.TrimSpace(opts.Commit)
	if commit != "" {
		commit, err = resolveCommitRef(commit)
		if err != nil {
			return fmt.Errorf("--commit: %w", err)
		}
	}

	var reaction string
	if opts.React != "" {
		reaction = reactions.ValidReactions[opts.React]
	}

	service := threads.NewService(apiClientFactory(identity.Host))
	results, err := service.ResolveAll(identity, threads.ResolveAllOptions{
		Author:     strings.TrimSpace(opts.Author),
		Commit:     commit,
		React:      reaction,
		Unresolved: !opts.IncludeResolved,
	})
	if err != nil {
		return err
	}
	return encodeJSON(cmd, results)
}

// resolveCommitRef converts symbolic git refs (e.g. HEAD, branch names) to
// short SHAs. If ref already looks like a hex SHA it is returned unchanged.
var hexSHARe = regexp.MustCompile(`(?i)^[0-9a-f]{7,40}$`)

func resolveCommitRef(ref string) (string, error) {
	if hexSHARe.MatchString(ref) {
		return ref, nil
	}
	out, err := exec.Command("git", "rev-parse", "--short", "--", ref).Output()
	if err != nil {
		return "", fmt.Errorf("cannot resolve git ref %q: %w", ref, err)
	}
	return strings.TrimSpace(string(out)), nil
}
