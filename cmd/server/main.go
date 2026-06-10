package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"streaming-audio-redes/internal/audio/capture"
	"streaming-audio-redes/internal/broadcast"
	"streaming-audio-redes/internal/source"
)

func main() {
	addr := flag.String("addr", ":9090", "endereço TCP do servidor (ex: :9090)")
	srcMode := flag.String("source", "file", "fonte de áudio: file ou live")
	audio := flag.String("audio", "assets/musica.wav", "arquivo de áudio quando -source=file (.wav ou .mp3)")
	inputDevice := flag.String("input", "", "dispositivo de entrada no modo live (substring do nome; vazio = padrão)")
	sampleRate := flag.Uint("sample-rate", 44100, "taxa de amostragem no modo live (Hz)")
	channels := flag.Uint("channels", 1, "canais no modo live (1=mono, 2=stéreo)")
	listDevices := flag.Bool("list-devices", false, "lista microfones disponíveis e encerra")
	flag.Parse()

	if *listDevices {
		names, err := source.ListCaptureDevices()
		if err != nil {
			log.Fatal(err)
		}
		if len(names) == 0 {
			fmt.Println("Nenhum dispositivo de captura encontrado.")
			os.Exit(0)
		}
		fmt.Println("Dispositivos de entrada:")
		for _, name := range names {
			fmt.Printf("  - %s\n", name)
		}
		os.Exit(0)
	}

	var src broadcast.Source
	var err error

	switch strings.ToLower(*srcMode) {
	case "file":
		src, err = source.NewFile(*audio)
		if err != nil {
			log.Fatal(err)
		}
	case "live":
		src, err = source.NewLive(capture.LiveConfig{
			DeviceName: *inputDevice,
			SampleRate: uint32(*sampleRate),
			Channels:   uint16(*channels),
		})
		if err != nil {
			log.Fatal(err)
		}
	default:
		log.Fatalf("fonte inválida: %q (use file ou live)", *srcMode)
	}

	r := broadcast.New(*addr, src)
	log.Fatal(r.Run())
}
