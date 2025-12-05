package comments

import (
	"errors"
	"strings"

	"github.com/Agyn-sandbox/gh-pr-review/internal/ghcli"
	"github.com/Agyn-sandbox/gh-pr-review/internal/resolver"
)

const addThreadReplyMutation = `mutation AddPullRequestReviewThreadReply($input: AddPullRequestReviewThreadReplyInput!) {
  addPullRequestReviewThreadReply(input: $input) {
    comment {
      id
      databaseId
      body
      diffHunk
      path
      url
      createdAt
      updatedAt
      author { login }
      pullRequestReview { id databaseId state }
      replyTo { id }
    }
    thread {
      id
      isResolved
      isOutdated
    }
  }
}`

// Service provides high-level review comment operations.
type Service struct {
	API ghcli.API
}

// ReplyOptions contains the payload for replying to a review comment thread.
type ReplyOptions struct {
	ThreadID string
	ReviewID string
	Body     string
}

// Reply represents the normalized GraphQL response after adding a thread reply.
type Reply struct {
	ID               string  `json:"id"`
	DatabaseID       *int    `json:"database_id,omitempty"`
	ReviewID         *string `json:"review_id,omitempty"`
	ReviewDatabaseID *int    `json:"review_database_id,omitempty"`
	ReviewState      *string `json:"review_state,omitempty"`
	ThreadID         string  `json:"thread_id"`
	ThreadIsResolved bool    `json:"thread_is_resolved"`
	ThreadIsOutdated bool    `json:"thread_is_outdated"`
	ReplyToCommentID *string `json:"reply_to_comment_id,omitempty"`
	Body             string  `json:"body"`
	DiffHunk         *string `json:"diff_hunk,omitempty"`
	Path             string  `json:"path"`
	HtmlURL          string  `json:"html_url"`
	AuthorLogin      string  `json:"author_login"`
	CreatedAt        string  `json:"created_at"`
	UpdatedAt        string  `json:"updated_at"`
}

// NewService constructs a Service using the provided API client.
func NewService(api ghcli.API) *Service {
	return &Service{API: api}
}

// Reply posts a reply to an existing review thread using the GraphQL API.
func (s *Service) Reply(_ resolver.Identity, opts ReplyOptions) (Reply, error) {
	threadID := strings.TrimSpace(opts.ThreadID)
	if threadID == "" {
		return Reply{}, errors.New("thread id is required")
	}
	if strings.TrimSpace(opts.Body) == "" {
		return Reply{}, errors.New("reply body is required")
	}

	input := map[string]interface{}{
		"pullRequestReviewThreadId": threadID,
		"body":                      opts.Body,
	}
	if reviewID := strings.TrimSpace(opts.ReviewID); reviewID != "" {
		input["pullRequestReviewId"] = reviewID
	}

	variables := map[string]interface{}{"input": input}

	var response struct {
		AddPullRequestReviewThreadReply struct {
			Comment *struct {
				ID         string  `json:"id"`
				DatabaseID *int    `json:"databaseId"`
				Body       string  `json:"body"`
				DiffHunk   *string `json:"diffHunk"`
				Path       string  `json:"path"`
				URL        string  `json:"url"`
				CreatedAt  string  `json:"createdAt"`
				UpdatedAt  string  `json:"updatedAt"`
				Author     *struct {
					Login string `json:"login"`
				} `json:"author"`
				PullRequestReview *struct {
					ID         string `json:"id"`
					DatabaseID *int   `json:"databaseId"`
					State      string `json:"state"`
				} `json:"pullRequestReview"`
				ReplyTo *struct {
					ID string `json:"id"`
				} `json:"replyTo"`
			} `json:"comment"`
			Thread *struct {
				ID         string `json:"id"`
				IsResolved bool   `json:"isResolved"`
				IsOutdated bool   `json:"isOutdated"`
			} `json:"thread"`
		} `json:"addPullRequestReviewThreadReply"`
	}

	if err := s.API.GraphQL(addThreadReplyMutation, variables, &response); err != nil {
		return Reply{}, err
	}

	comment := response.AddPullRequestReviewThreadReply.Comment
	if comment == nil {
		return Reply{}, errors.New("mutation response missing comment")
	}
	if strings.TrimSpace(comment.ID) == "" {
		return Reply{}, errors.New("mutation response missing comment id")
	}
	if comment.Author == nil || strings.TrimSpace(comment.Author.Login) == "" {
		return Reply{}, errors.New("mutation response missing author login")
	}
	thread := response.AddPullRequestReviewThreadReply.Thread
	if thread == nil || strings.TrimSpace(thread.ID) == "" {
		return Reply{}, errors.New("mutation response missing thread id")
	}

	reply := Reply{
		ID:               comment.ID,
		ThreadID:         thread.ID,
		ThreadIsResolved: thread.IsResolved,
		ThreadIsOutdated: thread.IsOutdated,
		Body:             comment.Body,
		Path:             comment.Path,
		HtmlURL:          comment.URL,
		AuthorLogin:      comment.Author.Login,
		CreatedAt:        comment.CreatedAt,
		UpdatedAt:        comment.UpdatedAt,
	}

	if comment.DatabaseID != nil {
		reply.DatabaseID = comment.DatabaseID
	}
	if comment.DiffHunk != nil {
		trimmed := strings.TrimSpace(*comment.DiffHunk)
		if trimmed != "" {
			value := *comment.DiffHunk
			reply.DiffHunk = &value
		}
	}
	if comment.PullRequestReview != nil {
		if reviewID := strings.TrimSpace(comment.PullRequestReview.ID); reviewID != "" {
			reply.ReviewID = &reviewID
		}
		if comment.PullRequestReview.DatabaseID != nil {
			reply.ReviewDatabaseID = comment.PullRequestReview.DatabaseID
		}
		if state := strings.TrimSpace(comment.PullRequestReview.State); state != "" {
			reply.ReviewState = &state
		}
	}
	if comment.ReplyTo != nil {
		if replyToID := strings.TrimSpace(comment.ReplyTo.ID); replyToID != "" {
			reply.ReplyToCommentID = &replyToID
		}
	}

	return reply, nil
}
