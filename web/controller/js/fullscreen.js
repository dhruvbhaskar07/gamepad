// ==========================================
// ROBUST CROSS-BROWSER FULLSCREEN & ORIENTATION
// ==========================================
let isTogglingFullscreen = false;
let lastFullscreenToggleTime = 0;

export function isFullscreen() {
    return !!(
        document.fullscreenElement ||
        document.webkitFullscreenElement ||
        document.webkitCurrentFullScreenElement ||
        document.mozFullScreenElement ||
        document.msFullscreenElement
    );
}

export function updateFullscreenUI(btnEl) {
    if (!btnEl) return;
    if (isFullscreen()) {
        btnEl.innerText = '🗗';
        btnEl.title = 'Exit Fullscreen';
        btnEl.classList.add('active');
    } else {
        btnEl.innerText = '⛶';
        btnEl.title = 'Enter Fullscreen';
        btnEl.classList.remove('active');
    }
}

export async function toggleFullscreen(btnEl = null, onToast = null) {
    const now = Date.now();
    if (now - lastFullscreenToggleTime < 450 || isTogglingFullscreen) return;
    lastFullscreenToggleTime = now;
    isTogglingFullscreen = true;

    const isIOS = /iPad|iPhone|iPod/.test(navigator.userAgent) && !window.MSStream;

    if (isFullscreen()) {
        try {
            if (document.exitFullscreen) {
                await document.exitFullscreen();
            } else if (document.webkitExitFullscreen) {
                await document.webkitExitFullscreen();
            } else if (document.webkitCancelFullScreen) {
                await document.webkitCancelFullScreen();
            } else if (document.mozCancelFullScreen) {
                await document.mozCancelFullScreen();
            } else if (document.msExitFullscreen) {
                await document.msExitFullscreen();
            }
            if (screen.orientation && screen.orientation.unlock) {
                screen.orientation.unlock();
            }
            if (onToast) onToast('🗗 Exited Fullscreen');
        } catch (err) {
            console.warn('Exit fullscreen error:', err);
        } finally {
            updateFullscreenUI(btnEl);
            setTimeout(() => { isTogglingFullscreen = false; }, 300);
        }
        return;
    }

    const docEl = document.documentElement;
    const requestFS = docEl.requestFullscreen ||
                      docEl.webkitRequestFullscreen ||
                      docEl.webkitRequestFullScreen ||
                      docEl.mozRequestFullScreen ||
                      docEl.msRequestFullscreen;

    if (!requestFS) {
        if (isIOS) {
            if (onToast) onToast('📱 iPhone Tip: Safari Share (📤) > "Add to Home Screen"');
            alert('📱 iPhone / iPad Safari Fullscreen:\n\nApple ne iPhone Safari browser me standard Fullscreen API disable kiya hai.\n\n100% Fullscreen ke liye:\n1. Safari me Share icon (📤) dabayein.\n2. "Add to Home Screen" (होम स्क्रीन में जोड़ें) chunein.\n3. Phir Home Screen se open karein — ye bina kisi address bar ke Fullscreen chalega!');
        } else {
            if (onToast) onToast('⚠️ Fullscreen not supported by this browser');
        }
        isTogglingFullscreen = false;
        return;
    }

    try {
        const p = docEl.requestFullscreen ? docEl.requestFullscreen() : requestFS.call(docEl);
        if (p && p.then) {
            await p;
        }

        if (screen.orientation && screen.orientation.lock) {
            screen.orientation.lock('landscape').catch(() => {});
        }

        if (onToast) onToast('⛶ Fullscreen Active');
    } catch (err) {
        console.error('Fullscreen request failed:', err);
        if (isIOS) {
            if (onToast) onToast('📱 iPhone: Safari Share (📤) > "Add to Home Screen"');
        } else {
            if (onToast) onToast('⚠️ Fullscreen block ho gaya. Tap again.');
        }
    } finally {
        updateFullscreenUI(btnEl);
        setTimeout(() => { isTogglingFullscreen = false; }, 300);
    }
}

export function initFullscreenListeners(btnEl, onToast = null) {
    if (!btnEl) return;

    ['fullscreenchange', 'webkitfullscreenchange', 'mozfullscreenchange', 'MSFullscreenChange'].forEach(ev => {
        document.addEventListener(ev, () => updateFullscreenUI(btnEl));
    });

    const handleFSToggle = (e) => {
        if (e.cancelable) e.preventDefault();
        toggleFullscreen(btnEl, onToast);
    };

    btnEl.addEventListener('pointerup', handleFSToggle);
    btnEl.addEventListener('click', handleFSToggle);
}
