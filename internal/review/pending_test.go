package review

import (
	"errors"
	"fmt"
	"testing"

	"github.com/Agyn-sandbox/gh-pr-review/internal/resolver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLatestPendingDefaultsToAuthenticatedReviewer(t *testing.T) {
	api := &fakeAPI{}
	api.restFunc = func(method, path string, params map[string]string, body interface{}, result interface{}) error {
		switch path {
		case "user":
			return assign(result, map[string]interface{}{"login": "casey"})
		case "repos/octo/demo/pulls/7/reviews":
			require.Equal(t, "100", params["per_page"])
			switch params["page"] {
			case "1":
				payload := []map[string]interface{}{
					{
						"id":                 5,
						"state":              "PENDING",
						"author_association": "MEMBER",
						"html_url":           "https://github.com/octo/demo/pull/7#review-5",
						"user": map[string]interface{}{
							"login": "casey",
							"id":    101,
						},
					},
					{
						"id":                 7,
						"state":              "PENDING",
						"author_association": "MEMBER",
						"html_url":           "https://github.com/octo/demo/pull/7#review-7",
						"user": map[string]interface{}{
							"login": "casey",
							"id":    101,
						},
					},
					{
						"id":    9,
						"state": "PENDING",
						"user":  map[string]interface{}{"login": "other"},
					},
				}
				return assign(result, payload)
			default:
				return assign(result, []map[string]interface{}{})
			}
		default:
			return errors.New("unexpected path")
		}
	}

	svc := NewService(api)
	pr := resolver.Identity{Owner: "octo", Repo: "demo", Number: 7, Host: "github.com"}
	summary, err := svc.LatestPending(pr, PendingOptions{})
	require.NoError(t, err)
	require.NotNil(t, summary)
	assert.Equal(t, int64(7), summary.ID)
	assert.Equal(t, "PENDING", summary.State)
	require.NotNil(t, summary.User)
	assert.Equal(t, "casey", summary.User.Login)
	assert.Equal(t, int64(101), summary.User.ID)
	assert.Equal(t, "https://github.com/octo/demo/pull/7#review-7", summary.HTMLURL)
	assert.Equal(t, "MEMBER", summary.AuthorAssociation)
}

func TestLatestPendingWithReviewerOverride(t *testing.T) {
	api := &fakeAPI{}
	api.restFunc = func(method, path string, params map[string]string, body interface{}, result interface{}) error {
		if path != "repos/octo/demo/pulls/7/reviews" {
			return errors.New("unexpected path")
		}
		require.Equal(t, "50", params["per_page"])
		require.Equal(t, "3", params["page"])
		payload := []map[string]interface{}{
			{
				"id":                 42,
				"state":              "PENDING",
				"author_association": "CONTRIBUTOR",
				"html_url":           "https://example.com/review/42",
				"user": map[string]interface{}{
					"login": "octocat",
					"id":    202,
				},
			},
		}
		return assign(result, payload)
	}

	svc := NewService(api)
	pr := resolver.Identity{Owner: "octo", Repo: "demo", Number: 7, Host: "github.com"}
	summary, err := svc.LatestPending(pr, PendingOptions{Reviewer: "octocat", PerPage: 50, Page: 3})
	require.NoError(t, err)
	require.NotNil(t, summary)
	assert.Equal(t, int64(42), summary.ID)
	require.NotNil(t, summary.User)
	assert.Equal(t, "octocat", summary.User.Login)
	assert.Equal(t, int64(202), summary.User.ID)
	assert.Equal(t, "https://example.com/review/42", summary.HTMLURL)
}

func TestLatestPendingNoMatches(t *testing.T) {
	api := &fakeAPI{}
	api.restFunc = func(method, path string, params map[string]string, body interface{}, result interface{}) error {
		switch path {
		case "user":
			return assign(result, map[string]interface{}{"login": "casey"})
		case "repos/octo/demo/pulls/7/reviews":
			return assign(result, []map[string]interface{}{})
		default:
			return fmt.Errorf("unexpected path %s", path)
		}
	}

	svc := NewService(api)
	pr := resolver.Identity{Owner: "octo", Repo: "demo", Number: 7, Host: "github.com"}
	_, err := svc.LatestPending(pr, PendingOptions{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no pending reviews")
}
