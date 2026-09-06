// ==========================================
// LOW-LATENCY WEBSOCKET GAMEPAD PROTOCOL
// ==========================================
export class NetworkClient {
    constructor(options = {}) {
        this.onStatusChange = options.onStatusChange || (() => {});
        this.onRumble = options.onRumble || (() => {});
        this.ws = null;
        this.isConnected = false;
        this.reconnectTimer = null;
    }

    connect() {
        if (this.ws) {
            try { this.ws.close(); } catch (e) {}
        }

        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const host = window.location.host || 'localhost:8080';
        const wsUrl = `${protocol}//${host}/ws`;

        try {
            this.ws = new WebSocket(wsUrl);
            this.ws.binaryType = 'arraybuffer';

            this.ws.onopen = () => {
                this.isConnected = true;
                this.onStatusChange(true);
            };

            this.ws.onclose = () => {
                this.isConnected = false;
                this.onStatusChange(false);
                this.scheduleReconnect();
            };

            this.ws.onerror = () => {
                this.isConnected = false;
                this.onStatusChange(false);
            };

            this.ws.onmessage = (event) => {
                if (event.data instanceof ArrayBuffer) {
                    const view = new DataView(event.data);
                    // Check magic rumble packet: [0xFE, largeMotor, smallMotor, slot]
                    if (view.byteLength >= 4 && view.getUint8(0) === 0xFE) {
                        const largeMotor = view.getUint8(1);
                        const smallMotor = view.getUint8(2);
                        const slot = view.getUint8(3);
                        this.onRumble(largeMotor, smallMotor, slot);
                    }
                }
            };
        } catch (e) {
            this.scheduleReconnect();
        }
    }

    scheduleReconnect() {
        clearTimeout(this.reconnectTimer);
        this.reconnectTimer = setTimeout(() => {
            this.connect();
        }, 1500);
    }

    sendInput(state) {
        if (!this.ws || this.ws.readyState !== WebSocket.OPEN) return;

        // Binary Protocol: 11 bytes
        // [0: LX (0-255), 1: LY (0-255), 2: RX (0-255), 3: RY (0-255), 4: L2 (0-255), 5: R2 (0-255), 6-7: Buttons low16, 8-9: Buttons high16, 10: Slot (1/2)]
        const buf = new ArrayBuffer(11);
        const view = new DataView(buf);

        const clampByte = (val) => Math.min(255, Math.max(0, Math.round(val)));

        view.setUint8(0, clampByte((state.lx * 127.0) + 128.0));
        view.setUint8(1, clampByte((state.ly * 127.0) + 128.0));
        view.setUint8(2, clampByte((state.rx * 127.0) + 128.0));
        view.setUint8(3, clampByte((state.ry * 127.0) + 128.0));
        view.setUint8(4, clampByte(state.l2));
        view.setUint8(5, clampByte(state.r2));

        const btn32 = state.buttons >>> 0;
        view.setUint16(6, btn32 & 0xFFFF, true);
        view.setUint16(8, (btn32 >>> 16) & 0xFFFF, true);
        view.setUint8(10, state.slot === 2 ? 2 : 1);

        try {
            this.ws.send(buf);
        } catch (e) {}
    }
}
