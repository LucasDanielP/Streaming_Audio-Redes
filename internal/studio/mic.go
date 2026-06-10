package studio

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/gen2brain/malgo"

	"streaming-audio-redes/internal/audio/capture"
	"streaming-audio-redes/internal/audio/pcm"
	"streaming-audio-redes/internal/protocol"
)

// MicCapture captura o microfone continuamente em um ring buffer.
type MicCapture struct {
	cfg  capture.LiveConfig
	ring *pcm.Ring

	mu     sync.Mutex
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func NewMicCapture(cfg capture.LiveConfig) *MicCapture {
	bps := int(pcm.BytesPerSecond(protocol.AudioMeta{
		SampleRate: cfg.SampleRate, Channels: cfg.Channels, BitsPerSample: 16,
	}))
	ringCap := bps / 10 // ~100 ms — evita voz atrasada ao ativar o microfone
	if ringCap < 4096 {
		ringCap = 4096
	}
	return &MicCapture{
		cfg:  cfg,
		ring: pcm.NewRing(ringCap),
	}
}

// Start inicia a captura em background.
func (m *MicCapture) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	m.mu.Lock()
	m.cancel = cancel
	m.mu.Unlock()

	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		if err := m.run(ctx); err != nil && ctx.Err() == nil {
			log.Printf("[captura] encerrada: %v", err)
		}
	}()
	return nil
}

func (m *MicCapture) Close() {
	m.mu.Lock()
	cancel := m.cancel
	m.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	m.wg.Wait()
}

func (m *MicCapture) Read(n int) []byte {
	return m.ring.Read(n)
}

// Flush descarta amostras antigas do microfone.
func (m *MicCapture) Flush() {
	m.ring.Clear()
}

func (m *MicCapture) run(ctx context.Context) error {
	audioCtx, err := malgo.InitContext(nil, malgo.ContextConfig{}, func(message string) {
		log.Printf("[captura] %s", message)
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
	deviceConfig.Capture.Channels = uint32(m.cfg.Channels)
	deviceConfig.SampleRate = m.cfg.SampleRate
	deviceConfig.PeriodSizeInFrames = 512
	deviceConfig.Periods = 3
	deviceConfig.Alsa.NoMMap = 1

	if m.cfg.DeviceName != "" {
		id, name, err := capture.FindDevice(audioCtx, m.cfg.DeviceName)
		if err != nil {
			return err
		}
		deviceConfig.Capture.DeviceID = id.Pointer()
		log.Printf("[captura] dispositivo: %s", name)
	}

	bytesPerFrame := uint32(m.cfg.Channels) * uint32(malgo.SampleSizeInBytes(deviceConfig.Capture.Format))

	callbacks := malgo.DeviceCallbacks{
		Data: func(_, input []byte, frameCount uint32) {
			if len(input) == 0 {
				return
			}
			sampleBytes := int(frameCount * bytesPerFrame)
			if sampleBytes > len(input) {
				sampleBytes = len(input)
			}
			chunk := make([]byte, sampleBytes)
			copy(chunk, input[:sampleBytes])
			m.ring.Write(chunk)
		},
	}

	device, err := malgo.InitDevice(audioCtx.Context, deviceConfig, callbacks)
	if err != nil {
		return fmt.Errorf("abrir microfone: %w", err)
	}
	defer device.Uninit()

	if err := device.Start(); err != nil {
		return fmt.Errorf("iniciar microfone: %w", err)
	}
	defer func() { _ = device.Stop() }()

	log.Printf("[captura] microfone ativo | %dHz %dch 16-bit", m.cfg.SampleRate, m.cfg.Channels)
	<-ctx.Done()
	return ctx.Err()
}
