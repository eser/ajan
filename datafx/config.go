package datafx

type ConfigDatasource struct {
	Provider string `conf:"PROVIDER"`
	DSN      string `conf:"DSN"`
}

type Config struct {
	Sources map[string]ConfigDatasource `conf:"SOURCES"`
}
