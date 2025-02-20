package queuefx

import (
	"errors"
	"fmt"
	"strings"
)

type Dialect string

var (
	ErrUnknownProvider          = errors.New("unknown provider")
	ErrUnableToDetermineDialect = errors.New("unable to determine dialect")
)

const (
	DialectAmqp Dialect = "amqp"
)

func DetermineDialect(provider string, dsn string) (Dialect, error) {
	if provider != "" {
		switch provider {
		case "amqp":
			return DialectAmqp, nil
		default:
			return "", fmt.Errorf("%w - %q", ErrUnknownProvider, provider)
		}
	}

	dsnLower := strings.ToLower(dsn)

	if strings.HasPrefix(dsnLower, "amqp://") {
		return DialectAmqp, nil
	}

	// Default to postgres if cannot determine
	return "", fmt.Errorf("%w - %q", ErrUnableToDetermineDialect, dsn)
}
