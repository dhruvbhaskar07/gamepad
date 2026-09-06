package main

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type wsWriter struct {
	conn      net.Conn
	mu        sync.Mutex
	isMonitor bool
}

// broadcastRumble sends rumble feedback packets over WebSocket
func (app *ServerApp) broadcastRumble(slotNum int, largeMotor, smallMotor uint8) {
	heavyPct := uint8(float64(largeMotor) * 100.0 / 255.0)
	lightPct := uint8(float64(smallMotor) * 100.0 / 255.0)

	app.mu.Lock()
	switch slotNum {
	case 1:
		app.p1RumbleHeavy = heavyPct
		app.p1RumbleLight = lightPct
	case 2:
		app.p2RumbleHeavy = heavyPct
		app.p2RumbleLight = lightPct
	}
	app.mu.Unlock()

	frame := []byte{
		0x82, // FIN + Binary Frame
		0x04, // 4 bytes payload
		0xFE, // Magic Rumble Byte
		largeMotor,
		smallMotor,
		byte(slotNum),
	}

	app.wsClientsMu.Lock()
	defer app.wsClientsMu.Unlock()

	for w, slot := range app.wsClients {
		if slot == slotNum || slot == 0 || w.isMonitor {
			go func(writer *wsWriter) {
				writer.mu.Lock()
				defer writer.mu.Unlock()
				writer.conn.SetWriteDeadline(time.Now().Add(50 * time.Millisecond))
				writer.conn.Write(frame)
			}(w)
		}
	}
}

// broadcastInputToMonitors sends live gamepad states to dashboard visualizer
func (app *ServerApp) broadcastInputToMonitors(inputFrame []byte) {
	app.wsClientsMu.Lock()
	defer app.wsClientsMu.Unlock()

	frame := make([]byte, 2+len(inputFrame))
	frame[0] = 0x82 // FIN + Binary
	frame[1] = byte(len(inputFrame))
	copy(frame[2:], inputFrame)

	for w := range app.wsClients {
		if w.isMonitor {
			go func(writer *wsWriter) {
				writer.mu.Lock()
				defer writer.mu.Unlock()
				writer.conn.SetWriteDeadline(time.Now().Add(30 * time.Millisecond))
				writer.conn.Write(frame)
			}(w)
		}
	}
}

// handleWebSocket manages full-duplex binary game controller WebSocket connections
func (app *ServerApp) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	if strings.ToLower(r.Header.Get("Upgrade")) != "websocket" {
		http.Error(w, "Expected WebSocket Upgrade", http.StatusBadRequest)
		return
	}

	key := r.Header.Get("Sec-WebSocket-Key")
	if key == "" {
		http.Error(w, "Sec-WebSocket-Key required", http.StatusBadRequest)
		return
	}

	h := sha1.New()
	h.Write([]byte(key + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))
	accept := base64.StdEncoding.EncodeToString(h.Sum(nil))

	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Hijack not supported", http.StatusInternalServerError)
		return
	}

	conn, bufrw, err := hj.Hijack()
	if err != nil {
		return
	}

	ua := r.Header.Get("User-Agent")
	remoteAddr := conn.RemoteAddr().String()
	isLocalDashboard := strings.HasPrefix(remoteAddr, "127.0.0.1") || strings.HasPrefix(remoteAddr, "[::1]")

	writer := &wsWriter{
		conn:      conn,
		isMonitor: isLocalDashboard,
	}

	app.wsClientsMu.Lock()
	app.wsClients[writer] = 1
	app.wsClientsMu.Unlock()

	defer func() {
		app.wsClientsMu.Lock()
		delete(app.wsClients, writer)
		app.wsClientsMu.Unlock()

		app.mu.Lock()
		if app.p1IP == remoteAddr {
			app.p1Connected = false
		}
		if app.p2IP == remoteAddr {
			app.p2Connected = false
		}
		app.mu.Unlock()

		conn.Close()
	}()

	bufrw.WriteString("HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: ")
	bufrw.WriteString(accept)
	bufrw.WriteString("\r\n\r\n")
	bufrw.Flush()

	fmt.Printf("[WS] Client Connected: %s (Monitor: %v) | UA: %s\n", remoteAddr, isLocalDashboard, ua)

	reader := bufrw.Reader
	for {
		b0, err := reader.ReadByte()
		if err != nil {
			break
		}

		opcode := b0 & 0x0F
		if opcode == 0x8 {
			break
		}

		b1, err := reader.ReadByte()
		if err != nil {
			break
		}

		isMasked := (b1 & 0x80) != 0
		payloadLen := int(b1 & 0x7F)

		switch payloadLen {
		case 126:
			var extLen uint16
			binary.Read(reader, binary.BigEndian, &extLen)
			payloadLen = int(extLen)
		case 127:
			var extLen uint64
			binary.Read(reader, binary.BigEndian, &extLen)
			payloadLen = int(extLen)
		}

		var maskKey [4]byte
		if isMasked {
			io.ReadFull(reader, maskKey[:])
		}

		payload := make([]byte, payloadLen)
		_, err = io.ReadFull(reader, payload)
		if err != nil {
			break
		}

		if isMasked {
			for i := 0; i < payloadLen; i++ {
				payload[i] ^= maskKey[i%4]
			}
		}

		// Process Gamepad Binary Input
		if payloadLen >= 8 {
			lx := (float32(payload[0]) - 128.0) / 127.0
			ly := (float32(payload[1]) - 128.0) / 127.0
			rx := (float32(payload[2]) - 128.0) / 127.0
			ry := (float32(payload[3]) - 128.0) / 127.0
			l2 := payload[4]
			r2 := payload[5]

			btnMask := uint32(binary.LittleEndian.Uint16(payload[6:8]))
			if payloadLen >= 10 {
				btnMask |= uint32(binary.LittleEndian.Uint16(payload[8:10])) << 16
			}

			slotNum := 1
			if payloadLen >= 11 {
				slotNum = int(payload[10])
				if slotNum != 2 {
					slotNum = 1
				}
			}

			// Update Telemetry for Dashboard
			app.mu.Lock()
			if slotNum == 1 {
				app.p1Connected = true
				app.p1IP = remoteAddr
				app.p1UA = ua
				app.p1LastSeen = time.Now()
			} else {
				app.p2Connected = true
				app.p2IP = remoteAddr
				app.p2UA = ua
				app.p2LastSeen = time.Now()
			}
			app.mu.Unlock()

			// Broadcast input to Dashboard Visualizer
			app.broadcastInputToMonitors(payload)

			pad := app.getPlayerSlot(slotNum)
			if pad != nil {
				pad.UpdateState(lx, ly, rx, ry, l2, r2, btnMask)
			}
		}
	}

	fmt.Printf("[WS] Client Disconnected: %s\n", remoteAddr)
}
