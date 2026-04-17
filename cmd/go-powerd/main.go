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

	"github.com/VicDeo/go-powerd/internal/app"
	"github.com/VicDeo/go-powerd/internal/config"
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

	a := app.New(version, cfg)
	if tray {
		slog.Info("Starting go-powerd", "version", version, "commit", commit, "verbose", verbose)
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		if err := a.Run(ctx); err != nil {
			slog.Error("Error starting the application", "error", err)
			slog.Info("Shutting down go-powerd", "version", version)
			os.Exit(1)
		}
		slog.Info("Shutting down go-powerd", "version", version)
	} else {
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
