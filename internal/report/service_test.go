package report

import (
	"encoding/json"
	"strings"
	"testing"

	_ "embed"

	"github.com/Agyn-sandbox/gh-pr-review/internal/resolver"
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
	if comment.ID != 301 {
		t.Fatalf("expected parent comment 301, got %d", comment.ID)
	}
	if len(comment.Thread) != 1 {
		t.Fatalf("expected 1 reply after tail filter, got %d", len(comment.Thread))
	}
	if comment.Thread[0].ID != 303 {
		t.Fatalf("expected reply 303, got %d", comment.Thread[0].ID)
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
