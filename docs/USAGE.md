# Usage reference

All commands accept pull request selectors as either:

- a pull request URL (`https://github.com/owner/repo/pull/123`)
- a pull request number when combined with `-R owner/repo`

Unless stated otherwise, commands emit JSON only. Optional fields are omitted instead of serializing as `null`. Array responses default to `[]`.

## review --start

- **Purpose:** Open (or resume) a pending review on the head commit.
- **Inputs:**
  - Optional pull request selector argument.
  - `--repo` / `--pr` flags when not using the selector shorthand.
  - `--commit` to pin the pending review to a specific commit SHA (defaults to the pull request head).
- **Output schema:** [`ReviewState`](SCHEMAS.md#reviewstate) — required fields `id` and `state`; optional `submitted_at`.

```sh
gh pr-review review --start -R owner/repo 42

{
  "id": "PRR_kwDOAAABbcdEFG12",
  "state": "PENDING"
}
```

## review --add-comment

- **Purpose:** Attach an inline thread to an existing pending review.
- **Inputs:**
  - `--review-id` **(required):** Review node ID (must start with `PRR_`). Numeric IDs are rejected.
  - `--path`, `--line`, `--body` **(required).**
  - `--body-file`: Read body from a file instead of `--body` (use `"-"` for stdin). Mutually exclusive with `--body`.
  - `--side`, `--start-line`, `--start-side` to describe diff positioning.
- **Output schema:** [`ReviewThread`](SCHEMAS.md#reviewthread) — required fields `id`, `path`, `is_outdated`; optional `line`.

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

## review --edit-comment

- **Purpose:** Edit/update the body of a comment in a pending review.
- **Inputs:**
  - `--comment-id` **(required):** Comment node ID (must start with `PRRC_`).
  - `--body` **(required):** New comment text.
- **Output schema:** Status payload `{"status": "Comment updated successfully"}`.

```sh
gh pr-review review --edit-comment \
  --comment-id PRRC_kwDOAAABbcdEFG12 \
  --body "Updated: use helper function here" \
  -R owner/repo 42

{
  "status": "Comment updated successfully"
}
```

> **Note:** This only works on comments in **pending** reviews. Once a review is submitted, comments cannot be edited.

## review view

- **Purpose:** Emit a consolidated snapshot of reviews, inline comments, and replies.
- **Inputs:**
  - Optional pull request selector argument (URL or number with `--repo`).
  - `--repo` / `--pr` flags when not providing the positional number.
  - Filters: `--reviewer`, `--states`, `--unresolved`, `--not_outdated`, `--tail`.
  - `--include-comment-node-id` to surface comment IDs on parent comments and replies.
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

## review --submit

- **Purpose:** Finalize a pending review as COMMENT, APPROVE, or REQUEST_CHANGES.
- **Inputs:**
  - `--review-id` **(required):** Review node ID (must start with `PRR_`). Numeric IDs are rejected.
  - `--event` **(required):** One of `COMMENT`, `APPROVE`, `REQUEST_CHANGES`.
  - `--body`: Optional message. GitHub requires a body for `REQUEST_CHANGES`.
  - `--body-file`: Read body from a file instead of `--body` (use `"-"` for stdin). Mutually exclusive with `--body`.
- **Output schema:** Status payload `{"status": "…"}`. On errors, the command emits `{ "status": "Review submission failed", "errors": [...] }` and exits non-zero.

```sh
gh pr-review review --submit \
  --review-id PRR_kwDOAAABbcdEFG12 \
  --event REQUEST_CHANGES \
  --body "Please cover edge cases" \
  -R owner/repo 42

{
  "status": "Review submitted successfully"
}
```

## comments reply

- **Purpose:** Reply to a review thread.
- **Inputs:**
  - `--thread-id` **(required):** Review thread identifier (`PRRT_…`).
  - `--review-id`: Review identifier when replying inside your pending review (`PRR_…`).
  - `--body` **(required,** or use `--body-file`**).**
  - `--body-file`: Read reply text from a file instead of `--body` (use `"-"` for stdin). Mutually exclusive with `--body`.
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

## threads list

- **Purpose:** Enumerate review threads for a pull request.
- **Inputs:**
  - `--unresolved` to filter unresolved threads only.
  - `--mine` to include only threads you can resolve or participated in.
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

## threads view

- **Purpose:** Show the full conversation and metadata for one or more review threads by thread ID.
- **Inputs:**
  - One or more `--thread-id` values.
  - Pull request selector (`-R owner/repo <pr-number>` or PR URL).
- **Output schema:** Array of [`ThreadDetail`](SCHEMAS.md#threaddetail).

```sh
gh pr-review threads view PRRT_kwDOAAABbFg12345 PRRT_kwDOAAABbFg67890 -R owner/repo 42

[
  {
    "threadId": "PRRT_kwDOAAABbFg12345",
    "isResolved": false,
    "isOutdated": false,
    "path": "internal/service.go",
    "line": 42,
    "comments": [
      {
        "author_login": "octocat",
        "body": "nit: prefer helper",
        "created_at": "2025-12-03T10:00:00Z"
      },
      {
        "author_login": "squirrel289",
        "body": "Fixed in latest commit.",
        "created_at": "2025-12-03T11:00:00Z"
      }
    ]
  }
]
```

## threads resolve / threads unresolve

- **Purpose:** Resolve or reopen a review thread.
- **Inputs:**
  - `--thread-id` **(required):** Review thread node ID (`PRRT_…`).
- **Output schema:** [`ThreadMutationResult`](SCHEMAS.md#threadmutationresult).

```sh
gh pr-review threads resolve --thread-id R_ywDoABC123 -R owner/repo 42

{
  "thread_node_id": "R_ywDoABC123",
  "is_resolved": true
}
```

`threads unresolve` emits the same schema with `is_resolved` set to `false`.
