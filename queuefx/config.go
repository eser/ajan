package queuefx

type ConfigBroker struct {
	Provider string `conf:"PROVIDER"`
	DSN      string `conf:"DSN"`
}

type Config struct {
	Brokers map[string]ConfigBroker `conf:"BROKERS"`
}
