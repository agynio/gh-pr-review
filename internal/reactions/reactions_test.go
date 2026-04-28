package reactions

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateValidReactions(t *testing.T) {
	for name := range ValidReactions {
		t.Run(name, func(t *testing.T) {
			err := Validate(name)
			require.NoError(t, err)
		})
	}
}

func TestValidateInvalidReaction(t *testing.T) {
	err := Validate("not_a_reaction")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid reaction")
}

func TestValidateEmptyReaction(t *testing.T) {
	err := Validate("")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid reaction")
}

func TestValidReactionNamesSorted(t *testing.T) {
	names := ValidReactionNames()
	require.NotEmpty(t, names)
	for i := 1; i < len(names); i++ {
		assert.True(t, names[i-1] < names[i],
			"expected sorted order, got %q before %q", names[i-1], names[i])
	}
}

func TestValidReactionNamesContainsAll(t *testing.T) {
	names := ValidReactionNames()
	assert.Len(t, names, len(ValidReactions))

	set := make(map[string]bool, len(names))
	for _, n := range names {
		set[n] = true
	}
	for key := range ValidReactions {
		assert.True(t, set[key], "missing key %q in ValidReactionNames()", key)
	}
}

func TestReactInvalidReaction(t *testing.T) {
	err := React(nil, "node123", "not_a_reaction")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid reaction")
}

func TestValidReactionsMapIsComplete(t *testing.T) {
	// All expected reaction types must be present
	expected := []string{
		"thumbs_up", "thumbs_down", "laugh", "hooray",
		"confused", "heart", "rocket", "eyes",
	}
	for _, e := range expected {
		_, ok := ValidReactions[e]
		assert.True(t, ok, "missing reaction %q", e)
	}
}
