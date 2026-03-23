package battery

import (
	"bytes"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path"
	"strings"
)

const (
	// typeFilename is a name of the file that contains a device type.
	typeFilename = "type"
)

var (
	// batteryType is a content of the file of the type battery.
	batteryType []byte = []byte("Battery\n")
)

// Batteries is a collection of all available batteries on the system.
type Batteries struct {
	root      string    // Path to a batteries root directory.
	batteries []Battery // Slice of all discovered batteries.
}

// NewBatteries return a new empty Batteries collection.
func NewBatteries(root string) *Batteries {
	return &Batteries{root: root}
}

// Enum provides the list of available battery directory names.
func (b *Batteries) Enum() ([]string, error) {
	var batteryNames []string

	if _, err := os.Stat(b.root); err != nil {
		return nil, fmt.Errorf("could not initialize Power Supply subsystem at %s: %w", b.root, err)
	}

	sysfs := os.DirFS(b.root)
	entries, err := fs.ReadDir(sysfs, ".")
	if err != nil {
		return nil, fmt.Errorf("could not read entries in %s: %w", b.root, err)
	}

	for _, device := range entries {
		deviceTypePath := path.Join(device.Name(), typeFilename)
		deviceType, err := b.isBattery(sysfs, deviceTypePath)
		if err != nil {
			// Just skip the device we can't read
			slog.Warn("Error while reading the device type", "deviceTypePath", deviceTypePath, "error", err)
			continue
		}

		slog.Debug("Type file content", "deviceTypePath", deviceTypePath, "deviceType", deviceType)
		if bytes.Equal(deviceType, batteryType) {
			batteryNames = append(batteryNames, device.Name())
		}
	}

	return batteryNames, nil
}

// Load adds battery data for all batteries found in the system.
func (b *Batteries) Load() error {
	b.batteries = b.batteries[:0]
	batteryNames, err := b.Enum()
	if err != nil {
		return fmt.Errorf("error loading batteries: %w", err)
	}
	for _, n := range batteryNames {
		bat := New(path.Join(b.root, n))
		err := bat.Load()
		if err != nil {
			return fmt.Errorf("error loading battery %s: %w", n, err)
		}
		b.batteries = append(b.batteries, *bat)
	}
	return nil
}

// isBattery checks if the power device is battery by matching the file content.
func (b *Batteries) isBattery(base fs.FS, rel string) ([]byte, error) {
	deviceTypeFile, err := base.Open(rel)
	if err != nil {
		// Having no type is not that expected but ok to continue
		return nil, fmt.Errorf("found no type file for the device %s: %w", rel, err)
	}
	defer deviceTypeFile.Close()

	deviceType := make([]byte, len(batteryType))
	n, err := deviceTypeFile.Read(deviceType)
	if err != nil {
		// Skip unread type file
		return nil, fmt.Errorf("failed to read type file for the device at %s: %w", rel, err)
	}
	deviceType = deviceType[:n]
	return deviceType, nil
}

// Tooltip returns a full description for all installed batteries.
func (b *Batteries) Tooltip(version string) string {
	builder := strings.Builder{}
	builder.WriteString("go-powerd v" + version + "\n")
	for _, bat := range b.batteries {
		health, err := bat.Health()
		if err != nil {
			slog.Warn("Error while getting battery health", "error", err)
			fmt.Fprintf(&builder,
				"\n%s [%s]\nPower: %d%%\nHealth: Unknown\n", bat.Name, bat.ExtendedStatus(), bat.Capacity)
		} else {
			fmt.Fprintf(&builder,
				"\n%s [%s]\nPower: %d%%\nHealth: %d%%\n", bat.Name, bat.ExtendedStatus(), bat.Capacity, health)
		}
	}
	return builder.String()
}

// Capacity returns a common capacity for all batteries.
func (b *Batteries) Capacity() int {
	if len(b.batteries) == 0 {
		return 0
	}
	var totalEnergyFull, totalEnergyNow int64
	for _, bat := range b.batteries {
		totalEnergyFull += bat.EnergyFull
		totalEnergyNow += bat.EnergyNow
	}

	if totalEnergyFull == 0 {
		slog.Error("No energy full capacity available for aggregate", "totalEnergyFull", totalEnergyFull, "batteriesCount", len(b.batteries))
		return 0
	}

	capacity := (100 * totalEnergyNow / totalEnergyFull)
	return int(capacity)
}

// IsCharging returns true if any battery is charging.
func (b *Batteries) IsCharging() bool {
	for _, bat := range b.batteries {
		if bat.Status == statusCharging {
			return true
		}
	}
	return false
}

// Status returns the status of the batteries.
func (b *Batteries) Status() string {
	for _, bat := range b.batteries {
		if bat.Status == statusCharging {
			return "Charging"
		}
	}
	return "Discharging"
}
