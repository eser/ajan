package middlewares

import (
	"strconv"

	"github.com/eser/ajan/httpfx"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

func MetricsMiddleware(httpMetrics *httpfx.Metrics) httpfx.Handler {
	return func(ctx *httpfx.Context) httpfx.Result {
		result := ctx.Next()

		attrs := metric.WithAttributes(
			attribute.String("method", ctx.Request.Method),
			attribute.String("endpoint", ctx.Request.URL.Path),
			attribute.String("status", strconv.Itoa(result.StatusCode())),
		)

		httpMetrics.RequestsTotal.Add(ctx.Request.Context(), 1, attrs)

		return result
	}
}
