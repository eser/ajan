package queuefx

type ConfigBroker struct {
	Provider string `conf:"provider"`
	DSN      string `conf:"dsn"`
}

type Config struct {
	Brokers map[string]ConfigBroker `conf:"brokers"`
}
