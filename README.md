# gh-pr-review

A GitHub CLI extension for inline PR review comments and thread inspection in the terminal.

This fork of [agynio/gh-pr-review](https://github.com/agynio/gh-pr-review) adds features for developers, DevOps teams, and AI systems that need complete pull request review context.

**Blog post:** [gh-pr-review: LLM-friendly PR review workflows in your CLI](https://agyn.io/blog/gh-pr-review-cli-agent-workflows)

## Features

GitHub's built-in `gh` tool does not show inline comments, review threads, or thread grouping. This extension adds:

- View inline review threads with file context
- Reply to comments from the terminal
- Resolve threads programmatically
- Group and inspect threads with `threads view`
- Export structured JSON for LLMs and automation

## Installation

```sh
gh extension install v2nic/gh-pr-review
gh extension upgrade v2nic/gh-pr-review  # Update existing installation
```

### Agent Skill

Register with your AI agent using the [SKILL.md](skills/gh-pr-review/SKILL.md) definition:

```bash
npx skills add v2nic/gh-pr-review
```

## Commands

| Command                         | Description                                           |
| ------------------------------- | ----------------------------------------------------- |
| `review --start`                | Opens a pending review                                |
| `review --add-comment`          | Adds inline comment (requires `PRR_…` review node ID) |
| `review --edit-comment`         | Updates a comment in a pending review                 |
| `review view`                   | Aggregates reviews, inline comments, and replies      |
| `review --submit`               | Finalizes a pending review                            |
| `comments reply`                | Replies to a review thread                            |
| `threads list`                  | Lists review threads for the pull request             |
| `threads resolve` / `unresolve` | Resolves or unresolves review threads                 |

### Filters

| Flag                        | Purpose                                                                           |
| --------------------------- | --------------------------------------------------------------------------------- |
| `--reviewer <login>`        | Only include reviews by specified user (case-insensitive)                         |
| `--states <list>`           | Comma-separated states: `APPROVED`, `CHANGES_REQUESTED`, `COMMENTED`, `DISMISSED` |
| `--unresolved`              | Keep only unresolved threads                                                      |
| `--not_outdated`            | Exclude threads marked as outdated                                                |
| `--tail <n>`                | Retain only last `n` replies per thread (0 = all)                                 |
| `--include-comment-node-id` | Add comment node identifiers to parent comments and replies                       |

**Note**: Commands accepting `--body` also support `--body-file <path>` to read from a file. Use `--body-file -` to read from stdin. These flags are mutually exclusive.

See [skills/references/USAGE.md](skills/references/USAGE.md) for detailed usage. See [docs/SCHEMAS.md](docs/SCHEMAS.md) for JSON response schemas.

## Usage

Basic workflow:

1. Start a review: `gh pr-review review --start`
2. Add comments: `gh pr-review review --add-comment --review-id <ID> --path <file> --line <N> --body "<msg>"`
3. Submit review: `gh pr-review review --submit --review-id <ID> --event APPROVE`
4. Resolve threads: `gh pr-review threads resolve --thread-id <ID>`

When inside a git repository, `-R owner/repo` and PR number are inferred automatically.

### Viewing Reviews

`gh pr-review review view` shows all reviews, inline comments, and replies:

```sh
gh pr-review review view -R owner/repo --pr 3
```

Common filters:
- `--unresolved` — Show only unresolved threads
- `--reviewer <user>` — Filter by reviewer
- `--states APPROVED,CHANGES_REQUESTED` — Filter by review state

Reply to threads using the `thread_id` from the view output:

```sh
gh pr-review comments reply --thread-id <ID> --body "<msg>"
```

### Managing Threads

List and resolve threads:

```sh
# List unresolved threads
gh pr-review threads list --unresolved

# Resolve a thread
gh pr-review threads resolve --thread-id <ID>
```

See [skills/references/USAGE.md](skills/references/USAGE.md) for detailed usage examples.

## Development

Run tests and linters locally with CGO disabled (matching release build):

```sh
CGO_ENABLED=0 go test ./...
CGO_ENABLED=0 golangci-lint run
```

Releases use the [`cli/gh-extension-precompile`](https://github.com/cli/gh-extension-precompile) workflow to publish binaries for macOS, Linux, and Windows.
