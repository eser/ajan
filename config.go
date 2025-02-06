package ajan

import (
	"github.com/eser/ajan/datafx"
	"github.com/eser/ajan/grpcfx"
	"github.com/eser/ajan/httpfx"
	"github.com/eser/ajan/logfx"
	"github.com/eser/ajan/queuefx"
)

type BaseConfig struct {
	AppName string `conf:"NAME" default:"ajansvc"`
	AppEnv  string `conf:"ENV"  default:"development"`

	// JwtSignature      string `conf:"JWT_SIGNATURE"`
	// CorsOrigin        string `conf:"CORS_ORIGIN"`
	// CorsStrictHeaders bool   `conf:"CORS_STRICT_HEADERS"`

	Data  datafx.Config  `conf:"DATA"`
	Queue queuefx.Config `conf:"QUEUE"`
	Log   logfx.Config   `conf:"LOG"`
	Grpc  grpcfx.Config  `conf:"GRPC"`
	Http  httpfx.Config  `conf:"HTTP"`
}
