package datafx

type ConfigDataSource struct {
	Provider string `conf:"PROVIDER"`
	DSN      string `conf:"DSN"`
}

type Config struct {
	Sources map[string]ConfigDataSource `conf:"SOURCES"`
}
