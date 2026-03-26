// Main package for the go-powerd app.
package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/VicDeo/go-powerd/internal/app"
	"github.com/VicDeo/go-powerd/internal/config"
)

const (
	version = "0.1.0"
)

func main() {
	var configPath string
	var verbose, tray, showHelp bool
	flag.StringVar(&configPath, "c", "", "path to config file")
	flag.BoolVar(&verbose, "v", false, "enable verbose/debug logging")
	flag.BoolVar(&tray, "t", false, "attach to tray")
	flag.BoolVar(&showHelp, "h", false, "show this help message and exit")
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
		slog.Info("Starting go-powerd", "version", version, "verbose", verbose)
		if err := a.Run(); err != nil {
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
	fmt.Println("Usage: go-powerd [options]")
	fmt.Println()
	fmt.Println("Without -t, prints battery status to stdout and exits.")
	fmt.Println("With -t, runs in the system tray.")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -c path    Path to config file (default: $XDG_CONFIG_HOME/go-powerd/config.toml or ~/.config/go-powerd/config.toml)")
	fmt.Println("  -v         Verbose/debug logging (with source locations)")
	fmt.Println("  -t         Attach to system tray instead of one-shot status")
	fmt.Println("  -h         Show this help and exit")
}
