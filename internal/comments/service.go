package comments

import (
	"errors"
	"strings"

	"github.com/agynio/gh-pr-review/internal/ghcli"
	"github.com/agynio/gh-pr-review/internal/resolver"
)

const addThreadReplyMutation = `mutation AddPullRequestReviewThreadReply($input: AddPullRequestReviewThreadReplyInput!) {
  addPullRequestReviewThreadReply(input: $input) {
    comment {
      id
      body
      publishedAt
      author { login }
    }
  }
}`

const commentDetailsQuery = `query PullRequestReviewCommentDetails($id: ID!) {
  node(id: $id) {
    ... on PullRequestReviewComment {
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
  }
}`

const threadDetailsQuery = `query PullRequestReviewThreadDetails($id: ID!) {
  node(id: $id) {
    ... on PullRequestReviewThread {
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
	CommentNodeID    string  `json:"comment_node_id"`
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

type commentDetails struct {
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
}

type threadDetails struct {
	ID         string `json:"id"`
	IsResolved bool   `json:"isResolved"`
	IsOutdated bool   `json:"isOutdated"`
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
				ID          string `json:"id"`
				Body        string `json:"body"`
				PublishedAt string `json:"publishedAt"`
				Author      *struct {
					Login string `json:"login"`
				} `json:"author"`
			} `json:"comment"`
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
	commentDetails, err := s.loadCommentDetails(comment.ID)
	if err != nil {
		return Reply{}, err
	}

	threadDetails, err := s.loadThreadDetails(threadID)
	if err != nil {
		return Reply{}, err
	}

	reply := Reply{
		CommentNodeID:    commentDetails.ID,
		ThreadID:         threadID,
		ThreadIsResolved: threadDetails.IsResolved,
		ThreadIsOutdated: threadDetails.IsOutdated,
		Body:             commentDetails.Body,
		Path:             commentDetails.Path,
		HtmlURL:          commentDetails.URL,
		AuthorLogin:      commentDetails.Author.Login,
		CreatedAt:        commentDetails.CreatedAt,
		UpdatedAt:        commentDetails.UpdatedAt,
	}

	if commentDetails.DatabaseID != nil {
		reply.DatabaseID = commentDetails.DatabaseID
	}
	if commentDetails.DiffHunk != nil {
		trimmed := strings.TrimSpace(*commentDetails.DiffHunk)
		if trimmed != "" {
			value := *commentDetails.DiffHunk
			reply.DiffHunk = &value
		}
	}
	if commentDetails.PullRequestReview != nil {
		if reviewID := strings.TrimSpace(commentDetails.PullRequestReview.ID); reviewID != "" {
			reply.ReviewID = &reviewID
		}
		if commentDetails.PullRequestReview.DatabaseID != nil {
			reply.ReviewDatabaseID = commentDetails.PullRequestReview.DatabaseID
		}
		if state := strings.TrimSpace(commentDetails.PullRequestReview.State); state != "" {
			reply.ReviewState = &state
		}
	}
	if commentDetails.ReplyTo != nil {
		if replyToID := strings.TrimSpace(commentDetails.ReplyTo.ID); replyToID != "" {
			reply.ReplyToCommentID = &replyToID
		}
	}

	return reply, nil
}

func (s *Service) loadCommentDetails(id string) (commentDetails, error) {
	variables := map[string]interface{}{"id": id}
	var response struct {
		Node *commentDetails `json:"node"`
	}
	if err := s.API.GraphQL(commentDetailsQuery, variables, &response); err != nil {
		return commentDetails{}, err
	}
	if response.Node == nil || strings.TrimSpace(response.Node.ID) == "" {
		return commentDetails{}, errors.New("failed to load comment details")
	}
	if response.Node.Author == nil || strings.TrimSpace(response.Node.Author.Login) == "" {
		return commentDetails{}, errors.New("comment details missing author")
	}
	return *response.Node, nil
}

func (s *Service) loadThreadDetails(id string) (threadDetails, error) {
	variables := map[string]interface{}{"id": id}
	var response struct {
		Node *threadDetails `json:"node"`
	}
	if err := s.API.GraphQL(threadDetailsQuery, variables, &response); err != nil {
		return threadDetails{}, err
	}
	if response.Node == nil || strings.TrimSpace(response.Node.ID) == "" {
		return threadDetails{}, errors.New("failed to load thread details")
	}
	return *response.Node, nil
}
