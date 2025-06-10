package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testRequest(t *testing.T, ts *httptest.Server, method, path string) (*http.Response, string) {
	req, err := http.NewRequest(method, ts.URL+path, nil)
	require.NoError(t, err)

	resp, err := ts.Client().Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return resp, string(respBody)
}

func TestRouter(t *testing.T) {
	ts := httptest.NewServer(MetricRouter())
	defer ts.Close()
	var testTable = []struct {
		url    string
		status int
	}{
		{"/update/counter/PollCount/123", http.StatusOK},
		{"/update/gauge/123.3", http.StatusNotFound},
		{"/update/blabla/PollCount/123.3", http.StatusBadRequest},
		{"/update/counter/PollCount/123.3", http.StatusBadRequest},
	}
	for _, v := range testTable {
		resp, _ := testRequest(t, ts, "POST", v.url)
		assert.Equal(t, v.status, resp.StatusCode)
		resp.Body.Close()
	}
}
