// Gamepad Button Mask Constants (matches XInput & DualSense layout)
export const BTN_UP = 0x0001;
export const BTN_DOWN = 0x0002;
export const BTN_LEFT = 0x0004;
export const BTN_RIGHT = 0x0008;
export const BTN_START = 0x0010;
export const BTN_BACK = 0x0020;
export const BTN_L3 = 0x0040;
export const BTN_R3 = 0x0080;
export const BTN_LB = 0x0100;
export const BTN_RB = 0x0200;
export const BTN_GUIDE = 0x0400;
export const BTN_A = 0x1000;
export const BTN_B = 0x2000;
export const BTN_X = 0x4000;
export const BTN_Y = 0x8000;
export const BTN_TOUCHPAD = 0x10000;
export const BTN_M1 = 0x20000;
export const BTN_M2 = 0x40000;

// Default Controller Configuration
export const DEFAULT_CONFIG = {
    profile: 'ps5', // 'ps5', 'xbox', 'fight'
    slot: 1,
    deadzone: 0.05,
    stickSens: 1.0,
    hapticIntensity: 40, // 0=off, 20=med, 40=strong
    gyroEnabled: false,
    gyroSens: 1.2,
    invertY: false,
    gyroCenterBeta: 0,
    gyroCenterGamma: 0
};
