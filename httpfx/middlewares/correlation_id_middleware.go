package middlewares

import (
	"github.com/eser/ajan/httpfx"
	"github.com/eser/ajan/lib"
)

const CorrelationIDHeader = "X-Correlation-ID"

func CorrelationIDMiddleware() httpfx.Handler {
	return func(ctx *httpfx.Context) httpfx.Result {
		// FIXME(@eser) no need to check if the header is specified
		correlationID := ctx.Request.Header.Get(CorrelationIDHeader)
		if correlationID == "" {
			correlationID = lib.IDsGenerateUnique()
		}

		result := ctx.Next()

		ctx.ResponseWriter.Header().Set(CorrelationIDHeader, correlationID)

		return result
	}
}
