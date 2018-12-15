package client

import "time"

// Release from github api
type Release struct {
	TagName     string    `json:"tag_name,omitempty"`
	Draft       bool      `json:"draft,omitempty"`
	Prerelease  bool      `json:"prerelease,omitempty"`
	PublishedAt time.Time `json:"published_at,omitempty"`
}

// Client a client
type Client interface {
	// Releases returns all releases for a given repository
	Releases(repo string) ([]Release, error)
}
