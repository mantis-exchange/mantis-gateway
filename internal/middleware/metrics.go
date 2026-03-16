package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// Simple metrics counters (no external dependency needed)
var (
	requestCount    = make(map[string]int64)
	requestDuration = make(map[string]float64)
)

// Metrics middleware records basic request metrics.
func Metrics() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		duration := time.Since(start).Seconds()

		key := c.Request.Method + " " + c.FullPath() + " " + strconv.Itoa(c.Writer.Status())
		requestCount[key]++
		requestDuration[key] += duration
	}
}

// MetricsHandler returns a simple /metrics endpoint.
func MetricsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var output string
		for key, count := range requestCount {
			output += "mantis_http_requests_total{route=\"" + key + "\"} " + strconv.FormatInt(count, 10) + "\n"
		}
		for key, dur := range requestDuration {
			output += "mantis_http_request_duration_seconds{route=\"" + key + "\"} " + strconv.FormatFloat(dur, 'f', 6, 64) + "\n"
		}
		c.String(http.StatusOK, output)
	}
}
