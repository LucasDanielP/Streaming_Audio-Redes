package protocol

import (
	"encoding/json"
	"fmt"
)

const (
	FrameTypeMeta  byte = 0x01
	FrameTypeAudio byte = 0x02
)

// AudioMeta descreve o formato do fluxo enviado pelo servidor.
type AudioMeta struct {
	Codec         string `json:"codec"`           // "pcm" ou "mp3"
	Container     string `json:"container"`       // "wav" ou "mp3"
	SampleRate    uint32 `json:"sample_rate"`     // Hz (PCM)
	Channels      uint16 `json:"channels"`        // PCM
	BitsPerSample uint16 `json:"bits_per_sample"` // PCM
	Source        string `json:"source"`          // arquivo de origem
}

func (m AudioMeta) Marshal() ([]byte, error) {
	return json.Marshal(m)
}

func ParseMeta(data []byte) (AudioMeta, error) {
	var m AudioMeta
	if err := json.Unmarshal(data, &m); err != nil {
		return m, fmt.Errorf("parse metadados: %w", err)
	}
	return m, nil
}
