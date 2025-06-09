package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReceiveMetrics(t *testing.T) {
	type want struct {
		code        int
		contentType string
	}
	tests := []struct {
		name    string
		want    want
		request string
	}{
		{
			name:    "positive test #1",
			request: "/counter/123/123",
			want: want{
				code:        200,
				contentType: "text/plain",
			},
		},
		{
			name:    "negative test #1",
			request: "/sdfsf/BuckHashSys/123",
			want: want{
				code:        400,
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:    "negative test #2",
			request: "/gauge//123",
			want: want{
				code:        404,
				contentType: "text/plain; charset=utf-8",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, test.request, nil)
			// создаём новый Recorder
			w := httptest.NewRecorder()

			handler := MetricsHandler{}
			handler.ReceiveMetrics(w, request)

			res := w.Result()
			// проверяем код ответа
			assert.Equal(t, test.want.code, res.StatusCode)
			// получаем и проверяем тело запроса
			defer res.Body.Close()
			assert.Equal(t, test.want.contentType, res.Header.Get("Content-Type"))
		})
	}
}
