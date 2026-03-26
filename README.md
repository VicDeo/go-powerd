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

# Explicit config path
./go-powerd -c /path/to/config.toml
```

### CLI options

| Option | Description |
|--------|-------------|
| `-c` | Path to `config.toml`. If omitted, uses the default under the user config directory (`$XDG_CONFIG_HOME/go-powerd/config.toml`, or `~/.config/go-powerd/config.toml` when `XDG_CONFIG_HOME` is unset). |
| `-v` | Verbose logging: debug level and source locations in log lines. |
| `-t` | Attach to the system tray instead of printing status and exiting. |
| `-h` | Print help and exit. |

```bash
./go-powerd -h
```

With **`-t`**, right-click the tray icon for Quit. Hover the icon for capacity, health, and charge/discharge time per battery.

### Configuration

If you do not pass **`-c`**, go-powerd loads:

**`$XDG_CONFIG_HOME/go-powerd/config.toml`**

When **`XDG_CONFIG_HOME`** is not set, that is the same as:

**`~/.config/go-powerd/config.toml`**

Copy the sample from [`configs/config.toml`](configs/config.toml) into that location (create the `go-powerd` directory if needed), then adjust thresholds and policies.

## License

GPL-3.0. See [LICENSE](LICENSE) for details.
