package main

import (
	"flag"
	"log"

	"streaming-audio-redes/internal/client"
)

func main() {
	/*
		flags para o cliente:
		- addr: endereço do servidor
		- output: arquivo de saída (- para não gravar)
		- play: reproduzir áudio em tempo real enquanto recebe
	*/
	addr := flag.String("addr", "localhost:9090", "endereço do servidor")
	output := flag.String("output", "output/recebido.wav", "arquivo de saída (- para não gravar)")
	play := flag.Bool("play", false, "reproduzir áudio em tempo real enquanto recebe")
	flag.Parse()

	l := client.New(*addr, *output, *play)
	if err := l.Run(); err != nil {
		log.Fatal(err)
	}
}
