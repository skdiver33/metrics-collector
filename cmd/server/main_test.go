package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/skdiver33/metrics-collector/internal/server"
	"github.com/skdiver33/metrics-collector/models"
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

func TestServer(t *testing.T) {

	newServer, err := server.NewServer()

	if err != nil {
		t.Error("error inialize server")
	}
	ts := httptest.NewServer(newServer.HandlersRouter)
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

func TestServerAudit(t *testing.T) {
	os.Setenv("AUDIT_FILE", "./auditfile")
	newServer, err := server.NewServer()
	if err != nil {
		t.Error("error inialize server")
	}
	ts := httptest.NewServer(newServer.HandlersRouter)
	defer ts.Close()
	testMetrics := models.Metrics{ID: "Lookups", MType: models.Gauge}
	testMetrics.SetMetricsValue("123.5")
	data, err := json.Marshal(testMetrics)
	if err != nil {
		t.Fatalf("could not prepare test json metrics")
	}
	tests := []struct {
		name        string
		requestData []byte
		want        int
	}{
		{
			name:        "positive update",
			requestData: data,
			want:        200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			request, err := http.NewRequest(http.MethodPost, ts.URL+"/update", bytes.NewReader(tt.requestData))
			if err != nil {
				t.Fatalf("err create new request: %v", err)
			}
			request.Header.Add("Content-Type", "application/json")

			res, err := ts.Client().Do(request)
			if err != nil {
				t.Fatalf("could not send request %s", err.Error())
			}
			defer res.Body.Close()

			assert.Equal(t, tt.want, res.StatusCode)
			defer res.Body.Close()

			if err != nil {
				t.Fatalf("could not construct receiver type: %v", err)
			}

		})
	}
}
