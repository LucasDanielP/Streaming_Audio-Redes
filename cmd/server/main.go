package main

import (
	"flag"
	"log"
	"time"

	"streaming-audio-redes/internal/server"
)

func main() {
	addr := flag.String("addr", ":9090", "endereço TCP do servidor (ex: :9090)")
	audio := flag.String("audio", "assets/sample.wav", "caminho do arquivo de áudio (.wav ou .mp3)")
	chunkMs := flag.Int("chunk-ms", 20, "intervalo em ms entre blocos (simula streaming)")
	flag.Parse()

	wait := time.Duration(*chunkMs) * time.Millisecond
	r := server.New(*addr, *audio, wait)
	log.Fatal(r.Run())
}
