# Output schemas (v1.4.0)

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
      "description": "GraphQL review node identifier (PRR_…)"
    },
    "state": {
      "type": "string",
      "enum": ["PENDING", "COMMENTED", "APPROVED", "DISMISSED", "REQUEST_CHANGES"]
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
      "description": "GraphQL review thread node identifier"
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

Emitted by `review report`.

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
        "thread"
      ],
      "properties": {
        "thread_id": {
          "type": "string",
          "description": "GraphQL review thread identifier"
        },
        "comment_node_id": {
          "type": "string",
          "description": "GraphQL comment node identifier when requested"
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
        "thread": {
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
          "description": "GraphQL comment node identifier when requested"
        },
        "in_reply_to_comment_node_id": {
          "type": "string",
          "description": "GraphQL node ID of the parent comment"
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

## ReplyComment

Default payload from `comments reply`.

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "title": "ReplyComment",
  "type": "object",
  "required": [
    "id",
    "thread_id",
    "thread_is_resolved",
    "thread_is_outdated",
    "body",
    "author_login",
    "html_url",
    "created_at",
    "updated_at"
  ],
  "properties": {
    "id": {
      "type": "string",
      "description": "GraphQL comment node identifier"
    },
    "database_id": {
      "type": "integer",
      "minimum": 1,
      "description": "Numeric comment identifier when persisted"
    },
    "review_id": {
      "type": "string",
      "description": "GraphQL review identifier when attached to a review"
    },
    "review_database_id": {
      "type": "integer",
      "minimum": 1
    },
    "review_state": {
      "type": "string"
    },
    "thread_id": {
      "type": "string"
    },
    "thread_is_resolved": {
      "type": "boolean"
    },
    "thread_is_outdated": {
      "type": "boolean"
    },
    "reply_to_comment_id": {
      "type": "string"
    },
    "body": {
      "type": "string"
    },
    "diff_hunk": {
      "type": "string"
    },
    "path": {
      "type": "string"
    },
    "html_url": {
      "type": "string",
      "format": "uri"
    },
    "author_login": {
      "type": "string"
    },
    "created_at": {
      "type": "string",
      "format": "date-time"
    },
    "updated_at": {
      "type": "string",
      "format": "date-time"
    }
  },
  "additionalProperties": false
}
```

## ReplyConcise

Minimal payload from `comments reply --concise`.

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "title": "ReplyConcise",
  "type": "object",
  "required": ["id"],
  "properties": {
    "id": {
      "type": "string"
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
  "required": ["threadId", "isResolved", "path", "isOutdated"],
  "properties": {
    "threadId": {
      "type": "string"
    },
    "isResolved": {
      "type": "boolean"
    },
    "resolvedBy": {
      "type": "string",
      "description": "Login of the user who resolved the thread"
    },
    "updatedAt": {
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
    "isOutdated": {
      "type": "boolean"
    }
  },
  "additionalProperties": false
}
```

## ThreadActionResult

Emitted by `threads resolve` and `threads unresolve`.

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "title": "ThreadActionResult",
  "type": "object",
  "required": ["threadId", "isResolved", "changed"],
  "properties": {
    "threadId": {
      "type": "string"
    },
    "isResolved": {
      "type": "boolean"
    },
    "changed": {
      "type": "boolean",
      "description": "False when the thread already matched the requested state"
    }
  },
  "additionalProperties": false
}
```

## ThreadFindResult

Returned by `threads find` and used internally when mapping REST comment IDs to
GraphQL threads.

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "title": "ThreadFindResult",
  "type": "object",
  "required": ["id", "isResolved"],
  "properties": {
    "id": {
      "type": "string"
    },
    "isResolved": {
      "type": "boolean"
    }
  },
  "additionalProperties": false
}
```
