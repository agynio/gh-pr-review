# gh-pr-review
[![Agyn badge](https://agyn.io/badges/badge_dark.svg)](http://agyn.io)

`gh-pr-review` is a precompiled GitHub CLI extension for high-signal pull
request reviews. It manages pending GraphQL reviews, surfaces REST identifiers,
and resolves threads without cloning repositories.

- [Quickstart](#quickstart)
- [Review report](#review-report)
- [Backend policy](#backend-policy)
- [Additional docs](#additional-docs)

## Quickstart

The quickest path from opening a pending review to resolving threads:

1. **Install or upgrade the extension.**

   ```sh
   gh extension install Agyn-sandbox/gh-pr-review
   # Update an existing installation
   gh extension upgrade Agyn-sandbox/gh-pr-review
   ```

   > v1.3.3 and newer ship precompiled linux-arm64 binaries in addition to the
   > existing macOS, Windows, and Linux targets.

2. **Start a pending review (GraphQL).** Capture the returned `id` (GraphQL
   node) and optional `database_id`.

   ```sh
   gh pr-review review --start owner/repo#42

   {
     "id": "PRR_kwDOAAABbcdEFG12",
     "state": "PENDING",
     "database_id": 3531807471,
     "html_url": "https://github.com/owner/repo/pull/42#pullrequestreview-3531807471"
   }
   ```

3. **Add inline comments with the pending review ID (GraphQL).** The
   `review --add-comment` command fails fast if you supply a numeric ID instead
   of the required `PRR_…` GraphQL identifier.

   ```sh
   gh pr-review review --add-comment \
     --review-id PRR_kwDOAAABbcdEFG12 \
     --path internal/service.go \
     --line 42 \
     --body "nit: use helper" \
     owner/repo#42

   {
     "id": "PRRT_kwDOAAABbcdEFG12",
     "path": "internal/service.go",
     "is_outdated": false,
     "line": 42
   }
   ```

4. **Inspect review threads (GraphQL).** `review report` surfaces pending
   review summaries and inline comment metadata (including numeric comment IDs)
   in a single payload.

   ```sh
   gh pr-review review report --reviewer octocat owner/repo#42

   {
     "reviews": [
       {
         "id": "PRR_kwDOAAABbcdEFG12",
         "state": "COMMENTED",
         "comments": [
           {
             "id": 3531807471,
             "path": "internal/service.go",
             "body": "nit: prefer helper"
           }
         ]
       }
     ]
   }
   ```

   Use the numeric `id` values with `comments reply` to continue threads:

   ```sh
   gh pr-review comments reply \
     --comment-id 3531807471 \
     --body "Follow-up addressed in commit abc123" \
     owner/repo#42
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
     owner/repo#42

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
   gh pr-review threads list --unresolved --mine owner/repo#42

   [
     {
       "threadId": "R_ywDoABC123",
       "isResolved": false,
       "path": "internal/service.go",
       "line": 42,
       "isOutdated": false
     }
   ]

   gh pr-review threads resolve --thread-id R_ywDoABC123 owner/repo#42

   {
     "threadId": "R_ywDoABC123",
     "isResolved": true,
     "changed": true
   }
   ```

## Review report

`gh pr-review review report` emits a GraphQL-only snapshot of pull request
discussion. The response groups reviews → parent inline comments → thread
replies, omitting optional fields entirely instead of returning `null`.

Run it with either a combined selector or explicit flags:

```sh
gh pr-review review report -R owner/repo --pr 3
```

Install or upgrade to **v1.3.3 or newer** (adds linux-arm64 precompiled
artifacts):

```sh
gh extension install Agyn-sandbox/gh-pr-review
# Update an existing installation
gh extension upgrade Agyn-sandbox/gh-pr-review
```

### Command behavior

- Single GraphQL operation per invocation (no REST mixing).
- Includes all reviewers, review states, and threads by default.
- Replies are sorted by `created_at` ascending.
- Output exposes `author_login` only—no user objects or `html_url` fields.
- Optional fields (`body`, `submitted_at`, `line`, `in_reply_to_id`,
  `comments`) are omitted when empty; empty reply lists render as
  `"thread": []`.

### Filters

| Flag | Purpose |
| --- | --- |
| `--reviewer <login>` | Only include reviews authored by `<login>` (case-insensitive). |
| `--states <list>` | Comma-separated review states (`APPROVED`, `CHANGES_REQUESTED`, `COMMENTED`, `DISMISSED`). |
| `--unresolved` | Keep only unresolved threads. |
| `--not_outdated` | Exclude threads marked as outdated. |
| `--tail <n>` | Retain only the last `n` replies per thread (0 = all). The parent inline comment is always kept; only replies are trimmed. |

### Examples

```sh
# Default: return all reviews, states, threads
gh pr-review review report -R owner/repo --pr 3

# Unresolved threads only
gh pr-review review report -R owner/repo --pr 3 --unresolved

# Focus changes requested from a single reviewer; keep only latest reply per thread
gh pr-review review report -R owner/repo --pr 3 --reviewer alice --states CHANGES_REQUESTED --tail 1

# Drop outdated threads
gh pr-review review report -R owner/repo --pr 3 --not_outdated
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
          "id": 258…,
          "path": "…",
          "line": 21,         // omitted if null
          "author_login": "…",
          "body": "…",
          "created_at": "…",
          "is_resolved": true,
          "is_outdated": false,
          "thread": [         // replies only; sorted asc; tail applies
            {
              "id": 259…,
              "in_reply_to_id": 258…,
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

### Replying by ID

Use the numeric `id` values surfaced in the report when replying by comment ID:

```sh
gh pr-review comments reply -R owner/repo --pr 3 \
  --comment-id 3531807472 \
  --body "Follow-up addressed in commit abc123"
```

## Backend policy

Each command binds to a single GitHub backend—there are no runtime fallbacks.

| Command | Backend | Notes |
| --- | --- | --- |
| `review --start` | GraphQL | Opens a pending review via `addPullRequestReview`. |
| `review --add-comment` | GraphQL | Requires a `PRR_…` review node ID. |
| `review report` | GraphQL | Aggregates reviews, inline comments, and replies (used for comment IDs). |
| `review --submit` | REST | Accepts only numeric review IDs and posts `/reviews/{id}/events`. |
| `comments reply` | REST (GraphQL only locates pending reviews before REST auto-submission) | Replies via REST; when GitHub blocks the reply due to a pending review, the extension discovers pending review IDs via GraphQL and submits them with REST before retrying. |
| `threads list` | GraphQL | Enumerates review threads for the pull request. |
| `threads resolve` / `unresolve` | GraphQL (+ REST when mapping `--comment-id`) | Mutates thread resolution with GraphQL; a REST lookup translates numeric comment IDs to node IDs. |
| `threads find` | GraphQL (+ REST when mapping `--comment_id`) | Returns `{ "id", "isResolved" }`. |


## Additional docs

- [docs/USAGE.md](docs/USAGE.md) — Command-by-command inputs, outputs, and
  examples for v1.2.1.
- [docs/SCHEMAS.md](docs/SCHEMAS.md) — JSON schemas for each structured
  response (optional fields omitted rather than set to null).
- [docs/AGENTS.md](docs/AGENTS.md) — Agent-focused workflows, prompts, and
  best practices.

## Design notes

- Each command binds to exactly one GitHub backend: review report is
  GraphQL-only, while comment listing/replying remain REST-only.
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
