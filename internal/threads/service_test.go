package threads

import (
	"encoding/json"
	"errors"
	"strconv"
	"testing"
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

func assign(dst interface{}, payload interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dst)
}
