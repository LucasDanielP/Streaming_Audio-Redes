# Streaming_Audio-Redes

Trabalho de Redes — rádio TCP em Go que transmite um arquivo de áudio em loop para vários clientes simultâneos.

## Funcionalidades (MVP)

| Requisito | Implementação |
|-----------|----------------|
| Servidor TCP em Go | `cmd/server` |
| Leitura contínua de arquivo de áudio | Loop em `internal/server` |
| Envio em blocos para todos os clientes | Broadcast com frames de 4 KB |
| Múltiplos clientes simultâneos | Mapa thread-safe de conexões |
| Cliente Go (salvar fluxo) | `cmd/client` grava em WAV/MP3 |
| Comandos pause / resume / stop | Prompt interativo no cliente |
| Logs de conexão, desconexão e pacotes | `log` no servidor e no cliente |

## Estrutura

```
cmd/
  server/     # servidor de rádio
  client/     # ouvinte
  gensample/  # gera WAV de teste
internal/
  protocol/   # framing [tamanho][dados]
  server/     # aceita TCP, broadcast
  client/     # recebe fluxo e comandos locais
assets/
  sample.wav  # áudio de exemplo (gerado)
```

## Pré-requisitos

- Go 1.22+

## Uso rápido

```bash
cd Streaming_Audio-Redes

# 1) Gerar áudio de teste (ou coloque seu .wav/.mp3 em assets/)
go run ./cmd/gensample

# 2) Iniciar o servidor
go run ./cmd/server -addr :9090 -audio assets/sample.wav

# 3) Em outro terminal — primeiro cliente
go run ./cmd/client -addr localhost:9090 -output output/cliente1.wav

# 4) Em outro terminal — segundo cliente (mesmo fluxo)
go run ./cmd/client -addr localhost:9090 -output output/cliente2.wav
```

### Comandos no cliente

No prompt `> `:

- `pause` ou `p` — pausa a gravação local (o fluxo continua chegando)
- `resume` ou `r` — retoma a gravação
- `stop` ou `s` — encerra o cliente
- `status` — mostra pacotes recebidos e bytes gravados

### Flags do servidor

| Flag | Padrão | Descrição |
|------|--------|-----------|
| `-addr` | `:9090` | Porta TCP |
| `-audio` | `assets/sample.wav` | Arquivo transmitido em loop |
| `-chunk-ms` | `20` | Intervalo entre blocos (ms) |

### Áudio próprio

Use qualquer `.wav` ou `.mp3`:

```bash
go run ./cmd/server -audio /caminho/para/musica.mp3
```

> A primeira versão reenvia o arquivo do início ao fim em loop, conforme o enunciado.

## Demonstração do MVP

1. Servidor iniciado com logs `rádio iniciada`
2. Dois clientes conectados — logs `ouvintes ativos: 2`
3. Pacotes sendo enviados/recebidos (`pacote #N`)
4. Comandos `pause` / `resume` / `stop` no cliente
5. Arquivos `output/cliente1.wav` reproduzíveis (ex.: `afplay output/cliente1.wav` no macOS)

## Protocolo (Fase 1)

Cada mensagem TCP:

```
[4 bytes: tamanho big-endian][payload]
```

Todos os clientes recebem os mesmos blocos no mesmo instante (broadcast).

## Próximos passos (fora do MVP)

- Captura de áudio ao vivo (microfone)
- Sincronização mais fina com RTP
- Reprodução em tempo real no cliente (sem só gravar)
