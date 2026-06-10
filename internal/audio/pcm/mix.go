package pcm

import (
	"encoding/binary"
	"math"
)

// Mix16 combina dois buffers PCM 16-bit little-endian com ganhos independentes.
func Mix16(a, b []byte, volA, volB float32) []byte {
	n := len(a)
	if len(b) > n {
		n = len(b)
	}
	if n%2 == 1 {
		n--
	}
	out := make([]byte, n)
	for i := 0; i < n; i += 2 {
		var sum float32
		if i+1 < len(a) {
			sum += float32(int16(binary.LittleEndian.Uint16(a[i:i+2]))) * volA
		}
		if i+1 < len(b) {
			sum += float32(int16(binary.LittleEndian.Uint16(b[i:i+2]))) * volB
		}
		binary.LittleEndian.PutUint16(out[i:i+2], uint16(clamp16(sum)))
	}
	return out
}

func clamp16(v float32) int16 {
	if v > math.MaxInt16 {
		return math.MaxInt16
	}
	if v < math.MinInt16 {
		return math.MinInt16
	}
	return int16(v)
}

// Silence retorna um bloco PCM zerado.
func Silence(n int) []byte {
	if n%2 == 1 {
		n--
	}
	return make([]byte, n)
}
