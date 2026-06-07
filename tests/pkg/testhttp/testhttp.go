package testhttp

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func DoGet(t *testing.T, client *http.Client, url string) *http.Response {
	t.Helper()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, http.NoBody)
	require.NoError(t, err, "new GET request")

	resp, err := client.Do(req)
	require.NoError(t, err, "GET %s", url)

	t.Cleanup(func() {
		if err := resp.Body.Close(); err != nil {
			t.Logf("close response body: %v", err)
		}
	})

	return resp
}

func DoPost(t *testing.T, client *http.Client, url, body string) *http.Response {
	t.Helper()

	req, err := http.NewRequestWithContext(
		context.Background(), http.MethodPost, url, strings.NewReader(body),
	)
	require.NoError(t, err, "new POST request")

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	require.NoError(t, err, "POST %s", url)

	t.Cleanup(func() {
		if err := resp.Body.Close(); err != nil {
			t.Logf("close response body: %v", err)
		}
	})

	return resp
}
