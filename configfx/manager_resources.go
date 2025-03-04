package configfx

import (
	"fmt"

	"github.com/eser/ajan/configfx/envparser"
	"github.com/eser/ajan/configfx/jsonparser"
	"github.com/eser/ajan/lib"
)

func (cl *ConfigManager) FromEnvFileDirect(filename string) ConfigResource {
	return func(target *map[string]any) error {
		err := envparser.TryParseFiles(target, filename)
		if err != nil {
			return fmt.Errorf("failed to parse env file - %q: %w", filename, err)
		}

		return nil
	}
}

func (cl *ConfigManager) FromEnvFile(filename string) ConfigResource {
	return func(target *map[string]any) error {
		env := lib.EnvGetCurrent()
		filenames := lib.EnvAwareFilenames(env, filename)

		err := envparser.TryParseFiles(target, filenames...)
		if err != nil {
			return fmt.Errorf("failed to parse env file - %q: %w", filename, err)
		}

		return nil
	}
}

func (cl *ConfigManager) FromSystemEnv() ConfigResource {
	return func(target *map[string]any) error {
		lib.EnvOverrideVariables(target)

		return nil
	}
}

func (cl *ConfigManager) FromJsonFileDirect(filename string) ConfigResource {
	return func(target *map[string]any) error {
		err := jsonparser.TryParseFiles(target, filename)
		if err != nil {
			return fmt.Errorf("failed to parse json file - %q: %w", filename, err)
		}

		return nil
	}
}

func (cl *ConfigManager) FromJsonFile(filename string) ConfigResource {
	return func(target *map[string]any) error {
		env := lib.EnvGetCurrent()
		filenames := lib.EnvAwareFilenames(env, filename)

		err := jsonparser.TryParseFiles(target, filenames...)
		if err != nil {
			return fmt.Errorf("failed to parse json file - %q: %w", filename, err)
		}

		return nil
	}
}
