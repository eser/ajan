package middlewares

import (
	"strconv"

	"github.com/eser/ajan/httpfx"
)

func MetricsMiddleware(httpMetrics *httpfx.Metrics) httpfx.Handler {
	return func(ctx *httpfx.Context) httpfx.Result {
		result := ctx.Next()

		httpMetrics.RequestsTotal.WithLabelValues(
			ctx.Request.Method,
			ctx.Request.URL.Path, strconv.Itoa(result.StatusCode())).Inc()

		return result
	}
}
