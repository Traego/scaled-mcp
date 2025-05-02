package auth

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traego/scaled-mcp/pkg/utils"
)

type MockAuthInfo struct {
	principalId string
}

func (m *MockAuthInfo) GetPrincipalId() string {
	return m.principalId
}

func TestGetAuthInfo(t *testing.T) {
	t.Run("with auth info in context", func(t *testing.T) {
		mockAuth := &MockAuthInfo{principalId: "test-user"}
		ctx := context.WithValue(context.Background(), utils.AuthInfoCtx, mockAuth)

		result := GetAuthInfo(ctx)

		require.NotNil(t, result)
		assert.Equal(t, "test-user", result.GetPrincipalId())
	})

	t.Run("with nil auth info in context", func(t *testing.T) {
		ctx := context.Background()

		result := GetAuthInfo(ctx)

		assert.Nil(t, result)
	})
}

func TestSetAuthInfo(t *testing.T) {
	mockAuth := &MockAuthInfo{principalId: "test-user"}
	ctx := context.Background()

	newCtx := SetAuthInfo(ctx, mockAuth)

	require.NotNil(t, newCtx)
	authInfo := newCtx.Value(utils.AuthInfoCtx)
	require.NotNil(t, authInfo)
	assert.Equal(t, mockAuth, authInfo)

	retrievedAuth := GetAuthInfo(newCtx)
	require.NotNil(t, retrievedAuth)
	assert.Equal(t, "test-user", retrievedAuth.GetPrincipalId())
}
