package threads

import (
	"encoding/json"
	"errors"
	"strconv"
	"testing"
	"strings"
	"time"

	"github.com/agynio/gh-pr-review/internal/resolver"
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

func restStub(t *testing.T, owner, repo, canonical string, number int, nodeID string, next func(method, path string, params map[string]string, body interface{}, result interface{}) error) func(string, string, map[string]string, interface{}, interface{}) error {
	return func(method, path string, params map[string]string, body interface{}, result interface{}) error {
		require.Equal(t, "GET", method)

		switch path {
		case "repos/" + owner + "/" + repo:
			if canonical == "" {
				return assign(result, map[string]interface{}{})
			}
			return assign(result, map[string]interface{}{"full_name": canonical})
		case "repos/" + owner + "/" + repo + "/pulls/" + strconv.Itoa(number):
			return assign(result, map[string]interface{}{"node_id": nodeID})
		default:
			if next != nil {
				return next(method, path, params, body, result)
			}
			return errors.New("unexpected REST path: " + path)
		}
	}
}

func TestServiceListFiltersAndSort(t *testing.T) {
	svc := &Service{}
	svc.API = &fakeAPI{
		restFunc: restStub(t, "octo", "demo", "octo/demo", 5, "PR_node", nil),
		graphqlFunc: func(query string, variables map[string]interface{}, result interface{}) error {
			require.Equal(t, listThreadsQuery, query)
			require.Equal(t, "PR_node", variables["id"])

			ts1 := time.Date(2025, 12, 1, 10, 0, 0, 0, time.UTC)
			payload := map[string]interface{}{
				"node": map[string]interface{}{
					"reviewThreads": map[string]interface{}{
						"nodes": []map[string]interface{}{
							{
								"id":                 "T1",
								"isResolved":         false,
								"isOutdated":         false,
								"path":               "internal/file.go",
								"line":               42,
								"viewerCanResolve":   false,
								"viewerCanUnresolve": false,
								"comments": map[string]interface{}{
									"nodes": []map[string]interface{}{
										{
											"viewerDidAuthor": true,
											"updatedAt":       ts1,
											"databaseId":      101,
										},
									},
								},
							},
							{
								"id":                 "T2",
								"isResolved":         true,
								"isOutdated":         false,
								"path":               "internal/ignore.go",
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

			return assign(result, payload)
		},
	}

	identity := resolver.Identity{Owner: "octo", Repo: "demo", Number: 5}
	threads, err := svc.List(identity, ListOptions{OnlyUnresolved: true, MineOnly: true})
	require.NoError(t, err)
	require.Len(t, threads, 1)

	entry := threads[0]
	assert.Equal(t, "T1", entry.ThreadID)
	assert.False(t, entry.IsResolved)
	require.NotNil(t, entry.UpdatedAt)
	assert.Equal(t, "internal/file.go", entry.Path)
	require.NotNil(t, entry.Line)
	assert.Equal(t, 42, *entry.Line)
}

func TestServiceListMineIncludesUnresolvePermission(t *testing.T) {
	svc := &Service{}
	svc.API = &fakeAPI{
		restFunc: restStub(t, "octo", "demo", "octo/demo", 5, "PR_node", nil),
		graphqlFunc: func(query string, variables map[string]interface{}, result interface{}) error {
			require.Equal(t, listThreadsQuery, query)
			require.Equal(t, "PR_node", variables["id"])

			updated := time.Date(2025, 12, 3, 12, 0, 0, 0, time.UTC)
			payload := map[string]interface{}{
				"node": map[string]interface{}{
					"reviewThreads": map[string]interface{}{
						"nodes": []map[string]interface{}{
							{
								"id":                 "T-resolved",
								"isResolved":         true,
								"isOutdated":         false,
								"path":               "internal/file.go",
								"viewerCanResolve":   false,
								"viewerCanUnresolve": true,
								"comments": map[string]interface{}{
									"nodes": []map[string]interface{}{
										{
											"viewerDidAuthor": false,
											"updatedAt":       updated,
											"databaseId":      201,
										},
									},
								},
							},
							{
								"id":                 "T-ignored",
								"isResolved":         true,
								"isOutdated":         false,
								"path":               "internal/ignore.go",
								"viewerCanResolve":   false,
								"viewerCanUnresolve": false,
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

			return assign(result, payload)
		},
	}

	identity := resolver.Identity{Owner: "octo", Repo: "demo", Number: 5}
	threads, err := svc.List(identity, ListOptions{MineOnly: true})
	require.NoError(t, err)
	require.Len(t, threads, 1)
	assert.Equal(t, "T-resolved", threads[0].ThreadID)
}

func TestServiceListUnresolvedEmptyReturnsSlice(t *testing.T) {
	svc := &Service{}
	svc.API = &fakeAPI{
		restFunc: restStub(t, "octo", "demo", "octo/demo", 5, "PR_node", nil),
		graphqlFunc: func(query string, variables map[string]interface{}, result interface{}) error {
			require.Equal(t, listThreadsQuery, query)
			require.Equal(t, "PR_node", variables["id"])

			payload := map[string]interface{}{
				"node": map[string]interface{}{
					"reviewThreads": map[string]interface{}{
						"nodes": []map[string]interface{}{},
						"pageInfo": map[string]interface{}{
							"hasNextPage": false,
							"endCursor":   "",
						},
					},
				},
			}

			return assign(result, payload)
		},
	}

	identity := resolver.Identity{Owner: "octo", Repo: "demo", Number: 5}
	threads, err := svc.List(identity, ListOptions{OnlyUnresolved: true})
	require.NoError(t, err)
	require.NotNil(t, threads)
	assert.Empty(t, threads)
}

func TestResolveRequiresPermission(t *testing.T) {
	svc := &Service{}
	svc.API = &fakeAPI{
		graphqlFunc: func(query string, variables map[string]interface{}, result interface{}) error {
			switch query {
			case threadDetailsQuery:
				payload := map[string]interface{}{
					"node": map[string]interface{}{
						"id":                 "T1",
						"isResolved":         false,
						"viewerCanResolve":   false,
						"viewerCanUnresolve": true,
					},
				}
				return assign(result, payload)
			default:
				return errors.New("unexpected query")
			}
		},
	}

	identity := resolver.Identity{Owner: "octo", Repo: "demo", Number: 5}
	_, err := svc.Resolve(identity, ActionOptions{ThreadID: "T1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot resolve")
}

func TestResolveNoop(t *testing.T) {
	svc := &Service{}
	callCount := 0
	svc.API = &fakeAPI{
		graphqlFunc: func(query string, variables map[string]interface{}, result interface{}) error {
			switch query {
			case threadDetailsQuery:
				callCount++
				payload := map[string]interface{}{
					"node": map[string]interface{}{
						"id":                 "T2",
						"isResolved":         true,
						"viewerCanResolve":   true,
						"viewerCanUnresolve": true,
					},
				}
				return assign(result, payload)
			default:
				return errors.New("unexpected query")
			}
		},
	}

	identity := resolver.Identity{Owner: "octo", Repo: "demo", Number: 5}
	res, err := svc.Resolve(identity, ActionOptions{ThreadID: "T2"})
	require.NoError(t, err)
	assert.True(t, res.IsResolved)
	assert.Equal(t, "T2", res.ThreadNodeID)
	assert.Equal(t, 1, callCount)
}

func TestResolveMutatesThread(t *testing.T) {
	svc := &Service{}
	mutationCalled := false
	svc.API = &fakeAPI{
		graphqlFunc: func(query string, variables map[string]interface{}, result interface{}) error {
			switch query {
			case threadDetailsQuery:
				require.Equal(t, "T3", variables["id"])
				payload := map[string]interface{}{
					"node": map[string]interface{}{
						"id":                 "T3",
						"isResolved":         false,
						"viewerCanResolve":   true,
						"viewerCanUnresolve": true,
					},
				}
				return assign(result, payload)
			case resolveThreadMutation:
				mutationCalled = true
				payload := map[string]interface{}{
					"resolveReviewThread": map[string]interface{}{
						"thread": map[string]interface{}{
							"id":         "T3",
							"isResolved": true,
						},
					},
				}
				return assign(result, payload)
			default:
				return errors.New("unexpected query")
			}
		},
	}

	identity := resolver.Identity{Owner: "octo", Repo: "demo", Number: 5}
	res, err := svc.Resolve(identity, ActionOptions{ThreadID: "T3"})
	require.NoError(t, err)
	assert.True(t, mutationCalled)
	assert.True(t, res.IsResolved)
	assert.Equal(t, "T3", res.ThreadNodeID)
}

func TestUnresolveRequiresPermission(t *testing.T) {
	svc := &Service{}
	svc.API = &fakeAPI{
		graphqlFunc: func(query string, variables map[string]interface{}, result interface{}) error {
			switch query {
			case threadDetailsQuery:
				payload := map[string]interface{}{
					"node": map[string]interface{}{
						"id":                 "T7",
						"isResolved":         true,
						"viewerCanResolve":   true,
						"viewerCanUnresolve": false,
					},
				}
				return assign(result, payload)
			default:
				return errors.New("unexpected query")
			}
		},
	}

	identity := resolver.Identity{Owner: "octo", Repo: "demo", Number: 5}
	_, err := svc.Unresolve(identity, ActionOptions{ThreadID: "T7"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot unresolve")
}

func TestUnresolveNoop(t *testing.T) {
	svc := &Service{}
	callCount := 0
	svc.API = &fakeAPI{
		graphqlFunc: func(query string, variables map[string]interface{}, result interface{}) error {
			switch query {
			case threadDetailsQuery:
				callCount++
				payload := map[string]interface{}{
					"node": map[string]interface{}{
						"id":                 "T8",
						"isResolved":         false,
						"viewerCanResolve":   true,
						"viewerCanUnresolve": true,
					},
				}
				return assign(result, payload)
			default:
				return errors.New("unexpected query")
			}
		},
	}

	identity := resolver.Identity{Owner: "octo", Repo: "demo", Number: 5}
	res, err := svc.Unresolve(identity, ActionOptions{ThreadID: "T8"})
	require.NoError(t, err)
	assert.False(t, res.IsResolved)
	assert.Equal(t, "T8", res.ThreadNodeID)
	assert.Equal(t, 1, callCount)
}

func TestUnresolveMutatesThread(t *testing.T) {
	svc := &Service{}
	mutationCalled := false
	svc.API = &fakeAPI{
		graphqlFunc: func(query string, variables map[string]interface{}, result interface{}) error {
			switch query {
			case threadDetailsQuery:
				payload := map[string]interface{}{
					"node": map[string]interface{}{
						"id":                 "T9",
						"isResolved":         true,
						"viewerCanResolve":   true,
						"viewerCanUnresolve": true,
					},
				}
				return assign(result, payload)
			case unresolveThreadMutation:
				mutationCalled = true
				payload := map[string]interface{}{
					"unresolveReviewThread": map[string]interface{}{
						"thread": map[string]interface{}{
							"id":         "T9",
							"isResolved": false,
						},
					},
				}
				return assign(result, payload)
			default:
				return errors.New("unexpected query")
			}
		},
	}

	identity := resolver.Identity{Owner: "octo", Repo: "demo", Number: 5}
	res, err := svc.Unresolve(identity, ActionOptions{ThreadID: "T9"})
	require.NoError(t, err)
	assert.True(t, mutationCalled)
	assert.False(t, res.IsResolved)
	assert.Equal(t, "T9", res.ThreadNodeID)
}

// ─── Tests for --commit (reply-then-resolve) ────────────────────────────────

func TestResolveWithCommitPostsReplyThenResolves(t *testing.T) {
	svc := &Service{}
	var calls []string
	var replyBody string

	svc.API = &fakeAPI{
		graphqlFunc: func(query string, variables map[string]interface{}, result interface{}) error {
			switch query {
			case threadDetailsQuery:
				payload := map[string]interface{}{
					"node": map[string]interface{}{
						"id":                 "T1",
						"isResolved":         false,
						"viewerCanResolve":   true,
						"viewerCanUnresolve": false,
					},
				}
				return assign(result, payload)
			case addThreadReplyMutation:
				calls = append(calls, "reply")
				replyBody = variables["body"].(string)
				return nil
			case resolveThreadMutation:
				calls = append(calls, "resolve")
				payload := map[string]interface{}{
					"resolveReviewThread": map[string]interface{}{
						"thread": map[string]interface{}{
							"id":         "T1",
							"isResolved": true,
						},
					},
				}
				return assign(result, payload)
			default:
				return errors.New("unexpected query")
			}
		},
	}

	_, err := svc.Resolve(resolver.Identity{Owner: "o", Repo: "r", Number: 1}, ActionOptions{
		ThreadID: "T1",
		Commit:   "abc123",
	})
	require.NoError(t, err)
	// Must call reply BEFORE resolve — order matters
	require.Equal(t, []string{"reply", "resolve"}, calls)
	require.Equal(t, "Addressed in [`abc123`](https://github.com/o/r/commit/abc123)", replyBody)
}

func TestResolveWithCommitBailsOnReplyFailure(t *testing.T) {
	svc := &Service{}
	resolveCalled := false

	svc.API = &fakeAPI{
		graphqlFunc: func(query string, variables map[string]interface{}, result interface{}) error {
			switch query {
			case threadDetailsQuery:
				payload := map[string]interface{}{
					"node": map[string]interface{}{
						"id":                 "T1",
						"isResolved":         false,
						"viewerCanResolve":   true,
						"viewerCanUnresolve": false,
					},
				}
				return assign(result, payload)
			case addThreadReplyMutation:
				return errors.New("reply failed: forbidden")
			case resolveThreadMutation:
				resolveCalled = true
				return nil
			default:
				return errors.New("unexpected query")
			}
		},
	}

	_, err := svc.Resolve(resolver.Identity{Owner: "o", Repo: "r", Number: 1}, ActionOptions{
		ThreadID: "T1",
		Commit:   "abc123",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "post commit reply")
	require.False(t, resolveCalled, "resolve must NOT be called when reply fails")
}

func TestResolveWithoutCommitSkipsReply(t *testing.T) {
	svc := &Service{}
	var calls []string

	svc.API = &fakeAPI{
		graphqlFunc: func(query string, variables map[string]interface{}, result interface{}) error {
			switch query {
			case threadDetailsQuery:
				calls = append(calls, "details")
				payload := map[string]interface{}{
					"node": map[string]interface{}{
						"id":                 "T1",
						"isResolved":         false,
						"viewerCanResolve":   true,
						"viewerCanUnresolve": false,
					},
				}
				return assign(result, payload)
			case resolveThreadMutation:
				calls = append(calls, "resolve")
				payload := map[string]interface{}{
					"resolveReviewThread": map[string]interface{}{
						"thread": map[string]interface{}{
							"id":         "T1",
							"isResolved": true,
						},
					},
				}
				return assign(result, payload)
			default:
				return errors.New("unexpected query: " + query[:40])
			}
		},
	}

	_, err := svc.Resolve(resolver.Identity{Owner: "o", Repo: "r", Number: 1}, ActionOptions{
		ThreadID: "T1",
		Commit:   "", // no commit
	})
	require.NoError(t, err)
	// Should be details → resolve, NO reply call
	require.Equal(t, []string{"details", "resolve"}, calls)
}

func assign(dst interface{}, payload interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dst)
}


// ─── F2: ResolveAll tests ─────────────────────────────────────────────────────

func TestResolveAllResolvesUnresolvedThreads(t *testing.T) {
	svc := &Service{}
	var resolvedIDs []string

	svc.API = &fakeAPI{
		restFunc: restStub(t, "octo", "demo", "octo/demo", 7, "PR_bulk", nil),
		graphqlFunc: func(query string, variables map[string]interface{}, result interface{}) error {
			switch query {
			case listThreadsQuery:
				ts := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
				return assign(result, map[string]interface{}{
					"node": map[string]interface{}{
						"reviewThreads": map[string]interface{}{
							"nodes": []map[string]interface{}{
								{
									"id": "TA", "isResolved": false, "isOutdated": false, "path": "a.go",
									"viewerCanResolve": true, "viewerCanUnresolve": false,
									"comments": map[string]interface{}{"nodes": []map[string]interface{}{
										{"viewerDidAuthor": false, "updatedAt": ts, "databaseId": 1},
									}},
								},
								{
									"id": "TB", "isResolved": false, "isOutdated": false, "path": "b.go",
									"viewerCanResolve": true, "viewerCanUnresolve": false,
									"comments": map[string]interface{}{"nodes": []map[string]interface{}{
										{"viewerDidAuthor": false, "updatedAt": ts, "databaseId": 2},
									}},
								},
							},
							"pageInfo": map[string]interface{}{"hasNextPage": false, "endCursor": ""},
						},
					},
				})
			case threadDetailsQuery:
				threadID := variables["id"].(string)
				return assign(result, map[string]interface{}{
					"node": map[string]interface{}{
						"id":                 threadID,
						"isResolved":         false,
						"viewerCanResolve":   true,
						"viewerCanUnresolve": false,
					},
				})
			case resolveThreadMutation:
				threadID := variables["threadId"].(string)
				resolvedIDs = append(resolvedIDs, threadID)
				return assign(result, map[string]interface{}{
					"resolveReviewThread": map[string]interface{}{
						"thread": map[string]interface{}{"id": threadID, "isResolved": true},
					},
				})
			default:
				return errors.New("unexpected query")
			}
		},
	}

	identity := resolver.Identity{Owner: "octo", Repo: "demo", Number: 7}
	results, err := svc.ResolveAll(identity, ResolveAllOptions{Unresolved: true})
	require.NoError(t, err)
	require.Len(t, results, 2)
	assert.ElementsMatch(t, []string{"TA", "TB"}, resolvedIDs)
	for _, r := range results {
		assert.True(t, r.IsResolved)
	}
}

func TestResolveAllContinuesOnError(t *testing.T) {
	svc := &Service{}

	svc.API = &fakeAPI{
		restFunc: restStub(t, "octo", "demo", "octo/demo", 8, "PR_err", nil),
		graphqlFunc: func(query string, variables map[string]interface{}, result interface{}) error {
			switch query {
			case listThreadsQuery:
				ts := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
				return assign(result, map[string]interface{}{
					"node": map[string]interface{}{
						"reviewThreads": map[string]interface{}{
							"nodes": []map[string]interface{}{
								{
									"id": "T_fail", "isResolved": false, "isOutdated": false, "path": "x.go",
									"viewerCanResolve": true, "viewerCanUnresolve": false,
									"comments": map[string]interface{}{"nodes": []map[string]interface{}{
										{"viewerDidAuthor": false, "updatedAt": ts, "databaseId": 10},
									}},
								},
								{
									"id": "T_ok", "isResolved": false, "isOutdated": false, "path": "y.go",
									"viewerCanResolve": true, "viewerCanUnresolve": false,
									"comments": map[string]interface{}{"nodes": []map[string]interface{}{
										{"viewerDidAuthor": false, "updatedAt": ts, "databaseId": 11},
									}},
								},
							},
							"pageInfo": map[string]interface{}{"hasNextPage": false, "endCursor": ""},
						},
					},
				})
			case threadDetailsQuery:
				threadID := variables["id"].(string)
				return assign(result, map[string]interface{}{
					"node": map[string]interface{}{
						"id":                 threadID,
						"isResolved":         false,
						"viewerCanResolve":   true,
						"viewerCanUnresolve": false,
					},
				})
			case resolveThreadMutation:
				threadID := variables["threadId"].(string)
				if threadID == "T_fail" {
					return errors.New("server error")
				}
				return assign(result, map[string]interface{}{
					"resolveReviewThread": map[string]interface{}{
						"thread": map[string]interface{}{"id": threadID, "isResolved": true},
					},
				})
			default:
				return errors.New("unexpected query")
			}
		},
	}

	identity := resolver.Identity{Owner: "octo", Repo: "demo", Number: 8}
	results, err := svc.ResolveAll(identity, ResolveAllOptions{Unresolved: true})
	require.NoError(t, err)
	require.Len(t, results, 2)

	byID := map[string]ActionResult{}
	for _, r := range results {
		byID[r.ThreadNodeID] = r
	}
	assert.False(t, byID["T_fail"].IsResolved)
	assert.True(t, byID["T_ok"].IsResolved)
}

// ─── F3: --since filter test ──────────────────────────────────────────────────

func TestServiceListSinceFiltersOldThreads(t *testing.T) {
	svc := &Service{}

	old := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	recent := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	cutoff := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)

	svc.API = &fakeAPI{
		restFunc: restStub(t, "octo", "demo", "octo/demo", 9, "PR_since", nil),
		graphqlFunc: func(query string, variables map[string]interface{}, result interface{}) error {
			return assign(result, map[string]interface{}{
				"node": map[string]interface{}{
					"reviewThreads": map[string]interface{}{
						"nodes": []map[string]interface{}{
							{
								"id": "T_old", "isResolved": false, "isOutdated": false, "path": "old.go",
								"viewerCanResolve": false, "viewerCanUnresolve": false,
								"comments": map[string]interface{}{"nodes": []map[string]interface{}{
									{"viewerDidAuthor": false, "updatedAt": old, "databaseId": 1},
								}},
							},
							{
								"id": "T_recent", "isResolved": false, "isOutdated": false, "path": "new.go",
								"viewerCanResolve": false, "viewerCanUnresolve": false,
								"comments": map[string]interface{}{"nodes": []map[string]interface{}{
									{"viewerDidAuthor": false, "updatedAt": recent, "databaseId": 2},
								}},
							},
						},
						"pageInfo": map[string]interface{}{"hasNextPage": false, "endCursor": ""},
					},
				},
			})
		},
	}

	identity := resolver.Identity{Owner: "octo", Repo: "demo", Number: 9}
	results, err := svc.List(identity, ListOptions{Since: cutoff})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "T_recent", results[0].ThreadID)
}

func TestResolveIncludesReplyBodyWhenCommitProvided(t *testing.T) {
	api := &fakeAPI{}
	api.graphqlFunc = func(query string, variables map[string]interface{}, result interface{}) error {
		switch {
		case strings.Contains(query, "ThreadDetails"):
			return assign(result, map[string]interface{}{
				"node": map[string]interface{}{
					"id":                 "T_commit",
					"isResolved":         false,
					"viewerCanResolve":   true,
					"viewerCanUnresolve": false,
				},
			})
		case strings.Contains(query, "addPullRequestReviewThreadReply"):
			// reply mutation — result is nil, just succeed
			return nil
		case strings.Contains(query, "resolveReviewThread"):
			return assign(result, map[string]interface{}{
				"resolveReviewThread": map[string]interface{}{
					"thread": map[string]interface{}{
						"id":         "T_commit",
						"isResolved": true,
					},
				},
			})
		default:
			return errors.New("unexpected query")
		}
	}

	svc := NewService(api)
	identity := resolver.Identity{Owner: "octo", Repo: "demo", Number: 1}
	result, err := svc.Resolve(identity, ActionOptions{ThreadID: "T_commit", Commit: "abc1234"})
	require.NoError(t, err)
	assert.Equal(t, "T_commit", result.ThreadNodeID)
	assert.True(t, result.IsResolved)
	assert.Contains(t, result.ReplyBody, "abc1234")
	assert.Contains(t, result.ReplyBody, "octo/demo/commit/abc1234")
}

func TestResolveNoReplyBodyWhenNoCommit(t *testing.T) {
	api := &fakeAPI{}
	api.graphqlFunc = func(query string, variables map[string]interface{}, result interface{}) error {
		switch {
		case strings.Contains(query, "ThreadDetails"):
			return assign(result, map[string]interface{}{
				"node": map[string]interface{}{
					"id":                 "T_no_commit",
					"isResolved":         false,
					"viewerCanResolve":   true,
					"viewerCanUnresolve": false,
				},
			})
		case strings.Contains(query, "resolveReviewThread"):
			return assign(result, map[string]interface{}{
				"resolveReviewThread": map[string]interface{}{
					"thread": map[string]interface{}{
						"id":         "T_no_commit",
						"isResolved": true,
					},
				},
			})
		default:
			return errors.New("unexpected query")
		}
	}

	svc := NewService(api)
	identity := resolver.Identity{Owner: "octo", Repo: "demo", Number: 1}
	result, err := svc.Resolve(identity, ActionOptions{ThreadID: "T_no_commit"})
	require.NoError(t, err)
	assert.Empty(t, result.ReplyBody)
}

// ─── Author substring matching tests ─────────────────────────────────────────

func TestServiceListAuthorSubstringMatch(t *testing.T) {
	svc := &Service{}
	svc.API = &fakeAPI{
		restFunc: restStub(t, "octo", "demo", "octo/demo", 9, "PR_sub", nil),
		graphqlFunc: func(query string, variables map[string]interface{}, result interface{}) error {
			require.Equal(t, listThreadsQuery, query)
			ts := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
			return assign(result, map[string]interface{}{
				"node": map[string]interface{}{
					"reviewThreads": map[string]interface{}{
						"nodes": []map[string]interface{}{
							{
								"id": "T_rabbit", "isResolved": false, "isOutdated": false, "path": "a.go",
								"viewerCanResolve": true, "viewerCanUnresolve": false,
								"comments": map[string]interface{}{"nodes": []map[string]interface{}{
									{"viewerDidAuthor": false, "updatedAt": ts, "databaseId": 1, "body": "nitpick", "author": map[string]interface{}{"login": "coderabbitai"}},
								}},
							},
							{
								"id": "T_codex", "isResolved": false, "isOutdated": false, "path": "b.go",
								"viewerCanResolve": true, "viewerCanUnresolve": false,
								"comments": map[string]interface{}{"nodes": []map[string]interface{}{
									{"viewerDidAuthor": false, "updatedAt": ts, "databaseId": 2, "body": "bug", "author": map[string]interface{}{"login": "chatgpt-codex-connector"}},
								}},
							},
							{
								"id": "T_human", "isResolved": false, "isOutdated": false, "path": "c.go",
								"viewerCanResolve": true, "viewerCanUnresolve": false,
								"comments": map[string]interface{}{"nodes": []map[string]interface{}{
									{"viewerDidAuthor": false, "updatedAt": ts, "databaseId": 3, "body": "looks good", "author": map[string]interface{}{"login": "reviewer42"}},
								}},
							},
						},
						"pageInfo": map[string]interface{}{"hasNextPage": false, "endCursor": ""},
					},
				},
			})
		},
	}

	identity := resolver.Identity{Owner: "octo", Repo: "demo", Number: 9}

	// Substring "codex" should match "chatgpt-codex-connector"
	t.Run("substring matches partial login", func(t *testing.T) {
		results, err := svc.List(identity, ListOptions{Author: "codex"})
		require.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, "T_codex", results[0].ThreadID)
	})

	// Substring "rabbit" should match "coderabbitai"
	t.Run("substring matches coderabbitai", func(t *testing.T) {
		results, err := svc.List(identity, ListOptions{Author: "rabbit"})
		require.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, "T_rabbit", results[0].ThreadID)
	})

	// Substring "CODEX" should match case-insensitively
	t.Run("substring is case insensitive", func(t *testing.T) {
		results, err := svc.List(identity, ListOptions{Author: "CODEX"})
		require.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, "T_codex", results[0].ThreadID)
	})

	// Substring "nobody" should match nothing
	t.Run("non-matching substring returns empty", func(t *testing.T) {
		results, err := svc.List(identity, ListOptions{Author: "nobody"})
		require.NoError(t, err)
		assert.Len(t, results, 0)
	})

	// No author filter returns all
	t.Run("empty author returns all threads", func(t *testing.T) {
		results, err := svc.List(identity, ListOptions{})
		require.NoError(t, err)
		assert.Len(t, results, 3)
	})
}

func TestResolveAllErrorFieldPopulated(t *testing.T) {
	svc := &Service{}
	svc.API = &fakeAPI{
		restFunc: restStub(t, "octo", "demo", "octo/demo", 10, "PR_errfield", nil),
		graphqlFunc: func(query string, variables map[string]interface{}, result interface{}) error {
			ts := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
			switch query {
			case listThreadsQuery:
				return assign(result, map[string]interface{}{
					"node": map[string]interface{}{
						"reviewThreads": map[string]interface{}{
							"nodes": []map[string]interface{}{
								{
									"id": "T_ok", "isResolved": false, "isOutdated": false, "path": "x.go",
									"viewerCanResolve": true, "viewerCanUnresolve": false,
									"comments": map[string]interface{}{"nodes": []map[string]interface{}{
										{"viewerDidAuthor": false, "updatedAt": ts, "databaseId": 1},
									}},
								},
							},
							"pageInfo": map[string]interface{}{"hasNextPage": false, "endCursor": ""},
						},
					},
				})
			case threadDetailsQuery:
				return assign(result, map[string]interface{}{
					"node": map[string]interface{}{
						"id": "T_ok", "isResolved": false,
						"viewerCanResolve": false, "viewerCanUnresolve": false,
					},
				})
			default:
				return errors.New("unexpected query")
			}
		},
	}

	identity := resolver.Identity{Owner: "octo", Repo: "demo", Number: 10}
	results, err := svc.ResolveAll(identity, ResolveAllOptions{Unresolved: true})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.False(t, results[0].IsResolved)
	assert.Contains(t, results[0].Error, "cannot resolve")
}
