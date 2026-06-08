package main

import (
	"flag"
	"log"

	"streaming-audio-redes/internal/client"
)

func main() {
	addr := flag.String("addr", "localhost:9090", "endereço do servidor")
	output := flag.String("output", "output/recebido.wav", "arquivo onde o fluxo será salvo")
	flag.Parse()

	l := client.New(*addr, *output)
	if err := l.Run(); err != nil {
		log.Fatal(err)
	}
}
