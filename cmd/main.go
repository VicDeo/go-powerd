package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/VicDeo/go-powerd/internal/battery"
	"github.com/VicDeo/go-powerd/internal/debounce"
	"github.com/VicDeo/go-powerd/internal/icon"
	"github.com/VicDeo/go-powerd/internal/netlink"
	"github.com/energye/systray"
)

const (
	sysfsPath      = "/sys/class/power_supply"
	pollInterval   = 60 * time.Second
	debounceWindow = 500 * time.Millisecond
)

func main() {
	var verbose bool
	flag.BoolVar(&verbose, "v", false, "enable verbose/debug logging")
	flag.Parse()

	setupLogger(verbose)
	systray.Run(onReady, onExit)
}

func onReady() {
	batteries := battery.NewBatteries(sysfsPath)
	err := batteries.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing batteries info: %v\n", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())

	var lastCapacity int = -1
	var lastIsCharging bool = false
	updateUI := func() {
		err := batteries.Load()
		if err != nil {
			slog.Error("Error reading batteries info", "error", err)
			return
		}
		systray.SetTitle(batteries.Tooltip())
		var buf bytes.Buffer
		cap, charging := batteries.Capacity(), batteries.IsCharging()
		if lastCapacity != cap || lastIsCharging != charging {
			lastCapacity = cap
			lastIsCharging = charging
			systray.SetIcon(icon.DrawIcon(cap, charging, 32.0, &buf))
		}
	}

	updateUI()

	deb := debounce.New(debounceWindow, updateUI)
	defer deb.Stop()

	onPowerEvent := func([]byte) {
		deb.Trigger()
	}
	go func() {
		if err := netlink.Listen(ctx, onPowerEvent); err != nil {
			cancel()
			fmt.Fprintf(os.Stderr, "Error establishing kernel socket connection: %v\n", err)
			systray.Quit()
		}
	}()

	go func() {
		ticker := time.NewTicker(pollInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				deb.Trigger()
			}
		}
	}()

	mQuit := systray.AddMenuItem("Quit", "Quit the application")
	mQuit.Enable()
	mQuit.Click(func() {
		cancel()
		systray.Quit()
	})
}

func onExit() {
	fmt.Println("Cleaning up...")
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
