package review

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Agyn-sandbox/gh-pr-review/internal/resolver"
)

// PendingOptions configures lookup of the latest pending review for a reviewer.
type PendingOptions struct {
	Reviewer string
	PerPage  int
	Page     int
}

// LatestPending locates the most recent pending review for the requested reviewer.
func (s *Service) LatestPending(pr resolver.Identity, opts PendingOptions) (*ReviewSummary, error) {
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
		latestPending restReview
		found         bool
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
			if !strings.EqualFold(strings.TrimSpace(review.State), "PENDING") {
				continue
			}
			if !found || review.ID > latestPending.ID {
				latestPending = review
				found = true
			}
		}

		if len(chunk) < perPage {
			break
		}
	}

	if !found {
		return nil, fmt.Errorf("no pending reviews for %s", reviewer)
	}

	summary := ReviewSummary{
		ID:                latestPending.ID,
		State:             strings.ToUpper(strings.TrimSpace(latestPending.State)),
		AuthorAssociation: strings.TrimSpace(latestPending.AuthorAssociation),
		HTMLURL:           strings.TrimSpace(latestPending.HTMLURL),
	}
	login := strings.TrimSpace(latestPending.User.Login)
	if login != "" || latestPending.User.ID != 0 {
		summary.User = &ReviewUser{Login: login, ID: latestPending.User.ID}
	}

	return &summary, nil
}
