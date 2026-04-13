# Usage reference (v1.6.0)

All commands accept pull request selectors as either:

- a pull request URL (`https://github.com/owner/repo/pull/123`)
- a pull request number when combined with `-R owner/repo`

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
gh pr-review review --start -R owner/repo 42

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
  -R owner/repo 42

{
  "id": "PRRT_kwDOAAABbcdEFG12",
  "path": "internal/service.go",
  "is_outdated": false,
  "line": 42
}
```

## review --delete-comment (GraphQL only)

- **Purpose:** Delete a comment from a pending review.
- **Inputs:**
  - `--comment-id` **(required):** GraphQL comment node ID (must start with
    `PRRC_`).
- **Backend:** GitHub GraphQL `deletePullRequestReviewComment` mutation.
- **Output schema:** Status payload `{"status": "Comment deleted successfully"}`.

```sh
gh pr-review review --delete-comment \
  --comment-id PRRC_kwDOAAABbcdEFG12 \
  -R owner/repo 42

{
  "status": "Comment deleted successfully"
}
```

> **Note:** This only works on comments in **pending** reviews. Once a review is
> submitted, comments cannot be deleted via this API.

## review view (GraphQL only)

- **Purpose:** Emit a consolidated snapshot of reviews, inline comments, and
  replies. Use it to capture thread identifiers before replying or resolving
  discussions.
- **Inputs:**
- Optional pull request selector argument (URL or number with `--repo`).
  - `--repo` / `--pr` flags when not providing the positional number.
  - Filters: `--reviewer`, `--states`, `--unresolved`, `--not_outdated`,
    `--tail`.
  - `--include-comment-node-id` to surface GraphQL comment IDs on parent
    comments and replies.
- **Backend:** GitHub GraphQL `pullRequest.reviews` query.
- **Output shape:**

```sh
gh pr-review review view --reviewer octocat --states CHANGES_REQUESTED -R owner/repo 42

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
          "thread_comments": []
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
  -R owner/repo 42

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

> **Tip:** `review view` is the preferred way to discover review metadata
> (pending review IDs, thread IDs, optional comment node IDs, thread state)
> before mutating threads or
> replying.

## comments reply (GraphQL only)

- **Purpose:** Reply to a review thread.
- **Inputs:**
  - `--thread-id` **(required):** GraphQL review thread identifier (`PRRT_…`).
  - `--review-id`: GraphQL review identifier when replying inside your pending
    review (`PRR_…`).
  - `--body` **(required).**
- **Backend:** GitHub GraphQL `addPullRequestReviewThreadReply` mutation.
- **Output schema:** [`ReplyMinimal`](SCHEMAS.md#replyminimal).

```sh
gh pr-review comments reply \
  --thread-id PRRT_kwDOAAABbFg12345 \
  --body "Ack" \
  -R owner/repo 42

{
  "comment_node_id": "PRRC_kwDOAAABbhi7890"
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
gh pr-review threads list --unresolved --mine -R owner/repo 42

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

## threads resolve / threads unresolve (GraphQL only)

- **Purpose:** Resolve or reopen a review thread.
- **Inputs:**
  - `--thread-id` **(required):** GraphQL review thread node ID (`PRRT_…`).
- **Backend:** GraphQL mutations `resolveReviewThread` / `unresolveReviewThread`.
- **Output schema:** [`ThreadMutationResult`](SCHEMAS.md#threadmutationresult).

```sh
gh pr-review threads resolve --thread-id R_ywDoABC123 -R owner/repo 42

{
  "thread_node_id": "R_ywDoABC123",
  "is_resolved": true
}
```

`threads unresolve` emits the same schema with `is_resolved` set to `false`.
