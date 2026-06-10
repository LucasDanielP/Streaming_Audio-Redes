package studio

import (
	"testing"

	"streaming-audio-redes/internal/audio/capture"
)

func TestValidateTrackStereo(t *testing.T) {
	board := New(Config{
		Live: capture.LiveConfig{SampleRate: 44100, Channels: 2},
	})
	if err := board.AddTrack("../../assets/musica.wav"); err != nil {
		t.Fatal(err)
	}
	if err := board.AddTrack("../../assets/musica2.wav"); err != nil {
		t.Fatal(err)
	}
	if len(board.Playlist()) != 2 {
		t.Fatalf("playlist=%d", len(board.Playlist()))
	}
}
