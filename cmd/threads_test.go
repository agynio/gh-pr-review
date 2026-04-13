package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"os/exec"
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
					"comments": map[string]interface{}{
						"nodes": []map[string]interface{}{
							{"id": "C_first"},
						},
					},
				},
			}
			return assignJSON(result, payload)
		case strings.Contains(query, "addPullRequestReviewThreadReply"):
			return nil
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
	root.SetArgs([]string{"threads", "resolve", "--thread-id", "T_thread", "--commit", "abc1234", "--repo", "octo/demo", "9"})

	err := root.Execute()
	require.NoError(t, err)
	assert.Empty(t, stderr.String())

	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &payload))
	assert.Equal(t, "T_thread", payload["thread_node_id"])
	assert.Equal(t, true, payload["is_resolved"])
	assert.Contains(t, payload["reply_body"], "abc1234")
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
		case strings.Contains(query, "ThreadDetails"):
			threadID, _ := variables["id"].(string)
			return assignJSON(result, map[string]interface{}{
				"node": map[string]interface{}{
					"id": threadID, "isResolved": false,
					"viewerCanResolve": true, "viewerCanUnresolve": false,
					"comments": map[string]interface{}{
						"nodes": []map[string]interface{}{
							{"id": "C_bulk_first"},
						},
					},
				},
			})
		case strings.Contains(query, "addPullRequestReviewThreadReply"):
			return nil
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
	root.SetArgs([]string{"threads", "resolve-all", "--repo", "octo/demo", "--commit", "def5678", "3"})

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

// F6: resolveCommitRef unit tests

func TestResolveCommitRefPassthroughHexSHA(t *testing.T) {
	// A valid hex SHA should be returned unchanged without calling git.
	sha := "abc1234"
	got, err := resolveCommitRef(sha)
	require.NoError(t, err)
	assert.Equal(t, sha, got)
}

func TestResolveCommitRefFullSHA(t *testing.T) {
	sha := "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2"
	got, err := resolveCommitRef(sha)
	require.NoError(t, err)
	assert.Equal(t, sha, got)
}

func TestResolveCommitRefResolvesSymbolicRefs(t *testing.T) {
	branchOut, err := exec.Command("git", "branch", "--show-current").Output()
	if err != nil {
		t.Skipf("git branch lookup unavailable: %v", err)
	}

	refs := []string{"HEAD", strings.TrimSpace(string(branchOut))}
	tests := make([]string, 0, len(refs))
	for _, ref := range refs {
		if strings.TrimSpace(ref) != "" {
			tests = append(tests, ref)
		}
	}
	for _, ref := range tests {
		ref := ref
		t.Run(ref, func(t *testing.T) {
			wantOut, err := exec.Command("git", "rev-parse", "--verify", "--short", "--end-of-options", ref+"^{commit}").Output()
			require.NoError(t, err)

			got, err := resolveCommitRef(ref)
			require.NoError(t, err)
			assert.Equal(t, strings.TrimSpace(string(wantOut)), got)
		})
	}
}

func TestResolveCommandWithHexSHACommit(t *testing.T) {
	originalFactory := apiClientFactory
	defer func() { apiClientFactory = originalFactory }()

	var capturedBody string
	fake := &commandFakeAPI{}
	fake.restFunc = func(method, path string, params map[string]string, body interface{}, result interface{}) error {
		return errors.New("unexpected REST call")
	}
	fake.graphqlFunc = func(query string, variables map[string]interface{}, result interface{}) error {
		switch {
		case strings.Contains(query, "ThreadDetails"):
			return assignJSON(result, map[string]interface{}{
				"node": map[string]interface{}{
					"id":                 "T_sha",
					"isResolved":         false,
					"viewerCanResolve":   true,
					"viewerCanUnresolve": false,
				},
			})
		case strings.Contains(query, "addPullRequestReviewThreadReply"):
			if b, ok := variables["body"].(string); ok {
				capturedBody = b
			}
			return nil
		case strings.Contains(query, "resolveReviewThread"):
			return assignJSON(result, map[string]interface{}{
				"resolveReviewThread": map[string]interface{}{
					"thread": map[string]interface{}{"id": "T_sha", "isResolved": true},
				},
			})
		default:
			return errors.New("unexpected query")
		}
	}
	apiClientFactory = func(host string) ghcli.API { return fake }

	root := newRootCommand()
	stdout := &bytes.Buffer{}
	root.SetOut(stdout)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"threads", "resolve", "--thread-id", "T_sha", "--commit", "abc1234", "--repo", "octo/demo", "9"})

	err := root.Execute()
	require.NoError(t, err)

	assert.Contains(t, capturedBody, "abc1234")

	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &payload))
	assert.Equal(t, "T_sha", payload["thread_node_id"])
	assert.Equal(t, true, payload["is_resolved"])
	assert.Contains(t, payload["reply_body"], "abc1234")
}

// ─── Validation gate tests ──────────────────────────────────────────────────

func TestResolveRejectsWithoutCommitOrThumbsDown(t *testing.T) {
	root := newRootCommand()
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"threads", "resolve", "--thread-id", "T_thread", "--repo", "octo/demo", "9"})

	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must provide --commit")
}

func TestResolveAcceptsThumbsDownWithoutCommit(t *testing.T) {
	originalFactory := apiClientFactory
	defer func() { apiClientFactory = originalFactory }()

	var reactCalled bool
	var reactContent string
	fake := &commandFakeAPI{}
	fake.restFunc = func(method, path string, params map[string]string, body interface{}, result interface{}) error {
		return errors.New("unexpected REST call")
	}
	fake.graphqlFunc = func(query string, variables map[string]interface{}, result interface{}) error {
		switch {
		case strings.Contains(query, "ThreadDetails"):
			return assignJSON(result, map[string]interface{}{
				"node": map[string]interface{}{
					"id": "T_td", "isResolved": false,
					"viewerCanResolve": true, "viewerCanUnresolve": false,
					"comments": map[string]interface{}{
						"nodes": []map[string]interface{}{
							{"id": "C_td_first"},
						},
					},
				},
			})
		case strings.Contains(query, "addPullRequestReviewThreadReply"):
			return nil
		case strings.Contains(query, "resolveReviewThread"):
			return assignJSON(result, map[string]interface{}{
				"resolveReviewThread": map[string]interface{}{
					"thread": map[string]interface{}{"id": "T_td", "isResolved": true},
				},
			})
		case strings.Contains(query, "addReaction"):
			reactCalled = true
			if c, ok := variables["content"].(string); ok {
				reactContent = c
			}
			return nil
		default:
			return errors.New("unexpected query")
		}
	}
	apiClientFactory = func(host string) ghcli.API { return fake }

	root := newRootCommand()
	stdout := &bytes.Buffer{}
	root.SetOut(stdout)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"threads", "resolve", "--thread-id", "T_td", "--react", "thumbs_down", "--message", "not a bug", "--repo", "octo/demo", "9"})

	err := root.Execute()
	require.NoError(t, err)

	assert.True(t, reactCalled, "addReaction should have been called")
	assert.Equal(t, "THUMBS_DOWN", reactContent)

	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &payload))
	assert.Equal(t, "T_td", payload["thread_node_id"])
	assert.Equal(t, true, payload["is_resolved"])
	assert.Equal(t, "THUMBS_DOWN", payload["reaction"])
}

func TestResolveRejectsInvalidReaction(t *testing.T) {
	root := newRootCommand()
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"threads", "resolve", "--thread-id", "T_thread", "--react", "invalid", "--repo", "octo/demo", "9"})

	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--react: invalid reaction")
}

func TestResolveAllRejectsWithoutCommit(t *testing.T) {
	root := newRootCommand()
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"threads", "resolve-all", "--repo", "octo/demo", "3"})

	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--commit is required for resolve-all")
}
