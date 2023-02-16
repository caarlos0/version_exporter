package config

import (
	"os"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestConfigReload(t *testing.T) {
	config := Config{}
	var n int32
	Load("testdata/config.yml", &config, func() {
		atomic.AddInt32(&n, 1)
	})

	p, err := os.FindProcess(os.Getpid())
	require.NoError(t, err)
	require.NoError(t, p.Signal(syscall.SIGHUP))
	time.Sleep(500 * time.Millisecond)
	require.Equal(t, int32(1), atomic.LoadInt32(&n))
}
