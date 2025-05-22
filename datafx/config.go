package datafx

type ConfigDatasource struct {
	Provider string `conf:"provider"`
	DSN      string `conf:"dsn"`
}

type Config struct {
	Sources map[string]ConfigDatasource `conf:"sources"`
}
