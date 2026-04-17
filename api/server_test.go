package api

import (
	"testing"
	"time"

	"github.com/simplebank/util"
	"github.com/stretchr/testify/require"
)

func TestNewServer_InvalidTokenKey(t *testing.T) {
	config := util.Config{
		TokenSymmetricKey:   "short-key",
		AccessTokenDuration: time.Minute,
	}

	server, err := NewServer(config, nil)
	require.Error(t, err)
	require.Nil(t, server)
	require.Contains(t, err.Error(), "cannot create token maker")
}

func TestNewServer_RegisterValidationError(t *testing.T) {
	originalValidCurrency := validCurrency
	validCurrency = nil
	t.Cleanup(func() {
		validCurrency = originalValidCurrency
	})

	config := util.Config{
		TokenSymmetricKey:   util.RandomString(32),
		AccessTokenDuration: time.Minute,
	}

	server, err := NewServer(config, nil)
	require.Error(t, err)
	require.Nil(t, server)
	require.Contains(t, err.Error(), "cannot register currency validation")
}

func TestServerStart_InvalidAddress(t *testing.T) {
	config := util.Config{
		TokenSymmetricKey:   util.RandomString(32),
		AccessTokenDuration: time.Minute,
	}

	server, err := NewServer(config, nil)
	require.NoError(t, err)
	require.NotNil(t, server)

	err = server.Start("127.0.0.1:99999")
	require.Error(t, err)
}
