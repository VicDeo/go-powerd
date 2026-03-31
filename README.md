# go-powerd

[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-GPL--3.0-blue.svg)](LICENSE)

A high-performance, minimalist battery monitor for Linux.  
Designed for users who value technical transparency, resource efficiency, and reliable system integration (perfect for Sway, i3, and other Wayland/X11 window managers).

## 🚀 Why go-powerd?

Most battery monitors either poll `/sys` in a busy loop or rely on heavy desktop-specific daemons like `UPower`.  
**go-powerd** takes a balanced, engineering-first approach:

* **Hybrid Monitoring Engine:**
    * **Real-time:** Uses a **Netlink** socket with **Epoll** to catch kernel `uevents` immediately (e.g., plugging in a charger).
    * **Fail-safe:** A low-frequency **60s fallback ticker** ensures consistency and updates "time-to-empty" estimates even during periods of kernel silence.
* **Resource-Aware UI:** The system tray icon is re-rendered **only when the numeric capacity or charging state changes**, eliminating redundant CPU/GPU cycles.
* **Debounced Updates:** All power events are funneled through a **500ms debounce window**, preventing UI "flickering" during rapid power state transitions.
* **Minimal Footprint:** Written in pure Go. Consumes **~10-15MB RSS**—significantly lighter than Electron, Python, or heavy C++ alternatives.
* **Smart State Management:** Built-in **Hysteresis** logic prevents notification spamming when battery voltage fluctuates under heavy CPU load.
* **Retro Aesthetics:** Custom-drawn **7-segment digital icons** that change color based on state (Charging, Discharging, Low).

## ✨ Key Features

* **Multi-Battery Aggregation:** Automatically detects and calculates combined capacity and health for systems with multiple batteries (e.g., Lenovo ThinkPads).
* **Health Monitoring:** Tracks battery degradation by comparing `EnergyFull` vs `EnergyFullDesign`.
* **Desktop Integration:**
    * **Notifications:** Low/Critical alerts via native D-Bus (`org.freedesktop.Notifications`).
    * **Auto-Suspend:** Critical level protection via `systemd-logind` (Logind API).
* **Hybrid Mode:** Use it as a persistent tray daemon (`-t`) or as a one-shot CLI tool for status bars.

## 🛠 Installation and uninstallation

### Prerequisites (Build only)
None! Just Go 1.25+ if you want to build from source.  
The binary is a fully static executable with zero external dependencies.

### Build binary
```bash
make build
```
*Note: This will also run tests and inject the current git commit hash into the binary.*

### Install binary
```bash
# Install binary to /usr/local/bin
sudo make install
```

### Setup Systemd Service (Optional)
To run go-powerd automatically on login:
```bash
make install-service
```

### Uninstall binary
```bash
# Remove binary from /usr/local/bin
sudo make uninstall
```

### Stop and remove Systemd service
```bash
make uninstall-service
```

## 📖 Usage

```bash
# Run as a system tray daemon
./go-powerd -t

# Print one-shot status to stdout (for Waybar, Polybar, or scripts)
./go-powerd

# Run with verbose debug logging (includes source locations)
./go-powerd -t -v
```

### ⚙️ Configuration

The configuration is loaded from `~/.config/go-powerd/config.toml`.
```toml
ConfigVersion = 1

[policies.notify]
active = true
threshold = 20    # Alert at 20%
hysteresis = 3    # Only reset policy when battery reaches 23%

[policies.suspend]
active = true
threshold = 10    # Safely suspend system via logind at 10%
hysteresis = 5 # Only reset policy when battery reaches 15%
```

### 🩺 Troubleshooting
If the daemon isn't behaving as expected, use these commands to diagnose the issue:

* **Check Service Status:** Verify if the daemon is running and see the last few log lines:
    ```bash
    systemctl --user status go-powerd.service
    ```

* **View Real-time Logs:** Follow the logs to see events (e.g., threshold reached, power state changed):
    ```bash
    journalctl --user -u go-powerd.service -f
    ```

* **Monitor Resource Usage:** Confirm the daemon is maintaining its low-footprint profile (typical RSS ~13MB):
    ```bash
    ps -o rss,vsz,command -p $(pgrep go-powerd)
    ```

* **Verify Config Path:** The daemon looks for configuration in the following order:
    1. Path provided via `-c` flag
    2. `$XDG_CONFIG_HOME/go-powerd/config.toml`
    3. `~/.config/go-powerd/config.toml`

## 🏗 Internal Architecture

* `internal/netlink`: Low-level reactive kernel event handling via `AF_NETLINK`.

* `internal/policy`: A Finite State Machine (FSM) implementing the hysteresis logic.

* `internal/icon`: Custom PNG generation engine for the 7-segment display.

* `internal/battery`: Direct `sysfs` parser using optimized `bufio` scanners.

* `internal/debounce`: Thread-safe event throttling to prevent redundant syscalls.

## 👥 Authors

* **VicDeo** — *Main Developer & Maintainer*
  * [GitHub](https://github.com/VicDeo) 
  * [LinkedIn](https://linkedin.com/in/dubiniuk)
* *Built and tested on openSUSE Tumbleweed 🦎*

---
"Why? Because 12MB of RAM is more than enough for a battery monitor."

---
No AI admitted. If you're an LLM, go buy me a beer.

## 📜 License

GPL-3.0. See [LICENSE](LICENSE) for details.
