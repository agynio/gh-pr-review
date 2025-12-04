package comments

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Agyn-sandbox/gh-pr-review/internal/ghcli"
	"github.com/Agyn-sandbox/gh-pr-review/internal/resolver"
)

const autoSubmitSummary = "Auto-submitting pending review to unblock threaded reply via gh-pr-review."

// Service provides high-level review comment operations.
type Service struct {
	API ghcli.API
}

// ListOptions defines filters for listing review comments.
type ListOptions struct {
	ReviewID int64
	Latest   bool
	Reviewer string
}

// ReplyOptions contains the payload for replying to a review comment.
type ReplyOptions struct {
	CommentID int64
	Body      string
}

// NewService constructs a Service using the provided API client.
func NewService(api ghcli.API) *Service {
	return &Service{API: api}
}

// List retrieves review comments for the requested review and returns raw JSON messages for serialization.
func (s *Service) List(pr resolver.Identity, opts ListOptions) ([]json.RawMessage, error) {
	reviewID, err := s.resolveReviewID(pr, opts)
	if err != nil {
		return nil, err
	}

	all := make([]json.RawMessage, 0)
	page := 1
	for {
		var chunk []json.RawMessage
		params := map[string]string{
			"per_page": "100",
			"page":     strconv.Itoa(page),
		}
		path := fmt.Sprintf("repos/%s/%s/pulls/%d/reviews/%d/comments", pr.Owner, pr.Repo, pr.Number, reviewID)
		if err := s.API.REST("GET", path, params, nil, &chunk); err != nil {
			return nil, err
		}

		if len(chunk) == 0 {
			break
		}

		all = append(all, chunk...)
		if len(chunk) < 100 {
			break
		}
		page++
	}

	return all, nil
}

// Reply posts a reply to an existing review comment, automatically submitting any pending reviews owned by the user when necessary.
func (s *Service) Reply(pr resolver.Identity, opts ReplyOptions) (json.RawMessage, error) {
	if opts.CommentID <= 0 {
		return nil, errors.New("invalid comment id")
	}
	if strings.TrimSpace(opts.Body) == "" {
		return nil, errors.New("reply body is required")
	}

	payload := map[string]interface{}{
		"body": opts.Body,
	}
	path := fmt.Sprintf("repos/%s/%s/pulls/%d/comments/%d/replies", pr.Owner, pr.Repo, pr.Number, opts.CommentID)

	var reply json.RawMessage
	err := s.API.REST("POST", path, nil, payload, &reply)
	if err == nil {
		return reply, nil
	}

	apiErr := &ghcli.APIError{}
	if !errors.As(err, &apiErr) || apiErr.StatusCode != 422 || !apiErr.ContainsLower("pending review") {
		return nil, err
	}

	if err := s.autoSubmitPendingReviews(pr); err != nil {
		return nil, fmt.Errorf("failed to submit pending review: %w", err)
	}

	if err := s.API.REST("POST", path, nil, payload, &reply); err != nil {
		return nil, err
	}

	return reply, nil
}

func (s *Service) resolveReviewID(pr resolver.Identity, opts ListOptions) (int64, error) {
	if opts.ReviewID > 0 {
		return opts.ReviewID, nil
	}
	if !opts.Latest {
		return 0, errors.New("either --review-id or --latest must be provided")
	}

	reviewer := strings.TrimSpace(opts.Reviewer)
	if reviewer == "" {
		login, err := s.currentLogin()
		if err != nil {
			return 0, fmt.Errorf("resolve authenticated user: %w", err)
		}
		reviewer = login
	}

	reviewID, err := s.latestSubmittedReview(pr, reviewer)
	if err != nil {
		return 0, err
	}
	return reviewID, nil
}

func (s *Service) currentLogin() (string, error) {
	var user struct {
		Login string `json:"login"`
	}
	if err := s.API.REST("GET", "user", nil, nil, &user); err != nil {
		return "", err
	}
	if user.Login == "" {
		return "", errors.New("unable to determine authenticated user")
	}
	return user.Login, nil
}

func (s *Service) latestSubmittedReview(pr resolver.Identity, reviewer string) (int64, error) {
	var (
		latestID   int64
		latestTime time.Time
	)

	page := 1
	for {
		var reviews []struct {
			ID          int64      `json:"id"`
			State       string     `json:"state"`
			SubmittedAt *time.Time `json:"submitted_at"`
			User        struct {
				Login string `json:"login"`
			} `json:"user"`
		}

		params := map[string]string{
			"per_page": "100",
			"page":     strconv.Itoa(page),
		}
		path := fmt.Sprintf("repos/%s/%s/pulls/%d/reviews", pr.Owner, pr.Repo, pr.Number)
		if err := s.API.REST("GET", path, params, nil, &reviews); err != nil {
			return 0, err
		}

		if len(reviews) == 0 {
			break
		}

		for _, review := range reviews {
			if !strings.EqualFold(review.User.Login, reviewer) {
				continue
			}
			if review.SubmittedAt == nil {
				continue
			}
			if review.SubmittedAt.After(latestTime) {
				latestTime = *review.SubmittedAt
				latestID = review.ID
			}
		}

		if len(reviews) < 100 {
			break
		}
		page++
	}

	if latestID == 0 {
		return 0, fmt.Errorf("no submitted reviews for %s", reviewer)
	}
	return latestID, nil
}

func (s *Service) autoSubmitPendingReviews(pr resolver.Identity) error {
	login, err := s.currentLogin()
	if err != nil {
		return err
	}

	pending, err := s.pendingReviews(pr, login)
	if err != nil {
		return err
	}
	if len(pending) == 0 {
		return fmt.Errorf("no pending reviews owned by %s found on pull request #%d", login, pr.Number)
	}

	for _, reviewID := range pending {
		path := fmt.Sprintf("repos/%s/%s/pulls/%d/reviews/%d/events", pr.Owner, pr.Repo, pr.Number, reviewID)
		payload := map[string]interface{}{
			"event": "COMMENT",
			"body":  autoSubmitSummary,
		}
		if err := s.API.REST("POST", path, nil, payload, nil); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) pendingReviews(pr resolver.Identity, reviewer string) ([]int64, error) {
	var ids []int64
	page := 1
	for {
		var reviews []struct {
			ID    int64  `json:"id"`
			State string `json:"state"`
			User  struct {
				Login string `json:"login"`
			} `json:"user"`
		}

		params := map[string]string{
			"per_page": "100",
			"page":     strconv.Itoa(page),
		}
		path := fmt.Sprintf("repos/%s/%s/pulls/%d/reviews", pr.Owner, pr.Repo, pr.Number)
		if err := s.API.REST("GET", path, params, nil, &reviews); err != nil {
			return nil, err
		}

		if len(reviews) == 0 {
			break
		}

		for _, review := range reviews {
			if !strings.EqualFold(review.User.Login, reviewer) {
				continue
			}
			if strings.EqualFold(review.State, "PENDING") {
				ids = append(ids, review.ID)
			}
		}

		if len(reviews) < 100 {
			break
		}
		page++
	}

	return ids, nil
}
