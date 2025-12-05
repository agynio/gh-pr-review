package report

const reportQuery = `query Report(
  $owner: String!,
  $name: String!,
  $number: Int!,
  $states: [PullRequestReviewState!],
  $firstReviews: Int,
  $firstThreads: Int,
  $firstComments: Int
) {
  repository(owner: $owner, name: $name) {
    pullRequest(number: $number) {
      reviews(first: $firstReviews, states: $states) {
        nodes {
          id
          state
          body
          submittedAt
          databaseId
          author { login }
        }
      }
      reviewThreads(first: $firstThreads) {
        nodes {
          id
          path
          line
          isResolved
          isOutdated
          comments(first: $firstComments) {
            nodes {
              databaseId
              body
              createdAt
              author { login }
              pullRequestReview {
                id
                state
                databaseId
              }
              replyTo {
                databaseId
              }
            }
          }
        }
      }
    }
  }
}`
