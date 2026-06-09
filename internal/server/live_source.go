package server

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/gen2brain/malgo"

	"streaming-audio-redes/internal/protocol"
)

// LiveConfig configura a captura do microfone.
type LiveConfig struct {
	DeviceName string // substring do nome; vazio = dispositivo padrão do sistema
	SampleRate uint32
	Channels   uint16
}

// LiveSource captura áudio ao vivo do microfone e expõe como PCM 16-bit.
type LiveSource struct {
	cfg  LiveConfig
	meta protocol.AudioMeta

	mu       sync.Mutex
	chunkCh  chan []byte
	stopOnce sync.Once
	stopCh   chan struct{}
}

// NewLiveSource prepara captura ao vivo a partir do microfone padrão ou nomeado.
func NewLiveSource(cfg LiveConfig) (*LiveSource, error) {
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

	return &LiveSource{
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

func (l *LiveSource) Meta() protocol.AudioMeta {
	return l.meta
}

func (l *LiveSource) Close() error {
	l.stopOnce.Do(func() { close(l.stopCh) })
	return nil
}

// ListCaptureDevices retorna os dispositivos de entrada disponíveis.
func ListCaptureDevices() ([]string, error) {
	ctx, err := malgo.InitContext(nil, malgo.ContextConfig{}, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = ctx.Uninit()
		ctx.Free()
	}()

	devices, err := ctx.Devices(malgo.Capture)
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(devices))
	for _, d := range devices {
		label := d.Name()
		if d.IsDefault != 0 {
			label += " (padrão)"
		}
		names = append(names, label)
	}
	return names, nil
}

func (l *LiveSource) Stream(ctx context.Context, emit func([]byte) error) error {
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
		id, name, err := findCaptureDevice(audioCtx, l.cfg.DeviceName)
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
					// Clientes lentos: descarta chunk antigo e mantém o fluxo ao vivo.
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

func findCaptureDevice(ctx *malgo.AllocatedContext, nameQuery string) (malgo.DeviceID, string, error) {
	devices, err := ctx.Devices(malgo.Capture)
	if err != nil {
		return malgo.DeviceID{}, "", err
	}

	query := strings.ToLower(strings.TrimSpace(nameQuery))
	for _, d := range devices {
		if strings.Contains(strings.ToLower(d.Name()), query) {
			return d.ID, d.Name(), nil
		}
	}
	return malgo.DeviceID{}, "", fmt.Errorf("dispositivo de entrada não encontrado: %q", nameQuery)
}
