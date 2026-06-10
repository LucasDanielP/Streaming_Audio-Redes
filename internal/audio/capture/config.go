package capture

// LiveConfig configura captura de microfone (estúdio ou servidor live).
type LiveConfig struct {
	DeviceName string // substring do nome; vazio = padrão do sistema
	SampleRate uint32
	Channels   uint16
}
