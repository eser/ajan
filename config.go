package ajan

import (
	"github.com/eser/ajan/cachefx"
	"github.com/eser/ajan/datafx"
	"github.com/eser/ajan/grpcfx"
	"github.com/eser/ajan/httpfx"
	"github.com/eser/ajan/logfx"
	"github.com/eser/ajan/queuefx"
)

type BaseConfig struct {
	AppName string `conf:"name" default:"ajansvc"`
	AppEnv  string `conf:"env"  default:"development"`

	// JwtSignature      string `conf:"jwt_signature"`
	// CorsOrigin        string `conf:"cors_origin"`
	// CorsStrictHeaders bool   `conf:"cors_strict_headers"`

	Data  datafx.Config  `conf:"data"`
	Cache cachefx.Config `conf:"cache"`
	Queue queuefx.Config `conf:"queue"`
	Log   logfx.Config   `conf:"log"`
	GRPC  grpcfx.Config  `conf:"grpc"`
	HTTP  httpfx.Config  `conf:"http"`
}
