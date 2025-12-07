# Output schemas (v1.6.0)

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
          "description": "GraphQL comment node identifier when requested"
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
      "description": "GraphQL comment node identifier"
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
