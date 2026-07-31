package middleware

import (
	"strconv"
	"time"

	ierr "github.com/flexprice/flexprice/internal/errors"
	"github.com/flexprice/flexprice/internal/metrics"
	"github.com/gin-gonic/gin"
)

func MetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		path := c.FullPath()
		if path == "" {
			// Unmatched routes (404s) — avoid high-cardinality raw URLs.
			path = "unknown"
		}

		method := c.Request.Method
		status := c.Writer.Status()
		duration := time.Since(start)

		metrics.HTTPRequestLatency.WithLabelValues(method, path).Observe(duration.Seconds())
		metrics.HTTPRequestsTotal.WithLabelValues(method, path, strconv.Itoa(status)).Inc()

		if status >= 400 {
			errorCode := ""
			if len(c.Errors) > 0 {
				_, code := ierr.ResolveError(c.Errors.Last().Err)
				errorCode = string(code)
			}
			metrics.HTTPRequestFailuresTotal.WithLabelValues(method, path, errorCode).Inc()
		}
	}
}
