package watch

import (
	"context"
	"fmt"
	"time"

	"github.com/agynio/gh-pr-review/internal/ghcli"
	"github.com/agynio/gh-pr-review/internal/resolver"
)

// Service provides PR comment watching operations.
type Service struct {
	API ghcli.API
}

// NewService constructs a watch Service.
func NewService(api ghcli.API) *Service {
	return &Service{API: api}
}

// WatchOptions configures the watch behavior.
type WatchOptions struct {
	Interval      time.Duration // polling interval (default 10s)
	Debounce      time.Duration // debounce duration (default 5s)
	Timeout       time.Duration // max watch duration (default 1h)
	IncludeIssue  bool          // include issue comments (not just review comments)
}

// Comment represents a PR comment (review or issue).
type Comment struct {
	ID          string  `json:"id"`
	NodeID      string  `json:"node_id,omitempty"`
	Body        string  `json:"body"`
	AuthorLogin string  `json:"author_login"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at,omitempty"`
	Path        *string `json:"path,omitempty"`
	Line        *int    `json:"line,omitempty"`
	Type        string  `json:"type"`
	ThreadID    *string `json:"thread_id,omitempty"`
	HTMLURL     string  `json:"html_url,omitempty"`
}

// WatchResult contains the watch outcome.
type WatchResult struct {
	Comments  []Comment `json:"comments"`
	TimedOut  bool      `json:"timed_out"`
	WatchedMs int64     `json:"watched_ms"`
}

// Watch polls for new comments on a PR and returns when new comments arrive (with debouncing).
func (s *Service) Watch(ctx context.Context, pr resolver.Identity, opts WatchOptions) (*WatchResult, error) {
	if opts.Interval <= 0 {
		opts.Interval = 10 * time.Second
	}
	if opts.Debounce <= 0 {
		opts.Debounce = 5 * time.Second
	}
	if opts.Timeout <= 0 {
		opts.Timeout = time.Hour
	}

	startTime := time.Now()
	timeoutCtx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()

	// Get initial state of comments
	knownIDs, err := s.fetchAllCommentIDs(pr, opts.IncludeIssue)
	if err != nil {
		return nil, fmt.Errorf("fetch initial comments: %w", err)
	}

	var (
		newComments     []Comment
		debounceTimer   *time.Timer
		debounceStarted bool
	)

	ticker := time.NewTicker(opts.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-timeoutCtx.Done():
			if debounceStarted && len(newComments) > 0 {
				return &WatchResult{
					Comments:  newComments,
					TimedOut:  true,
					WatchedMs: time.Since(startTime).Milliseconds(),
				}, nil
			}
			return &WatchResult{
				Comments:  nil,
				TimedOut:  true,
				WatchedMs: time.Since(startTime).Milliseconds(),
			}, nil

		case <-ticker.C:
			currentComments, err := s.fetchAllComments(pr, opts.IncludeIssue)
			if err != nil {
				continue // ignore transient errors, keep polling
			}

			for _, c := range currentComments {
				if _, seen := knownIDs[c.ID]; !seen {
					knownIDs[c.ID] = struct{}{}
					newComments = append(newComments, c)

					// Start or reset debounce timer
					if debounceTimer != nil {
						debounceTimer.Stop()
					}
					debounceTimer = time.NewTimer(opts.Debounce)
					debounceStarted = true
				}
			}

		case <-func() <-chan time.Time {
			if debounceTimer != nil {
				return debounceTimer.C
			}
			return nil
		}():
			// Debounce timer fired - return collected comments
			return &WatchResult{
				Comments:  newComments,
				TimedOut:  false,
				WatchedMs: time.Since(startTime).Milliseconds(),
			}, nil
		}
	}
}

func (s *Service) fetchAllCommentIDs(pr resolver.Identity, includeIssue bool) (map[string]struct{}, error) {
	comments, err := s.fetchAllComments(pr, includeIssue)
	if err != nil {
		return nil, err
	}

	ids := make(map[string]struct{}, len(comments))
	for _, c := range comments {
		ids[c.ID] = struct{}{}
	}
	return ids, nil
}

func (s *Service) fetchAllComments(pr resolver.Identity, includeIssue bool) ([]Comment, error) {
	var allComments []Comment

	// Fetch review comments via GraphQL
	reviewComments, err := s.fetchReviewComments(pr)
	if err != nil {
		return nil, fmt.Errorf("fetch review comments: %w", err)
	}
	allComments = append(allComments, reviewComments...)

	// Optionally fetch issue comments
	if includeIssue {
		issueComments, err := s.fetchIssueComments(pr)
		if err != nil {
			return nil, fmt.Errorf("fetch issue comments: %w", err)
		}
		allComments = append(allComments, issueComments...)
	}

	return allComments, nil
}

func (s *Service) fetchReviewComments(pr resolver.Identity) ([]Comment, error) {
	const query = `query ReviewComments($owner: String!, $name: String!, $number: Int!) {
  repository(owner: $owner, name: $name) {
    pullRequest(number: $number) {
      reviewThreads(first: 100) {
        nodes {
          id
          path
          line
          comments(first: 100) {
            nodes {
              id
              databaseId
              body
              createdAt
              updatedAt
              url
              author { login }
            }
          }
        }
      }
    }
  }
}`

	variables := map[string]interface{}{
		"owner":  pr.Owner,
		"name":   pr.Repo,
		"number": pr.Number,
	}

	var response struct {
		Repository *struct {
			PullRequest *struct {
				ReviewThreads struct {
					Nodes []struct {
						ID       string  `json:"id"`
						Path     string  `json:"path"`
						Line     *int    `json:"line"`
						Comments struct {
							Nodes []struct {
								ID         string `json:"id"`
								DatabaseID int    `json:"databaseId"`
								Body       string `json:"body"`
								CreatedAt  string `json:"createdAt"`
								UpdatedAt  string `json:"updatedAt"`
								URL        string `json:"url"`
								Author     *struct {
									Login string `json:"login"`
								} `json:"author"`
							} `json:"nodes"`
						} `json:"comments"`
					} `json:"nodes"`
				} `json:"reviewThreads"`
			} `json:"pullRequest"`
		} `json:"repository"`
	}

	if err := s.API.GraphQL(query, variables, &response); err != nil {
		return nil, err
	}

	if response.Repository == nil || response.Repository.PullRequest == nil {
		return nil, fmt.Errorf("pull request %d not found on %s/%s", pr.Number, pr.Owner, pr.Repo)
	}

	var comments []Comment
	for _, thread := range response.Repository.PullRequest.ReviewThreads.Nodes {
		threadID := thread.ID
		path := thread.Path
		line := thread.Line

		for _, c := range thread.Comments.Nodes {
			authorLogin := ""
			if c.Author != nil {
				authorLogin = c.Author.Login
			}

			comments = append(comments, Comment{
				ID:          fmt.Sprintf("review-%d", c.DatabaseID),
				NodeID:      c.ID,
				Body:        c.Body,
				AuthorLogin: authorLogin,
				CreatedAt:   c.CreatedAt,
				UpdatedAt:   c.UpdatedAt,
				Path:        &path,
				Line:        line,
				Type:        "review",
				ThreadID:    &threadID,
				HTMLURL:     c.URL,
			})
		}
	}

	return comments, nil
}

func (s *Service) fetchIssueComments(pr resolver.Identity) ([]Comment, error) {
	const query = `query IssueComments($owner: String!, $name: String!, $number: Int!) {
  repository(owner: $owner, name: $name) {
    pullRequest(number: $number) {
      comments(first: 100) {
        nodes {
          id
          databaseId
          body
          createdAt
          updatedAt
          url
          author { login }
        }
      }
    }
  }
}`

	variables := map[string]interface{}{
		"owner":  pr.Owner,
		"name":   pr.Repo,
		"number": pr.Number,
	}

	var response struct {
		Repository *struct {
			PullRequest *struct {
				Comments struct {
					Nodes []struct {
						ID         string `json:"id"`
						DatabaseID int    `json:"databaseId"`
						Body       string `json:"body"`
						CreatedAt  string `json:"createdAt"`
						UpdatedAt  string `json:"updatedAt"`
						URL        string `json:"url"`
						Author     *struct {
							Login string `json:"login"`
						} `json:"author"`
					} `json:"nodes"`
				} `json:"comments"`
			} `json:"pullRequest"`
		} `json:"repository"`
	}

	if err := s.API.GraphQL(query, variables, &response); err != nil {
		return nil, err
	}

	if response.Repository == nil || response.Repository.PullRequest == nil {
		return nil, fmt.Errorf("pull request %d not found on %s/%s", pr.Number, pr.Owner, pr.Repo)
	}

	var comments []Comment
	for _, c := range response.Repository.PullRequest.Comments.Nodes {
		authorLogin := ""
		if c.Author != nil {
			authorLogin = c.Author.Login
		}

		comments = append(comments, Comment{
			ID:          fmt.Sprintf("issue-%d", c.DatabaseID),
			NodeID:      c.ID,
			Body:        c.Body,
			AuthorLogin: authorLogin,
			CreatedAt:   c.CreatedAt,
			UpdatedAt:   c.UpdatedAt,
			Type:        "issue",
			HTMLURL:     c.URL,
		})
	}

	return comments, nil
}
