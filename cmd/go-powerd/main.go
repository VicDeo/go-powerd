// Package main implements the go-powerd app.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/VicDeo/go-powerd/internal/app"
	"github.com/VicDeo/go-powerd/internal/config"
	"github.com/VicDeo/go-powerd/internal/debounce"
	"github.com/VicDeo/go-powerd/internal/icon"
	"github.com/VicDeo/go-powerd/internal/policy"
)

const (
	// debounce window for the battery information
	debounceWindow = 500 * time.Millisecond

	// tray icon size in pixels
	iconSize = 32.0
)

var (
	version = "none"
	commit  = "none"
)

func main() {
	var configPath string
	var verbose, tray, showHelp bool
	flag.StringVar(&configPath, "c", "", "path to config file")
	flag.BoolVar(&verbose, "v", false, "enable verbose/debug logging")
	flag.BoolVar(&tray, "t", false, "attach to tray")
	flag.BoolVar(&showHelp, "h", false, "show this help message and exit")
	flag.BoolVar(&showHelp, "help", false, "show this help message and exit")
	flag.Usage = help
	flag.Parse()

	if showHelp {
		help()
		os.Exit(0)
	}

	setupLogger(verbose)

	if configPath == "" {
		var err error
		configPath, err = config.DefaultPath()
		if err != nil {
			slog.Error("Error while getting default config path", "error", err)
			os.Exit(1)
		}
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		slog.Error("Error loading config", "error", err)
		os.Exit(1)
	}

	if tray {
		icn := icon.New(iconSize)
		applyTheme(icn, cfg)
		dischargingPolicies := parsePolicies(cfg)
		coordinator := initCoordinator(dischargingPolicies, nil)

		deb := debounce.New(debounceWindow)
		defer deb.Stop()

		a := app.New(version, icn, coordinator, deb)

		slog.Info("Starting go-powerd", "version", version, "commit", commit, "verbose", verbose)
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGHUP)
		defer signal.Stop(sigChan)

		go reloadListener(ctx, a, coordinator, icn, sigChan, configPath)

		if err := a.Run(ctx); err != nil {
			slog.Error("Error starting the application", "error", err)
			slog.Info("Shutting down go-powerd", "version", version)
			os.Exit(1)
		}
		slog.Info("Shutting down go-powerd", "version", version)
	} else {
		a := app.New(version, nil, nil, nil)
		status, err := a.Status()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error while getting battery status: %v", err)
			os.Exit(1)
		}
		fmt.Println(status)
	}
}

func setupLogger(verbose bool) {
	logLevel := new(slog.LevelVar)
	if verbose {
		logLevel.Set(slog.LevelDebug)
	}
	// Set up the logger
	h := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel, AddSource: verbose})
	logger := slog.New(h)
	slog.SetDefault(logger)
}

func applyTheme(icn *icon.Icon, cfg *config.Config) {
	icn.SetColors(&cfg.Theme.Colors)
	icn.Reset()
}

func parsePolicies(cfg *config.Config) []*policy.Policy {
	dischargingPolicies := make([]*policy.Policy, 0)
	if cfg.Policies.Notify.Active {
		lowPolicy := policy.Policy{
			Name:       "Low",
			Threshold:  cfg.Policies.Notify.Threshold,
			Hysteresis: cfg.Policies.Notify.Hysteresis,
			OnTrigger:  sendNotification,
		}
		dischargingPolicies = append(dischargingPolicies, &lowPolicy)
	}

	if cfg.Policies.Suspend.Active {
		criticalPolicy := policy.Policy{
			Name:       "Critical",
			Threshold:  cfg.Policies.Suspend.Threshold,
			Hysteresis: cfg.Policies.Suspend.Hysteresis,
			OnTrigger:  sendSuspendSystem,
		}
		dischargingPolicies = append(dischargingPolicies, &criticalPolicy)
	}
	return dischargingPolicies
}

func initCoordinator(dischargingPolicies, chargingPolicies []*policy.Policy) *policy.Coordinator {
	discharging := &policy.Manager{
		Name:     "On Battery",
		Policies: dischargingPolicies,
	}

	charging := &policy.Manager{
		Name:     "Charging",
		Policies: chargingPolicies,
	}

	return &policy.Coordinator{
		ChargingMngr:    charging,
		DischargingMngr: discharging,
		ActiveMngr:      nil,
		LastStatus:      true,
	}
}

func reloadListener(ctx context.Context, a *app.App, coordinator *policy.Coordinator, icn *icon.Icon, sigChan <-chan os.Signal, configPath string) {
	for {
		select {
		case <-ctx.Done():
			return
		case sig := <-sigChan:
			if sig == syscall.SIGHUP {
				cfg, err := config.Load(configPath)
				if err != nil {
					slog.Error("Error reloading config", "error", err)
					continue
				}
				applyTheme(icn, cfg)

				dischargingPolicies := parsePolicies(cfg)
				next := initCoordinator(dischargingPolicies, nil)
				coordinator.CopyStateTo(next)
				a.Reload(next)
				coordinator = next
			}
		}
	}
}

func help() {
	fmt.Fprintf(os.Stderr, "go-powerd %s (commit: %s)\n", version, commit)
	fmt.Fprintf(os.Stderr, "Copyright (c) 2026 Viktar Dubiniuk\n")
	fmt.Fprintf(os.Stderr, "License: GPL-3.0-only\n\n")
	fmt.Fprintf(os.Stderr, "A high-performance, minimalist battery monitor for Linux.\n\n")
	fmt.Fprintf(os.Stderr, "Usage: go-powerd [options]\n\n")
	fmt.Fprintf(os.Stderr, "Without -t, prints battery status to stdout and exits.\n")
	fmt.Fprintf(os.Stderr, "With -t, runs in the system tray.\n\n")
	fmt.Fprintf(os.Stderr, "Options:\n")
	fmt.Fprintf(os.Stderr, "  -c path    Path to config file (default: $XDG_CONFIG_HOME/go-powerd/config.toml or ~/.config/go-powerd/config.toml)\n")
	fmt.Fprintf(os.Stderr, "  -v         Verbose/debug logging (with source locations)\n")
	fmt.Fprintf(os.Stderr, "  -t         Attach to system tray instead of one-shot status\n")
	fmt.Fprintf(os.Stderr, "  -h         Show this help and exit\n")
}
