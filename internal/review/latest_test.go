package review

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/agynio/gh-pr-review/internal/resolver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLatestSubmittedDefaultsToAuthenticatedReviewer(t *testing.T) {
	api := &fakeAPI{}
	api.restFunc = func(method, path string, params map[string]string, body interface{}, result interface{}) error {
		switch path {
		case "user":
			payload := map[string]interface{}{"login": "casey"}
			return assign(result, payload)
		case "repos/octo/demo/pulls/7/reviews":
			page := params["page"]
			perPage := params["per_page"]
			require.Equal(t, "100", perPage)
			switch page {
			case "1":
				payload := make([]map[string]interface{}, 100)
				for i := range payload {
					payload[i] = map[string]interface{}{
						"id":           i + 1,
						"state":        "COMMENTED",
						"submitted_at": "2024-05-01T12:00:00Z",
						"user":         map[string]interface{}{"login": "other"},
					}
				}
				return assign(result, payload)
			case "2":
				payload := []map[string]interface{}{
					{
						"id":                 200,
						"state":              "COMMENTED",
						"submitted_at":       "2024-06-10T08:00:00Z",
						"author_association": "MEMBER",
						"html_url":           "https://github.com/octo/demo/pull/7#review-200",
						"user": map[string]interface{}{
							"login": "casey",
							"id":    101,
						},
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
	summary, err := svc.LatestSubmitted(pr, LatestOptions{})
	require.NoError(t, err)
	require.NotNil(t, summary)
	assert.Equal(t, int64(200), summary.ID)
	assert.Equal(t, "COMMENTED", summary.State)
	require.NotNil(t, summary.SubmittedAt)
	parsed, parseErr := time.Parse(time.RFC3339, *summary.SubmittedAt)
	require.NoError(t, parseErr)
	assert.Equal(t, time.Date(2024, 6, 10, 8, 0, 0, 0, time.UTC), parsed)
	require.NotNil(t, summary.User)
	assert.Equal(t, "casey", summary.User.Login)
	assert.Equal(t, int64(101), summary.User.ID)
	assert.Equal(t, "https://github.com/octo/demo/pull/7#review-200", summary.HTMLURL)
	assert.Equal(t, "MEMBER", summary.AuthorAssociation)
}

func TestLatestSubmittedWithReviewerOverride(t *testing.T) {
	api := &fakeAPI{}
	api.restFunc = func(method, path string, params map[string]string, body interface{}, result interface{}) error {
		if path != "repos/octo/demo/pulls/7/reviews" {
			return errors.New("unexpected path")
		}
		payload := []map[string]interface{}{
			{
				"id":           20,
				"state":        "APPROVED",
				"submitted_at": "2024-07-01T09:30:00Z",
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
	summary, err := svc.LatestSubmitted(pr, LatestOptions{Reviewer: "octocat", PerPage: 50, Page: 1})
	require.NoError(t, err)
	assert.Equal(t, int64(20), summary.ID)
	require.NotNil(t, summary.User)
	assert.Equal(t, "octocat", summary.User.Login)
	assert.Equal(t, int64(202), summary.User.ID)
}

func TestLatestSubmittedNoMatches(t *testing.T) {
	api := &fakeAPI{}
	api.restFunc = func(method, path string, params map[string]string, body interface{}, result interface{}) error {
		if path == "user" {
			payload := map[string]interface{}{"login": "casey"}
			return assign(result, payload)
		}
		if path == "repos/octo/demo/pulls/7/reviews" {
			return assign(result, []map[string]interface{}{})
		}
		return fmt.Errorf("unexpected path %s", path)
	}

	svc := NewService(api)
	pr := resolver.Identity{Owner: "octo", Repo: "demo", Number: 7, Host: "github.com"}
	_, err := svc.LatestSubmitted(pr, LatestOptions{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no submitted reviews")
}
