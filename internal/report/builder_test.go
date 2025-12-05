package report_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/Agyn-sandbox/gh-pr-review/internal/report"
)

func TestBuildReportAggregatesThreads(t *testing.T) {
	reviewBody := "Looks good"
	submittedAt := time.Date(2025, 12, 3, 10, 0, 0, 0, time.UTC)
	reviews := []report.Review{
		{
			ID:          "R1",
			State:       report.StateApproved,
			Body:        &reviewBody,
			SubmittedAt: &submittedAt,
			AuthorLogin: "alice",
			DatabaseID:  101,
		},
		{
			ID:          "R2",
			State:       report.StateCommented,
			Body:        strPtr(""),
			AuthorLogin: "bob",
			DatabaseID:  202,
		},
	}

	threadWithReplies := report.Thread{
		ID:         "T1",
		Path:       "main.go",
		Line:       intPtr(42),
		IsResolved: true,
		IsOutdated: false,
		Comments: []report.ThreadComment{
			{
				DatabaseID:        301,
				Body:              "Parent comment",
				CreatedAt:         time.Date(2025, 12, 3, 10, 1, 0, 0, time.UTC),
				AuthorLogin:       "alice",
				ReviewDatabaseID:  intPtr(101),
				ReplyToDatabaseID: nil,
			},
			{
				DatabaseID:        302,
				Body:              "First reply",
				CreatedAt:         time.Date(2025, 12, 3, 10, 2, 0, 0, time.UTC),
				AuthorLogin:       "bob",
				ReviewDatabaseID:  intPtr(101),
				ReplyToDatabaseID: intPtr(301),
			},
			{
				DatabaseID:        303,
				Body:              "Second reply",
				CreatedAt:         time.Date(2025, 12, 3, 10, 3, 0, 0, time.UTC),
				AuthorLogin:       "alice",
				ReviewDatabaseID:  intPtr(101),
				ReplyToDatabaseID: intPtr(302),
			},
		},
	}

	threadNoReplies := report.Thread{
		ID:         "T2",
		Path:       "main.go",
		Line:       nil,
		IsResolved: true,
		IsOutdated: false,
		Comments: []report.ThreadComment{
			{
				DatabaseID:        401,
				Body:              "Solo parent",
				CreatedAt:         time.Date(2025, 12, 3, 10, 4, 0, 0, time.UTC),
				AuthorLogin:       "alice",
				ReviewDatabaseID:  intPtr(101),
				ReplyToDatabaseID: nil,
			},
		},
	}

	result := report.BuildReport(reviews, []report.Thread{threadWithReplies, threadNoReplies}, report.FilterOptions{})

	if len(result.Reviews) != 2 {
		t.Fatalf("expected 2 reviews, got %d", len(result.Reviews))
	}

	first := result.Reviews[0]
	if first.ID != "R1" {
		t.Fatalf("expected first review to be R1, got %s", first.ID)
	}
	if first.SubmittedAt == nil || *first.SubmittedAt != "2025-12-03T10:00:00Z" {
		t.Fatalf("unexpected submitted_at: %v", first.SubmittedAt)
	}
	if len(first.Comments) != 2 {
		t.Fatalf("expected 2 comments for first review, got %d", len(first.Comments))
	}
	comment := mustFindComment(first.Comments, 301)
	if comment.ID != 301 {
		t.Fatalf("expected parent ID 301, got %d", comment.ID)
	}
	if comment.Line == nil || *comment.Line != 42 {
		t.Fatalf("expected line 42, got %v", comment.Line)
	}
	if len(comment.Thread) != 2 {
		t.Fatalf("expected 2 replies, got %d", len(comment.Thread))
	}
	if comment.Thread[0].ID != 302 || comment.Thread[1].ID != 303 {
		t.Fatalf("unexpected reply IDs: %#v", comment.Thread)
	}
	noReplyComment := mustFindComment(first.Comments, 401)
	if noReplyComment.ID != 401 {
		t.Fatalf("expected parent ID 401, got %d", noReplyComment.ID)
	}
	if len(noReplyComment.Thread) != 0 {
		t.Fatalf("expected no replies for comment 401, got %d", len(noReplyComment.Thread))
	}

	second := result.Reviews[1]
	if second.Body != nil {
		t.Fatalf("expected empty body to be omitted, got %q", *second.Body)
	}
	if second.SubmittedAt != nil {
		t.Fatalf("expected submitted_at to be nil, got %v", *second.SubmittedAt)
	}
	if second.Comments != nil {
		t.Fatalf("expected nil comments for second review, got %#v", second.Comments)
	}

	jsonBytes, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	if !strings.Contains(string(jsonBytes), `"thread":[]`) {
		t.Fatal("expected empty thread array encoded")
	}
	if strings.Contains(string(jsonBytes), `"body":""`) {
		t.Fatal("expected empty body fields to be omitted from JSON")
	}
}

func TestBuildReportFilterOptions(t *testing.T) {
	reviews := []report.Review{
		{ID: "R1", State: report.StateApproved, AuthorLogin: "alice", DatabaseID: 1},
		{ID: "R2", State: report.StateChangesRequested, AuthorLogin: "bob", DatabaseID: 2},
	}

	threads := []report.Thread{
		{
			ID:         "T1",
			Path:       "file.go",
			IsResolved: false,
			IsOutdated: true,
			Comments: []report.ThreadComment{
				{DatabaseID: 10, Body: "Parent", CreatedAt: time.Date(2025, 12, 3, 0, 0, 0, 0, time.UTC), AuthorLogin: "alice", ReviewDatabaseID: intPtr(1)},
				{DatabaseID: 11, Body: "Reply", CreatedAt: time.Date(2025, 12, 3, 0, 1, 0, 0, time.UTC), AuthorLogin: "carol", ReviewDatabaseID: intPtr(1), ReplyToDatabaseID: intPtr(10)},
			},
		},
		{
			ID:         "T2",
			Path:       "file.go",
			IsResolved: false,
			IsOutdated: false,
			Comments: []report.ThreadComment{
				{DatabaseID: 20, Body: "Parent", CreatedAt: time.Date(2025, 12, 3, 0, 2, 0, 0, time.UTC), AuthorLogin: "bob", ReviewDatabaseID: intPtr(2)},
				{DatabaseID: 21, Body: "Reply1", CreatedAt: time.Date(2025, 12, 3, 0, 3, 0, 0, time.UTC), AuthorLogin: "dave", ReviewDatabaseID: intPtr(2), ReplyToDatabaseID: intPtr(20)},
				{DatabaseID: 22, Body: "Reply2", CreatedAt: time.Date(2025, 12, 3, 0, 4, 0, 0, time.UTC), AuthorLogin: "eve", ReviewDatabaseID: intPtr(2), ReplyToDatabaseID: intPtr(21)},
			},
		},
	}

	filters := report.FilterOptions{
		Reviewer:           "bob",
		States:             []report.State{report.StateChangesRequested},
		RequireUnresolved:  true,
		RequireNotOutdated: true,
		TailReplies:        1,
	}

	result := report.BuildReport(reviews, threads, filters)

	if len(result.Reviews) != 1 {
		t.Fatalf("expected 1 review, got %d", len(result.Reviews))
	}
	review := result.Reviews[0]
	if review.ID != "R2" {
		t.Fatalf("expected review R2, got %s", review.ID)
	}
	if len(review.Comments) != 1 {
		t.Fatalf("expected 1 comment, got %d", len(review.Comments))
	}
	comment := review.Comments[0]
	if comment.ID != 20 {
		t.Fatalf("expected parent ID 20, got %d", comment.ID)
	}
	if len(comment.Thread) != 1 {
		t.Fatalf("expected 1 reply after tail filter, got %d", len(comment.Thread))
	}
	if comment.Thread[0].ID != 22 {
		t.Fatalf("expected last reply ID 22, got %d", comment.Thread[0].ID)
	}
	if comment.IsOutdated {
		t.Fatal("expected is_outdated to be false after filtering")
	}
	if comment.IsResolved {
		t.Fatal("expected unresolved thread to remain unresolved")
	}
}

func intPtr(v int) *int {
	return &v
}

func mustFindComment(comments []report.ReportComment, id int) report.ReportComment {
	for _, comment := range comments {
		if comment.ID == id {
			return comment
		}
	}
	return report.ReportComment{}
}

func strPtr(v string) *string {
	return &v
}
