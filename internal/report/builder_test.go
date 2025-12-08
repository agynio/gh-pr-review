package report_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/agynio/gh-pr-review/internal/report"
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
				NodeID:             "C301",
				DatabaseID:         301,
				Body:               "Parent comment",
				CreatedAt:          time.Date(2025, 12, 3, 10, 1, 0, 0, time.UTC),
				AuthorLogin:        "alice",
				ReviewDatabaseID:   intPtr(101),
				ReplyToDatabaseID:  nil,
				ReplyToCommentNode: nil,
			},
			{
				NodeID:             "C302",
				DatabaseID:         302,
				Body:               "First reply",
				CreatedAt:          time.Date(2025, 12, 3, 10, 2, 0, 0, time.UTC),
				AuthorLogin:        "bob",
				ReviewDatabaseID:   intPtr(101),
				ReplyToDatabaseID:  intPtr(301),
				ReplyToCommentNode: strPtr("C301"),
			},
			{
				NodeID:             "C303",
				DatabaseID:         303,
				Body:               "Second reply",
				CreatedAt:          time.Date(2025, 12, 3, 10, 3, 0, 0, time.UTC),
				AuthorLogin:        "alice",
				ReviewDatabaseID:   intPtr(101),
				ReplyToDatabaseID:  intPtr(302),
				ReplyToCommentNode: strPtr("C302"),
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
				NodeID:             "C401",
				DatabaseID:         401,
				Body:               "Solo parent",
				CreatedAt:          time.Date(2025, 12, 3, 10, 4, 0, 0, time.UTC),
				AuthorLogin:        "alice",
				ReviewDatabaseID:   intPtr(101),
				ReplyToDatabaseID:  nil,
				ReplyToCommentNode: nil,
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
	comment := mustFindComment(first.Comments, "T1")
	if comment.ThreadID != "T1" {
		t.Fatalf("expected thread T1, got %s", comment.ThreadID)
	}
	if comment.CommentNodeID != nil {
		t.Fatalf("expected comment_node_id to be omitted by default, got %v", *comment.CommentNodeID)
	}
	if comment.Line == nil || *comment.Line != 42 {
		t.Fatalf("expected line 42, got %v", comment.Line)
	}
	if len(comment.ThreadComments) != 2 {
		t.Fatalf("expected 2 replies, got %d", len(comment.ThreadComments))
	}
	if comment.ThreadComments[0].Body != "First reply" || comment.ThreadComments[1].Body != "Second reply" {
		t.Fatalf("unexpected reply ordering: %#v", comment.ThreadComments)
	}
	if comment.ThreadComments[0].CommentNodeID != nil || comment.ThreadComments[1].CommentNodeID != nil {
		t.Fatal("expected reply comment_node_id omitted by default")
	}
	noReplyComment := mustFindComment(first.Comments, "T2")
	if noReplyComment.ThreadID != "T2" {
		t.Fatalf("expected thread T2, got %s", noReplyComment.ThreadID)
	}
	if len(noReplyComment.ThreadComments) != 0 {
		t.Fatalf("expected no replies for comment 401, got %d", len(noReplyComment.ThreadComments))
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

	withIDs := report.BuildReport(reviews, []report.Thread{threadWithReplies, threadNoReplies}, report.FilterOptions{IncludeCommentNodeID: true})
	if len(withIDs.Reviews) == 0 {
		t.Fatal("expected reviews to be present when including comment node IDs")
	}
	commentWithIDs := mustFindComment(withIDs.Reviews[0].Comments, "T1")
	if commentWithIDs.CommentNodeID == nil || *commentWithIDs.CommentNodeID != "C301" {
		t.Fatalf("expected comment_node_id C301, got %v", commentWithIDs.CommentNodeID)
	}
	if len(commentWithIDs.ThreadComments) != 2 {
		t.Fatalf("expected replies to remain when including node IDs, got %d", len(commentWithIDs.ThreadComments))
	}
	if commentWithIDs.ThreadComments[0].CommentNodeID == nil || *commentWithIDs.ThreadComments[0].CommentNodeID != "C302" {
		t.Fatalf("expected reply comment_node_id C302, got %v", commentWithIDs.ThreadComments[0].CommentNodeID)
	}

	jsonBytes, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	if !strings.Contains(string(jsonBytes), `"thread_comments":[]`) {
		t.Fatal("expected empty thread_comments array encoded")
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
				{NodeID: "C10", DatabaseID: 10, Body: "Parent", CreatedAt: time.Date(2025, 12, 3, 0, 0, 0, 0, time.UTC), AuthorLogin: "alice", ReviewDatabaseID: intPtr(1)},
				{NodeID: "C11", DatabaseID: 11, Body: "Reply", CreatedAt: time.Date(2025, 12, 3, 0, 1, 0, 0, time.UTC), AuthorLogin: "carol", ReviewDatabaseID: intPtr(1), ReplyToDatabaseID: intPtr(10), ReplyToCommentNode: strPtr("C10")},
			},
		},
		{
			ID:         "T2",
			Path:       "file.go",
			IsResolved: false,
			IsOutdated: false,
			Comments: []report.ThreadComment{
				{NodeID: "C20", DatabaseID: 20, Body: "Parent", CreatedAt: time.Date(2025, 12, 3, 0, 2, 0, 0, time.UTC), AuthorLogin: "bob", ReviewDatabaseID: intPtr(2)},
				{NodeID: "C21", DatabaseID: 21, Body: "Reply1", CreatedAt: time.Date(2025, 12, 3, 0, 3, 0, 0, time.UTC), AuthorLogin: "dave", ReviewDatabaseID: intPtr(2), ReplyToDatabaseID: intPtr(20), ReplyToCommentNode: strPtr("C20")},
				{NodeID: "C22", DatabaseID: 22, Body: "Reply2", CreatedAt: time.Date(2025, 12, 3, 0, 4, 0, 0, time.UTC), AuthorLogin: "eve", ReviewDatabaseID: intPtr(2), ReplyToDatabaseID: intPtr(21), ReplyToCommentNode: strPtr("C21")},
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
	if comment.ThreadID != "T2" {
		t.Fatalf("expected thread ID T2, got %s", comment.ThreadID)
	}
	if len(comment.ThreadComments) != 1 {
		t.Fatalf("expected 1 reply after tail filter, got %d", len(comment.ThreadComments))
	}
	if comment.ThreadComments[0].Body != "Reply2" {
		t.Fatalf("expected last reply body Reply2, got %s", comment.ThreadComments[0].Body)
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

func mustFindComment(comments []report.ReportComment, threadID string) report.ReportComment {
	for _, comment := range comments {
		if comment.ThreadID == threadID {
			return comment
		}
	}
	return report.ReportComment{}
}

func strPtr(v string) *string {
	return &v
}
