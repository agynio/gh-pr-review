package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/Agyn-sandbox/gh-pr-review/internal/ghcli"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type obj = map[string]interface{}
type objSlice = []map[string]interface{}

func TestReviewStartCommand_GraphQLOnly(t *testing.T) {
	originalFactory := apiClientFactory
	defer func() { apiClientFactory = originalFactory }()

	fake := &commandFakeAPI{}
	call := 0
	fake.graphqlFunc = func(query string, variables map[string]interface{}, result interface{}) error {
		call++
		switch call {
		case 1:
			payload := map[string]interface{}{
				"repository": map[string]interface{}{
					"pullRequest": map[string]interface{}{
						"id":         "PRR_node",
						"headRefOid": "abc123",
					},
				},
			}
			return assignJSON(result, payload)
		case 2:
			payload := map[string]interface{}{
				"addPullRequestReview": map[string]interface{}{
					"pullRequestReview": map[string]interface{}{
						"id":         "PRR_review",
						"state":      "PENDING",
						"databaseId": float64(88),
						"url":        "https://example.com/review/PRR_review",
					},
				},
			}
			return assignJSON(result, payload)
		default:
			return errors.New("unexpected graphql invocation")
		}
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
	assert.Equal(t, "PRR_review", payload["id"])
	assert.Equal(t, "PENDING", payload["state"])
	assert.Equal(t, float64(88), payload["database_id"])
	assert.Equal(t, "https://example.com/review/PRR_review", payload["html_url"])
	_, hasSubmitted := payload["submitted_at"]
	assert.False(t, hasSubmitted)
	assert.Equal(t, 2, call)
}

func TestReviewAddCommentCommand_GraphQLOnly(t *testing.T) {
	originalFactory := apiClientFactory
	defer func() { apiClientFactory = originalFactory }()

	fake := &commandFakeAPI{}
	fake.graphqlFunc = func(query string, variables map[string]interface{}, result interface{}) error {
		input, ok := variables["input"].(map[string]interface{})
		require.True(t, ok)
		require.Equal(t, "PRR_review", input["pullRequestReviewId"])
		require.Equal(t, "scenario.md", input["path"])
		require.Equal(t, 12, input["line"])
		require.Equal(t, "RIGHT", input["side"])
		require.Equal(t, "note", input["body"])

		payload := map[string]interface{}{
			"addPullRequestReviewThread": map[string]interface{}{
				"thread": map[string]interface{}{
					"id":         "THREAD1",
					"path":       "scenario.md",
					"isOutdated": false,
					"line":       12,
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
	root.SetArgs([]string{"review", "--add-comment", "--review-id", "PRR_review", "--path", "scenario.md", "--line", "12", "--body", "note", "octo/demo#7"})

	err := root.Execute()
	require.NoError(t, err)
	assert.Empty(t, stderr.String())

	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &payload))
	assert.Equal(t, "THREAD1", payload["id"])
	assert.Equal(t, "scenario.md", payload["path"])
	assert.Equal(t, false, payload["is_outdated"])
	assert.Equal(t, float64(12), payload["line"])
}

func TestReviewAddCommentCommandRequiresGraphQLReviewID(t *testing.T) {
	originalFactory := apiClientFactory
	defer func() { apiClientFactory = originalFactory }()

	fake := &commandFakeAPI{}
	fake.graphqlFunc = func(query string, variables map[string]interface{}, result interface{}) error {
		return errors.New("unexpected graphql invocation")
	}
	apiClientFactory = func(host string) ghcli.API { return fake }

	root := newRootCommand()
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"review", "--add-comment", "--review-id", "123", "--path", "scenario.md", "--line", "12", "--body", "note", "octo/demo#7"})

	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "GraphQL node id")
}

func TestReviewSubmitCommand(t *testing.T) {
	originalFactory := apiClientFactory
	defer func() { apiClientFactory = originalFactory }()

	fake := &commandFakeAPI{}
	fake.graphqlFunc = func(query string, variables map[string]interface{}, result interface{}) error {
		require.Contains(t, query, "submitPullRequestReview")
		payload, ok := variables["input"].(map[string]interface{})
		require.True(t, ok)
		require.Equal(t, "PRR_kwM123", payload["pullRequestReviewId"])
		require.Equal(t, "COMMENT", payload["event"])
		require.Equal(t, "Please update", payload["body"])

		return assignJSON(result, obj{
			"data": obj{
				"submitPullRequestReview": obj{
					"pullRequestReview": obj{"id": "PRR_kwM123"},
				},
			},
		})
	}
	apiClientFactory = func(host string) ghcli.API { return fake }

	root := newRootCommand()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetArgs([]string{"review", "--submit", "--review-id", "PRR_kwM123", "--event", "COMMENT", "--body", "Please update", "octo/demo#7"})

	err := root.Execute()
	require.NoError(t, err)
	assert.Empty(t, stderr.String())

	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &payload))
	assert.Equal(t, "Review submitted successfully", payload["status"])
}

func TestReviewSubmitCommandRequiresGraphQLReviewID(t *testing.T) {
	originalFactory := apiClientFactory
	defer func() { apiClientFactory = originalFactory }()

	fake := &commandFakeAPI{}
	fake.graphqlFunc = func(query string, variables map[string]interface{}, result interface{}) error {
		return errors.New("unexpected GraphQL call")
	}
	apiClientFactory = func(host string) ghcli.API { return fake }

	root := newRootCommand()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetArgs([]string{"review", "--submit", "--review-id", "511", "--event", "APPROVE", "octo/demo#7"})

	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "REST review id")
}

func TestReviewSubmitCommandRejectsNonPRRPrefix(t *testing.T) {
	originalFactory := apiClientFactory
	defer func() { apiClientFactory = originalFactory }()

	fake := &commandFakeAPI{}
	fake.graphqlFunc = func(query string, variables map[string]interface{}, result interface{}) error {
		return errors.New("unexpected GraphQL call")
	}
	apiClientFactory = func(host string) ghcli.API { return fake }

	root := newRootCommand()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetArgs([]string{"review", "--submit", "--review-id", "RANDOM_ID", "--event", "COMMENT", "octo/demo#7"})

	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "GraphQL review node id")
}

func TestReviewSubmitCommandAllowsNullReview(t *testing.T) {
	originalFactory := apiClientFactory
	defer func() { apiClientFactory = originalFactory }()

	fake := &commandFakeAPI{}
	fake.graphqlFunc = func(query string, variables map[string]interface{}, result interface{}) error {
		response := obj{
			"data": obj{
				"submitPullRequestReview": obj{
					"pullRequestReview": nil,
				},
			},
		}
		return assignJSON(result, response)
	}
	apiClientFactory = func(host string) ghcli.API { return fake }

	root := newRootCommand()
	stdout := &bytes.Buffer{}
	root.SetOut(stdout)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"review", "--submit", "--review-id", "PRR_kwM123", "--event", "COMMENT", "octo/demo#7"})

	err := root.Execute()
	require.NoError(t, err)
	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &payload))
	assert.Equal(t, "Review submitted successfully", payload["status"])
}

func TestReviewSubmitCommandHandlesGraphQLErrors(t *testing.T) {
	originalFactory := apiClientFactory
	defer func() { apiClientFactory = originalFactory }()

	fake := &commandFakeAPI{}
	fake.graphqlFunc = func(query string, variables map[string]interface{}, result interface{}) error {
		return &ghcli.GraphQLError{Errors: []ghcli.GraphQLErrorEntry{{Message: "mutation failed", Path: []interface{}{"mutation", "submitPullRequestReview"}}}}
	}
	apiClientFactory = func(host string) ghcli.API { return fake }

	root := newRootCommand()
	stdout := &bytes.Buffer{}
	root.SetOut(stdout)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"review", "--submit", "--review-id", "PRR_kwM123", "--event", "COMMENT", "octo/demo#7"})

	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "review submission failed")
	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &payload))
	assert.Equal(t, "Review submission failed", payload["status"])
	errorsField, ok := payload["errors"].([]interface{})
	require.True(t, ok)
	require.Len(t, errorsField, 1)
	first, ok := errorsField[0].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "mutation failed", first["message"])
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

func TestReviewPendingIDCommand(t *testing.T) {
	originalFactory := apiClientFactory
	defer func() { apiClientFactory = originalFactory }()

	fake := &commandFakeAPI{}
	fake.graphqlFunc = func(query string, variables map[string]interface{}, result interface{}) error {
		if strings.Contains(query, "ViewerLogin") {
			payload := obj{
				"data": obj{
					"viewer": obj{
						"login": "casey",
					},
				},
			}
			return assignJSON(result, payload)
		}

		payload := obj{
			"data": obj{
				"repository": obj{
					"pullRequest": obj{
						"reviews": obj{
							"nodes": objSlice{
								obj{
									"id":         "PRR_node_old",
									"databaseId": 10,
									"url":        "https://example.com/review/10",
									"state":      "PENDING",
									"author":     obj{"login": "casey"},
									"updatedAt":  "2024-06-01T12:00:00Z",
									"createdAt":  "2024-06-01T11:00:00Z",
								},
								obj{
									"id":         "PRR_node_new",
									"databaseId": 22,
									"url":        "https://example.com/review/22",
									"state":      "PENDING",
									"author":     obj{"login": "casey"},
									"updatedAt":  "2024-06-01T13:00:00Z",
									"createdAt":  "2024-06-01T12:30:00Z",
								},
							},
							"pageInfo": obj{"hasNextPage": false, "endCursor": ""},
						},
					},
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
	root.SetArgs([]string{"review", "pending-id", "octo/demo#7"})

	err := root.Execute()
	require.NoError(t, err)
	assert.Empty(t, stderr.String())

	var payload obj
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &payload))
	assert.Equal(t, "PRR_node_new", payload["id"])
	assert.Equal(t, float64(22), payload["database_id"])
	assert.Equal(t, "PENDING", payload["state"])
	assert.Equal(t, "https://example.com/review/22", payload["html_url"])

	user, ok := payload["user"].(obj)
	require.True(t, ok)
	assert.Equal(t, "casey", user["login"])
}

func TestReviewPendingIDCommandViewerMissing(t *testing.T) {
	originalFactory := apiClientFactory
	defer func() { apiClientFactory = originalFactory }()

	fake := &commandFakeAPI{}
	fake.graphqlFunc = func(query string, variables map[string]interface{}, result interface{}) error {
		if strings.Contains(query, "ViewerLogin") {
			payload := obj{
				"data": obj{
					"viewer": obj{
						"login": " ",
					},
				},
			}
			return assignJSON(result, payload)
		}
		return errors.New("unexpected query: " + query)
	}
	apiClientFactory = func(host string) ghcli.API { return fake }

	root := newReviewPendingIDCommand()
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"octo/demo#7"})

	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "pass --reviewer")
}
