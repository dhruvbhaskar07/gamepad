package main

import (
	"fmt"
	"path/filepath"
	"sync"
	"syscall"
	"unsafe"
)

const (
	VIGEM_ERROR_NONE = 0x20000000

	// Xbox 360 Buttons
	XUSB_GAMEPAD_DPAD_UP        = 0x0001
	XUSB_GAMEPAD_DPAD_DOWN      = 0x0002
	XUSB_GAMEPAD_DPAD_LEFT      = 0x0004
	XUSB_GAMEPAD_DPAD_RIGHT     = 0x0008
	XUSB_GAMEPAD_START          = 0x0010
	XUSB_GAMEPAD_BACK           = 0x0020
	XUSB_GAMEPAD_LEFT_THUMB     = 0x0040
	XUSB_GAMEPAD_RIGHT_THUMB    = 0x0080
	XUSB_GAMEPAD_LEFT_SHOULDER  = 0x0100
	XUSB_GAMEPAD_RIGHT_SHOULDER = 0x0200
	XUSB_GAMEPAD_GUIDE          = 0x0400
	XUSB_GAMEPAD_A              = 0x1000
	XUSB_GAMEPAD_B              = 0x2000
	XUSB_GAMEPAD_X              = 0x4000
	XUSB_GAMEPAD_Y              = 0x8000

	// DualShock 4 Buttons
	DS4_BUTTON_SQUARE         = 1 << 4
	DS4_BUTTON_CROSS          = 1 << 5
	DS4_BUTTON_CIRCLE         = 1 << 6
	DS4_BUTTON_TRIANGLE       = 1 << 7
	DS4_BUTTON_SHOULDER_LEFT  = 1 << 8
	DS4_BUTTON_SHOULDER_RIGHT = 1 << 9
	DS4_BUTTON_TRIGGER_LEFT   = 1 << 10
	DS4_BUTTON_TRIGGER_RIGHT  = 1 << 11
	DS4_BUTTON_SHARE          = 1 << 12
	DS4_BUTTON_OPTIONS        = 1 << 13
	DS4_BUTTON_THUMB_LEFT     = 1 << 14
	DS4_BUTTON_THUMB_RIGHT    = 1 << 15

	// DualShock 4 Special Buttons
	DS4_SPECIAL_BUTTON_PS       = 1 << 0
	DS4_SPECIAL_BUTTON_TOUCHPAD = 1 << 1

	// DualShock 4 D-Pad (HAT)
	DS4_BUTTON_DPAD_NORTH     = 0x0
	DS4_BUTTON_DPAD_NORTHEAST = 0x1
	DS4_BUTTON_DPAD_EAST      = 0x2
	DS4_BUTTON_DPAD_SOUTHEAST = 0x3
	DS4_BUTTON_DPAD_SOUTH     = 0x4
	DS4_BUTTON_DPAD_SOUTHWEST = 0x5
	DS4_BUTTON_DPAD_WEST      = 0x6
	DS4_BUTTON_DPAD_NORTHWEST = 0x7
	DS4_BUTTON_DPAD_NONE      = 0x8
)

// XUSB_REPORT structure matching ViGEmBus
type XusbReport struct {
	WButtons      uint16
	BLeftTrigger  uint8
	BRightTrigger uint8
	SThumbLX      int16
	SThumbLY      int16
	SThumbRX      int16
	SThumbRY      int16
}

// DS4_REPORT structure matching ViGEmBus
type Ds4Report struct {
	BThumbLX  uint8
	BThumbLY  uint8
	BThumbRX  uint8
	BThumbRY  uint8
	WButtons  uint16
	BSpecial  uint8
	BTriggerL uint8
	BTriggerR uint8
}

type ViGEmDriver struct {
	client                    uintptr
	vigemTargetX360Alloc      *syscall.LazyProc
	vigemTargetDS4Alloc       *syscall.LazyProc
	vigemTargetAdd            *syscall.LazyProc
	vigemTargetRemove         *syscall.LazyProc
	vigemTargetFree           *syscall.LazyProc
	vigemTargetX360Upd        *syscall.LazyProc
	vigemTargetDS4Upd         *syscall.LazyProc
	vigemTargetX360RegNotif   *syscall.LazyProc
	vigemTargetX360UnregNotif *syscall.LazyProc
	vigemTargetDS4RegNotif    *syscall.LazyProc
	vigemTargetDS4UnregNotif  *syscall.LazyProc
	vigemDisconnect           *syscall.LazyProc
	vigemFree                 *syscall.LazyProc
}

type DualGamepadSlot struct {
	driver     *ViGEmDriver
	x360Target uintptr
	ds4Target  uintptr
	x360Report XusbReport
	ds4Report  Ds4Report
	onRumble   func(largeMotor, smallMotor uint8)
}

var (
	slotRegistryMu sync.RWMutex
	slotRegistry   = make(map[uintptr]*DualGamepadSlot)

	x360RumbleCallback uintptr
	ds4RumbleCallback  uintptr
)

func initRumbleCallbacks() {
	if x360RumbleCallback == 0 {
		x360RumbleCallback = syscall.NewCallback(func(client, target, largeMotor, smallMotor, ledNumber, userData uintptr) uintptr {
			slotRegistryMu.RLock()
			slot, ok := slotRegistry[target]
			slotRegistryMu.RUnlock()
			if ok && slot != nil && slot.onRumble != nil {
				slot.onRumble(uint8(largeMotor), uint8(smallMotor))
			}
			return 0
		})
	}
	if ds4RumbleCallback == 0 {
		ds4RumbleCallback = syscall.NewCallback(func(client, target, largeMotor, smallMotor, lightbarColor, userData uintptr) uintptr {
			slotRegistryMu.RLock()
			slot, ok := slotRegistry[target]
			slotRegistryMu.RUnlock()
			if ok && slot != nil && slot.onRumble != nil {
				slot.onRumble(uint8(largeMotor), uint8(smallMotor))
			}
			return 0
		})
	}
}

func NewViGEmDriver(dllPath string) (*ViGEmDriver, error) {
	absDll, err := filepath.Abs(dllPath)
	if err != nil {
		return nil, err
	}

	dll := syscall.NewLazyDLL(absDll)
	allocProc := dll.NewProc("vigem_alloc")
	connectProc := dll.NewProc("vigem_connect")

	client, _, _ := allocProc.Call()
	if client == 0 {
		return nil, fmt.Errorf("vigem_alloc returned null handle")
	}

	ret, _, _ := connectProc.Call(client)
	if ret != VIGEM_ERROR_NONE {
		return nil, fmt.Errorf("vigem_connect failed with error: 0x%X", ret)
	}

	return &ViGEmDriver{
		client:                    client,
		vigemTargetX360Alloc:      dll.NewProc("vigem_target_x360_alloc"),
		vigemTargetDS4Alloc:       dll.NewProc("vigem_target_ds4_alloc"),
		vigemTargetAdd:            dll.NewProc("vigem_target_add"),
		vigemTargetRemove:         dll.NewProc("vigem_target_remove"),
		vigemTargetFree:           dll.NewProc("vigem_target_free"),
		vigemTargetX360Upd:        dll.NewProc("vigem_target_x360_update"),
		vigemTargetDS4Upd:         dll.NewProc("vigem_target_ds4_update"),
		vigemTargetX360RegNotif:   dll.NewProc("vigem_target_x360_register_notification"),
		vigemTargetX360UnregNotif: dll.NewProc("vigem_target_x360_unregister_notification"),
		vigemTargetDS4RegNotif:    dll.NewProc("vigem_target_ds4_register_notification"),
		vigemTargetDS4UnregNotif:  dll.NewProc("vigem_target_ds4_unregister_notification"),
		vigemDisconnect:           dll.NewProc("vigem_disconnect"),
		vigemFree:                 dll.NewProc("vigem_free"),
	}, nil
}

// CreateDualGamepad creates both Xbox 360 and DS4 for maximum compatibility
func (d *ViGEmDriver) CreateDualGamepad() (*DualGamepadSlot, error) {
	initRumbleCallbacks()

	x360, _, _ := d.vigemTargetX360Alloc.Call()
	if x360 == 0 {
		return nil, fmt.Errorf("vigem_target_x360_alloc failed")
	}

	ret, _, _ := d.vigemTargetAdd.Call(d.client, x360)
	if ret != VIGEM_ERROR_NONE {
		d.vigemTargetFree.Call(x360)
		return nil, fmt.Errorf("failed to plug in Xbox 360 target: 0x%X", ret)
	}

	ds4, _, _ := d.vigemTargetDS4Alloc.Call()
	if ds4 != 0 {
		retDs4, _, _ := d.vigemTargetAdd.Call(d.client, ds4)
		if retDs4 != VIGEM_ERROR_NONE {
			d.vigemTargetFree.Call(ds4)
			ds4 = 0
		}
	}

	slot := &DualGamepadSlot{
		driver:     d,
		x360Target: x360,
		ds4Target:  ds4,
	}

	slotRegistryMu.Lock()
	slotRegistry[x360] = slot
	if ds4 != 0 {
		slotRegistry[ds4] = slot
	}
	slotRegistryMu.Unlock()

	// Register vibration/rumble notifications with ViGEmBus
	d.vigemTargetX360RegNotif.Call(d.client, x360, x360RumbleCallback, 0)
	if ds4 != 0 {
		d.vigemTargetDS4RegNotif.Call(d.client, ds4, ds4RumbleCallback, 0)
	}

	// Initialize DS4 D-Pad to Neutral
	slot.ds4Report.WButtons = (slot.ds4Report.WButtons &^ 0xF) | DS4_BUTTON_DPAD_NONE
	slot.ds4Report.BThumbLX = 128
	slot.ds4Report.BThumbLY = 128
	slot.ds4Report.BThumbRX = 128
	slot.ds4Report.BThumbRY = 128

	// Pulse button once so Windows & Chrome register the gamepad immediately
	slot.x360Report.WButtons = XUSB_GAMEPAD_A
	slot.Update()
	slot.x360Report.WButtons = 0
	slot.Update()

	return slot, nil
}

// UpdateState processes analog axes (normalized -1.0 to 1.0), triggers (0..255), and 32-bit button mask
func (slot *DualGamepadSlot) UpdateState(lx, ly, rx, ry float32, l2, r2 uint8, btnMask uint32) error {
	// Clamp analog stick values
	if lx < -1.0 {
		lx = -1.0
	} else if lx > 1.0 {
		lx = 1.0
	}
	if ly < -1.0 {
		ly = -1.0
	} else if ly > 1.0 {
		ly = 1.0
	}
	if rx < -1.0 {
		rx = -1.0
	} else if rx > 1.0 {
		rx = 1.0
	}
	if ry < -1.0 {
		ry = -1.0
	} else if ry > 1.0 {
		ry = 1.0
	}

	// 1. UPDATE XBOX 360 REPORT
	slot.x360Report.SThumbLX = int16(lx * 32767.0)
	slot.x360Report.SThumbLY = int16(-ly * 32767.0) // Invert Y for Windows standard
	slot.x360Report.SThumbRX = int16(rx * 32767.0)
	slot.x360Report.SThumbRY = int16(-ry * 32767.0)
	slot.x360Report.BLeftTrigger = l2
	slot.x360Report.BRightTrigger = r2

	var xBtns uint16 = 0
	if btnMask&(1<<0) != 0 {
		xBtns |= XUSB_GAMEPAD_A
	}
	if btnMask&(1<<1) != 0 {
		xBtns |= XUSB_GAMEPAD_B
	}
	if btnMask&(1<<2) != 0 {
		xBtns |= XUSB_GAMEPAD_X
	}
	if btnMask&(1<<3) != 0 {
		xBtns |= XUSB_GAMEPAD_Y
	}
	if btnMask&(1<<4) != 0 {
		xBtns |= XUSB_GAMEPAD_LEFT_SHOULDER
	}
	if btnMask&(1<<5) != 0 {
		xBtns |= XUSB_GAMEPAD_RIGHT_SHOULDER
	}
	if btnMask&(1<<6) != 0 {
		xBtns |= XUSB_GAMEPAD_BACK
	}
	if btnMask&(1<<7) != 0 {
		xBtns |= XUSB_GAMEPAD_START
	}
	if btnMask&(1<<8) != 0 {
		xBtns |= XUSB_GAMEPAD_LEFT_THUMB
	}
	if btnMask&(1<<9) != 0 {
		xBtns |= XUSB_GAMEPAD_RIGHT_THUMB
	}
	if btnMask&(1<<10) != 0 {
		xBtns |= XUSB_GAMEPAD_GUIDE
	}
	if btnMask&(1<<12) != 0 {
		xBtns |= XUSB_GAMEPAD_DPAD_UP
	}
	if btnMask&(1<<13) != 0 {
		xBtns |= XUSB_GAMEPAD_DPAD_DOWN
	}
	if btnMask&(1<<14) != 0 {
		xBtns |= XUSB_GAMEPAD_DPAD_LEFT
	}
	if btnMask&(1<<15) != 0 {
		xBtns |= XUSB_GAMEPAD_DPAD_RIGHT
	}
	// Pro Paddles M1 & M2 (mapped to L3 / R3)
	if btnMask&(1<<16) != 0 {
		xBtns |= XUSB_GAMEPAD_LEFT_THUMB
	}
	if btnMask&(1<<17) != 0 {
		xBtns |= XUSB_GAMEPAD_RIGHT_THUMB
	}
	slot.x360Report.WButtons = xBtns

	// 2. UPDATE DUALSHOCK 4 REPORT (if available)
	if slot.ds4Target != 0 {
		slot.ds4Report.BThumbLX = uint8((lx * 127.0) + 128.0)
		slot.ds4Report.BThumbLY = uint8((ly * 127.0) + 128.0)
		slot.ds4Report.BThumbRX = uint8((rx * 127.0) + 128.0)
		slot.ds4Report.BThumbRY = uint8((ry * 127.0) + 128.0)
		slot.ds4Report.BTriggerL = l2
		slot.ds4Report.BTriggerR = r2

		var ds4Btns uint16 = 0
		if btnMask&(1<<0) != 0 {
			ds4Btns |= DS4_BUTTON_CROSS
		}
		if btnMask&(1<<1) != 0 {
			ds4Btns |= DS4_BUTTON_CIRCLE
		}
		if btnMask&(1<<2) != 0 {
			ds4Btns |= DS4_BUTTON_SQUARE
		}
		if btnMask&(1<<3) != 0 {
			ds4Btns |= DS4_BUTTON_TRIANGLE
		}
		if btnMask&(1<<4) != 0 {
			ds4Btns |= DS4_BUTTON_SHOULDER_LEFT
		}
		if btnMask&(1<<5) != 0 {
			ds4Btns |= DS4_BUTTON_SHOULDER_RIGHT
		}
		if l2 > 60 {
			ds4Btns |= DS4_BUTTON_TRIGGER_LEFT
		}
		if r2 > 60 {
			ds4Btns |= DS4_BUTTON_TRIGGER_RIGHT
		}
		if btnMask&(1<<6) != 0 {
			ds4Btns |= DS4_BUTTON_SHARE
		}
		if btnMask&(1<<7) != 0 {
			ds4Btns |= DS4_BUTTON_OPTIONS
		}
		if btnMask&(1<<8) != 0 || btnMask&(1<<16) != 0 {
			ds4Btns |= DS4_BUTTON_THUMB_LEFT
		}
		if btnMask&(1<<9) != 0 || btnMask&(1<<17) != 0 {
			ds4Btns |= DS4_BUTTON_THUMB_RIGHT
		}

		// 8-Way D-Pad
		up := btnMask&(1<<12) != 0
		down := btnMask&(1<<13) != 0
		left := btnMask&(1<<14) != 0
		right := btnMask&(1<<15) != 0

		var dpad uint16 = DS4_BUTTON_DPAD_NONE
		if up && left {
			dpad = DS4_BUTTON_DPAD_NORTHWEST
		} else if up && right {
			dpad = DS4_BUTTON_DPAD_NORTHEAST
		} else if down && left {
			dpad = DS4_BUTTON_DPAD_SOUTHWEST
		} else if down && right {
			dpad = DS4_BUTTON_DPAD_SOUTHEAST
		} else if up {
			dpad = DS4_BUTTON_DPAD_NORTH
		} else if down {
			dpad = DS4_BUTTON_DPAD_SOUTH
		} else if left {
			dpad = DS4_BUTTON_DPAD_WEST
		} else if right {
			dpad = DS4_BUTTON_DPAD_EAST
		}

		ds4Btns = (ds4Btns &^ 0xF) | dpad
		slot.ds4Report.WButtons = ds4Btns

		var special uint8 = 0
		if btnMask&(1<<10) != 0 {
			special |= DS4_SPECIAL_BUTTON_PS
		}
		if btnMask&(1<<11) != 0 {
			special |= DS4_SPECIAL_BUTTON_TOUCHPAD
		}
		slot.ds4Report.BSpecial = special
	}

	return slot.Update()
}

func (slot *DualGamepadSlot) Update() error {
	if slot.x360Target != 0 {
		slot.driver.vigemTargetX360Upd.Call(
			slot.driver.client,
			slot.x360Target,
			uintptr(unsafe.Pointer(&slot.x360Report)),
		)
	}
	if slot.ds4Target != 0 {
		slot.driver.vigemTargetDS4Upd.Call(
			slot.driver.client,
			slot.ds4Target,
			uintptr(unsafe.Pointer(&slot.ds4Report)),
		)
	}
	return nil
}

func (slot *DualGamepadSlot) SetRumbleCallback(cb func(largeMotor, smallMotor uint8)) {
	slot.onRumble = cb
}

func (slot *DualGamepadSlot) Close() {
	if slot.x360Target != 0 {
		slot.driver.vigemTargetX360UnregNotif.Call(slot.x360Target)
		slot.driver.vigemTargetRemove.Call(slot.driver.client, slot.x360Target)
		slot.driver.vigemTargetFree.Call(slot.x360Target)
		slotRegistryMu.Lock()
		delete(slotRegistry, slot.x360Target)
		slotRegistryMu.Unlock()
		slot.x360Target = 0
	}
	if slot.ds4Target != 0 {
		slot.driver.vigemTargetDS4UnregNotif.Call(slot.ds4Target)
		slot.driver.vigemTargetRemove.Call(slot.driver.client, slot.ds4Target)
		slot.driver.vigemTargetFree.Call(slot.ds4Target)
		slotRegistryMu.Lock()
		delete(slotRegistry, slot.ds4Target)
		slotRegistryMu.Unlock()
		slot.ds4Target = 0
	}
}

func (d *ViGEmDriver) Close() {
	if d.client != 0 {
		d.vigemDisconnect.Call(d.client)
		d.vigemFree.Call(d.client)
		d.client = 0
	}
}
