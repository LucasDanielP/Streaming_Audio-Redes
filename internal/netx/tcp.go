package netx

import "net"

// EnableLowLatency desativa o algoritmo de Nagle (TCP_NODELAY) para reduzir atraso no streaming.
func EnableLowLatency(conn net.Conn) {
	if tc, ok := conn.(*net.TCPConn); ok {
		_ = tc.SetNoDelay(true)
	}
}
