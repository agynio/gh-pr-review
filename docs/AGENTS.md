# Agent Usage with `gh pr-review`

This guide provides ready-to-run prompts for scripted or agent-driven use of
`gh pr-review`. All commands emit JSON with REST/GraphQL-aligned field names and
include only values present in upstream responses (no null placeholders).
Empty collections are emitted as `[]` instead of `null`, and the
`comments reply --concise` mode returns `{ "id" }` when you only need the
database identifier.

## 1. Review a pull request end-to-end

```sh
# Start or resume a pending review for PR 42
gh pr-review review --start owner/repo#42

# Add an inline comment to the pending review
gh pr-review review --add-comment \
  --review-id PRR_kwM123456 \
  --path internal/service.go \
  --line 42 \
  --body "nit: use helper" \
  owner/repo#42

# Submit the review (COMMENT | APPROVE | REQUEST_CHANGES)
gh pr-review review --submit \
  --review-id PRR_kwM123456 \
  --event APPROVE \
  --body "Looks good" \
  owner/repo#42

# Successful submissions emit:
# { "status": "Review submitted successfully" }
#
# GraphQL errors emit:
# { "status": "Review submission failed", "errors": [ ... ] }

# Fetch the latest pending review identifier (GraphQL only)
gh pr-review review pending-id --reviewer octocat owner/repo#42
```

> **Note:** `--review-id` always expects the GraphQL review node ID (prefixed
> with `PRR_`) when submitting a review or adding inline comments. Pending-
> review helpers use GitHub's GraphQL API exclusively. They fail fast if the
> payload is missing required data or the authenticated viewer login cannot be
> resolved—pass `--reviewer` in that scenario.

## 2. Read and reply to inline comments

```sh
# List comment identifiers (IDs + text) for a specific review
gh pr-review comments ids --review_id 3531807471 --limit 20 owner/repo#42

# Reply to a comment by database identifier
gh pr-review comments reply \
  --comment-id <comment-id> \
  --body "Thanks for catching this" \
  owner/repo#42
```

Inspect the JSON returned by `comments ids`, select the desired `id`, and supply
that value as `<comment-id>` when invoking `comments reply`.

## 3. Resolve or reopen discussion threads

```sh
# Locate the thread for a specific comment; emits { "id", "isResolved" }
gh pr-review threads find --comment_id 2582545223 owner/repo#42

# Resolve the thread
gh pr-review threads resolve --thread-id <thread-id> owner/repo#42

# Reopen the thread if needed
gh pr-review threads unresolve --thread-id <thread-id> owner/repo#42
```

Take the `id` returned by `threads find` and reuse it as `<thread-id>` with
`threads resolve` or `threads unresolve`. Responses remain JSON-only with
GitHub-aligned field names and include only fields surfaced by the upstream
APIs. Threads can also be resolved directly via `--comment-id` if preferred.
