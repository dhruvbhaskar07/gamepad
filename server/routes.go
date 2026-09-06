package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

//go:embed web_controller.html
var embeddedWebControllerHTML []byte

//go:embed dashboard.html
var embeddedDashboardHTML []byte

// setupRoutes registers all HTTP endpoints, APIs, and static handlers
func (app *ServerApp) setupRoutes() http.Handler {
	mux := http.NewServeMux()

	// Smart Root Route: Mobile UA -> Controller, Desktop UA -> Dashboard
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		ua := strings.ToLower(r.Header.Get("User-Agent"))
		isMobile := strings.Contains(ua, "mobile") || strings.Contains(ua, "android") || strings.Contains(ua, "iphone") || strings.Contains(ua, "ipad")

		if !isMobile && (strings.HasPrefix(r.Host, "localhost") || strings.HasPrefix(r.Host, "127.0.0.1")) {
			serveDashboard(w, r)
			return
		}

		serveWebController(w, r)
	})

	// Dedicated Controller Page
	mux.HandleFunc("/controller", func(w http.ResponseWriter, r *http.Request) {
		serveWebController(w, r)
	})

	// Dedicated Desktop Dashboard Page
	mux.HandleFunc("/dashboard", func(w http.ResponseWriter, r *http.Request) {
		serveDashboard(w, r)
	})

	// Static Assets Server for modular frontend
	webDir := resolveWebDir()
	if _, err := os.Stat(webDir); err == nil {
		mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(webDir))))
	}

	// Status API
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		app.mu.Lock()
		p1Active := app.p1Connected && time.Since(app.p1LastSeen) < 5*time.Second
		p2Active := app.p2Connected && time.Since(app.p2LastSeen) < 5*time.Second

		status := map[string]interface{}{
			"ip":               app.localIP,
			"http_port":        HTTP_PORT,
			"udp_port":         UDP_PORT,
			"vigem_loaded":     app.driver != nil,
			"p1_connected":     p1Active,
			"p1_ip":            app.p1IP,
			"p1_ua":            app.p1UA,
			"p1_rtt":           1.2,
			"p1_rumble_heavy":  app.p1RumbleHeavy,
			"p1_rumble_light":  app.p1RumbleLight,
			"p2_connected":     p2Active,
			"p2_ip":            app.p2IP,
			"p2_ua":            app.p2UA,
			"p2_rtt":           1.5,
			"p2_rumble_heavy":  app.p2RumbleHeavy,
			"p2_rumble_light":  app.p2RumbleLight,
			"server_timestamp": time.Now().Unix(),
		}
		app.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(status)
	})

	// Hotspot Query & Control API
	mux.HandleFunc("/api/hotspot", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			details := QueryWindowsHotspotDetails()
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(details)
			return
		}

		if r.Method == http.MethodPost {
			var body struct {
				Action string `json:"action"`
			}
			json.NewDecoder(r.Body).Decode(&body)
			enable := body.Action == "on" || body.Action == "start"
			if body.Action == "toggle" {
				cur := QueryWindowsHotspotDetails()
				enable = (cur.State != "On")
			}
			details := ToggleWindowsHotspot(enable)
			app.localIP = getLocalIP()
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(details)
			return
		}
	})

	// Hotspot Settings Shortcut API
	mux.HandleFunc("/api/hotspot/settings", func(w http.ResponseWriter, r *http.Request) {
		OpenWindowsHotspotSettings()
		w.WriteHeader(http.StatusOK)
	})

	// Settings Persistence API
	mux.HandleFunc("/api/settings", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(app.config)
			return
		}
		if r.Method == http.MethodPost {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			var cfg AppConfig
			if err := json.Unmarshal(body, &cfg); err != nil {
				http.Error(w, fmt.Sprintf("Decode error: %v", err), http.StatusBadRequest)
				return
			}
			app.config = cfg
			saveConfig(cfg)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]bool{"ok": true})
			return
		}
	})

	// System Controls (joy.cpl & exit)
	mux.HandleFunc("/api/system/joycpl", func(w http.ResponseWriter, r *http.Request) {
		LaunchJoyCPL()
		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("/api/system/exit", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		go func() {
			time.Sleep(300 * time.Millisecond)
			app.shutdown()
		}()
	})

	// WebSocket Handler
	mux.HandleFunc("/ws", app.handleWebSocket)

	return mux
}

func serveWebController(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate, max-age=0")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	w.Header().Set("Permissions-Policy", "vibrate=*, accelerometer=*, gyroscope=*, fullscreen=*")
	w.Header().Set("Feature-Policy", "vibrate 'self'; accelerometer 'self'; gyroscope 'self'; fullscreen 'self'")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	htmlPath := resolveWebControllerPath()
	if _, err := os.Stat(htmlPath); err == nil {
		http.ServeFile(w, r, htmlPath)
	} else {
		w.Write(embeddedWebControllerHTML)
	}
}

func serveDashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate, max-age=0")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	w.Header().Set("Permissions-Policy", "vibrate=*, accelerometer=*, gyroscope=*, fullscreen=*")
	w.Header().Set("Feature-Policy", "vibrate 'self'; accelerometer 'self'; gyroscope 'self'; fullscreen 'self'")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	dashPath := resolveDashboardPath()
	if _, err := os.Stat(dashPath); err == nil {
		http.ServeFile(w, r, dashPath)
	} else {
		w.Write(embeddedDashboardHTML)
	}
}
