# Engineer Review Guide

## `gh pr-review review view`

High-level summary of the pull request review activity. The command issues a
single GraphQL operation and emits JSON grouped by review → parent inline
comment → thread replies.

```bash
gh pr-review review view -R <owner>/<repo> <number>
```

### Default scope

- Includes every reviewer and review state (APPROVED, CHANGES_REQUESTED,
  COMMENTED, DISMISSED).
- Threads are grouped by parent inline comment; replies are sorted by
  `created_at` ascending.
- Optional fields are omitted rather than rendered as `null`.

### Useful filters

| Flag | Description |
| --- | --- |
| `--reviewer <login>` | Limit to a specific reviewer login (case-insensitive). |
| `--states <list>` | Comma-separated list of review states. |
| `--unresolved` | Only include unresolved threads. |
| `--not_outdated` | Drop threads marked as outdated. |
| `--tail <n>` | Keep the last `n` replies per thread (0 keeps all). |

Example capturing the latest actionable work:

```bash
gh pr-review review view -R agyn/repo 51 \
  --reviewer emerson \
  --states CHANGES_REQUESTED,COMMENTED \
  --unresolved \
  --not_outdated \
  --tail 2
```

### Output schema

```json
{
  "reviews": [
    {
      "id": "PRR_…",
      "state": "COMMENTED",
      "submitted_at": "2025-12-03T10:00:00Z",
      "author_login": "emerson",
      "comments": [
        {
          "id": 123456789,
          "path": "internal/service.go",
          "line": 42,
          "is_resolved": false,
          "is_outdated": false,
          "thread_comments": [
            {
              "id": 123456790,
              "in_reply_to_id": 123456789,
              "author_login": "casey",
              "body": "Followed up in commit abc123",
              "created_at": "2025-12-03T10:05:00Z"
            }
          ]
        }
      ]
    }
  ]
}
```

Empty optional fields are omitted from the JSON payload (for example, `body` and
`submitted_at` are only emitted when the data is present).
