package resolver

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeSelector(t *testing.T) {
	selector, err := NormalizeSelector("octo/demo#7", 7)
	require.NoError(t, err)
	assert.Equal(t, "octo/demo#7", selector)

	selector, err = NormalizeSelector("", 42)
	require.NoError(t, err)
	assert.Equal(t, "42", selector)

	_, err = NormalizeSelector("octo/demo#7", 8)
	require.Error(t, err)
}

func TestResolveURL(t *testing.T) {
	id, err := Resolve("https://github.com/octo/demo/pull/9", "", "")
	require.NoError(t, err)
	assert.Equal(t, Identity{Owner: "octo", Repo: "demo", Host: "github.com", Number: 9}, id)
}

func TestResolveFullReference(t *testing.T) {
	id, err := Resolve("octo/demo#5", "", "github.acme.com")
	require.NoError(t, err)
	assert.Equal(t, Identity{Owner: "octo", Repo: "demo", Host: "github.acme.com", Number: 5}, id)
}

func TestResolveNumberRequiresRepo(t *testing.T) {
	_, err := Resolve("7", "", "")
	require.Error(t, err)

	id, err := Resolve("7", "octo/demo", "github.com")
	require.NoError(t, err)
	assert.Equal(t, Identity{Owner: "octo", Repo: "demo", Host: "github.com", Number: 7}, id)
}
