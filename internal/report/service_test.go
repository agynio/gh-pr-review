package report

import (
	"encoding/json"
	"strings"
	"testing"

	_ "embed"

	"github.com/agynio/gh-pr-review/internal/resolver"
)

//go:embed testdata/report_response.json
var reportResponseFixture []byte

func TestServiceFetchShapesReport(t *testing.T) {
	fake := &stubAPI{t: t, payload: reportResponseFixture}
	svc := NewService(fake)

	identity := resolver.Identity{Owner: "agyn", Repo: "sandbox", Number: 51}
	result, err := svc.Fetch(identity, Options{
		Reviewer:           "alice",
		States:             []State{StateApproved, StateCommented},
		StatesProvided:     true,
		RequireNotOutdated: true,
		TailReplies:        1,
	})
	if err != nil {
		t.Fatalf("fetch report: %v", err)
	}

	if len(result.Reviews) != 1 {
		t.Fatalf("expected 1 review after filtering, got %d", len(result.Reviews))
	}
	review := result.Reviews[0]
	if review.ID != "R1" {
		t.Fatalf("expected review R1, got %s", review.ID)
	}
	if len(review.Comments) != 1 {
		t.Fatalf("expected 1 comment, got %d", len(review.Comments))
	}
	comment := review.Comments[0]
	if comment.ThreadID != "T1" {
		t.Fatalf("expected parent thread T1, got %s", comment.ThreadID)
	}
	if comment.CommentNodeID != nil {
		t.Fatalf("expected comment_node_id omitted by default, got %v", comment.CommentNodeID)
	}
	if len(comment.ThreadComments) != 1 {
		t.Fatalf("expected 1 reply after tail filter, got %d", len(comment.ThreadComments))
	}
	if comment.ThreadComments[0].Body != "Reply beta" {
		t.Fatalf("expected reply body 'Reply beta', got %s", comment.ThreadComments[0].Body)
	}
	if comment.ThreadComments[0].CommentNodeID != nil {
		t.Fatalf("expected reply comment_node_id omitted by default, got %v", comment.ThreadComments[0].CommentNodeID)
	}

	rawStates, ok := fake.lastVariables["states"]
	if !ok {
		t.Fatalf("expected states variable propagated, variables: %#v", fake.lastVariables)
	}
	statesVar, ok := rawStates.([]string)
	if !ok || len(statesVar) != 2 {
		t.Fatalf("expected states variable propagated as []string, got %#v", rawStates)
	}
}

func TestServiceFetchIncludesCommentNodeID(t *testing.T) {
	fake := &stubAPI{t: t, payload: reportResponseFixture}
	svc := NewService(fake)

	identity := resolver.Identity{Owner: "agyn", Repo: "sandbox", Number: 51}
	result, err := svc.Fetch(identity, Options{IncludeCommentNodeID: true})
	if err != nil {
		t.Fatalf("fetch report with comment node ids: %v", err)
	}

	if len(result.Reviews) == 0 {
		t.Fatal("expected reviews in result")
	}
	review := result.Reviews[0]
	if len(review.Comments) == 0 {
		t.Fatal("expected comments for review")
	}
	comment := review.Comments[0]
	if comment.CommentNodeID == nil || *comment.CommentNodeID != "C301" {
		t.Fatalf("expected comment_node_id C301, got %v", comment.CommentNodeID)
	}
	if len(comment.ThreadComments) == 0 {
		t.Fatal("expected replies to be present")
	}
	if comment.ThreadComments[0].CommentNodeID == nil || *comment.ThreadComments[0].CommentNodeID == "" {
		t.Fatalf("expected reply comment node id, got %v", comment.ThreadComments[0].CommentNodeID)
	}
}

func TestServiceFetchErrorsOnMissingReviewDBID(t *testing.T) {
	broken := map[string]any{}
	if err := json.Unmarshal(reportResponseFixture, &broken); err != nil {
		t.Fatalf("unmarshal fixture: %v", err)
	}
	repo := broken["repository"].(map[string]any)
	pr := repo["pullRequest"].(map[string]any)
	reviews := pr["reviews"].(map[string]any)
	nodes := reviews["nodes"].([]any)
	first := nodes[0].(map[string]any)
	delete(first, "databaseId")

	modified, err := json.Marshal(broken)
	if err != nil {
		t.Fatalf("marshal modified: %v", err)
	}

	fake := &stubAPI{t: t, payload: modified}
	svc := NewService(fake)

	_, err = svc.Fetch(resolver.Identity{Owner: "agyn", Repo: "sandbox", Number: 51}, Options{})
	if err == nil {
		t.Fatal("expected error for missing databaseId")
	}
	if !strings.Contains(err.Error(), "review missing databaseId") {
		t.Fatalf("expected missing databaseId error, got %v", err)
	}
}

type stubAPI struct {
	t             *testing.T
	payload       []byte
	lastQuery     string
	lastVariables map[string]interface{}
}

func (s *stubAPI) REST(string, string, map[string]string, interface{}, interface{}) error {
	s.t.Fatalf("unexpected REST call in report service test")
	return nil
}

func (s *stubAPI) GraphQL(query string, variables map[string]interface{}, result interface{}) error {
	s.lastQuery = query
	s.lastVariables = variables
	if query != reportQuery {
		s.t.Fatalf("unexpected query: %s", query)
	}
	return json.Unmarshal(s.payload, result)
}
