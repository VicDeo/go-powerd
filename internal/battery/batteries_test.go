package battery

import (
	"os"
	"path/filepath"
	"testing"
)

var (
	bats = map[string]string{
		"bat0": `DEVTYPE=power_supply
POWER_SUPPLY_NAME=BAT1
POWER_SUPPLY_TYPE=Battery
POWER_SUPPLY_STATUS=Not charging
POWER_SUPPLY_PRESENT=1
POWER_SUPPLY_TECHNOLOGY=Li-ion
POWER_SUPPLY_CYCLE_COUNT=78
POWER_SUPPLY_VOLTAGE_MIN_DESIGN=11100000
POWER_SUPPLY_VOLTAGE_NOW=9753000
POWER_SUPPLY_POWER_NOW=0
POWER_SUPPLY_ENERGY_FULL_DESIGN=71040000
POWER_SUPPLY_ENERGY_FULL=66220000
POWER_SUPPLY_ENERGY_NOW=4570000
POWER_SUPPLY_CAPACITY=7
POWER_SUPPLY_CAPACITY_LEVEL=Normal
POWER_SUPPLY_TYPE=Battery
POWER_SUPPLY_MODEL_NAME=01AV425
POWER_SUPPLY_MANUFACTURER=SANYO
POWER_SUPPLY_SERIAL_NUMBER=30831
`,
		"bat1": `DEVTYPE=power_supply
POWER_SUPPLY_NAME=BAT0
POWER_SUPPLY_TYPE=Battery
POWER_SUPPLY_STATUS=Not charging
POWER_SUPPLY_PRESENT=1
POWER_SUPPLY_TECHNOLOGY=Li-poly
POWER_SUPPLY_CYCLE_COUNT=236
POWER_SUPPLY_VOLTAGE_MIN_DESIGN=15200000
POWER_SUPPLY_VOLTAGE_NOW=16575000
POWER_SUPPLY_POWER_NOW=0
POWER_SUPPLY_ENERGY_FULL_DESIGN=31920000
POWER_SUPPLY_ENERGY_FULL=17940000
POWER_SUPPLY_ENERGY_NOW=14270000
POWER_SUPPLY_CAPACITY=80
POWER_SUPPLY_CAPACITY_LEVEL=Normal
POWER_SUPPLY_TYPE=Battery
POWER_SUPPLY_MODEL_NAME=01AV493
POWER_SUPPLY_MANUFACTURER=LGC
POWER_SUPPLY_SERIAL_NUMBER= 1321`,
	}
)

func TestLoad(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "battery_test")
	if err != nil {
		t.Fatalf("Failed to create a temp directory: %s", err)
	}
	defer os.RemoveAll(tmpDir)
	if err := mockBattery(tmpDir, "bat0"); err != nil {
		t.Fatalf("Failed to add test battery bat0: %s", err)
	}
	if err := mockBattery(tmpDir, "bat1"); err != nil {
		t.Fatalf("Failed to add test battery bat1: %s", err)
	}
	if err := mockDeviceWithTypeMissing(tmpDir, "bat1"); err != nil {
		t.Fatalf("Failed to add device with the missing type: %s", err)
	}
	if err := mockDeviceNonBatteryType(tmpDir, "bat1"); err != nil {
		t.Fatalf("Failed to add device with the non-battery type: %s", err)
	}

	batteries := NewBatteries(tmpDir)
	if err := batteries.Load(); err != nil {
		t.Fatalf("Unexpected error while loading batteries: %s", err)
	}

	expectedCapacity := 22
	if batteries.Capacity() != expectedCapacity {
		t.Fatalf("Expected capacity is %d, got %d", expectedCapacity, batteries.Capacity())
	}
}

func mockBattery(baseDir string, name string) error {
	batDir := filepath.Join(baseDir, name)
	if err := os.Mkdir(batDir, 0755); err != nil {
		return err
	}

	mockType := filepath.Join(batDir, typeFilename)
	if err := os.WriteFile(mockType, batteryType, 0644); err != nil {
		return err
	}
	mockUevent := filepath.Join(batDir, batteryStatsFilename)
	if err := os.WriteFile(mockUevent, []byte(bats[name]), 0644); err != nil {
		return err
	}
	return nil
}

func mockDeviceWithTypeMissing(baseDir string, name string) error {
	dirName := name + "fake1"
	batDir := filepath.Join(baseDir, dirName)
	if err := os.Mkdir(batDir, 0755); err != nil {
		return err
	}
	mockUevent := filepath.Join(batDir, batteryStatsFilename)
	if err := os.WriteFile(mockUevent, []byte(bats[name]), 0644); err != nil {
		return err
	}
	return nil
}

func mockDeviceNonBatteryType(baseDir string, name string) error {
	dirName := name + "fake2"
	batDir := filepath.Join(baseDir, dirName)
	if err := os.Mkdir(batDir, 0755); err != nil {
		return err
	}

	mockType := filepath.Join(batDir, typeFilename)
	if err := os.WriteFile(mockType, []byte("teapot"), 0644); err != nil {
		return err
	}
	mockUevent := filepath.Join(batDir, batteryStatsFilename)
	if err := os.WriteFile(mockUevent, []byte(bats[name]), 0644); err != nil {
		return err
	}
	return nil
}
