# Usage reference (v1.3.4)

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
  replies. Use it to discover numeric comment identifiers before replying.
- **Inputs:**
  - Optional pull request selector argument (`owner/repo#123` or URL).
  - `--repo` / `--pr` flags when not using the selector shorthand.
  - Filters: `--reviewer`, `--states`, `--unresolved`, `--not_outdated`,
    `--tail`.
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
          "id": 3531807471,
          "path": "internal/service.go",
          "body": "nit: prefer helper",
          "thread": []
        }
      ]
    }
  ]
}
```

The numeric comment `id` values surfaced in the report feed directly into
`comments reply`.

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
> (pending review IDs, comment IDs, thread state) before mutating threads or
> replying.

## comments reply (REST, optional concise mode)

- **Purpose:** Reply to a review comment.
- **Inputs:**
  - `--comment-id` **(required):** Numeric comment identifier.
  - `--body` **(required).**
  - `--concise`: Emit the minimal `{ "id": <reply-id> }` response.
- **Backend:** GitHub REST `POST /pulls/comments/{comment_id}/replies`.
- **Auto-submit behavior:** When GitHub blocks the reply because you own a
  pending review, the extension uses GraphQL only to locate pending review IDs
  and submits them via REST before retrying.
- **Output schema:**
  - Default: GitHub REST pull request review comment object (see
    [`ReplyComment`](SCHEMAS.md#replycomment)).
  - `--concise`: [`ReplyConcise`](SCHEMAS.md#replyconcise).

```sh
# Full REST payload
gh pr-review comments reply \
  --comment-id 987654321 \
  --body "Thanks for catching this" \
  owner/repo#42

# Sample output (subset of fields; GitHub returns additional metadata)
{
  "id": 1122334455,
  "node_id": "MDEyOkNvbW1lbnQxMTIyMzM0NDU1",
  "pull_request_review_id": 3531807471,
  "in_reply_to_id": 987654321,
  "body": "Thanks for catching this",
  "user": { "login": "octocat", "id": 6752317 },
  "path": "internal/service.go",
  "line": 42,
  "side": "RIGHT",
  "html_url": "https://github.com/owner/repo/pull/42#discussion_r1122334455",
  "created_at": "2024-12-19T18:35:02Z",
  "updated_at": "2024-12-19T18:35:02Z"
}

# Concise payload
gh pr-review comments reply \
  --comment-id 987654321 \
  --body "Ack" \
  --concise \
  owner/repo#42

{
  "id": 123456789
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
