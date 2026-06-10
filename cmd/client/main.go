package main

import (
	"flag"
	"log"

	"streaming-audio-redes/internal/listener"
)

func main() {
	addr := flag.String("addr", "localhost:9090", "endereço do servidor")
	output := flag.String("output", "output/recebido.wav", "arquivo de saída (- para não gravar)")
	play := flag.Bool("play", false, "reproduzir áudio em tempo real enquanto recebe")
	flag.Parse()

	l := listener.New(*addr, *output, *play)
	if err := l.Run(); err != nil {
		log.Fatal(err)
	}
}
