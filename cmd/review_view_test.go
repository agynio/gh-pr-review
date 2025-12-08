package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"testing"

	_ "embed"

	"github.com/agynio/gh-pr-review/internal/ghcli"
)

//go:embed testdata/report_response.json
var viewResponse []byte

func TestReviewViewCommandFiltersOutput(t *testing.T) {
	originalFactory := apiClientFactory
	defer func() { apiClientFactory = originalFactory }()

	fake := &fakeViewAPI{payload: viewResponse, t: t}
	apiClientFactory = func(host string) ghcli.API {
		if host == "" {
			t.Fatalf("expected host to be resolved, got empty")
		}
		return fake
	}

	root := newRootCommand()
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(io.Discard)
	root.SetArgs([]string{"review", "view", "--repo", "agyn/repo", "--reviewer", "alice", "--states", "APPROVED,COMMENTED", "--not_outdated", "--tail", "1", "51"})

	if err := root.Execute(); err != nil {
		t.Fatalf("execute command: %v", err)
	}

	var payload struct {
		Reviews []struct {
			ID       string  `json:"id"`
			Body     *string `json:"body"`
			Comments []struct {
				ThreadID       string  `json:"thread_id"`
				CommentNodeID  *string `json:"comment_node_id"`
				ThreadComments []struct {
					Body          string  `json:"body"`
					CommentNodeID *string `json:"comment_node_id"`
				} `json:"thread_comments"`
			} `json:"comments"`
		} `json:"reviews"`
	}
	if err := json.Unmarshal(buf.Bytes(), &payload); err != nil {
		t.Fatalf("parse json: %v", err)
	}
	if len(payload.Reviews) != 1 {
		t.Fatalf("expected 1 review in filtered output, got %d", len(payload.Reviews))
	}
	review := payload.Reviews[0]
	if review.ID != "R1" {
		t.Fatalf("expected review R1, got %s", review.ID)
	}
	if review.Body == nil {
		t.Fatalf("expected review body to be present for R1")
	}
	if len(review.Comments) != 1 {
		t.Fatalf("expected 1 comment for R1, got %d", len(review.Comments))
	}
	comment := review.Comments[0]
	if comment.ThreadID == "" {
		t.Fatal("expected thread_id to be present")
	}
	if comment.CommentNodeID != nil {
		t.Fatalf("expected comment_node_id omitted by default, got %v", *comment.CommentNodeID)
	}
	if len(comment.ThreadComments) != 1 {
		t.Fatalf("expected 1 reply after tail filter, got %d", len(comment.ThreadComments))
	}
	if comment.ThreadComments[0].Body != "Reply beta" {
		t.Fatalf("expected last reply body 'Reply beta', got %s", comment.ThreadComments[0].Body)
	}
	if comment.ThreadComments[0].CommentNodeID != nil {
		t.Fatalf("expected reply comment_node_id omitted by default, got %v", *comment.ThreadComments[0].CommentNodeID)
	}

	rawStates, ok := fake.variables["states"].([]string)
	if !ok || len(rawStates) != 2 {
		t.Fatalf("expected states variable propagated, got %#v", fake.variables["states"])
	}
}

func TestReviewViewCommandInvalidState(t *testing.T) {
	root := newRootCommand()
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)
	root.SetArgs([]string{"review", "view", "--repo", "agyn/repo", "--states", "unknown", "51"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for invalid state")
	}
	if !strings.Contains(err.Error(), "invalid review state") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReviewViewCommandIncludesCommentNodeID(t *testing.T) {
	originalFactory := apiClientFactory
	defer func() { apiClientFactory = originalFactory }()

	fake := &fakeViewAPI{payload: viewResponse, t: t}
	apiClientFactory = func(host string) ghcli.API { return fake }

	root := newRootCommand()
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(io.Discard)
	root.SetArgs([]string{"review", "view", "--repo", "agyn/repo", "--include-comment-node-id", "51"})

	if err := root.Execute(); err != nil {
		t.Fatalf("execute command: %v", err)
	}

	var payload struct {
		Reviews []struct {
			Comments []struct {
				CommentNodeID  *string `json:"comment_node_id"`
				ThreadComments []struct {
					CommentNodeID *string `json:"comment_node_id"`
				} `json:"thread_comments"`
			} `json:"comments"`
		} `json:"reviews"`
	}
	if err := json.Unmarshal(buf.Bytes(), &payload); err != nil {
		t.Fatalf("parse json: %v", err)
	}
	if len(payload.Reviews) == 0 || len(payload.Reviews[0].Comments) == 0 {
		t.Fatal("expected comments in report output")
	}
	comment := payload.Reviews[0].Comments[0]
	if comment.CommentNodeID == nil || *comment.CommentNodeID == "" {
		t.Fatalf("expected comment_node_id to be populated, got %v", comment.CommentNodeID)
	}
	if len(comment.ThreadComments) > 0 {
		if comment.ThreadComments[0].CommentNodeID == nil || *comment.ThreadComments[0].CommentNodeID == "" {
			t.Fatalf("expected reply comment_node_id populated, got %v", comment.ThreadComments[0].CommentNodeID)
		}
	}
}

type fakeViewAPI struct {
	t         *testing.T
	payload   []byte
	variables map[string]interface{}
}

func (f *fakeViewAPI) REST(string, string, map[string]string, interface{}, interface{}) error {
	f.t.Fatalf("unexpected REST call in view command")
	return nil
}

func (f *fakeViewAPI) GraphQL(query string, variables map[string]interface{}, result interface{}) error {
	f.variables = variables
	return json.Unmarshal(f.payload, result)
}
