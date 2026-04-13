# gh-pr-review
[![Agyn badge](https://agyn.io/badges/badge_dark.svg)](http://agyn.io)

`gh-pr-review` is a GitHub CLI extension that brings **inline PR review threads** to the terminal — with filtering, bulk resolution, and structured JSON output designed for agents.

GitHub's built-in `gh` tool does *not* expose inline review threads or let you resolve them programmatically. This extension does, and it goes further: it was built under real agent-scale load — 178 review threads across CodeRabbit, Codex, and human reviewers on a single PR — and every flag was added to solve a real triage problem.

**Blog post:** [gh-pr-review: LLM-friendly PR review workflows in your CLI](https://agyn.io/blog/gh-pr-review-cli-agent-workflows)

---

- [Quickstart](#quickstart)
- [Origin Story](#origin-story)
- [Agent Triage Workflow](#agent-triage-workflow)
- [Command Reference](#command-reference)
  - [threads list](#threads-list)
  - [threads resolve](#threads-resolve)
  - [threads resolve-all](#threads-resolve-all)
  - [review view](#review-view)
  - [review start / add-comment / submit](#review-start--add-comment--submit)
  - [comments reply](#comments-reply)
- [Git Context Inference](#git-context-inference)
- [Backend Policy](#backend-policy)
- [Design for LLMs & Automated Agents](#design-for-llms--automated-agents)
- [Additional Docs](#additional-docs)
- [Using as a Skill](#using-as-a-skill)
- [Development](#development)

---

## Origin Story

This tool was born from a real pain point on `nightfox-agent` PR #18: **178 review threads** from three automated reviewers (CodeRabbit, Codex, and humans). The workflow before this extension was hand-rolled GraphQL mutations, Python scripts, and purely manual triage. After: single CLI commands that an agent can compose.

GitHub has had open requests for this capability since at least [cli/cli#359](https://github.com/cli/cli/issues/359) (6+ years) and [cli/cli#12419](https://github.com/cli/cli/issues/12419). `gh-pr-review` fills that gap today.

---

## Quickstart

Install or upgrade:

```sh
gh extension install agynio/gh-pr-review
# Update an existing installation
gh extension upgrade agynio/gh-pr-review
```

List unresolved threads on PR #18:

```sh
gh pr-review threads list 18 -R owner/repo --unresolved
```

Resolve a thread with a commit link:

```sh
gh pr-review threads resolve 18 -R owner/repo --thread-id PRRT_xxx --commit HEAD
```

Resolve every unresolved thread at once:

```sh
gh pr-review threads resolve-all 18 -R owner/repo --commit abc1234
```

---

## Agent Triage Workflow

This is the end-to-end workflow an agent uses when facing a PR with hundreds of review threads from multiple automated reviewers.

```sh
# 1. Get the lay of the land
gh pr-review threads list 18 -R owner/repo --unresolved
# -> 29 unresolved threads

# 2. Triage by reviewer
gh pr-review threads list 18 -R owner/repo --author rabbit --unresolved
# -> 27 from CodeRabbit

gh pr-review threads list 18 -R owner/repo --author codex --unresolved
# -> 2 from Codex (matches chatgpt-codex-connector)

# 3. Fix the Codex findings, commit
git commit -m "fix: lazy logger init + cross-platform path containment"

# 4. Resolve with full commit traceability (reply then resolve, atomic)
gh pr-review threads resolve 18 -R owner/repo --thread-id PRRT_xxx --commit HEAD
# -> {"thread_node_id":"PRRT_xxx","is_resolved":true,"reply_body":"Addressed in [`bbd60fc`](https://github.com/owner/repo/commit/bbd60fc)"}

# 5. Bulk-resolve all CodeRabbit threads once you've addressed them
gh pr-review threads resolve-all 18 -R owner/repo --author rabbit --commit HEAD

# 6. Check what's new since your last push
gh pr-review threads list 18 -R owner/repo --since 2026-04-05T20:00:00Z

# 7. Pipe unresolved IDs to xargs for custom bulk operations
gh pr-review threads list 18 -R owner/repo --unresolved -o ids | \
  xargs -I{} gh pr-review threads resolve 18 -R owner/repo --thread-id {} --commit HEAD
```

---

## Command Reference

### threads list

List review threads on a pull request with rich filtering.

```
threads list [<number>] [flags]

Flags:
  --author <login>         Filter threads by comment author (substring, case-insensitive)
  --unresolved             Show only unresolved threads
  --include-resolved       Show resolved threads alongside unresolved
  --since <RFC3339>        Filter to threads updated after this timestamp
  -o, --output <mode>      Output mode: "json" (default) or "ids" (bare thread IDs, one per line)
  -R, --repo <owner/repo>  Repository (inferred from git context if omitted)
  --pr <number>            PR number (inferred from git context if omitted)
```

**Examples:**

```sh
# All unresolved threads
gh pr-review threads list 18 -R owner/repo --unresolved

# Filter to CodeRabbit findings only (substring match, case-insensitive)
gh pr-review threads list 18 -R owner/repo --author rabbit
# -> 163 threads

# Filter to Codex findings (matches chatgpt-codex-connector)
gh pr-review threads list 18 -R owner/repo --author codex
# -> 15 threads

# Threads updated after a specific timestamp
gh pr-review threads list 18 -R owner/repo --since 2026-04-05T17:00:00Z

# Bare thread IDs for piping (one per line)
gh pr-review threads list 18 -R owner/repo --unresolved -o ids
# PRRT_kwDORiDhb8547bD4
# PRRT_kwDORiDhb8547bD5

# Inside a repo with an open PR -- no -R or PR number needed
gh pr-review threads list
```

**Sample JSON output:**

```json
[
  {
    "threadId": "PRRT_kwDORiDhb8547bD4",
    "isResolved": false,
    "path": "internal/service.go",
    "line": 42,
    "isOutdated": false,
    "body": "nit: prefer helper",
    "author": "coderabbitai"
  }
]
```

---

### threads resolve

Resolve a single thread, optionally posting a commit-linked reply first.

```
threads resolve [<number>] [flags]

Flags:
  --thread-id <id>         GraphQL thread node ID (PRRT_...) -- required
  --commit <ref>           Post "Addressed in [<sha>](link)" reply before resolving.
                           Accepts raw SHA or any symbolic ref (HEAD, branch name).
                           Reply failure prevents resolution (atomic).
  -R, --repo <owner/repo>  Repository (inferred from git context if omitted)
  --pr <number>            PR number (inferred from git context if omitted)
```

**Examples:**

```sh
# Resolve with clickable commit link (HEAD resolved to short SHA automatically)
gh pr-review threads resolve 18 -R owner/repo --thread-id PRRT_xxx --commit HEAD

# Output:
# {"thread_node_id":"PRRT_xxx","is_resolved":true,"reply_body":"Addressed in [`bbd60fc`](https://github.com/owner/repo/commit/bbd60fc)"}

# Resolve with explicit SHA
gh pr-review threads resolve 18 -R owner/repo --thread-id PRRT_xxx --commit abc1234

# Resolve without a reply
gh pr-review threads resolve 18 -R owner/repo --thread-id PRRT_xxx
```

The `--commit` flag uses symbolic ref resolution: pass `HEAD`, a branch name, or any `git rev-parse`-compatible ref and it resolves to the short SHA before posting. The reply is posted first; if it fails, the thread is **not** resolved.

---

### threads resolve-all

Bulk-resolve multiple threads in one command, with optional author and commit filters.

```
threads resolve-all [<number>] [flags]

Flags:
  --author <login>         Resolve only threads matching this author (substring, case-insensitive)
  --commit <ref>           Post commit-linked reply before each resolution
  --include-resolved       Include already-resolved threads (re-resolve them)
  -R, --repo <owner/repo>  Repository (inferred from git context if omitted)
  --pr <number>            PR number (inferred from git context if omitted)
```

**Examples:**

```sh
# Resolve all unresolved threads with commit traceability
gh pr-review threads resolve-all 18 -R owner/repo --commit abc1234

# Only resolve CodeRabbit threads
gh pr-review threads resolve-all 18 -R owner/repo --author rabbit --commit abc1234

# Use HEAD (auto-resolved to short SHA)
gh pr-review threads resolve-all 18 -R owner/repo --commit HEAD

# Include already-resolved threads
gh pr-review threads resolve-all 18 -R owner/repo --include-resolved
```

Each thread is resolved independently. Output is a JSON array of per-thread results.

---

### review view

Get a full GraphQL snapshot of all PR reviews grouped by reviewer, with inline threads and replies.

```
review view [<number>] [flags]

Flags:
  --reviewer <login>              Only include reviews by this author
  --states <list>                 Comma-separated: APPROVED, CHANGES_REQUESTED, COMMENTED, DISMISSED
  --unresolved                    Keep only unresolved threads
  --not_outdated                  Exclude outdated threads
  --tail <n>                      Keep only last n replies per thread (0 = all)
  --include-comment-node-id       Add GraphQL comment node IDs to output
  --include-resolved              Include resolved threads alongside unresolved
  -R, --repo <owner/repo>         Repository (inferred from git context if omitted)
  --pr <number>                   PR number (inferred from git context if omitted)
```

**Examples:**

```sh
# All reviews and threads
gh pr-review review view -R owner/repo --pr 18

# Include resolved threads for audit
gh pr-review review view 18 -R owner/repo --include-resolved

# Focus on one reviewer's change requests, last reply only
gh pr-review review view -R owner/repo --pr 18 --reviewer alice --states CHANGES_REQUESTED --tail 1

# Drop outdated threads
gh pr-review review view -R owner/repo --pr 18 --not_outdated
```

**Sample output:**

```json
{
  "reviews": [
    {
      "id": "PRR_kwDOAAABbcdEFG12",
      "state": "COMMENTED",
      "author_login": "coderabbitai",
      "comments": [
        {
          "thread_id": "PRRT_kwDOAAABbcdEFG12",
          "path": "internal/service.go",
          "author_login": "coderabbitai",
          "body": "nit: prefer helper",
          "created_at": "2026-04-05T18:21:37Z",
          "is_resolved": false,
          "is_outdated": false,
          "thread_comments": []
        }
      ]
    }
  ]
}
```

---

### review start / add-comment / submit

Start a pending review, add inline comments, and submit.

```sh
# Start a pending review
gh pr-review review --start -R owner/repo 42
# -> {"id": "PRR_kwDOAAABbcdEFG12", "state": "PENDING"}

# Add an inline comment to the pending review
gh pr-review review --add-comment \
  --review-id PRR_kwDOAAABbcdEFG12 \
  --path internal/service.go \
  --line 42 \
  --body "nit: use helper" \
  -R owner/repo 42

# Submit the review
gh pr-review review --submit \
  --review-id PRR_kwDOAAABbcdEFG12 \
  --event REQUEST_CHANGES \
  --body "Please add tests" \
  -R owner/repo 42
# -> {"status": "Review submitted successfully"}
```

---

### comments reply

Reply to a specific thread.

```sh
gh pr-review comments reply 42 -R owner/repo \
  --thread-id PRRT_kwDOAAABbcdEFG12 \
  --body "Addressed in commit abc123"
```

Pass `--review-id PRR_...` when replying from inside a pending review.

---

## Git Context Inference

When run inside a git repository on a branch with an open PR, `gh-pr-review` infers the repository (`-R`) and PR number automatically from the git remote and current branch. You only need to specify them explicitly when working outside a repo or targeting a different repo or PR.

```sh
# Inside the repo on the PR branch -- no flags needed
gh pr-review threads list
gh pr-review review view
```

This inference is available for all subcommands and was added to make agent workflows cleaner when operating in-repo.

---

## Backend Policy

Every command uses a single GitHub backend with no runtime fallbacks.

| Command | Backend | Notes |
| --- | --- | --- |
| `review --start` | GraphQL | Opens a pending review via `addPullRequestReview`. |
| `review --add-comment` | GraphQL | Requires a `PRR_...` review node ID. |
| `review view` | GraphQL | Aggregates reviews, inline comments, and replies. |
| `review --submit` | GraphQL | Finalizes via `submitPullRequestReview`. |
| `comments reply` | GraphQL | Replies via `addPullRequestReviewThreadReply`. |
| `threads list` | GraphQL | Enumerates review threads with filtering. |
| `threads resolve` / `unresolve` | GraphQL | Mutates thread resolution; supply `PRRT_...` node IDs. |
| `threads resolve-all` | GraphQL | Bulk resolution with per-thread reply+resolve. |

---

## Design for LLMs & Automated Agents

`gh-pr-review` is purpose-built for agents operating at scale. Everything about its output format, filter flags, and output modes was designed around the constraint that an agent is consuming the output.

### Why it's agent-native

- **One command replaces a chain of API calls.**  
  Instead of `list reviews -> list threads -> list comments per thread`, a single `gh pr-review review view` returns the fully assembled review structure.

- **`--author` substring filtering.**  
  Agents often know the bot name but not the exact login. `--author rabbit` matches `coderabbitai`, `coderabbit-bot`, and any variation — case-insensitively.

- **`-o ids` output mode for piping.**  
  Bare thread IDs on stdout, one per line — designed to pipe into `xargs` or agent loops without any parsing.

- **`--commit HEAD` with symbolic ref resolution.**  
  Agents commit code then need to reference that commit. Pass `HEAD` and the tool resolves it to the short SHA and constructs a clickable link automatically.

- **Atomic reply-then-resolve.**  
  When `--commit` is set, the reply is posted first. If posting fails, the thread stays unresolved. No silent partial states.

- **`--since` for incremental workflows.**  
  Agents re-running after a push only want threads updated since the last run. `--since <RFC3339>` makes this exact.

- **`threads resolve-all` for O(1) triage.**  
  After addressing CodeRabbit findings, one command resolves all matching threads with commit links. No loops needed.

- **Compact, deterministic JSON.**  
  Optional fields are omitted rather than null. Field order is stable. Token cost is minimized.

> "A good tool definition should define a clear, narrow purpose, return exactly the meaningful context the agent needs, and avoid burdening the model with low-signal intermediate results."

---

## Additional Docs

- [docs/USAGE.md](docs/USAGE.md) — Command-by-command inputs, outputs, and examples.
- [docs/SCHEMAS.md](docs/SCHEMAS.md) — JSON schemas for every structured response.
- [docs/AGENTS.md](docs/AGENTS.md) — Agent-focused workflows, prompts, and best practices.

---

## Using as a Skill

`gh-pr-review` can be used as a reusable skill for AI coding agents via [Vercel's add-skill package](https://github.com/vercel-labs/add-skill).

### Installation as a Skill

```sh
npx skills add agynio/gh-pr-review
```

Additional options:

```sh
# Install to a specific agent
npx skills add agynio/gh-pr-review -a claude-code

# Install globally (available across all projects)
npx skills add agynio/gh-pr-review -g
```

This command will:
- Install the gh-pr-review extension via `gh extension install`
- Register the skill with your AI agent using the [SKILL.md](SKILL.md) definition
- Make all gh-pr-review commands available as skill actions

### What the Skill Provides

Once installed, your AI coding agent can:

- **Triage PR threads**: Filter by author, resolution state, and timestamp
- **View PR reviews**: Get complete inline comment threads with file context
- **Reply to comments**: Respond to review feedback programmatically
- **Resolve threads**: Mark discussions as resolved, with commit links, atomically
- **Bulk resolve**: Resolve all threads matching a filter in one command
- **Create reviews**: Start pending reviews and add inline comments

All commands return structured JSON optimized for agent consumption with minimal tokens and maximum context.

### Example Agent Workflow

```
User: "Address all CodeRabbit findings on this PR"
Agent: gh pr-review threads list --author rabbit --unresolved
Agent: [Reads threads, applies fixes, commits]
Agent: gh pr-review threads resolve-all --author rabbit --commit HEAD
Agent: [All CodeRabbit threads resolved with clickable commit links]
```

### Skill Documentation

See [SKILL.md](SKILL.md) for complete skill documentation including:
- Core commands reference
- JSON output schemas
- Best practices for agents
- Common workflow patterns

---

## Development

Run the test suite and linters locally with cgo disabled (matching the release build):

```sh
CGO_ENABLED=0 go test ./...
CGO_ENABLED=0 golangci-lint run
```

Releases are built using the [`cli/gh-extension-precompile`](https://github.com/cli/gh-extension-precompile) workflow to publish binaries for macOS, Linux, and Windows.
