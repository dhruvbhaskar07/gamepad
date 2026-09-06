package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

// AppConfig defines server configuration options
type AppConfig struct {
	AutoStart        bool `json:"auto_start"`
	AutoHotspot      bool `json:"auto_hotspot"`
	MinimizeTray     bool `json:"minimize_tray"`
	VibrationEnabled bool `json:"vibration_enabled"`
}

// resolveDLLPath locates the ViGEmClient.dll driver
func resolveDLLPath() string {
	exePath, err := os.Executable()
	if err == nil {
		candidate := filepath.Join(filepath.Dir(exePath), "ViGEmClient.dll")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	if _, err := os.Stat("ViGEmClient.dll"); err == nil {
		return "ViGEmClient.dll"
	}
	return "ViGEmClient.dll"
}

// resolveWebControllerPath locates web_controller.html on disk
func resolveWebControllerPath() string {
	exePath, err := os.Executable()
	if err == nil {
		candidate := filepath.Join(filepath.Dir(exePath), "server", "web_controller.html")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		candidate2 := filepath.Join(filepath.Dir(exePath), "web_controller.html")
		if _, err := os.Stat(candidate2); err == nil {
			return candidate2
		}
	}
	if _, err := os.Stat(filepath.Join("server", "web_controller.html")); err == nil {
		return filepath.Join("server", "web_controller.html")
	}
	return filepath.Join("server", "web_controller.html")
}

// resolveDashboardPath locates dashboard.html on disk
func resolveDashboardPath() string {
	exePath, err := os.Executable()
	if err == nil {
		candidate := filepath.Join(filepath.Dir(exePath), "server", "dashboard.html")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		candidate2 := filepath.Join(filepath.Dir(exePath), "dashboard.html")
		if _, err := os.Stat(candidate2); err == nil {
			return candidate2
		}
	}
	if _, err := os.Stat(filepath.Join("server", "dashboard.html")); err == nil {
		return filepath.Join("server", "dashboard.html")
	}
	return filepath.Join("server", "dashboard.html")
}

// resolveWebDir locates the modular web/ assets directory
func resolveWebDir() string {
	exePath, err := os.Executable()
	if err == nil {
		candidate := filepath.Join(filepath.Dir(exePath), "web")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	if _, err := os.Stat("web"); err == nil {
		return "web"
	}
	return "web"
}

// loadConfig loads configuration from config.json or returns default
func loadConfig() AppConfig {
	cfg := AppConfig{
		AutoStart:        false,
		AutoHotspot:      false,
		MinimizeTray:     true,
		VibrationEnabled: true,
	}
	exePath, err := os.Executable()
	configPath := "config.json"
	if err == nil {
		configPath = filepath.Join(filepath.Dir(exePath), "config.json")
	}

	data, err := os.ReadFile(configPath)
	if err == nil {
		json.Unmarshal(data, &cfg)
	}
	return cfg
}

// saveConfig writes configuration to disk and updates Windows Run key
func saveConfig(cfg AppConfig) {
	exePath, err := os.Executable()
	configPath := "config.json"
	if err == nil {
		configPath = filepath.Join(filepath.Dir(exePath), "config.json")
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err == nil {
		os.WriteFile(configPath, data, 0644)
	}

	// Update Windows Registry Run key
	if err == nil {
		if cfg.AutoStart {
			cmd := exec.Command("reg", "add", `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`, "/v", "DualSenseServer", "/t", "REG_SZ", "/d", fmt.Sprintf("\"%s\"", exePath), "/f")
			cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
			cmd.Run()
		} else {
			cmd := exec.Command("reg", "delete", `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`, "/v", "DualSenseServer", "/f")
			cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
			cmd.Run()
		}
	}
}

// getLocalIP queries the local IPv4 address
func getLocalIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err == nil {
		defer conn.Close()
		localAddr := conn.LocalAddr().(*net.UDPAddr)
		return localAddr.IP.String()
	}

	addrs, err := net.InterfaceAddrs()
	if err == nil {
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil {
					return ipnet.IP.String()
				}
			}
		}
	}
	return "127.0.0.1"
}

// launchDesktopWindow opens the dashboard in Chrome/Edge app mode
func launchDesktopWindow(url string) {
	edgePaths := []string{
		`C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`,
		`C:\Program Files\Microsoft\Edge\Application\msedge.exe`,
		`C:\Program Files\Google\Chrome\Application\chrome.exe`,
		`C:\Program Files (x86)\Google\Chrome\Application\chrome.exe`,
	}
	for _, p := range edgePaths {
		if _, err := os.Stat(p); err == nil {
			cmd := exec.Command(p, fmt.Sprintf("--app=%s", url), "--window-size=1150,800")
			cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
			if err := cmd.Start(); err == nil {
				return
			}
		}
	}
	exec.Command("cmd", "/c", "start", url).Start()
}
