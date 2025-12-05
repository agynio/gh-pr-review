# Usage reference (v1.4.0)

All commands accept pull request selectors in any GitHub CLI format:

- `owner/repo#123`
- a pull request URL (`https://github.com/owner/repo/pull/123`)
- `-R owner/repo 123`

Unless stated otherwise, commands emit JSON only. Optional fields are omitted
instead of serializing as `null`. Array responses default to `[]`.

## review --start (GraphQL only)

- **Purpose:** Open (or resume) a pending review on the head commit.
- **Inputs:**
  - Optional pull request selector argument.
  - `--repo` / `--pr` flags when not using the selector shorthand.
  - `--commit` to pin the pending review to a specific commit SHA (defaults to
    the pull request head).
- **Backend:** GitHub GraphQL `addPullRequestReview` mutation.
- **Output schema:** [`ReviewState`](SCHEMAS.md#reviewstate) — required fields
  `id` and `state`; optional `submitted_at`.

```sh
gh pr-review review --start owner/repo#42

{
  "id": "PRR_kwDOAAABbcdEFG12",
  "state": "PENDING"
}
```

## review --add-comment (GraphQL only)

- **Purpose:** Attach an inline thread to an existing pending review.
- **Inputs:**
  - `--review-id` **(required):** GraphQL review node ID (must start with
    `PRR_`). Numeric IDs are rejected.
  - `--path`, `--line`, `--body` **(required).**
  - `--side`, `--start-line`, `--start-side` to describe diff positioning.
- **Backend:** GitHub GraphQL `addPullRequestReviewThread` mutation.
- **Output schema:** [`ReviewThread`](SCHEMAS.md#reviewthread) — required fields
  `id`, `path`, `is_outdated`; optional `line`.

```sh
gh pr-review review --add-comment \
  --review-id PRR_kwDOAAABbcdEFG12 \
  --path internal/service.go \
  --line 42 \
  --body "nit: prefer helper" \
  owner/repo#42

{
  "id": "PRRT_kwDOAAABbcdEFG12",
  "path": "internal/service.go",
  "is_outdated": false,
  "line": 42
}
```

## review report (GraphQL only)

- **Purpose:** Emit a consolidated snapshot of reviews, inline comments, and
  replies. Use it to capture thread identifiers before replying or resolving
  discussions.
- **Inputs:**
  - Optional pull request selector argument (`owner/repo#123` or URL).
  - `--repo` / `--pr` flags when not using the selector shorthand.
  - Filters: `--reviewer`, `--states`, `--unresolved`, `--not_outdated`,
    `--tail`.
  - `--include-comment-node-id` to surface GraphQL comment IDs on parent
    comments and replies.
- **Backend:** GitHub GraphQL `pullRequest.reviews` query.
- **Output shape:**

```sh
gh pr-review review report --reviewer octocat --states CHANGES_REQUESTED owner/repo#42

{
  "reviews": [
    {
      "id": "PRR_kwDOAAABbcdEFG12",
      "state": "CHANGES_REQUESTED",
      "author_login": "octocat",
      "comments": [
        {
          "thread_id": "PRRT_kwDOAAABbFg12345",
          "path": "internal/service.go",
          "line": 42,
          "author_login": "octocat",
          "body": "nit: prefer helper",
          "created_at": "2025-12-03T10:00:00Z",
          "is_resolved": false,
          "is_outdated": false,
          "thread": []
        }
      ]
    }
  ]
}
```

The `thread_id` values surfaced in the report feed directly into
`comments reply`. Enable `--include-comment-node-id` to decorate parent
comments and replies with GraphQL `comment_node_id` fields; those keys remain
omitted otherwise.

## review --submit (GraphQL only)

- **Purpose:** Finalize a pending review as COMMENT, APPROVE, or
  REQUEST_CHANGES.
- **Inputs:**
  - `--review-id` **(required):** GraphQL review node ID (must start with
    `PRR_`). Numeric REST identifiers are rejected.
  - `--event` **(required):** One of `COMMENT`, `APPROVE`, `REQUEST_CHANGES`.
  - `--body`: Optional message. GitHub requires a body for
    `REQUEST_CHANGES`.
- **Backend:** GitHub GraphQL `submitPullRequestReview` mutation.
- **Output schema:** Status payload `{"status": "…"}`. When GraphQL returns
  errors, the command emits `{ "status": "Review submission failed",
  "errors": [...] }` and exits non-zero.

```sh
gh pr-review review --submit \
  --review-id PRR_kwDOAAABbcdEFG12 \
  --event REQUEST_CHANGES \
  --body "Please cover edge cases" \
  owner/repo#42

{
  "status": "Review submitted successfully"
}

# GraphQL error example
{
  "status": "Review submission failed",
  "errors": [
    { "message": "mutation failed", "path": ["mutation", "submitPullRequestReview"] }
  ]
}
```

> **Tip:** `review report` is the preferred way to discover review metadata
> (pending review IDs, thread IDs, optional comment node IDs, thread state)
> before mutating threads or
> replying.

## comments reply (GraphQL, optional concise mode)

- **Purpose:** Reply to a review thread.
- **Inputs:**
  - `--thread-id` **(required):** GraphQL review thread identifier (`PRRT_…`).
  - `--review-id`: GraphQL review identifier when replying inside your pending
    review (`PRR_…`).
  - `--body` **(required).**
  - `--concise`: Emit the minimal `{ "id": "<comment-id>" }` response.
- **Backend:** GitHub GraphQL `addPullRequestReviewThreadReply` mutation.
- **Output schema:**
  - Default: [`ReplyComment`](SCHEMAS.md#replycomment).
  - `--concise`: [`ReplyConcise`](SCHEMAS.md#replyconcise).

```sh
# Full GraphQL payload
gh pr-review comments reply \
  --thread-id PRRT_kwDOAAABbFg12345 \
  --review-id PRR_kwDOAAABbcdEFG12 \
  --body "Thanks for catching this" \
  owner/repo#42

{
  "id": "PRRC_kwDOAAABbhi7890",
  "database_id": 1122334455,
  "review_id": "PRR_kwDOAAABbcdEFG12",
  "review_database_id": 3531807471,
  "review_state": "PENDING",
  "thread_id": "PRRT_kwDOAAABbFg12345",
  "thread_is_resolved": false,
  "thread_is_outdated": false,
  "reply_to_comment_id": "PRRC_kwDOAAABbparent",
  "body": "Thanks for catching this",
  "diff_hunk": "@@ -10,5 +10,7 @@",
  "path": "internal/service.go",
  "html_url": "https://github.com/owner/repo/pull/42#discussion_r1122334455",
  "author_login": "octocat",
  "created_at": "2025-12-19T18:35:02Z",
  "updated_at": "2025-12-19T18:35:02Z"
}

# Concise payload
gh pr-review comments reply \
  --thread-id PRRT_kwDOAAABbFg12345 \
  --body "Ack" \
  --concise \
  owner/repo#42

{
  "id": "PRRC_kwDOAAABbhi7890"
}
```

## threads list (GraphQL)

- **Purpose:** Enumerate review threads for a pull request.
- **Inputs:**
  - `--unresolved` to filter unresolved threads only.
  - `--mine` to include only threads you can resolve or participated in.
- **Backend:** GitHub GraphQL `reviewThreads` query.
- **Output schema:** Array of [`ThreadSummary`](SCHEMAS.md#threadsummary).

```sh
gh pr-review threads list --unresolved --mine owner/repo#42

[
  {
    "threadId": "R_ywDoABC123",
    "isResolved": false,
    "updatedAt": "2024-12-19T18:40:11Z",
    "path": "internal/service.go",
    "line": 42,
    "isOutdated": false
  }
]
```

## threads resolve / threads unresolve (GraphQL + REST lookup when needed)

- **Purpose:** Resolve or reopen a review thread.
- **Inputs:**
  - Provide either `--thread-id` (GraphQL node) or `--comment-id` (REST review
    comment). Supplying both is rejected.
- **Backend:**
  - GraphQL mutations `resolveReviewThread` / `unresolveReviewThread`.
  - REST `GET /pulls/comments/{comment_id}` when mapping a numeric comment ID to
    a thread node.
- **Output schema:** [`ThreadActionResult`](SCHEMAS.md#threadactionresult).

```sh
# Resolve by GraphQL thread id
gh pr-review threads resolve --thread-id R_ywDoABC123 owner/repo#42

# Resolve by comment id (REST lookup + GraphQL mutation)
gh pr-review threads resolve --comment-id 2582545223 owner/repo#42

{
  "threadId": "R_ywDoABC123",
  "isResolved": true,
  "changed": true
}
```

`threads unresolve` emits the same schema, with `isResolved` equal to `false`
after reopening the thread.
