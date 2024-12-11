package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	require.True(t, cfg.IService.Enable)
	require.Equal(t, cfg.IService.Address, DefaultIndexerServiceAddress)
}
