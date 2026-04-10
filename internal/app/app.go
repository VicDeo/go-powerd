// Package app provides the application logic for the go-powerd app.
package app

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"
	"time"

	"github.com/VicDeo/go-powerd/internal/battery"
	"github.com/VicDeo/go-powerd/internal/config"
	"github.com/VicDeo/go-powerd/internal/dbus"
	"github.com/VicDeo/go-powerd/internal/debounce"
	"github.com/VicDeo/go-powerd/internal/icon"
	"github.com/VicDeo/go-powerd/internal/netlink"
	"github.com/VicDeo/go-powerd/internal/policy"
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
	config              *config.Config
	batteries           *battery.Batteries
	dischargingPolicies []*policy.Policy
	version             string
}

// New creates a new App instance.
func New(version string, cfg *config.Config) *App {
	return &App{
		batteries: battery.NewBatteries(sysfsPath),
		config:    cfg,
		version:   version,
	}
}

// Run runs the application.
func (a *App) Run() error {
	err := a.batteries.Load()
	if err != nil {
		return fmt.Errorf("error initializing batteries info: %w", err)
	}

	if a.batteries.Len() == 0 {
		return fmt.Errorf("no batteries found")
	}

	if a.batteries.Capacity() == 0 {
		return fmt.Errorf("no batteries with capacity found")
	}

	a.parseConfig()

	systray.Run(a.onReady, a.onExit)
	return nil
}

func (a *App) Status() (string, error) {
	err := a.batteries.Load()
	if err != nil {
		return "", fmt.Errorf("error initializing batteries info: %w", err)
	}
	return a.batteries.Tooltip(a.version), nil
}

// onReady is the callback function for the systray.
func (a *App) onReady() {
	ctx, cancel := context.WithCancel(context.Background())

	discharging := &policy.Manager{
		Name:     "On Battery",
		Policies: a.dischargingPolicies,
	}

	coordinator := &policy.Coordinator{
		ChargingMngr:    &policy.Manager{Name: "Charging"},
		DischargingMngr: discharging,
		ActiveMngr:      nil,
		LastStatus:      "",
	}

	var lastCapacity int = -1
	var lastIsCharging bool = false
	trayIcon := icon.New(iconSize)
	trayIcon.SetColors(&a.config.Theme.Colors)
	updateUI := func() {
		err := a.batteries.Load()
		if err != nil {
			slog.Error("Error reading batteries info", "error", err)
			return
		}
		systray.SetTitle(a.batteries.Tooltip(a.version))
		cap, charging := a.batteries.Capacity(), a.batteries.IsCharging()
		if lastCapacity != cap || lastIsCharging != charging {
			coordinator.HandleUpdate(cap, a.batteries.Status())
			lastCapacity = cap
			lastIsCharging = charging
			systray.SetIcon(trayIcon.PNG(cap, charging))
			debug.FreeOSMemory()
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
	// Nothing to do here
}

func (a *App) parseConfig() {

	if a.config.Policies.Notify.Active != nil && *a.config.Policies.Notify.Active == true {
		lowPolicy := policy.Policy{
			Name:       "Low",
			Threshold:  a.config.Policies.Notify.Threshold,
			Hysteresis: a.config.Policies.Notify.Hysteresis,
			OnTrigger: func() {
				err := dbus.SendNotification("Low Battery!", "Connect a power source as soon as possible.", "battery-caution", true)
				if err != nil {
					slog.Error("Error sending notification", "error", err)
				}
			},
		}
		a.dischargingPolicies = append(a.dischargingPolicies, &lowPolicy)
	}

	if a.config.Policies.Suspend.Active != nil && *a.config.Policies.Suspend.Active == true {
		criticalPolicy := policy.Policy{
			Name:       "Critical",
			Threshold:  a.config.Policies.Suspend.Threshold,
			Hysteresis: a.config.Policies.Suspend.Hysteresis,
			OnTrigger: func() {
				err := dbus.SuspendSystem()
				if err != nil {
					slog.Error("Error suspending system", "error", err)
				}
			},
		}
		a.dischargingPolicies = append(a.dischargingPolicies, &criticalPolicy)
	}
}
