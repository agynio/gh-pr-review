package draft

import (
	"fmt"

	"github.com/agynio/gh-pr-review/internal/ghcli"
	"github.com/agynio/gh-pr-review/internal/resolver"
)

// Service exposes pull request draft operations.
type Service struct {
	API ghcli.API
}

// NewService constructs a Service with the provided API client.
func NewService(api ghcli.API) *Service {
	return &Service{API: api}
}

// ActionOptions controls draft/ready operations.
type ActionOptions struct {
	PRNumber int
}

// ActionResult captures the outcome of a draft/ready mutation.
type ActionResult struct {
	PRNumber   int    `json:"pr_number"`
	IsDraft    bool   `json:"is_draft"`
	Status     string `json:"status"`
}

// DraftInfo contains information about a PR's draft status.
type DraftInfo struct {
	PRNumber int    `json:"pr_number"`
	IsDraft  bool   `json:"is_draft"`
	Title    string `json:"title"`
}

// Draft marks a pull request as draft when permissions allow it.
func (s *Service) Draft(pr resolver.Identity, opts ActionOptions) (ActionResult, error) {
	return s.changeDraftState(pr, opts, true)
}

// Ready marks a pull request as ready for review when permissions allow it.
func (s *Service) Ready(pr resolver.Identity, opts ActionOptions) (ActionResult, error) {
	return s.changeDraftState(pr, opts, false)
}

// Status returns the current draft status of a pull request.
func (s *Service) Status(pr resolver.Identity, opts ActionOptions) (DraftInfo, error) {
	prNumber := opts.PRNumber
	if prNumber == 0 {
		prNumber = pr.Number
	}

	variables := map[string]interface{}{
		"owner":  pr.Owner,
		"repo":   pr.Repo,
		"number": prNumber,
	}

	var resp struct {
		Repository struct {
			PullRequest *struct {
				Number int    `json:"number"`
				Title  string `json:"title"`
				IsDraft bool  `json:"isDraft"`
			} `json:"pullRequest"`
		} `json:"repository"`
	}

	if err := s.API.GraphQL(pullRequestStatusQuery, variables, &resp); err != nil {
		return DraftInfo{}, err
	}

	if resp.Repository.PullRequest == nil {
		return DraftInfo{}, fmt.Errorf("pull request %d not found in %s/%s", prNumber, pr.Owner, pr.Repo)
	}

	return DraftInfo{
		PRNumber: resp.Repository.PullRequest.Number,
		IsDraft:  resp.Repository.PullRequest.IsDraft,
		Title:    resp.Repository.PullRequest.Title,
	}, nil
}

// List returns all draft pull requests in the repository.
func (s *Service) List(pr resolver.Identity) ([]DraftInfo, error) {
	variables := map[string]interface{}{
		"owner": pr.Owner,
		"repo":  pr.Repo,
	}

	var resp struct {
		Repository struct {
			PullRequests struct {
				Nodes []struct {
					Number int    `json:"number"`
					Title  string `json:"title"`
					IsDraft bool  `json:"isDraft"`
				} `json:"nodes"`
			} `json:"pullRequests"`
		} `json:"repository"`
	}

	if err := s.API.GraphQL(draftListQuery, variables, &resp); err != nil {
		return nil, err
	}

	drafts := make([]DraftInfo, 0, len(resp.Repository.PullRequests.Nodes))
	for _, pr := range resp.Repository.PullRequests.Nodes {
		if pr.IsDraft {
			drafts = append(drafts, DraftInfo{
				PRNumber: pr.Number,
				IsDraft:  pr.IsDraft,
				Title:    pr.Title,
			})
		}
	}

	return drafts, nil
}

func (s *Service) changeDraftState(pr resolver.Identity, opts ActionOptions, makeDraft bool) (ActionResult, error) {
	prNumber := opts.PRNumber
	if prNumber == 0 {
		prNumber = pr.Number
	}

	// First check current status
	current, err := s.Status(pr, ActionOptions{PRNumber: prNumber})
	if err != nil {
		return ActionResult{}, err
	}

	// If already in desired state, return early
	if current.IsDraft == makeDraft {
		action := "ready for review"
		if makeDraft {
			action = "draft"
		}
		return ActionResult{
			PRNumber: current.PRNumber,
			IsDraft:  current.IsDraft,
			Status:   fmt.Sprintf("already %s", action),
		}, nil
	}

	// Get the pull request node ID for the mutation
	nodeID, err := s.getPullRequestNodeID(pr, prNumber)
	if err != nil {
		return ActionResult{}, err
	}

	// Perform the appropriate mutation
	if makeDraft {
		return s.convertToDraft(nodeID)
	}
	return s.markReadyForReview(nodeID)
}

func (s *Service) getPullRequestNodeID(pr resolver.Identity, prNumber int) (string, error) {
	variables := map[string]interface{}{
		"owner":  pr.Owner,
		"repo":   pr.Repo,
		"number": prNumber,
	}

	var resp struct {
		Repository struct {
			PullRequest *struct {
				ID string `json:"id"`
			} `json:"pullRequest"`
		} `json:"repository"`
	}

	if err := s.API.GraphQL(pullRequestNodeIDQuery, variables, &resp); err != nil {
		return "", err
	}

	if resp.Repository.PullRequest == nil {
		return "", fmt.Errorf("pull request %d not found in %s/%s", prNumber, pr.Owner, pr.Repo)
	}

	return resp.Repository.PullRequest.ID, nil
}

func (s *Service) convertToDraft(nodeID string) (ActionResult, error) {
	variables := map[string]interface{}{"pullRequestId": nodeID}
	var resp struct {
		ConvertPullRequestToDraft struct {
			PullRequest struct {
				Number  int    `json:"number"`
				IsDraft bool   `json:"isDraft"`
			} `json:"pullRequest"`
		} `json:"convertPullRequestToDraft"`
	}

	if err := s.API.GraphQL(convertToDraftMutation, variables, &resp); err != nil {
		return ActionResult{}, err
	}

	return ActionResult{
		PRNumber: resp.ConvertPullRequestToDraft.PullRequest.Number,
		IsDraft:  resp.ConvertPullRequestToDraft.PullRequest.IsDraft,
		Status:   "marked as draft",
	}, nil
}

func (s *Service) markReadyForReview(nodeID string) (ActionResult, error) {
	variables := map[string]interface{}{"pullRequestId": nodeID}
	var resp struct {
		MarkPullRequestReadyForReview struct {
			PullRequest struct {
				Number  int    `json:"number"`
				IsDraft bool   `json:"isDraft"`
			} `json:"pullRequest"`
		} `json:"markPullRequestReadyForReview"`
	}

	if err := s.API.GraphQL(markReadyForReviewMutation, variables, &resp); err != nil {
		return ActionResult{}, err
	}

	return ActionResult{
		PRNumber: resp.MarkPullRequestReadyForReview.PullRequest.Number,
		IsDraft:  resp.MarkPullRequestReadyForReview.PullRequest.IsDraft,
		Status:   "marked as ready for review",
	}, nil
}

const pullRequestStatusQuery = `
query PullRequestStatus($owner: String!, $repo: String!, $number: Int!) {
  repository(owner: $owner, name: $repo) {
    pullRequest(number: $number) {
      number
      title
      isDraft
    }
  }
}
`

const draftListQuery = `
query DraftList($owner: String!, $repo: String!) {
  repository(owner: $owner, name: $repo) {
    pullRequests(states: [OPEN], first: 100, orderBy: {field: CREATED_AT, direction: DESC}) {
      nodes {
        number
        title
        isDraft
      }
    }
  }
}
`

const pullRequestNodeIDQuery = `
query PullRequestNodeID($owner: String!, $repo: String!, $number: Int!) {
  repository(owner: $owner, name: $repo) {
    pullRequest(number: $number) {
      id
    }
  }
}
`

const convertToDraftMutation = `
mutation ConvertToDraft($pullRequestId: ID!) {
  convertPullRequestToDraft(input: {pullRequestId: $pullRequestId}) {
    pullRequest {
      number
      isDraft
    }
  }
}
`

const markReadyForReviewMutation = `
mutation MarkReadyForReview($pullRequestId: ID!) {
  markPullRequestReadyForReview(input: {pullRequestId: $pullRequestId}) {
    pullRequest {
      number
      isDraft
    }
  }
}
`
