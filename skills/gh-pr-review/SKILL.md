---
name: gh-pr-review
description: View and manage inline GitHub PR review comments with full thread context from the terminal
---

# gh-pr-review

A GitHub CLI extension that provides complete inline PR review comment access from the terminal with LLM-friendly JSON output.

## When to Use

Use this skill when you need to:

- View inline review comments and threads on a pull request
- Reply to review comments programmatically
- Resolve or unresolve review threads
- Create and submit PR reviews with inline comments
- Edit or delete comments in pending reviews
- Access PR review context for automated workflows
- Filter reviews by state, reviewer, or resolution status
- Poll PRs for updates (comments, conflicts, CI failures)
- Add reactions to comments or reviews
- View full conversation history for specific threads
- Manage pull request draft status (mark as draft/ready for review)
- List all draft pull requests in a repository

This tool is particularly useful for:

- Automated PR review workflows
- LLM-based code review agents
- Terminal-based PR review processes
- Getting structured review data without multiple API calls
- CI/CD pipelines that need to wait for PR attention

## Installation

First, ensure the extension is installed:

```sh
gh extension install v2nic/gh-pr-review
```

## Core Commands

### 1. View All Reviews and Threads

Get complete review context with inline comments and thread replies:

```sh
gh pr-review review view -R owner/repo --pr <number>
```

**Useful filters:**

- `--unresolved` - Only show unresolved threads
- `--reviewer <login>` - Filter by specific reviewer
- `--states <APPROVED|CHANGES_REQUESTED|COMMENTED|DISMISSED|PENDING>` - Filter by review state
- `--tail <n>` - Keep only last n replies per thread
- `--not_outdated` - Exclude outdated threads
- `--author <login>` - Filter threads to those containing a comment by this author
- `--include-resolved` - Include resolved threads (overrides --unresolved)
- `--include-comment-node-id` - Add comment node identifiers to parent comments and replies

**Output:** Structured JSON with reviews, comments, thread_ids, and resolution status.

### 2. Reply to Review Threads

Reply to an existing inline comment thread:

```sh
gh pr-review comments reply <pr-number> -R owner/repo \
  --thread-id <thread_id> \
  --body "Your reply message"
```

Alternatively, read the reply from a file (use `"-"` for stdin):

```sh
gh pr-review comments reply <pr-number> -R owner/repo \
  --thread-id <thread_id> \
  --body-file reply.md
```

### 3. List Review Threads

Get a filtered list of review threads:

```sh
gh pr-review threads list -R owner/repo <pr-number> --unresolved --mine
```

**Filters:**

- `--unresolved` - Only show unresolved threads
- `--mine` - Show only threads involving or resolvable by the viewer

### 4. View Specific Threads

View full conversation for one or more threads by ID:

```sh
gh pr-review threads view <thread_id> <thread_id>
```

This returns the complete comment history for the specified threads.

### 5. Resolve/Unresolve Threads

Mark threads as resolved:

```sh
gh pr-review threads resolve -R owner/repo <pr-number> --thread-id <thread_id>
```

### 6. Add Reactions

Add reactions to any GitHub node (comments, reviews, etc.):

```sh
gh pr-review react <comment_id> --type thumbs_up
```

**Valid reaction types:** `thumbs_up`, `thumbs_down`, `laugh`, `hooray`, `confused`, `heart`, `rocket`, `eyes`

### 7. Manage Draft Status

Check if a pull request is a draft:

```sh
gh pr-review draft status -R owner/repo <pr-number>
```

Mark a pull request as draft:

```sh
gh pr-review draft mark -R owner/repo <pr-number>
```

Mark a pull request as ready for review:

```sh
gh pr-review draft ready -R owner/repo <pr-number>
```

List all draft pull requests in the repository:

```sh
gh pr-review draft list -R owner/repo
```

**Output formats:**

- `draft status`: `{"pr_number": 1, "is_draft": false, "title": "PR Title"}`
- `draft mark/ready`: `{"pr_number": 1, "is_draft": true, "status": "marked as draft"}`
- `draft list`: `[{"pr_number": 1, "is_draft": true, "title": "Draft PR"}]`

### 8. Create and Submit Reviews

Start a pending review:

```sh
gh pr-review review --start -R owner/repo <pr-number>
```

Add inline comments to pending review:

```sh
gh pr-review review --add-comment \
  --review-id <review_id> \
  --path <file-path> \
  --line <line-number> \
  --body "Your comment" \
  -R owner/repo <pr-number>
```

Or read the comment body from a file (use `"-"` for stdin):

```sh
gh pr-review review --add-comment \
  --review-id <review_id> \
  --path <file-path> \
  --line <line-number> \
  --body-file comment.md \
  -R owner/repo <pr-number>
```

**Edit a comment in pending review** (requires comment node ID):

```sh
gh pr-review review --edit-comment \
  --comment-id <comment_id> \
  --body "Updated comment text" \
  -R owner/repo <pr-number>
```

**Delete a comment from pending review** (requires comment node ID):

```sh
gh pr-review review --delete-comment \
  --comment-id <comment_id> \
  -R owner/repo <pr-number>
```

**Additional flags for add-comment:**

- `--side <LEFT|RIGHT>` - Diff side for inline comment (default: RIGHT)
- `--start-line <n>` - Start line for multi-line comments
- `--start-side <LEFT|RIGHT>` - Start side for multi-line comments

**Additional flags for review start:**

- `--commit <sha>` - Pin the pending review to a specific commit (defaults to current head)

Submit the review:

```sh
gh pr-review review --submit \
  --review-id <review_id> \
  --event <APPROVE|REQUEST_CHANGES|COMMENT> \
  --body "Overall review summary" \
  -R owner/repo <pr-number>
```

## Output Format

All commands return structured JSON optimized for programmatic use:

- Consistent field names
- Stable ordering
- Omitted fields instead of null values
- Essential data only (no URLs or metadata noise)
- Pre-joined thread replies

Example output structure:

```json
{
  "reviews": [
    {
      "id": "PRR_...",
      "state": "CHANGES_REQUESTED",
      "author_login": "reviewer",
      "comments": [
        {
          "thread_id": "PRRT_...",
          "path": "src/file.go",
          "author_login": "reviewer",
          "body": "Consider refactoring this",
          "created_at": "2024-01-15T10:30:00Z",
          "is_resolved": false,
          "is_outdated": false,
          "thread_comments": [
            {
              "author_login": "author",
              "body": "Good point, will fix",
              "created_at": "2024-01-15T11:00:00Z"
            }
          ]
        }
      ]
    }
  ]
}
```

## Best Practices

1. **Always use `-R owner/repo`** to specify the repository explicitly
2. **Use `--unresolved` and `--not_outdated`** to focus on actionable comments
3. **Save thread_id values** from `review view` output for replying
4. **Filter by reviewer** when dealing with specific review feedback
5. **Use `--tail 1`** to reduce output size by keeping only latest replies
6. **Parse JSON output** instead of trying to scrape text

## Common Workflows

### Get Unresolved Comments for Current PR

```sh
gh pr-review review view --unresolved --not_outdated -R owner/repo --pr $(gh pr view --json number -q .number)
```

### Claim a Thread Before Working on It

**IMPORTANT**: Before addressing any thread, add an 👀 reaction to the FIRST comment in the thread to claim it. This prevents multiple agents from working on the same thread simultaneously.

```sh
# Add 👀 reaction to claim the thread
gh pr-review react <comment_node_id> --type eyes
```

Then address the thread, reply, and resolve. If you cannot address the thread (e.g., it needs human input), reply and leave it unresolved.

**Note:** Adding a reaction (👍, 👀, etc.) does NOT resolve a thread. To resolve, use `gh pr-review threads resolve`.

### Reply to All Unresolved Comments

1. Get unresolved threads: `gh pr-review threads list --unresolved -R owner/repo <pr>`
2. For each thread_id, reply: `gh pr-review comments reply <pr> -R owner/repo --thread-id <id> --body "..."`
3. Resolve the thread: `gh pr-review threads resolve <pr> -R owner/repo --thread-id <id>`
   (If the thread is non-actionable, resolve it with a brief reply explaining why it's being dismissed.)

### Wait for PR Attention

Poll a PR until it needs attention (comments, conflicts, or CI failures):

```sh
# Check once and exit
gh pr-review await --check-only -R owner/repo <pr>

# Poll until work detected (default: 1 day timeout, 5 minute interval)
gh pr-review await -R owner/repo <pr>

# Poll for comments only with custom timeout
gh pr-review await --mode comments --timeout 3600 -R owner/repo <pr>
```

**Exit codes:**

- `0` - Work detected (PR needs attention)
- `1` - Error occurred
- `2` - Timed out with no work detected

**Await flags:**

- `--mode <all|comments|conflicts|actions>` - Watch mode (default: all)
- `--timeout <seconds>` - Maximum polling time (default: 86400 = 1 day)
- `--interval <seconds>` - Polling interval (default: 300 = 5 minutes)
- `--debounce <seconds>` - Debounce duration (default: 30)
- `--check-only` - Check once and exit without polling

### Create Review with Inline Comments

1. Start: `gh pr-review review --start -R owner/repo <pr>`
2. Add comments: `gh pr-review review --add-comment -R owner/repo <pr> --review-id <review_id> --path <file> --line <num> --body "..."` (or use `--body-file <path>`)
3. Edit comments (if needed): `gh pr-review review --edit-comment -R owner/repo <pr> --comment-id <comment_id> --body "Updated text"`
4. Delete comments (if needed): `gh pr-review review --delete-comment -R owner/repo <pr> --comment-id <comment_id>`
5. Submit: `gh pr-review review --submit -R owner/repo <pr> --review-id <review_id> --event REQUEST_CHANGES --body "Summary"`

### 6. Add Reactions to Comments

Add reactions to any GitHub node (comments, reviews, etc.):

```sh
gh pr-review react <comment_id> --type thumbs_up
```

**Valid reaction types:** `thumbs_up`, `thumbs_down`, `laugh`, `hooray`, `confused`, `heart`, `rocket`, `eyes`

## Important Notes

- All IDs use GraphQL format (PRR*... for reviews, PRRT*... for threads, PRRC\_... for comments)
- Commands use pure GraphQL (no REST API fallbacks)
- Empty arrays `[]` are returned when no data matches filters
- Thread replies are sorted by created_at ascending
- Comments can only be edited/deleted while the review is in pending state
- Numeric review IDs are rejected; use GraphQL node IDs (PRR\_...)
- The `--body-file` flag supports "-" for stdin input
- The `--review-id` flag in `comments reply` is for replying inside your pending review

## Documentation Links

- [docs/SCHEMAS.md](docs/SCHEMAS.md) — JSON schemas
- [skills/references/USAGE.md](skills/references/USAGE.md) — Detailed usage examples
- This is a fork of [agynio/gh-pr-review](https://github.com/agynio/gh-pr-review).
