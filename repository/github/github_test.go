package github

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQueryingReleaseRealWorld(t *testing.T) {
	t.Skipf("Just for testing things on the GitHub API, not a stable test")
	repo, err := NewRepo("dereulenspiegel", "firmware_craftbeerpi")
	require.NoError(t, err)
	updates, err := repo.Updates(context.Background())
	require.NoError(t, err)
	assert.NotEmpty(t, updates)
}
