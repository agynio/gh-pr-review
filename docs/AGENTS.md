# Agent Usage with `gh pr-review`

This guide provides ready-to-run prompts for scripted or agent-driven use of
`gh pr-review`. All commands emit JSON with REST/GraphQL-aligned field names and
include only values present in upstream responses (no null placeholders).
Empty collections are emitted as `[]` instead of `null`; `comments reply`
returns `{ "comment_node_id": "PRRC_…" }` for minimal responses.

## 1. Review a pull request end-to-end

```sh
# Start or resume a pending review for PR 42
gh pr-review review --start -R owner/repo 42

# Add an inline comment to the pending review
gh pr-review review --add-comment \
  --review-id PRR_kwM123456 \
  --path internal/service.go \
  --line 42 \
  --body "nit: use helper" \
  -R owner/repo 42

# Submit the review (COMMENT | APPROVE | REQUEST_CHANGES)
gh pr-review review --submit \
  --review-id PRR_kwM123456 \
  --event APPROVE \
  --body "Looks good" \
  -R owner/repo 42

# Successful submissions emit:
# { "status": "Review submitted successfully" }
#
# GraphQL errors emit:
# { "status": "Review submission failed", "errors": [ ... ] }

# Summarize reviews and collect thread IDs
gh pr-review review view --reviewer octocat --include-comment-node-id -R owner/repo 42
```

> **Note:** `--review-id` always expects the GraphQL review node ID (prefixed
> with `PRR_`) when submitting a review or adding inline comments. Pending-
> review helpers use GitHub's GraphQL API exclusively. They fail fast if the
> payload is missing required data or the authenticated viewer login cannot be
> resolved—pass `--reviewer` in that scenario.

## 2. Read and reply to inline comments

```sh
# Capture review threads (optionally include comment node IDs)
gh pr-review review view --unresolved --include-comment-node-id -R owner/repo 42

# Reply to a thread
gh pr-review comments reply \
  --thread-id <thread-id> \
  --body "Thanks for catching this" \
  -R owner/repo 42

# Reply to a pending review thread by providing the review ID
gh pr-review comments reply \
  --thread-id <thread-id> \
  --review-id <review-id> \
  --body "Submitting from pending" \
  -R owner/repo 42
```

Inspect the JSON returned by `review view`, select the desired `thread_id`
and, when needed, the `comment_node_id` to correlate individual replies.
Supply the `thread_id` (and `--review-id` when replying inside a pending
review) when invoking `comments reply`.

## 3. Resolve or reopen discussion threads

```sh
# Resolve the thread using its GraphQL node ID
gh pr-review threads resolve --thread-id <thread-id> -R owner/repo 42

# Reopen the thread if needed
gh pr-review threads unresolve --thread-id <thread-id> -R owner/repo 42
```

Use `review view` (optionally with `--include-comment-node-id`) or
`threads list` to gather `thread_id` values before mutating them. Thread
mutations return minimal JSON of the form
`{ "thread_node_id": "PRRT_…", "is_resolved": true|false }`.
