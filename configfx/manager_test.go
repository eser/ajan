package configfx_test

import (
	"reflect"
	"testing"

	"github.com/eser/ajan/configfx"
	"github.com/stretchr/testify/assert"
)

type TestConfig struct {
	Host string `conf:"host" default:"localhost"`
}

type TestConfigNested struct {
	TestConfig
	Port     int    `conf:"port"      default:"8080"`
	MaxRetry uint16 `conf:"max_retry" default:"10"`

	Dictionary map[string]string `conf:"dict"`
}

func TestLoad(t *testing.T) {
	t.Parallel()

	t.Run("should load config", func(t *testing.T) {
		t.Parallel()

		config := TestConfigNested{} //nolint:exhaustruct

		cl := configfx.NewConfigManager()
		err := cl.Load(&config)

		if assert.NoError(t, err) {
			assert.Equal(t, "localhost", config.Host)
			assert.Equal(t, 8080, config.Port)
			assert.Equal(t, uint16(10), config.MaxRetry)
		}
	})

	t.Run("should load config from string", func(t *testing.T) {
		t.Parallel()

		configStr := `{"host": "localhost", "port": 8080, "max_retry": 10, "dict": {"key": "value"}}`
		config := TestConfigNested{} //nolint:exhaustruct

		cl := configfx.NewConfigManager()
		err := cl.Load(&config, cl.FromJsonString(configStr))

		if assert.NoError(t, err) {
			assert.Equal(t, "localhost", config.Host)
			assert.Equal(t, 8080, config.Port)
			assert.Equal(t, uint16(10), config.MaxRetry)
			assert.Equal(t, map[string]string{"KEY": "value"}, config.Dictionary)
		}
	})
}

func TestLoadMeta(t *testing.T) { //nolint:funlen
	t.Parallel()

	t.Run("should get config meta", func(t *testing.T) {
		t.Parallel()

		config := TestConfig{} //nolint:exhaustruct

		cl := configfx.NewConfigManager()
		meta, err := cl.LoadMeta(&config)

		expected := []configfx.ConfigItemMeta{
			{
				Name:            "host",
				Field:           meta.Children[0].Field,
				Type:            reflect.TypeFor[string](),
				IsRequired:      false,
				HasDefaultValue: true,
				DefaultValue:    "localhost",

				Children: nil,
			},
		}

		if assert.NoError(t, err) {
			assert.Equal(t, "root", meta.Name)
			assert.Nil(t, meta.Type)

			assert.ElementsMatch(t, expected, meta.Children)
		}
	})

	t.Run("should get config meta from nested definition", func(t *testing.T) {
		t.Parallel()

		config := TestConfigNested{} //nolint:exhaustruct

		cl := configfx.NewConfigManager()
		meta, err := cl.LoadMeta(&config)

		expected := []configfx.ConfigItemMeta{
			{
				Name:            "host",
				Field:           meta.Children[0].Field,
				Type:            reflect.TypeFor[string](),
				IsRequired:      false,
				HasDefaultValue: true,
				DefaultValue:    "localhost",

				Children: nil,
			},
			{
				Name:            "port",
				Field:           meta.Children[1].Field,
				Type:            reflect.TypeFor[int](),
				IsRequired:      false,
				HasDefaultValue: true,
				DefaultValue:    "8080",

				Children: nil,
			},
			{
				Name:            "max_retry",
				Field:           meta.Children[2].Field,
				Type:            reflect.TypeFor[uint16](),
				IsRequired:      false,
				HasDefaultValue: true,
				DefaultValue:    "10",

				Children: nil,
			},
			{
				Name:            "dict",
				Field:           meta.Children[3].Field,
				Type:            reflect.TypeFor[map[string]string](),
				IsRequired:      false,
				HasDefaultValue: false,
				DefaultValue:    "",

				Children: nil,
			},
		}

		if assert.NoError(t, err) {
			assert.Equal(t, "root", meta.Name)
			assert.Nil(t, meta.Type)

			assert.ElementsMatch(t, expected, meta.Children)
		}
	})
}
