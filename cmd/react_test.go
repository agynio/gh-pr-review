package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"

	"github.com/agynio/gh-pr-review/internal/ghcli"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReactCommandValid(t *testing.T) {
	originalFactory := apiClientFactory
	defer func() { apiClientFactory = originalFactory }()

	var capturedQuery string
	var capturedVars map[string]interface{}

	fake := &commandFakeAPI{}
	fake.graphqlFunc = func(query string, variables map[string]interface{}, result interface{}) error {
		capturedQuery = query
		capturedVars = variables
		return nil
	}
	apiClientFactory = func(host string) ghcli.API { return fake }

	root := newRootCommand()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetArgs([]string{"react", "IC_kwDOTest", "--type", "thumbs_up"})

	err := root.Execute()
	require.NoError(t, err)

	// Verify the GraphQL mutation was called
	assert.Contains(t, capturedQuery, "addReaction")
	assert.Equal(t, "IC_kwDOTest", capturedVars["subjectId"])
	assert.Equal(t, "THUMBS_UP", capturedVars["content"])

	// Verify JSON output
	var output map[string]string
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &output))
	assert.Equal(t, "IC_kwDOTest", output["node_id"])
	assert.Equal(t, "thumbs_up", output["reaction"])
	assert.Equal(t, "added", output["status"])
}

func TestReactCommandInvalidType(t *testing.T) {
	originalFactory := apiClientFactory
	defer func() { apiClientFactory = originalFactory }()

	fake := &commandFakeAPI{}
	apiClientFactory = func(host string) ghcli.API { return fake }

	root := newRootCommand()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetArgs([]string{"react", "IC_kwDOTest", "--type", "invalid_reaction"})

	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid reaction")
}

func TestReactCommandMissingNodeID(t *testing.T) {
	originalFactory := apiClientFactory
	defer func() { apiClientFactory = originalFactory }()

	root := newRootCommand()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetArgs([]string{"react", "--type", "thumbs_up"})

	err := root.Execute()
	// cobra treats --type as the positional arg when no node-id is given,
	// or fails on ExactArgs(1) — either way it should error
	require.Error(t, err)
}

func TestReactCommandMissingType(t *testing.T) {
	originalFactory := apiClientFactory
	defer func() { apiClientFactory = originalFactory }()

	root := newRootCommand()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetArgs([]string{"react", "IC_kwDOTest"})

	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "required flag")
}

func TestReactCommandGraphQLError(t *testing.T) {
	originalFactory := apiClientFactory
	defer func() { apiClientFactory = originalFactory }()

	fake := &commandFakeAPI{}
	fake.graphqlFunc = func(query string, variables map[string]interface{}, result interface{}) error {
		return errors.New("GraphQL: Could not resolve to a node")
	}
	apiClientFactory = func(host string) ghcli.API { return fake }

	root := newRootCommand()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetArgs([]string{"react", "INVALID_NODE", "--type", "heart"})

	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Could not resolve to a node")
}

func TestReactCommandAllReactionTypes(t *testing.T) {
	originalFactory := apiClientFactory
	defer func() { apiClientFactory = originalFactory }()

	reactionTypes := []struct {
		cli     string
		graphql string
	}{
		{"thumbs_up", "THUMBS_UP"},
		{"thumbs_down", "THUMBS_DOWN"},
		{"laugh", "LAUGH"},
		{"hooray", "HOORAY"},
		{"confused", "CONFUSED"},
		{"heart", "HEART"},
		{"rocket", "ROCKET"},
		{"eyes", "EYES"},
	}

	for _, rt := range reactionTypes {
		t.Run(rt.cli, func(t *testing.T) {
			var capturedContent string
			fake := &commandFakeAPI{}
			fake.graphqlFunc = func(query string, variables map[string]interface{}, result interface{}) error {
				capturedContent = variables["content"].(string)
				return nil
			}
			apiClientFactory = func(host string) ghcli.API { return fake }

			root := newRootCommand()
			stdout := &bytes.Buffer{}
			root.SetOut(stdout)
			root.SetErr(&bytes.Buffer{})
			root.SetArgs([]string{"react", "IC_test", "--type", rt.cli})

			err := root.Execute()
			require.NoError(t, err)
			assert.Equal(t, rt.graphql, capturedContent)
		})
	}
}
