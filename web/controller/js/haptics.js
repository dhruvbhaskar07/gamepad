// ==========================================
// ADVANCED DUAL-ENGINE HAPTIC & GAME RUMBLE
// ==========================================
let audioCtx = null;
let rumbleOsc = null;
let rumbleGain = null;
let rumbleInterval = null;
let isRumbleActive = false;
let lastLargeMotor = 0;
let lastSmallMotor = 0;

export function initAudio() {
    if (!audioCtx) {
        const AudioContextClass = window.AudioContext || window.webkitAudioContext;
        if (AudioContextClass) {
            audioCtx = new AudioContextClass();
        }
    }
    if (audioCtx && audioCtx.state === 'suspended') {
        audioCtx.resume().catch(() => {});
    }
}

export function playHapticThump(intensity = 40, freq = 55, duration = 0.04, gainVal = 0.35) {
    if (!intensity || !audioCtx) return;
    try {
        if (audioCtx.state === 'suspended') audioCtx.resume().catch(() => {});
        const osc = audioCtx.createOscillator();
        const gain = audioCtx.createGain();
        const now = audioCtx.currentTime;

        osc.type = 'sine';
        osc.frequency.setValueAtTime(freq, now);
        osc.frequency.exponentialRampToValueAtTime(30, now + duration);

        gain.gain.setValueAtTime(gainVal * (intensity / 25), now);
        gain.gain.exponentialRampToValueAtTime(0.001, now + duration);

        osc.connect(gain);
        gain.connect(audioCtx.destination);

        osc.start(now);
        osc.stop(now + duration);
    } catch (e) {}
}

export function checkVibeSupport(diagEl, hintEl) {
    const isIOS = /iPad|iPhone|iPod/.test(navigator.userAgent) || (navigator.platform === 'MacIntel' && navigator.maxTouchPoints > 1);
    if (isIOS) {
        if (diagEl) {
            diagEl.innerText = '🍎 iOS (Audio Haptic Active)';
            diagEl.style.color = '#ffb703';
        }
        if (hintEl) {
            hintEl.innerHTML = '⚠️ <b>Apple iOS</b>: Apple Safari does not support Web Vibration API. Sub-bass acoustic haptic active hai jo phone speaker coil ko vibrate karta hai.';
        }
    } else if ('vibrate' in navigator) {
        if (diagEl) {
            diagEl.innerText = '🟢 Supported (Hardware Ready)';
            diagEl.style.color = 'var(--accent-emerald)';
        }
        if (hintEl) {
            hintEl.innerHTML = '💡 <b>Agar phone vibrate na kare</b>:<br>1. Phone Settings > <b>Sound & Vibration</b> > <b>Haptic Feedback / Touch Vibration</b> ON karein.<br>2. <b>Battery Saver</b> OFF karein.';
        }
    } else {
        if (diagEl) {
            diagEl.innerText = '❌ Browser Not Supported';
            diagEl.style.color = '#ef4444';
        }
    }
}

export function triggerHaptic(type = 'click', intensity = 40, forceMs = 0) {
    initAudio();
    if (intensity === 0) return;

    let duration = forceMs || (intensity === 40 ? 65 : 40);
    if (type === 'tick') {
        duration = 35;
        playHapticThump(intensity, 120, 0.02, 0.2);
    } else if (type === 'click') {
        duration = 65;
        playHapticThump(intensity, 65, 0.04, 0.4);
    } else if (type === 'heavy') {
        duration = 140;
        playHapticThump(intensity, 45, 0.08, 0.7);
    } else if (typeof type === 'number') {
        duration = type;
        playHapticThump(intensity, 55, Math.min(0.08, duration / 1000), 0.4);
    }

    if ('vibrate' in navigator) {
        try {
            navigator.vibrate(duration);
        } catch (e) {}
    }
}

export function handleGameRumble(largeMotor, smallMotor, intensity = 40, hudPill = null) {
    initAudio();
    lastLargeMotor = largeMotor;
    lastSmallMotor = smallMotor;

    if (intensity === 0 || (largeMotor === 0 && smallMotor === 0)) {
        stopGameRumble(hudPill);
        return;
    }

    document.body.classList.add('rumble-vibrating');
    if (hudPill) hudPill.style.display = 'flex';

    if (!isRumbleActive) {
        isRumbleActive = true;
        runRumbleLoop(intensity);
    }

    if (audioCtx) {
        try {
            if (audioCtx.state === 'suspended') audioCtx.resume().catch(() => {});
            if (!rumbleOsc) {
                rumbleOsc = audioCtx.createOscillator();
                rumbleGain = audioCtx.createGain();
                rumbleOsc.type = 'triangle';
                rumbleOsc.frequency.setValueAtTime(45, audioCtx.currentTime);
                rumbleGain.gain.setValueAtTime(0.01, audioCtx.currentTime);
                rumbleOsc.connect(rumbleGain);
                rumbleGain.connect(audioCtx.destination);
                rumbleOsc.start();
            }

            const freq = largeMotor >= smallMotor ? 38 + (largeMotor * 0.12) : 75 + (smallMotor * 0.25);
            const now = audioCtx.currentTime;
            rumbleOsc.frequency.cancelScheduledValues(now);
            rumbleOsc.frequency.linearRampToValueAtTime(freq, now + 0.05);

            const gain = (Math.max(largeMotor, smallMotor) / 255) * 0.35 * (intensity / 30);
            rumbleGain.gain.cancelScheduledValues(now);
            rumbleGain.gain.linearRampToValueAtTime(gain, now + 0.05);
        } catch (e) {}
    }
}

function runRumbleLoop(intensity) {
    if (!isRumbleActive) return;
    const maxVal = Math.max(lastLargeMotor, lastSmallMotor);
    if (maxVal > 0 && 'vibrate' in navigator) {
        const dur = Math.max(25, Math.round(maxVal / 3));
        try { navigator.vibrate(dur); } catch (e) {}
    }
    rumbleInterval = setTimeout(() => runRumbleLoop(intensity), 75);
}

export function stopGameRumble(hudPill = null) {
    isRumbleActive = false;
    clearTimeout(rumbleInterval);
    document.body.classList.remove('rumble-vibrating');
    if (hudPill) hudPill.style.display = 'none';

    if (audioCtx && rumbleGain) {
        try {
            rumbleGain.gain.linearRampToValueAtTime(0.0001, audioCtx.currentTime + 0.08);
            setTimeout(() => {
                if (!isRumbleActive && rumbleOsc) {
                    try { rumbleOsc.stop(); rumbleOsc.disconnect(); } catch (e) {}
                    rumbleOsc = null;
                    rumbleGain = null;
                }
            }, 90);
        } catch (e) {}
    }
}
