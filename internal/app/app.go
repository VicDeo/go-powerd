// Package app provides the application logic for the go-powerd app.
package app

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/VicDeo/go-powerd/internal/battery"
	"github.com/VicDeo/go-powerd/internal/debounce"
	"github.com/VicDeo/go-powerd/internal/icon"
	"github.com/VicDeo/go-powerd/internal/netlink"
	"github.com/energye/systray"
)

const (
	// sysfs path to the battery information
	sysfsPath = "/sys/class/power_supply"
	// poll interval for the battery information
	pollInterval = 60 * time.Second
	// debounce window for the battery information
	debounceWindow = 500 * time.Millisecond
	// tray icon size in pixels
	iconSize = 32.0
)

// App is the main application struct.
type App struct {
	batteries *battery.Batteries
}

// New creates a new App instance.
func New() *App {
	return &App{
		batteries: battery.NewBatteries(sysfsPath),
	}
}

// Run runs the application.
func (a *App) Run() error {
	err := a.batteries.Load()
	if err != nil {
		return fmt.Errorf("error initializing batteries info: %w", err)
	}

	systray.Run(a.onReady, a.onExit)
	return nil
}

// onReady is the callback function for the systray.
func (a *App) onReady() {
	ctx, cancel := context.WithCancel(context.Background())

	var lastCapacity int = -1
	var lastIsCharging bool = false
	var buf bytes.Buffer
	updateUI := func() {
		err := a.batteries.Load()
		if err != nil {
			slog.Error("Error reading batteries info", "error", err)
			return
		}
		systray.SetTitle(a.batteries.Tooltip())
		cap, charging := a.batteries.Capacity(), a.batteries.IsCharging()
		if lastCapacity != cap || lastIsCharging != charging {
			lastCapacity = cap
			lastIsCharging = charging
			systray.SetIcon(icon.DrawIcon(cap, charging, iconSize, &buf))
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
			slog.Error("Error establishing kernel socket connection", "error", err)
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

func (a *App) onExit() {
	slog.Info("Quitting application...")
}
