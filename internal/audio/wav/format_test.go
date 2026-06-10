package wav

import "testing"

func TestDetectMetaAssets(t *testing.T) {
	for _, path := range []string{"../../../assets/musica.wav", "../../../assets/musica2.wav"} {
		meta, offset, pcmSize, err := DetectMeta(path)
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
