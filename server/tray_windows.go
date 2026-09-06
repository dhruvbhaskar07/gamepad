package main

import (
	"fmt"
	"runtime"
	"syscall"
	"unsafe"
)

var (
	u32 = syscall.NewLazyDLL("user32.dll")
	s32 = syscall.NewLazyDLL("shell32.dll")
	k32 = syscall.NewLazyDLL("kernel32.dll")

	procRegClassExW      = u32.NewProc("RegisterClassExW")
	procCreateWinExW     = u32.NewProc("CreateWindowExW")
	procDefWndProcW      = u32.NewProc("DefWindowProcW")
	procDestroyWin       = u32.NewProc("DestroyWindow")
	procPostQuitMsg      = u32.NewProc("PostQuitMessage")
	procGetMsgW          = u32.NewProc("GetMessageW")
	procTranslateMsg     = u32.NewProc("TranslateMessage")
	procDispatchMsgW     = u32.NewProc("DispatchMessageW")
	procLoadIconW        = u32.NewProc("LoadIconW")
	procExtractIconW     = s32.NewProc("ExtractIconW")
	procShellNotifyIconW = s32.NewProc("Shell_NotifyIconW")
	procCreatePopupMenu  = u32.NewProc("CreatePopupMenu")
	procAppendMenuW      = u32.NewProc("AppendMenuW")
	procTrackPopupMenu   = u32.NewProc("TrackPopupMenu")
	procDestroyMenu      = u32.NewProc("DestroyMenu")
	procGetCursorPos     = u32.NewProc("GetCursorPos")
	procSetForegroundWin = u32.NewProc("SetForegroundWindow")
	procGetModuleHandleW = k32.NewProc("GetModuleHandleW")
)

const (
	WM_USER          = 0x0400
	WM_TRAYICON      = WM_USER + 1
	WM_COMMAND       = 0x0111
	WM_DESTROY       = 0x0002
	WM_LBUTTONUP     = 0x0202
	WM_LBUTTONDBLCLK = 0x0203
	WM_RBUTTONUP     = 0x0205

	NIM_ADD    = 0x00000000
	NIM_MODIFY = 0x00000001
	NIM_DELETE = 0x00000002

	NIF_MESSAGE = 0x00000001
	NIF_ICON    = 0x00000002
	NIF_TIP     = 0x00000004

	MF_STRING    = 0x00000000
	MF_SEPARATOR = 0x00000800
	MF_DISABLED  = 0x00000002

	TPM_RIGHTBUTTON = 0x0002
	TPM_BOTTOMALIGN = 0x0020

	IDI_APPLICATION = 32512

	CMD_DASHBOARD  = 2001
	CMD_CONTROLLER = 2002
	CMD_HOTSPOT    = 2003
	CMD_JOYCPL     = 2004
	CMD_EXIT       = 2005
)

type POINT struct {
	X, Y int32
}

type MSG struct {
	HWnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      POINT
}

type WNDCLASSEXW struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     uintptr
	HIcon         uintptr
	HCursor       uintptr
	HbrBackground uintptr
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       uintptr
}

type NOTIFYICONDATA struct {
	CbSize            uint32
	HWnd              uintptr
	UID               uint32
	UFlags            uint32
	UCallbackMessage  uint32
	HIcon             uintptr
	SzTip             [128]uint16
	DwState           uint32
	DwStateMask       uint32
	SzInfo            [256]uint16
	UTimeoutOrVersion uint32
	SzInfoTitle       [64]uint16
	DwInfoFlags       uint32
	GuidItem          [16]byte
	HBalloonIcon      uintptr
}

type TrayCallbacks struct {
	OnDashboard  func()
	OnController func()
	OnHotspot    func()
	OnJoyCPL     func()
	OnExit       func()
}

var (
	trayHwnd      uintptr
	trayNID       NOTIFYICONDATA
	trayCallbacks TrayCallbacks
)

func utf16Ptr(s string) *uint16 {
	p, _ := syscall.UTF16PtrFromString(s)
	return p
}

func trayWndProc(hwnd uintptr, msg uint32, wparam, lparam uintptr) uintptr {
	switch msg {
	case WM_TRAYICON:
		switch lparam {
		case WM_LBUTTONUP, WM_LBUTTONDBLCLK:
			if trayCallbacks.OnDashboard != nil {
				go trayCallbacks.OnDashboard()
			}
		case WM_RBUTTONUP:
			showTrayMenu(hwnd)
		}
		return 0

	case WM_COMMAND:
		switch wparam {
		case CMD_DASHBOARD:
			if trayCallbacks.OnDashboard != nil {
				go trayCallbacks.OnDashboard()
			}
		case CMD_CONTROLLER:
			if trayCallbacks.OnController != nil {
				go trayCallbacks.OnController()
			}
		case CMD_HOTSPOT:
			if trayCallbacks.OnHotspot != nil {
				go trayCallbacks.OnHotspot()
			}
		case CMD_JOYCPL:
			if trayCallbacks.OnJoyCPL != nil {
				go trayCallbacks.OnJoyCPL()
			}
		case CMD_EXIT:
			if trayCallbacks.OnExit != nil {
				go trayCallbacks.OnExit()
			}
		}
		return 0

	case WM_DESTROY:
		procPostQuitMsg.Call(0)
		return 0
	}

	ret, _, _ := procDefWndProcW.Call(hwnd, uintptr(msg), wparam, lparam)
	return ret
}

func showTrayMenu(hwnd uintptr) {
	hMenu, _, _ := procCreatePopupMenu.Call()
	if hMenu == 0 {
		return
	}
	defer procDestroyMenu.Call(hMenu)

	procAppendMenuW.Call(hMenu, MF_STRING, CMD_DASHBOARD, uintptr(unsafe.Pointer(utf16Ptr("🎮 Open Dashboard"))))
	procAppendMenuW.Call(hMenu, MF_STRING, CMD_CONTROLLER, uintptr(unsafe.Pointer(utf16Ptr("📱 Open Mobile Controller"))))
	procAppendMenuW.Call(hMenu, MF_SEPARATOR, 0, 0)
	procAppendMenuW.Call(hMenu, MF_STRING, CMD_HOTSPOT, uintptr(unsafe.Pointer(utf16Ptr("📶 Toggle Mobile Hotspot"))))
	procAppendMenuW.Call(hMenu, MF_STRING, CMD_JOYCPL, uintptr(unsafe.Pointer(utf16Ptr("⚙️ Gamepad Settings (joy.cpl)"))))
	procAppendMenuW.Call(hMenu, MF_SEPARATOR, 0, 0)
	procAppendMenuW.Call(hMenu, MF_STRING, CMD_EXIT, uintptr(unsafe.Pointer(utf16Ptr("❌ Exit DualSense Server"))))

	var pt POINT
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))

	procSetForegroundWin.Call(hwnd)
	procTrackPopupMenu.Call(hMenu, TPM_RIGHTBUTTON|TPM_BOTTOMALIGN, uintptr(pt.X), uintptr(pt.Y), 0, hwnd, 0)
}

// StartSystemTray starts the Windows Notification Area tray icon and message loop
func StartSystemTray(callbacks TrayCallbacks) error {
	runtime.LockOSThread()

	trayCallbacks = callbacks

	hInstance, _, _ := procGetModuleHandleW.Call(0)
	className := utf16Ptr("DualSenseTrayClass")

	wc := WNDCLASSEXW{
		CbSize:        uint32(unsafe.Sizeof(WNDCLASSEXW{})),
		LpfnWndProc:   syscall.NewCallback(trayWndProc),
		HInstance:     hInstance,
		LpszClassName: className,
	}
	procRegClassExW.Call(uintptr(unsafe.Pointer(&wc)))

	hwnd, _, _ := procCreateWinExW.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(utf16Ptr("DualSenseTrayWindow"))),
		0, 0, 0, 0, 0,
		0, 0, hInstance, 0,
	)
	if hwnd == 0 {
		return fmt.Errorf("failed to create tray message window")
	}
	trayHwnd = hwnd

	hIcon, _, _ := procLoadIconW.Call(0, uintptr(IDI_APPLICATION))

	trayNID = NOTIFYICONDATA{
		CbSize:           uint32(unsafe.Sizeof(NOTIFYICONDATA{})),
		HWnd:             trayHwnd,
		UID:              1,
		UFlags:           NIF_MESSAGE | NIF_ICON | NIF_TIP,
		UCallbackMessage: WM_TRAYICON,
		HIcon:            hIcon,
	}
	copy(trayNID.SzTip[:], syscall.StringToUTF16("DualSense Mobile - Running (Click to open)"))

	procShellNotifyIconW.Call(NIM_ADD, uintptr(unsafe.Pointer(&trayNID)))

	var msg MSG
	for {
		ret, _, _ := procGetMsgW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if int32(ret) <= 0 {
			break
		}
		procTranslateMsg.Call(uintptr(unsafe.Pointer(&msg)))
		procDispatchMsgW.Call(uintptr(unsafe.Pointer(&msg)))
	}

	procShellNotifyIconW.Call(NIM_DELETE, uintptr(unsafe.Pointer(&trayNID)))
	return nil
}

// StopSystemTray removes the tray icon and exits the message loop
func StopSystemTray() {
	if trayHwnd != 0 {
		procShellNotifyIconW.Call(NIM_DELETE, uintptr(unsafe.Pointer(&trayNID)))
		procDestroyWin.Call(trayHwnd)
		trayHwnd = 0
	}
}
