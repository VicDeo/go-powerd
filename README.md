# go-powerd

Linux system tray battery monitor. Reads battery status from sysfs and listens to kernel uevents for real-time updates.

## Build

```bash
go build -o go-powerd ./cmd
```

## Usage

```bash
# Run (starts in system tray)
./go-powerd

# Run with verbose/debug logging
./go-powerd -v
```

Right-click the tray icon for the Quit option. Hover to see capacity, health, and charge/discharge time for each battery.

## License

GPL-3.0. See [LICENSE](LICENSE) for details.
