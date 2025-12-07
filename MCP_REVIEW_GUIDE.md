# MCP Review Guide

## Quick reference: `gh pr-review review view`

The command is GraphQL-only and performs a single query per invocation. It
returns structured JSON that is agent-friendly and omits nullable fields.

### Invocation pattern

```json
{
  "cmd": [
    "gh", "pr-review", "review", "view",
    "--repo", "agyn/repo",
    "51",
    "--reviewer", "casey",
    "--states", "CHANGES_REQUESTED,COMMENTED",
    "--unresolved",
    "--not_outdated",
    "--tail", "5"
  ]
}
```

### Response shape

`reviews[]` → `comments[]` → `thread[]`:

```json
{
  "id": "PRR_kwDOAAABbcdEFG12",
  "state": "COMMENTED",
  "author_login": "casey",
  "submitted_at": "2025-12-03T10:00:00Z",
  "comments": [
    {
      "id": 3531807471,
      "path": "internal/service.go",
      "line": 42,
      "is_resolved": false,
      "is_outdated": false,
      "thread_comments": [
        {
          "id": 3531807472,
          "in_reply_to_id": 3531807471,
          "author_login": "emerson",
          "body": "Can we reuse the helper?",
          "created_at": "2025-12-03T10:05:00Z"
        }
      ]
    }
  ]
}
```

Optional properties (`body`, `submitted_at`, `line`, `in_reply_to_id`,
`comments`) are omitted instead of set to `null`. Empty reply arrays are
serialized as `"thread_comments": []`.

### Filtering cheatsheet

- `--reviewer <login>` — narrow to a single reviewer (case-insensitive).
- `--states <comma list>` — choose from `APPROVED`, `CHANGES_REQUESTED`,
  `COMMENTED`, `DISMISSED`.
- `--unresolved` — drop resolved threads.
- `--not_outdated` — drop outdated threads.
- `--tail <n>` — keep the last `n` replies per thread (omit to show all).

### Failure modes

- Missing authentication token: surfaced directly from the underlying `gh` CLI.
- Unknown review state value: the command exits non-zero with a descriptive
  validation error before calling GraphQL.
- Pull request not accessible: returns `pull request not found or inaccessible`.
