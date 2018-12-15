package client

import (
	"github.com/patrickmn/go-cache"
	"github.com/prometheus/common/log"
)

// NewCachedClient returns a new cached client
func NewCachedClient(client Client, cache *cache.Cache) Client {
	return cachedClient{
		client: client,
		cache:  cache,
	}
}

type cachedClient struct {
	client Client
	cache  *cache.Cache
}

func (c cachedClient) Releases(repo string) ([]Release, error) {
	cached, found := c.cache.Get(repo)
	if found {
		log.Debugf("using result from cache for %s", repo)
		return cached.([]Release), nil
	}
	log.Debugf("using result from API for %s", repo)
	live, err := c.client.Releases(repo)
	c.cache.Set(repo, live, cache.DefaultExpiration)
	return live, err
}
