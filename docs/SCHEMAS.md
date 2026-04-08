# Output schemas

Optional fields are omitted entirely (never serialized as `null`). Unless noted,
schemas disallow additional properties to surface unexpected payload changes.

## ReviewState

Used by `review --start` and `review --submit`.

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "title": "ReviewState",
  "type": "object",
  "required": ["id", "state"],
  "properties": {
    "id": {
      "type": "string",
      "description": "review node identifier (PRR_…)"
    },
    "state": {
      "type": "string",
      "enum": [
        "PENDING",
        "COMMENTED",
        "APPROVED",
        "DISMISSED",
        "REQUEST_CHANGES"
      ]
    },
    "submitted_at": {
      "type": "string",
      "format": "date-time",
      "description": "RFC3339 timestamp of the submission (omitted when pending)"
    }
  },
  "additionalProperties": false
}
```

## ReviewThread

Produced by `review --add-comment`.

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "title": "ReviewThread",
  "type": "object",
  "required": ["id", "path", "is_outdated"],
  "properties": {
    "id": {
      "type": "string",
      "description": "review thread node identifier"
    },
    "path": {
      "type": "string",
      "description": "File path for the inline thread"
    },
    "is_outdated": {
      "type": "boolean"
    },
    "line": {
      "type": "integer",
      "minimum": 1,
      "description": "Updated diff line (omitted for multi-line threads)"
    }
  },
  "additionalProperties": false
}
```

## ReviewReport

Emitted by `review view`.

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "title": "ReviewReport",
  "type": "object",
  "required": ["reviews"],
  "properties": {
    "reviews": {
      "type": "array",
      "items": {
        "$ref": "#/$defs/ReportReview"
      }
    }
  },
  "additionalProperties": false,
  "$defs": {
    "ReportReview": {
      "type": "object",
      "required": ["id", "state", "author_login"],
      "properties": {
        "id": {
          "type": "string"
        },
        "state": {
          "type": "string",
          "enum": ["APPROVED", "CHANGES_REQUESTED", "COMMENTED", "DISMISSED"]
        },
        "body": {
          "type": "string"
        },
        "submitted_at": {
          "type": "string",
          "format": "date-time"
        },
        "author_login": {
          "type": "string"
        },
        "comments": {
          "type": "array",
          "items": {
            "$ref": "#/$defs/ReportComment"
          }
        }
      },
      "additionalProperties": false
    },
    "ReportComment": {
      "type": "object",
      "required": [
        "thread_id",
        "path",
        "author_login",
        "body",
        "created_at",
        "is_resolved",
        "is_outdated",
        "thread_comments"
      ],
      "properties": {
        "thread_id": {
          "type": "string",
          "description": "review thread identifier"
        },
        "comment_node_id": {
          "type": "string",
          "description": "comment node identifier when requested"
        },
        "path": {
          "type": "string"
        },
        "line": {
          "type": ["integer", "null"],
          "minimum": 1
        },
        "author_login": {
          "type": "string"
        },
        "body": {
          "type": "string"
        },
        "created_at": {
          "type": "string",
          "format": "date-time"
        },
        "is_resolved": {
          "type": "boolean"
        },
        "is_outdated": {
          "type": "boolean"
        },
        "thread_comments": {
          "type": "array",
          "items": {
            "$ref": "#/$defs/ThreadReply"
          }
        }
      },
      "additionalProperties": false
    },
    "ThreadReply": {
      "type": "object",
      "required": ["id", "author_login", "body", "created_at"],
      "properties": {
        "comment_node_id": {
          "type": "string",
          "description": "comment node identifier when requested"
        },
        "author_login": {
          "type": "string"
        },
        "body": {
          "type": "string"
        },
        "created_at": {
          "type": "string",
          "format": "date-time"
        }
      },
      "additionalProperties": false
    }
  }
}
```

## ReplyMinimal

Returned by `comments reply`.

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "title": "ReplyMinimal",
  "type": "object",
  "required": ["comment_node_id"],
  "properties": {
    "comment_node_id": {
      "type": "string",
      "description": "comment node identifier"
    }
  },
  "additionalProperties": false
}
```

## StatusResult

Returned by `review --edit-comment`, `review --delete-comment`, and `review --submit`.

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "title": "StatusResult",
  "type": "object",
  "required": ["status"],
  "properties": {
    "status": {
      "type": "string",
      "description": "operation status message"
    }
  },
  "additionalProperties": false
}
```

## ThreadSummary

Returned by `threads list`.

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "title": "ThreadSummary",
  "type": "object",
  "required": ["thread_id", "is_resolved", "path", "is_outdated"],
  "properties": {
    "thread_id": {
      "type": "string"
    },
    "is_resolved": {
      "type": "boolean"
    },
    "resolved_by": {
      "type": "string",
      "description": "Login of the user who resolved the thread"
    },
    "updated_at": {
      "type": "string",
      "format": "date-time"
    },
    "path": {
      "type": "string"
    },
    "line": {
      "type": "integer",
      "minimum": 1
    },
    "is_outdated": {
      "type": "boolean"
    }
  },
  "additionalProperties": false
}
```

## ThreadMutationResult

Returned by `threads resolve` and `threads unresolve`.

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "title": "ThreadMutationResult",
  "type": "object",
  "required": ["thread_node_id", "is_resolved"],
  "properties": {
    "thread_node_id": {
      "type": "string"
    },
    "is_resolved": {
      "type": "boolean"
    }
  },
  "additionalProperties": false
}
```

## ReactionResult

Returned by `react`.

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "title": "ReactionResult",
  "type": "object",
  "required": ["node_id", "reaction", "status"],
  "properties": {
    "node_id": {
      "type": "string",
      "description": "GraphQL node ID of the reacted object"
    },
    "reaction": {
      "type": "string",
      "description": "Reaction type (thumbs_up, thumbs_down, laugh, hooray, confused, heart, rocket, eyes)"
    },
    "status": {
      "type": "string",
      "enum": ["added"]
    }
  },
  "additionalProperties": false
}
```

## AwaitResult

Returned by `await`.

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "title": "AwaitResult",
  "type": "object",
  "required": [
    "conditions",
    "unresolved",
    "general",
    "conflicts",
    "failing",
    "pending",
    "timed_out",
    "cancelled",
    "watched_ms"
  ],
  "properties": {
    "conditions": {
      "type": "array",
      "items": {
        "type": "string"
      },
      "description": "List of detected conditions (e.g., new_comments, merge_conflicts, ci_failures)"
    },
    "unresolved": {
      "type": "integer",
      "description": "Count of unresolved review threads"
    },
    "general": {
      "type": "integer",
      "description": "Count of general PR comments"
    },
    "conflicts": {
      "type": "boolean",
      "description": "Whether the PR has merge conflicts"
    },
    "failing": {
      "type": "array",
      "items": {
        "type": "string"
      },
      "description": "List of failing CI check names"
    },
    "pending": {
      "type": "array",
      "items": {
        "type": "string"
      },
      "description": "List of pending CI check names"
    },
    "timed_out": {
      "type": "boolean",
      "description": "Whether the await operation timed out"
    },
    "cancelled": {
      "type": "boolean",
      "description": "Whether the await operation was cancelled"
    },
    "watched_ms": {
      "type": "integer",
      "description": "Time spent watching in milliseconds"
    }
  },
  "additionalProperties": false
}
```
