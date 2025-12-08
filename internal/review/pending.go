package review

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/agynio/gh-pr-review/internal/resolver"
)

// PendingOptions configures lookup of the latest pending review for a reviewer.
type PendingOptions struct {
	Reviewer string
	PerPage  int
}

// PendingSummary captures pending review metadata for output.
type PendingSummary struct {
	ID                string      `json:"id"`
	DatabaseID        int64       `json:"database_id"`
	State             string      `json:"state"`
	AuthorAssociation string      `json:"author_association,omitempty"`
	HTMLURL           string      `json:"html_url,omitempty"`
	User              *ReviewUser `json:"user,omitempty"`
}

// PendingSummaries retrieves pending reviews for the requested reviewer.

func (s *Service) PendingSummaries(pr resolver.Identity, opts PendingOptions) ([]PendingSummary, string, error) {
	reviewerFilter := strings.TrimSpace(opts.Reviewer)
	reviewer := reviewerFilter
	perPage := clampPerPage(opts.PerPage)

	useViewer := false
	if reviewer == "" {
		login, err := s.currentViewer()
		if err != nil {
			return nil, "", err
		}
		reviewer = login
		useViewer = true
	}

	query := `query PendingReviews($owner: String!, $name: String!, $number: Int!, $pageSize: Int!, $cursor: String) {
  repository(owner: $owner, name: $name) {
    pullRequest(number: $number) {
      reviews(states: [PENDING], first: $pageSize, after: $cursor) {
        nodes {
          id
          databaseId
          state
          authorAssociation
          url
          updatedAt
          createdAt
          author {
            login
            ... on User {
              databaseId
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
}`

	type pendingNode struct {
		ID                string `json:"id"`
		DatabaseID        *int64 `json:"databaseId"`
		State             string `json:"state"`
		AuthorAssociation string `json:"authorAssociation"`
		URL               string `json:"url"`
		UpdatedAt         string `json:"updatedAt"`
		CreatedAt         string `json:"createdAt"`
		Author            *struct {
			Login      string `json:"login"`
			DatabaseID *int64 `json:"databaseId"`
		} `json:"author"`
	}

	type timedSummary struct {
		summary PendingSummary
		when    time.Time
	}

	var (
		timedSummaries []timedSummary
		cursor         string
		hasCursor      bool
	)

	for {
		variables := map[string]interface{}{
			"owner":    pr.Owner,
			"name":     pr.Repo,
			"number":   pr.Number,
			"pageSize": perPage,
		}
		if hasCursor {
			variables["cursor"] = cursor
		}

		var response struct {
			Data struct {
				Repository *struct {
					PullRequest *struct {
						Reviews *struct {
							Nodes    []pendingNode `json:"nodes"`
							PageInfo struct {
								HasNextPage bool   `json:"hasNextPage"`
								EndCursor   string `json:"endCursor"`
							} `json:"pageInfo"`
						} `json:"reviews"`
					} `json:"pullRequest"`
				} `json:"repository"`
			} `json:"data"`
		}

		if err := s.API.GraphQL(query, variables, &response); err != nil {
			return nil, "", err
		}

		repo := response.Data.Repository
		if repo == nil || repo.PullRequest == nil || repo.PullRequest.Reviews == nil {
			return nil, reviewer, fmt.Errorf("pull request %s/%s#%d not found", pr.Owner, pr.Repo, pr.Number)
		}

		reviews := repo.PullRequest.Reviews
		filterLogin := reviewer
		for _, node := range reviews.Nodes {
			if filterLogin == "" {
				continue
			}

			var authorLogin string
			if node.Author != nil {
				authorLogin = strings.TrimSpace(node.Author.Login)
			}
			if authorLogin == "" && useViewer {
				authorLogin = reviewer
			}
			if authorLogin == "" || !strings.EqualFold(authorLogin, filterLogin) {
				continue
			}

			id := strings.TrimSpace(node.ID)
			if id == "" {
				return nil, reviewer, errors.New("pending review missing node identifier")
			}
			if node.DatabaseID == nil {
				return nil, reviewer, errors.New("pending review missing database identifier")
			}

			timestamp := strings.TrimSpace(node.UpdatedAt)
			if timestamp == "" {
				timestamp = strings.TrimSpace(node.CreatedAt)
			}
			if timestamp == "" {
				return nil, reviewer, errors.New("pending review missing timestamp")
			}
			when, err := time.Parse(time.RFC3339, timestamp)
			if err != nil {
				return nil, reviewer, fmt.Errorf("parse pending review timestamp: %w", err)
			}

			summary := PendingSummary{
				ID:                id,
				DatabaseID:        *node.DatabaseID,
				State:             strings.ToUpper(strings.TrimSpace(node.State)),
				AuthorAssociation: strings.TrimSpace(node.AuthorAssociation),
				HTMLURL:           strings.TrimSpace(node.URL),
			}
			userID := int64(0)
			if node.Author != nil && node.Author.DatabaseID != nil {
				userID = *node.Author.DatabaseID
			}
			if authorLogin != "" || userID != 0 {
				summary.User = &ReviewUser{Login: authorLogin, ID: userID}
			}

			timedSummaries = append(timedSummaries, timedSummary{summary: summary, when: when})
		}

		if !reviews.PageInfo.HasNextPage {
			break
		}

		nextCursor := strings.TrimSpace(reviews.PageInfo.EndCursor)
		if nextCursor == "" {
			return nil, reviewer, errors.New("pending review pagination cursor missing")
		}
		cursor = nextCursor
		hasCursor = true
	}

	if reviewer == "" {
		reviewer = reviewerFilter
	}

	if len(timedSummaries) == 0 {
		return nil, reviewer, fmt.Errorf("no pending reviews for %s", reviewer)
	}

	sort.Slice(timedSummaries, func(i, j int) bool {
		if timedSummaries[i].when.Equal(timedSummaries[j].when) {
			return timedSummaries[i].summary.DatabaseID < timedSummaries[j].summary.DatabaseID
		}
		return timedSummaries[i].when.Before(timedSummaries[j].when)
	})

	summaries := make([]PendingSummary, len(timedSummaries))
	for i, item := range timedSummaries {
		summaries[i] = item.summary
	}

	return summaries, reviewer, nil
}

// LatestPending locates the most recent pending review for the requested reviewer.
func (s *Service) LatestPending(pr resolver.Identity, opts PendingOptions) (*PendingSummary, error) {
	summaries, reviewer, err := s.PendingSummaries(pr, opts)
	if err != nil {
		return nil, err
	}
	if len(summaries) == 0 {
		return nil, fmt.Errorf("no pending reviews for %s", reviewer)
	}

	latest := summaries[len(summaries)-1]
	return &latest, nil
}
