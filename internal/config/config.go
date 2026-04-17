// Package config provides a way to load the config for the app.
package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

var (
	defaultColors = map[string]string{
		"segments_ok":       "#ffffffff", // Normal digit color
		"segments_low":      "#ffffffff", // Low power digit color
		"segments_charging": "#3399ffff", // On AC digit color

		"bar_ok":       "#33cc33ff", // Normal battery level color
		"bar_low":      "#e53333ff", // Low battery level color
		"bar_charging": "#f2d11fff", // On AC battery level color

		"border":  "#ccccccff", // Battery border color
		"charger": "#f2d11fff", // Charger indication color
	}
)

type Colors struct {
	SegmentsOk       string `toml:"segments_ok"`
	SegmentsLow      string `toml:"segments_low"`
	SegmentsCharging string `toml:"segments_charging"`
	BarOk            string `toml:"bar_ok"`
	BarLow           string `toml:"bar_low"`
	BarCharging      string `toml:"bar_charging"`
	Border           string `toml:"border"`
	Charger          string `toml:"charger"`
}

type Theme struct {
	Colors Colors
}

type Policy struct {
	Active     bool
	Threshold  int
	Hysteresis int
}

type Policies struct {
	Notify  Policy
	Suspend Policy
}

type Config struct {
	ConfigVersion int
	Policies      Policies
	Theme         Theme
}

// Load loads the config from the given path.
func Load(path string) (*Config, error) {
	var config Config

	if _, err := os.Stat(path); os.IsNotExist(err) {
		slog.Info("Config file not found, using internal defaults", "path", path)
		return DefaultConfig(), nil
	}

	_, err := toml.DecodeFile(path, &config)
	if err != nil {
		return nil, fmt.Errorf("error while decoding config file %s: %w", path, err)
	}

	if err := validate(&config); err != nil {
		return nil, fmt.Errorf("error while validating config: %w", err)
	}
	validateColors(&config)

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

func validateColors(config *Config) {
	colorChecks := []struct {
		field string
		value *string
	}{
		{field: "segments_ok", value: &config.Theme.Colors.SegmentsOk},
		{field: "segments_low", value: &config.Theme.Colors.SegmentsLow},
		{field: "segments_charging", value: &config.Theme.Colors.SegmentsCharging},
		{field: "bar_ok", value: &config.Theme.Colors.BarOk},
		{field: "bar_low", value: &config.Theme.Colors.BarLow},
		{field: "bar_charging", value: &config.Theme.Colors.BarCharging},
		{field: "border", value: &config.Theme.Colors.Border},
		{field: "charger", value: &config.Theme.Colors.Charger},
	}
	for _, check := range colorChecks {
		if err := validRGBA(*check.value); err != nil {
			slog.Warn("color validation failed. falling back to default", "option", check.field, "value", *check.value)
			*check.value = defaultColors[check.field]
		}
	}
}

// DefaultPath returns the default path to the config file.
func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("error while getting user config directory: %w", err)
	}
	return filepath.Join(dir, "go-powerd", "config.toml"), nil
}

// DefaultConfig returns the default config.
func DefaultConfig() *Config {
	return &Config{
		ConfigVersion: 1,
		Policies:      DefaultPolicies(),
		Theme: Theme{
			Colors: DefaultColors(),
		},
	}
}

// DefaultPolicies returns the default policies.
func DefaultPolicies() Policies {
	return Policies{
		Notify: Policy{
			Active:     true,
			Threshold:  20,
			Hysteresis: 3,
		},
		Suspend: Policy{
			Active:     true,
			Threshold:  10,
			Hysteresis: 5,
		},
	}
}

func DefaultColors() Colors {
	return Colors{
		SegmentsOk:       defaultColors["segments_ok"],
		SegmentsLow:      defaultColors["segments_low"],
		SegmentsCharging: defaultColors["segments_charging"],
		BarOk:            defaultColors["bar_ok"],
		BarLow:           defaultColors["bar_low"],
		BarCharging:      defaultColors["bar_charging"],
		Border:           defaultColors["border"],
		Charger:          defaultColors["charger"],
	}
}

func between0And100(field string, value int) error {
	if value < 0 || value > 100 {
		return fmt.Errorf("%s must be between 0 and 100", field)
	}
	return nil
}

func validRGBA(rgba string) error {
	if len(rgba) != 9 {
		return fmt.Errorf("expected 9 characters. Not a valid RGBA: %s", rgba)
	}
	if rgba[0] != '#' {
		return fmt.Errorf("rgba should start with #. Not a valid RGBA: %s", rgba)
	}

	// lowercase and check that we have 8 digits 0-f
	for _, c := range strings.ToLower(rgba[1:]) {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return fmt.Errorf("invalid character found. Not a valid RGBA: %s", rgba)
		}
	}
	return nil
}
