# 🎮 DualSense Mobile Studio - Architecture & Directory Structure

## 📂 Categorized Directory Structure

```
d:\vib\
├── DualSenseServer.exe            # High-performance compiled Windows server binary
├── START_DUALSENSE_SERVER.bat     # 1-Click launcher script
├── build.bat                      # 1-Click automated compiler script
├── ViGEmClient.dll                # Windows Virtual Gamepad Bus DLL
├── config.json                    # Server persistent settings
├── go.mod                         # Go module definition
│
├── server/                        # BACKEND SERVICES (Go)
│   ├── main.go                    # Entrypoint, flags, lifecycle & clean shutdown
│   ├── config.go                  # Settings persistence, IP & path resolvers
│   ├── routes.go                  # HTTP routes, REST APIs & static file serving
│   ├── websocket.go               # Low-latency WebSocket binary gamepad parser
│   ├── udp.go                     # High-speed UDP receiver (Port 9050)
│   ├── vigem.go                   # Virtual DualSense/Xbox controller emulation & rumble
│   ├── hotspot_windows.go         # Windows Mobile Hotspot manager
│   ├── tray_windows.go            # Windows System Tray menu & callbacks
│   ├── web_controller.html        # Production controller HTML bundle
│   └── dashboard.html             # Production desktop dashboard bundle
│
├── web/                           # FRONTEND MODULAR ASSETS
│   ├── controller/                # Mobile Controller
│   │   ├── index.html             # Modular controller markup
│   │   ├── css/
│   │   │   ├── theme.css          # Theme tokens, font variables & resets
│   │   │   ├── controller.css     # Gamepad buttons, D-pad, joysticks, touchpad
│   │   │   ├── aurora.css         # Dynamic aurora mesh, cyber grid, floating glyphs
│   │   │   └── modals.css         # Settings drawer, profile modal, HUD toasts
│   │   └── js/
│   │       ├── constants.js       # Button bitmasks & default configurations
│   │       ├── haptics.js         # Acoustic speaker coil + hardware vibration engine
│   │       ├── network.js         # Full-duplex binary WebSocket communication
│   │       ├── input.js           # Touch tracking, button handlers & gyro POV
│   │       ├── profiles.js        # ProfileManager engine & localStorage persistence
│   │       └── fullscreen.js      # Debounced cross-browser fullscreen & orientation lock
│   │
│   └── dashboard/                 # Desktop Monitor
│       ├── index.html             # Modular dashboard markup
│       ├── css/
│       │   └── dashboard.css      # Dashboard styling, visualizer & stats
│       └── js/
│           └── dashboard.js       # Telemetry & real-time gamepad monitor
│
└── docs/
    └── ARCHITECTURE.md            # System architecture & file map
```

---

## ⚡ Data Flow Architecture

```
[Mobile Phone / Web Controller]
        │
        ▼ (Binary Packets via WebSocket / UDP)
[DualSense Server (Go)]
        ├── websocket.go / udp.go (Binary Protocol Decoder)
        ├── routes.go (/api/status, /controller, /dashboard, /static/)
        └── vigem.go (ViGEm Kernel Driver)
                 │
                 ▼
     [Windows OS Gamepad Subsystem]
                 │
                 ▼ (Rumble Feedback Event)
     [Live Games: Steam / EA / Epic]
```
