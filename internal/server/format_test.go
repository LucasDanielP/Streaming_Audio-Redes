package server

import (
	"testing"
)

func TestParseWAVMetaAssets(t *testing.T) {
	for _, path := range []string{"../../assets/musica.wav", "../../assets/musica2.wav"} {
		meta, offset, pcmSize, err := detectAudioMeta(path)
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		if meta.Codec != "pcm" {
			t.Fatalf("%s: codec=%s", path, meta.Codec)
		}
		if meta.SampleRate != 44100 {
			t.Fatalf("%s: sample rate=%d", path, meta.SampleRate)
		}
		if meta.Channels != 2 {
			t.Fatalf("%s: channels=%d", path, meta.Channels)
		}
		if meta.BitsPerSample != 16 {
			t.Fatalf("%s: bits=%d", path, meta.BitsPerSample)
		}
		if offset < 44 {
			t.Fatalf("%s: offset=%d esperado > 44 (chunk LIST)", path, offset)
		}
		if pcmSize == 0 {
			t.Fatalf("%s: pcmSize=0", path)
		}
	}
}

func TestMixPCM16(t *testing.T) {
	a := []byte{0xFF, 0x7F, 0x00, 0x00}
	b := []byte{0x01, 0x00, 0x00, 0x00}
	out := MixPCM16(a, b, 0.5, 1.0)
	if len(out) != 4 {
		t.Fatalf("len=%d", len(out))
	}
}

func TestStudioValidateTrackStereo(t *testing.T) {
	studio := NewStudio(StudioConfig{
		Live: LiveConfig{SampleRate: 44100, Channels: 2},
	})
	if err := studio.AddTrack("../../assets/musica.wav"); err != nil {
		t.Fatal(err)
	}
	if err := studio.AddTrack("../../assets/musica2.wav"); err != nil {
		t.Fatal(err)
	}
	if len(studio.Playlist()) != 2 {
		t.Fatalf("playlist=%d", len(studio.Playlist()))
	}
}
