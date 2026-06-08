package main

import (
	"flag"
	"log"
	"time"

	"streaming-audio-redes/internal/server"
)

func main() {
	addr := flag.String("addr", ":9090", "endereço TCP do servidor (ex: :9090)")
	audio := flag.String("audio", "assets/musica.wav", "caminho do arquivo de áudio (.wav ou .mp3)")
	chunkMs := flag.Int("chunk-ms", 20, "intervalo em ms entre blocos (simula streaming)")
	flag.Parse()

	wait := time.Duration(*chunkMs) * time.Millisecond
	r, err := server.New(*addr, *audio, wait)
	if err != nil {
		log.Fatal(err)
	}
	log.Fatal(r.Run())
}
