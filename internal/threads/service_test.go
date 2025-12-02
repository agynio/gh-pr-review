package threads

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

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

func TestServiceListFiltersAndSort(t *testing.T) {
	svc := &Service{}
	svc.API = &fakeAPI{
		graphqlFunc: func(query string, variables map[string]interface{}, result interface{}) error {
			require.Equal(t, listThreadsQuery, query)

			ts1 := time.Date(2025, 12, 1, 10, 0, 0, 0, time.UTC)
			line := 42
			payload := threadsQueryResponse{
				Repository: &struct {
					PullRequest *struct {
						ReviewThreads *struct {
							Nodes    []threadNode `json:"nodes"`
							PageInfo *struct {
								HasNextPage bool   `json:"hasNextPage"`
								EndCursor   string `json:"endCursor"`
							} `json:"pageInfo"`
						} `json:"reviewThreads"`
					} `json:"pullRequest"`
				}{
					PullRequest: &struct {
						ReviewThreads *struct {
							Nodes    []threadNode `json:"nodes"`
							PageInfo *struct {
								HasNextPage bool   `json:"hasNextPage"`
								EndCursor   string `json:"endCursor"`
							} `json:"pageInfo"`
						} `json:"reviewThreads"`
					}{
						ReviewThreads: &struct {
							Nodes    []threadNode `json:"nodes"`
							PageInfo *struct {
								HasNextPage bool   `json:"hasNextPage"`
								EndCursor   string `json:"endCursor"`
							} `json:"pageInfo"`
						}{
							Nodes: []threadNode{
								{
									ID:               "T1",
									IsResolved:       false,
									IsOutdated:       false,
									Path:             "internal/file.go",
									Line:             &line,
									ViewerCanResolve: false,
									Comments: struct {
										Nodes []struct {
											ViewerDidAuthor bool      `json:"viewerDidAuthor"`
											UpdatedAt       time.Time `json:"updatedAt"`
										} `json:"nodes"`
									}{
										Nodes: []struct {
											ViewerDidAuthor bool      `json:"viewerDidAuthor"`
											UpdatedAt       time.Time `json:"updatedAt"`
										}{
											{ViewerDidAuthor: true, UpdatedAt: ts1},
										},
									},
								},
								{
									ID:               "T2",
									IsResolved:       true,
									IsOutdated:       false,
									Path:             "internal/ignore.go",
									ViewerCanResolve: true,
									Comments: struct {
										Nodes []struct {
											ViewerDidAuthor bool      `json:"viewerDidAuthor"`
											UpdatedAt       time.Time `json:"updatedAt"`
										} `json:"nodes"`
									}{},
								},
							},
							PageInfo: &struct {
								HasNextPage bool   `json:"hasNextPage"`
								EndCursor   string `json:"endCursor"`
							}{HasNextPage: false},
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

func TestResolveRequiresPermission(t *testing.T) {
	svc := &Service{}
	svc.API = &fakeAPI{
		graphqlFunc: func(query string, variables map[string]interface{}, result interface{}) error {
			switch query {
			case threadDetailsQuery:
				payload := struct {
					Node *threadDetails `json:"node"`
				}{Node: &threadDetails{ID: "T1", IsResolved: false, ViewerCanResolve: false, ViewerCanUnresolve: true}}
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
	calls := 0
	svc.API = &fakeAPI{
		graphqlFunc: func(query string, variables map[string]interface{}, result interface{}) error {
			if query == threadDetailsQuery {
				calls++
				payload := struct {
					Node *threadDetails `json:"node"`
				}{Node: &threadDetails{ID: "T2", IsResolved: true, ViewerCanResolve: true, ViewerCanUnresolve: true}}
				return assign(result, payload)
			}
			return errors.New("unexpected query")
		},
	}

	identity := resolver.Identity{Owner: "octo", Repo: "demo", Number: 5}
	res, err := svc.Resolve(identity, ActionOptions{ThreadID: "T2"})
	require.NoError(t, err)
	assert.False(t, res.Changed)
	assert.True(t, res.IsResolved)
	assert.Equal(t, "T2", res.ThreadID)
	assert.Equal(t, 1, calls)
}

func TestResolveViaCommentID(t *testing.T) {
	svc := &Service{}
	mutationCalled := false
	svc.API = &fakeAPI{
		restFunc: func(method, path string, params map[string]string, body interface{}, result interface{}) error {
			assert.Equal(t, "GET", method)
			assert.Equal(t, "repos/octo/demo/pulls/comments/9", path)
			payload := map[string]interface{}{"node_id": "C_node"}
			return assign(result, payload)
		},
		graphqlFunc: func(query string, variables map[string]interface{}, result interface{}) error {
			switch query {
			case commentThreadQuery:
				payload := map[string]interface{}{
					"node": map[string]interface{}{
						"pullRequestReviewThread": map[string]interface{}{"id": "T3"},
					},
				}
				return assign(result, payload)
			case threadDetailsQuery:
				payload := struct {
					Node *threadDetails `json:"node"`
				}{Node: &threadDetails{ID: "T3", IsResolved: false, ViewerCanResolve: true, ViewerCanUnresolve: true}}
				return assign(result, payload)
			case resolveThreadMutation:
				mutationCalled = true
				payload := map[string]interface{}{
					"resolveReviewThread": map[string]interface{}{
						"thread": map[string]interface{}{"id": "T3", "isResolved": true},
					},
				}
				return assign(result, payload)
			default:
				return errors.New("unexpected query")
			}
		},
	}

	identity := resolver.Identity{Owner: "octo", Repo: "demo", Number: 5}
	res, err := svc.Resolve(identity, ActionOptions{CommentID: 9})
	require.NoError(t, err)
	assert.True(t, res.Changed)
	assert.True(t, res.IsResolved)
	assert.Equal(t, "T3", res.ThreadID)
	assert.True(t, mutationCalled)
}

func TestUnresolveRequiresPermission(t *testing.T) {
	svc := &Service{}
	svc.API = &fakeAPI{
		graphqlFunc: func(query string, variables map[string]interface{}, result interface{}) error {
			if query == threadDetailsQuery {
				payload := struct {
					Node *threadDetails `json:"node"`
				}{Node: &threadDetails{ID: "T4", IsResolved: true, ViewerCanResolve: true, ViewerCanUnresolve: false}}
				return assign(result, payload)
			}
			return errors.New("unexpected query")
		},
	}

	identity := resolver.Identity{Owner: "octo", Repo: "demo", Number: 5}
	_, err := svc.Unresolve(identity, ActionOptions{ThreadID: "T4"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot unresolve")
}

func TestUnresolveNoop(t *testing.T) {
	svc := &Service{}
	calls := 0
	svc.API = &fakeAPI{
		graphqlFunc: func(query string, variables map[string]interface{}, result interface{}) error {
			if query == threadDetailsQuery {
				calls++
				payload := struct {
					Node *threadDetails `json:"node"`
				}{Node: &threadDetails{ID: "T5", IsResolved: false, ViewerCanResolve: true, ViewerCanUnresolve: true}}
				return assign(result, payload)
			}
			return errors.New("unexpected query")
		},
	}

	identity := resolver.Identity{Owner: "octo", Repo: "demo", Number: 5}
	res, err := svc.Unresolve(identity, ActionOptions{ThreadID: "T5"})
	require.NoError(t, err)
	assert.False(t, res.Changed)
	assert.False(t, res.IsResolved)
	assert.Equal(t, "T5", res.ThreadID)
	assert.Equal(t, 1, calls)
}

func TestUnresolveViaCommentID(t *testing.T) {
	svc := &Service{}
	mutationCalled := false
	svc.API = &fakeAPI{
		restFunc: func(method, path string, params map[string]string, body interface{}, result interface{}) error {
			assert.Equal(t, "GET", method)
			assert.Equal(t, "repos/octo/demo/pulls/comments/11", path)
			payload := map[string]interface{}{"node_id": "C_node"}
			return assign(result, payload)
		},
		graphqlFunc: func(query string, variables map[string]interface{}, result interface{}) error {
			switch query {
			case commentThreadQuery:
				payload := map[string]interface{}{
					"node": map[string]interface{}{
						"pullRequestReviewThread": map[string]interface{}{"id": "T6"},
					},
				}
				return assign(result, payload)
			case threadDetailsQuery:
				payload := struct {
					Node *threadDetails `json:"node"`
				}{Node: &threadDetails{ID: "T6", IsResolved: true, ViewerCanResolve: true, ViewerCanUnresolve: true}}
				return assign(result, payload)
			case unresolveThreadMutation:
				mutationCalled = true
				payload := map[string]interface{}{
					"unresolveReviewThread": map[string]interface{}{
						"thread": map[string]interface{}{"id": "T6", "isResolved": false},
					},
				}
				return assign(result, payload)
			default:
				return errors.New("unexpected query")
			}
		},
	}

	identity := resolver.Identity{Owner: "octo", Repo: "demo", Number: 5}
	res, err := svc.Unresolve(identity, ActionOptions{CommentID: 11})
	require.NoError(t, err)
	assert.True(t, res.Changed)
	assert.False(t, res.IsResolved)
	assert.Equal(t, "T6", res.ThreadID)
	assert.True(t, mutationCalled)
}

func assign(target interface{}, payload interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}
