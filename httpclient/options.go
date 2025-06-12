package httpclient

type NewClientOption func(*Client)

func WithConfig(config *Config) NewClientOption {
	return func(client *Client) {
		client.Config = config
	}
}
