package resolver

import (
	"testing"

	"github.com/agynio/gh-pr-review/internal/autodetect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeSelector(t *testing.T) {
	selector, err := NormalizeSelector("https://github.com/octo/demo/pull/7", 7)
	require.NoError(t, err)
	assert.Equal(t, "https://github.com/octo/demo/pull/7", selector)

	selector, err = NormalizeSelector("7", 7)
	require.NoError(t, err)
	assert.Equal(t, "7", selector)

	selector, err = NormalizeSelector("", 42)
	require.NoError(t, err)
	assert.Equal(t, "42", selector)

	_, err = NormalizeSelector("https://github.com/octo/demo/pull/7", 8)
	require.Error(t, err)

	_, err = NormalizeSelector("octo/demo#7", 0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "URL or number")
}

func TestResolveURL(t *testing.T) {
	id, err := Resolve("https://github.com/octo/demo/pull/9", "", "")
	require.NoError(t, err)
	assert.Equal(t, Identity{Owner: "octo", Repo: "demo", Host: "github.com", Number: 9}, id)
}

func TestResolveHostSanitization(t *testing.T) {
	id, err := Resolve("11", "octo/demo", "HTTPS://GHE.EXAMPLE.COM:8443/")
	require.NoError(t, err)
	assert.Equal(t, "ghe.example.com", id.Host)
}

func TestResolveURLHostPrecedence(t *testing.T) {
	id, err := Resolve("https://git.enterprise.local:8443/octo/demo/pull/13", "", "github.com")
	require.NoError(t, err)
	assert.Equal(t, Identity{Owner: "octo", Repo: "demo", Host: "git.enterprise.local", Number: 13}, id)
}

func TestResolveNumberRequiresRepo(t *testing.T) {
	_, err := Resolve("7", "", "")
	require.Error(t, err)

	id, err := Resolve("7", "octo/demo", "github.com")
	require.NoError(t, err)
	assert.Equal(t, Identity{Owner: "octo", Repo: "demo", Host: "github.com", Number: 7}, id)
}

func TestResolveWithAutoDetect_ExplicitValuesPreferred(t *testing.T) {
	autoCtx := &autodetect.Context{
		Owner:  "autoowner",
		Repo:   "autorepo",
		Number: 99,
	}

	// Explicit selector and repo should be used
	id, err := ResolveWithAutoDetect("42", "explicit/repo", "github.com", autoCtx)
	require.NoError(t, err)
	assert.Equal(t, Identity{Owner: "explicit", Repo: "repo", Host: "github.com", Number: 42}, id)
}

func TestResolveWithAutoDetect_UsesAutoDetectedRepo(t *testing.T) {
	autoCtx := &autodetect.Context{
		Owner:  "autoowner",
		Repo:   "autorepo",
		Number: 99,
	}

	// No explicit repo, should use auto-detected
	id, err := ResolveWithAutoDetect("42", "", "github.com", autoCtx)
	require.NoError(t, err)
	assert.Equal(t, Identity{Owner: "autoowner", Repo: "autorepo", Host: "github.com", Number: 42}, id)
}

func TestResolveWithAutoDetect_UsesAutoDetectedPR(t *testing.T) {
	autoCtx := &autodetect.Context{
		Owner:  "autoowner",
		Repo:   "autorepo",
		Number: 99,
	}

	// No selector, should use auto-detected PR number
	id, err := ResolveWithAutoDetect("", "autoowner/autorepo", "github.com", autoCtx)
	require.NoError(t, err)
	assert.Equal(t, Identity{Owner: "autoowner", Repo: "autorepo", Host: "github.com", Number: 99}, id)
}

func TestResolveWithAutoDetect_UsesFullAutoDetection(t *testing.T) {
	autoCtx := &autodetect.Context{
		Owner:  "autoowner",
		Repo:   "autorepo",
		Number: 99,
	}

	// No selector or repo, should use both auto-detected values
	id, err := ResolveWithAutoDetect("", "", "github.com", autoCtx)
	require.NoError(t, err)
	assert.Equal(t, Identity{Owner: "autoowner", Repo: "autorepo", Host: "github.com", Number: 99}, id)
}

func TestResolveWithAutoDetect_NilContext(t *testing.T) {
	// Should fail without auto-detection context
	_, err := ResolveWithAutoDetect("", "", "github.com", nil)
	require.Error(t, err)
}

func TestResolveWithAutoDetect_URLOverridesAutoDetection(t *testing.T) {
	autoCtx := &autodetect.Context{
		Owner:  "autoowner",
		Repo:   "autorepo",
		Number: 99,
	}

	// URL selector should override everything
	id, err := ResolveWithAutoDetect("https://github.com/urlowner/urlrepo/pull/42", "", "github.com", autoCtx)
	require.NoError(t, err)
	assert.Equal(t, Identity{Owner: "urlowner", Repo: "urlrepo", Host: "github.com", Number: 42}, id)
}
