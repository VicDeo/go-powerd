// battery is the package to deal with battery info.
package battery

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path"

	"github.com/VicDeo/go-powerd/internal/pool"
)

const (
	keyDevType          = "DEVTYPE"           // Unused
	keyType             = "POWER_SUPPLY_TYPE" // Unused
	keyName             = "POWER_SUPPLY_NAME"
	keyStatus           = "POWER_SUPPLY_STATUS"
	keyPresent          = "POWER_SUPPLY_PRESENT"
	keyTechnology       = "POWER_SUPPLY_TECHNOLOGY"
	keyCycleCount       = "POWER_SUPPLY_CYCLE_COUNT"
	keyVoltageMinDesign = "POWER_SUPPLY_VOLTAGE_MIN_DESIGN"
	keyVoltageNow       = "POWER_SUPPLY_VOLTAGE_NOW"
	keyPowerNow         = "POWER_SUPPLY_POWER_NOW"
	keyEnergyFullDesign = "POWER_SUPPLY_ENERGY_FULL_DESIGN"
	keyEnergyFull       = "POWER_SUPPLY_ENERGY_FULL"
	keyEnergyNow        = "POWER_SUPPLY_ENERGY_NOW"
	keyCapacity         = "POWER_SUPPLY_CAPACITY"
	keyCapacityLevel    = "POWER_SUPPLY_CAPACITY_LEVEL"
	keyModelName        = "POWER_SUPPLY_MODEL_NAME"
	keyManufacturer     = "POWER_SUPPLY_MANUFACTURER"
	keySerialNumber     = "POWER_SUPPLY_SERIAL_NUMBER"
)

const (
	// batteryStatsFilename is the name of the file that contains the battery statistics.
	batteryStatsFilename = "uevent"
)

const (
	// statusFull is the status of the battery when it is full.
	statusFull = "Full"
	// statusCharging is the status of the battery when it is charging.
	statusCharging = "Charging"
	// statusDischarging is the status of the battery when it is discharging.
	statusDischarging = "Discharging"
	// statusNotCharging is the status of the battery when it is not charging.
	statusNotCharging = "Not charging"
)

var (
	setters = map[string]func(*Battery, []byte) error{
		keyType:             func(b *Battery, v []byte) error { /* unused */ return nil },
		keyDevType:          func(b *Battery, v []byte) error { /* unused */ return nil },
		keyName:             func(b *Battery, v []byte) error { b.Name = string(v); return nil },
		keyStatus:           func(b *Battery, v []byte) error { b.Status = string(v); return nil },
		keyPresent:          func(b *Battery, v []byte) error { b.Present = len(v) > 0 && v[0] == '1'; return nil },
		keyTechnology:       func(b *Battery, v []byte) error { b.Technology = string(v); return nil },
		keyCycleCount:       func(b *Battery, v []byte) error { return parseInt(v, &b.CycleCount) },
		keyVoltageMinDesign: func(b *Battery, v []byte) error { return parseInt64(v, &b.VoltageMinDesign) },
		keyVoltageNow:       func(b *Battery, v []byte) error { return parseInt64(v, &b.VoltageNow) },
		keyPowerNow:         func(b *Battery, v []byte) error { return parseInt64(v, &b.PowerNow) },
		keyEnergyFullDesign: func(b *Battery, v []byte) error { return parseInt64(v, &b.EnergyFullDesign) },
		keyEnergyFull:       func(b *Battery, v []byte) error { return parseInt64(v, &b.EnergyFull) },
		keyEnergyNow:        func(b *Battery, v []byte) error { return parseInt64(v, &b.EnergyNow) },
		keyCapacity:         func(b *Battery, v []byte) error { return parseInt(v, &b.Capacity) },
		keyCapacityLevel:    func(b *Battery, v []byte) error { b.CapacityLevel = string(v); return nil },
		keyModelName:        func(b *Battery, v []byte) error { b.ModelName = string(v); return nil },
		keyManufacturer:     func(b *Battery, v []byte) error { b.Manufacturer = string(v); return nil },
		keySerialNumber:     func(b *Battery, v []byte) error { b.SerialNumber = string(v); return nil },
	}
)

// Battery represents a crucial battery params.
type Battery struct {
	Path             string // Path to the battery directory
	Name             string // Battery name
	ModelName        string // Battery model
	Manufacturer     string // Battery vendor
	Technology       string // Battery technology
	SerialNumber     string // Battery serial
	Status           string // Charging/Not charging/Discharging
	CapacityLevel    string // Normal/Low
	Capacity         int    // 0-100
	VoltageMinDesign int64  // microvolts
	VoltageNow       int64  // microvolts
	EnergyNow        int64  // microwatt-hours
	EnergyFull       int64  // microwatt-hours
	EnergyFullDesign int64  // microwatt-hours
	PowerNow         int64  // microwatts
	CycleCount       int    // number of full charge-discharge cycles
	Present          bool   // If the battery attached right now
}

// New creates a new battery.
func New(path string) *Battery {
	return &Battery{Path: path}
}

// Load loads the battery info from the uevent file.
func (b *Battery) Load() error {
	sysfs := os.DirFS(b.Path)
	batteryStatsFile, err := sysfs.Open(batteryStatsFilename)
	if err != nil {
		return fmt.Errorf("failed to open battery stats file at %s: %w", path.Join(b.Path, batteryStatsFilename), err)
	}
	defer batteryStatsFile.Close()

	buf := pool.Get()
	defer pool.Put(buf)
	n, err := batteryStatsFile.Read(buf.Bytes())
	if err != nil && err != io.EOF {
		return err
	}

	buf.SetLen(n)
	remaining := buf.Data()
	for len(remaining) > 0 {
		idx := bytes.IndexByte(remaining, '\n')
		var line []byte
		if idx == -1 {
			line = remaining
			remaining = nil
		} else {
			line = remaining[:idx]
			remaining = remaining[idx+1:]
		}
		if len(line) == 0 {
			continue
		}

		key, rawValue, ok := bytes.Cut(line, []byte("="))
		if !ok {
			slog.Debug("Line does not match key=value format", "line", string(line))
			continue
		}

		parser, ok := setters[string(key)]
		if !ok {
			slog.Debug("Unknown attribute in battery stats", "line", string(line))
			continue
		}
		if err := parser(b, rawValue); err != nil {
			slog.Error("Battery attribute has invalid value", "line", string(line), "error", err)
		}
	}
	return nil
}

// Health returns the battery health in percent.
func (b *Battery) Health() (int, error) {
	if b.EnergyFull != 0 && b.EnergyFullDesign != 0 {
		return int(100 * b.EnergyFull / b.EnergyFullDesign), nil
	}
	return 0, fmt.Errorf("not enough data to proceed")
}

// ExtendedStatus returns a battery status augmented with the time to charge or discharge.
func (b *Battery) ExtendedStatus() string {
	extendedStatus := b.Status
	switch {
	case b.Status == statusCharging && b.PowerNow != 0:
		timeToFull := float64(3600*(b.EnergyFull-b.EnergyNow)) / float64(b.PowerNow)
		extendedStatus = fmt.Sprintf("full in %s", formatDuration(timeToFull))
	case b.Status == statusDischarging && b.PowerNow != 0:
		timeToEmpty := float64(3600*b.EnergyNow) / float64(b.PowerNow)
		extendedStatus = fmt.Sprintf("empty in %s", formatDuration(timeToEmpty))
	case b.Status == statusFull ||
		b.Status == statusNotCharging ||
		b.Status == statusCharging ||
		b.Status == statusDischarging:
		return b.Status
	default:
		slog.Warn("Unknown status detected", "battery", b.Name, "status", b.Status)
	}
	return extendedStatus
}

// parseInt is a universal helper for integers.
func parseInt(raw []byte, target *int) error {
	val, err := atoi64(raw)
	if err != nil {
		return err
	}
	*target = int(val) // Update the actual struct field
	return nil
}

// parseInt64 is a universal helper for int64 (microunits).
func parseInt64(raw []byte, target *int64) error {
	val, err := atoi64(raw)
	if err != nil {
		return err
	}
	*target = val
	return nil
}

// formatDuration is a helper to format seconds to the human readable format.
func formatDuration(seconds float64) string {
	hours := int64(seconds / 3600)
	minutes := (int64(seconds) % 3600) / 60

	return fmt.Sprintf("%dh %02dm", hours, minutes)
}

func atoi64(b []byte) (int64, error) {
	var res int64
	for i := 0; i < len(b); i++ {
		if b[i] >= byte('0') && b[i] <= byte('9') {
			res = res*10 + int64(b[i]-'0')
		} else {
			return 0, fmt.Errorf("ascii to int conversion failed. invalid source: %s", string(b))
		}
	}
	return res, nil
}
