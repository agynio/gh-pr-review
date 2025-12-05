package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/Agyn-sandbox/gh-pr-review/internal/ghcli"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type commandFakeAPI struct {
	restFunc    func(method, path string, params map[string]string, body interface{}, result interface{}) error
	graphqlFunc func(query string, variables map[string]interface{}, result interface{}) error
}

func (f *commandFakeAPI) REST(method, path string, params map[string]string, body interface{}, result interface{}) error {
	if f.restFunc == nil {
		return errors.New("unexpected REST call")
	}
	return f.restFunc(method, path, params, body, result)
}

func (f *commandFakeAPI) GraphQL(query string, variables map[string]interface{}, result interface{}) error {
	if f.graphqlFunc == nil {
		return errors.New("unexpected GraphQL call")
	}
	return f.graphqlFunc(query, variables, result)
}

func TestCommentsCommandRootShowsGuidance(t *testing.T) {
	originalFactory := apiClientFactory
	defer func() { apiClientFactory = originalFactory }()

	root := newRootCommand()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetArgs([]string{"comments"})

	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "comments reply")
	assert.Contains(t, err.Error(), "review report")
}

func TestCommentsReplyCommand(t *testing.T) {
	originalFactory := apiClientFactory
	defer func() { apiClientFactory = originalFactory }()

	fake := &commandFakeAPI{}
	fake.graphqlFunc = func(query string, variables map[string]interface{}, result interface{}) error {
		require.Contains(t, query, "AddPullRequestReviewThreadReply")
		input, ok := variables["input"].(map[string]interface{})
		require.True(t, ok)
		require.Equal(t, "PRRT_thread", input["pullRequestReviewThreadId"])
		require.Equal(t, "ack", input["body"])
		require.Equal(t, "PRR_pending", input["pullRequestReviewId"])

		payload := map[string]interface{}{
			"addPullRequestReviewThreadReply": map[string]interface{}{
				"comment": map[string]interface{}{
					"id":         "PRRC_reply",
					"databaseId": 101,
					"body":       "ack",
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
					"isResolved": false,
					"isOutdated": false,
				},
			},
		}
		return assignJSON(result, payload)
	}
	apiClientFactory = func(host string) ghcli.API { return fake }

	root := newRootCommand()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetArgs([]string{"comments", "reply", "--thread-id", "PRRT_thread", "--review-id", "PRR_pending", "--body", "ack", "octo/demo#7"})

	err := root.Execute()
	require.NoError(t, err)
	assert.Empty(t, stderr.String())

	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &payload))
	assert.Equal(t, "PRRC_reply", payload["id"])
	assert.Equal(t, "ack", payload["body"])
	assert.Equal(t, "PRRT_thread", payload["thread_id"])
	assert.Equal(t, "octocat", payload["author_login"])
	assert.Equal(t, "https://example.com/comment", payload["html_url"])
	assert.Equal(t, "internal/service.go", payload["path"])
	assert.Equal(t, float64(101), payload["database_id"])
	assert.Equal(t, "@@ -10,5 +10,7 @@", payload["diff_hunk"])
	assert.Equal(t, false, payload["thread_is_resolved"])
	assert.Equal(t, false, payload["thread_is_outdated"])
	assert.Equal(t, "PRRC_parent", payload["reply_to_comment_id"])
	assert.Equal(t, "PRR_pending", payload["review_id"])
	assert.Equal(t, float64(202), payload["review_database_id"])
	assert.Equal(t, "PENDING", payload["review_state"])
}

func TestCommentsReplyCommandConcise(t *testing.T) {
	originalFactory := apiClientFactory
	defer func() { apiClientFactory = originalFactory }()

	fake := &commandFakeAPI{}
	fake.graphqlFunc = func(query string, variables map[string]interface{}, result interface{}) error {
		require.Contains(t, query, "AddPullRequestReviewThreadReply")
		input, ok := variables["input"].(map[string]interface{})
		require.True(t, ok)
		require.Equal(t, "PRRT_thread", input["pullRequestReviewThreadId"])
		require.Equal(t, "ack", input["body"])
		_, hasReview := input["pullRequestReviewId"]
		require.False(t, hasReview)

		payload := map[string]interface{}{
			"addPullRequestReviewThreadReply": map[string]interface{}{
				"comment": map[string]interface{}{
					"id":                "PRRC_reply",
					"databaseId":        nil,
					"body":              "ack",
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
					"isResolved": true,
					"isOutdated": false,
				},
			},
		}
		return assignJSON(result, payload)
	}
	apiClientFactory = func(host string) ghcli.API { return fake }

	root := newRootCommand()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetArgs([]string{"comments", "reply", "--thread-id", "PRRT_thread", "--body", "ack", "--concise", "octo/demo#7"})

	err := root.Execute()
	require.NoError(t, err)
	assert.Empty(t, stderr.String())

	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &payload))
	assert.Equal(t, 1, len(payload))
	assert.Equal(t, "PRRC_reply", payload["id"])
}

func assignJSON(result interface{}, payload interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, result)
}

func TestMain(m *testing.M) {
	// Ensure tests don't inherit GH_HOST requirements.
	_ = os.Unsetenv("GH_HOST")
	os.Exit(m.Run())
}
