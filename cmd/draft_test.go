package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/agynio/gh-pr-review/internal/ghcli"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDraftMarkCommandOutputsJSON(t *testing.T) {
	originalFactory := apiClientFactory
	defer func() { apiClientFactory = originalFactory }()

	fake := &commandFakeAPI{}
	fake.restFunc = func(method, path string, params map[string]string, body interface{}, result interface{}) error {
		if method != "GET" {
			return errors.New("unexpected method")
		}
		switch path {
		case "repos/octo/demo":
			payload := map[string]interface{}{"full_name": "octo/demo"}
			return assignJSON(result, payload)
		case "repos/octo/demo/pulls/5":
			payload := map[string]interface{}{"node_id": "PR_node"}
			return assignJSON(result, payload)
		default:
			return errors.New("unexpected path")
		}
	}
	fake.graphqlFunc = func(query string, variables map[string]interface{}, result interface{}) error {
		if strings.Contains(query, "PullRequestStatus") {
			// Mock status check - PR is currently ready
			payload := map[string]interface{}{
				"repository": map[string]interface{}{
					"pullRequest": map[string]interface{}{
						"number":   5,
						"title":    "Test PR",
						"isDraft":  false,
					},
				},
			}
			return assignJSON(result, payload)
		}
		if strings.Contains(query, "PullRequestNodeID") {
			// Mock node ID query
			payload := map[string]interface{}{
				"repository": map[string]interface{}{
					"pullRequest": map[string]interface{}{
						"id": "PR_node_123",
					},
				},
			}
			return assignJSON(result, payload)
		}
		if strings.Contains(query, "convertPullRequestToDraft") {
			// Mock the convert to draft mutation
			payload := map[string]interface{}{
				"convertPullRequestToDraft": map[string]interface{}{
					"pullRequest": map[string]interface{}{
						"number":   5,
						"isDraft":  true,
					},
				},
			}
			return assignJSON(result, payload)
		}
		return errors.New("unexpected query")
	}
	apiClientFactory = func(host string) ghcli.API { return fake }

	cmd := newDraftMarkCommand()
	cmd.SetArgs([]string{"5", "-R", "octo/demo"})
	
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	
	err := cmd.Execute()
	require.NoError(t, err)

	var output map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &output)
	require.NoError(t, err)

	assert.Equal(t, float64(5), output["pr_number"])
	assert.Equal(t, true, output["is_draft"])
	assert.Equal(t, "marked as draft", output["status"])
}

func TestDraftReadyCommandOutputsJSON(t *testing.T) {
	originalFactory := apiClientFactory
	defer func() { apiClientFactory = originalFactory }()

	fake := &commandFakeAPI{}
	fake.restFunc = func(method, path string, params map[string]string, body interface{}, result interface{}) error {
		if method != "GET" {
			return errors.New("unexpected method")
		}
		switch path {
		case "repos/octo/demo":
			payload := map[string]interface{}{"full_name": "octo/demo"}
			return assignJSON(result, payload)
		case "repos/octo/demo/pulls/5":
			payload := map[string]interface{}{"node_id": "PR_node"}
			return assignJSON(result, payload)
		default:
			return errors.New("unexpected path")
		}
	}
	fake.graphqlFunc = func(query string, variables map[string]interface{}, result interface{}) error {
		if strings.Contains(query, "PullRequestStatus") {
			// Mock status check - PR is currently draft
			payload := map[string]interface{}{
				"repository": map[string]interface{}{
					"pullRequest": map[string]interface{}{
						"number":   5,
						"title":    "Test PR",
						"isDraft":  true,
					},
				},
			}
			return assignJSON(result, payload)
		}
		if strings.Contains(query, "PullRequestNodeID") {
			// Mock node ID query
			payload := map[string]interface{}{
				"repository": map[string]interface{}{
					"pullRequest": map[string]interface{}{
						"id": "PR_node_123",
					},
				},
			}
			return assignJSON(result, payload)
		}
		if strings.Contains(query, "markPullRequestReadyForReview") {
			// Mock the mark ready mutation
			payload := map[string]interface{}{
				"markPullRequestReadyForReview": map[string]interface{}{
					"pullRequest": map[string]interface{}{
						"number":   5,
						"isDraft":  false,
					},
				},
			}
			return assignJSON(result, payload)
		}
		return errors.New("unexpected query")
	}
	apiClientFactory = func(host string) ghcli.API { return fake }

	cmd := newDraftReadyCommand()
	cmd.SetArgs([]string{"5", "-R", "octo/demo"})
	
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	
	err := cmd.Execute()
	require.NoError(t, err)

	var output map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &output)
	require.NoError(t, err)

	assert.Equal(t, float64(5), output["pr_number"])
	assert.Equal(t, false, output["is_draft"])
	assert.Equal(t, "marked as ready for review", output["status"])
}

func TestDraftStatusCommandOutputsJSON(t *testing.T) {
	originalFactory := apiClientFactory
	defer func() { apiClientFactory = originalFactory }()

	fake := &commandFakeAPI{}
	fake.restFunc = func(method, path string, params map[string]string, body interface{}, result interface{}) error {
		if method != "GET" {
			return errors.New("unexpected method")
		}
		switch path {
		case "repos/octo/demo":
			payload := map[string]interface{}{"full_name": "octo/demo"}
			return assignJSON(result, payload)
		case "repos/octo/demo/pulls/5":
			payload := map[string]interface{}{"node_id": "PR_node"}
			return assignJSON(result, payload)
		default:
			return errors.New("unexpected path")
		}
	}
	fake.graphqlFunc = func(query string, variables map[string]interface{}, result interface{}) error {
		if !strings.Contains(query, "PullRequestStatus") {
			return errors.New("unexpected query")
		}
		payload := map[string]interface{}{
			"repository": map[string]interface{}{
				"pullRequest": map[string]interface{}{
					"number":   5,
					"title":    "Test Draft PR",
					"isDraft":  true,
				},
			},
		}
		return assignJSON(result, payload)
	}
	apiClientFactory = func(host string) ghcli.API { return fake }

	cmd := newDraftStatusCommand()
	cmd.SetArgs([]string{"5", "-R", "octo/demo"})
	
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	
	err := cmd.Execute()
	require.NoError(t, err)

	var output map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &output)
	require.NoError(t, err)

	assert.Equal(t, float64(5), output["pr_number"])
	assert.Equal(t, true, output["is_draft"])
	assert.Equal(t, "Test Draft PR", output["title"])
}

func TestDraftListCommandOutputsJSON(t *testing.T) {
	originalFactory := apiClientFactory
	defer func() { apiClientFactory = originalFactory }()

	fake := &commandFakeAPI{}
	fake.restFunc = func(method, path string, params map[string]string, body interface{}, result interface{}) error {
		if method != "GET" {
			return errors.New("unexpected method")
		}
		switch path {
		case "repos/octo/demo":
			payload := map[string]interface{}{"full_name": "octo/demo"}
			return assignJSON(result, payload)
		case "repos/octo/demo/pulls/1":
			payload := map[string]interface{}{"node_id": "PR_node"}
			return assignJSON(result, payload)
		default:
			return errors.New("unexpected path")
		}
	}
	fake.graphqlFunc = func(query string, variables map[string]interface{}, result interface{}) error {
		if !strings.Contains(query, "DraftList") {
			return errors.New("unexpected query")
		}
		payload := map[string]interface{}{
			"repository": map[string]interface{}{
				"pullRequests": map[string]interface{}{
					"nodes": []map[string]interface{}{
						{
							"number":   3,
							"title":    "Draft PR 1",
							"isDraft":  true,
						},
						{
							"number":   4,
							"title":    "Ready PR",
							"isDraft":  false,
						},
						{
							"number":   5,
							"title":    "Draft PR 2",
							"isDraft":  true,
						},
					},
				},
			},
		}
		return assignJSON(result, payload)
	}
	apiClientFactory = func(host string) ghcli.API { return fake }

	cmd := newDraftListCommand()
	cmd.SetArgs([]string{"-R", "octo/demo"})
	
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	
	err := cmd.Execute()
	require.NoError(t, err)

	var output []map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &output)
	require.NoError(t, err)

	// Should only return draft PRs (2 out of 3)
	require.Len(t, output, 2)
	
	assert.Equal(t, float64(3), output[0]["pr_number"])
	assert.Equal(t, true, output[0]["is_draft"])
	assert.Equal(t, "Draft PR 1", output[0]["title"])
	
	assert.Equal(t, float64(5), output[1]["pr_number"])
	assert.Equal(t, true, output[1]["is_draft"])
	assert.Equal(t, "Draft PR 2", output[1]["title"])
}

func TestDraftAlreadyInDesiredState(t *testing.T) {
	originalFactory := apiClientFactory
	defer func() { apiClientFactory = originalFactory }()

	fake := &commandFakeAPI{}
	fake.restFunc = func(method, path string, params map[string]string, body interface{}, result interface{}) error {
		if method != "GET" {
			return errors.New("unexpected method")
		}
		switch path {
		case "repos/octo/demo":
			payload := map[string]interface{}{"full_name": "octo/demo"}
			return assignJSON(result, payload)
		case "repos/octo/demo/pulls/5":
			payload := map[string]interface{}{"node_id": "PR_node"}
			return assignJSON(result, payload)
		default:
			return errors.New("unexpected path")
		}
	}
	fake.graphqlFunc = func(query string, variables map[string]interface{}, result interface{}) error {
		if strings.Contains(query, "PullRequestStatus") {
			// Mock status check - PR is already draft
			payload := map[string]interface{}{
				"repository": map[string]interface{}{
					"pullRequest": map[string]interface{}{
						"number":   5,
						"title":    "Test Draft PR",
						"isDraft":  true,
					},
				},
			}
			return assignJSON(result, payload)
		}
		// Should not reach the mutation if already in desired state
		return errors.New("unexpected mutation call")
	}
	apiClientFactory = func(host string) ghcli.API { return fake }

	cmd := newDraftMarkCommand()
	cmd.SetArgs([]string{"5", "-R", "octo/demo"})
	
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	
	err := cmd.Execute()
	require.NoError(t, err)

	var output map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &output)
	require.NoError(t, err)

	assert.Equal(t, float64(5), output["pr_number"])
	assert.Equal(t, true, output["is_draft"])
	assert.Equal(t, "already draft", output["status"])
}

func TestDraftCommandRequiresPRNumber(t *testing.T) {
	cmd := newDraftMarkCommand()
	cmd.SetArgs([]string{"-R", "octo/demo"})
	
	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "pull request number is required")
}

func TestDraftStatusWithSelector(t *testing.T) {
	originalFactory := apiClientFactory
	defer func() { apiClientFactory = originalFactory }()

	fake := &commandFakeAPI{}
	fake.restFunc = func(method, path string, params map[string]string, body interface{}, result interface{}) error {
		if method != "GET" {
			return errors.New("unexpected method")
		}
		switch path {
		case "repos/octo/demo":
			payload := map[string]interface{}{"full_name": "octo/demo"}
			return assignJSON(result, payload)
		case "repos/octo/demo/pulls/5":
			payload := map[string]interface{}{"node_id": "PR_node"}
			return assignJSON(result, payload)
		default:
			return errors.New("unexpected path")
		}
	}
	fake.graphqlFunc = func(query string, variables map[string]interface{}, result interface{}) error {
		if !strings.Contains(query, "PullRequestStatus") {
			return errors.New("unexpected query")
		}
		payload := map[string]interface{}{
			"repository": map[string]interface{}{
				"pullRequest": map[string]interface{}{
					"number":   5,
					"title":    "Test PR",
					"isDraft":  false,
				},
			},
		}
		return assignJSON(result, payload)
	}
	apiClientFactory = func(host string) ghcli.API { return fake }

	cmd := newDraftStatusCommand()
	cmd.SetArgs([]string{"https://github.com/octo/demo/pull/5"})
	
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	
	err := cmd.Execute()
	require.NoError(t, err)

	var output map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &output)
	require.NoError(t, err)

	assert.Equal(t, float64(5), output["pr_number"])
	assert.Equal(t, false, output["is_draft"])
}
