package client

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/pkg/errors"
)

// NewClient returns a new github client
func NewClient(token string) Client {
	return githubClient{
		token: token,
	}
}

type githubClient struct {
	token string
}

func (c githubClient) Releases(repo string) ([]Release, error) {
	var releases []Release
	req, _ := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("https://api.github.com/repos/%s/releases", repo),
		nil,
	)
	if c.token != "" {
		req.Header.Add("Authorization", fmt.Sprintf("token %s", c.token))
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return releases, errors.Wrap(err, "failed to get repository releases")
	}
	if resp.StatusCode != http.StatusOK {
		return releases, errors.Wrap(err, "github responded a non-200 status code")
	}
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return releases, errors.Wrap(err, "failed to parse the response body")
	}
	return releases, nil
}
