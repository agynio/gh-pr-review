package report

import (
	"sort"
	"strings"
	"time"
)

// BuildReport aggregates reviews and threads into the serialized report format.
func BuildReport(reviews []Review, threads []Thread, filters FilterOptions) Report {
	allowedStates := allowedStateSet(filters.States)

	var reviewerFilter string
	if filters.Reviewer != "" {
		reviewerFilter = strings.ToLower(filters.Reviewer)
	}

	reportReviews := make([]ReportReview, 0, len(reviews))
	reviewIndexByID := make(map[int]int, len(reviews))

	for _, review := range reviews {
		if _, ok := allowedStates[review.State]; !ok {
			continue
		}
		if reviewerFilter != "" && strings.ToLower(review.AuthorLogin) != reviewerFilter {
			continue
		}

		var submittedAt *string
		if review.SubmittedAt != nil {
			formatted := review.SubmittedAt.UTC().Format(time.RFC3339)
			submittedAt = &formatted
		}

		var body *string
		if review.Body != nil {
			trimmed := strings.TrimSpace(*review.Body)
			if trimmed != "" {
				body = &trimmed
			}
		}

		rep := ReportReview{
			ID:          review.ID,
			State:       review.State,
			Body:        body,
			SubmittedAt: submittedAt,
			AuthorLogin: review.AuthorLogin,
		}

		reviewIndexByID[review.DatabaseID] = len(reportReviews)
		reportReviews = append(reportReviews, rep)
	}

	if len(reportReviews) == 0 {
		return Report{Reviews: []ReportReview{}}
	}

	for _, thread := range threads {
		if filters.RequireUnresolved && thread.IsResolved {
			continue
		}
		if filters.RequireNotOutdated && thread.IsOutdated {
			continue
		}

		var parent *ThreadComment
		replies := make([]ThreadComment, 0, len(thread.Comments))
		for _, comment := range thread.Comments {
			if comment.ReplyToDatabaseID == nil {
				if parent == nil {
					c := comment
					parent = &c
				}
				continue
			}
			replies = append(replies, comment)
		}
		if parent == nil || parent.ReviewDatabaseID == nil {
			continue
		}

		reviewIdx, ok := reviewIndexByID[*parent.ReviewDatabaseID]
		if !ok {
			continue
		}

		sort.SliceStable(replies, func(i, j int) bool {
			return replies[i].CreatedAt.Before(replies[j].CreatedAt)
		})

		if filters.TailReplies > 0 && len(replies) > filters.TailReplies {
			replies = replies[len(replies)-filters.TailReplies:]
		}

		reportReplies := make([]ThreadReply, len(replies))
		for i, reply := range replies {
			createdAt := reply.CreatedAt.UTC().Format(time.RFC3339)
			reportReplies[i] = ThreadReply{
				ID:          reply.DatabaseID,
				InReplyToID: reply.ReplyToDatabaseID,
				AuthorLogin: reply.AuthorLogin,
				Body:        reply.Body,
				CreatedAt:   createdAt,
			}
		}

		createdAt := parent.CreatedAt.UTC().Format(time.RFC3339)
		reportComment := ReportComment{
			ID:          parent.DatabaseID,
			Path:        thread.Path,
			Line:        thread.Line,
			AuthorLogin: parent.AuthorLogin,
			Body:        parent.Body,
			CreatedAt:   createdAt,
			IsResolved:  thread.IsResolved,
			IsOutdated:  thread.IsOutdated,
			Thread:      reportReplies,
		}

		if len(reportReplies) == 0 {
			reportComment.Thread = []ThreadReply{}
		}

		review := &reportReviews[reviewIdx]
		review.Comments = append(review.Comments, reportComment)
	}

	for i := range reportReviews {
		if len(reportReviews[i].Comments) == 0 {
			reportReviews[i].Comments = nil
		}
	}

	return Report{Reviews: reportReviews}
}

func allowedStateSet(states []State) map[State]struct{} {
	if len(states) == 0 {
		return map[State]struct{}{
			StateApproved:         {},
			StateChangesRequested: {},
			StateCommented:        {},
			StateDismissed:        {},
		}
	}

	set := make(map[State]struct{}, len(states))
	for _, st := range states {
		set[st] = struct{}{}
	}
	return set
}
