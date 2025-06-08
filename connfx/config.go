package connfx

import (
	"errors"
	"time"
)

var (
	ErrInvalidConnectionName     = errors.New("invalid connection name")
	ErrInvalidConnectionBehavior = errors.New("invalid connection behavior")
	ErrInvalidConnectionProtocol = errors.New("invalid connection protocol")
	ErrInvalidDSN                = errors.New("invalid DSN")
	ErrInvalidURL                = errors.New("invalid URL")
	ErrInvalidConfigType         = errors.New("invalid config type")
)

// Config represents the main configuration for connfx.
type Config struct {
	Connections map[string]ConnectionConfigData `conf:"connections"`
}

// ConnectionConfigData represents the configuration data for a connection.
type ConnectionConfigData struct {
	Properties map[string]any `conf:"properties,omitempty"`

	VaultConfig    *VaultConfig `conf:"vault,omitempty"`
	Protocol       string       `conf:"protocol"` // e.g., "postgres", "redis", "http"
	DSN            string       `conf:"dsn,omitempty"`
	URL            string       `conf:"url,omitempty"`
	Host           string       `conf:"host,omitempty"`
	Database       string       `conf:"database,omitempty"`
	Username       string       `conf:"username,omitempty"`
	Password       string       `conf:"password,omitempty"`
	CertFile       string       `conf:"cert_file,omitempty"`
	KeyFile        string       `conf:"key_file,omitempty"`
	CAFile         string       `conf:"ca_file,omitempty"`
	ServiceAccount string       `conf:"service_account,omitempty"`

	// External credential management
	CredentialsSource string        `conf:"credentials_source,omitempty"` // "static", "file", "vault"
	Port              int           `conf:"port,omitempty"`
	Timeout           time.Duration `conf:"timeout,omitempty"`
	MaxRetries        int           `conf:"max_retries,omitempty"`

	// Authentication and security
	TLS           bool `conf:"tls,omitempty"`
	TLSSkipVerify bool `conf:"tls_skip_verify,omitempty"`
}

// VaultConfig represents configuration for external vault services.
type VaultConfig struct {
	Provider string `conf:"provider"` // "azure", "hashicorp", "aws", "gcp"
	Endpoint string `conf:"endpoint"`
	Path     string `conf:"path"`
	Token    string `conf:"token,omitempty"`
}

// BaseConnectionConfig provides common functionality for all connection configs.
type BaseConnectionConfig struct {
	Name     string
	Protocol string
	Data     ConnectionConfigData
}

// NewConnectionConfig creates a new connection configuration.
func NewConnectionConfig(name string, data ConnectionConfigData) *BaseConnectionConfig {
	return &BaseConnectionConfig{
		Name:     name,
		Protocol: data.Protocol,
		Data:     data,
	}
}

func (c *BaseConnectionConfig) GetName() string {
	return c.Name
}

func (c *BaseConnectionConfig) GetProtocol() string {
	return c.Protocol
}

func (c *BaseConnectionConfig) Validate() error {
	if c.Name == "" {
		return ErrInvalidConnectionName
	}

	if c.Protocol == "" {
		return ErrInvalidConnectionProtocol
	}

	return nil
}
