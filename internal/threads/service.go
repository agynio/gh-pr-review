package threads

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/agynio/gh-pr-review/internal/ghcli"
	"github.com/agynio/gh-pr-review/internal/resolver"
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
	ThreadID string
}

// ActionResult captures the outcome of a resolve/unresolve mutation.
type ActionResult struct {
	ThreadNodeID string `json:"thread_node_id"`
	IsResolved   bool   `json:"is_resolved"`
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

	allThreads := make([]Thread, 0)

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
	allThreads := make([]threadNode, 0)
	var after *string

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
	threadID := strings.TrimSpace(opts.ThreadID)
	if threadID == "" {
		return ActionResult{}, errors.New("thread id is required")
	}

	thread, err := s.fetchThread(pr.Host, threadID)
	if err != nil {
		return ActionResult{}, err
	}

	desired := resolve
	if thread.IsResolved == desired {
		return ActionResult{ThreadNodeID: thread.ID, IsResolved: thread.IsResolved}, nil
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
	return ActionResult{ThreadNodeID: resp.Resolve.Thread.ID, IsResolved: resp.Resolve.Thread.IsResolved}, nil
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
	return ActionResult{ThreadNodeID: resp.Unresolve.Thread.ID, IsResolved: resp.Unresolve.Thread.IsResolved}, nil
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
