package collector

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/caarlos0/version_exporter/client"
	"github.com/caarlos0/version_exporter/config"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/stretchr/testify/require"
)

func TestCollectorError(t *testing.T) {
	config := config.Config{
		Repositories: map[string]string{
			"foo": "v0.1.1",
		},
	}
	client := client.NewFakeClient([]client.Release{}, fmt.Errorf("failed to blah"))
	testCollector(t, NewVersionCollector(&config, client), func(t *testing.T, status int, body string) {
		require.Equal(t, 200, status)
		require.Contains(t, body, "version_up 0")
	})
}

func TestRepoUpToDate(t *testing.T) {
	config := config.Config{
		Repositories: map[string]string{
			"foo": "v0.1.1",
		},
	}
	client := client.NewFakeClient([]client.Release{
		{
			TagName: "v0.1.1",
		},
	}, nil)
	testCollector(t, NewVersionCollector(&config, client), func(t *testing.T, status int, body string) {
		require.Equal(t, 200, status)
		require.Contains(t, body, "version_up 1")
		require.Contains(t, body, `version_up_to_date{constraint="v0.1.1",latest="0.1.1",repository="foo"} 1`)
	})
}

func TestRepoOutOfDate(t *testing.T) {
	config := config.Config{
		Repositories: map[string]string{
			"foo": "v0.1.1",
		},
	}
	client := client.NewFakeClient([]client.Release{
		{
			TagName: "v0.1.2",
		},
	}, nil)
	testCollector(t, NewVersionCollector(&config, client), func(t *testing.T, status int, body string) {
		require.Equal(t, 200, status)
		require.Contains(t, body, "version_up 1")
		require.Contains(t, body, `version_up_to_date{constraint="v0.1.1",latest="0.1.2",repository="foo"} 0`)
	})
}

func TestDraftRelease(t *testing.T) {
	config := config.Config{
		Repositories: map[string]string{
			"foo": "v0.1.1",
		},
	}
	client := client.NewFakeClient([]client.Release{
		{
			TagName: "v0.1.2",
			Draft:   true,
		},
		{
			TagName: "v0.1.1",
		},
	}, nil)
	testCollector(t, NewVersionCollector(&config, client), func(t *testing.T, status int, body string) {
		require.Equal(t, 200, status)
		require.Contains(t, body, "version_up 1")
		require.Contains(t, body, `version_up_to_date{constraint="v0.1.1",latest="0.1.1",repository="foo"} 1`)
	})
}

func TestPrerelease(t *testing.T) {
	config := config.Config{
		Repositories: map[string]string{
			"foo": "v0.1.1",
		},
	}
	client := client.NewFakeClient([]client.Release{
		{
			TagName:    "v0.1.2",
			Prerelease: true,
		},
		{
			TagName: "v0.1.1",
		},
	}, nil)
	testCollector(t, NewVersionCollector(&config, client), func(t *testing.T, status int, body string) {
		require.Equal(t, 200, status)
		require.Contains(t, body, "version_up 1")
		require.Contains(t, body, `version_up_to_date{constraint="v0.1.1",latest="0.1.1",repository="foo"} 1`)
	})
}

func TestTagWithPrelease(t *testing.T) {
	config := config.Config{
		Repositories: map[string]string{
			"foo": "v0.1.1",
		},
	}
	client := client.NewFakeClient([]client.Release{
		{
			TagName: "v0.1.2-beta",
		},
		{
			TagName: "v0.1.1",
		},
	}, nil)
	testCollector(t, NewVersionCollector(&config, client), func(t *testing.T, status int, body string) {
		require.Equal(t, 200, status)
		require.Contains(t, body, "version_up 1")
		require.Contains(t, body, `version_up_to_date{constraint="v0.1.1",latest="0.1.1",repository="foo"} 1`)
	})
}

func TestInvalidConstraintOnConfig(t *testing.T) {
	config := config.Config{
		Repositories: map[string]string{
			"foo": "invalid-tag-on-config",
		},
	}
	client := client.NewFakeClient([]client.Release{
		{
			TagName: "v0.1.1",
		},
	}, nil)
	testCollector(t, NewVersionCollector(&config, client), func(t *testing.T, status int, body string) {
		require.Equal(t, 200, status)
		require.Contains(t, body, "version_up 0")
	})
}

func TestInvalidSemVerOnRelease(t *testing.T) {
	config := config.Config{
		Repositories: map[string]string{
			"foo": "1.2.0",
		},
	}
	client := client.NewFakeClient([]client.Release{
		{
			TagName: "invalid-tag-on-release",
		},
	}, nil)
	testCollector(t, NewVersionCollector(&config, client), func(t *testing.T, status int, body string) {
		require.Equal(t, 200, status)
		require.Contains(t, body, "version_up 1")
	})
}

func testCollector(t *testing.T, collector prometheus.Collector, checker func(t *testing.T, status int, body string)) {
	registry := prometheus.NewRegistry()
	registry.MustRegister(collector)

	srv := httptest.NewServer(promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	defer srv.Close()

	resp, err := http.Get(srv.URL)
	require.NoError(t, err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	checker(t, resp.StatusCode, string(body))
}
