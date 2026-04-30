package battery

import (
	"bytes"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path"
	"slices"
	"strings"

	"github.com/VicDeo/go-powerd/internal/pool"
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
	root      string              // Path to a batteries root directory.
	batteries []*Battery          // All discovered batteries as a slice
	lookup    map[string]*Battery // All discovered batteries as a map BATn -> *data
	builder   strings.Builder     // Builder for the tooltip
}

// NewBatteries creates a new empty Batteries collection.
func NewBatteries(root string) *Batteries {
	return &Batteries{root: root, builder: strings.Builder{}}
}

// Enum provides the list of available battery directory names.
func (b *Batteries) Enum() ([]string, error) {
	if _, err := os.Stat(b.root); err != nil {
		return nil, fmt.Errorf("could not initialize Power Supply subsystem at %s: %w", b.root, err)
	}

	sysfs := os.DirFS(b.root)
	entries, err := fs.ReadDir(sysfs, ".")
	if err != nil {
		return nil, fmt.Errorf("could not read entries in %s: %w", b.root, err)
	}

	foundNames := make([]string, 0, len(entries))
	dt := pool.Get()
	defer pool.Put(dt)

	for _, device := range entries {
		dt.Reset()
		deviceTypePath := path.Join(device.Name(), typeFilename)
		err := b.readType(sysfs, deviceTypePath, dt)
		if err != nil {
			// Just skip the device we can't read
			slog.Warn("Error while reading the device type", "deviceTypePath", deviceTypePath, "error", err)
			continue
		}

		if bytes.Equal(dt.Data(), batteryType) {
			foundNames = append(foundNames, device.Name())
		} else {
			slog.Debug("Device is not a battery", "deviceTypePath", deviceTypePath, "deviceType", dt.Data())
		}
	}

	slices.Sort(foundNames)
	slog.Debug("Devices found", "devices", foundNames)

	return foundNames, nil
}

// Load adds battery data for all batteries found in the system.
func (b *Batteries) Load() error {
	names, err := b.Enum()
	if err != nil {
		return err
	}

	if b.lookup == nil {
		b.lookup = make(map[string]*Battery, len(names))
	}

	b.batteries = b.batteries[:0]

	for _, name := range names {
		bat, ok := b.lookup[name]
		if !ok {
			bat = New(path.Join(b.root, name))
			b.lookup[name] = bat
		}

		if err := bat.Load(); err != nil {
			slog.Warn("Error loading battery", "name", name, "error", err)
			continue
		}

		b.batteries = append(b.batteries, bat)
	}

	// Cleanup map entries
	if len(b.lookup) > len(b.batteries) {
		for name := range b.lookup {
			if !slices.Contains(names, name) {
				delete(b.lookup, name)
			}
		}
	}

	return nil
}

// readType reads the type of the power device from the file.
func (b *Batteries) readType(base fs.FS, rel string, buf *pool.Buffer) error {
	deviceTypeFile, err := base.Open(rel)
	if err != nil {
		// Having no type is not that expected but ok to continue
		return fmt.Errorf("found no type file for the device %s: %w", rel, err)
	}
	defer deviceTypeFile.Close()

	n, err := deviceTypeFile.Read(buf.Bytes())
	if err != nil {
		// Skip unread type file
		return fmt.Errorf("failed to read type file for the device at %s: %w", rel, err)
	}
	buf.SetLen(n)
	return nil
}

// Tooltip returns a full description for all installed batteries.
func (b *Batteries) Tooltip(version string) string {
	b.builder.Reset()
	b.builder.WriteString("go-powerd v" + version + "\n")
	for _, bat := range b.batteries {
		health, err := bat.Health()
		if err != nil {
			slog.Warn("Error while getting battery health", "error", err)
			fmt.Fprintf(&b.builder,
				"\n%s [%s]\nPower: %d%%\nHealth: Unknown\n",
				bat.Name, bat.ExtendedStatus(), bat.Capacity)
		} else {
			fmt.Fprintf(&b.builder,
				"\n%s [%s]\nPower: %d%%\nHealth: %s (%d%%)\n",
				bat.Name, bat.ExtendedStatus(), bat.Capacity, b.healthToString(health), health)
		}
	}
	return b.builder.String()
}

// Capacity returns an aggregated capacity for all batteries.
func (b *Batteries) Capacity() int {
	if len(b.batteries) == 0 {
		return 0
	}
	var totalEnergyFull, totalEnergyNow WattHour
	for _, bat := range b.batteries {
		totalEnergyFull += bat.EnergyFull
		totalEnergyNow += bat.EnergyNow
	}

	if totalEnergyFull == 0 {
		slog.Error("No energy full capacity available for aggregate", "totalEnergyFull", totalEnergyFull, "batteriesCount", len(b.lookup))
		return 0
	}

	capacity := (100 * totalEnergyNow / totalEnergyFull)
	return int(capacity)
}

// IsPluggedIn returns true if the system is connected to AC power,
// i.e. no battery is actively discharging.
func (b *Batteries) IsPluggedIn() bool {
	if len(b.batteries) == 0 {
		return false
	}
	for _, bat := range b.batteries {
		// If any battery is discharging, the system is not plugged in
		if bat.Status == statusDischarging {
			return false
		}
	}
	return true
}

func (b *Batteries) Log() {
	for _, bat := range b.batteries {
		slog.Info("battery metrics",
			"name", bat.Name,
			"manufacturer", bat.Manufacturer,
			"model_name", bat.ModelName,
			"serial_number", bat.SerialNumber,
			"technology", bat.Technology,
			"status", bat.Status,
			"capacity", bat.Capacity,
			"capacity_level", bat.CapacityLevel,
			"present", bat.Present,
			"voltage_v", bat.VoltageNow,
			"voltage_min_design_v", bat.VoltageMinDesign,
			"power_now_w", bat.PowerNow,
			"energy_now_wh", bat.EnergyNow,
			"energy_full_wh", bat.EnergyFull,
			"energy_full_design_wh", bat.EnergyFullDesign,
			"cycle_count", bat.CycleCount,
		)
	}
}

// Len returns the number of batteries.
func (b *Batteries) Len() int {
	return len(b.batteries)
}

func (b *Batteries) healthToString(health int) string {
	if health <= 0 || health > 100 {
		return "Unknown"
	}
	switch {
	case health >= 90:
		return "New"
	case health >= 70:
		return "Good"
	case health >= 50:
		return "Fair"
	case health >= 30:
		return "Weak"
	default:
		return "Poor"
	}
}
