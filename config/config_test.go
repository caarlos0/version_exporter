package config

import (
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestConfigReload(t *testing.T) {
	var config = Config{}
	var reload = false
	Load("testdata/config.yml", &config, func() {
		reload = true
	})

	p, err := os.FindProcess(os.Getpid())
	require.NoError(t, err)
	require.NoError(t, p.Signal(syscall.SIGHUP))
	time.Sleep(500 * time.Millisecond)
	require.True(t, reload)
}
