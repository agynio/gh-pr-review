# gh-pr-review
[![Agyn badge](https://agyn.io/badges/badge_dark.svg)](http://agyn.io)

`gh-pr-review` is a GitHub CLI extension that finally brings **inline PR review comments** to the terminal.  
GitHub’s built-in `gh` tool does *not* show inline comments or review threads — but this extension does.

With `gh-pr-review`, you can:

- View complete **inline review threads** with file and line context  
- See **unresolved comments** during code review  
- Reply to inline comments directly from the terminal  
- Resolve review threads programmatically  
- Export structured output ideal for **LLMs and automated PR review agents**

Designed for developers, DevOps teams, and AI systems that need **full pull request review context**, not just top-level comments.

**Blog post:** [gh-pr-review: LLM-friendly PR review workflows in your CLI](https://agyn.io/blog/gh-pr-review-cli-agent-workflows) — explains the motivation, design principles, and CLI + JSON output examples.  

- [Quickstart](#quickstart)
- [Review view](#review-view)
- [Backend policy](#backend-policy)
- [Additional docs](#additional-docs)
- [Design for LLMs & Automated Agents](#design-for-llms--automated-agents)




## Quickstart

The quickest path from opening a pending review to resolving threads:

1. **Install or upgrade the extension.**

   ```sh
   gh extension install agynio/gh-pr-review
   # Update an existing installation
   gh extension upgrade agynio/gh-pr-review
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
         "author_login": "octocat",
         "comments": [
           {
             "thread_id": "PRRT_kwDOAAABbcdEFG12",
             "path": "internal/service.go",
             "author_login": "octocat",
             "body": "nit: prefer helper",
             "created_at": "2024-05-25T18:21:37Z",
             "is_resolved": false,
             "is_outdated": false,
             "thread_comments": []
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
gh extension install agynio/gh-pr-review
# Update an existing installation
gh extension upgrade agynio/gh-pr-review
```

### Command behavior

- Single GraphQL operation per invocation (no REST mixing).
- Includes all reviewers, review states, and threads by default.
- Replies are sorted by `created_at` ascending.
- Output exposes `author_login` only—no user objects or `html_url` fields.
- Optional fields (`body`, `submitted_at`, `line`) are omitted when empty.
- `comments` is omitted entirely when a review has no inline comments.
- `thread_comments` is required for every inline comment and always present
  (empty arrays indicate no replies).

For the full canonical response structure, see docs/SCHEMAS.md.

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

### Replying to threads

Use the `thread_id` values surfaced in the report when replying.

```sh
gh pr-review comments reply 3 -R owner/repo \
  --thread-id PRRT_kwDOAAABbcdEFG12 \
  --body "Follow-up addressed in commit abc123"

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

## Design for LLMs & Automated Agents

`gh-pr-review` is designed to give LLMs and agents the **exact PR review context they need** — without the noisy, multi-step GitHub API workflow.

### Why it's LLM-friendly

- **Replaces multi-call API chains with one command**  
  Instead of calling `list reviews → list thread comments → list comments`,  
  a single `gh pr-review review view` command returns the entire, assembled review structure.

- **Deterministic, stable output**  
  Consistent formatting, stable ordering, and predictable field names make parsing reliable for agents.

- **Compact, meaningful JSON**  
  Only essential fields are returned. Low-signal metadata (URLs, hashes, unused fields) is stripped out to reduce token usage.

- **Pre-joined review threads**  
  Threads come fully reconstructed with inline context — no need for agents to merge comments manually.

- **Server-side filters for token efficiency**  
  Options like `--unresolved` and `--tail` help reduce payload size and keep inputs affordable for LLMs.


> “A good tool definition should define a clear, narrow purpose, return exactly the meaningful context the agent needs, and avoid burdening the model with low-signal intermediate results.”


## Development

Run the test suite and linters locally with cgo disabled (matching the release build):

```sh
CGO_ENABLED=0 go test ./...
CGO_ENABLED=0 golangci-lint run
```

Releases are built using the
[`cli/gh-extension-precompile`](https://github.com/cli/gh-extension-precompile)
workflow to publish binaries for macOS, Linux, and Windows.
