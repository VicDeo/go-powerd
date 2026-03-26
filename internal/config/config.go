// Package config provides a way to load the config for the app.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Policy struct {
	Active     *bool
	Threshold  int
	Hysteresis int
}

type Config struct {
	ConfigVersion int
	Policies      struct {
		Notify  Policy
		Suspend Policy
	}
}

// Load loads the config from the given path.
func Load(path string) (*Config, error) {
	var config Config
	_, err := toml.DecodeFile(path, &config)
	if err != nil {
		return nil, fmt.Errorf("error while decoding config file %s: %w", path, err)
	}

	if err := validate(&config); err != nil {
		return nil, fmt.Errorf("error while validating config: %w", err)
	}

	return &config, nil
}

func validate(config *Config) error {
	if config.ConfigVersion != 1 {
		return fmt.Errorf("invalid config version: %d", config.ConfigVersion)
	}
	if config.Policies.Notify.Threshold <= config.Policies.Suspend.Threshold {
		return fmt.Errorf("notify threshold must be greater than suspend threshold")
	}

	checks := []struct {
		field string
		value int
	}{
		{field: "notify.threshold", value: config.Policies.Notify.Threshold},
		{field: "suspend.threshold", value: config.Policies.Suspend.Threshold},
		{field: "notify.hysteresis", value: config.Policies.Notify.Hysteresis},
		{field: "suspend.hysteresis", value: config.Policies.Suspend.Hysteresis},
	}
	errs := []error{}
	for _, check := range checks {
		if err := between0And100(check.field, check.value); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// DefaultConfigPath returns the default config path.
func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("error while getting user config directory: %w", err)
	}
	return filepath.Join(dir, "go-powerd", "config.toml"), nil
}

func between0And100(field string, value int) error {
	if value < 0 || value > 100 {
		return fmt.Errorf("%s must be between 0 and 100", field)
	}
	return nil
}
