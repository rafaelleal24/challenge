package middleware

import (
	"bytes"
	"context"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rafaelleal24/challenge/internal/core/logger"
)

func logHTTPRequest(ctx context.Context, method, path, route string, statusCode int, duration time.Duration, extraAttributes map[string]interface{}) {
	attrs := map[string]interface{}{
		"http.method":      method,
		"http.path":        path,
		"http.route":       route,
		"http.status_code": statusCode,
		"http.duration_ms": duration.Milliseconds(),
	}

	for key, value := range extraAttributes {
		attrs[key] = value
	}

	level := logger.LogLevelInfo
	if statusCode >= 500 {
		level = logger.LogLevelError
	} else if statusCode >= 400 {
		level = logger.LogLevelWarn
	}

	logger.Log(ctx, logger.LogEntry{
		Level:      level,
		Message:    "HTTP Request",
		Attributes: attrs,
	})
}

var bufferPool = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

const maxResponseBodySize = 250 * 1024 // 250KB

type responseBodyWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *responseBodyWriter) Write(b []byte) (int, error) {
	if w.body.Len()+len(b) <= maxResponseBodySize {
		w.body.Write(b)
	}
	return w.ResponseWriter.Write(b)
}

func (w *responseBodyWriter) WriteString(s string) (int, error) {
	if w.body.Len()+len(s) <= maxResponseBodySize {
		w.body.WriteString(s)
	}
	return w.ResponseWriter.WriteString(s)
}

func LogRequest() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		buf := bufferPool.Get().(*bytes.Buffer)
		defer bufferPool.Put(buf)
		buf.Reset()
		bodyWriter := &responseBodyWriter{
			ResponseWriter: c.Writer,
			body:           buf,
		}
		c.Writer = bodyWriter

		c.Next()

		duration := time.Since(start)

		extraAttributes := map[string]interface{}{}

		if contentLength := c.Request.Header.Get("Content-Length"); contentLength != "" {
			if size, err := strconv.ParseInt(contentLength, 10, 64); err == nil {
				extraAttributes["http.request_size"] = size
			}
		}

		contentType := c.Writer.Header().Get("Content-Type")
		if strings.Contains(contentType, "application/json") && bodyWriter.body.Len() > 0 && bodyWriter.body.Len() <= maxResponseBodySize {
			extraAttributes["http.response_body"] = bodyWriter.body.String()
			extraAttributes["http.response_size"] = bodyWriter.body.Len()
		}

		logHTTPRequest(
			c.Request.Context(),
			c.Request.Method,
			c.Request.URL.Path,
			c.FullPath(),
			c.Writer.Status(),
			duration,
			extraAttributes,
		)
	}
}
