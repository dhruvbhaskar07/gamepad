package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

const (
	UDP_PORT  = 9050
	HTTP_PORT = 8080
)

// ServerApp manages server state, gamepad instances, and clients
type ServerApp struct {
	mu          sync.Mutex
	driver      *ViGEmDriver
	p1Gamepad   *DualGamepadSlot
	p2Gamepad   *DualGamepadSlot
	localIP     string
	config      AppConfig
	wsClientsMu sync.Mutex
	wsClients   map[*wsWriter]int

	// Real-time telemetry for Dashboard
	p1Connected   bool
	p1IP          string
	p1UA          string
	p1LastSeen    time.Time
	p1RumbleHeavy uint8
	p1RumbleLight uint8

	p2Connected   bool
	p2IP          string
	p2UA          string
	p2LastSeen    time.Time
	p2RumbleHeavy uint8
	p2RumbleLight uint8
}

func main() {
	headlessFlag := flag.Bool("headless", false, "Run in background without opening desktop GUI window")
	flag.Parse()

	fmt.Println("==========================================================")
	fmt.Println("🎮 DualSense Mobile Studio - Desktop Controller Server")
	fmt.Println("==========================================================")

	localIP := getLocalIP()
	cfg := loadConfig()
	fmt.Printf("[NET] Active Local IP: %s\n", localIP)

	app := &ServerApp{
		localIP:   localIP,
		config:    cfg,
		wsClients: make(map[*wsWriter]int),
	}

	// 1. Auto Hotspot if configured
	if cfg.AutoHotspot {
		go func() {
			time.Sleep(1 * time.Second)
			fmt.Println("[HOTSPOT] Auto-enabling Windows Mobile Hotspot...")
			ToggleWindowsHotspot(true)
		}()
	}

	// 2. Initialize ViGEm Kernel Driver
	dllPath := resolveDLLPath()
	fmt.Printf("[VIGEM] Connecting to ViGEmBus driver (%s)...\n", dllPath)
	driver, err := NewViGEmDriver(dllPath)
	if err != nil {
		fmt.Printf("[VIGEM WARNING] Driver failed to load: %v\n", err)
	} else {
		app.driver = driver
		p1, err := driver.CreateDualGamepad()
		if err != nil {
			fmt.Printf("[VIGEM WARNING] Failed to create Player 1 gamepad: %v\n", err)
		} else {
			p1.SetRumbleCallback(func(largeMotor, smallMotor uint8) {
				if app.config.VibrationEnabled {
					app.broadcastRumble(1, largeMotor, smallMotor)
				}
			})
			app.p1Gamepad = p1
			fmt.Println("[VIGEM] 🟢 Player 1 Virtual Controller is ACTIVE with Live Game Rumble Feedback")
		}
	}

	// 3. Setup Clean Shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		app.shutdown()
	}()

	// 4. Start Windows System Tray in Background Thread
	go func() {
		callbacks := TrayCallbacks{
			OnDashboard: func() {
				launchDesktopWindow(fmt.Sprintf("http://127.0.0.1:%d/dashboard", HTTP_PORT))
			},
			OnController: func() {
				exec.Command("cmd", "/c", "start", fmt.Sprintf("http://127.0.0.1:%d/controller", HTTP_PORT)).Start()
			},
			OnHotspot: func() {
				cur := QueryWindowsHotspotDetails()
				ToggleWindowsHotspot(cur.State != "On")
			},
			OnJoyCPL: func() {
				LaunchJoyCPL()
			},
			OnExit: func() {
				app.shutdown()
			},
		}
		StartSystemTray(callbacks)
	}()

	// 5. Start UDP Server for Native App (Port 9050)
	go app.startUDPListener()

	// 6. Launch Desktop Window on Startup (unless --headless)
	if !*headlessFlag {
		go func() {
			time.Sleep(700 * time.Millisecond)
			launchDesktopWindow(fmt.Sprintf("http://127.0.0.1:%d/dashboard", HTTP_PORT))
		}()
	}

	// 7. Start Captive Portal Redirector on Port 80 (Non-blocking / Best effort)
	go func() {
		captiveMux := http.NewServeMux()
		captiveMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			host := app.localIP
			hotspotIP := GetHotspotIP()
			if hotspotIP != "" {
				host = hotspotIP
			}
			targetURL := fmt.Sprintf("http://%s:%d/controller", host, HTTP_PORT)
			w.Header().Set("Location", targetURL)
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusFound)
			fmt.Fprintf(w, `<meta http-equiv="refresh" content="0;url=%s"><p>Redirecting to <a href="%s">DualSense Controller</a>...</p>`, targetURL, targetURL)
		})
		srv80 := &http.Server{
			Addr:    ":80",
			Handler: captiveMux,
		}
		if err := srv80.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("[CAPTIVE] Port 80 listener (%v), skipping captive portal redirector\n", err)
		}
	}()

	// 8. Start Main HTTP & WebSocket Server (Port 8080)
	handler := app.setupRoutes()
	serverAddr := fmt.Sprintf("0.0.0.0:%d", HTTP_PORT)
	fmt.Printf("[HTTP] DualSense Studio Server ready at: http://%s:%d\n", app.localIP, HTTP_PORT)
	fmt.Println("----------------------------------------------------------")
	fmt.Println("🚀 Scan QR code on dashboard to connect phone automatically!")
	fmt.Println("----------------------------------------------------------")

	server := &http.Server{
		Addr:    serverAddr,
		Handler: handler,
	}

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Printf("[HTTP ERROR] Server failed: %v\n", err)
	}
}

// shutdown cleanly releases all controller handles and drivers
func (app *ServerApp) shutdown() {
	fmt.Println("\n[SERVER] Shutting down cleanly. Releasing handles...")
	StopSystemTray()
	app.mu.Lock()
	if app.p1Gamepad != nil {
		app.p1Gamepad.Close()
	}
	if app.p2Gamepad != nil {
		app.p2Gamepad.Close()
	}
	if app.driver != nil {
		app.driver.Close()
	}
	app.mu.Unlock()
	fmt.Println("[SERVER] Goodbye!")
	os.Exit(0)
}

// getPlayerSlot returns or initializes the virtual gamepad for slot 1 or 2
func (app *ServerApp) getPlayerSlot(slotNum int) *DualGamepadSlot {
	app.mu.Lock()
	defer app.mu.Unlock()

	if slotNum == 2 {
		if app.p2Gamepad == nil && app.driver != nil {
			p2, err := app.driver.CreateDualGamepad()
			if err == nil {
				p2.SetRumbleCallback(func(largeMotor, smallMotor uint8) {
					if app.config.VibrationEnabled {
						app.broadcastRumble(2, largeMotor, smallMotor)
					}
				})
				app.p2Gamepad = p2
				fmt.Println("[VIGEM] 🔴 Player 2 Virtual Controller is ACTIVE")
			}
		}
		return app.p2Gamepad
	}
	return app.p1Gamepad
}
