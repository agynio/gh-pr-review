# gh-pr-review
[![Agyn badge](https://agyn.io/badges/badge_dark.svg)](http://agyn.io)

`gh-pr-review` is a precompiled GitHub CLI extension for high-signal pull
request reviews. It manages pending GraphQL reviews, surfaces REST identifiers,
and resolves threads without cloning repositories.

- [Quickstart](#quickstart)
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

4. **Locate the review identifier (GraphQL).** `review pending-id` reads
   GraphQL only; when the authenticated viewer login cannot be resolved, the
   command errors with guidance to pass `--reviewer`. The response includes the
   GraphQL review node ID and matching numeric database ID.

   ```sh
   gh pr-review review pending-id --reviewer octocat owner/repo#42

   {
     "id": "PRR_kwDOAAABbcdEFG12",
     "database_id": 3531807471,
     "state": "PENDING",
     "html_url": "https://github.com/owner/repo/pull/42#pullrequestreview-3531807471",
     "user": { "login": "octocat", "id": 6752317 }
   }
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

## Backend policy

Each command binds to a single GitHub backend—there are no runtime fallbacks.

| Command | Backend | Notes |
| --- | --- | --- |
| `review --start` | GraphQL | Opens a pending review via `addPullRequestReview`. |
| `review --add-comment` | GraphQL | Requires a `PRR_…` review node ID. |
| `review pending-id` | GraphQL | Fails with guidance if the viewer login is unavailable; pass `--reviewer`. |
| `review latest-id` | REST | Walks `/pulls/{number}/reviews` to find the latest submitted review. |
| `review --submit` | REST | Accepts only numeric review IDs and posts `/reviews/{id}/events`. |
| `comments ids` | REST | Pages `/reviews/{id}/comments`; optional reviewer resolution uses REST only. |
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

## Development

Run the test suite and linters locally with cgo disabled (matching the release build):

```sh
CGO_ENABLED=0 go test ./...
CGO_ENABLED=0 golangci-lint run
```

Releases are built using the
[`cli/gh-extension-precompile`](https://github.com/cli/gh-extension-precompile)
workflow to publish binaries for macOS, Linux, and Windows.
