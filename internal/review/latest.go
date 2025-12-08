package review

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/agynio/gh-pr-review/internal/resolver"
)

// LatestOptions configures lookup of the latest submitted review for a reviewer.
type LatestOptions struct {
	Reviewer string
	PerPage  int
	Page     int
}

// ReviewSummary captures a subset of review metadata returned to callers.
type ReviewSummary struct {
	ID                int64       `json:"id"`
	User              *ReviewUser `json:"user,omitempty"`
	SubmittedAt       *string     `json:"submitted_at,omitempty"`
	State             string      `json:"state,omitempty"`
	AuthorAssociation string      `json:"author_association,omitempty"`
	HTMLURL           string      `json:"html_url,omitempty"`
}

// ReviewUser mirrors the minimal REST user schema exposed in summaries.
type ReviewUser struct {
	Login string `json:"login,omitempty"`
	ID    int64  `json:"id,omitempty"`
}

// LatestSubmitted locates the most recent submitted review for the requested reviewer.
func (s *Service) LatestSubmitted(pr resolver.Identity, opts LatestOptions) (*ReviewSummary, error) {
	reviewer := strings.TrimSpace(opts.Reviewer)
	if reviewer == "" {
		login, err := s.currentLogin()
		if err != nil {
			return nil, fmt.Errorf("resolve authenticated user: %w", err)
		}
		reviewer = login
	}

	perPage := clampPerPage(opts.PerPage)
	page := opts.Page
	if page <= 0 {
		page = 1
	}

	var (
		latest        restReview
		hasSubmission bool
	)

	for current := page; ; current++ {
		var chunk []restReview
		params := map[string]string{
			"per_page": strconv.Itoa(perPage),
			"page":     strconv.Itoa(current),
		}
		path := fmt.Sprintf("repos/%s/%s/pulls/%d/reviews", pr.Owner, pr.Repo, pr.Number)
		if err := s.API.REST("GET", path, params, nil, &chunk); err != nil {
			return nil, err
		}

		if len(chunk) == 0 {
			break
		}

		for _, review := range chunk {
			if !strings.EqualFold(review.User.Login, reviewer) {
				continue
			}
			if review.SubmittedAt == nil {
				continue
			}
			if !hasSubmission || review.SubmittedAt.After(*latest.SubmittedAt) || (review.SubmittedAt.Equal(*latest.SubmittedAt) && review.ID > latest.ID) {
				latest = review
				hasSubmission = true
			}
		}

		if len(chunk) < perPage {
			break
		}
	}

	if !hasSubmission {
		return nil, fmt.Errorf("no submitted reviews for %s", reviewer)
	}

	result := ReviewSummary{
		ID:                latest.ID,
		State:             latest.State,
		AuthorAssociation: strings.TrimSpace(latest.AuthorAssociation),
		HTMLURL:           strings.TrimSpace(latest.HTMLURL),
	}
	if latest.SubmittedAt != nil {
		ts := latest.SubmittedAt.UTC().Format(time.RFC3339)
		result.SubmittedAt = &ts
	}
	login := strings.TrimSpace(latest.User.Login)
	if login != "" || latest.User.ID != 0 {
		result.User = &ReviewUser{Login: login, ID: latest.User.ID}
	}
	return &result, nil
}

type restReview struct {
	ID                int64      `json:"id"`
	NodeID            string     `json:"node_id"`
	State             string     `json:"state"`
	SubmittedAt       *time.Time `json:"submitted_at"`
	AuthorAssociation string     `json:"author_association"`
	HTMLURL           string     `json:"html_url"`
	User              struct {
		Login string `json:"login"`
		ID    int64  `json:"id"`
	} `json:"user"`
}

func clampPerPage(value int) int {
	switch {
	case value <= 0:
		return 100
	case value > 100:
		return 100
	default:
		return value
	}
}

func (s *Service) currentLogin() (string, error) {
	var user struct {
		Login string `json:"login"`
	}
	if err := s.API.REST("GET", "user", nil, nil, &user); err != nil {
		return "", err
	}
	login := strings.TrimSpace(user.Login)
	if login == "" {
		return "", errors.New("unable to determine authenticated user")
	}
	return login, nil
}
