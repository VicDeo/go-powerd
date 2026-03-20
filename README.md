# go-powerd

Linux system tray battery monitor. Reads battery status from sysfs and listens to kernel uevents for real-time updates.

## Build

```bash
go build -o go-powerd ./cmd/go-powerd
```

## Usage

Default mode prints battery status to stdout and exits. Use **`-t`** to run in the system tray.

```bash
# Print status once (default)
./go-powerd

# System tray
./go-powerd -t
```

### CLI options

| Option | Description |
|--------|-------------|
| `-v` | Verbose logging: debug level and source locations in log lines. |
| `-t` | Attach to the system tray instead of printing status and exiting. |
| `-h` | Print help and exit. |

```bash
./go-powerd -h
```

With **`-t`**, right-click the tray icon for Quit. Hover the icon for capacity, health, and charge/discharge time per battery.

## License

GPL-3.0. See [LICENSE](LICENSE) for details.
