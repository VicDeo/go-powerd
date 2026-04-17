package app

import (
	"log/slog"

	"github.com/VicDeo/go-powerd/internal/dbus"
)

func sendNotification() {
	err := dbus.SendNotification("Low Battery!", "Connect a power source as soon as possible.", "battery-caution", true)
	if err != nil {
		slog.Error("Error sending notification", "error", err)
	}
}

func sendSuspendSystem() {
	err := dbus.SuspendSystem()
	if err != nil {
		slog.Error("Error suspending system", "error", err)
	}
}
