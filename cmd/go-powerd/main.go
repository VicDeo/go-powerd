// Main package for the go-powerd app.
package main

import (
	"flag"
	"log/slog"
	"os"

	"github.com/VicDeo/go-powerd/internal/app"
)

const (
	version = "0.1.0"
)

func main() {
	var verbose bool
	flag.BoolVar(&verbose, "v", false, "enable verbose/debug logging")
	flag.Parse()

	setupLogger(verbose)
	slog.Info("Starting go-powerd", "version", version, "verbose", verbose)

	if err := app.New().Run(); err != nil {
		slog.Error("Error starting the application", "error", err)
		slog.Info("Shutting down go-powerd", "version", version)
		os.Exit(1)
	}
	slog.Info("Shutting down go-powerd", "version", version)
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
