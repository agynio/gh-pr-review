package comments

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/Agyn-sandbox/gh-pr-review/internal/resolver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeAPI struct {
	restFunc    func(method, path string, params map[string]string, body interface{}, result interface{}) error
	graphqlFunc func(query string, variables map[string]interface{}, result interface{}) error
}

func (f *fakeAPI) REST(method, path string, params map[string]string, body interface{}, result interface{}) error {
	if f.restFunc == nil {
		return errors.New("unexpected REST call")
	}
	return f.restFunc(method, path, params, body, result)
}

func (f *fakeAPI) GraphQL(query string, variables map[string]interface{}, result interface{}) error {
	if f.graphqlFunc == nil {
		return errors.New("unexpected GraphQL call")
	}
	return f.graphqlFunc(query, variables, result)
}

func assign(result interface{}, payload interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, result)
}

func TestServiceReply_RejectsMissingThreadID(t *testing.T) {
	api := &fakeAPI{}
	svc := NewService(api)

	_, err := svc.Reply(resolver.Identity{}, ReplyOptions{ThreadID: "", Body: "hello"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "thread id is required")
}

func TestServiceReply_RejectsBlankBody(t *testing.T) {
	api := &fakeAPI{}
	svc := NewService(api)

	_, err := svc.Reply(resolver.Identity{}, ReplyOptions{ThreadID: "PRRT_thread", Body: "   "})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "reply body is required")
}

func TestServiceReply_SendsMutation(t *testing.T) {
	api := &fakeAPI{}
	api.graphqlFunc = func(query string, variables map[string]interface{}, result interface{}) error {
		require.Contains(t, query, "AddPullRequestReviewThreadReply")
		input, ok := variables["input"].(map[string]interface{})
		require.True(t, ok)
		require.Equal(t, "PRRT_thread", input["pullRequestReviewThreadId"])
		require.Equal(t, "Body text", input["body"])
		require.Equal(t, "PRR_pending", input["pullRequestReviewId"])

		payload := map[string]interface{}{
			"addPullRequestReviewThreadReply": map[string]interface{}{
				"comment": map[string]interface{}{
					"id":         "PRRC_reply",
					"databaseId": 101,
					"body":       "Body text",
					"diffHunk":   "@@ -10,5 +10,7 @@",
					"path":       "internal/service.go",
					"url":        "https://example.com/comment",
					"createdAt":  "2025-12-03T10:00:00Z",
					"updatedAt":  "2025-12-03T10:05:00Z",
					"author":     map[string]interface{}{"login": "octocat"},
					"pullRequestReview": map[string]interface{}{
						"id":         "PRR_pending",
						"databaseId": 202,
						"state":      "PENDING",
					},
					"replyTo": map[string]interface{}{"id": "PRRC_parent"},
				},
				"thread": map[string]interface{}{
					"id":         "PRRT_thread",
					"isResolved": true,
					"isOutdated": false,
				},
			},
		}
		return assign(result, payload)
	}

	svc := NewService(api)
	reply, err := svc.Reply(resolver.Identity{}, ReplyOptions{ThreadID: "PRRT_thread", ReviewID: "PRR_pending", Body: "Body text"})
	require.NoError(t, err)
	assert.Equal(t, "PRRC_reply", reply.ID)
	assert.Equal(t, "PRRT_thread", reply.ThreadID)
	assert.True(t, reply.ThreadIsResolved)
	assert.False(t, reply.ThreadIsOutdated)
	assert.Equal(t, "Body text", reply.Body)
	assert.Equal(t, "internal/service.go", reply.Path)
	assert.Equal(t, "https://example.com/comment", reply.HtmlURL)
	assert.Equal(t, "octocat", reply.AuthorLogin)
	assert.Equal(t, "2025-12-03T10:00:00Z", reply.CreatedAt)
	assert.Equal(t, "2025-12-03T10:05:00Z", reply.UpdatedAt)
	if assert.NotNil(t, reply.DatabaseID) {
		assert.Equal(t, 101, *reply.DatabaseID)
	}
	if assert.NotNil(t, reply.DiffHunk) {
		assert.Equal(t, "@@ -10,5 +10,7 @@", *reply.DiffHunk)
	}
	if assert.NotNil(t, reply.ReviewID) {
		assert.Equal(t, "PRR_pending", *reply.ReviewID)
	}
	if assert.NotNil(t, reply.ReviewDatabaseID) {
		assert.Equal(t, 202, *reply.ReviewDatabaseID)
	}
	if assert.NotNil(t, reply.ReviewState) {
		assert.Equal(t, "PENDING", *reply.ReviewState)
	}
	if assert.NotNil(t, reply.ReplyToCommentID) {
		assert.Equal(t, "PRRC_parent", *reply.ReplyToCommentID)
	}
}

func TestServiceReply_OmitsOptionalFields(t *testing.T) {
	api := &fakeAPI{}
	api.graphqlFunc = func(query string, variables map[string]interface{}, result interface{}) error {
		input, ok := variables["input"].(map[string]interface{})
		require.True(t, ok)
		_, hasReview := input["pullRequestReviewId"]
		require.False(t, hasReview)

		payload := map[string]interface{}{
			"addPullRequestReviewThreadReply": map[string]interface{}{
				"comment": map[string]interface{}{
					"id":                "PRRC_reply",
					"databaseId":        nil,
					"body":              "Ack",
					"diffHunk":          "",
					"path":              "",
					"url":               "https://example.com/comment",
					"createdAt":         "2025-12-03T10:00:00Z",
					"updatedAt":         "2025-12-03T10:05:00Z",
					"author":            map[string]interface{}{"login": "octocat"},
					"pullRequestReview": nil,
					"replyTo":           nil,
				},
				"thread": map[string]interface{}{
					"id":         "PRRT_thread",
					"isResolved": false,
					"isOutdated": true,
				},
			},
		}
		return assign(result, payload)
	}

	svc := NewService(api)
	reply, err := svc.Reply(resolver.Identity{}, ReplyOptions{ThreadID: "PRRT_thread", Body: "Ack"})
	require.NoError(t, err)
	assert.Equal(t, "PRRC_reply", reply.ID)
	assert.Equal(t, "PRRT_thread", reply.ThreadID)
	assert.False(t, reply.ThreadIsResolved)
	assert.True(t, reply.ThreadIsOutdated)
	assert.Equal(t, "Ack", reply.Body)
	assert.Equal(t, "", reply.Path)
	assert.Equal(t, "https://example.com/comment", reply.HtmlURL)
	assert.Equal(t, "octocat", reply.AuthorLogin)
	assert.Nil(t, reply.DatabaseID)
	assert.Nil(t, reply.DiffHunk)
	assert.Nil(t, reply.ReviewID)
	assert.Nil(t, reply.ReviewDatabaseID)
	assert.Nil(t, reply.ReviewState)
	assert.Nil(t, reply.ReplyToCommentID)
}

func TestServiceReply_ErrorsOnMissingComment(t *testing.T) {
	api := &fakeAPI{}
	api.graphqlFunc = func(query string, variables map[string]interface{}, result interface{}) error {
		payload := map[string]interface{}{
			"addPullRequestReviewThreadReply": map[string]interface{}{
				"comment": nil,
				"thread":  nil,
			},
		}
		return assign(result, payload)
	}

	svc := NewService(api)
	_, err := svc.Reply(resolver.Identity{}, ReplyOptions{ThreadID: "PRRT_thread", Body: "Ack"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mutation response missing comment")
}

func TestServiceReply_ErrorsOnMissingThread(t *testing.T) {
	api := &fakeAPI{}
	api.graphqlFunc = func(query string, variables map[string]interface{}, result interface{}) error {
		payload := map[string]interface{}{
			"addPullRequestReviewThreadReply": map[string]interface{}{
				"comment": map[string]interface{}{
					"id":        "PRRC_reply",
					"body":      "Ack",
					"path":      "",
					"url":       "https://example.com/comment",
					"createdAt": "2025-12-03T10:00:00Z",
					"updatedAt": "2025-12-03T10:05:00Z",
					"author":    map[string]interface{}{"login": "octocat"},
				},
				"thread": nil,
			},
		}
		return assign(result, payload)
	}

	svc := NewService(api)
	_, err := svc.Reply(resolver.Identity{}, ReplyOptions{ThreadID: "PRRT_thread", Body: "Ack"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mutation response missing thread id")
}
