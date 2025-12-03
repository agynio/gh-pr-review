package comments

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/Agyn-sandbox/gh-pr-review/internal/resolver"
)

// IDsOptions configures comment identifier listings.
type IDsOptions struct {
	ReviewID int64
	Latest   bool
	Reviewer string
	Limit    int
	PerPage  int
	Page     int
}

// CommentReference exposes a minimal comment payload with identifiers and metadata.
type CommentReference struct {
	ID                int64          `json:"id"`
	Body              string         `json:"body"`
	User              *CommentAuthor `json:"user,omitempty"`
	AuthorAssociation string         `json:"author_association,omitempty"`
	CreatedAt         string         `json:"created_at,omitempty"`
	UpdatedAt         string         `json:"updated_at,omitempty"`
	HTMLURL           string         `json:"html_url,omitempty"`
	Path              string         `json:"path,omitempty"`
	Line              *int           `json:"line,omitempty"`
}

// CommentAuthor represents the user who authored a comment.
type CommentAuthor struct {
	Login string `json:"login,omitempty"`
	ID    int64  `json:"id,omitempty"`
}

// IDs returns comment identifiers (and selected metadata) for the requested review.
func (s *Service) IDs(pr resolver.Identity, opts IDsOptions) ([]CommentReference, error) {
	if opts.Limit < 0 {
		return nil, errors.New("limit must be non-negative")
	}

	reviewID, err := s.resolveReviewID(pr, ListOptions{ReviewID: opts.ReviewID, Latest: opts.Latest, Reviewer: opts.Reviewer})
	if err != nil {
		return nil, err
	}

	perPage := clampIDsPerPage(opts.PerPage)
	page := opts.Page
	if page <= 0 {
		page = 1
	}

	limit := opts.Limit
	var results []CommentReference

	for current := page; ; current++ {
		var chunk []restComment
		params := map[string]string{
			"per_page": strconv.Itoa(perPage),
			"page":     strconv.Itoa(current),
		}
		path := fmt.Sprintf("repos/%s/%s/pulls/%d/reviews/%d/comments", pr.Owner, pr.Repo, pr.Number, reviewID)
		if err := s.API.REST("GET", path, params, nil, &chunk); err != nil {
			return nil, err
		}

		if len(chunk) == 0 {
			break
		}

		for _, comment := range chunk {
			entry := CommentReference{ID: comment.ID, Body: comment.Body}

			login := strings.TrimSpace(comment.User.Login)
			if login != "" || comment.User.ID != 0 {
				entry.User = &CommentAuthor{Login: login, ID: comment.User.ID}
			}
			if assoc := strings.TrimSpace(comment.AuthorAssociation); assoc != "" {
				entry.AuthorAssociation = assoc
			}
			if created := strings.TrimSpace(comment.CreatedAt); created != "" {
				entry.CreatedAt = created
			}
			if updated := strings.TrimSpace(comment.UpdatedAt); updated != "" {
				entry.UpdatedAt = updated
			}
			if url := strings.TrimSpace(comment.HTMLURL); url != "" {
				entry.HTMLURL = url
			}
			if path := strings.TrimSpace(comment.Path); path != "" {
				entry.Path = path
			}
			if comment.Line != nil {
				line := *comment.Line
				entry.Line = &line
			}

			results = append(results, entry)
			if limit > 0 && len(results) >= limit {
				return results[:limit], nil
			}
		}

		if len(chunk) < perPage {
			break
		}
	}

	return results, nil
}

type restComment struct {
	ID                int64  `json:"id"`
	Body              string `json:"body"`
	AuthorAssociation string `json:"author_association"`
	CreatedAt         string `json:"created_at"`
	UpdatedAt         string `json:"updated_at"`
	HTMLURL           string `json:"html_url"`
	Path              string `json:"path"`
	Line              *int   `json:"line"`
	User              struct {
		Login string `json:"login"`
		ID    int64  `json:"id"`
	} `json:"user"`
}

func clampIDsPerPage(value int) int {
	switch {
	case value <= 0:
		return 100
	case value > 100:
		return 100
	default:
		return value
	}
}
