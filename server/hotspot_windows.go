package main

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

// hotspotScriptPS is a self-contained PowerShell script utilizing WinRT NetworkOperatorTetheringManager
const hotspotScriptPS = `
Add-Type -AssemblyName System.Runtime.WindowsRuntime
$asTaskGeneric = ([System.WindowsRuntimeSystemExtensions].GetMethods() | Where-Object { $_.Name -eq 'AsTask' -and $_.GetParameters().Count -eq 1 -and $_.GetParameters()[0].ParameterType.Name -eq 'IAsyncOperation` + "`" + `1' })[0]

function Await($WinRtTask, $ResultType) {
    $asTask = $asTaskGeneric.MakeGenericMethod($ResultType)
    $netTask = $asTask.Invoke($null, @($WinRtTask))
    $netTask.Wait(14000) | Out-Null
    $netTask.Result
}

[Windows.Networking.Connectivity.NetworkInformation, Windows.Networking.Connectivity, ContentType = WindowsRuntime] | Out-Null
[Windows.Networking.NetworkOperators.NetworkOperatorTetheringManager, Windows.Networking.NetworkOperators, ContentType = WindowsRuntime] | Out-Null

$profile = [Windows.Networking.Connectivity.NetworkInformation]::GetInternetConnectionProfile()
if ($null -eq $profile) {
    Write-Output "STATE:NO_PROFILE"
    exit 0
}

$action = "{{ACTION}}"

try {
    $mgr = [Windows.Networking.NetworkOperators.NetworkOperatorTetheringManager]::CreateFromConnectionProfile($profile)
    if ($action -eq "on") {
        if ($mgr.TetheringOperationalState -ne [Windows.Networking.NetworkOperators.TetheringOperationalState]::On) {
            $res = Await ($mgr.StartTetheringAsync()) ([Windows.Networking.NetworkOperators.NetworkOperatorTetheringOperationResult])
        }
    } elseif ($action -eq "off") {
        if ($mgr.TetheringOperationalState -ne [Windows.Networking.NetworkOperators.TetheringOperationalState]::Off) {
            $res = Await ($mgr.StopTetheringAsync()) ([Windows.Networking.NetworkOperators.NetworkOperatorTetheringOperationResult])
        }
    }

    $cfg = $mgr.GetCurrentAccessPointConfiguration()
    $ssid = if ($cfg) { $cfg.Ssid } else { "" }
    $pass = if ($cfg) { $cfg.Passphrase } else { "" }

    Write-Output "STATE:$($mgr.TetheringOperationalState)"
    Write-Output "SSID:$ssid"
    Write-Output "PASSPHRASE:$pass"
} catch {
    Write-Output "ERROR:$($_.Exception.Message)"
}
`

type HotspotDetails struct {
	State      string `json:"state"`
	SSID       string `json:"ssid"`
	Passphrase string `json:"passphrase"`
	HotspotIP  string `json:"hotspot_ip"`
	WifiIP     string `json:"wifi_ip"`
	WifiQR     string `json:"wifi_qr"`
	GamepadURL string `json:"gamepad_url"`
}

// GetHotspotIP returns the virtual adapter IP assigned by Windows Mobile Hotspot (usually 192.168.137.1)
func GetHotspotIP() string {
	addrs, err := net.InterfaceAddrs()
	if err == nil {
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				ip := ipnet.IP.To4()
				if ip != nil && ip.String() == "192.168.137.1" {
					return "192.168.137.1"
				}
			}
		}
	}
	return ""
}

// QueryWindowsHotspotState queries the current Windows Mobile Hotspot state and credentials
func QueryWindowsHotspotDetails() HotspotDetails {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	script := strings.Replace(hotspotScriptPS, "{{ACTION}}", "status", 1)
	cmd := exec.CommandContext(ctx, "powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.Output()

	details := HotspotDetails{
		State:  "Off",
		WifiIP: getLocalIP(),
	}

	if err != nil {
		details.State = "Unknown"
		return details
	}

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "STATE:") {
			st := strings.TrimPrefix(line, "STATE:")
			if st == "NO_PROFILE" {
				details.State = "No Profile"
			} else {
				details.State = st
			}
		} else if strings.HasPrefix(line, "SSID:") {
			details.SSID = strings.TrimPrefix(line, "SSID:")
		} else if strings.HasPrefix(line, "PASSPHRASE:") {
			details.Passphrase = strings.TrimPrefix(line, "PASSPHRASE:")
		}
	}

	details.HotspotIP = GetHotspotIP()
	if details.SSID != "" {
		details.WifiQR = fmt.Sprintf("WIFI:T:WPA;S:%s;P:%s;;", details.SSID, details.Passphrase)
	}

	bestIP := details.WifiIP
	if details.HotspotIP != "" {
		bestIP = details.HotspotIP
	}
	details.GamepadURL = fmt.Sprintf("http://%s:%d", bestIP, HTTP_PORT)

	return details
}

// ToggleWindowsHotspot enables or disables Windows Mobile Hotspot and returns full details
func ToggleWindowsHotspot(enable bool) HotspotDetails {
	action := "off"
	if enable {
		action = "on"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 16*time.Second)
	defer cancel()

	script := strings.Replace(hotspotScriptPS, "{{ACTION}}", action, 1)
	cmd := exec.CommandContext(ctx, "powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.Output()

	details := HotspotDetails{
		State:  "Off",
		WifiIP: getLocalIP(),
	}

	if err != nil {
		fmt.Printf("[HOTSPOT ERROR] Toggle failed: %v\n", err)
		details.State = "Failed"
		return details
	}

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "STATE:") {
			details.State = strings.TrimPrefix(line, "STATE:")
		} else if strings.HasPrefix(line, "SSID:") {
			details.SSID = strings.TrimPrefix(line, "SSID:")
		} else if strings.HasPrefix(line, "PASSPHRASE:") {
			details.Passphrase = strings.TrimPrefix(line, "PASSPHRASE:")
		}
	}

	// Give Windows a brief moment to assign the virtual hotspot adapter IP
	time.Sleep(500 * time.Millisecond)
	details.HotspotIP = GetHotspotIP()
	if details.SSID != "" {
		details.WifiQR = fmt.Sprintf("WIFI:T:WPA;S:%s;P:%s;;", details.SSID, details.Passphrase)
	}

	bestIP := details.WifiIP
	if details.HotspotIP != "" {
		bestIP = details.HotspotIP
	}
	details.GamepadURL = fmt.Sprintf("http://%s:%d", bestIP, HTTP_PORT)

	return details
}

// OpenWindowsHotspotSettings opens the native Windows Mobile Hotspot Settings app
func OpenWindowsHotspotSettings() {
	cmd := exec.Command("cmd", "/c", "start", "ms-settings:network-mobilehotspot")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	cmd.Start()
}

// LaunchJoyCPL opens the Windows Game Controllers dialog (joy.cpl)
func LaunchJoyCPL() {
	cmd := exec.Command("control", "joy.cpl")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	cmd.Start()
}
