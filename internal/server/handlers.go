package server

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type (
	ResponseData struct {
		Status int
		Size   int
	}
	LoggingResponseWriter struct {
		http.ResponseWriter
		ResponseData *ResponseData
	}
)

func (r *LoggingResponseWriter) Write(b []byte) (int, error) {
	size, err := r.ResponseWriter.Write(b)
	r.ResponseData.Size += size
	return size, err
}

func (r *LoggingResponseWriter) WriteHeader(statusCode int) {
	r.ResponseWriter.WriteHeader(statusCode)
	r.ResponseData.Status = statusCode
}

type gzipWriter struct {
	http.ResponseWriter
	Writer io.Writer
}

func (w gzipWriter) Write(b []byte) (int, error) {
	typeForGzip := []string{"application/json", "text/html"}
	contentTypes := strings.Join(w.Header().Values("Content-Type"), " ")
	for _, value := range typeForGzip {
		if strings.Contains(contentTypes, value) {
			w.Header().Set("Content-Encoding", "gzip")
			return w.Writer.Write(b)
		}
	}
	return w.ResponseWriter.Write(b)
}

func GzipHandle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if strings.Compare(r.Header.Get("Content-Encoding"), "gzip") == 0 {
			gz, err := gzip.NewReader(r.Body)
			if err != nil {
				fmt.Println("error create gzip")
				return
			}
			decompressBody, err := io.ReadAll(gz)
			if err != nil {
				fmt.Println("error decompress body")
				return
			}
			gz.Close()
			r.Body = io.NopCloser(bytes.NewReader(decompressBody))
			r.ContentLength = int64(len(decompressBody))

		}

		//support compression client check
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}
		gz, err := gzip.NewWriterLevel(w, gzip.BestSpeed)
		if err != nil {
			io.WriteString(w, err.Error())
			return
		}
		defer gz.Close()

		next.ServeHTTP(gzipWriter{ResponseWriter: w, Writer: gz}, r)
	})
}
