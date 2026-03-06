package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveBody_OnlyBody(t *testing.T) {
	got, err := resolveBody("hello", "")
	require.NoError(t, err)
	assert.Equal(t, "hello", got)
}

func TestResolveBody_NeitherSet(t *testing.T) {
	got, err := resolveBody("", "")
	require.NoError(t, err)
	assert.Equal(t, "", got)
}

func TestResolveBody_BothSet(t *testing.T) {
	_, err := resolveBody("hello", "file.txt")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "specify --body or --body-file, not both")
}

func TestResolveBody_FromFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "body.txt")
	require.NoError(t, os.WriteFile(p, []byte("from file"), 0600))

	got, err := resolveBody("", p)
	require.NoError(t, err)
	assert.Equal(t, "from file", got)
}

func TestResolveBody_FromFileNotFound(t *testing.T) {
	_, err := resolveBody("", "/nonexistent/path/body.txt")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read body file")
}

func TestResolveBody_FromStdin(t *testing.T) {
	// Replace os.Stdin with a pipe containing test data.
	r, w, err := os.Pipe()
	require.NoError(t, err)

	origStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = origStdin }()

	_, err = w.WriteString("stdin content")
	require.NoError(t, err)
	require.NoError(t, w.Close())

	got, err := resolveBody("", "-")
	require.NoError(t, err)
	assert.Equal(t, "stdin content", got)
}
