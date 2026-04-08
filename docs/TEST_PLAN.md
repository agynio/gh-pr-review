# Test Plan for gh-pr-review Extension

## What to Look For

- Commands execute without errors
- JSON output is valid
- Field names use snake_case (e.g., `thread_id`, `is_resolved`)
- GraphQL queries have no unused variables
- Exit codes are correct

## Create Private Test Repo

```bash
gh repo create <username>/gh-pr-review-test --private --description "Test repo for gh-pr-review extension"
cd /tmp
git clone git@github.com:<username>/gh-pr-review-test.git
cd gh-pr-review-test
echo "# Test Repo" > README.md
git add .
git commit -m "Initial commit"
git push
git checkout -b test-branch
echo "test change" >> README.md
git add .
git commit -m "Test change"
git push -u origin test-branch
gh pr create --title "Test PR" --body "Test PR for smoke testing gh-pr-review"
```

## Delete Test Repo

```bash
gh auth refresh -h github.com -s delete_repo
gh repo delete <username>/gh-pr-review-test --yes
```

## Smoke Testing Commands

```bash
gh extension install v2nic/gh-pr-review
cd /tmp/gh-pr-review-test
gh pr-review review view --repo <username>/gh-pr-review-test --pr 1
gh pr-review review --start --repo <username>/gh-pr-review-test --pr 1
gh pr-review review --add-comment --repo <username>/gh-pr-review-test --pr 1 --review-id <PRR_ID> --path README.md --line 1 --body "Test comment"
gh pr-review review --edit-comment --repo <username>/gh-pr-review-test --pr 1 --comment-id <PRRC_ID> --body "Edited comment"
gh pr-review review --delete-comment --repo <username>/gh-pr-review-test --pr 1 --comment-id <PRRC_ID>
gh pr-review comments reply --repo <username>/gh-pr-review-test --pr 1 --thread-id <PRRT_ID> --body "Test reply"
gh pr-review threads list --repo <username>/gh-pr-review-test --pr 1
gh pr-review threads view <PRRT_ID>
gh pr-review threads resolve --thread-id <PRRT_ID> --repo <username>/gh-pr-review-test --pr 1
gh pr-review threads unresolve --thread-id <PRRT_ID> --repo <username>/gh-pr-review-test --pr 1
gh pr-review react <PRRC_ID> --type thumbs_up
gh pr-review await --check-only --repo <username>/gh-pr-review-test --pr 1
gh pr-review review --submit --repo <username>/gh-pr-review-test --pr 1 --review-id <PRR_ID> --event COMMENT --body "Test review submission"

# Draft Management Commands
gh pr-review draft status --repo <username>/gh-pr-review-test --pr 1
gh pr-review draft mark --repo <username>/gh-pr-review-test --pr 1
gh pr-review draft status --repo <username>/gh-pr-review-test --pr 1
gh pr-review draft ready --repo <username>/gh-pr-review-test --pr 1
gh pr-review draft status --repo <username>/gh-pr-review-test --pr 1
gh pr-review draft list --repo <username>/gh-pr-review-test
```

## Expected Output Format

### Thread Output
```json
{
  "thread_id": "PRRT_...",
  "is_resolved": false,
  "updated_at": "2026-04-08T01:51:27Z",
  "path": "README.md",
  "line": 1,
  "is_outdated": false
}
```

### Draft Status Output
```json
{
  "pr_number": 1,
  "is_draft": false,
  "title": "Test PR"
}
```

### Draft Action Output
```json
{
  "pr_number": 1,
  "is_draft": true,
  "status": "marked as draft"
}
```

### Draft List Output
```json
[
  {
    "pr_number": 1,
    "is_draft": true,
    "title": "Draft PR 1"
  },
  {
    "pr_number": 2,
    "is_draft": true,
    "title": "Draft PR 2"
  }
]
```
