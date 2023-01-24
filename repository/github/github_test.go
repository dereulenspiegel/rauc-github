//go:build github_integration
// +build github_integration

package github

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQueryingReleaseRealWorld(t *testing.T) {
	repo, err := NewRepo("dereulenspiegel", "firmware_craftbeerpi")
	require.NoError(t, err)
	updates, err := repo.Updates(context.Background())
	require.NoError(t, err)
	assert.NotEmpty(t, updates)
}
