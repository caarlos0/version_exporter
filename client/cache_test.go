package client

import (
	"testing"
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/stretchr/testify/require"
)

func TestCachedClient(t *testing.T) {
	c := cache.New(1*time.Minute, 1*time.Minute)
	rel := []Release{
		{
			TagName: "v1.1.1",
		},
	}
	cli := NewCachedClient(cacheTestClient{result: &rel}, c)
	oldRel := rel

	t.Run("get fresh", func(t *testing.T) {
		res, err := cli.Releases("foo")
		require.NoError(t, err)
		require.Equal(t, oldRel, res)
	})

	t.Run("get from cache", func(t *testing.T) {
		rel = append(rel, Release{TagName: "1"})
		res, err := cli.Releases("foo")
		require.NoError(t, err)
		require.Equal(t, oldRel, res)
	})

	t.Run("flush cache", func(t *testing.T) {
		c.Flush()
		res, err := cli.Releases("foo")
		require.NoError(t, err)
		require.Equal(t, rel, res)
	})
}

type cacheTestClient struct {
	result *[]Release
}

func (f cacheTestClient) Releases(repo string) ([]Release, error) {
	return *f.result, nil
}
