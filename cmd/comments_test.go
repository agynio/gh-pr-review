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

func TestCommentsListCommandJSON(t *testing.T) {
	originalFactory := apiClientFactory
	defer func() { apiClientFactory = originalFactory }()

	fake := &commandFakeAPI{}
	fake.restFunc = func(method, path string, params map[string]string, body interface{}, result interface{}) error {
		if path == "repos/octo/demo/pulls/7/reviews/55/comments" {
			payload := []map[string]interface{}{{"id": 1, "body": "hi"}}
			return assignJSON(result, payload)
		}
		return errors.New("unexpected path")
	}
	apiClientFactory = func(host string) ghcli.API { return fake }

	root := newRootCommand()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetArgs([]string{"comments", "--list", "--review-id", "55", "octo/demo#7"})

	err := root.Execute()
	require.NoError(t, err)
	assert.Empty(t, stderr.String())

	var payload []map[string]interface{}
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &payload))
	require.Len(t, payload, 1)
	assert.Equal(t, float64(1), payload[0]["id"])
}

func TestCommentsReplyCommand(t *testing.T) {
	originalFactory := apiClientFactory
	defer func() { apiClientFactory = originalFactory }()

	fake := &commandFakeAPI{}
	fake.restFunc = func(method, path string, params map[string]string, body interface{}, result interface{}) error {
		switch path {
		case "repos/octo/demo/pulls/7/comments/5/replies":
			payload := map[string]interface{}{"id": 99, "body": "ack"}
			return assignJSON(result, payload)
		default:
			return errors.New("unexpected path")
		}
	}
	apiClientFactory = func(host string) ghcli.API { return fake }

	root := newRootCommand()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetArgs([]string{"comments", "reply", "--comment-id", "5", "--body", "ack", "octo/demo#7"})

	err := root.Execute()
	require.NoError(t, err)
	assert.Empty(t, stderr.String())

	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &payload))
	assert.Equal(t, float64(99), payload["id"])
}

func TestCommentsRequiresSelectorOrPR(t *testing.T) {
	originalFactory := apiClientFactory
	defer func() { apiClientFactory = originalFactory }()

	root := newRootCommand()
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"comments", "--list", "--review-id", "55"})

	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must specify a pull request")
}

func TestCommentsIDsCommand(t *testing.T) {
	originalFactory := apiClientFactory
	defer func() { apiClientFactory = originalFactory }()

	fake := &commandFakeAPI{}
	fake.restFunc = func(method, path string, params map[string]string, body interface{}, result interface{}) error {
		switch path {
		case "repos/octo/demo/pulls/7/reviews/55/comments":
			require.Equal(t, "50", params["per_page"])
			require.Equal(t, "2", params["page"])
			payload := []map[string]interface{}{
				{
					"id":       201,
					"body":     "hello",
					"line":     42,
					"user":     map[string]interface{}{"login": "octocat", "id": 77},
					"path":     "file.go",
					"html_url": "https://example.com/comment",
				},
			}
			return assignJSON(result, payload)
		default:
			return errors.New("unexpected path")
		}
	}
	apiClientFactory = func(host string) ghcli.API { return fake }

	root := newRootCommand()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetArgs([]string{"comments", "ids", "--review_id", "55", "--limit", "1", "--per_page", "50", "--page", "2", "octo/demo#7"})

	err := root.Execute()
	require.NoError(t, err)
	assert.Empty(t, stderr.String())

	var payload []map[string]interface{}
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &payload))
	require.Len(t, payload, 1)
	assert.Equal(t, float64(201), payload[0]["id"])
	assert.Equal(t, "hello", payload[0]["body"])
	user, ok := payload[0]["user"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "octocat", user["login"])
	assert.Equal(t, "file.go", payload[0]["path"])
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
