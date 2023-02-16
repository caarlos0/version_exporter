package client

import "github.com/patrickmn/go-cache"

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
		return cached.([]Release), nil
	}
	live, err := c.client.Releases(repo)
	c.cache.Set(repo, live, cache.DefaultExpiration)
	return live, err
}
