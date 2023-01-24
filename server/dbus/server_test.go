//go:build dbus_test
// +build dbus_test

package dbus

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateDBusServer(t *testing.T) {
	dbusServer, err := Start(context.Background(), nil)
	require.NoError(t, err)
	assert.NotNil(t, dbusServer)
}
