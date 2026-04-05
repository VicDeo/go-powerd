package dbus

import (
	"fmt"

	"github.com/godbus/dbus/v5"
)

func SendNotification(title, message, icon string, critical bool) error {
	conn, err := dbus.SessionBus() // Notifications live on the Session Bus
	if err != nil {
		return fmt.Errorf("failed to connect to session bus: %w", err)
	}
	defer conn.Close()

	obj := conn.Object("org.freedesktop.Notifications", "/org/freedesktop/Notifications")

	// Urgency hint: 0 = low, 1 = normal, 2 = critical
	urgency := byte(1)
	if critical {
		urgency = 2
	}

	hints := map[string]dbus.Variant{
		"urgency": dbus.MakeVariant(urgency),
	}

	call := obj.Call("org.freedesktop.Notifications.Notify", 0,
		"go-powerd", // app_name
		uint32(0),   // replaces_id
		icon,        // app_icon (e.g., "battery-caution")
		title,       // summary
		message,     // body
		[]string{},  // actions
		hints,       // hints
		int32(5000), // expire_timeout (5 seconds)
	)

	return call.Err
}

func SuspendSystem() error {
	// Connect to the System Bus (not Session Bus!)
	conn, err := dbus.SystemBus()
	if err != nil {
		return fmt.Errorf("failed to connect to system bus: %w", err)
	}
	defer conn.Close()

	obj := conn.Object("org.freedesktop.login1", "/org/freedesktop/login1")

	// The 'false' argument means "non-interactive"
	call := obj.Call("org.freedesktop.login1.Manager.Suspend", 0, false)
	if call.Err != nil {
		return fmt.Errorf("suspend call failed: %w", call.Err)
	}

	return nil
}
