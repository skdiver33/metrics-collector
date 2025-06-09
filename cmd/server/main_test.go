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

// func TestReceiveMetricsHandler(t *testing.T) {
// 	type want struct {
// 		code        int
// 		contentType string
// 	}
// 	tests := []struct {
// 		name    string
// 		want    want
// 		request string
// 	}{
// 		{
// 			name:    "positive test #1",
// 			request: "/update/counter/PollCount/123",
// 			want: want{
// 				code:        200,
// 				contentType: "text/plain",
// 			},
// 		},
// 		{
// 			name:    "negative test #1",
// 			request: "/sdfsf/BuckHashSys/123",
// 			want: want{
// 				code:        400,
// 				contentType: "text/plain; charset=utf-8",
// 			},
// 		},
// 		{
// 			name:    "negative test #2",
// 			request: "/gauge//123",
// 			want: want{
// 				code:        404,
// 				contentType: "text/plain; charset=utf-8",
// 			},
// 		},
// 	}

// 	for _, test := range tests {
// 		t.Run(test.name, func(t *testing.T) {
// 			request := httptest.NewRequest(http.MethodPost, test.request, nil)
// 			// создаём новый Recorder
// 			w := httptest.NewRecorder()

// 			handler := MetricsHandler{}
// 			handler.metricsStorage.InitializeStorage()
// 			handler.receiveMetricsHandler(w, request)

// 			res := w.Result()
// 			// проверяем код ответа
// 			assert.Equal(t, test.want.code, res.StatusCode)
// 			// получаем и проверяем тело запроса
// 			defer res.Body.Close()
// 			assert.Equal(t, test.want.contentType, res.Header.Get("Content-Type"))
// 		})
// 	}
// }
