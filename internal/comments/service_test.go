package comments

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/Agyn-sandbox/gh-pr-review/internal/ghcli"
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

func assign(result interface{}, payload interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, result)
}

func TestServiceList_WithReviewIDPagination(t *testing.T) {
	api := &fakeAPI{}
	calls := 0
	api.restFunc = func(method, path string, params map[string]string, body interface{}, result interface{}) error {
		calls++
		require.Equal(t, "GET", method)
		switch {
		case path == "repos/octo/demo/pulls/7/reviews/55/comments" && params["page"] == "1":
			payload := make([]map[string]interface{}, 100)
			for i := range payload {
				payload[i] = map[string]interface{}{"id": i + 1}
			}
			return assign(result, payload)
		case path == "repos/octo/demo/pulls/7/reviews/55/comments" && params["page"] == "2":
			payload := []map[string]interface{}{
				{"id": 101, "body": "second"},
			}
			return assign(result, payload)
		default:
			return assign(result, []map[string]interface{}{})
		}
	}

	svc := NewService(api)
	pr := resolver.Identity{Owner: "octo", Repo: "demo", Number: 7, Host: "github.com"}
	comments, err := svc.List(pr, ListOptions{ReviewID: 55})
	require.NoError(t, err)
	assert.Len(t, comments, 101)
	assert.GreaterOrEqual(t, calls, 2)
}

func TestServiceList_LatestReview(t *testing.T) {
	api := &fakeAPI{}
	api.restFunc = func(method, path string, params map[string]string, body interface{}, result interface{}) error {
		switch path {
		case "user":
			return assign(result, map[string]interface{}{"login": "octocat"})
		case "repos/octo/demo/pulls/7/reviews":
			page := params["page"]
			if page == "1" {
				submitted := time.Date(2024, 5, 1, 12, 0, 0, 0, time.UTC)
				payload := []map[string]interface{}{
					{"id": 55, "state": "SUBMITTED", "submitted_at": submitted.Format(time.RFC3339), "user": map[string]interface{}{"login": "octocat"}},
				}
				return assign(result, payload)
			}
			return assign(result, []map[string]interface{}{})
		case "repos/octo/demo/pulls/7/reviews/55/comments":
			return assign(result, []map[string]interface{}{{"id": 1}})
		default:
			return errors.New("unexpected path")
		}
	}

	svc := NewService(api)
	pr := resolver.Identity{Owner: "octo", Repo: "demo", Number: 7, Host: "github.com"}
	comments, err := svc.List(pr, ListOptions{Latest: true})
	require.NoError(t, err)
	assert.Len(t, comments, 1)
}

func TestServiceList_LatestReviewNotFound(t *testing.T) {
	api := &fakeAPI{}
	api.restFunc = func(method, path string, params map[string]string, body interface{}, result interface{}) error {
		switch path {
		case "user":
			return assign(result, map[string]interface{}{"login": "octocat"})
		case "repos/octo/demo/pulls/7/reviews":
			return assign(result, []map[string]interface{}{})
		default:
			return errors.New("unexpected path")
		}
	}

	svc := NewService(api)
	pr := resolver.Identity{Owner: "octo", Repo: "demo", Number: 7, Host: "github.com"}
	_, err := svc.List(pr, ListOptions{Latest: true})
	require.Error(t, err)
}

func TestServiceReply_AutoSubmitPending(t *testing.T) {
	api := &fakeAPI{}
	var submitted []int64
	attempt := 0
	api.restFunc = func(method, path string, params map[string]string, body interface{}, result interface{}) error {
		switch {
		case path == "user":
			return assign(result, map[string]interface{}{"login": "octocat"})
		case path == "repos/octo/demo/pulls/7/reviews":
			payload := []map[string]interface{}{
				{"id": 99, "state": "PENDING", "user": map[string]interface{}{"login": "octocat"}},
			}
			return assign(result, payload)
		case path == "repos/octo/demo/pulls/7/reviews/99/events" && method == "POST":
			submitted = append(submitted, 99)
			return nil
		case path == "repos/octo/demo/pulls/7/comments/5/replies" && method == "POST":
			attempt++
			if attempt == 1 {
				return &ghcli.APIError{StatusCode: 422, Message: "pending review"}
			}
			return assign(result, map[string]interface{}{"id": 123, "body": "ok"})
		default:
			return errors.New("unexpected request: " + path)
		}
	}

	svc := NewService(api)
	pr := resolver.Identity{Owner: "octo", Repo: "demo", Number: 7, Host: "github.com"}
	reply, err := svc.Reply(pr, ReplyOptions{CommentID: 5, Body: "ack"})
	require.NoError(t, err)
	assert.Contains(t, string(reply), "\"id\":123")
	assert.Equal(t, []int64{99}, submitted)
}

func TestServiceReply_PendingMissing(t *testing.T) {
	api := &fakeAPI{}
	api.restFunc = func(method, path string, params map[string]string, body interface{}, result interface{}) error {
		switch path {
		case "repos/octo/demo/pulls/7/comments/5/replies":
			return &ghcli.APIError{StatusCode: 422, Message: "pending review"}
		case "user":
			return assign(result, map[string]interface{}{"login": "octocat"})
		case "repos/octo/demo/pulls/7/reviews":
			return assign(result, []map[string]interface{}{})
		default:
			return errors.New("unexpected path")
		}
	}

	svc := NewService(api)
	pr := resolver.Identity{Owner: "octo", Repo: "demo", Number: 7, Host: "github.com"}
	_, err := svc.Reply(pr, ReplyOptions{CommentID: 5, Body: "ack"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "pending reviews owned by octocat")
}
