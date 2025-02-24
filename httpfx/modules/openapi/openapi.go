package openapi

import (
	"github.com/eser/ajan/httpfx"
)

type ApiIdentity struct {
	name    string
	version string
}

func RegisterHttpRoutes(routes *httpfx.Router, config *httpfx.Config) {
	if !config.OpenApiEnabled {
		return
	}

	routes.
		Route("GET /openapi.json", func(ctx *httpfx.Context) httpfx.Result {
			spec := &ApiIdentity{
				name:    "golang-service",
				version: "0.0.0",
			}

			result := GenerateOpenApiSpec(spec, routes)

			return ctx.Results.Json(result)
		}).
		HasSummary("OpenAPI Spec").
		HasDescription("OpenAPI Spec Endpoint")
}
