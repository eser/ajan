package datafx

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
	// DialectPostgresPgx Dialect = "pgx".
	DialectPostgres Dialect = "postgres"
	DialectSQLite   Dialect = "sqlite"
	DialectMySQL    Dialect = "mysql"
)

func DetermineDialect(provider string, dsn string) (Dialect, error) {
	if provider != "" {
		switch provider {
		case "postgres":
			return DialectPostgres, nil
		case "mysql":
			return DialectMySQL, nil
		case "sqlite":
			return DialectSQLite, nil
		default:
			return "", fmt.Errorf("%w: %s", ErrUnknownProvider, provider)
		}
	}

	dsnLower := strings.ToLower(dsn)

	// if strings.HasPrefix(dsnLower, "pgx://") {
	// 	return DialectPostgresPgx
	// }

	if strings.HasPrefix(dsnLower, "postgres://") {
		return DialectPostgres, nil
	}

	if strings.HasPrefix(dsnLower, "mysql://") {
		return DialectMySQL, nil
	}

	if strings.HasPrefix(dsnLower, "sqlite://") {
		return DialectSQLite, nil
	}

	// Default to postgres if cannot determine
	return "", fmt.Errorf("%w: %s", ErrUnableToDetermineDialect, dsn)
}
