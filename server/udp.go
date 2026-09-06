package main

import (
	"encoding/binary"
	"net"
	"strings"
)

// startUDPListener handles ultra-low latency UDP packets from native client apps
func (app *ServerApp) startUDPListener() {
	addr := net.UDPAddr{
		Port: UDP_PORT,
		IP:   net.ParseIP("0.0.0.0"),
	}
	conn, err := net.ListenUDP("udp", &addr)
	if err != nil {
		return
	}
	defer conn.Close()

	buf := make([]byte, 1024)
	for {
		n, remoteAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			continue
		}
		data := buf[:n]
		if strings.HasPrefix(string(data), "VIBE_PING") {
			conn.WriteToUDP([]byte("VIBE_PONG|1|OK"), remoteAddr)
			continue
		}
		if n >= 8 {
			lx := (float32(data[0]) - 128.0) / 127.0
			ly := (float32(data[1]) - 128.0) / 127.0
			rx := (float32(data[2]) - 128.0) / 127.0
			ry := (float32(data[3]) - 128.0) / 127.0
			l2 := data[4]
			r2 := data[5]
			btnMask := uint32(binary.LittleEndian.Uint16(data[6:8]))
			if n >= 10 {
				btnMask |= uint32(binary.LittleEndian.Uint16(data[8:10])) << 16
			}
			pad := app.getPlayerSlot(1)
			if pad != nil {
				pad.UpdateState(lx, ly, rx, ry, l2, r2, btnMask)
			}
		}
	}
}
