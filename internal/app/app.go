// Package app implements the application logic for the go-powerd app.
package app

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"
	"sync"
	"time"

	"github.com/VicDeo/go-powerd/internal/battery"
	"github.com/VicDeo/go-powerd/internal/config"
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

type uiState struct {
	capacity    int
	isPluggedIn bool
}

// App is the main application struct.
type App struct {
	config              *config.Config
	batteries           *battery.Batteries
	dischargingPolicies []*policy.Policy
	version             string
	uiState             uiState
	uiStateMu           sync.Mutex
	icon                *icon.Icon
	coordinator         *policy.Coordinator
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
func (a *App) Run(ctx context.Context) error {
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
	a.uiState = uiState{capacity: -1, isPluggedIn: false}
	a.coordinator = a.initCoordinator()
	a.icon = icon.New(iconSize)
	a.icon.SetColors(&a.config.Theme.Colors)

	runCtx, cancel := context.WithCancel(ctx)
	systray.Run(
		func() { a.onReady(runCtx, cancel) },
		a.onExit,
	)
	return nil
}

func (a *App) Status() (string, error) {
	err := a.batteries.Load()
	if err != nil {
		return "", fmt.Errorf("error initializing batteries info: %w", err)
	}
	return a.batteries.Tooltip(a.version), nil
}

func (a *App) onExit() {
	// Nothing to do here
}

// onReady is the callback function for the systray.
func (a *App) onReady(ctx context.Context, cancel context.CancelFunc) {
	a.updateUI()

	deb := debounce.New(debounceWindow, a.updateUI)
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

	a.setupMenu(cancel)
}

func (a *App) initCoordinator() *policy.Coordinator {
	discharging := &policy.Manager{
		Name:     "On Battery",
		Policies: a.dischargingPolicies,
	}

	return &policy.Coordinator{
		ChargingMngr:    &policy.Manager{Name: "Charging"},
		DischargingMngr: discharging,
		ActiveMngr:      nil,
		LastStatus:      true,
	}
}

func (a *App) updateUI() {
	err := a.batteries.Load()
	if err != nil {
		slog.Error("Error reading batteries info", "error", err)
		return
	}
	systray.SetTitle(a.batteries.Tooltip(a.version))
	newState := uiState{capacity: a.batteries.Capacity(), isPluggedIn: a.batteries.IsPluggedIn()}
	a.uiStateMu.Lock()
	defer a.uiStateMu.Unlock()
	if a.uiState.capacity != newState.capacity || a.uiState.isPluggedIn != newState.isPluggedIn {
		a.coordinator.HandleUpdate(newState.capacity, newState.isPluggedIn)
		appIcon, fromCache := a.icon.Get(newState.capacity, newState.isPluggedIn)
		a.uiState.capacity = newState.capacity
		a.uiState.isPluggedIn = newState.isPluggedIn
		systray.SetIcon(appIcon)
		if !fromCache {
			debug.FreeOSMemory()
		}
	}
}

func (a *App) setupMenu(cancel context.CancelFunc) {
	mQuit := systray.AddMenuItem("Quit", "Quit the application")
	mQuit.Enable()
	mQuit.Click(func() {
		cancel()
		systray.Quit()
	})
}

func (a *App) parseConfig() {
	if a.config.Policies.Notify.Active {
		lowPolicy := policy.Policy{
			Name:       "Low",
			Threshold:  a.config.Policies.Notify.Threshold,
			Hysteresis: a.config.Policies.Notify.Hysteresis,
			OnTrigger:  sendNotification,
		}
		a.dischargingPolicies = append(a.dischargingPolicies, &lowPolicy)
	}

	if a.config.Policies.Suspend.Active {
		criticalPolicy := policy.Policy{
			Name:       "Critical",
			Threshold:  a.config.Policies.Suspend.Threshold,
			Hysteresis: a.config.Policies.Suspend.Hysteresis,
			OnTrigger:  sendSuspendSystem,
		}
		a.dischargingPolicies = append(a.dischargingPolicies, &criticalPolicy)
	}
}
