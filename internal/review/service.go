package review

import (
	"errors"
	"fmt"
	"strings"

	"github.com/Agyn-sandbox/gh-pr-review/internal/ghcli"
	"github.com/Agyn-sandbox/gh-pr-review/internal/resolver"
)

// Service coordinates review GraphQL operations through the gh CLI.
type Service struct {
	API ghcli.API
}

// ReviewState contains metadata about a review after opening or submitting it.
type ReviewState struct {
	ID          string  `json:"id"`
	State       string  `json:"state"`
	SubmittedAt *string `json:"submitted_at"`
	DatabaseID  *int64  `json:"database_id,omitempty"`
	HTMLURL     string  `json:"html_url,omitempty"`
}

// ReviewThread represents an inline comment thread added to a pending review.
type ReviewThread struct {
	ID         string `json:"id"`
	Path       string `json:"path"`
	IsOutdated bool   `json:"is_outdated"`
}

// ThreadInput describes the inline comment details for AddThread.
type ThreadInput struct {
	ReviewID  string
	Path      string
	Line      int
	Side      string
	StartLine *int
	StartSide *string
	Body      string
}

// SubmitInput contains the payload for submitting a pending review.
type SubmitInput struct {
	ReviewID string
	Event    string
	Body     string
}

// NewService constructs a review Service.
func NewService(api ghcli.API) *Service {
	return &Service{API: api}
}

// Start opens a pending review for the specified pull request.
func (s *Service) Start(pr resolver.Identity, commitOID string) (*ReviewState, error) {
	nodeID, headSHA, err := s.pullRequestIdentifiers(pr)
	if err != nil {
		return nil, err
	}
	if commitOID == "" {
		commitOID = headSHA
	}

	query := `mutation AddPullRequestReview($input: AddPullRequestReviewInput!) {
  addPullRequestReview(input: $input) {
    pullRequestReview { id state submittedAt databaseId url }
  }
}`

	payload := map[string]interface{}{
		"input": map[string]interface{}{
			"pullRequestId": nodeID,
			"commitOID":     commitOID,
		},
	}

	var response struct {
		Data struct {
			AddPullRequestReview struct {
				PullRequestReview struct {
					ID          string  `json:"id"`
					State       string  `json:"state"`
					SubmittedAt *string `json:"submittedAt"`
					DatabaseID  *int64  `json:"databaseId"`
					URL         string  `json:"url"`
				} `json:"pullRequestReview"`
			} `json:"addPullRequestReview"`
		} `json:"data"`
	}

	if err := s.API.GraphQL(query, payload, &response); err != nil {
		return nil, err
	}

	review := response.Data.AddPullRequestReview.PullRequestReview
	state := ReviewState{
		ID:          review.ID,
		State:       review.State,
		SubmittedAt: review.SubmittedAt,
		DatabaseID:  review.DatabaseID,
		HTMLURL:     strings.TrimSpace(review.URL),
	}
	return &state, nil
}

// AddThread adds an inline review comment thread to an existing pending review.
func (s *Service) AddThread(pr resolver.Identity, input ThreadInput) (*ReviewThread, error) {
	if input.ReviewID == "" {
		return nil, errors.New("review id is required")
	}
	if input.Path == "" {
		return nil, errors.New("path is required")
	}
	if input.Line <= 0 {
		return nil, errors.New("line must be positive")
	}
	if input.Body == "" {
		return nil, errors.New("body is required")
	}

	query := `mutation AddPullRequestReviewThread($input: AddPullRequestReviewThreadInput!) {
  addPullRequestReviewThread(input: $input) {
    thread { id path isOutdated }
  }
}`

	graphqlInput := map[string]interface{}{
		"pullRequestReviewId": input.ReviewID,
		"path":                input.Path,
		"line":                input.Line,
		"side":                input.Side,
		"body":                input.Body,
	}
	if input.StartLine != nil {
		graphqlInput["startLine"] = *input.StartLine
	}
	if input.StartSide != nil {
		graphqlInput["startSide"] = *input.StartSide
	}

	payload := map[string]interface{}{
		"input": graphqlInput,
	}

	var response struct {
		Data struct {
			AddPullRequestReviewThread struct {
				Thread struct {
					ID         string `json:"id"`
					Path       string `json:"path"`
					IsOutdated bool   `json:"isOutdated"`
				} `json:"thread"`
			} `json:"addPullRequestReviewThread"`
		} `json:"data"`
	}

	if err := s.API.GraphQL(query, payload, &response); err != nil {
		return nil, err
	}

	thread := response.Data.AddPullRequestReviewThread.Thread
	return &ReviewThread{ID: thread.ID, Path: thread.Path, IsOutdated: thread.IsOutdated}, nil
}

// Submit finalizes a pending review with the given event and optional body.
func (s *Service) Submit(pr resolver.Identity, input SubmitInput) (*ReviewState, error) {
	if input.ReviewID == "" {
		return nil, errors.New("review id is required")
	}

	query := `mutation SubmitPullRequestReview($input: SubmitPullRequestReviewInput!) {
  submitPullRequestReview(input: $input) {
    pullRequestReview { id state submittedAt databaseId url }
  }
}`

	graphqlInput := map[string]interface{}{
		"pullRequestReviewId": input.ReviewID,
		"event":               input.Event,
	}
	if strings.TrimSpace(input.Body) != "" {
		graphqlInput["body"] = input.Body
	}

	payload := map[string]interface{}{
		"input": graphqlInput,
	}

	var response struct {
		Data struct {
			SubmitPullRequestReview struct {
				PullRequestReview struct {
					ID          string  `json:"id"`
					State       string  `json:"state"`
					SubmittedAt *string `json:"submittedAt"`
					DatabaseID  *int64  `json:"databaseId"`
					URL         string  `json:"url"`
				} `json:"pullRequestReview"`
			} `json:"submitPullRequestReview"`
		} `json:"data"`
	}

	if err := s.API.GraphQL(query, payload, &response); err != nil {
		return nil, err
	}

	review := response.Data.SubmitPullRequestReview.PullRequestReview
	state := ReviewState{
		ID:          review.ID,
		State:       review.State,
		SubmittedAt: review.SubmittedAt,
		DatabaseID:  review.DatabaseID,
		HTMLURL:     strings.TrimSpace(review.URL),
	}
	return &state, nil
}

func (s *Service) pullRequestIdentifiers(pr resolver.Identity) (string, string, error) {
	path := fmt.Sprintf("repos/%s/%s/pulls/%d", pr.Owner, pr.Repo, pr.Number)
	var data struct {
		NodeID string `json:"node_id"`
		Head   struct {
			SHA string `json:"sha"`
		} `json:"head"`
	}
	if err := s.API.REST("GET", path, nil, nil, &data); err != nil {
		return "", "", err
	}
	if data.NodeID == "" || data.Head.SHA == "" {
		return "", "", errors.New("pull request metadata incomplete")
	}
	return data.NodeID, data.Head.SHA, nil
}
