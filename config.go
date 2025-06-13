package ajan

import (
	"github.com/eser/ajan/connfx"
	"github.com/eser/ajan/grpcfx"
	"github.com/eser/ajan/httpclient"
	"github.com/eser/ajan/httpfx"
	"github.com/eser/ajan/logfx"
	"github.com/eser/ajan/metricsfx"
)

type BaseConfig struct {
	Conn    connfx.Config `conf:"conn"`
	AppName string        `conf:"name" default:"ajansvc"`
	AppEnv  string        `conf:"env"  default:"development"`

	// JwtSignature      string `conf:"jwt_signature"`
	// CorsOrigin        string `conf:"cors_origin"`
	// CorsStrictHeaders bool   `conf:"cors_strict_headers"`

	Log        logfx.Config      `conf:"log"`
	Metrics    metricsfx.Config  `conf:"metrics"`
	GRPC       grpcfx.Config     `conf:"grpc"`
	HTTP       httpfx.Config     `conf:"http"`
	HTTPClient httpclient.Config `conf:"http_client"`
}
