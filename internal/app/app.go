// Package app implements the application logic for the go-powerd app.
package app

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"
	"sync"
	"time"

	"github.com/energye/systray"
)

const (
	// poll interval for the battery information
	pollInterval = 60 * time.Second
	// log battery metrics interval
	logInterval = 10 * 60 * time.Second
)

type uiState struct {
	capacity    int
	isPluggedIn bool
}

// App is the main application struct.
type App struct {
	batteries     statusProvider
	version       string
	uiState       uiState
	uiStateMu     sync.Mutex
	icon          iconGetter
	coordinator   actionTrigger
	coordinatorMu sync.Mutex
	deb           debouncer
	debMu         sync.Mutex
	watcher       eventWatcher
	lastLogTime   time.Time
}

// New creates a new App instance.
func New(version string, bats statusProvider, icon iconGetter, coordinator actionTrigger, deb debouncer, w eventWatcher) *App {
	return &App{
		batteries:   bats,
		version:     version,
		icon:        icon,
		coordinator: coordinator,
		deb:         deb,
		watcher:     w,
		lastLogTime: time.Now().Add(-logInterval),
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

	a.uiState = uiState{capacity: -1, isPluggedIn: false}

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

	a.debMu.Lock()
	a.deb.Start(a.updateUI)
	a.debMu.Unlock()

	go func() {
		if err := a.watcher.Watch(ctx, a.deb.Trigger); err != nil {
			cancel()
			if err != context.Canceled {
				slog.Error("Failed to start kernel event watcher", "error", err)
			}
			systray.Quit()
		}
	}()

	go a.startPoller(ctx)

	a.setupMenu(cancel)
}

func (a *App) Reload(coordinator actionTrigger) {
	a.coordinatorMu.Lock()
	a.coordinator = coordinator
	a.coordinatorMu.Unlock()

	a.uiStateMu.Lock()
	// Force redraw icon
	a.uiState = uiState{capacity: -1, isPluggedIn: false}
	a.uiStateMu.Unlock()
	a.debMu.Lock()
	deb := a.deb
	a.debMu.Unlock()
	if deb != nil {
		a.deb.Trigger()
	}
}

func (a *App) updateUI() {
	err := a.batteries.Load()
	if err != nil {
		slog.Error("Error reading batteries info", "error", err)
		return
	}

	if time.Since(a.lastLogTime) > logInterval {
		a.lastLogTime = time.Now()
		a.batteries.Log()
	}

	systray.SetTitle(a.batteries.Tooltip(a.version))
	newState := uiState{capacity: a.batteries.Capacity(), isPluggedIn: a.batteries.IsPluggedIn()}
	a.uiStateMu.Lock()
	defer a.uiStateMu.Unlock()
	if a.uiState.capacity != newState.capacity || a.uiState.isPluggedIn != newState.isPluggedIn {
		a.coordinatorMu.Lock()
		c := a.coordinator
		a.coordinatorMu.Unlock()
		if c != nil {
			c.HandleUpdate(newState.capacity, newState.isPluggedIn)
		}

		appIcon, fromCache := a.icon.Get(newState.capacity, newState.isPluggedIn)
		a.uiState.capacity = newState.capacity
		a.uiState.isPluggedIn = newState.isPluggedIn
		systray.SetIcon(appIcon)
		if !fromCache {
			debug.FreeOSMemory()
		}
	}
}

func (a *App) startPoller(ctx context.Context) {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.deb.Trigger()
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
