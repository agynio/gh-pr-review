# gh-pr-review
[![Agyn badge](https://agyn.io/badges/badge_dark.svg)](http://agyn.io)

`gh-pr-review` is a precompiled GitHub CLI extension for high-signal pull
request reviews. It manages pending GraphQL reviews, surfaces REST identifiers,
and resolves threads without cloning repositories.

- [Quickstart](#quickstart)
- [Backend policy](#backend-policy)
- [Installation & upgrade](#installation--upgrade)
- [Command examples](#command-examples)
- [Output conventions](#output-conventions)
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

## Installation & upgrade

### Precompiled binaries (macOS, Linux x64, Windows x64)

```sh
gh extension install Agyn-sandbox/gh-pr-review
# or upgrade in-place
gh extension upgrade Agyn-sandbox/gh-pr-review
```

Verify the installation with:

```sh
gh pr-review --version
```

### Build from source (including linux-arm64)

When a release does not include a linux-arm64 binary, compile from source with
Go 1.22 or newer:

```sh
git clone https://github.com/Agyn-sandbox/gh-pr-review.git
cd gh-pr-review
GOOS=linux GOARCH=arm64 go build -trimpath -ldflags "-s -w" -o gh-pr-review .
gh extension install . --force
```

The final command installs (or reinstalls) the locally built binary by
symlinking it into the GitHub CLI extension directory. Re-run `gh pr-review
--version` after upgrades to confirm that GitHub CLI sees the freshly built
binary.

## Command examples

The snippets below highlight the most common review workflows. Each example
targets `owner/repo#42`, but pull request URLs and `-R owner/repo 42` are also
accepted.

### Manage pending reviews (GraphQL)

Start or resume a review and add inline comments with GraphQL-only helpers:

```sh
# Start a pending review and capture the PRR node id
gh pr-review review --start owner/repo#42

# Append an inline comment to the same pending review
gh pr-review review --add-comment \
  --review-id PRR_kwDOAAABbcdEFG12 \
  --path internal/service.go \
  --line 42 \
  --body "nit: prefer helper" \
  owner/repo#42

<<<<<<< HEAD
# Locate the latest pending review for a specific reviewer
gh pr-review review pending-id --reviewer octocat owner/repo#42
```

### Submit the review (REST only)

`review --submit` enforces numeric review identifiers and emits the normalized
review state. Swap the event to produce `COMMENT`, `APPROVE`, or
`REQUEST_CHANGES` submissions:

```sh
# Leave a general comment on the review
gh pr-review review --submit \
  --review-id 3531807471 \
  --event COMMENT \
  --body "Nice refactor" \
  owner/repo#42

# Approve without a body
gh pr-review review --submit \
  --review-id 3531807471 \
  --event APPROVE \
  owner/repo#42

# Request changes with guidance
=======
# Submit the review with a specific event (GraphQL review ID required)
>>>>>>> 7b50bf4 (feat(review): require graphql submit)
gh pr-review review --submit \
  --review-id PRR_kwM123456 \
  --event REQUEST_CHANGES \
  --body "Missing negative tests" \
  owner/repo#42
```

Each invocation returns the same `ReviewState` object documented in
[docs/SCHEMAS.md](docs/SCHEMAS.md), with optional fields omitted when empty.

### Comment helpers (REST)

Use REST-focused helpers to map identifiers and post replies:

```sh
# Emit minimal comment metadata (arrays default to [])
gh pr-review comments ids --review_id 3531807471 --limit 20 owner/repo#42

# Reply with the full REST payload
gh pr-review comments reply \
  --comment-id 987654321 \
  --body "Thanks for catching this" \
  owner/repo#42

<<<<<<< HEAD
# Emit only the reply id when you do not need the full payload
gh pr-review comments reply \
  --comment-id 987654321 \
  --body "Ack" \
  --concise \
  owner/repo#42
=======
The `--review-id` flag always expects the GraphQL review node ID (prefixed with
`PRR_`), including when submitting a review or adding inline comments. Use
`gh pr-review review pending-id` or `gh pr-review review latest-id` to locate
the desired identifier.
>>>>>>> 7b50bf4 (feat(review): require graphql submit)
```

When GitHub blocks a reply because you have an outstanding pending review, the
extension locates those pending review identifiers with GraphQL and then
auto-submits them via REST using a `COMMENT` event before retrying the reply.

### Thread helpers (GraphQL with optional REST lookups)

```sh
# List unresolved threads you can act on
gh pr-review threads list --unresolved --mine owner/repo#42

# Resolve a thread by GraphQL id
gh pr-review threads resolve --thread-id R_ywDoABC123 owner/repo#42

# Resolve by database comment id (REST lookup + GraphQL mutation)
gh pr-review threads resolve --comment-id 2582545223 owner/repo#42

# Reopen a thread
gh pr-review threads unresolve --thread-id R_ywDoABC123 owner/repo#42
```

`threads find` returns exactly `{ "id", "isResolved" }` to connect REST
comment identifiers to GraphQL thread nodes.

## Output conventions

All commands emit JSON aligned with GitHub REST/GraphQL schemas.

- Arrays serialize as `[]` when empty—never `null`.
- Optional fields are omitted entirely instead of emitting explicit `null`.
- `gh pr-review comments reply --concise` trims the payload to `{ "id" }`; the
  default reply returns the full REST comment object.
- Errors bubble up through the GitHub CLI without plaintext side channels.

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
