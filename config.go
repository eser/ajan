package ajan

import (
	"github.com/eser/ajan/connfx"
	"github.com/eser/ajan/grpcfx"
	"github.com/eser/ajan/httpfx"
	"github.com/eser/ajan/logfx"
)

type BaseConfig struct {
	AppName string `conf:"name" default:"ajansvc"`
	AppEnv  string `conf:"env"  default:"development"`

	// JwtSignature      string `conf:"jwt_signature"`
	// CorsOrigin        string `conf:"cors_origin"`
	// CorsStrictHeaders bool   `conf:"cors_strict_headers"`

	Conn connfx.Config `conf:"conn"`
	Log  logfx.Config  `conf:"log"`
	GRPC grpcfx.Config `conf:"grpc"`
	HTTP httpfx.Config `conf:"http"`
}
