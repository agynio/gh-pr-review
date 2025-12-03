package threads

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Agyn-sandbox/gh-pr-review/internal/ghcli"
	"github.com/Agyn-sandbox/gh-pr-review/internal/resolver"
)

// Service exposes pull request review thread operations.
type Service struct {
	API ghcli.API
}

// NewService constructs a Service with the provided API client.
func NewService(api ghcli.API) *Service {
	return &Service{API: api}
}

// ListOptions configures list filtering.
type ListOptions struct {
	OnlyUnresolved bool
	MineOnly       bool
}

// Thread represents a normalized review thread payload for JSON output.
type Thread struct {
	ThreadID   string     `json:"threadId"`
	IsResolved bool       `json:"isResolved"`
	ResolvedBy *string    `json:"resolvedBy,omitempty"`
	UpdatedAt  *time.Time `json:"updatedAt,omitempty"`
	Path       string     `json:"path"`
	Line       *int       `json:"line,omitempty"`
	IsOutdated bool       `json:"isOutdated"`
}

// ActionOptions controls resolve/unresolve operations.
type ActionOptions struct {
	ThreadID  string
	CommentID int64
}

// ActionResult captures the outcome of a resolve/unresolve mutation.
type ActionResult struct {
	ThreadID   string `json:"threadId"`
	IsResolved bool   `json:"isResolved"`
	Changed    bool   `json:"changed"`
}

// FindOptions configures lookup of a specific review thread.
type FindOptions struct {
	ThreadID  string
	CommentID int64
}

// FindResult exposes the minimal schema required by callers when locating threads.
type FindResult struct {
	ID         string `json:"id"`
	IsResolved bool   `json:"isResolved"`
}

type pullContext struct {
	identity resolver.Identity
	nodeID   string
}

// List fetches review threads for the provided pull request, applies filters, and returns sorted results.
func (s *Service) List(pr resolver.Identity, opts ListOptions) ([]Thread, error) {
	ctx, err := s.loadPullContext(pr)
	if err != nil {
		return nil, err
	}

	nodes, err := s.collectThreads(ctx)
	if err != nil {
		return nil, err
	}

	var allThreads []Thread

	for _, node := range nodes {
		if opts.OnlyUnresolved && node.IsResolved {
			continue
		}

		mine := node.ViewerCanResolve || node.ViewerCanUnresolve
		var (
			latest   time.Time
			hasStamp bool
		)

		for _, comment := range node.Comments.Nodes {
			if comment.ViewerDidAuthor {
				mine = true
			}
			if !hasStamp || comment.UpdatedAt.After(latest) {
				latest = comment.UpdatedAt
				hasStamp = true
			}
		}

		if opts.MineOnly && !mine {
			continue
		}

		var resolvedBy *string
		if node.ResolvedBy != nil && node.ResolvedBy.Login != "" {
			login := node.ResolvedBy.Login
			resolvedBy = &login
		}

		var updatedAt *time.Time
		if hasStamp {
			ts := latest
			updatedAt = &ts
		}

		var linePtr *int
		if node.Line != nil {
			value := *node.Line
			linePtr = &value
		}

		allThreads = append(allThreads, Thread{
			ThreadID:   node.ID,
			IsResolved: node.IsResolved,
			ResolvedBy: resolvedBy,
			UpdatedAt:  updatedAt,
			Path:       node.Path,
			Line:       linePtr,
			IsOutdated: node.IsOutdated,
		})
	}

	sort.SliceStable(allThreads, func(i, j int) bool {
		left := allThreads[i].UpdatedAt
		right := allThreads[j].UpdatedAt

		switch {
		case left == nil && right == nil:
			return allThreads[i].ThreadID < allThreads[j].ThreadID
		case left == nil:
			return false
		case right == nil:
			return true
		default:
			if left.Equal(*right) {
				return allThreads[i].ThreadID < allThreads[j].ThreadID
			}
			return left.After(*right)
		}
	})

	return allThreads, nil
}

// Resolve marks a thread as resolved when permissions and current state allow it.
func (s *Service) Resolve(pr resolver.Identity, opts ActionOptions) (ActionResult, error) {
	return s.changeResolution(pr, opts, true)
}

// Unresolve reopens a thread when permitted.
func (s *Service) Unresolve(pr resolver.Identity, opts ActionOptions) (ActionResult, error) {
	return s.changeResolution(pr, opts, false)
}

// Find returns the thread identifier and resolution state derived from either a thread or comment ID.
func (s *Service) Find(pr resolver.Identity, opts FindOptions) (FindResult, error) {
	threadID := strings.TrimSpace(opts.ThreadID)
	hasThread := threadID != ""
	hasComment := opts.CommentID > 0

	switch {
	case hasThread && hasComment:
		return FindResult{}, errors.New("specify either --thread_id or --comment_id, not both")
	case !hasThread && !hasComment:
		return FindResult{}, errors.New("must provide --thread_id or --comment_id")
	}

	ctx, err := s.loadPullContext(pr)
	if err != nil {
		return FindResult{}, err
	}

	if !hasThread {
		resolvedID, err := s.resolveThreadID(ctx, ActionOptions{CommentID: opts.CommentID})
		if err != nil {
			return FindResult{}, err
		}
		threadID = resolvedID
	}

	thread, err := s.fetchThread(ctx.identity.Host, threadID)
	if err != nil {
		return FindResult{}, err
	}

	return FindResult{ID: thread.ID, IsResolved: thread.IsResolved}, nil
}

type threadsQueryResponse struct {
	Node *struct {
		ReviewThreads *struct {
			Nodes    []threadNode `json:"nodes"`
			PageInfo *struct {
				HasNextPage bool   `json:"hasNextPage"`
				EndCursor   string `json:"endCursor"`
			} `json:"pageInfo"`
		} `json:"reviewThreads"`
	} `json:"node"`
}

type threadNode struct {
	ID                 string `json:"id"`
	IsResolved         bool   `json:"isResolved"`
	IsOutdated         bool   `json:"isOutdated"`
	Path               string `json:"path"`
	Line               *int   `json:"line"`
	ViewerCanResolve   bool   `json:"viewerCanResolve"`
	ViewerCanUnresolve bool   `json:"viewerCanUnresolve"`
	ResolvedBy         *struct {
		Login string `json:"login"`
	} `json:"resolvedBy"`
	Comments struct {
		Nodes []struct {
			ViewerDidAuthor bool      `json:"viewerDidAuthor"`
			UpdatedAt       time.Time `json:"updatedAt"`
			DatabaseID      int64     `json:"databaseId"`
		} `json:"nodes"`
	} `json:"comments"`
}

func (s *Service) fetchThreads(nodeID string, after *string) (*threadsQueryResponse, error) {
	variables := map[string]interface{}{
		"id": nodeID,
	}
	if after != nil {
		variables["after"] = *after
	}

	var resp threadsQueryResponse
	if err := s.API.GraphQL(listThreadsQuery, variables, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) collectThreads(ctx pullContext) ([]threadNode, error) {
	var (
		allThreads []threadNode
		after      *string
	)

	for {
		resp, err := s.fetchThreads(ctx.nodeID, after)
		if err != nil {
			return nil, err
		}

		node := resp.Node
		if node == nil || node.ReviewThreads == nil {
			return nil, fmt.Errorf("pull request %d not found on %s", ctx.identity.Number, ctx.identity.Host)
		}

		threads := node.ReviewThreads
		allThreads = append(allThreads, threads.Nodes...)

		if threads.PageInfo == nil || !threads.PageInfo.HasNextPage {
			break
		}
		cursor := threads.PageInfo.EndCursor
		after = &cursor
	}

	return allThreads, nil
}

func (s *Service) canonicalizeIdentity(pr resolver.Identity) (resolver.Identity, error) {
	var repo struct {
		FullName string `json:"full_name"`
	}
	path := fmt.Sprintf("repos/%s/%s", pr.Owner, pr.Repo)
	if err := s.API.REST("GET", path, nil, nil, &repo); err != nil {
		return resolver.Identity{}, fmt.Errorf("repository %s/%s not found on %s: %w", pr.Owner, pr.Repo, pr.Host, err)
	}

	if repo.FullName != "" {
		parts := strings.Split(repo.FullName, "/")
		if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
			pr.Owner = parts[0]
			pr.Repo = parts[1]
		}
	}

	return pr, nil
}

func (s *Service) loadPullContext(pr resolver.Identity) (pullContext, error) {
	canonical, err := s.canonicalizeIdentity(pr)
	if err != nil {
		return pullContext{}, err
	}

	var pull struct {
		NodeID string `json:"node_id"`
	}
	path := fmt.Sprintf("repos/%s/%s/pulls/%d", canonical.Owner, canonical.Repo, canonical.Number)
	if err := s.API.REST("GET", path, nil, nil, &pull); err != nil {
		return pullContext{}, fmt.Errorf("pull request %d not found on %s: %w", canonical.Number, canonical.Host, err)
	}
	if strings.TrimSpace(pull.NodeID) == "" {
		return pullContext{}, fmt.Errorf("pull request %d missing node identifier on %s", canonical.Number, canonical.Host)
	}

	return pullContext{identity: canonical, nodeID: pull.NodeID}, nil
}

func (s *Service) changeResolution(pr resolver.Identity, opts ActionOptions, resolve bool) (ActionResult, error) {
	ctx, err := s.loadPullContext(pr)
	if err != nil {
		return ActionResult{}, err
	}

	threadID, err := s.resolveThreadID(ctx, opts)
	if err != nil {
		return ActionResult{}, err
	}

	thread, err := s.fetchThread(ctx.identity.Host, threadID)
	if err != nil {
		return ActionResult{}, err
	}

	desired := resolve
	if thread.IsResolved == desired {
		return ActionResult{ThreadID: thread.ID, IsResolved: thread.IsResolved, Changed: false}, nil
	}

	if resolve && !thread.ViewerCanResolve {
		return ActionResult{}, errors.New("viewer cannot resolve this thread")
	}
	if !resolve && !thread.ViewerCanUnresolve {
		return ActionResult{}, errors.New("viewer cannot unresolve this thread")
	}

	if resolve {
		return s.performResolve(threadID)
	}
	return s.performUnresolve(threadID)
}

func (s *Service) resolveThreadID(ctx pullContext, opts ActionOptions) (string, error) {
	if opts.ThreadID != "" && opts.CommentID > 0 {
		return "", errors.New("specify either --thread-id or --comment-id, not both")
	}
	if opts.ThreadID == "" && opts.CommentID == 0 {
		return "", errors.New("must provide --thread-id or --comment-id")
	}
	if opts.ThreadID != "" {
		return opts.ThreadID, nil
	}

	var comment struct {
		NodeID string `json:"node_id"`
	}
	path := fmt.Sprintf("repos/%s/%s/pulls/comments/%d", ctx.identity.Owner, ctx.identity.Repo, opts.CommentID)
	if err := s.API.REST("GET", path, nil, nil, &comment); err != nil {
		return "", fmt.Errorf("comment %d not found on %s: %w", opts.CommentID, ctx.identity.Host, err)
	}
	if strings.TrimSpace(comment.NodeID) == "" {
		return "", fmt.Errorf("comment %d missing node identifier on %s", opts.CommentID, ctx.identity.Host)
	}

	threadID, err := s.lookupThreadID(comment.NodeID)
	if err != nil {
		if isReviewThreadSchemaError(err) {
			return s.findThreadByComment(ctx, opts.CommentID)
		}
		return "", err
	}
	return threadID, nil
}

func (s *Service) findThreadByComment(ctx pullContext, commentID int64) (string, error) {
	nodes, err := s.collectThreads(ctx)
	if err != nil {
		return "", err
	}

	for _, node := range nodes {
		for _, comment := range node.Comments.Nodes {
			if comment.DatabaseID == commentID {
				return node.ID, nil
			}
		}
	}

	return "", fmt.Errorf("comment %d not found in pull request %s/%s#%d on %s", commentID, ctx.identity.Owner, ctx.identity.Repo, ctx.identity.Number, ctx.identity.Host)
}

func (s *Service) lookupThreadID(commentNodeID string) (string, error) {
	variables := map[string]interface{}{"id": commentNodeID}
	var resp struct {
		Node *struct {
			PullRequestReviewThread *struct {
				ID string `json:"id"`
			} `json:"pullRequestReviewThread"`
		} `json:"node"`
	}
	if err := s.API.GraphQL(commentThreadQuery, variables, &resp); err != nil {
		return "", err
	}
	if resp.Node == nil || resp.Node.PullRequestReviewThread == nil {
		return "", fmt.Errorf("no thread found for comment node %s", commentNodeID)
	}
	return resp.Node.PullRequestReviewThread.ID, nil
}

func (s *Service) fetchThread(host, threadID string) (*threadDetails, error) {
	variables := map[string]interface{}{"id": threadID}
	var resp struct {
		Node *threadDetails `json:"node"`
	}
	if err := s.API.GraphQL(threadDetailsQuery, variables, &resp); err != nil {
		return nil, err
	}
	if resp.Node == nil {
		return nil, fmt.Errorf("thread %s not found on %s", threadID, host)
	}
	return resp.Node, nil
}

type threadDetails struct {
	ID                 string `json:"id"`
	IsResolved         bool   `json:"isResolved"`
	ViewerCanResolve   bool   `json:"viewerCanResolve"`
	ViewerCanUnresolve bool   `json:"viewerCanUnresolve"`
}

func (s *Service) performResolve(threadID string) (ActionResult, error) {
	variables := map[string]interface{}{"threadId": threadID}
	var resp struct {
		Resolve struct {
			Thread struct {
				ID         string `json:"id"`
				IsResolved bool   `json:"isResolved"`
			} `json:"thread"`
		} `json:"resolveReviewThread"`
	}
	if err := s.API.GraphQL(resolveThreadMutation, variables, &resp); err != nil {
		return ActionResult{}, err
	}
	return ActionResult{ThreadID: resp.Resolve.Thread.ID, IsResolved: resp.Resolve.Thread.IsResolved, Changed: true}, nil
}

func (s *Service) performUnresolve(threadID string) (ActionResult, error) {
	variables := map[string]interface{}{"threadId": threadID}
	var resp struct {
		Unresolve struct {
			Thread struct {
				ID         string `json:"id"`
				IsResolved bool   `json:"isResolved"`
			} `json:"thread"`
		} `json:"unresolveReviewThread"`
	}
	if err := s.API.GraphQL(unresolveThreadMutation, variables, &resp); err != nil {
		return ActionResult{}, err
	}
	return ActionResult{ThreadID: resp.Unresolve.Thread.ID, IsResolved: resp.Unresolve.Thread.IsResolved, Changed: true}, nil
}

func isReviewThreadSchemaError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	if !strings.Contains(msg, "pullrequestreviewthread") {
		return false
	}
	return strings.Contains(msg, "does not exist") || strings.Contains(msg, "doesn't exist") || strings.Contains(msg, "cannot query field")
}

const listThreadsQuery = `
query Threads($id: ID!, $after: String) {
  node(id: $id) {
    ... on PullRequest {
      reviewThreads(first: 100, after: $after) {
        nodes {
          id
          isResolved
          isOutdated
          path
          line
          viewerCanResolve
          viewerCanUnresolve
          resolvedBy { login }
          comments(first: 100) {
            nodes {
              databaseId
              viewerDidAuthor
              updatedAt
            }
          }
        }
        pageInfo {
          hasNextPage
          endCursor
        }
      }
    }
  }
}
`

const commentThreadQuery = `
query CommentThread($id: ID!) {
  node(id: $id) {
    ... on PullRequestReviewComment {
      pullRequestReviewThread {
        id
      }
    }
  }
}
`

const threadDetailsQuery = `
query ThreadDetails($id: ID!) {
  node(id: $id) {
    ... on PullRequestReviewThread {
      id
      isResolved
      viewerCanResolve
      viewerCanUnresolve
    }
  }
}
`

const resolveThreadMutation = `
mutation ResolveThread($threadId: ID!) {
  resolveReviewThread(input: {threadId: $threadId}) {
    thread {
      id
      isResolved
    }
  }
}
`

const unresolveThreadMutation = `
mutation UnresolveThread($threadId: ID!) {
  unresolveReviewThread(input: {threadId: $threadId}) {
    thread {
      id
      isResolved
    }
  }
}
`
