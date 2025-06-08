# ajan/datafx

## Overview

**datafx** package is a flexible database access package that provides a
unified interface for different SQL database backends. Currently, it supports
Postgres, MySQL, and SQLite as database backends.

The documentation below provides an overview of the package, its types,
functions, and usage examples. For more detailed information, refer to the
source code and tests.

## Configuration

Configuration struct for the database:

```go
type Config struct {
  Sources map[string]ConfigDatasource `conf:"sources"`
}

type ConfigDatasource struct {
  Provider string `conf:"provider"`
  DSN      string `conf:"dsn"`
}
```

Example configuration:

```go
config := &datafx.Config{
  Sources: map[string]datafx.ConfigDatasource{
    "default": {
      Provider: "postgres",
      DSN:      "postgres://user:pass@localhost:5432/dbname",
    },
    "readonly": {
      Provider: "mysql",
      DSN:      "mysql://user:pass@localhost:3306/dbname",
    },
  },
}
```

## Features

- Multiple SQL database backend support (Postgres, MySQL, SQLite)
- Configurable database dialects
- Managing multiple database instances
- Unit of Work pattern for transaction management
- Easy to extend for additional database backends

## API

### Usage

```go
import "github.com/eser/ajan/datafx"

// Create a new Postgres database instance
db, err := datafx.NewSQLDatasource(ctx, datafx.DialectPostgres, "postgres://localhost:5432/mydb")
if err != nil {
  log.Fatal(err)
}

// Get the database connection
conn := db.GetConnection()

// Use Unit of Work for transaction management
uow, err := db.CreateUnitOfWork(ctx)
if err != nil {
  log.Fatal(err)
}

// Perform operations within transaction
err = uow.Execute(func(tx *sql.Tx) error {
  // Your database operations here
  return nil
})
```

### Supported Dialects

- Postgres (`postgres://`)
- MySQL (`mysql://`)
- SQLite (`sqlite://`)

### Registry

The package includes a registry pattern for managing multiple database
instances. This allows you to register and retrieve database connections by
name:

```go
// Register a database instance
err := datafx.Register("main", db)

// Get a registered database instance
db, err := datafx.Get("main")
```
