package cachefx

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
	DialectRedis Dialect = "redis"
)

func DetermineDialect(provider string, dsn string) (Dialect, error) {
	if provider != "" {
		switch provider {
		case "redis":
			return DialectRedis, nil
		default:
			return "", fmt.Errorf("%w (provider=%q)", ErrUnknownProvider, provider)
		}
	}

	dsnLower := strings.ToLower(dsn)

	if strings.HasPrefix(dsnLower, "redis://") {
		return DialectRedis, nil
	}

	return "", fmt.Errorf("%w (dsn=%q)", ErrUnableToDetermineDialect, dsn)
}
