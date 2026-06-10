package studio

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"sync"

	"streaming-audio-redes/internal/audio/capture"
	"streaming-audio-redes/internal/audio/pace"
	"streaming-audio-redes/internal/audio/pcm"
	"streaming-audio-redes/internal/audio/wav"
	"streaming-audio-redes/internal/protocol"
)

// Studio é a mesa do locutor com mixagem de música + microfone.
type Studio struct {
	liveCfg capture.LiveConfig

	mu sync.Mutex

	playlist     []string
	musicPlaying bool

	micEnabled  bool
	micVolume   float32
	musicVolume float32

	music *MusicPlayer

	onMetaChange func(protocol.AudioMeta)
}

// Config configura o estúdio.
type Config struct {
	Live capture.LiveConfig
}

// NewStudio cria a mesa do locutor.
func New(cfg Config) *Studio {
	if cfg.Live.SampleRate == 0 {
		cfg.Live.SampleRate = 44100
	}
	if cfg.Live.Channels == 0 {
		cfg.Live.Channels = 2
	}
	return &Studio{
		liveCfg:     cfg.Live,
		playlist:    make([]string, 0),
		micEnabled:  false,
		micVolume:   1.0,
		musicVolume: 0.75,
		music:       NewMusicPlayer(),
	}
}

func (s *Studio) SetMetaChangeHandler(fn func(protocol.AudioMeta)) {
	s.mu.Lock()
	s.onMetaChange = fn
	s.mu.Unlock()
}

func (s *Studio) Meta() protocol.AudioMeta {
	return s.outputMeta()
}

func (s *Studio) Close() error {
	s.music.Close()
	return nil
}

func (s *Studio) Stream(ctx context.Context, emit func([]byte) error) error {
	mic := NewMicCapture(s.liveCfg)
	if err := mic.Start(ctx); err != nil {
		return err
	}
	defer mic.Close()

	outMeta := s.outputMeta()
	s.notifyMeta(outMeta)

	clock := pace.New(outMeta)
	log.Printf("[estúdio] mixer | %dHz %dch 16-bit PCM — pacing por relógio",
		outMeta.SampleRate, outMeta.Channels)

	chunkBytes := pcm.AlignedChunkSize(outMeta.Channels)

	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		s.mu.Lock()
		micOn := s.micEnabled
		micVol := s.micVolume
		musicVol := s.musicVolume
		playing := s.musicPlaying
		s.mu.Unlock()

		var musicPCM []byte
		if playing && s.music.IsOpen() {
			musicPCM = s.music.Read(chunkBytes)
			if !s.music.IsOpen() {
				s.mu.Lock()
				s.musicPlaying = false
				s.mu.Unlock()
				log.Printf("[estúdio] faixa encerrada")
			}
		} else {
			musicPCM = pcm.Silence(chunkBytes)
		}

		var micPCM []byte
		if micOn {
			micPCM = mic.Read(chunkBytes)
		} else {
			mic.Flush()
			micPCM = pcm.Silence(chunkBytes)
		}

		mixed := pcm.Mix16(musicPCM, micPCM, musicVol, micVol)

		curMeta := s.outputMeta()
		if !pace.SameFormat(curMeta, clock.Meta()) {
			clock.Reset(curMeta)
		}

		if err := emit(mixed); err != nil {
			return err
		}
		if err := clock.Wait(ctx, len(mixed)); err != nil {
			return err
		}
	}
}

func (s *Studio) outputMeta() protocol.AudioMeta {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.musicPlaying && s.music.CurrentPath() != "" {
		m := s.music.Meta()
		m.Container = "studio"
		m.Source = s.onAirLabelLocked()
		return m
	}

	return s.broadcastMetaLocked()
}

func (s *Studio) SetMicEnabled(on bool) {
	s.mu.Lock()
	was := s.micEnabled
	s.micEnabled = on
	s.mu.Unlock()
	if on && !was {
		s.notifyMeta(s.outputMeta())
	}
}

func (s *Studio) SetMicVolume(vol float32) {
	s.mu.Lock()
	s.micVolume = clampVolume(vol)
	s.mu.Unlock()
}

func (s *Studio) SetMusicVolume(vol float32) {
	s.mu.Lock()
	s.musicVolume = clampVolume(vol)
	s.mu.Unlock()
}

func (s *Studio) SeekMusicForward(seconds float64) {
	s.music.SeekForward(seconds)
}

// SeekMusicTo posiciona a faixa em um instante (segundos).
func (s *Studio) SeekMusicTo(seconds float64) {
	if !s.musicPlaying {
		return
	}
	s.music.SeekTo(seconds)
}

func (s *Studio) PlayTrack(path string) error {
	fileMeta, _, _, err := wav.DetectMeta(path)
	if err != nil {
		return err
	}
	if err := s.validateTrackMeta(fileMeta, path); err != nil {
		return err
	}

	s.mu.Lock()
	s.liveCfg.SampleRate = fileMeta.SampleRate
	s.liveCfg.Channels = fileMeta.Channels
	s.mu.Unlock()

	if err := s.music.Open(path); err != nil {
		return err
	}

	s.mu.Lock()
	s.musicPlaying = true
	s.mu.Unlock()

	meta := s.outputMeta()
	s.notifyMeta(meta)
	log.Printf("[estúdio] tocando: %s | %dHz %dch", filepath.Base(path), meta.SampleRate, meta.Channels)
	return nil
}

func (s *Studio) StopMusic() {
	s.music.Close()
	s.mu.Lock()
	s.musicPlaying = false
	s.mu.Unlock()
}

func (s *Studio) StopOnAir() {
	s.StopMusic()
	s.SetMicEnabled(false)
}

func (s *Studio) AddTrack(path string) error {
	meta, _, _, err := wav.DetectMeta(path)
	if err != nil {
		return err
	}
	if err := s.validateTrackMeta(meta, path); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, p := range s.playlist {
		if p == path {
			return fmt.Errorf("faixa já está na playlist: %s", filepath.Base(path))
		}
	}
	s.playlist = append(s.playlist, path)
	return nil
}

func (s *Studio) RemoveTrack(path string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := s.playlist[:0]
	for _, p := range s.playlist {
		if p != path {
			out = append(out, p)
		}
	}
	s.playlist = out
	if s.music.CurrentPath() == path {
		s.musicPlaying = false
		s.music.Close()
	}
}

func (s *Studio) Playlist() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, len(s.playlist))
	copy(out, s.playlist)
	return out
}

func (s *Studio) Snapshot() Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()

	onAir := "Silêncio"
	switch {
	case s.musicPlaying && s.micEnabled:
		onAir = filepath.Base(s.music.CurrentPath()) + " + microfone"
	case s.musicPlaying:
		onAir = filepath.Base(s.music.CurrentPath())
	case s.micEnabled:
		onAir = "Microfone ao vivo"
	}

	playlist := make([]string, len(s.playlist))
	copy(playlist, s.playlist)

	progress := s.music.Progress()

	return Snapshot{
		OnAir:            onAir,
		Playlist:         playlist,
		MusicPath:        s.music.CurrentPath(),
		MicEnabled:       s.micEnabled,
		MusicPlaying:     s.musicPlaying,
		MicVolume:        s.micVolume,
		MusicVolume:      s.musicVolume,
		MusicPositionSec: progress.PositionSec,
		MusicDurationSec: progress.DurationSec,
	}
}

type Snapshot struct {
	OnAir            string
	Playlist         []string
	MusicPath        string
	MicEnabled       bool
	MusicPlaying     bool
	MicVolume        float32
	MusicVolume      float32
	MusicPositionSec float64
	MusicDurationSec float64
}

func (s *Studio) validateTrack(path string) error {
	meta, _, _, err := wav.DetectMeta(path)
	if err != nil {
		return err
	}
	return s.validateTrackMeta(meta, path)
}

func (s *Studio) validateTrackMeta(meta protocol.AudioMeta, path string) error {
	if meta.Codec != "pcm" {
		return fmt.Errorf("%s: apenas WAV PCM é suportado na playlist", filepath.Base(path))
	}
	if meta.BitsPerSample != 16 {
		return fmt.Errorf("%s: apenas PCM 16-bit (arquivo: %dbit)", filepath.Base(path), meta.BitsPerSample)
	}
	if meta.SampleRate != s.liveCfg.SampleRate {
		return fmt.Errorf("%s: taxa deve ser %d Hz (arquivo: %d Hz)",
			filepath.Base(path), s.liveCfg.SampleRate, meta.SampleRate)
	}
	if meta.Channels != s.liveCfg.Channels {
		return fmt.Errorf("%s: canais deve ser %d (arquivo: %d ch). Use -channels %d",
			filepath.Base(path), s.liveCfg.Channels, meta.Channels, meta.Channels)
	}
	return nil
}

func (s *Studio) broadcastMetaLocked() protocol.AudioMeta {
	return protocol.AudioMeta{
		Codec:         "pcm",
		Container:     "studio",
		SampleRate:    s.liveCfg.SampleRate,
		Channels:      s.liveCfg.Channels,
		BitsPerSample: 16,
		Source:        s.onAirLabelLocked(),
	}
}

func (s *Studio) onAirLabelLocked() string {
	switch {
	case s.musicPlaying && s.micEnabled:
		return filepath.Base(s.music.CurrentPath()) + " + microfone"
	case s.musicPlaying:
		return filepath.Base(s.music.CurrentPath())
	case s.micEnabled:
		return "microfone"
	default:
		return "silêncio"
	}
}

func (s *Studio) notifyMeta(meta protocol.AudioMeta) {
	s.mu.Lock()
	fn := s.onMetaChange
	s.mu.Unlock()
	if fn != nil {
		fn(meta)
	}
}

func clampVolume(v float32) float32 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
