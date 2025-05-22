package cachefx

type ConfigCache struct {
	Provider string `conf:"provider"`
	DSN      string `conf:"dsn"`
}

type Config struct {
	Caches map[string]ConfigCache `conf:"caches"`
}
