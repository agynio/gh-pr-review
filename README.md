# gh-pr-review
[![Agyn badge](https://agyn.io/badges/badge_dark.svg)](http://agyn.io)

`gh-pr-review` is a precompiled GitHub CLI extension for high-signal pull
request reviews. It manages pending GraphQL reviews, surfaces thread metadata,
and resolves discussions without cloning repositories.

- [Quickstart](#quickstart)
- [Review view](#review-view)
- [Backend policy](#backend-policy)
- [Additional docs](#additional-docs)
- [Release 1.6.0](#release-160)

## Quickstart

The quickest path from opening a pending review to resolving threads:

1. **Install or upgrade the extension.**

   ```sh
   gh extension install Agynio/gh-pr-review
   # Update an existing installation
   gh extension upgrade Agynio/gh-pr-review
   ```


2. **Start a pending review (GraphQL).** Capture the returned `id` (GraphQL
   node).

   ```sh
   gh pr-review review --start -R owner/repo 42

   {
     "id": "PRR_kwDOAAABbcdEFG12",
     "state": "PENDING"
   }
   ```

   Pending reviews omit `submitted_at`; the field appears after submission.

3. **Add inline comments with the pending review ID (GraphQL).** The
   `review --add-comment` command fails fast if you supply a numeric ID instead
   of the required `PRR_…` GraphQL identifier.

   ```sh
   gh pr-review review --add-comment \
     --review-id PRR_kwDOAAABbcdEFG12 \
     --path internal/service.go \
     --line 42 \
     --body "nit: use helper" \
     -R owner/repo 42

   {
     "id": "PRRT_kwDOAAABbcdEFG12",
     "path": "internal/service.go",
     "is_outdated": false,
     "line": 42
   }
   ```

4. **Inspect review threads (GraphQL).** `review view` surfaces pending
   review summaries, thread state, and inline comment metadata. Thread IDs are
   always included; enable `--include-comment-node-id` when you also need the
   individual comment node identifiers.

   ```sh
   gh pr-review review view --reviewer octocat -R owner/repo 42

   {
     "reviews": [
       {
         "id": "PRR_kwDOAAABbcdEFG12",
         "state": "COMMENTED",
         "comments": [
           {
             "thread_id": "PRRT_kwDOAAABbcdEFG12",
             "path": "internal/service.go",
             "body": "nit: prefer helper",
             "is_resolved": false,
             "is_outdated": false,
             "thread": []
           }
         ]
       }
     ]
   }
   ```

   Use the `thread_id` values with `comments reply` to continue discussions. If
   you are replying inside your own pending review, pass the associated
   `PRR_…` identifier with `--review-id`.

   ```sh
   gh pr-review comments reply \
     --thread-id PRRT_kwDOAAABbcdEFG12 \
     --body "Follow-up addressed in commit abc123" \
     -R owner/repo 42
   ```

5. **Submit the review (GraphQL).** Reuse the pending review `PRR_…`
   identifier when finalizing. Successful submissions emit a status-only
   payload. GraphQL-level errors are returned as structured JSON for
   troubleshooting.

   ```sh
   gh pr-review review --submit \
     --review-id PRR_kwDOAAABbcdEFG12 \
     --event REQUEST_CHANGES \
     --body "Please add tests" \
     -R owner/repo 42

   {
     "status": "Review submitted successfully"
   }
   ```

   On GraphQL errors, the command exits non-zero after emitting:

   ```json
   {
     "status": "Review submission failed",
     "errors": [
       { "message": "mutation failed", "path": ["mutation", "submitPullRequestReview"] }
     ]
   }
   ```

6. **Inspect and resolve threads (GraphQL).** Array responses are always `[]`
   when no threads match.

   ```sh
   gh pr-review threads list --unresolved --mine -R owner/repo 42

   [
     {
       "threadId": "R_ywDoABC123",
       "isResolved": false,
       "path": "internal/service.go",
       "line": 42,
       "isOutdated": false
     }
   ]
   ```

   ```sh
   gh pr-review threads resolve --thread-id R_ywDoABC123 -R owner/repo 42
   
   {
     "thread_node_id": "R_ywDoABC123",
     "is_resolved": true
   }
   ```

## Review view

`gh pr-review review view` emits a GraphQL-only snapshot of pull request
discussion. The response groups reviews → parent inline comments → thread
replies, omitting optional fields entirely instead of returning `null`.

Run it with either a combined selector or explicit flags:

```sh
gh pr-review review view -R owner/repo --pr 3
```

Install or upgrade to **v1.6.0 or newer** (GraphQL-only thread resolution and minimal comment replies):

```sh
gh extension install Agynio/gh-pr-review
# Update an existing installation
gh extension upgrade Agynio/gh-pr-review
```

### Command behavior

- Single GraphQL operation per invocation (no REST mixing).
- Includes all reviewers, review states, and threads by default.
- Replies are sorted by `created_at` ascending.
- Output exposes `author_login` only—no user objects or `html_url` fields.
- Optional fields (`body`, `submitted_at`, `line`, `thread`) are omitted when
  empty; empty reply lists render as `"thread": []`.

### Filters

| Flag | Purpose |
| --- | --- |
| `--reviewer <login>` | Only include reviews authored by `<login>` (case-insensitive). |
| `--states <list>` | Comma-separated review states (`APPROVED`, `CHANGES_REQUESTED`, `COMMENTED`, `DISMISSED`). |
| `--unresolved` | Keep only unresolved threads. |
| `--not_outdated` | Exclude threads marked as outdated. |
| `--tail <n>` | Retain only the last `n` replies per thread (0 = all). The parent inline comment is always kept; only replies are trimmed. |
| `--include-comment-node-id` | Add GraphQL comment node identifiers to parent comments and replies. |

### Examples

```sh
# Default: return all reviews, states, threads
gh pr-review review view -R owner/repo --pr 3

# Unresolved threads only
gh pr-review review view -R owner/repo --pr 3 --unresolved

# Focus changes requested from a single reviewer; keep only latest reply per thread
gh pr-review review view -R owner/repo --pr 3 --reviewer alice --states CHANGES_REQUESTED --tail 1

# Drop outdated threads and include comment node IDs
gh pr-review review view -R owner/repo --pr 3 --not_outdated --include-comment-node-id
```

### Output schema

```json
{
  "reviews": [
    {
      "id": "PRR_…",
      "state": "APPROVED|CHANGES_REQUESTED|COMMENTED|DISMISSED",
      "author_login": "…",
      "body": "…",          // omitted if empty
      "submitted_at": "…",   // omitted if absent
      "comments": [           // omitted if none
        {
          "thread_id": "PRRT_…",
          "comment_node_id": "PRRC_…",  // omitted unless requested
          "path": "…",
          "line": 21,         // omitted if null
          "author_login": "…",
          "body": "…",
          "created_at": "…",
          "is_resolved": true,
          "is_outdated": false,
          "thread": [         // replies only; sorted asc; tail applies
            {
              "comment_node_id": "PRRC_…",  // omitted unless requested
              "author_login": "…",
              "body": "…",
              "created_at": "…"
            }
          ]
        }
      ]
    }
  ]
}
```

### Replying to threads

Use the `thread_id` values surfaced in the report when replying. Provide
`--review-id` alongside `--thread-id` when continuing a pending review you own.

```sh
gh pr-review comments reply -R owner/repo --pr 3 \
  --thread-id PRRT_kwDOAAABbcdEFG12 \
  --body "Follow-up addressed in commit abc123"

gh pr-review comments reply -R owner/repo --pr 3 \
  --thread-id PRRT_kwDOAAABbcdEFG12 \
  --review-id PRR_kwDOAAABbcdEFG12 \
  --body "Reply from pending review"
```

## Backend policy

Each command binds to a single GitHub backend—there are no runtime fallbacks.

| Command | Backend | Notes |
| --- | --- | --- |
| `review --start` | GraphQL | Opens a pending review via `addPullRequestReview`. |
| `review --add-comment` | GraphQL | Requires a `PRR_…` review node ID. |
| `review view` | GraphQL | Aggregates reviews, inline comments, and replies (used for thread IDs). |
| `review --submit` | GraphQL | Finalizes a pending review via `submitPullRequestReview` using the `PRR_…` review node ID (executed through the internal `gh api graphql` wrapper). |
| `comments reply` | GraphQL | Replies via `addPullRequestReviewThreadReply`; supply `--review-id` when responding from a pending review. |
| `threads list` | GraphQL | Enumerates review threads for the pull request. |
| `threads resolve` / `unresolve` | GraphQL | Mutates thread resolution via `resolveReviewThread` / `unresolveReviewThread`; supply GraphQL thread node IDs (`PRRT_…`). |


## Additional docs

- [docs/USAGE.md](docs/USAGE.md) — Command-by-command inputs, outputs, and
  examples for v1.6.0.
- [docs/SCHEMAS.md](docs/SCHEMAS.md) — JSON schemas for each structured
  response (optional fields omitted rather than set to null).
- [docs/AGENTS.md](docs/AGENTS.md) — Agent-focused workflows, prompts, and
  best practices.

## Design notes

- Each command binds to exactly one GitHub backend: review view is
  GraphQL-only, while comment listing/replying remain GraphQL-only. Optional
  REST lookups appear only when translating legacy IDs.
- Optional fields are omitted entirely—never backfilled with empty strings or
  `null` placeholders.
- Output is optimized for headless and LLM workflows (stable ordering, minimal
  decoration, machine-friendly JSON).

## Development

Run the test suite and linters locally with cgo disabled (matching the release build):

```sh
CGO_ENABLED=0 go test ./...
CGO_ENABLED=0 golangci-lint run
```

Releases are built using the
[`cli/gh-extension-precompile`](https://github.com/cli/gh-extension-precompile)
workflow to publish binaries for macOS, Linux, and Windows.

