package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Agyn-sandbox/gh-pr-review/internal/ghcli"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestThreadsListCommandJSON(t *testing.T) {
	originalFactory := apiClientFactory
	defer func() { apiClientFactory = originalFactory }()

	fake := &commandFakeAPI{}
	fake.restFunc = func(method, path string, params map[string]string, body interface{}, result interface{}) error {
		if method != "GET" {
			return errors.New("unexpected method")
		}
		switch path {
		case "repos/octo/demo":
			payload := map[string]interface{}{"full_name": "octo/demo"}
			return assignJSON(result, payload)
		case "repos/octo/demo/pulls/5":
			payload := map[string]interface{}{"node_id": "PR_node"}
			return assignJSON(result, payload)
		default:
			return errors.New("unexpected path")
		}
	}
	fake.graphqlFunc = func(query string, variables map[string]interface{}, result interface{}) error {
		if !strings.Contains(query, "reviewThreads") {
			return errors.New("unexpected query")
		}
		payload := map[string]interface{}{
			"node": map[string]interface{}{
				"reviewThreads": map[string]interface{}{
					"nodes": []map[string]interface{}{
						{
							"id":                 "T_node",
							"isResolved":         false,
							"isOutdated":         false,
							"path":               "internal/service.go",
							"line":               27,
							"viewerCanResolve":   false,
							"viewerCanUnresolve": true,
							"comments": map[string]interface{}{
								"nodes": []map[string]interface{}{
									{
										"viewerDidAuthor": true,
										"updatedAt":       time.Date(2025, 12, 2, 15, 0, 0, 0, time.UTC).Format(time.RFC3339),
										"databaseId":      101,
									},
								},
							},
						},
						{
							"id":                 "T_resolved",
							"isResolved":         true,
							"isOutdated":         false,
							"path":               "ignored.go",
							"viewerCanResolve":   true,
							"viewerCanUnresolve": true,
							"comments": map[string]interface{}{
								"nodes": []map[string]interface{}{},
							},
						},
					},
					"pageInfo": map[string]interface{}{
						"hasNextPage": false,
						"endCursor":   "",
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
	root.SetArgs([]string{"threads", "list", "--json", "--unresolved", "--mine", "octo/demo#5"})

	err := root.Execute()
	require.NoError(t, err)
	assert.Empty(t, stderr.String())

	var payload []map[string]interface{}
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &payload))
	require.Len(t, payload, 1)
	assert.Equal(t, "T_node", payload[0]["threadId"])
	assert.Equal(t, "internal/service.go", payload[0]["path"])
	assert.Equal(t, float64(27), payload[0]["line"])
}

func TestThreadsResolveCommandByCommentID(t *testing.T) {
	originalFactory := apiClientFactory
	defer func() { apiClientFactory = originalFactory }()

	fake := &commandFakeAPI{}
	fake.restFunc = func(method, path string, params map[string]string, body interface{}, result interface{}) error {
		if method != "GET" {
			return errors.New("unexpected method")
		}
		switch path {
		case "repos/octo/demo":
			return assignJSON(result, map[string]interface{}{"full_name": "octo/demo"})
		case "repos/octo/demo/pulls/9":
			return assignJSON(result, map[string]interface{}{"node_id": "PR_node"})
		case "repos/octo/demo/pulls/comments/88":
			return assignJSON(result, map[string]interface{}{"node_id": "C_node"})
		default:
			return errors.New("unexpected path")
		}
	}
	fake.graphqlFunc = func(query string, variables map[string]interface{}, result interface{}) error {
		switch {
		case strings.Contains(query, "CommentThread"):
			payload := map[string]interface{}{
				"node": map[string]interface{}{
					"pullRequestReviewThread": map[string]interface{}{"id": "T_comment"},
				},
			}
			return assignJSON(result, payload)
		case strings.Contains(query, "ThreadDetails"):
			payload := map[string]interface{}{
				"node": map[string]interface{}{
					"id":                 "T_comment",
					"isResolved":         false,
					"viewerCanResolve":   true,
					"viewerCanUnresolve": true,
				},
			}
			return assignJSON(result, payload)
		case strings.Contains(query, "resolveReviewThread"):
			payload := map[string]interface{}{
				"resolveReviewThread": map[string]interface{}{
					"thread": map[string]interface{}{
						"id":         "T_comment",
						"isResolved": true,
					},
				},
			}
			return assignJSON(result, payload)
		default:
			return errors.New("unexpected query")
		}
	}
	apiClientFactory = func(host string) ghcli.API { return fake }

	root := newRootCommand()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetArgs([]string{"threads", "resolve", "--json", "--comment-id", "88", "octo/demo#9"})

	err := root.Execute()
	require.NoError(t, err)
	assert.Empty(t, stderr.String())

	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &payload))
	assert.Equal(t, "T_comment", payload["threadId"])
	assert.Equal(t, true, payload["changed"])
	assert.Equal(t, true, payload["isResolved"])
}

func TestThreadsUnresolveCommandByThreadID(t *testing.T) {
	originalFactory := apiClientFactory
	defer func() { apiClientFactory = originalFactory }()

	fake := &commandFakeAPI{}
	fake.restFunc = func(method, path string, params map[string]string, body interface{}, result interface{}) error {
		if method != "GET" {
			return errors.New("unexpected method")
		}
		switch path {
		case "repos/octo/demo":
			return assignJSON(result, map[string]interface{}{"full_name": "octo/demo"})
		case "repos/octo/demo/pulls/9":
			return assignJSON(result, map[string]interface{}{"node_id": "PR_node"})
		default:
			return errors.New("unexpected path")
		}
	}
	fake.graphqlFunc = func(query string, variables map[string]interface{}, result interface{}) error {
		switch {
		case strings.Contains(query, "ThreadDetails"):
			payload := map[string]interface{}{
				"node": map[string]interface{}{
					"id":                 "T_thread",
					"isResolved":         true,
					"viewerCanResolve":   true,
					"viewerCanUnresolve": true,
				},
			}
			return assignJSON(result, payload)
		case strings.Contains(query, "unresolveReviewThread"):
			payload := map[string]interface{}{
				"unresolveReviewThread": map[string]interface{}{
					"thread": map[string]interface{}{
						"id":         "T_thread",
						"isResolved": false,
					},
				},
			}
			return assignJSON(result, payload)
		default:
			return errors.New("unexpected query")
		}
	}
	apiClientFactory = func(host string) ghcli.API { return fake }

	root := newRootCommand()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetArgs([]string{"threads", "unresolve", "--json", "--thread-id", "T_thread", "octo/demo#9"})

	err := root.Execute()
	require.NoError(t, err)
	assert.Empty(t, stderr.String())

	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &payload))
	assert.Equal(t, "T_thread", payload["threadId"])
	assert.Equal(t, true, payload["changed"])
	assert.Equal(t, false, payload["isResolved"])
}

func TestThreadsUnresolveRequiresIdentifier(t *testing.T) {
	root := newRootCommand()
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"threads", "unresolve", "--json", "octo/demo#2"})

	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "thread-id or --comment-id")
}

func TestThreadsResolveRequiresExclusiveSelector(t *testing.T) {
	root := newRootCommand()
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"threads", "resolve", "--json", "--thread-id", "T1", "--comment-id", "2", "octo/demo#1"})

	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "either --thread-id or --comment-id")
}

func TestThreadsListRequiresJSONFlag(t *testing.T) {
	root := newRootCommand()
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"threads", "list", "octo/demo#5"})

	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--json")
}
