package review

import (
	"strings"
	"testing"

	"github.com/agynio/gh-pr-review/internal/resolver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func int64Ptr(v int64) *int64 { return &v }

type testReviewNode struct {
	ID                string `json:"id"`
	DatabaseID        *int64 `json:"databaseId"`
	State             string `json:"state"`
	AuthorAssociation string `json:"authorAssociation"`
	URL               string `json:"url"`
	UpdatedAt         string `json:"updatedAt"`
	CreatedAt         string `json:"createdAt"`
	Author            *struct {
		Login      string `json:"login"`
		DatabaseID *int64 `json:"databaseId"`
	} `json:"author"`
}

func TestLatestPendingDefaultsToAuthenticatedReviewer(t *testing.T) {
	api := &fakeAPI{}
	viewerCalls := 0
	pendingCalls := 0
	api.graphqlFunc = func(query string, variables map[string]interface{}, result interface{}) error {
		if strings.Contains(query, "ViewerLogin") {
			viewerCalls++
			payload := struct {
				Data struct {
					Viewer struct {
						Login string `json:"login"`
					} `json:"viewer"`
				} `json:"data"`
			}{}
			payload.Data.Viewer.Login = "casey"
			return assign(result, payload)
		}

		pendingCalls++
		require.Equal(t, "octo", variables["owner"])
		require.Equal(t, "demo", variables["name"])
		require.EqualValues(t, 7, variables["number"])
		require.EqualValues(t, 100, variables["pageSize"])
		_, hasCursor := variables["cursor"]
		require.False(t, hasCursor)

		nodes := []testReviewNode{
			{
				ID:                "R_pending_5",
				DatabaseID:        int64Ptr(5),
				State:             "PENDING",
				AuthorAssociation: "MEMBER",
				URL:               "https://github.com/octo/demo/pull/7#review-5",
				UpdatedAt:         "",
				CreatedAt:         "2024-06-01T10:00:00Z",
				Author: &struct {
					Login      string `json:"login"`
					DatabaseID *int64 `json:"databaseId"`
				}{Login: "casey", DatabaseID: int64Ptr(101)},
			},
			{
				ID:                "R_pending_7",
				DatabaseID:        int64Ptr(7),
				State:             "PENDING",
				AuthorAssociation: "MEMBER",
				URL:               "https://github.com/octo/demo/pull/7#review-7",
				UpdatedAt:         "2024-06-01T12:00:00Z",
				CreatedAt:         "2024-06-01T11:00:00Z",
				Author: &struct {
					Login      string `json:"login"`
					DatabaseID *int64 `json:"databaseId"`
				}{Login: "casey", DatabaseID: int64Ptr(101)},
			},
		}

		payload := struct {
			Data struct {
				Repository struct {
					PullRequest struct {
						Reviews struct {
							Nodes    []testReviewNode `json:"nodes"`
							PageInfo struct {
								HasNextPage bool   `json:"hasNextPage"`
								EndCursor   string `json:"endCursor"`
							} `json:"pageInfo"`
						} `json:"reviews"`
					} `json:"pullRequest"`
				} `json:"repository"`
			} `json:"data"`
		}{}
		payload.Data.Repository.PullRequest.Reviews.Nodes = nodes
		payload.Data.Repository.PullRequest.Reviews.PageInfo.HasNextPage = false
		payload.Data.Repository.PullRequest.Reviews.PageInfo.EndCursor = ""

		return assign(result, payload)
	}

	svc := NewService(api)
	pr := resolver.Identity{Owner: "octo", Repo: "demo", Number: 7, Host: "github.com"}
	summary, err := svc.LatestPending(pr, PendingOptions{})
	require.NoError(t, err)
	require.NotNil(t, summary)
	assert.Equal(t, "R_pending_7", summary.ID)
	assert.Equal(t, int64(7), summary.DatabaseID)
	assert.Equal(t, "PENDING", summary.State)
	require.NotNil(t, summary.User)
	assert.Equal(t, "casey", summary.User.Login)
	assert.Equal(t, int64(101), summary.User.ID)
	assert.Equal(t, "https://github.com/octo/demo/pull/7#review-7", summary.HTMLURL)
	assert.Equal(t, "MEMBER", summary.AuthorAssociation)
	assert.Equal(t, 1, viewerCalls)
	assert.Equal(t, 1, pendingCalls)
}

func TestLatestPendingWithReviewerOverride(t *testing.T) {
	api := &fakeAPI{}
	page := 0
	api.graphqlFunc = func(query string, variables map[string]interface{}, result interface{}) error {
		if strings.Contains(query, "ViewerLogin") {
			t.Fatalf("unexpected viewer query")
		}
		page++
		require.Equal(t, "octo", variables["owner"])
		require.Equal(t, "demo", variables["name"])
		require.EqualValues(t, 7, variables["number"])
		require.EqualValues(t, 50, variables["pageSize"])

		payload := struct {
			Data struct {
				Viewer struct {
					Login      string `json:"login"`
					DatabaseID int64  `json:"databaseId"`
				} `json:"viewer"`
				Repository struct {
					PullRequest struct {
						Reviews struct {
							Nodes    []testReviewNode `json:"nodes"`
							PageInfo struct {
								HasNextPage bool   `json:"hasNextPage"`
								EndCursor   string `json:"endCursor"`
							} `json:"pageInfo"`
						} `json:"reviews"`
					} `json:"pullRequest"`
				} `json:"repository"`
			} `json:"data"`
		}{}
		payload.Data.Viewer.Login = "casey"
		payload.Data.Viewer.DatabaseID = 101

		if page == 1 {
			_, hasCursor := variables["cursor"]
			require.False(t, hasCursor)
			nodes := []testReviewNode{
				{
					ID:                "R_pending_other",
					DatabaseID:        int64Ptr(9),
					State:             "PENDING",
					AuthorAssociation: "MEMBER",
					URL:               "https://example.com/review/9",
					UpdatedAt:         "2024-06-01T08:00:00Z",
					CreatedAt:         "2024-06-01T07:00:00Z",
					Author: &struct {
						Login      string `json:"login"`
						DatabaseID *int64 `json:"databaseId"`
					}{Login: "someone", DatabaseID: int64Ptr(404)},
				},
			}
			payload.Data.Repository.PullRequest.Reviews.Nodes = nodes
			payload.Data.Repository.PullRequest.Reviews.PageInfo.HasNextPage = true
			payload.Data.Repository.PullRequest.Reviews.PageInfo.EndCursor = "CURSOR1"
		} else {
			cursorValue, hasCursor := variables["cursor"]
			require.True(t, hasCursor)
			require.Equal(t, "CURSOR1", cursorValue)
			nodes := []testReviewNode{
				{
					ID:                "R_pending_42",
					DatabaseID:        int64Ptr(42),
					State:             "PENDING",
					AuthorAssociation: "CONTRIBUTOR",
					URL:               "https://example.com/review/42",
					UpdatedAt:         "2024-06-02T12:00:00Z",
					CreatedAt:         "2024-06-02T11:30:00Z",
					Author: &struct {
						Login      string `json:"login"`
						DatabaseID *int64 `json:"databaseId"`
					}{Login: "octocat", DatabaseID: int64Ptr(202)},
				},
			}
			payload.Data.Repository.PullRequest.Reviews.Nodes = nodes
			payload.Data.Repository.PullRequest.Reviews.PageInfo.HasNextPage = false
			payload.Data.Repository.PullRequest.Reviews.PageInfo.EndCursor = ""
		}

		return assign(result, payload)
	}

	svc := NewService(api)
	pr := resolver.Identity{Owner: "octo", Repo: "demo", Number: 7, Host: "github.com"}
	summary, err := svc.LatestPending(pr, PendingOptions{Reviewer: "octocat", PerPage: 50})
	require.NoError(t, err)
	require.NotNil(t, summary)
	assert.Equal(t, "R_pending_42", summary.ID)
	assert.Equal(t, int64(42), summary.DatabaseID)
	require.NotNil(t, summary.User)
	assert.Equal(t, "octocat", summary.User.Login)
	assert.Equal(t, int64(202), summary.User.ID)
	assert.Equal(t, "https://example.com/review/42", summary.HTMLURL)
	assert.Equal(t, "CONTRIBUTOR", summary.AuthorAssociation)
	assert.Equal(t, 2, page)
}

func TestLatestPendingAllowsMissingUserID(t *testing.T) {
	api := &fakeAPI{}
	api.graphqlFunc = func(query string, variables map[string]interface{}, result interface{}) error {
		if strings.Contains(query, "ViewerLogin") {
			payload := struct {
				Data struct {
					Viewer struct {
						Login string `json:"login"`
					} `json:"viewer"`
				} `json:"data"`
			}{}
			payload.Data.Viewer.Login = "casey"
			return assign(result, payload)
		}

		nodes := []testReviewNode{
			{
				ID:                "R_pending_viewer",
				DatabaseID:        int64Ptr(11),
				State:             "PENDING",
				AuthorAssociation: "MEMBER",
				URL:               "https://example.com/review/11",
				UpdatedAt:         "2024-06-03T10:00:00Z",
				CreatedAt:         "2024-06-03T09:30:00Z",
				Author: &struct {
					Login      string `json:"login"`
					DatabaseID *int64 `json:"databaseId"`
				}{Login: "casey", DatabaseID: nil},
			},
		}

		payload := struct {
			Data struct {
				Repository struct {
					PullRequest struct {
						Reviews struct {
							Nodes    []testReviewNode `json:"nodes"`
							PageInfo struct {
								HasNextPage bool   `json:"hasNextPage"`
								EndCursor   string `json:"endCursor"`
							} `json:"pageInfo"`
						} `json:"reviews"`
					} `json:"pullRequest"`
				} `json:"repository"`
			} `json:"data"`
		}{}
		payload.Data.Repository.PullRequest.Reviews.Nodes = nodes
		payload.Data.Repository.PullRequest.Reviews.PageInfo.HasNextPage = false
		payload.Data.Repository.PullRequest.Reviews.PageInfo.EndCursor = ""
		return assign(result, payload)
	}

	svc := NewService(api)
	pr := resolver.Identity{Owner: "octo", Repo: "demo", Number: 7, Host: "github.com"}
	summary, err := svc.LatestPending(pr, PendingOptions{})
	require.NoError(t, err)
	require.NotNil(t, summary)
	require.NotNil(t, summary.User)
	assert.Equal(t, "casey", summary.User.Login)
	assert.Zero(t, summary.User.ID)
}

func TestLatestPendingNoMatches(t *testing.T) {
	api := &fakeAPI{}
	api.graphqlFunc = func(query string, variables map[string]interface{}, result interface{}) error {
		if strings.Contains(query, "ViewerLogin") {
			payload := struct {
				Data struct {
					Viewer struct {
						Login string `json:"login"`
					} `json:"viewer"`
				} `json:"data"`
			}{}
			payload.Data.Viewer.Login = "casey"
			return assign(result, payload)
		}

		payload := struct {
			Data struct {
				Repository struct {
					PullRequest struct {
						Reviews struct {
							Nodes    []testReviewNode `json:"nodes"`
							PageInfo struct {
								HasNextPage bool   `json:"hasNextPage"`
								EndCursor   string `json:"endCursor"`
							} `json:"pageInfo"`
						} `json:"reviews"`
					} `json:"pullRequest"`
				} `json:"repository"`
			} `json:"data"`
		}{}
		payload.Data.Repository.PullRequest.Reviews.Nodes = []testReviewNode{}
		payload.Data.Repository.PullRequest.Reviews.PageInfo.HasNextPage = false
		payload.Data.Repository.PullRequest.Reviews.PageInfo.EndCursor = ""
		return assign(result, payload)
	}

	svc := NewService(api)
	pr := resolver.Identity{Owner: "octo", Repo: "demo", Number: 7, Host: "github.com"}
	_, err := svc.LatestPending(pr, PendingOptions{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no pending reviews for casey")
}
