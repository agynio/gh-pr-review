package await

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCountUnresolvedThreads(t *testing.T) {
	tests := []struct {
		name     string
		threads  []ReviewThread
		expected int
	}{
		{
			name:     "empty threads",
			threads:  []ReviewThread{},
			expected: 0,
		},
		{
			name: "all resolved",
			threads: []ReviewThread{
				{IsResolved: true},
				{IsResolved: true},
			},
			expected: 0,
		},
		{
			name: "some unresolved",
			threads: []ReviewThread{
				{IsResolved: true},
				{IsResolved: false},
				{IsResolved: false},
			},
			expected: 2,
		},
		{
			name: "none resolved",
			threads: []ReviewThread{
				{IsResolved: false},
				{IsResolved: false},
			},
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pr := &PullRequest{
				ReviewThreads: ThreadNodes{Nodes: tt.threads},
			}
			assert.Equal(t, tt.expected, CountUnresolvedThreads(pr))
		})
	}
}

func TestHasConflicts(t *testing.T) {
	tests := []struct {
		name      string
		mergeable string
		expected  bool
	}{
		{"mergeable", "MERGEABLE", false},
		{"clean", "CLEAN", false},
		{"conflicting", "CONFLICTING", true},
		{"unknown", "UNKNOWN", false},
		{"", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pr := &PullRequest{Mergeable: tt.mergeable}
			assert.Equal(t, tt.expected, HasConflicts(pr))
		})
	}
}

func TestFailingChecks(t *testing.T) {
	t.Run("no suites", func(t *testing.T) {
		pr := &PullRequest{Commits: CommitNodes{Nodes: []Commit{{}}}}
		assert.Empty(t, FailingChecks(pr))
	})

	t.Run("failure conclusion", func(t *testing.T) {
		pr := &PullRequest{
			Commits: CommitNodes{
				Nodes: []Commit{
					{
						Commit: CommitDetails{
							CheckSuites: SuiteNodes{
								Nodes: []CheckSuite{
									{Conclusion: "FAILURE", App: AppInfo{Name: "CI"}},
								},
							},
						},
					},
				},
			},
		}
		assert.Equal(t, []string{"CI"}, FailingChecks(pr))
	})

	t.Run("error conclusion", func(t *testing.T) {
		pr := &PullRequest{
			Commits: CommitNodes{
				Nodes: []Commit{
					{
						Commit: CommitDetails{
							CheckSuites: SuiteNodes{
								Nodes: []CheckSuite{
									{Conclusion: "ERROR", App: AppInfo{Name: "Build"}},
								},
							},
						},
					},
				},
			},
		}
		assert.Equal(t, []string{"Build"}, FailingChecks(pr))
	})

	t.Run("success conclusion", func(t *testing.T) {
		pr := &PullRequest{
			Commits: CommitNodes{
				Nodes: []Commit{
					{
						Commit: CommitDetails{
							CheckSuites: SuiteNodes{
								Nodes: []CheckSuite{
									{Conclusion: "SUCCESS", App: AppInfo{Name: "CI"}},
								},
							},
						},
					},
				},
			},
		}
		assert.Empty(t, FailingChecks(pr))
	})

	t.Run("failing check run", func(t *testing.T) {
		pr := &PullRequest{
			Commits: CommitNodes{
				Nodes: []Commit{
					{
						Commit: CommitDetails{
							CheckSuites: SuiteNodes{
								Nodes: []CheckSuite{
									{
										Conclusion: "SUCCESS",
										CheckRuns: RunNodes{
											Nodes: []CheckRun{
												{Name: "test", Conclusion: "FAILURE"},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}
		assert.Equal(t, []string{"test"}, FailingChecks(pr))
	})
}

func TestPendingChecks(t *testing.T) {
	t.Run("no suites", func(t *testing.T) {
		pr := &PullRequest{Commits: CommitNodes{Nodes: []Commit{{}}}}
		assert.Empty(t, PendingChecks(pr))
	})

	t.Run("in_progress status", func(t *testing.T) {
		pr := &PullRequest{
			Commits: CommitNodes{
				Nodes: []Commit{
					{
						Commit: CommitDetails{
							CheckSuites: SuiteNodes{
								Nodes: []CheckSuite{
									{Status: "IN_PROGRESS", App: AppInfo{Name: "CI"}},
								},
							},
						},
					},
				},
			},
		}
		assert.Equal(t, []string{"CI"}, PendingChecks(pr))
	})

	t.Run("queued status", func(t *testing.T) {
		pr := &PullRequest{
			Commits: CommitNodes{
				Nodes: []Commit{
					{
						Commit: CommitDetails{
							CheckSuites: SuiteNodes{
								Nodes: []CheckSuite{
									{Status: "QUEUED", App: AppInfo{Name: "Build"}},
								},
							},
						},
					},
				},
			},
		}
		assert.Equal(t, []string{"Build"}, PendingChecks(pr))
	})

	t.Run("completed status", func(t *testing.T) {
		pr := &PullRequest{
			Commits: CommitNodes{
				Nodes: []Commit{
					{
						Commit: CommitDetails{
							CheckSuites: SuiteNodes{
								Nodes: []CheckSuite{
									{Status: "COMPLETED", App: AppInfo{Name: "CI"}},
								},
							},
						},
					},
				},
			},
		}
		assert.Empty(t, PendingChecks(pr))
	})
}

func TestParseMode(t *testing.T) {
	tests := []struct {
		input    string
		expected Mode
		err      bool
	}{
		{"all", ModeAll, false},
		{"comments", ModeComments, false},
		{"conflicts", ModeConflicts, false},
		{"actions", ModeActions, false},
		{"ALL", ModeAll, false},
		{"Comments", ModeComments, false},
		{"invalid", "", true},
		{"", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			mode, err := ParseMode(tt.input)
			if tt.err {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, mode)
			}
		})
	}
}

func TestWorkConditions(t *testing.T) {
	t.Run("all mode - clean PR", func(t *testing.T) {
		pr := &PullRequest{
			Mergeable: "MERGEABLE",
			ReviewThreads: ThreadNodes{
				Nodes: []ReviewThread{
					{IsResolved: true},
				},
			},
			Commits: CommitNodes{
				Nodes: []Commit{
					{
						Commit: CommitDetails{
							CheckSuites: SuiteNodes{
								Nodes: []CheckSuite{
									{Conclusion: "SUCCESS"},
								},
							},
						},
					},
				},
			},
		}
		assert.Empty(t, WorkConditions(pr, ModeAll))
	})

	t.Run("all mode - has unresolved", func(t *testing.T) {
		pr := &PullRequest{
			Mergeable: "MERGEABLE",
			ReviewThreads: ThreadNodes{
				Nodes: []ReviewThread{
					{IsResolved: false},
				},
			},
		}
		conds := WorkConditions(pr, ModeAll)
		assert.Contains(t, conds, "unresolved-threads")
	})

	t.Run("all mode - has conflicts", func(t *testing.T) {
		pr := &PullRequest{Mergeable: "CONFLICTING"}
		conds := WorkConditions(pr, ModeAll)
		assert.Contains(t, conds, "conflicts")
	})

	t.Run("all mode - has failing checks", func(t *testing.T) {
		pr := &PullRequest{
			Commits: CommitNodes{
				Nodes: []Commit{
					{
						Commit: CommitDetails{
							CheckSuites: SuiteNodes{
								Nodes: []CheckSuite{
									{Conclusion: "FAILURE"},
								},
							},
						},
					},
				},
			},
		}
		conds := WorkConditions(pr, ModeAll)
		assert.Contains(t, conds, "actions:failing")
	})

	t.Run("comments mode only", func(t *testing.T) {
		pr := &PullRequest{
			Mergeable: "CONFLICTING",
			ReviewThreads: ThreadNodes{
				Nodes: []ReviewThread{
					{IsResolved: false},
				},
			},
		}
		conds := WorkConditions(pr, ModeComments)
		assert.Contains(t, conds, "unresolved-threads")
		assert.NotContains(t, conds, "conflicts")
	})

	t.Run("conflicts mode only", func(t *testing.T) {
		pr := &PullRequest{
			Mergeable: "CONFLICTING",
			ReviewThreads: ThreadNodes{
				Nodes: []ReviewThread{
					{IsResolved: false},
				},
			},
		}
		conds := WorkConditions(pr, ModeConflicts)
		assert.Contains(t, conds, "conflicts")
		assert.NotContains(t, conds, "unresolved-threads")
	})
}

func TestSecondsToHuman(t *testing.T) {
	tests := []struct {
		seconds  int
		expected string
	}{
		{30, "30 second(s)"},
		{60, "1 minute(s)"},
		{120, "2 minute(s)"},
		{3600, "1 hour(s)"},
		{7200, "2 hour(s)"},
		{86400, "1 day(s)"},
		{172800, "2 day(s)"},
		{90, "1 minute(s)"},
		{3661, "1 hour(s)"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, SecondsToHuman(tt.seconds))
		})
	}
}

func TestSuiteName(t *testing.T) {
	t.Run("uses app name", func(t *testing.T) {
		suite := &CheckSuite{App: AppInfo{Name: "GitHub Actions", Slug: "github-actions"}}
		assert.Equal(t, "GitHub Actions", suiteName(suite))
	})

	t.Run("falls back to slug", func(t *testing.T) {
		suite := &CheckSuite{App: AppInfo{Name: "", Slug: "github-actions"}}
		assert.Equal(t, "github-actions", suiteName(suite))
	})

	t.Run("empty", func(t *testing.T) {
		suite := &CheckSuite{App: AppInfo{Name: "", Slug: ""}}
		assert.Equal(t, "", suiteName(suite))
	})
}
