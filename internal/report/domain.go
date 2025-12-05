package report

import "time"

// State represents the pull request review state supported by the report command.
type State string

const (
	StateApproved         State = "APPROVED"
	StateChangesRequested State = "CHANGES_REQUESTED"
	StateCommented        State = "COMMENTED"
	StateDismissed        State = "DISMISSED"
)

// FilterOptions controls shaping of reviews and threads.
type FilterOptions struct {
	Reviewer           string
	States             []State
	RequireUnresolved  bool
	RequireNotOutdated bool
	TailReplies        int
}

// Review models a pull request review fetched from GraphQL.
type Review struct {
	ID          string
	State       State
	Body        *string
	SubmittedAt *time.Time
	AuthorLogin string
	DatabaseID  int
}

// Thread captures a review thread and its constituent comments.
type Thread struct {
	ID         string
	Path       string
	Line       *int
	IsResolved bool
	IsOutdated bool
	Comments   []ThreadComment
}

// ThreadComment represents a single comment node within a thread.
type ThreadComment struct {
	DatabaseID        int
	Body              string
	CreatedAt         time.Time
	AuthorLogin       string
	ReviewDatabaseID  *int
	ReplyToDatabaseID *int
}

// Report is the serialized output structure for the report command.
type Report struct {
	Reviews []ReportReview `json:"reviews"`
}

// ReportReview aggregates review data and associated thread comments.
type ReportReview struct {
	ID          string          `json:"id"`
	State       State           `json:"state"`
	Body        *string         `json:"body,omitempty"`
	SubmittedAt *string         `json:"submitted_at,omitempty"`
	AuthorLogin string          `json:"author_login"`
	Comments    []ReportComment `json:"comments,omitempty"`
}

// ReportComment contains the shaped parent comment for a thread.
type ReportComment struct {
	ID          int           `json:"id"`
	Path        string        `json:"path"`
	Line        *int          `json:"line,omitempty"`
	AuthorLogin string        `json:"author_login"`
	Body        string        `json:"body"`
	CreatedAt   string        `json:"created_at"`
	IsResolved  bool          `json:"is_resolved"`
	IsOutdated  bool          `json:"is_outdated"`
	Thread      []ThreadReply `json:"thread"`
}

// ThreadReply captures a reply within a thread.
type ThreadReply struct {
	ID          int    `json:"id"`
	InReplyToID *int   `json:"in_reply_to_id,omitempty"`
	AuthorLogin string `json:"author_login"`
	Body        string `json:"body"`
	CreatedAt   string `json:"created_at"`
}
