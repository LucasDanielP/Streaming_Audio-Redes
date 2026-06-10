package source

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/gen2brain/malgo"

	"streaming-audio-redes/internal/audio/capture"
	"streaming-audio-redes/internal/protocol"
)

// Live captura áudio ao vivo do microfone e expõe como PCM 16-bit.
type Live struct {
	cfg  capture.LiveConfig
	meta protocol.AudioMeta

	mu       sync.Mutex
	chunkCh  chan []byte
	stopOnce sync.Once
	stopCh   chan struct{}
}

// NewLive prepara captura ao vivo a partir do microfone padrão ou nomeado.
func NewLive(cfg capture.LiveConfig) (*Live, error) {
	if cfg.SampleRate == 0 {
		cfg.SampleRate = 44100
	}
	if cfg.Channels == 0 {
		cfg.Channels = 1
	}

	sourceLabel := "microfone (padrão)"
	if cfg.DeviceName != "" {
		sourceLabel = cfg.DeviceName
	}

	return &Live{
		cfg: cfg,
		meta: protocol.AudioMeta{
			Codec:         "pcm",
			Container:     "live",
			SampleRate:    cfg.SampleRate,
			Channels:      cfg.Channels,
			BitsPerSample: 16,
			Source:        sourceLabel,
		},
		chunkCh: make(chan []byte, 32),
		stopCh:  make(chan struct{}),
	}, nil
}

func (l *Live) Meta() protocol.AudioMeta {
	return l.meta
}

func (l *Live) Close() error {
	l.stopOnce.Do(func() { close(l.stopCh) })
	return nil
}

// ListCaptureDevices retorna os dispositivos de entrada disponíveis.
func ListCaptureDevices() ([]string, error) {
	return capture.ListDevices()
}

func (l *Live) Stream(ctx context.Context, emit func([]byte) error) error {
	audioCtx, err := malgo.InitContext(nil, malgo.ContextConfig{}, func(message string) {
		log.Printf("[servidor/captura] %s", message)
	})
	if err != nil {
		return fmt.Errorf("inicializar áudio: %w", err)
	}
	defer func() {
		_ = audioCtx.Uninit()
		audioCtx.Free()
	}()

	deviceConfig := malgo.DefaultDeviceConfig(malgo.Capture)
	deviceConfig.Capture.Format = malgo.FormatS16
	deviceConfig.Capture.Channels = uint32(l.cfg.Channels)
	deviceConfig.SampleRate = l.cfg.SampleRate
	deviceConfig.Alsa.NoMMap = 1

	if l.cfg.DeviceName != "" {
		id, name, err := capture.FindDevice(audioCtx, l.cfg.DeviceName)
		if err != nil {
			return err
		}
		deviceConfig.Capture.DeviceID = id.Pointer()
		l.mu.Lock()
		l.meta.Source = name
		l.mu.Unlock()
		log.Printf("[servidor] captura em: %s", name)
	} else {
		log.Printf("[servidor] captura no microfone padrão do sistema")
	}

	bytesPerFrame := uint32(l.cfg.Channels) * uint32(malgo.SampleSizeInBytes(deviceConfig.Capture.Format))
	var pending []byte

	callbacks := malgo.DeviceCallbacks{
		Data: func(_, input []byte, frameCount uint32) {
			if len(input) == 0 {
				return
			}
			sampleBytes := frameCount * bytesPerFrame
			if sampleBytes > uint32(len(input)) {
				sampleBytes = uint32(len(input))
			}

			chunk := make([]byte, sampleBytes)
			copy(chunk, input[:sampleBytes])

			l.mu.Lock()
			pending = append(pending, chunk...)
			for len(pending) >= protocol.ChunkSize {
				chunk := make([]byte, protocol.ChunkSize)
				copy(chunk, pending[:protocol.ChunkSize])
				pending = pending[protocol.ChunkSize:]
				select {
				case l.chunkCh <- chunk:
				case <-l.stopCh:
					l.mu.Unlock()
					return
				default:
					select {
					case <-l.chunkCh:
					default:
					}
					l.chunkCh <- chunk
				}
			}
			l.mu.Unlock()
		},
	}

	device, err := malgo.InitDevice(audioCtx.Context, deviceConfig, callbacks)
	if err != nil {
		return fmt.Errorf("abrir dispositivo de captura: %w", err)
	}
	defer device.Uninit()

	if err := device.Start(); err != nil {
		return fmt.Errorf("iniciar captura: %w", err)
	}
	defer func() {
		_ = device.Stop()
	}()

	log.Printf("[servidor] captura ao vivo iniciada | %dHz %dch 16-bit",
		l.cfg.SampleRate, l.cfg.Channels)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-l.stopCh:
			return nil
		case chunk := <-l.chunkCh:
			if err := emit(chunk); err != nil {
				return err
			}
		}
	}
}
