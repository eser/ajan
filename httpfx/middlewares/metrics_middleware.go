package middlewares

import (
	"strconv"
	"time"

	"github.com/eser/ajan/httpfx"
	"github.com/eser/ajan/metricsfx"
)

// MetricsMiddleware creates HTTP metrics middleware using the simplified MetricsBuilder approach.
func MetricsMiddleware(httpMetrics *httpfx.Metrics) httpfx.Handler {
	return func(ctx *httpfx.Context) httpfx.Result {
		startTime := time.Now()

		result := ctx.Next()

		duration := time.Since(startTime)

		// Use the new HTTP-specific attribute helpers
		attrs := metricsfx.HTTPAttrs(
			ctx.Request.Method,
			ctx.Request.URL.Path,
			strconv.Itoa(result.StatusCode()),
		)

		// Clean, simple metric recording
		httpMetrics.RequestsTotal.Inc(ctx.Request.Context(), attrs...)
		httpMetrics.RequestDuration.RecordDuration(ctx.Request.Context(), duration, attrs...)

		return result
	}
}
