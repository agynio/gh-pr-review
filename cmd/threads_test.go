package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/agynio/gh-pr-review/internal/ghcli"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestThreadsListCommandOutputsJSON(t *testing.T) {
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
	root.SetArgs([]string{"threads", "list", "--unresolved", "--mine", "--repo", "octo/demo", "5"})

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

func TestThreadsResolveCommandByThreadID(t *testing.T) {
	originalFactory := apiClientFactory
	defer func() { apiClientFactory = originalFactory }()

	fake := &commandFakeAPI{}
	fake.restFunc = func(method, path string, params map[string]string, body interface{}, result interface{}) error {
		return errors.New("unexpected REST call")
	}
	fake.graphqlFunc = func(query string, variables map[string]interface{}, result interface{}) error {
		switch {
		case strings.Contains(query, "ThreadDetails"):
			payload := map[string]interface{}{
				"node": map[string]interface{}{
					"id":                 "T_thread",
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
						"id":         "T_thread",
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
	root.SetArgs([]string{"threads", "resolve", "--thread-id", "T_thread", "--repo", "octo/demo", "9"})

	err := root.Execute()
	require.NoError(t, err)
	assert.Empty(t, stderr.String())

	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &payload))
	assert.Equal(t, "T_thread", payload["thread_node_id"])
	assert.Equal(t, true, payload["is_resolved"])
}

func TestThreadsUnresolveCommandByThreadID(t *testing.T) {
	originalFactory := apiClientFactory
	defer func() { apiClientFactory = originalFactory }()

	fake := &commandFakeAPI{}
	fake.restFunc = func(method, path string, params map[string]string, body interface{}, result interface{}) error {
		return errors.New("unexpected REST call")
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
	root.SetArgs([]string{"threads", "unresolve", "--thread-id", "T_thread", "--repo", "octo/demo", "9"})

	err := root.Execute()
	require.NoError(t, err)
	assert.Empty(t, stderr.String())

	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &payload))
	assert.Equal(t, "T_thread", payload["thread_node_id"])
	assert.Equal(t, false, payload["is_resolved"])
}

func TestThreadsUnresolveRequiresIdentifier(t *testing.T) {
	root := newRootCommand()
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"threads", "unresolve", "octo/demo#2"})

	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--thread-id is required")
}

func TestThreadsResolveRequiresThreadID(t *testing.T) {
	root := newRootCommand()
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"threads", "resolve", "octo/demo#1"})

	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--thread-id is required")
}

// ─── F2: resolve-all command test ─────────────────────────────────────────────

func TestThreadsResolveAllCommand(t *testing.T) {
	originalFactory := apiClientFactory
	defer func() { apiClientFactory = originalFactory }()

	fake := &commandFakeAPI{}
	fake.restFunc = func(method, path string, params map[string]string, body interface{}, result interface{}) error {
		switch path {
		case "repos/octo/demo":
			return assignJSON(result, map[string]interface{}{"full_name": "octo/demo"})
		case "repos/octo/demo/pulls/3":
			return assignJSON(result, map[string]interface{}{"node_id": "PR_bulk"})
		default:
			return errors.New("unexpected path: " + path)
		}
	}
	fake.graphqlFunc = func(query string, variables map[string]interface{}, result interface{}) error {
		ts := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		switch {
		case strings.Contains(query, "reviewThreads"):
			return assignJSON(result, map[string]interface{}{
				"node": map[string]interface{}{
					"reviewThreads": map[string]interface{}{
						"nodes": []map[string]interface{}{
							{
								"id": "TBulk1", "isResolved": false, "isOutdated": false, "path": "x.go",
								"viewerCanResolve": true, "viewerCanUnresolve": false,
								"comments": map[string]interface{}{"nodes": []map[string]interface{}{
									{"viewerDidAuthor": false, "updatedAt": ts.Format(time.RFC3339), "databaseId": 5},
								}},
							},
						},
						"pageInfo": map[string]interface{}{"hasNextPage": false, "endCursor": ""},
					},
				},
			})
		case strings.Contains(query, "resolveReviewThread"):
			return assignJSON(result, map[string]interface{}{
				"resolveReviewThread": map[string]interface{}{
					"thread": map[string]interface{}{"id": "TBulk1", "isResolved": true},
				},
			})
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
	root.SetArgs([]string{"threads", "resolve-all", "--repo", "octo/demo", "3"})

	err := root.Execute()
	require.NoError(t, err)
	assert.Empty(t, stderr.String())

	var payload []map[string]interface{}
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &payload))
	require.Len(t, payload, 1)
	assert.Equal(t, "TBulk1", payload[0]["thread_node_id"])
	assert.Equal(t, true, payload[0]["is_resolved"])
}

func TestThreadsListSinceInvalidTimestamp(t *testing.T) {
	root := newRootCommand()
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"threads", "list", "--repo", "octo/demo", "--since", "not-a-timestamp", "5"})
	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--since")
}

func TestThreadsListOutputIDs(t *testing.T) {
	originalFactory := apiClientFactory
	defer func() { apiClientFactory = originalFactory }()

	fake := &commandFakeAPI{}
	fake.restFunc = func(method, path string, params map[string]string, body interface{}, result interface{}) error {
		switch path {
		case "repos/octo/demo":
			return assignJSON(result, map[string]interface{}{"full_name": "octo/demo"})
		case "repos/octo/demo/pulls/5":
			return assignJSON(result, map[string]interface{}{"node_id": "PR_node"})
		default:
			return errors.New("unexpected path: " + path)
		}
	}
	fake.graphqlFunc = func(query string, variables map[string]interface{}, result interface{}) error {
		return assignJSON(result, map[string]interface{}{
			"node": map[string]interface{}{
				"reviewThreads": map[string]interface{}{
					"nodes": []map[string]interface{}{
						{
							"id": "T_alpha", "isResolved": false, "isOutdated": false, "path": "a.go",
							"viewerCanResolve": true, "viewerCanUnresolve": false,
							"comments": map[string]interface{}{"nodes": []map[string]interface{}{
								{"viewerDidAuthor": false, "updatedAt": "2026-01-01T00:00:00Z", "databaseId": 1},
							}},
						},
						{
							"id": "T_beta", "isResolved": false, "isOutdated": false, "path": "b.go",
							"viewerCanResolve": true, "viewerCanUnresolve": false,
							"comments": map[string]interface{}{"nodes": []map[string]interface{}{
								{"viewerDidAuthor": false, "updatedAt": "2025-01-01T00:00:00Z", "databaseId": 2},
							}},
						},
					},
					"pageInfo": map[string]interface{}{"hasNextPage": false, "endCursor": ""},
				},
			},
		})
	}
	apiClientFactory = func(host string) ghcli.API { return fake }

	root := newRootCommand()
	stdout := &bytes.Buffer{}
	root.SetOut(stdout)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"threads", "list", "-o", "ids", "--repo", "octo/demo", "5"})

	err := root.Execute()
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	require.Len(t, lines, 2)
	assert.Equal(t, "T_alpha", lines[0])
	assert.Equal(t, "T_beta", lines[1])
}
