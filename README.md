# gh-pr-review

`gh-pr-review` is a precompiled GitHub CLI extension that streamlines
pull request review workflows. It adds helpers for listing review comments,
replying to threads, and managing pending reviews without cloning repositories.

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

All commands accept `-R owner/repo`, pull request URLs, or the `owner/repo#123`
shorthand and do not require a local git checkout. Authentication and host
resolution defer to the existing `gh` CLI configuration, including `GH_HOST` for
GitHub Enterprise environments.

## Development

Run the test suite and linters locally with cgo disabled (matching the release build):

```sh
CGO_ENABLED=0 go test ./...
CGO_ENABLED=0 golangci-lint run
```

Releases are built using the
[`cli/gh-extension-precompile`](https://github.com/cli/gh-extension-precompile)
workflow to publish binaries for macOS, Linux, and Windows.
