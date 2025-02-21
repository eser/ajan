package cachefx

type ConfigCache struct {
	Provider string `conf:"PROVIDER"`
	DSN      string `conf:"DSN"`
}

type Config struct {
	Caches map[string]ConfigCache `conf:"CACHES"`
}
