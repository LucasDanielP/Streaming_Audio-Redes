package pcm

import "testing"

func TestMix16(t *testing.T) {
	a := []byte{0xFF, 0x7F, 0x00, 0x00}
	b := []byte{0x01, 0x00, 0x00, 0x00}
	out := Mix16(a, b, 0.5, 1.0)
	if len(out) != 4 {
		t.Fatalf("len=%d", len(out))
	}
}
