package client

// NewFakeClient returns a new fake client
func NewFakeClient(result []Release, err error) Client {
	return fakeClient{
		result: result,
		err:    err,
	}
}

type fakeClient struct {
	result []Release
	err    error
}

func (f fakeClient) Releases(repo string) ([]Release, error) {
	return f.result, f.err
}
