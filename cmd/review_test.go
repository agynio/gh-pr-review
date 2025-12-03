package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"

	"github.com/Agyn-sandbox/gh-pr-review/internal/ghcli"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReviewStartCommand(t *testing.T) {
	originalFactory := apiClientFactory
	defer func() { apiClientFactory = originalFactory }()

	fake := &commandFakeAPI{}
	fake.restFunc = func(method, path string, params map[string]string, body interface{}, result interface{}) error {
		if path == "repos/octo/demo/pulls/7" {
			payload := map[string]interface{}{"node_id": "PR1", "head": map[string]interface{}{"sha": "abc"}}
			return assignJSON(result, payload)
		}
		return errors.New("unexpected path")
	}
	fake.graphqlFunc = func(query string, variables map[string]interface{}, result interface{}) error {
		payload := map[string]interface{}{
			"data": map[string]interface{}{
				"addPullRequestReview": map[string]interface{}{
					"pullRequestReview": map[string]interface{}{"id": "RV1", "state": "PENDING"},
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
	root.SetArgs([]string{"review", "--start", "octo/demo#7"})

	err := root.Execute()
	require.NoError(t, err)
	assert.Empty(t, stderr.String())

	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &payload))
	assert.Equal(t, "RV1", payload["id"])
}

func TestReviewAddCommentCommand(t *testing.T) {
	originalFactory := apiClientFactory
	defer func() { apiClientFactory = originalFactory }()

	fake := &commandFakeAPI{}
	fake.graphqlFunc = func(query string, variables map[string]interface{}, result interface{}) error {
		payload := map[string]interface{}{
			"data": map[string]interface{}{
				"addPullRequestReviewThread": map[string]interface{}{
					"thread": map[string]interface{}{"id": "THREAD1", "path": "file.go", "isOutdated": false},
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
	root.SetArgs([]string{"review", "--add-comment", "--review-id", "RV1", "--path", "file.go", "--line", "12", "--body", "note", "octo/demo#7"})

	err := root.Execute()
	require.NoError(t, err)
	assert.Empty(t, stderr.String())

	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &payload))
	assert.Equal(t, "THREAD1", payload["id"])
}

func TestReviewSubmitCommand(t *testing.T) {
	originalFactory := apiClientFactory
	defer func() { apiClientFactory = originalFactory }()

	fake := &commandFakeAPI{}
	fake.graphqlFunc = func(query string, variables map[string]interface{}, result interface{}) error {
		payload := map[string]interface{}{
			"data": map[string]interface{}{
				"submitPullRequestReview": map[string]interface{}{
					"pullRequestReview": map[string]interface{}{"id": "RV1", "state": "COMMENTED"},
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
	root.SetArgs([]string{"review", "--submit", "--review-id", "RV1", "--event", "COMMENT", "octo/demo#7"})

	err := root.Execute()
	require.NoError(t, err)
	assert.Empty(t, stderr.String())

	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &payload))
	assert.Equal(t, "RV1", payload["id"])
}

func TestReviewLatestIDCommand(t *testing.T) {
	originalFactory := apiClientFactory
	defer func() { apiClientFactory = originalFactory }()

	fake := &commandFakeAPI{}
	fake.restFunc = func(method, path string, params map[string]string, body interface{}, result interface{}) error {
		switch path {
		case "user":
			return assignJSON(result, map[string]interface{}{"login": "casey"})
		case "repos/octo/demo/pulls/7/reviews":
			require.Equal(t, "50", params["per_page"])
			require.Equal(t, "2", params["page"])
			payload := []map[string]interface{}{
				{
					"id":                 10,
					"state":              "COMMENTED",
					"submitted_at":       "2024-06-01T12:00:00Z",
					"author_association": "MEMBER",
					"html_url":           "https://example.com/review",
					"user":               map[string]interface{}{"login": "casey", "id": 77},
				},
			}
			return assignJSON(result, payload)
		default:
			return errors.New("unexpected path: " + path)
		}
	}
	apiClientFactory = func(host string) ghcli.API { return fake }

	root := newRootCommand()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetArgs([]string{"review", "latest-id", "--per_page", "50", "--page", "2", "octo/demo#7"})

	err := root.Execute()
	require.NoError(t, err)
	assert.Empty(t, stderr.String())

	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &payload))
	assert.Equal(t, float64(10), payload["id"])
	assert.Equal(t, "COMMENTED", payload["state"])
	assert.Equal(t, "https://example.com/review", payload["html_url"])

	user, ok := payload["user"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "casey", user["login"])
}
