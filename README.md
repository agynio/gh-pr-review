# gh-pr-review
[![Agyn badge](https://agyn.io/badges/badge_dark.svg)](http://agyn.io)

`gh-pr-review` is a precompiled GitHub CLI extension that streamlines
pull request review workflows. It adds helpers for listing review comments,
replying to threads, and managing pending reviews without cloning repositories.

- [Commands](#commands)
- [Agent usage guide](#agent-usage-guide)
- [Development](#development)

## Installation

Install or upgrade directly from GitHub:

```sh
gh extension install Agyn-sandbox/gh-pr-review
# or update
gh extension upgrade Agyn-sandbox/gh-pr-review
```

## Commands

### List review comments

Fetch review comments for a specific review or the latest submission:

```sh
# Provide a review ID explicitly
gh pr-review comments --list --review-id 123456 owner/repo#42

# Resolve the latest review for the authenticated user
gh pr-review comments --list --latest -R owner/repo 42

# Filter by reviewer login
gh pr-review comments --list --latest --reviewer octocat owner/repo#42
```

The command prints JSON and supports pagination automatically.

### Reply to a review comment

```sh
gh pr-review comments reply --comment-id 987654 --body "LGTM" owner/repo#42
```

If the reply is blocked by an existing pending review, the extension
auto-submits that review and retries the reply.

### Manage pending reviews

```sh
# Start a new pending review (defaults to the head commit)
gh pr-review review --start owner/repo#42

# Add an inline comment to an existing pending review
gh pr-review review --add-comment \
  --review-id R_kwM123456789 \
  --path internal/service.go \
  --line 42 \
  --body "nit: use helper" \
  owner/repo#42

# Submit the review with a specific event
gh pr-review review --submit \
  --review-id R_kwM123456789 \
  --event REQUEST_CHANGES \
  --body "Please update tests" \
  owner/repo#42
```

### Manage review threads

List threads and filter by resolution state or participation. Output is always JSON:

```sh
# List unresolved threads you can resolve or participated in
gh pr-review threads list --unresolved --mine owner/repo#42

# Include all review threads for a pull request URL
gh pr-review threads list https://github.com/owner/repo/pull/42
```

Resolve or unresolve threads using either the thread node ID or a REST
comment identifier:

```sh
# Resolve by thread node ID
gh pr-review threads resolve --thread-id R_ywDoABC123 owner/repo#42

# Resolve by comment identifier (maps to thread automatically)
gh pr-review threads resolve --comment-id 987654 owner/repo#42

# Reopen a thread
gh pr-review threads unresolve --thread-id R_ywDoABC123 owner/repo#42
```

All commands accept `-R owner/repo`, pull request URLs, or the `owner/repo#123`
shorthand and do not require a local git checkout. Authentication and host
resolution defer to the existing `gh` CLI configuration, including `GH_HOST` for
GitHub Enterprise environments.

### Helper commands for identifiers

Emit minimal JSON for frequently needed identifiers:

```sh
# Locate the latest submitted review for a reviewer
gh pr-review review latest-id --per_page 100 --page 1 --reviewer octocat owner/repo#42

# List comment identifiers (with bodies) for a review
gh pr-review comments ids --review_id 3531807471 --limit 50 owner/repo#42

# Map a comment to its thread (or fetch by thread ID) with minimal schema
gh pr-review threads find --comment_id 2582545223 owner/repo#42
```

Outputs are pure JSON with REST/GraphQL field names and include only fields that
are present from the source APIs (no null placeholders). The `threads find`
command always emits exactly `{ "id", "isResolved" }`.

## Output conventions

All commands emit JSON aligned with GitHub REST/GraphQL schemas.

- Arrays are serialized as `[]` when no results are available.
- No plaintext is printed; even errors bubble up through the CLI.
- `gh pr-review comments reply --concise` trims the payload to `{ "id" }` while
  the default mode returns the full REST response.

## Agent usage guide

See [docs/AGENTS.md](docs/AGENTS.md) for agent-focused workflows, prompts, and
best practices when invoking `gh pr-review` from automation.

## Development

Run the test suite and linters locally with cgo disabled (matching the release build):

```sh
CGO_ENABLED=0 go test ./...
CGO_ENABLED=0 golangci-lint run
```

Releases are built using the
[`cli/gh-extension-precompile`](https://github.com/cli/gh-extension-precompile)
workflow to publish binaries for macOS, Linux, and Windows.
