// Gera um arquivo WAV curto (tom de 440 Hz) para testes locais.
package main

import (
	"encoding/binary"
	"flag"
	"log"
	"math"
	"os"
)

func main() {
	out := flag.String("out", "assets/sample.wav", "caminho de saída")
	duration := flag.Int("duration", 3, "duração em segundos")
	flag.Parse()

	const (
		sampleRate = 44100
		freq       = 440.0
	)

	numSamples := sampleRate * (*duration)
	dataSize := numSamples * 2 // 16-bit mono

	f, err := os.Create(*out)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	writeString(f, "RIFF")
	_ = binary.Write(f, binary.LittleEndian, uint32(36+dataSize))
	writeString(f, "WAVE")
	writeString(f, "fmt ")
	_ = binary.Write(f, binary.LittleEndian, uint32(16))
	_ = binary.Write(f, binary.LittleEndian, uint16(1)) // PCM
	_ = binary.Write(f, binary.LittleEndian, uint16(1)) // mono
	_ = binary.Write(f, binary.LittleEndian, uint32(sampleRate))
	byteRate := sampleRate * 2
	_ = binary.Write(f, binary.LittleEndian, uint32(byteRate))
	_ = binary.Write(f, binary.LittleEndian, uint16(2))
	_ = binary.Write(f, binary.LittleEndian, uint16(16))
	writeString(f, "data")
	_ = binary.Write(f, binary.LittleEndian, uint32(dataSize))

	for i := 0; i < numSamples; i++ {
		t := float64(i) / sampleRate
		sample := int16(32767 * 0.3 * math.Sin(2*math.Pi*freq*t))
		_ = binary.Write(f, binary.LittleEndian, sample)
	}

	log.Printf("arquivo gerado: %s (%d s, %d amostras, %d bytes)",
		*out, *duration, numSamples, 44+dataSize)
}

func writeString(f *os.File, s string) {
	_, _ = f.Write([]byte(s))
}
